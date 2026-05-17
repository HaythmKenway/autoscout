package burp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

type BurpRequest struct {
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	Tool            string            `json:"tool"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"` // Base64 encoded
	ResponseStatus  int               `json:"res_status,omitempty"`
	ResponseHeaders map[string]string `json:"res_headers,omitempty"`
	ResponseBody    string            `json:"res_body,omitempty"` // Base64 encoded
	UserContext     string            `json:"user_context,omitempty"`
}

type BurpResponse struct {
	Status  int               `json:"status"`
	Tool    string            `json:"tool"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"` // Base64 encoded
}

type BurpModified struct {
	Modified bool   `json:"modified"`
	Body     string `json:"body"` // Base64 encoded
}

type BurpManualRequest struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Tool        string            `json:"tool"`
	Headers     map[string]string `json:"headers"`
	RawRequest  string            `json:"raw_request"`  // Base64 encoded raw HTTP request
	Status      int               `json:"status"`       // Optional
	RawResponse string            `json:"raw_response"` // Base64 encoded raw HTTP response, Optional
}

type ProxyEntry struct {
	SessionID string    `json:"id"`
	Method    string    `json:"method"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	Tool      string    `json:"tool"`
	Timestamp time.Time `json:"timestamp"`
}

var (
	mu      sync.Mutex
	running bool
	server  *http.Server

	// Traffic Stats
	RequestsIntercepted  int
	ResponsesIntercepted int

	// Proxy History & Sessions
	CurrentSession = "Autoscout"
	ProxyHistory   []ProxyEntry
	historyMu      sync.RWMutex

	// Analysis Feed
	AnalysisQueue []string

	// Orchestrator integration
	WorkQueue chan BurpRequest

	// AI Rewrite Registry (URL -> Modified Body)
	RewriteRules map[string]string

	// Analysis Cache
	analysisCache map[string]time.Time
	cacheMu       sync.Mutex
)

func GetProxyHistory() []ProxyEntry {
	historyMu.RLock()
	defer historyMu.RUnlock()
	if len(ProxyHistory) > 0 {
		localUtils.Logger(fmt.Sprintf("[DEBUG] GetProxyHistory: Returning %d entries", len(ProxyHistory)), 3)
	}
	return ProxyHistory
}

func ClearProxyHistory() {
	historyMu.Lock()
	defer historyMu.Unlock()
	ProxyHistory = []ProxyEntry{}
}

func RemoveProxyEntry(id string) {
	historyMu.Lock()
	defer historyMu.Unlock()
	newH := []ProxyEntry{}
	for _, e := range ProxyHistory {
		if e.SessionID != id {
			newH = append(newH, e)
		}
	}
	ProxyHistory = newH
}

func GetSessions() []string {
	base := os.ExpandEnv("$HOME/.autoscout/sessions")
	os.MkdirAll(base, 0755)
	files, err := os.ReadDir(base)
	if err != nil {
		return []string{}
	}
	var sessions []string
	for _, f := range files {
		if f.IsDir() {
			sessions = append(sessions, f.Name())
		}
	}
	return sessions
}

func SetSession(name string) {
	mu.Lock()
	CurrentSession = name
	mu.Unlock()
	
	// Load history index if it exists
	var dir string
	if name != "" && name != "Autoscout" {
		dir = filepath.Join(os.ExpandEnv("$HOME/.autoscout/sessions"), name)
	} else {
		dir = "/tmp/autoscout/session_Autoscout"
	}

	historyPath := filepath.Join(dir, "history.json")
	data, err := os.ReadFile(historyPath)
	if err == nil {
		var h []ProxyEntry
		if err := json.Unmarshal(data, &h); err == nil {
			historyMu.Lock()
			ProxyHistory = h
			historyMu.Unlock()
		}
	} else {
		ClearProxyHistory()
	}
	localUtils.Logger(fmt.Sprintf("Session set to: %s", name), 1)
}

func saveHistoryIndex() {
	mu.Lock()
	session := CurrentSession
	mu.Unlock()
	if session == "" { return }

	historyMu.RLock()
	data, _ := json.MarshalIndent(ProxyHistory, "", "  ")
	historyMu.RUnlock()

	var dir string
	if session != "" && session != "Autoscout" {
		dir = filepath.Join(os.ExpandEnv("$HOME/.autoscout/sessions"), session)
	} else {
		dir = "/tmp/autoscout/session_Autoscout"
	}
	
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "history.json"), data, 0644)
}

func shouldAnalyze(method, url string) bool {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if analysisCache == nil {
		analysisCache = make(map[string]time.Time)
	}
	key := method + ":" + url
	last, exists := analysisCache[key]
	if exists && time.Since(last) < 30*time.Second {
		return false
	}
	analysisCache[key] = time.Now()
	return true
}

func RegisterRewrite(url, newBody string) {
	mu.Lock()
	defer mu.Unlock()
	if RewriteRules == nil {
		RewriteRules = make(map[string]string)
	}
	RewriteRules[url] = newBody
}

func checkRewrite(url string) (string, bool) {
	mu.Lock()
	defer mu.Unlock()
	if RewriteRules == nil {
		return "", false
	}
	body, ok := RewriteRules[url]
	return body, ok
}

func IsRunning() bool {
	mu.Lock()
	defer mu.Unlock()
	return running
}

func GetStats() (int, int, []string) {
	mu.Lock()
	defer mu.Unlock()
	q := AnalysisQueue
	AnalysisQueue = []string{} // Clear queue after polling
	return RequestsIntercepted, ResponsesIntercepted, q
}

func AddAnalysis(entry string) {
	mu.Lock()
	defer mu.Unlock()
	AnalysisQueue = append(AnalysisQueue, entry)
	if len(AnalysisQueue) > 100 {
		AnalysisQueue = AnalysisQueue[1:]
	}
}

func StartServer(port string, workQueue chan BurpRequest) error {
	mu.Lock()
	if running {
		mu.Unlock()
		return nil
	}
	// Don't set running=true yet, wait for successful listen or at least attempt
	mu.Unlock()

	WorkQueue = workQueue

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		localUtils.Logger(fmt.Sprintf("[Burp -> Ping] %s %s", r.Method, r.URL.Path), 1)
		w.Write([]byte("Autoscout Burp API Online"))
	})
	mux.HandleFunc("/request", handleRequest)
	mux.HandleFunc("/response", handleResponse)
	mux.HandleFunc("/manual", handleManual)

	addr := ":" + port // Listen on all interfaces
	server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	localUtils.Logger(fmt.Sprintf("Attempting to start Burp Integration Server on %s", addr), 1)

	// Create a listener first to check for port conflicts immediately
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("CRITICAL: Burp Server failed to bind to %s: %v", addr, err), 2)
		return err
	}

	mu.Lock()
	running = true
	mu.Unlock()

	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			localUtils.Logger(fmt.Sprintf("Burp Integration Server runtime error: %v", err), 2)
			mu.Lock()
			running = false
			mu.Unlock()
		}
	}()

	return nil
}

func StopServer() {
	mu.Lock()
	defer mu.Unlock()

	if !running || server == nil {
		localUtils.Logger("Burp Server is not running, skipping stop.", 1)
		return
	}

	localUtils.Logger("Stopping Burp Integration Server...", 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		localUtils.Logger(fmt.Sprintf("Burp Server shutdown error: %v", err), 2)
	}

	running = false
	server = nil
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	localUtils.Logger("[DEBUG] Received request on /request", 3)

	mu.Lock()
	RequestsIntercepted++
	session := CurrentSession
	mu.Unlock()

	if r.Method != http.MethodPost {
		localUtils.Logger("[DEBUG] Method not allowed", 3)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}

	var req BurpRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	decodedBody, _ := base64.StdEncoding.DecodeString(req.Body)
	localUtils.Logger(fmt.Sprintf("[Burp -> %s] %s %s (%d bytes)", req.Tool, req.Method, req.URL, len(decodedBody)), 1)

	// Record in Proxy History
	id := time.Now().Format("20060102-150405.000")
	entry := ProxyEntry{
		SessionID: id,
		Method:    req.Method,
		URL:       req.URL,
		Status:    "REQ",
		Tool:      req.Tool,
		Timestamp: time.Now(),
	}
	historyMu.Lock()
	ProxyHistory = append(ProxyHistory, entry)
	historyMu.Unlock()
	
	saveData(id, req.Body, "", session)
	saveHistoryIndex()

	// Auto-add domain to targets
	go func() {
		u, err := url.Parse(req.URL)
		if err == nil {
			db.AddTarget(u.Hostname())
		}
	}()

	// Send to AI Orchestrator
	if WorkQueue != nil && shouldAnalyze(req.Method, req.URL) {
		WorkQueue <- req
	}

	// Apply AI Rewrite if exists
	if modifiedBody, ok := checkRewrite(req.URL); ok {
		resp := BurpModified{Modified: true, Body: modifiedBody}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := BurpModified{Modified: false, Body: req.Body}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleResponse(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	ResponsesIntercepted++
	session := CurrentSession
	mu.Unlock()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}

	var resp BurpResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Update existing REQ entry with status if found (approximate)
	historyMu.Lock()
	for i := len(ProxyHistory) - 1; i >= 0; i-- {
		if ProxyHistory[i].Status == "REQ" && ProxyHistory[i].Tool == resp.Tool {
			ProxyHistory[i].Status = fmt.Sprintf("%d", resp.Status)
			saveData(ProxyHistory[i].SessionID, "", resp.Body, session)
			break
		}
	}
	historyMu.Unlock()
	saveHistoryIndex()

	modResp := BurpModified{Modified: false, Body: resp.Body}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modResp)
}

func handleManual(w http.ResponseWriter, r *http.Request) {
	localUtils.Logger(fmt.Sprintf("[DEBUG] handleManual hit: %s %s", r.Method, r.URL.Path), 3)
	if r.Method != http.MethodPost {
		localUtils.Logger("[DEBUG] handleManual: Method not allowed", 3)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("[DEBUG] handleManual: Failed to read body: %v", err), 2)
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}

	var req BurpManualRequest
	if err := json.Unmarshal(body, &req); err != nil {
		localUtils.Logger(fmt.Sprintf("[DEBUG] handleManual: Invalid JSON: %v. Body: %s", err, string(body)), 2)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	localUtils.Logger(fmt.Sprintf("[Burp -> MANUAL] Forwarded %s %s", req.Method, req.URL), 1)

	mu.Lock()
	session := CurrentSession
	mu.Unlock()

	id := time.Now().Format("20060102-150405.000")
	statusStr := "REQ"
	if req.Status > 0 {
		statusStr = fmt.Sprintf("%d", req.Status)
	}

	entry := ProxyEntry{
		SessionID: id,
		Method:    req.Method,
		URL:       req.URL,
		Status:    statusStr,
		Tool:      req.Tool,
		Timestamp: time.Now(),
	}
	historyMu.Lock()
	ProxyHistory = append(ProxyHistory, entry)
	historyMu.Unlock()

	go func() {
		saveData(id, req.RawRequest, req.RawResponse, session)
		saveHistoryIndex()
	}()

	w.WriteHeader(http.StatusOK)
}

func saveData(id, reqB64, resB64, session string) {
	var dir string
	if session != "" && session != "Autoscout" {
		dir = filepath.Join(os.ExpandEnv("$HOME/.autoscout/sessions"), session)
	} else {
		dir = "/tmp/autoscout/session_Autoscout"
	}
	os.MkdirAll(dir, 0755)

	if reqB64 != "" {
		reqData, _ := base64.StdEncoding.DecodeString(reqB64)
		os.WriteFile(filepath.Join(dir, id+".request"), reqData, 0644)
	}
	if resB64 != "" {
		resData, _ := base64.StdEncoding.DecodeString(resB64)
		os.WriteFile(filepath.Join(dir, id+".response"), resData, 0644)
	}
}

func saveManualSession(id string, req BurpManualRequest) {
	mu.Lock()
	session := CurrentSession
	mu.Unlock()

	var dir string
	if session != "" && session != "Autoscout" {
		dir = filepath.Join(os.ExpandEnv("$HOME/.autoscout/sessions"), session)
	} else {
		dir = "/tmp/autoscout/session_Autoscout"
	}
	os.MkdirAll(dir, 0755)

	// Save Raw Request
	reqPath := filepath.Join(dir, id+".request")
	decodedReq, _ := base64.StdEncoding.DecodeString(req.RawRequest)
	if err := os.WriteFile(reqPath, decodedReq, 0644); err != nil {
		localUtils.Logger(fmt.Sprintf("Failed to save .request file: %v", err), 2)
	}

	// Save Raw Response if exists
	if req.RawResponse != "" {
		resPath := filepath.Join(dir, id+".response")
		decodedRes, _ := base64.StdEncoding.DecodeString(req.RawResponse)
		if err := os.WriteFile(resPath, decodedRes, 0644); err != nil {
			localUtils.Logger(fmt.Sprintf("Failed to save .response file: %v", err), 2)
		}
	}

	localUtils.Logger(fmt.Sprintf("Manual session saved to %s/%s.*", dir, id), 1)
}

func isStaticAsset(u string) bool {
	extensions := []string{".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".woff", ".woff2", ".ttf", ".ico"}
	for _, ext := range extensions {
		if strings.HasSuffix(strings.Split(u, "?")[0], ext) {
			return true
		}
	}
	return false
}

func delegateToAgent(agentName string, target string) {
	msg := fmt.Sprintf("[AI Fleet] %s is investigating %s", agentName, target)
	localUtils.Logger(msg, 1)
	AddAnalysis(msg)
}
