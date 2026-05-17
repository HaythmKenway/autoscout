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

type CodexBackend struct {
	Model string
}

func NewCodexBackend(model string) *CodexBackend {
	// The model parameter is ignored as we use the system 'codex' binary's 
	// default configuration (OpenAI/ChatGPT)
	return &CodexBackend{
		Model: "System-Codex",
	}
}

func (c *CodexBackend) Name() string {
	return "Codex (System Binary)"
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

### AVAILABLE SECURITY TOOLS (MCP-INTERFACE)
The following JSON defines the available tools and their accepted parameters. Use this to construct your "actions".

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
`, userInstructions, req.Method, req.URL, req.Tool, string(headersJSON), bodyStr, responseContext, userKnowledge, userSkills, toolContext)

	// Use context with timeout for codex command
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Use system 'codex' binary
	cmd := exec.CommandContext(ctx, "codex", "exec", "--ephemeral", "--json", "-")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start codex: %v", err)
	}

	// Write prompt and close stdin to signal EOF
	go func() {
		defer stdin.Close()
		fmt.Fprint(stdin, prompt)
	}()

	var finalResponse string
	var codexError string

	// Scanner to read JSONL output
	go func() {
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
					finalResponse = event.Item.Text
				}
			}
		}
	}()

	// Scanner for errors
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			codexError += scanner.Text()
		}
	}()

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("codex error: %v (stderr: %s)", err, codexError)
	}

	if finalResponse == "" {
		return nil, fmt.Errorf("codex returned empty response (stderr: %s)", codexError)
	}

	// Attempt to extract JSON if there was conversational fluff
	rawJSON := finalResponse
	if start := strings.Index(rawJSON, "{"); start != -1 {
		if end := strings.LastIndex(rawJSON, "}"); end != -1 && end > start {
			rawJSON = rawJSON[start : end+1]
		}
	}

	var plan AIPlan
	if err := json.Unmarshal([]byte(rawJSON), &plan); err != nil {
		localUtils.Logger(fmt.Sprintf("[DEBUG] Codex invalid JSON: %s", rawJSON), 3)
		return nil, fmt.Errorf("failed to parse AI plan: %v", err)
	}

	localUtils.Logger(fmt.Sprintf("[Codex Agent] Analysis complete for %s", req.URL), 1)
	return &plan, nil
}
