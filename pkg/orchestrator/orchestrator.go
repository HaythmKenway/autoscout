package orchestrator

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

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
	
	decodedBody, _ := base64.StdEncoding.DecodeString(req.Body)
	bodyStr := string(decodedBody)

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
		
		rateLimit, _ := action.Params["rate_limit"].(string)

		switch action.Tool {
		case "dalfox":
			go tools.RunDalfox(action.Target, req.Method, bodyStr, req.Headers)
		case "sqlmap":
			// Reconstruct a proper raw request for sqlmap
			u, _ := url.Parse(action.Target)
			host := "target"
			path := action.Target
			if u != nil && u.Host != "" {
				host = u.Host
				path = u.RequestURI()
			}
			
			var headerStr strings.Builder
			for k, v := range req.Headers {
				headerStr.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
			}
			
			raw := fmt.Sprintf("%s %s HTTP/1.1\r\nHost: %s\r\n%s\r\n%s", req.Method, path, host, headerStr.String(), bodyStr)
			go tools.RunSQLMap(action.Target, raw)
		case "nuclei":
			tags, _ := action.Params["tags"].(string)
			go tools.RunNuclei(action.Target, tags, rateLimit)
		case "katana":
			go tools.RunKatana(action.Target, rateLimit)
		case "ffuf":
			go tools.RunFFUF(action.Target, req.Method, bodyStr, req.Headers, rateLimit)
		case "arjun":
			go tools.RunArjun(action.Target, req.Method, bodyStr, req.Headers)
		case "gospider":
			go tools.RunGoSpider(action.Target, rateLimit)
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
