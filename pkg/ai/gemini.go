package ai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/HaythmKenway/autoscout/pkg/burp"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/HaythmKenway/autoscout/pkg/tools"
)

type GeminiBackend struct {
	APIKey string
	Model  string
}

func NewGeminiBackend(apiKey, model string) *GeminiBackend {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	return &GeminiBackend{
		APIKey: apiKey,
		Model:  model,
	}
}

func (g *GeminiBackend) Name() string {
	return "Gemini (" + g.Model + ")"
}

func (g *GeminiBackend) Analyze(req burp.BurpRequest) (*AIPlan, error) {
	if g.APIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

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

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.Model, g.APIKey)

	var userInstructions string
	if req.UserContext != "" {
		userInstructions = fmt.Sprintf("\n### SPECIAL USER INSTRUCTIONS (PRIORITY):\n%s\n", req.UserContext)
	}

	prompt := fmt.Sprintf(`You are an expert penetration tester. Analyze the following HTTP request (and response if provided) for vulnerabilities.
%s
Method: %s
URL: %s
Source: %s
Headers:
%s
Body: %s
%s
### User Training & Expertise:
%s

### Available Security Tools (MCP-Interface):
%s

### Instructions:
1. **Analyze Request**: Carefully inspect headers and the body for sensitive data or injection points. Check for interesting headers like Authorization, Cookies, or custom headers. Use the provided "User Training" to guide your analysis.
   - **BE SELECTIVE**: Do not recommend tools for general "recon" if the request looks benign. 
   - **CRITICAL REASONING**: Only recommend tools if you see strong, clear evidence of a specific vulnerability class. Avoid "just-in-case" scanning.
   - **Selective Fuzzing**: If the request is a simple GET with no parameters, or a POST with a static/irrelevant body, DO NOT trigger parameter fuzzing (ffuf, dalfox with parameters) unless there's a specific reason. Avoid "waste of time" scans on obviously static endpoints.
   - **GraphQL/API**: Prioritize targeted checks for these endpoints rather than generic fuzzing.
2. **Rules for Tool Selection**:
   - API/GraphQL: If the URL contains '/api/' or 'graphql', DO NOT use web crawlers. Use nuclei or ffuf instead.
   - Parameters: If parameters are detected, use dalfox or sqlmap.

3. **Stealth and Rate Limiting**:
   - YOU MUST include a "rate_limit" parameter in "params" for high-volume tools (ffuf, katana, nuclei).
   - Use "5" as a standard safe value for req/s.
   - FAILURE to provide a rate limit will result in system blocks.

4. **Output Format**: Output ONLY a JSON object with this exact structure:
{
  "thinking": "Deep, critical reasoning for why specific tools are (or are NOT) necessary. THIS FIELD MUST NOT BE EMPTY.",
  "vulnerabilities_suspected": ["Specific Pattern Name"],
  "actions": [{"tool": "toolname", "target": "string", "params": {"key": "val"}}],
  "rewrite_rules": ["modified_body_base64_string_if_needed"]
}
No preamble, no markdown formatting. Just raw JSON.`, userInstructions, req.Method, req.URL, req.Tool, string(headersJSON), bodyStr, responseContext, userKnowledge, toolContext)

	localUtils.Logger(fmt.Sprintf("[Gemini DEBUG] Outgoing Prompt (Truncated 500 chars): %s...", prompt[:500]), 3)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"response_mime_type": "application/json",
		},
	}

	jsonData, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 30 * time.Second}
	
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		localUtils.Logger(fmt.Sprintf("[Gemini DEBUG] Network Error: %v", err), 2)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	localUtils.Logger(fmt.Sprintf("[Gemini DEBUG] Status: %d, Raw Response: %s", resp.StatusCode, string(body)), 3)
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("gemini api error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no candidates returned from Gemini")
	}

	rawJSON := result.Candidates[0].Content.Parts[0].Text
	var plan AIPlan
	if err := json.Unmarshal([]byte(rawJSON), &plan); err != nil {
		localUtils.Logger(fmt.Sprintf("[DEBUG] Gemini invalid JSON: %s", rawJSON), 3)
		return nil, fmt.Errorf("failed to parse AI plan: %v", err)
	}

	return &plan, nil
}
