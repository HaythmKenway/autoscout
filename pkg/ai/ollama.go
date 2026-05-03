package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/HaythmKenway/autoscout/pkg/burp"
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
		model = "llama3"
	}
	return &OllamaBackend{BaseURL: baseURL, Model: model}
}

func (o *OllamaBackend) Analyze(req burp.BurpRequest) (*AIPlan, error) {
	prompt := fmt.Sprintf(`Analyze the following HTTP request for security vulnerabilities.
Method: %s
URL: %s
Tool Source: %s
Body: %s

Think step-by-step about possible bugs like XSS, SQLi, SSRF, IDOR, or API misconfigurations.
Provide your analysis in STRICT JSON format with the following keys:
- vulnerabilities_suspected: [list of strings]
- actions: [list of {"tool": "dalfox|sqlmap|nuclei|ffuf|arjun|katana|gospider", "target": "url", "params": {"key": "val"}}]
- rewrite_rules: [list of strings for future modifications]

Example Action for SQLMap: {"tool": "sqlmap", "target": "URL", "params": {"batch": "true", "risk": "3"}}
Example Action for DalFox: {"tool": "dalfox", "target": "URL", "params": {"args": "--mining-dict"}}

Only output the JSON object. No preamble.`, req.Method, req.URL, req.Tool, req.Body)

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
		return nil, err
	}

	var plan AIPlan
	if err := json.Unmarshal([]byte(ollamaResp.Response), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse AI plan JSON: %v", err)
	}

	return &plan, nil
}
