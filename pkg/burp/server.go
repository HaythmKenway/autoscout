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

var (
	mu      sync.Mutex
	running bool
	server  *http.Server

	// Traffic Stats
	RequestsIntercepted  int
	ResponsesIntercepted int

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
	} else {
		localUtils.Logger(fmt.Sprintf("[AI] Skipping redundant analysis for %s", req.URL), 1)
	}

	// Apply AI Rewrite if exists
	if modifiedBody, ok := checkRewrite(req.URL); ok {
		localUtils.Logger("[AI] Applying active rewrite rule for this target", 1)
		AddAnalysis("AI: Applied active rewrite rule to request")
		resp := BurpModified{Modified: true, Body: modifiedBody}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// --- AI Heuristic Routing ---
	if !isStaticAsset(req.URL) {
		AddAnalysis(fmt.Sprintf("[%s] REQ: %s %s", req.Tool, req.Method, req.URL))

		// Check for parameters (Potential SQLi/XSS/Fuzzing)
		if strings.Contains(req.URL, "?") || len(decodedBody) > 0 {
			delegateToAgent("ParameterFuzzer", req.URL)
		}

		// Check for JSON/API (Potential IDOR/BOLA)
		if strings.Contains(strings.ToLower(req.URL), "/api/") {
			delegateToAgent("APIAnalyzer", req.URL)
		}

		// DEMO: Specific keyword modification
		if strings.Contains(strings.ToLower(string(decodedBody)), "fuzz-me") {
			localUtils.Logger("[AI] Detected 'fuzz-me' keyword. Modifying request...", 1)
			fuzzed := strings.ReplaceAll(string(decodedBody), "fuzz-me", "AUTOSCOUT-FUZZED")
			req.Body = base64.StdEncoding.EncodeToString([]byte(fuzzed))
			AddAnalysis("AI: Modified request body (keyword 'fuzz-me' detected)")

			resp := BurpModified{Modified: true, Body: req.Body}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	}

	resp := BurpModified{
		Modified: false,
		Body:     req.Body,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleResponse(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	ResponsesIntercepted++
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

	decodedBody, _ := base64.StdEncoding.DecodeString(resp.Body)
	localUtils.Logger(fmt.Sprintf("[Burp -> %s] Response Status %d (%d bytes)", resp.Tool, resp.Status, len(decodedBody)), 1)

	// --- AI Heuristic Routing ---
	if resp.Status == 200 && len(decodedBody) > 0 {
		AddAnalysis(fmt.Sprintf("[%s] RES: Status %d (%d bytes)", resp.Tool, resp.Status, len(decodedBody)))

		// Check for sensitive info (PII/Secrets)
		bodyStr := string(decodedBody)
		if strings.Contains(bodyStr, "AWS_ACCESS_KEY") || strings.Contains(bodyStr, "API_KEY") || strings.Contains(bodyStr, "secret") {
			delegateToAgent("InfoLeakScanner", "Check logs for details")
		}

		if strings.Contains(strings.ToLower(bodyStr), "admin") {
			AddAnalysis("AI ALERT: 'admin' keyword detected in response body!")
		}
	}

	modResp := BurpModified{
		Modified: false,
		Body:     resp.Body,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modResp)
}

func handleManual(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	localUtils.Logger(fmt.Sprintf("[DEBUG] Raw Manual JSON: %s", string(body)), 3)

	var req BurpManualRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	localUtils.Logger(fmt.Sprintf("[Burp -> MANUAL] High-Priority Analysis for %s", req.URL), 1)
	AddAnalysis("CRITICAL: Received Manual Investigation Task!")
	AddAnalysis(fmt.Sprintf("TARGET: %s %s", req.Method, req.URL))

	// Save session to /tmp/autoscout
	sessionID := time.Now().Format("20060102-150405")
	saveManualSession(sessionID, req)

	// Update urls table with session_id
	go func() {
		database, err := db.OpenDatabase()
		if err == nil {
			defer database.Close()
			u, _ := url.Parse(req.URL)
			host := u.Hostname()
			db.AddUrl(database, host, "", req.URL, host, u.Scheme, "", "", "", "", u.Port(), fmt.Sprintf("%d", req.Status), sessionID)
		}
	}()

	// Send to Orchestrator
	if WorkQueue != nil {
		decodedRawReq, _ := base64.StdEncoding.DecodeString(req.RawRequest)
		WorkQueue <- BurpRequest{
			URL:            req.URL,
			Method:         req.Method,
			Tool:           req.Tool,
			Headers:        req.Headers,
			Body:           base64.StdEncoding.EncodeToString(decodedRawReq), // Pass raw request as context
			ResponseStatus: req.Status,
			ResponseBody:   req.RawResponse,
		}
	}

	if req.Status > 0 {
		AddAnalysis(fmt.Sprintf("STATUS: %d", req.Status))
	}

	delegateToAgent("DeepScanner", req.URL)

	w.WriteHeader(http.StatusOK)
}

func saveManualSession(id string, req BurpManualRequest) {
	dir := "/tmp/autoscout"
	if err := os.MkdirAll(dir, 0755); err != nil {
		localUtils.Logger(fmt.Sprintf("Failed to create session dir: %v", err), 2)
		return
	}

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
