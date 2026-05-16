package orchestrator

import (
	"testing"

	"github.com/HaythmKenway/autoscout/pkg/ai"
	"github.com/HaythmKenway/autoscout/pkg/burp"
)

func TestBuildRoutesDetectsSQLi(t *testing.T) {
	req := burp.BurpRequest{
		Method:  "GET",
		URL:     "https://example.com/search?q=1%27%20or%201=1",
		Headers: map[string]string{"Host": "example.com"},
	}

	actions := buildRoutes(req, "", &ai.AIPlan{})

	if !hasRoute(actions, "sqlmap", "https://example.com/search?q=1%27%20or%201=1") {
		t.Fatalf("expected sqlmap route, got %#v", actions)
	}
}

func TestBuildRoutesDetectsXSS(t *testing.T) {
	req := burp.BurpRequest{
		Method:  "GET",
		URL:     "https://example.com/search?q=%3Cscript%3Ealert(1)%3C/script%3E",
		Headers: map[string]string{"Host": "example.com"},
	}

	actions := buildRoutes(req, "", &ai.AIPlan{})

	if !hasRoute(actions, "dalfox", "https://example.com/search?q=%3Cscript%3Ealert(1)%3C/script%3E") {
		t.Fatalf("expected dalfox route, got %#v", actions)
	}
}

func TestBuildRoutesNormalizesAISuggestedHTTPAction(t *testing.T) {
	req := burp.BurpRequest{
		Method:  "GET",
		URL:     "https://example.com/api/users?id=123",
		Headers: map[string]string{"Host": "example.com"},
	}
	plan := &ai.AIPlan{
		Actions: []ai.AIAction{
			{Tool: "send_http_request", Target: "https://example.com/api/users?id=124"},
		},
	}

	actions := buildRoutes(req, "", plan)

	if !hasRoute(actions, "nuclei", "https://example.com/api/users?id=124") {
		t.Fatalf("expected normalized nuclei route, got %#v", actions)
	}
}

func TestBuildRoutesDoesNotRewriteTargetHost(t *testing.T) {
	req := burp.BurpRequest{
		Method:  "GET",
		URL:     "https://example.com/api/users?id=123",
		Headers: map[string]string{"Host": "example.com"},
	}

	actions := buildRoutes(req, "", &ai.AIPlan{})

	for _, action := range actions {
		if action.Target == "https://meow.com/api/users?id=123" {
			t.Fatalf("unexpected host rewrite in action: %#v", action)
		}
	}
}

func hasRoute(actions []ai.AIAction, tool, target string) bool {
	for _, action := range actions {
		if action.Tool == tool && action.Target == target {
			return true
		}
	}
	return false
}
