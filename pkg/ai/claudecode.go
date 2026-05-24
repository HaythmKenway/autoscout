package ai

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/HaythmKenway/autoscout/pkg/burp"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/HaythmKenway/autoscout/pkg/tools"
)

// ClaudeCodeBackend invokes the locally-installed `claude` CLI binary.
// It requires no Anthropic API key — it uses the user's existing Claude Code subscription.
type ClaudeCodeBackend struct {
	Model string
}

func NewClaudeCodeBackend(model string) *ClaudeCodeBackend {
	return &ClaudeCodeBackend{Model: model}
}

func (c *ClaudeCodeBackend) Name() string {
	if c.Model != "" {
		return "ClaudeCode (" + c.Model + ")"
	}
	return "ClaudeCode (subscription)"
}

// aiPlanJSONSchema is the --json-schema value used to constrain claude CLI output.
var aiPlanJSONSchema = map[string]interface{}{
	"type":     "object",
	"required": []string{"thinking", "vulnerabilities_suspected", "actions", "rewrite_rules"},
	"properties": map[string]interface{}{
		"thinking": map[string]interface{}{
			"type":        "string",
			"description": "Deep critical reasoning. Must not be empty.",
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
}

func (c *ClaudeCodeBackend) Analyze(req burp.BurpRequest) (*AIPlan, error) {
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

	prompt := fmt.Sprintf(`You are a senior penetration tester analyzing intercepted web traffic for security vulnerabilities.
%s
### REFERENCE KNOWLEDGE
%s

### AUTOSCOUT SKILLS
%s

### AVAILABLE SECURITY TOOLS
%s

### TARGET DATA
Method: %s
URL: %s
Source: %s

[HEADERS]
%s

[BODY]
%s
%s
### TASK
Be highly selective — only recommend tools if you see strong, clear evidence of a specific vulnerability class.
Always include a "rate_limit" parameter (default "5" req/s) for high-volume tools.
Output ONLY valid JSON matching the required schema. The "thinking" field must always be populated.`,
		userInstructions, userKnowledge, userSkills, toolContext,
		req.Method, req.URL, req.Tool, string(headersJSON), bodyStr, responseContext)

	schemaJSON, err := json.Marshal(aiPlanJSONSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json schema: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	args := []string{
		"-p",
		"--output-format", "json",
		"--json-schema", string(schemaJSON),
		"--no-session-persistence",
	}
	if c.Model != "" {
		args = append(args, "--model", c.Model)
	}

	cmd := exec.CommandContext(ctx, "claude", args...)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start claude CLI: %v (is Claude Code installed?)", err)
	}

	go func() {
		defer stdin.Close()
		fmt.Fprint(stdin, prompt)
	}()

	var stderrLines strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrLines.WriteString(scanner.Text() + "\n")
		}
	}()

	// The --output-format json envelope: {"type":"result","result":"<json>","is_error":false,...}
	var envelope struct {
		Type    string          `json:"type"`
		Result  json.RawMessage `json:"result"`
		IsError bool            `json:"is_error"`
	}
	if err := json.NewDecoder(stdout).Decode(&envelope); err != nil {
		cmd.Wait()
		return nil, fmt.Errorf("failed to decode claude CLI output: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("claude CLI error: %v (stderr: %s)", err, stderrLines.String())
	}

	if envelope.IsError {
		return nil, fmt.Errorf("claude CLI returned error: %s", string(envelope.Result))
	}

	// envelope.Result may be a JSON string (most common) or an inline JSON object.
	// Try direct unmarshal first, then unwrap as a quoted string.
	var plan AIPlan
	if err := json.Unmarshal(envelope.Result, &plan); err != nil {
		// Result is a JSON-encoded string; unwrap it.
		var resultStr string
		if jsonErr := json.Unmarshal(envelope.Result, &resultStr); jsonErr != nil {
			localUtils.Logger(fmt.Sprintf("[ClaudeCode DEBUG] Raw result: %s", string(envelope.Result)), 3)
			return nil, fmt.Errorf("failed to parse claude CLI result: %v", err)
		}
		// Extract JSON from the unwrapped string (handles any conversational wrapper).
		raw := resultStr
		if start := strings.Index(raw, "{"); start != -1 {
			if end := strings.LastIndex(raw, "}"); end > start {
				raw = raw[start : end+1]
			}
		}
		if err2 := json.Unmarshal([]byte(raw), &plan); err2 != nil {
			localUtils.Logger(fmt.Sprintf("[ClaudeCode DEBUG] Extracted JSON: %s", raw), 3)
			return nil, fmt.Errorf("failed to parse AI plan from claude CLI: %v", err2)
		}
	}

	localUtils.Logger(fmt.Sprintf("[ClaudeCode] Analysis complete for %s", req.URL), 1)
	return &plan, nil
}
