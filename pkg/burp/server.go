package burp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

func StartServer(port string) {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/request", handleRequest)
	mux.HandleFunc("/response", handleResponse)

	addr := fmt.Sprintf("127.0.0.1:%s", port)
	localUtils.Logger(fmt.Sprintf("Starting Burp Integration Server on %s", addr), 1)
	
	if err := http.ListenAndServe(addr, mux); err != nil {
		localUtils.Logger(fmt.Sprintf("Burp Integration Server error: %v", err), 2)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	localUtils.Logger(fmt.Sprintf("[Burp -> Autoscout] Intercepted Request: %s %s (Body: %d bytes)", req.Method, req.URL, len(decodedBody)), 3)

	// Here you would pass the request to the AI Fleet or other analyzers
	// For now, we just pass it back unmodified
	
	resp := BurpModified{
		Modified: false,
		Body:     req.Body,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleResponse(w http.ResponseWriter, r *http.Request) {
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
	localUtils.Logger(fmt.Sprintf("[Burp -> Autoscout] Intercepted Response: Status %d (Body: %d bytes)", resp.Status, len(decodedBody)), 3)

	// Analyzer logic here
	
	modResp := BurpModified{
		Modified: false,
		Body:     resp.Body,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modResp)
}
