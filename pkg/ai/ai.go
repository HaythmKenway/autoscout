package ai

import (
	"os"

	"github.com/HaythmKenway/autoscout/pkg/burp"
	"gopkg.in/yaml.v3"
)

// AIAction defines a specific tool execution decided by the AI
type AIAction struct {
	Tool   string                 `json:"tool"`
	Target string                 `json:"target"`
	Params map[string]interface{} `json:"params"`
}

// AIPlan is the structured output from the LLM
type AIPlan struct {
	Thinking                 string     `json:"thinking"`
	VulnerabilitiesSuspected []string   `json:"vulnerabilities_suspected"`
	Actions                  []AIAction `json:"actions"`
	RewriteRules             []string   `json:"rewrite_rules"`
}

// AIAgent is the interface for different AI providers (Ollama, Gemini, etc.)
type AIAgent interface {
	Analyze(req burp.BurpRequest) (*AIPlan, error)
	Name() string
}

// LoadKnowledge reads the custom user knowledge/training file
func LoadKnowledge() string {
	knowledgePath := os.ExpandEnv("$HOME/.config/autoscout/knowledge.md")
	data, err := os.ReadFile(knowledgePath)
	if err != nil {
		// If file doesn't exist, create a default one with instructions
		defaultKnowledge := "### User Training & Knowledge Base\n" +
			"- Look for IDOR in /api/v1/user/ settings\n" +
			"- Check for BOLA in UUID parameters\n" +
			"- Inspect 'state' parameters for OAuth misconfigurations\n"

		configDir := os.ExpandEnv("$HOME/.config/autoscout")
		os.MkdirAll(configDir, 0755)
		os.WriteFile(knowledgePath, []byte(defaultKnowledge), 0644)
		return defaultKnowledge
	}
	return string(data)
}

// LoadBackend reads the user-config.yaml and returns the chosen AI backend
func LoadBackend() AIAgent {
	settingsPath := os.ExpandEnv("$HOME/.config/autoscout/user-config.yaml")
	data, err := os.ReadFile(settingsPath)

	agentType := "ClaudeCode" // Default: uses Claude Code subscription, no API key needed
	if err == nil {
		var config struct {
			Settings struct {
				Agent string `yaml:"agent"`
			} `yaml:"settings"`
		}
		yaml.Unmarshal(data, &config)
		if config.Settings.Agent != "" {
			agentType = config.Settings.Agent
		}
	}

	switch agentType {
	case "ClaudeCode":
		return NewClaudeCodeBackend("")
	case "Claude":
		return NewClaudeBackend("", "claude-sonnet-4-6")
	case "Gemini":
		return NewGeminiBackend("", "gemini-1.5-flash")
	case "Ollama":
		return NewOllamaBackend("")
	default: // "Codex" and legacy values
		return NewCodexBackend("")
	}
}

// TruncateBody limits the size of a string to avoid exceeding LLM context windows.
func TruncateBody(body string, maxLen int) string {
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "\n\n[... TRUNCATED DUE TO SIZE ...]"
}
