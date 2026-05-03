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
)

type GeminiBackend struct {
	APIKey string
	Model  string
}

func NewGeminiBackend(apiKey, model string) *GeminiBackend {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if model == "" {
		model = "gemini-1.5-flash"
	}
	return &GeminiBackend{APIKey: apiKey, Model: model}
}

func (g *GeminiBackend) Name() string {
	return "Gemini"
}

func (g *GeminiBackend) Analyze(req burp.BurpRequest) (*AIPlan, error) {
	if g.APIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	decodedBody, _ := base64.StdEncoding.DecodeString(req.Body)
	bodyStr := string(decodedBody)
	if bodyStr == "" {
		bodyStr = "[Empty Body]"
	}

	headersJSON, _ := json.MarshalIndent(req.Headers, "", "  ")
	userKnowledge := LoadKnowledge()

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.Model, g.APIKey)

	prompt := fmt.Sprintf(`You are an expert penetration tester. Analyze the following HTTP request for vulnerabilities.
Method: %s
URL: %s
Source: %s
Headers:
%s
Body: %s

### User Training & Expertise:
%s

### Instructions:
1. **Analyze Request**: Carefully inspect headers and the body for sensitive data or injection points. Check for interesting headers like Authorization, Cookies, or custom headers. Use the provided "User Training" to guide your analysis.
   - **Selective Fuzzing**: If the request is a simple GET with no parameters, or a POST with a static/irrelevant body, DO NOT trigger parameter fuzzing (ffuf, dalfox with parameters) unless there's a specific reason. Avoid "waste of time" scans on obviously static endpoints.
   - **GraphQL/API**: Prioritize targeted checks for these endpoints rather than generic fuzzing.
2. **Rules for Tool Selection**:
   - API/GraphQL: If the URL contains '/api/' or 'graphql', DO NOT use web crawlers. Use nuclei or ffuf instead.
   - Parameters: If parameters are detected, use dalfox or sqlmap.

3. **Stealth and Rate Limiting**:
   - ALWAYS include a "rate_limit" parameter in "params" for high-volume tools (ffuf, katana, gospider, nuclei).
   - If the target is a major platform (e.g., reddit, google, github), set "rate_limit" to a low value (e.g., 5-10 requests per second) to avoid blocking.
   - Example for FFUF on a sensitive target: {"tool": "ffuf", "target": "URL", "params": {"rate_limit": "5"}}

4. **Output Format**: Output ONLY a JSON object with this exact structure:
{
  "thinking": "Your detailed reasoning here.",
  "vulnerabilities_suspected": ["type1", "type2"],
  "actions": [{"tool": "toolname", "target": "string", "params": {"key": "val"}}],
  "rewrite_rules": ["modified_body_base64_string_if_needed"]
}
No preamble, no markdown formatting. Just raw JSON.`, req.Method, req.URL, req.Tool, string(headersJSON), bodyStr, userKnowledge)

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
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
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
		return nil, fmt.Errorf("gemini returned empty result")
	}

	rawJSON := result.Candidates[0].Content.Parts[0].Text
	var plan AIPlan
	if err := json.Unmarshal([]byte(rawJSON), &plan); err != nil {
		localUtils.Logger(fmt.Sprintf("[DEBUG] Gemini invalid JSON: %s", rawJSON), 3)
		return nil, fmt.Errorf("failed to parse AI plan: %v", err)
	}

	return &plan, nil
}
