package ai

import "github.com/HaythmKenway/autoscout/pkg/burp"

// AIAction defines a specific tool execution decided by the AI
type AIAction struct {
	Tool   string            `json:"tool"`
	Target string            `json:"target"`
	Params map[string]string `json:"params"`
}

// AIPlan is the structured output from the LLM
type AIPlan struct {
	VulnerabilitiesSuspected []string   `json:"vulnerabilities_suspected"`
	Actions                  []AIAction `json:"actions"`
	RewriteRules             []string   `json:"rewrite_rules"`
}

// AIAgent is the interface for different AI providers (Ollama, Gemini, etc.)
type AIAgent interface {
	Analyze(req burp.BurpRequest) (*AIPlan, error)
}
