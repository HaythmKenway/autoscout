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
	"github.com/HaythmKenway/autoscout/pkg/tools"
)

type OllamaBackend struct {
	Model string
}

func NewOllamaBackend(model string) *OllamaBackend {
	if model == "" {
		model = "llama3:8b"
	}
	return &OllamaBackend{
		Model: model,
	}
}

func (o *OllamaBackend) Name() string {
	return "Ollama (" + o.Model + ")"
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format,omitempty"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

func (o *OllamaBackend) Analyze(req burp.BurpRequest) (*AIPlan, error) {
	decodedBody, _ := base64.StdEncoding.DecodeString(req.Body)
	bodyStr := TruncateBody(string(decodedBody), 10000)
	if bodyStr == "" {
		bodyStr = "[Empty Body]"
	}

	headersJSON, _ := json.MarshalIndent(req.Headers, "", "  ")
	userKnowledge := LoadKnowledge()
	toolContext := tools.GetToolCapabilitiesJSON()

	var responseContext string
	if req.ResponseStatus > 0 {
		resHeadersJSON, _ := json.MarshalIndent(req.ResponseHeaders, "", "  ")
		decodedResBody, _ := base64.StdEncoding.DecodeString(req.ResponseBody)
		resBodyStr := TruncateBody(string(decodedResBody), 10000)
		if resBodyStr == "" {
			resBodyStr = "[Empty Response Body]"
		}
		responseContext = fmt.Sprintf("\n### RESPONSE DATA\nStatus: %d\nHeaders:\n%s\nBody: %s\n", req.ResponseStatus, string(resHeadersJSON), resBodyStr)
	}

	var userInstructions string
	if req.UserContext != "" {
		userInstructions = fmt.Sprintf("\n### SPECIAL USER INSTRUCTIONS (PRIORITY):\n%s\n", req.UserContext)
	}

	prompt := fmt.Sprintf(`Analyze the following HTTP request (and response if provided) for security vulnerabilities.
%s
Method: %s
URL: %s
Tool Source: %s
Headers: 
%s
Body: %s
%s
### User Training & Expertise:
%s

### Available Security Tools (MCP-Interface):
%s

### Instructions:
1. **Analyze Request**: Carefully inspect headers and the body for sensitive data, injection points, or complex logic.
   - **STRICT NO-FUZZ RULE**: DO NOT trigger fuzzing tools (ffuf, dalfox, sqlmap) on static assets (.js, .css, .png, etc.) or simple informational GET requests with no parameters.
   - **BODY ANALYSIS**: If the request is a POST, you MUST find actual user-controlled data in the body before suggesting a tool. If the body is empty or static JSON/XML with no user input, SKIP fuzzing.
   - **GraphQL/API**: Only perform targeted scans if you see complex queries or potential for BOLA/IDOR. Avoid generic wordlist fuzzing on established APIs unless necessary.

2. **Stealth and Rate Limiting**:
   - YOU MUST include a "rate_limit" parameter in "params" for high-volume tools (ffuf, katana, nuclei).
   - FAILURE to provide a rate limit will result in system blocks. Use "5" as a standard safe value.

3. **Output Format**: Provide your analysis in STRICT JSON format with the following structure:
   {
     "thinking": "detailed reasoning",
     "vulnerabilities_suspected": ["type1", "type2"],
     "actions": [{"tool": "toolname", "target": "url", "params": {"rate_limit": "5"}}],
     "rewrite_rules": ["modified_body_base64"]
   }
   Only output the JSON object. No preamble.`, userInstructions, req.Method, req.URL, req.Tool, string(headersJSON), bodyStr, responseContext, userKnowledge, toolContext)

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

	url := "http://localhost:11434/api/generate"
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ollama connection failed: %v", err)
	}
	defer resp.Body.Close()

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %v", err)
	}

	var plan AIPlan
	if err := json.Unmarshal([]byte(ollamaResp.Response), &plan); err != nil {
		localUtils.Logger(fmt.Sprintf("[DEBUG] AI returned invalid JSON: %s", ollamaResp.Response), 3)
		return nil, fmt.Errorf("failed to parse AI plan JSON: %v", err)
	}

	return &plan, nil
}
