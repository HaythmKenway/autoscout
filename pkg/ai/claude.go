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

type ClaudeBackend struct {
	APIKey string
	Model  string
}

func NewClaudeBackend(apiKey, model string) *ClaudeBackend {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &ClaudeBackend{APIKey: apiKey, Model: model}
}

func (c *ClaudeBackend) Name() string {
	return "Claude (" + c.Model + ")"
}

// aiPlanToolSchema is the tool use schema that mirrors AIPlan.
// Forcing Claude to call this tool guarantees valid structured output on every request.
var aiPlanToolSchema = map[string]interface{}{
	"name":        "submit_analysis_plan",
	"description": "Submit the security analysis result as a structured plan.",
	"input_schema": map[string]interface{}{
		"type":     "object",
		"required": []string{"thinking", "vulnerabilities_suspected", "actions", "rewrite_rules"},
		"properties": map[string]interface{}{
			"thinking": map[string]interface{}{
				"type":        "string",
				"description": "Deep critical reasoning for why specific tools are (or are NOT) necessary. Must not be empty.",
			},
			"vulnerabilities_suspected": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"actions": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type":     "object",
					"required": []string{"tool", "target", "params"},
					"properties": map[string]interface{}{
						"tool":   map[string]interface{}{"type": "string"},
						"target": map[string]interface{}{"type": "string"},
						"params": map[string]interface{}{
							"type":                 "object",
							"additionalProperties": true,
						},
					},
				},
			},
			"rewrite_rules": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
		},
	},
}

func (c *ClaudeBackend) Analyze(req burp.BurpRequest) (*AIPlan, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	decodedBody, _ := base64.StdEncoding.DecodeString(req.Body)
	bodyStr := TruncateBody(string(decodedBody), 10000)
	if bodyStr == "" {
		bodyStr = "[Empty Body]"
	}

	headersJSON, _ := json.MarshalIndent(req.Headers, "", "  ")
	userKnowledge := LoadKnowledge()
	userSkills := LoadSkills()
	toolContext := tools.GetToolCapabilitiesJSON()

	var responseContext string
	if req.ResponseStatus > 0 {
		resHeadersJSON, _ := json.MarshalIndent(req.ResponseHeaders, "", "  ")
		decodedResBody, _ := base64.StdEncoding.DecodeString(req.ResponseBody)
		resBodyStr := TruncateBody(string(decodedResBody), 10000)
		if resBodyStr == "" {
			resBodyStr = "[Empty Response Body]"
		}
		responseContext = fmt.Sprintf("\n### RESPONSE DATA\nStatus: %d\nHeaders:\n%s\nBody: %s\n",
			req.ResponseStatus, string(resHeadersJSON), resBodyStr)
	}

	var userInstructions string
	if req.UserContext != "" {
		userInstructions = fmt.Sprintf("\n### SPECIAL USER INSTRUCTIONS (PRIORITY):\n%s\n", req.UserContext)
	}

	// Static system content is marked for prompt caching — sent once, reused across
	// repeated analyses in the same 5-minute cache window.
	systemText := fmt.Sprintf(`You are a senior penetration tester analyzing intercepted web traffic for security vulnerabilities.

### Instructions:
1. Carefully inspect headers and body for injection points, authentication flaws, and logical vulnerabilities.
2. BE SELECTIVE: Only recommend tools if you see strong, clear evidence of a specific vulnerability class. Avoid "just-in-case" scans.
3. STEALTH: Always include a "rate_limit" parameter (default "5" req/s) for high-volume tools (ffuf, katana, nuclei).
4. API/GraphQL: If the URL contains '/api/' or 'graphql', prefer nuclei over crawlers.
5. Static assets (.js, .css, images): Do NOT trigger any scanning tools.
6. Call the submit_analysis_plan tool with your findings. The "thinking" field must always be populated.

### User Training & Knowledge Base:
%s

### User Skills & Playbooks:
%s

### Available Security Tools:
%s`, userKnowledge, userSkills, toolContext)

	// Per-request user message with the dynamic HTTP data.
	userText := fmt.Sprintf(`%sAnalyze this HTTP request:

Method: %s
URL: %s
Source: %s
Headers:
%s
Body: %s%s`,
		userInstructions, req.Method, req.URL, req.Tool,
		string(headersJSON), bodyStr, responseContext)

	payload := map[string]interface{}{
		"model":      c.Model,
		"max_tokens": 2048,
		"system": []map[string]interface{}{
			{
				"type": "text",
				"text": systemText,
				// Cache the large static system prompt across repeated analyses.
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		},
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "text", "text": userText},
				},
			},
		},
		"tools":       []map[string]interface{}{aiPlanToolSchema},
		"tool_choice": map[string]string{"type": "any"},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("anthropic-beta", "prompt-caching-2024-07-31")
	httpReq.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("claude api request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	localUtils.Logger(fmt.Sprintf("[Claude DEBUG] Status: %d", resp.StatusCode), 3)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("claude api error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Content []struct {
			Type  string          `json:"type"`
			Name  string          `json:"name,omitempty"`
			Input json.RawMessage `json:"input,omitempty"`
		} `json:"content"`
		Usage struct {
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode claude response: %v", err)
	}

	for _, block := range result.Content {
		if block.Type == "tool_use" && block.Name == "submit_analysis_plan" {
			var plan AIPlan
			if err := json.Unmarshal(block.Input, &plan); err != nil {
				return nil, fmt.Errorf("failed to parse analysis plan from Claude: %v", err)
			}
			localUtils.Logger(fmt.Sprintf("[Claude] Analysis complete for %s (cache: %d read / %d created tokens)",
				req.URL, result.Usage.CacheReadInputTokens, result.Usage.CacheCreationInputTokens), 1)
			return &plan, nil
		}
	}

	return nil, fmt.Errorf("claude did not call submit_analysis_plan (raw: %s)", string(body))
}
