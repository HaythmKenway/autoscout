package ai

import (
	"bufio"
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
	prompt := fmt.Sprintf(`You are an expert penetration tester. Analyze the following HTTP request for vulnerabilities.
Method: %s
URL: %s
Source: %s
Body: %s

### Rules for Tool Selection:
1. **API/GraphQL**: If the URL contains '/api/' or 'graphql', DO NOT use web crawlers (gospider, katana). Use nuclei or ffuf instead.
2. **Parameters**: If parameters are detected, use dalfox (for XSS) or sqlmap (for SQLi).
3. **Censys**: Only use if you see an IP address or want to check for exposed services on a new domain.

Identify risks like SQLi, XSS, SSRF, IDOR, or Auth Bypass.
Output ONLY a JSON object with this exact structure:
{
  "vulnerabilities_suspected": ["type1", "type2"],
  "actions": [{"tool": "dalfox|sqlmap|nuclei|ffuf|arjun|katana|gospider|censys", "target": "string", "params": {"key": "val"}}],
  "rewrite_rules": ["modified_body_base64_string_if_needed"]
}
No preamble, no markdown formatting. Just raw JSON.`, req.Method, req.URL, req.Tool, req.Body)

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
