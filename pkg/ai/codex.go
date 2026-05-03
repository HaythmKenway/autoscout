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

func (c *CodexBackend) Name() string {
	return "Codex"
}

func (c *CodexBackend) Analyze(req burp.BurpRequest) (*AIPlan, error) {
	decodedBody, _ := base64.StdEncoding.DecodeString(req.Body)
	bodyStr := string(decodedBody)
	if bodyStr == "" {
		bodyStr = "[Empty Body]"
	}

	// Use full headers for maximum context
	headersJSON, _ := json.MarshalIndent(req.Headers, "", "  ")
	userKnowledge := LoadKnowledge()

	prompt := fmt.Sprintf(`You are a world-class penetration tester. Perform a deep security analysis on this HTTP request.
ALL headers are provided below - do not ignore them as they may contain session tokens, custom security headers, or injection points.

### HTTP REQUEST DATA
Method: %s
URL: %s
Source: %s

[FULL HEADERS]
%s

[BODY]
%s

### USER-SPECIFIC KNOWLEDGE
%s

### MANDATORY ANALYSIS GUIDELINES
1. **Header Analysis**: Deeply inspect every header (Authorization, Cookies, X-Forwarded-For, etc.) for misconfigurations or vulnerabilities like IDOR, session fixation, or header injection.
2. **Selective Tooling**: Trigger specific tools ONLY if relevant to the request type. 
   - Use 'sqlmap' if parameters or JSON bodies are present.
   - Use 'dalfox' for reflected input.
   - Use 'nuclei' for known vulnerability templates on APIs.
3. **Safety**: ALWAYS include "rate_limit" in tool params. Default to "5" for high-traffic targets.

### RESPONSE SPECIFICATION
Output ONLY raw JSON. No markdown, no preamble.
{
  "thinking": "Concise security reasoning.",
  "vulnerabilities_suspected": ["List suspected flaws"],
  "actions": [{"tool": "name", "target": "url", "params": {"key": "val"}}],
  "rewrite_rules": ["Optional: base64 encoded modified body"]
}
`, req.Method, req.URL, req.Tool, string(headersJSON), bodyStr, userKnowledge)

	// Use codex exec --json --ephemeral
	// Added --ignore-user-config and --ignore-rules to ensure environment consistency
	// Added "-" to explicitly read from stdin and suppress "Reading prompt from stdin..." message
	args := []string{
		"exec",
		"--json",
		"--ephemeral",
		"--skip-git-repo-check",
		"--dangerously-bypass-approvals-and-sandbox",
		"--ignore-user-config",
		"--ignore-rules",
		"-",
	}
	if c.Model != "" {
		args = append(args, "--model", c.Model)
	}

	cmd := exec.Command("codex", args...)
	cmd.Stdin = strings.NewReader(prompt)
	localUtils.Logger(fmt.Sprintf("[DEBUG] Codex Prompt length: %d (Piped to Stdin with -)", len(prompt)), 3)
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, _ := cmd.StderrPipe()

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
				// Continue scanning to drain pipe, but we have our result
			}
		}
	}

	errScanner := bufio.NewScanner(stderr)
	var errLines []string
	for errScanner.Scan() {
		line := errScanner.Text()
		// Filter out the "Reading prompt from stdin..." or similar informational messages
		if !strings.Contains(line, "Reading") && !strings.Contains(line, "prompt") {
			errLines = append(errLines, line)
		}
	}

	cmdErr := cmd.Wait()
	if cmdErr != nil {
		errMsg := strings.Join(errLines, " | ")
		// If we have rawJSON, it might have actually succeeded despite a non-zero exit (e.g. sandbox warning)
		if rawJSON == "" {
			localUtils.Logger(fmt.Sprintf("[DEBUG] Codex Error Output: %s", errMsg), 3)
			return nil, fmt.Errorf("codex execution failed (%v): %s", cmdErr, errMsg)
		}
		localUtils.Logger(fmt.Sprintf("[DEBUG] Codex exited with error but returned JSON: %v", cmdErr), 3)
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
