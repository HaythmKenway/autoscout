package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

// runWithLogs starts a command and streams its output to the global logger in real-time
func runWithLogs(toolName string, cmd *exec.Cmd, onComplete func(string)) {
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	
	if err := cmd.Start(); err != nil {
		localUtils.Logger(fmt.Sprintf("[%s] Failed to start: %v", toolName, err), 2)
		return
	}

	var fullOutput strings.Builder
	multi := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(multi)
	
	// Increase buffer size to 1MB to handle large JSON lines (e.g. from dalfox/katana)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	
	for scanner.Scan() {
		line := scanner.Text()
		fullOutput.WriteString(line + "\n")
		localUtils.Logger(fmt.Sprintf("[%s] %s", toolName, line), 1)
	}

	if err := scanner.Err(); err != nil {
		localUtils.Logger(fmt.Sprintf("[%s] Scanner error: %v", toolName, err), 2)
	}

	err := cmd.Wait()
	if err != nil {
		localUtils.Logger(fmt.Sprintf("[%s] Finished with error: %v", toolName, err), 2)
	} else {
		localUtils.Logger(fmt.Sprintf("[%s] Finished successfully", toolName), 1)
	}

	if onComplete != nil {
		onComplete(fullOutput.String())
	}
}

// DalfoxResult is a subset of DalFox's JSON output
type DalfoxResult struct {
	Type     string `json:"type"`
	Evidence string `json:"evidence"`
	PoC      string `json:"poc"`
}

func RunDalfox(target string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting DalFox scan on %s (Proxy: %s)", target, proxy), 1)
	cmd := exec.Command("dalfox", "url", target, "--format", "json", "--proxy", proxy)
	
	runWithLogs("DalFox", cmd, func(output string) {
		lines := strings.Split(output, "\n")
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
	})
}

func RunSQLMap(targetURL string, rawRequest string) {
	proxy := localUtils.GetProxyURL()
	dir := localUtils.GetWorkingDirectory()
	reqFile := fmt.Sprintf("%s/sqlmap_req_%d.txt", dir, time.Now().Unix())
	
	err := os.WriteFile(reqFile, []byte(rawRequest), 0644)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("[SQLMap] Failed to write request file: %v", err), 2)
		return
	}

	localUtils.Logger(fmt.Sprintf("[Tool] Starting SQLMap on %s (Proxy: %s)", targetURL, proxy), 1)
	cmd := exec.Command("sqlmap", "-r", reqFile, "--batch", "--random-agent", "--level", "1", "--risk", "1", "--proxy", proxy)
	
	runWithLogs("SQLMap", cmd, func(output string) {
		if strings.Contains(output, "is vulnerable") {
			database, _ := db.OpenDatabase()
			defer database.Close()
			db.AddVulnerability(database, targetURL, "SQL Injection", "Confirmed via SQLMap", "SQLMap", "Critical")
			localUtils.Logger(fmt.Sprintf("[AI ALERT] SQLMap confirmed vulnerability at %s", targetURL), 1)
			localUtils.ReportToBurp("SQL Injection Confirmed", targetURL, "High")
		}
	})
}

func RunNuclei(target string, tags string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Nuclei scan on %s (Proxy: %s)", target, proxy), 1)
	args := []string{"-u", target, "-silent", "-nc", "-proxy", proxy}
	if tags != "" {
		args = append(args, "-tags", tags)
	} else {
		args = append(args, "-as")
	}

	cmd := exec.Command("nuclei", args...)
	runWithLogs("Nuclei", cmd, nil)
}

func RunFFUF(target string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting FFUF on %s (Proxy: %s)", target, proxy), 1)
	wordlist := "/usr/share/wordlists/dirb/common.txt"
	if _, err := os.Stat(wordlist); err != nil {
		localUtils.Logger("[FFUF] Wordlist not found, skipping", 2)
		return
	}
	
	cmd := exec.Command("ffuf", "-u", target+"/FUZZ", "-w", wordlist, "-s", "-x", proxy)
	runWithLogs("FFUF", cmd, func(output string) {
		lines := strings.Split(output, "\n")
		database, _ := db.OpenDatabase()
		defer database.Close()
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				db.AddFuzzResult(database, target, line, "FUZZ", "FFUF")
			}
		}
	})
}

func RunArjun(target string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Arjun on %s (Proxy: %s)", target, proxy), 1)
	cmd := exec.Command("arjun", "-u", target, "--quiet", "--proxy", proxy)
	runWithLogs("Arjun", cmd, nil)
}

func RunGoSpider(target string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting GoSpider on %s (Proxy: %s)", target, proxy), 1)
	cmd := exec.Command("gospider", "-s", target, "--quiet", "-p", proxy)
	runWithLogs("GoSpider", cmd, nil)
}

func RunKatana(target string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Katana on %s (Proxy: %s)", target, proxy), 1)
	cmd := exec.Command("katana", "-u", target, "-silent", "-proxy", proxy)
	
	runWithLogs("Katana", cmd, func(output string) {
		database, _ := db.OpenDatabase()
		defer database.Close()
		endpoints := strings.Split(output, "\n")
		db.AddSpiderTargets(database, target, endpoints)
	})
}

func RunCensys(target string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Censys search: %s", target), 1)
	cmd := exec.Command("censys", "search", target)
	runWithLogs("Censys", cmd, nil)
}
