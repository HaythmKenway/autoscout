package httpx

import (
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
func Httpx(dbConn *sql.DB, domain string) {
	localUtils.Logger("Running httpx on "+domain, 1)

	// Note: Ensure 'httpx' is in your system PATH
	cmd := exec.Command("httpx", "-u", domain, "-title", "-x", "get", "-status-code", "-ip", "-json", "-fr")

	stdout, err := cmd.Output()
	if err != nil {
		// Don't crash if httpx fails (e.g., domain not found), just log it
		localUtils.Logger(fmt.Sprintf("httpx failed for %s: %v", domain, err), 2)
		return
	}

	if len(stdout) == 0 {
		localUtils.Logger("httpx returned no output for "+domain, 2)
		return
	}

	var result map[string]interface{}
	// Unmarshal only parses the first JSON object it finds.
	// If httpx returns multiple lines, we capture the first one (primary result).
	if err := json.Unmarshal(stdout, &result); err != nil {
		localUtils.Logger(fmt.Sprintf("Failed to parse httpx JSON for %s: %v", domain, err), 2)
		return
	}

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
	if err := db.AddUrl(dbConn, title, url, host, scheme, a, cname, tech, ip, port, statusCode); err != nil {
		localUtils.Logger(fmt.Sprintf("Error saving URL to DB: %v", err), 2)
	}

	localUtils.Logger("httpx on "+domain+" is done", 1)
}
