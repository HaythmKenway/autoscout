package ai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/HaythmKenway/autoscout/pkg/burp"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

type OllamaBackend struct {
	BaseURL string
	Model   string
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"` // "json" for structured output
}

type ollamaResponse struct {
	Response string `json:"response"`
}

func NewOllamaBackend(baseURL, model string) *OllamaBackend {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3.2:latest"
	}
	return &OllamaBackend{BaseURL: baseURL, Model: model}
}

func (o *OllamaBackend) Analyze(req burp.BurpRequest) (*AIPlan, error) {
	decodedBody, _ := base64.StdEncoding.DecodeString(req.Body)
	bodyStr := string(decodedBody)
	if bodyStr == "" {
		bodyStr = "[Empty Body]"
	}

	headersJSON, _ := json.MarshalIndent(req.Headers, "", "  ")
	userKnowledge := LoadKnowledge()

	prompt := fmt.Sprintf(`Analyze the following HTTP request for security vulnerabilities.
Method: %s
URL: %s
Tool Source: %s
Headers: 
%s
Body: %s

### User Training & Expertise:
%s

### Instructions:
1. **Analyze Request**: Carefully inspect headers and the body for sensitive data, injection points, or complex logic. Check for interesting headers like Authorization, Cookies, or custom headers. Use the provided "User Training" to guide your analysis.
   - **Selective Fuzzing**: If the request is a simple GET with no parameters, or a POST with a static/irrelevant body, DO NOT trigger parameter fuzzing (ffuf, dalfox with parameters) unless there's a specific reason. Avoid "waste of time" scans on obviously static endpoints.
   - **GraphQL/API**: Prioritize targeted checks for these endpoints rather than generic fuzzing.
2. **Rules for Tool Selection**:
   - API/GraphQL: If the URL contains '/api/' or 'graphql', DO NOT use web crawlers (gospider, katana). Use nuclei or ffuf instead.
   - Parameters: If parameters are detected, use dalfox (for XSS) or sqlmap (for SQLi).
   - Censys: Only use if you see an IP address or want to check for exposed services on a new domain.

3. **Stealth and Rate Limiting**:
   - ALWAYS include a "rate_limit" parameter in "params" for high-volume tools (ffuf, katana, gospider, nuclei).
   - If the target is a major platform (e.g., reddit, google, github), set "rate_limit" to a low value (e.g., 5-10 requests per second) to avoid blocking.
   - Example for FFUF on a sensitive target: {"tool": "ffuf", "target": "URL", "params": {"rate_limit": "5"}}

4. **Output Format**: Provide your analysis in STRICT JSON format with these keys:
   - thinking: A detailed, step-by-step reasoning of your analysis. Explain WHY you suspect certain bugs.
   - vulnerabilities_suspected: [list of strings]
   - actions: [list of {"tool": "toolname", "target": "url", "params": {"key": "val"}}]
   - rewrite_rules: [list of strings for future modifications]

Only output the JSON object. No preamble.`, req.Method, req.URL, req.Tool, string(headersJSON), bodyStr, userKnowledge)

	ollamaReq := ollamaRequest{
		Model:  o.Model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	jsonData, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(o.BaseURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ollama connection failed: %v", err)
	}
	defer resp.Body.Close()

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %v", err)
	}

	if ollamaResp.Response == "" {
		return nil, fmt.Errorf("ollama returned an empty response (check if model '%s' is installed)", o.Model)
	}

	var plan AIPlan
	if err := json.Unmarshal([]byte(ollamaResp.Response), &plan); err != nil {
		// Log the failed JSON for debugging
		localUtils.Logger(fmt.Sprintf("[DEBUG] AI returned invalid JSON: %s", ollamaResp.Response), 3)
		return nil, fmt.Errorf("failed to parse AI plan JSON: %v", err)
	}

	return &plan, nil
}
