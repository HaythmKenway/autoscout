package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

// DalfoxResult is a subset of DalFox's JSON output
type DalfoxResult struct {
	Type     string `json:"type"`
	Evidence string `json:"evidence"`
	PoC      string `json:"poc"`
}

func RunDalfox(target string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting DalFox scan on %s", target), 1)
	
	// Command: dalfox url <url> --format json
	cmd := exec.Command("dalfox", "url", target, "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		localUtils.Logger(fmt.Sprintf("[DalFox] Error: %v", err), 2)
		return
	}

	// Parse JSON output line by line (DalFox outputs one JSON per finding)
	lines := strings.Split(string(output), "\n")
	database, _ := db.OpenDatabase()
	defer database.Close()

	for _, line := range lines {
		if strings.TrimSpace(line) == "" { continue }
		var res DalfoxResult
		if err := json.Unmarshal([]byte(line), &res); err == nil {
			db.AddVulnerability(database, target, "XSS", res.PoC, "DalFox", "High")
			localUtils.Logger(fmt.Sprintf("[AI ALERT] DalFox found XSS: %s", res.PoC), 1)
		}
	}
}

func RunSQLMap(targetURL string, rawRequest string) {
	dir := localUtils.GetWorkingDirectory()
	reqFile := fmt.Sprintf("%s/sqlmap_req_%d.txt", dir, time.Now().Unix())
	
	err := os.WriteFile(reqFile, []byte(rawRequest), 0644)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("[SQLMap] Failed to write request file: %v", err), 2)
		return
	}

	localUtils.Logger(fmt.Sprintf("[Tool] Starting SQLMap on %s (Req: %s)", targetURL, reqFile), 1)
	
	// sqlmap -r <file> --batch --random-agent --level 1 --risk 1
	cmd := exec.Command("sqlmap", "-r", reqFile, "--batch", "--random-agent")
	output, _ := cmd.CombinedOutput()
	
	// Basic parsing for demo; in production, use --json or parse logs
	if strings.Contains(string(output), "is vulnerable") || strings.Contains(string(output), "confirming") {
		database, _ := db.OpenDatabase()
		defer database.Close()
		db.AddVulnerability(database, targetURL, "SQL Injection", "See "+reqFile, "SQLMap", "Critical")
		localUtils.Logger(fmt.Sprintf("[AI ALERT] SQLMap confirmed vulnerability at %s", targetURL), 1)
	}
}

func RunNuclei(target string, tags string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Nuclei scan on %s (Tags: %s)", target, tags), 1)
	
	args := []string{"-u", target, "-json-export", "/tmp/nuclei_out.json"}
	if tags != "" {
		args = append(args, "-tags", tags)
	} else {
		args = append(args, "-as") // Automatic scan if no tags
	}

	cmd := exec.Command("nuclei", args...)
	cmd.Run() // Run in background

	// Logic to parse nuclei_out.json and add to DB would go here
}

func RunFFUF(target string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting FFUF discovery on %s", target), 1)
	// Example: ffuf -u target/FUZZ -w wordlist.txt
}

func RunArjun(target string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Arjun parameter discovery on %s", target), 1)
	// Example: arjun -u target
}

func RunGoSpider(target string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting GoSpider crawl on %s", target), 1)
	// Example: gospider -s target
}

func RunCensys(target string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Censys investigation on %s", target), 1)
	// Example: censys search "ip:1.1.1.1" or "domain:example.com"
	cmd := exec.Command("censys", "search", target)
	cmd.Run()
}

func RunKatana(target string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Katana crawl on %s", target), 1)
	cmd := exec.Command("katana", "-u", target, "-json")
	output, err := cmd.CombinedOutput()
	if err != nil { return }

	database, _ := db.OpenDatabase()
	defer database.Close()

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		var res struct {
			Request struct {
				Endpoint string `json:"endpoint"`
			} `json:"request"`
		}
		if err := json.Unmarshal([]byte(line), &res); err == nil {
			db.AddSpiderTargets(database, target, []string{res.Request.Endpoint})
		}
	}
}
