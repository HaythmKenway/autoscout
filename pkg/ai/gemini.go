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

func (g *GeminiBackend) Analyze(req burp.BurpRequest) (*AIPlan, error) {
	if g.APIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	decodedBody, _ := base64.StdEncoding.DecodeString(req.Body)
	bodyStr := string(decodedBody)
	if bodyStr == "" {
		bodyStr = "[Empty Body]"
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.Model, g.APIKey)

	prompt := fmt.Sprintf(`You are an expert penetration tester. Analyze the following HTTP request for vulnerabilities.
Method: %s
URL: %s
Source: %s
Body: %s

### Instructions:
1. **Analyze Body**: Carefully inspect the POST body or parameters for sensitive data or injection points.
2. **Rules for Tool Selection**:
   - API/GraphQL: If the URL contains '/api/' or 'graphql', DO NOT use web crawlers. Use nuclei or ffuf instead.
   - Parameters: If parameters are detected, use dalfox or sqlmap.

3. **Output Format**: Output ONLY a JSON object with this exact structure:
{
  "thinking": "Your detailed reasoning here.",
  "vulnerabilities_suspected": ["type1", "type2"],
  "actions": [{"tool": "dalfox|sqlmap|nuclei|ffuf|arjun|katana|gospider|censys", "target": "string", "params": {"key": "val"}}],
  "rewrite_rules": ["modified_body_base64_string_if_needed"]
}
No preamble, no markdown formatting. Just raw JSON.`, req.Method, req.URL, req.Tool, bodyStr)

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
