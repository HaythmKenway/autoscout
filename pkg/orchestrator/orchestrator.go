package orchestrator

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/internal/scheduler"
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
	localUtils.Logger("AI Orchestrator started [v2.2-log-fix]", 1)
	for req := range o.Requests {
		// Only process automated requests if the global scanner is running
		// Always process manual requests from Burp or the GUI
		if req.Tool == "MANUAL" || req.Tool == "MANUAL_GUI" || scheduler.IsRunning() {
			go o.processRequest(req)
		} else {
			localUtils.Logger(fmt.Sprintf("[Orchestrator] Skipping automated analysis for %s (Scanner is OFFLINE)", req.URL), 3)
		}
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
	// Auto-add target to database
	go func() {
		db.AddTarget(req.URL)

		// If it's a manual request with a response, we might also want to log it in urls table
		if req.Tool == "MANUAL" && req.ResponseStatus > 0 {
			database, err := db.OpenDatabase()
			if err == nil {
				defer database.Close()
				u, _ := url.Parse(req.URL)
				host := u.Hostname()
				// We don't have all the info for AddUrl (tech, etc.) but we can add what we have
				db.AddUrl(database, host, "", req.URL, host, u.Scheme, "", "", "", "", u.Port(), fmt.Sprintf("%d", req.ResponseStatus), "")
			}
		}
	}()

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

	actions := buildRoutes(req, bodyStr, plan)
	if len(actions) == 0 {
		localUtils.Logger(fmt.Sprintf("[%s] Analysis complete: No tools triggered for %s", agentName, req.URL), 1)
		// Log the AI's thinking process to the analysis panel if no actions were taken
		if plan.Thinking != "" {
			addAnalysis(fmt.Sprintf("[%s] ANALYSIS: %s", agentName, plan.Thinking))
		} else {
			// If thinking is empty, log a default message indicating no specific vulnerabilities were found
			addAnalysis(fmt.Sprintf("[%s] ANALYSIS: No specific vulnerabilities detected or tools warranted for this request.", agentName))
		}
		return
	}

	for _, action := range actions {
		if !tools.IsToolAvailable(action.Tool) {
			localUtils.Logger(fmt.Sprintf("[%s] Tool '%s' not found in PATH, skipping. Install it to enable this scan.", agentName, action.Tool), 2)
			continue
		}

		if !o.shouldRun(action.Tool, action.Target) {
			localUtils.Logger(fmt.Sprintf("[%s] Skipping redundant tool: %s on %s", agentName, action.Tool, action.Target), 1)
			continue
		}

		localUtils.Logger(fmt.Sprintf("[%s] Triggering tool: %s on %s", agentName, action.Tool, action.Target), 1)

		rateLimit, _ := action.Params["rate_limit"].(string)
		if rateLimit == "" {
			rateLimit = localUtils.GetRateLimit()
			localUtils.Logger(fmt.Sprintf("[%s] Provided no rate limit for %s. Using global default: %s.", agentName, action.Tool, rateLimit), 2)
		} else {
			localUtils.Logger(fmt.Sprintf("[%s] AI-decided rate limit for %s: %s", agentName, action.Tool, rateLimit), 1)
		}
		rateLimit = clampRateLimit(rateLimit, agentName)

		switch action.Tool {
		case "dalfox":
			go tools.RunDalfox(action.Target, req.Method, bodyStr, req.Headers, rateLimit)
		case "sqlmap":
			raw := buildRawRequest(req, action.Target, bodyStr)
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
		case "httpx":
			go tools.RunHTTPX(action.Target, rateLimit)
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

func buildRoutes(req burp.BurpRequest, bodyStr string, plan *ai.AIPlan) []ai.AIAction {
	var actions []ai.AIAction

	// 1. Rule-based Heuristic Fallback
	classes := classifyRequest(req, bodyStr, plan)
	actions = append(actions, routesForClassifications(req, classes)...)

	// 2. AI-Suggested Actions
	if plan != nil {
		for _, action := range plan.Actions {
			if normalized, ok := normalizeAction(req, action); ok {
				actions = append(actions, normalized)
			}
		}
	}

	return dedupeActions(actions)
}

func classifyRequest(req burp.BurpRequest, bodyStr string, plan *ai.AIPlan) []string {
	joined := strings.ToLower(req.URL + "\n" + bodyStr + "\n" + headerText(req.Headers))
	if plan != nil {
		joined += "\n" + strings.ToLower(plan.Thinking+" "+strings.Join(plan.VulnerabilitiesSuspected, " "))
	}

	var classes []string
	add := func(name string) {
		for _, existing := range classes {
			if existing == name {
				return
			}
		}
		classes = append(classes, name)
	}

	u, _ := url.Parse(req.URL)
	hasParams := false
	if u != nil {
		hasParams = len(u.Query()) > 0
		joined += "\n" + strings.ToLower(u.Query().Encode())
		for _, values := range u.Query() {
			joined += "\n" + strings.ToLower(strings.Join(values, "\n"))
		}
	}
	if bodyStr != "" && bodyStr != "[Empty Body]" && (strings.Contains(bodyStr, "=") || strings.Contains(bodyStr, "{")) {
		hasParams = true
	}

	if hasParams {
		add("param-discovery")
	}
	if hasParams && hasAny(joined, "sqli", "sql injection", " union ", "select ", "sleep(", "benchmark(", "' or ", "\" or ", " order by ") {
		add("sqli")
	}
	if hasParams && hasAny(joined, "xss", "<script", "javascript:", "onerror=", "onload=", "callback=", "redirect_uri=", "return_url=", "next=") {
		add("xss")
	}
	if hasAny(joined, "../", "..%2f", "%2e%2e", "etc/passwd", "path traversal", "lfi", "file inclusion") {
		add("lfi")
	}
	if hasAny(joined, "ssrf", "url=http", "url=https", "callback=http", "webhook", "metadata.google.internal", "169.254.169.254") {
		add("ssrf")
	}
	if hasAny(joined, "{{", "${", "<%=", "ssti", "template injection") {
		add("ssti")
	}
	if hasAny(joined, "idor", "bola", "broken object", "/user/", "/users/", "/account/", "/accounts/", "/org/", "/tenant/", "uuid") || numericIDPattern.MatchString(joined) {
		add("access-control")
	}
	if hasAny(joined, "graphql", "__schema", "operationname") {
		add("graphql")
	}
	if hasAny(joined, "upload", "multipart/form-data", "filename=", "content-type: image/", "content-type: application/octet-stream") {
		add("upload")
	}
	if hasAny(joined, "/api/", "swagger", "openapi", "api-docs") {
		add("api")
	}
	if req.Method == "GET" && len(classes) == 0 {
		add("crawl")
	}

	return classes
}

var numericIDPattern = regexp.MustCompile(`(?i)(^|[?&/])(id|user_id|account_id|org_id|tenant_id|invoice_id)[=/][0-9]+`)

func routesForClassifications(req burp.BurpRequest, classes []string) []ai.AIAction {
	target := req.URL
	base := map[string]interface{}{"rate_limit": localUtils.GetRateLimit()}
	var actions []ai.AIAction

	add := func(tool string, params map[string]interface{}) {
		if params == nil {
			params = map[string]interface{}{}
		}
		for k, v := range base {
			if _, ok := params[k]; !ok {
				params[k] = v
			}
		}
		actions = append(actions, ai.AIAction{Tool: tool, Target: target, Params: params})
	}

	for _, class := range classes {
		switch class {
		case "sqli":
			add("sqlmap", nil)
		case "xss":
			add("dalfox", nil)
		case "lfi":
			add("nuclei", map[string]interface{}{"tags": "lfi,traversal"})
		case "ssrf":
			add("nuclei", map[string]interface{}{"tags": "ssrf"})
		case "ssti":
			add("nuclei", map[string]interface{}{"tags": "ssti"})
		case "access-control":
			add("arjun", nil)
			add("nuclei", map[string]interface{}{"tags": "exposure,misconfig"})
		case "graphql":
			add("nuclei", map[string]interface{}{"tags": "graphql"})
		case "upload":
			add("nuclei", map[string]interface{}{"tags": "file-upload,rce"})
		case "api":
			add("katana", nil)
			add("arjun", nil)
		case "param-discovery":
			add("arjun", nil)
		case "crawl":
			add("katana", nil)
		}
	}

	return actions
}

func normalizeAction(req burp.BurpRequest, action ai.AIAction) (ai.AIAction, bool) {
	tool := strings.ToLower(strings.TrimSpace(action.Tool))
	switch tool {
	case "dalfox", "sqlmap", "nuclei", "katana", "ffuf", "arjun", "gospider", "censys", "httpx":
	case "send_http_request", "http", "curl":
		tool = "nuclei"
	default:
		return ai.AIAction{}, false
	}

	target := strings.TrimSpace(action.Target)
	if target == "" {
		target = req.URL
	}
	params := action.Params
	if params == nil {
		params = map[string]interface{}{}
	}
	if _, ok := params["rate_limit"]; !ok && tool != "censys" {
		params["rate_limit"] = localUtils.GetRateLimit()
	}

	return ai.AIAction{Tool: tool, Target: target, Params: params}, true
}

func dedupeActions(actions []ai.AIAction) []ai.AIAction {
	seen := make(map[string]bool)
	var out []ai.AIAction
	for _, action := range actions {
		key := strings.ToLower(action.Tool + "\x00" + action.Target + "\x00" + fmt.Sprint(action.Params["tags"]))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, action)
	}
	return out
}

func buildRawRequest(req burp.BurpRequest, target, bodyStr string) string {
	u, _ := url.Parse(target)
	host := "target"
	path := target
	if u != nil && u.Host != "" {
		host = u.Host
		path = u.RequestURI()
	}

	var headerStr strings.Builder
	headerStr.WriteString(fmt.Sprintf("Host: %s\r\n", host))
	for k, v := range req.Headers {
		if strings.ToLower(k) == "host" {
			continue
		}
		headerStr.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	return fmt.Sprintf("%s %s HTTP/1.1\r\n%s\r\n%s", req.Method, path, headerStr.String(), bodyStr)
}

func headerText(headers map[string]string) string {
	var b strings.Builder
	for k, v := range headers {
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(v)
		b.WriteByte('\n')
	}
	return b.String()
}

func hasAny(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}

const maxRateLimit = 50

// clampRateLimit caps the rate limit at maxRateLimit req/s to prevent accidental hammering.
func clampRateLimit(rl string, agentName string) string {
	v, err := strconv.Atoi(rl)
	if err != nil || v <= 0 {
		return localUtils.GetRateLimit()
	}
	if v > maxRateLimit {
		localUtils.Logger(fmt.Sprintf("[%s] Rate limit %d exceeds max (%d), clamping.", agentName, v, maxRateLimit), 2)
		return strconv.Itoa(maxRateLimit)
	}
	return rl
}
