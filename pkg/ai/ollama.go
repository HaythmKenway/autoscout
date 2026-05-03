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

func (o *OllamaBackend) Name() string {
	return "Ollama (" + o.Model + ")"
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
1. **Analyze Request**: Carefully inspect headers and the body for sensitive data, injection points, or complex logic.
   - **STRICT NO-FUZZ RULE**: DO NOT trigger fuzzing tools (ffuf, dalfox, sqlmap) on static assets (.js, .css, .png, etc.) or simple informational GET requests with no parameters.
   - **BODY ANALYSIS**: If the request is a POST, you MUST find actual user-controlled data in the body before suggesting a tool. If the body is empty or static JSON/XML with no user input, SKIP fuzzing.
   - **GraphQL/API**: Only perform targeted scans if you see complex queries or potential for BOLA/IDOR. Avoid generic wordlist fuzzing on established APIs unless necessary.

2. **Stealth and Rate Limiting**:
   - YOU MUST include a "rate_limit" parameter in "params" for high-volume tools (ffuf, katana, gospider, nuclei).
   - FAILURE to provide a rate limit will result in system blocks. Use "5" as a standard safe value.

3. **Output Format**: Provide your analysis in STRICT JSON format with the following structure:
   {
     "thinking": "detailed reasoning",
     "vulnerabilities_suspected": ["type1", "type2"],
     "actions": [{"tool": "toolname", "target": "url", "params": {"key": "val"}}],
     "rewrite_rules": ["modified_body_base64"]
   }
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
