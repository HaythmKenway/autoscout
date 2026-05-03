package orchestrator

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/HaythmKenway/autoscout/pkg/ai"
	"github.com/HaythmKenway/autoscout/pkg/burp"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/HaythmKenway/autoscout/pkg/tools"
)

type Orchestrator struct {
	Agent     ai.AIAgent
	Requests  chan burp.BurpRequest
	history   map[string]time.Time
	historyMu sync.Mutex
}

func NewOrchestrator(agent ai.AIAgent) *Orchestrator {
	return &Orchestrator{
		Agent:    agent,
		Requests: make(chan burp.BurpRequest, 100),
		history:  make(map[string]time.Time),
	}
}

func (o *Orchestrator) Start() {
	localUtils.Logger("AI Orchestrator started", 1)
	for req := range o.Requests {
		go o.processRequest(req)
	}
}

func (o *Orchestrator) shouldRun(tool, target string) bool {
	o.historyMu.Lock()
	defer o.historyMu.Unlock()

	key := fmt.Sprintf("%s:%s", tool, target)
	lastRun, exists := o.history[key]
	if exists && time.Since(lastRun) < 5*time.Minute {
		return false
	}
	o.history[key] = time.Now()
	return true
}

func (o *Orchestrator) processRequest(req burp.BurpRequest) {
	agentName := o.Agent.Name()
	localUtils.Logger(fmt.Sprintf("[%s Agent] Analyzing request: %s", agentName, req.URL), 1)
	
	decodedBody, _ := base64.StdEncoding.DecodeString(req.Body)
	bodyStr := string(decodedBody)

	plan, err := o.Agent.Analyze(req)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("[%s Agent] Analysis failed: %v", agentName, err), 2)
		return
	}

	// Log AI Thinking
	if plan.Thinking != "" {
		addAnalysis(fmt.Sprintf("[%s] THINKING: %s", agentName, plan.Thinking))
	}

	for _, action := range plan.Actions {
		if !o.shouldRun(action.Tool, action.Target) {
			localUtils.Logger(fmt.Sprintf("[%s] Skipping redundant tool: %s on %s", agentName, action.Tool, action.Target), 1)
			continue
		}

		localUtils.Logger(fmt.Sprintf("[%s] Triggering tool: %s on %s", agentName, action.Tool, action.Target), 1)
		
		rateLimit, _ := action.Params["rate_limit"].(string)
		if rateLimit == "" {
			localUtils.Logger(fmt.Sprintf("[%s] Provided no rate limit for %s. Defaulting to 5.", agentName, action.Tool), 2)
			rateLimit = "5" // Default safety rate limit
		} else {
			localUtils.Logger(fmt.Sprintf("[%s] AI-decided rate limit for %s: %s", agentName, action.Tool, rateLimit), 1)
		}

		switch action.Tool {
		case "dalfox":
			go tools.RunDalfox(action.Target, req.Method, bodyStr, req.Headers, rateLimit)
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
			go tools.RunSQLMap(action.Target, raw, rateLimit)
		case "nuclei":
			tags, _ := action.Params["tags"].(string)
			go tools.RunNuclei(action.Target, tags, rateLimit)
		case "katana":
			go tools.RunKatana(action.Target, rateLimit)
		case "ffuf":
			go tools.RunFFUF(action.Target, req.Method, bodyStr, req.Headers, rateLimit)
		case "arjun":
			go tools.RunArjun(action.Target, req.Method, bodyStr, req.Headers, rateLimit)
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
