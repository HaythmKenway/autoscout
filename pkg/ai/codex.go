package ai

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/HaythmKenway/autoscout/pkg/burp"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

type CodexBackend struct {
	Model string
}

func NewCodexBackend(model string) *CodexBackend {
	return &CodexBackend{Model: model}
}

func (c *CodexBackend) Analyze(req burp.BurpRequest) (*AIPlan, error) {
	decodedBody, _ := base64.StdEncoding.DecodeString(req.Body)
	bodyStr := string(decodedBody)
	if bodyStr == "" {
		bodyStr = "[Empty Body]"
	}

	headersJSON, _ := json.MarshalIndent(req.Headers, "", "  ")
	userKnowledge := LoadKnowledge()

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

	// Use codex exec --json --ephemeral
	args := []string{"exec", "--json", "--ephemeral", "--skip-git-repo-check", "--ask-for-approval", "never"}
	if c.Model != "" {
		args = append(args, "--model", c.Model)
	}
	args = append(args, prompt)

	cmd := exec.Command("codex", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start codex: %v", err)
	}

	var rawJSON string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		var event struct {
			Type string `json:"type"`
			Item struct {
				Text string `json:"text"`
			} `json:"item"`
		}
		if err := json.Unmarshal([]byte(line), &event); err == nil {
			if event.Type == "item.completed" && event.Item.Text != "" {
				rawJSON = event.Item.Text
				break
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("codex execution failed: %v", err)
	}

	if rawJSON == "" {
		return nil, fmt.Errorf("codex returned no agent message")
	}

	// Sometimes LLMs wrap JSON in backticks
	rawJSON = strings.TrimPrefix(rawJSON, "```json")
	rawJSON = strings.TrimPrefix(rawJSON, "```")
	rawJSON = strings.TrimSuffix(rawJSON, "```")
	rawJSON = strings.TrimSpace(rawJSON)

	var plan AIPlan
	if err := json.Unmarshal([]byte(rawJSON), &plan); err != nil {
		localUtils.Logger(fmt.Sprintf("[DEBUG] Codex invalid JSON: %s", rawJSON), 3)
		return nil, fmt.Errorf("failed to parse AI plan: %v", err)
	}

	return &plan, nil
}
