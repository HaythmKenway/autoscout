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
	bodyStr := TruncateBody(string(decodedBody), 10000)
	if bodyStr == "" {
		bodyStr = "[Empty Body]"
	}

	// Use full headers for maximum context
	headersJSON, _ := json.MarshalIndent(req.Headers, "", "  ")
	userKnowledge := LoadKnowledge()
	userSkills := LoadSkills()

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

	prompt := fmt.Sprintf(`You are a senior security researcher analyzing web traffic.
Analyze the following HTTP request (and response if provided) for potential security patterns and provide a structured assessment.
%s
### TARGET DATA
Method: %s
URL: %s
Context: %s

[HEADERS]
%s

[DATA]
%s
%s
### REFERENCE KNOWLEDGE
%s

### AUTOSCOUT SKILLS
The following Markdown skills are user-authored playbooks. Follow them when they match the target data, and ignore irrelevant skills.

%s

### TASK
Provide a structured JSON assessment. Do not execute any tools yourself. 
Be highly selective. ONLY recommend tools if you see strong, clear evidence of a specific vulnerability class. 
Avoid over-scanning; do not recommend tools for general "recon" if the request looks benign. 
**Ensure the 'thinking' field is ALWAYS populated with human-readable reasoning**, even if it's to explain why no tools are being triggered.
If specific tools are recommended for further automated verification, list them in the "actions" field.

OUTPUT FORMAT (RAW JSON ONLY):
{
  "thinking": "Deep, critical reasoning for why specific tools are (or are NOT) necessary. THIS FIELD MUST NOT BE EMPTY.",
  "vulnerabilities_suspected": ["Specific Pattern Name"],
  "actions": [{"tool": "nuclei|sqlmap|dalfox|ffuf", "target": "url", "params": {"rate_limit": "5"}}],
  "rewrite_rules": ["<base64_modified_payload>"]
}
`, userInstructions, req.Method, req.URL, req.Tool, string(headersJSON), bodyStr, responseContext, userKnowledge, userSkills)

	// Use context with timeout for codex command
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

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

	cmd := exec.CommandContext(ctx, "codex", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start codex: %v", err)
	}

	// Write prompt and close stdin to signal EOF
	go func() {
		defer stdin.Close()
		localUtils.Logger(fmt.Sprintf("[Codex DEBUG] Sending prompt (Log truncated to 500 chars): %s...", prompt[:min(len(prompt), 500)]), 3)
		fmt.Fprint(stdin, prompt)
	}()

	var rawJSON string
	var codexError string
	scanner := bufio.NewScanner(stdout)
	// Use a larger buffer (1MB) for Codex output
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		localUtils.Logger(fmt.Sprintf("[Codex DEBUG] Incoming Stream: %s", line), 3)
		var event struct {
			Type string `json:"type"`
			Item struct {
				Text string `json:"text"`
			} `json:"item"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &event); err == nil {
			if event.Type == "item.completed" && event.Item.Text != "" {
				rawJSON = event.Item.Text
			} else if event.Type == "error" {
				codexError = event.Error.Message
			}
		}
	}

	var stderrBuf strings.Builder
	errScanner := bufio.NewScanner(stderr)
	for errScanner.Scan() {
		line := errScanner.Text()
		stderrBuf.WriteString(line + "\n")
	}

	cmdErr := cmd.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		localUtils.Logger("[Codex ERROR] Command timed out after 60s", 2)
		return nil, fmt.Errorf("codex timed out")
	}

	if cmdErr != nil || codexError != "" {
		errMsg := stderrBuf.String()
		if codexError != "" {
			errMsg = fmt.Sprintf("Codex Event Error: %s | Raw Stderr: %s", codexError, errMsg)
		}
		// If we have rawJSON, it might have actually succeeded despite a non-zero exit
		if rawJSON == "" {
			localUtils.Logger(fmt.Sprintf("[DEBUG] Codex Failed (%v). Stderr: %s", cmdErr, errMsg), 3)
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

	localUtils.Logger(fmt.Sprintf("[Codex Agent] Analysis complete for %s", req.URL), 1)
	return &plan, nil
}
