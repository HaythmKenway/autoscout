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

	prompt := fmt.Sprintf(`Analyze the following HTTP request for security vulnerabilities.
Method: %s
URL: %s
Tool Source: %s
Body: %s

### Instructions:
1. **Analyze Body**: Carefully inspect the POST body or parameters for sensitive data, injection points, or complex logic.
2. **Rules for Tool Selection**:
   - API/GraphQL: If the URL contains '/api/' or 'graphql', DO NOT use web crawlers (gospider, katana). Use nuclei or ffuf instead.
   - Parameters: If parameters are detected, use dalfox (for XSS) or sqlmap (for SQLi).
   - Censys: Only use if you see an IP address or want to check for exposed services on a new domain.

3. **Output Format**: Provide your analysis in STRICT JSON format with these keys:
   - thinking: A detailed, step-by-step reasoning of your analysis. Explain WHY you suspect certain bugs.
   - vulnerabilities_suspected: [list of strings]
   - actions: [list of {"tool": "dalfox|sqlmap|nuclei|ffuf|arjun|katana|gospider", "target": "url", "params": {"key": "val"}}]
   - rewrite_rules: [list of strings for future modifications]

Example Action for SQLMap: {"tool": "sqlmap", "target": "URL", "params": {"batch": "true", "risk": "3"}}

Only output the JSON object. No preamble.`, req.Method, req.URL, req.Tool, bodyStr)

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
