package burp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

type BurpRequest struct {
	URL    string `json:"url"`
	Method string `json:"method"`
	Body   string `json:"body"` // Base64 encoded
}

type BurpResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body"` // Base64 encoded
}

type BurpModified struct {
	Modified bool   `json:"modified"`
	Body     string `json:"body"` // Base64 encoded
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
)

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

func addAnalysis(entry string) {
	mu.Lock()
	defer mu.Unlock()
	AnalysisQueue = append(AnalysisQueue, entry)
	if len(AnalysisQueue) > 100 {
		AnalysisQueue = AnalysisQueue[1:]
	}
}

func StartServer(port string) error {
	mu.Lock()
	if running {
		mu.Unlock()
		return nil
	}
	running = true
	mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		localUtils.Logger(fmt.Sprintf("[Burp -> Ping] %s %s", r.Method, r.URL.Path), 1)
		w.Write([]byte("Autoscout Burp API Online"))
	})
	mux.HandleFunc("/request", handleRequest)
	mux.HandleFunc("/response", handleResponse)

	addr := fmt.Sprintf("127.0.0.1:%s", port)
	server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	localUtils.Logger(fmt.Sprintf("Starting Burp Integration Server on %s", addr), 1)
	
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			localUtils.Logger(fmt.Sprintf("Burp Integration Server error: %v", err), 2)
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
	localUtils.Logger(fmt.Sprintf("[Burp -> Request] %s %s (%d bytes)", req.Method, req.URL, len(decodedBody)), 1)

	// Add to Analysis UI
	addAnalysis(fmt.Sprintf("REQ: %s %s (%d bytes)", req.Method, req.URL, len(decodedBody)))

	// DEMO: AI Modification Loop
	modified := false
	newBody := req.Body
	if strings.Contains(strings.ToLower(string(decodedBody)), "fuzz-me") {
		localUtils.Logger("[AI] Detected 'fuzz-me' keyword. Modifying request...", 1)
		fuzzed := strings.ReplaceAll(string(decodedBody), "fuzz-me", "AUTOSCOUT-FUZZED")
		newBody = base64.StdEncoding.EncodeToString([]byte(fuzzed))
		modified = true
		addAnalysis("AI: Modified request body (keyword 'fuzz-me' detected)")
	}

	resp := BurpModified{
		Modified: modified,
		Body:     newBody,
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
	localUtils.Logger(fmt.Sprintf("[Burp -> Response] Status %d (%d bytes)", resp.Status, len(decodedBody)), 1)

	// Add to Analysis UI
	addAnalysis(fmt.Sprintf("RES: Status %d (%d bytes)", resp.Status, len(decodedBody)))

	// DEMO: AI Analysis Loop
	if strings.Contains(strings.ToLower(string(decodedBody)), "admin") {
		localUtils.Logger("[AI] Found 'admin' in response. Flagging for review.", 1)
		addAnalysis("AI ALERT: 'admin' keyword detected in response body!")
	}

	modResp := BurpModified{
		Modified: false,
		Body:     resp.Body,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modResp)
}
