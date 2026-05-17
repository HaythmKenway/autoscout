package tools

import (
	"testing"
)

func TestTranslateRateLimit(t *testing.T) {
	testCases := []struct {
		tool         string
		rate         int
		expectedFlag string
		expectedVal  string
	}{
		{"dalfox", 5, "--delay", "200"},
		{"sqlmap", 5, "--delay", "0.20"},
		{"nuclei", 10, "-rl", "10"},
		{"ffuf", 50, "-rate", "50"},
		{"arjun", 2, "-d", "0.50"},
		{"katana", 1, "-rl", "1"},
	}

	for _, tc := range testCases {
		t.Run(tc.tool, func(t *testing.T) {
			capability, exists := ToolRegistry[tc.tool]
			if !exists {
				t.Fatalf("Tool %s not found in registry", tc.tool)
			}

			flag, val := capability.TranslateRateLimit(tc.rate)
			if flag != tc.expectedFlag {
				t.Errorf("expected flag %s, got %s", tc.expectedFlag, flag)
			}
			if val != tc.expectedVal {
				t.Errorf("expected value %s, got %s", tc.expectedVal, val)
			}
		})
	}
}

func TestToolRegistryCompleteness(t *testing.T) {
	requiredTools := []string{"dalfox", "sqlmap", "nuclei", "ffuf", "katana", "arjun"}
	for _, tool := range requiredTools {
		if _, exists := ToolRegistry[tool]; !exists {
			t.Errorf("Required tool %s missing from registry", tool)
		}
	}
}
