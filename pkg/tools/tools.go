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
	cmd := exec.Command("dalfox", "url", target, "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		localUtils.Logger(fmt.Sprintf("[DalFox] Error: %v", err), 2)
		return
	}

	lines := strings.Split(string(output), "\n")
	database, _ := db.OpenDatabase()
	defer database.Close()

	for _, line := range lines {
		if strings.TrimSpace(line) == "" { continue }
		var res DalfoxResult
		if err := json.Unmarshal([]byte(line), &res); err == nil {
			db.AddVulnerability(database, target, "XSS", res.PoC, "DalFox", "High")
			localUtils.Logger(fmt.Sprintf("[AI ALERT] DalFox found XSS: %s", res.PoC), 1)
			localUtils.ReportToBurp("XSS Found (DalFox)", res.PoC, "High")
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

	localUtils.Logger(fmt.Sprintf("[Tool] Starting SQLMap on %s", targetURL), 1)
	cmd := exec.Command("sqlmap", "-r", reqFile, "--batch", "--random-agent", "--level", "1", "--risk", "1")
	output, _ := cmd.CombinedOutput()
	
	if strings.Contains(string(output), "is vulnerable") {
		database, _ := db.OpenDatabase()
		defer database.Close()
		db.AddVulnerability(database, targetURL, "SQL Injection", "Confirmed via SQLMap", "SQLMap", "Critical")
		localUtils.Logger(fmt.Sprintf("[AI ALERT] SQLMap confirmed vulnerability at %s", targetURL), 1)
		localUtils.ReportToBurp("SQL Injection Confirmed", targetURL, "High")
	}
}

func RunNuclei(target string, tags string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Nuclei scan on %s", target), 1)
	args := []string{"-u", target, "-silent", "-nc"}
	if tags != "" {
		args = append(args, "-tags", tags)
	} else {
		args = append(args, "-as")
	}

	cmd := exec.Command("nuclei", args...)
	output, _ := cmd.CombinedOutput()
	
	if len(output) > 0 {
		localUtils.Logger(fmt.Sprintf("[Nuclei] Findings:\n%s", string(output)), 1)
		// Logic to parse nuclei findings and add to DB
	}
}

func RunFFUF(target string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting FFUF on %s", target), 1)
	// Example: ffuf -u target/FUZZ -w wordlist -mc 200,301
	wordlist := "/usr/share/wordlists/dirb/common.txt" // Default path
	if _, err := os.Stat(wordlist); err != nil {
		localUtils.Logger("[FFUF] Wordlist not found, skipping", 2)
		return
	}
	
	cmd := exec.Command("ffuf", "-u", target+"/FUZZ", "-w", wordlist, "-s")
	output, _ := cmd.CombinedOutput()
	
	lines := strings.Split(string(output), "\n")
	database, _ := db.OpenDatabase()
	defer database.Close()
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			db.AddFuzzResult(database, target, line, "FUZZ", "FFUF")
		}
	}
}

func RunArjun(target string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Arjun on %s", target), 1)
	cmd := exec.Command("arjun", "-u", target, "--quiet")
	output, _ := cmd.CombinedOutput()
	if len(output) > 0 {
		localUtils.Logger(fmt.Sprintf("[Arjun] Discovered parameters at %s", target), 1)
	}
}

func RunGoSpider(target string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting GoSpider on %s", target), 1)
	cmd := exec.Command("gospider", "-s", target, "--quiet")
	cmd.Run()
}

func RunKatana(target string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Katana on %s", target), 1)
	cmd := exec.Command("katana", "-u", target, "-silent")
	output, _ := cmd.CombinedOutput()
	
	database, _ := db.OpenDatabase()
	defer database.Close()

	endpoints := strings.Split(string(output), "\n")
	db.AddSpiderTargets(database, target, endpoints)
}

func RunCensys(target string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Censys search: %s", target), 1)
	cmd := exec.Command("censys", "search", target)
	cmd.Run()
}
