package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

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
