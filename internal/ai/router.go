package ai

import (
	"fmt"
	"strings"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/HaythmKenway/autoscout/pkg/proxy"
)

func StartRouter() {
	localUtils.Logger("AI Agent Router started", 1)
	for req := range proxy.RequestQueue {
		routeRequest(req)
	}
}

func routeRequest(req proxy.InterceptedRequest) {
	// Heuristic 1: Filter out static assets
	if isStaticAsset(req.URL) {
		return
	}

	localUtils.Logger(fmt.Sprintf("[AI Router] Analyzing: %s %s", req.Method, req.URL), 1)

	// Heuristic 2: Check for parameters (Potential SQLi/XSS)
	if strings.Contains(req.URL, "?") || len(req.Body) > 0 {
		delegateToAgent("ParameterFuzzer", req)
	}

	// Heuristic 3: Check for JSON/API (Potential IDOR/BOLA)
	if strings.Contains(req.URL, "/api/") || strings.Contains(req.Headers["Content-Type"][0], "json") {
		delegateToAgent("APIAnalyzer", req)
	}

	// Heuristic 4: Check for Sensitive Info in Responses
	if req.ResponseCode == 200 && len(req.ResponseBody) > 0 {
		delegateToAgent("InfoLeakScanner", req)
	}
}

func isStaticAsset(u string) bool {
	extensions := []string{".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".woff", ".woff2", ".ttf", ".ico"}
	for _, ext := range extensions {
		if strings.HasSuffix(strings.Split(u, "?")[0], ext) {
			return true
		}
	}
	return false
}

func delegateToAgent(agentName string, req proxy.InterceptedRequest) {
	// SIMULATION: In Phase 3, this will call the actual LLM
	localUtils.Logger(fmt.Sprintf("[AI Fleet] %s is investigating %s", agentName, req.URL), 1)

	// Simulated logic for POC demonstration
	if agentName == "InfoLeakScanner" {
		body := string(req.ResponseBody)
		if strings.Contains(body, "AWS_ACCESS_KEY") || strings.Contains(body, "API_KEY") {
			localUtils.Logger(fmt.Sprintf("[AI CRITICAL] %s found a potential leak at %s", agentName, req.URL), 1)
		}
	}

	if agentName == "ParameterFuzzer" {
		if strings.Contains(req.URL, "id=") || strings.Contains(req.URL, "user=") {
			localUtils.Logger(fmt.Sprintf("[AI ALERT] %s detected sensitive parameters at %s", agentName, req.URL), 1)
		}
	}
}
