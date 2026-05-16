package httpx

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

func assertInterfaces(v interface{}) string {
	if v == nil {
		return ""
	}

	if s, ok := v.(string); ok {
		return s
	}

	if slice, ok := v.([]interface{}); ok {
		var result strings.Builder
		for i, val := range slice {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(assertInterfaces(val))
		}
		return result.String()
	}

	return ""
}

// Httpx runs the tool and saves results to the DB.
// It accepts *sql.DB to reuse the worker's connection.
func Httpx(dbConn *sql.DB, domain string, rateLimit string) {
	localUtils.Logger(fmt.Sprintf("Running httpx on %s (Rate: %s)", domain, rateLimit), 1)

	if rateLimit == "" {
		rateLimit = localUtils.GetRateLimit()
	}

	// Note: Ensure 'httpx' is in your system PATH
	args := []string{"-u", domain, "-title", "-x", "get", "-status-code", "-ip", "-json", "-fr", "-silent"}
	if rateLimit != "" {
		args = append(args, "-rl", rateLimit)
	}
	
	cmd := exec.Command("httpx", args...)

	stdout, err := cmd.Output()
	if err != nil {
		// Log raw stdout/stderr for better debugging when command fails
		stderrOutput, _ := cmd.CombinedOutput()
		localUtils.Logger(fmt.Sprintf("httpx failed for %s: %v\nStdout/Stderr: %s", domain, err, string(stderrOutput)), 2)
		return
	}

	if len(stdout) == 0 {
		localUtils.Logger("httpx returned no output for "+domain, 2)
		return
	}

	var result map[string]interface{}
	// Use a scanner to find the line that contains the JSON object
	foundJSON := false
	scanner := bufio.NewScanner(strings.NewReader(string(stdout)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
			if err := json.Unmarshal([]byte(line), &result); err == nil {
				foundJSON = true
				break
			}
		}
	}

	if !foundJSON {
		localUtils.Logger(fmt.Sprintf("Failed to find valid httpx JSON in output for %s. Raw output: %s", domain, string(stdout)), 2)
		// Attempt to parse the entire output as a single JSON object if line-by-line fails
		if err := json.Unmarshal([]byte(strings.TrimSpace(string(stdout))), &result); err != nil {
			localUtils.Logger(fmt.Sprintf("Failed to parse httpx output as single JSON object for %s: %v. Raw output: %s", domain, err, string(stdout)), 2)
			return // Skip saving to DB if we can't parse anything
		}
		// If the above unmarshal succeeded, foundJSON should be true, but we will proceed assuming it worked.
		foundJSON = true // This is a fallback for cases where the entire output is one JSON line.
	}
	subdomain := domain
	title := assertInterfaces(result["title"])
	url := assertInterfaces(result["url"])
	host := assertInterfaces(result["host"])
	scheme := assertInterfaces(result["scheme"])
	a := assertInterfaces(result["a"])
	cname := assertInterfaces(result["cname"])
	tech := assertInterfaces(result["tech"])
	statusCode := assertInterfaces(result["status_code"])
	port := assertInterfaces(result["port"])
	ip := assertInterfaces(result["ip"])

	// Pass the existing DB connection to AddUrl
	if err := db.AddUrl(dbConn, subdomain, title, url, host, scheme, a, cname, tech, ip, port, statusCode, ""); err != nil {
		localUtils.Logger(fmt.Sprintf("Error saving URL to DB: %v", err), 2)
	}

	localUtils.Logger("httpx on "+domain+" is done", 1)
}
