package orchestrator

import (
	"fmt"

	"github.com/HaythmKenway/autoscout/pkg/ai"
	"github.com/HaythmKenway/autoscout/pkg/burp"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/HaythmKenway/autoscout/pkg/tools"
)

type Orchestrator struct {
	Agent    ai.AIAgent
	Requests chan burp.BurpRequest
}

func NewOrchestrator(agent ai.AIAgent) *Orchestrator {
	return &Orchestrator{
		Agent:    agent,
		Requests: make(chan burp.BurpRequest, 100),
	}
}

func (o *Orchestrator) Start() {
	localUtils.Logger("AI Orchestrator started", 1)
	for req := range o.Requests {
		go o.processRequest(req)
	}
}

func (o *Orchestrator) processRequest(req burp.BurpRequest) {
	localUtils.Logger(fmt.Sprintf("[Plan Agent] Analyzing request: %s", req.URL), 1)
	
	plan, err := o.Agent.Analyze(req)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("[Plan Agent] Analysis failed: %v", err), 2)
		return
	}

	// Log AI Thinking
	if plan.Thinking != "" {
		addAnalysis(fmt.Sprintf("AI THINKING: %s", plan.Thinking))
	}

	for _, action := range plan.Actions {
		localUtils.Logger(fmt.Sprintf("[Orchestrator] Triggering tool: %s on %s", action.Tool, action.Target), 1)
		
		switch action.Tool {
		case "dalfox":
			go tools.RunDalfox(action.Target)
		case "sqlmap":
			// Reconstruct a simple raw request for sqlmap
			raw := fmt.Sprintf("%s %s HTTP/1.1\nHost: %s\n\n%s", req.Method, req.URL, "target", req.Body)
			go tools.RunSQLMap(action.Target, raw)
		case "nuclei":
			go tools.RunNuclei(action.Target, action.Params["tags"])
		case "katana":
			go tools.RunKatana(action.Target)
		case "ffuf":
			go tools.RunFFUF(action.Target)
		case "arjun":
			go tools.RunArjun(action.Target)
		case "gospider":
			go tools.RunGoSpider(action.Target)
		case "censys":
			go tools.RunCensys(action.Target)
		}
	}

	for _, rule := range plan.RewriteRules {
		localUtils.Logger(fmt.Sprintf("[Orchestrator] AI registered rewrite rule for: %s", req.URL), 1)
		burp.AddAnalysis("AI: Registered automatic rewrite for this target")
		burp.RegisterRewrite(req.URL, rule)
	}
}

func addAnalysis(msg string) {
	burp.AddAnalysis(msg)
}
