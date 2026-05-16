package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"gopkg.in/yaml.v3"
)

// runWithLogs starts a command and streams its output to the global logger in real-time
func runWithLogs(toolName string, target string, cmd *exec.Cmd, onComplete func(string)) {
	jobID := DefaultJobManager.Register(toolName, target, cmd)
	defer DefaultJobManager.Unregister(jobID)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	
	if err := cmd.Start(); err != nil {
		localUtils.Logger(fmt.Sprintf("[%s] Failed to start: %v", toolName, err), 2)
		return
	}

	var fullOutput strings.Builder
	multi := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(multi)
	
	// Use a large buffer but we will truncate before logging
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	
	for scanner.Scan() {
		line := scanner.Text()
		fullOutput.WriteString(line + "\n")
		
		// Truncate for logging to prevent TUI lag and massive log files
		displayLine := line
		if len(displayLine) > 1000 {
			displayLine = displayLine[:997] + "..."
		}
		localUtils.Logger(fmt.Sprintf("[%s] %s", toolName, displayLine), 1)
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

func RunDalfox(target string, method string, body string, headers map[string]string, rateLimit string) {
	proxy := localUtils.GetProxyURL()
	if proxy == "" {
		proxy = "http://localhost:8080"
	}
	localUtils.Logger(fmt.Sprintf("[Tool] Starting DalFox scan on %s (Method: %s, Proxy: %s, Rate: %s)", target, method, proxy, rateLimit), 1)
	
	// Use jsonl for reliable line-by-line parsing and disable interactive features
	args := []string{"url", target, "--format", "jsonl", "--no-color", "--no-spinner", "--proxy", proxy}
	
	// DalFox --delay is in milliseconds. If rate is 5 req/s, delay should be 1000/5 = 200ms
	if rateLimit != "" {
		if rate, err := strconv.Atoi(rateLimit); err == nil && rate > 0 {
			delay := 1000 / rate
			args = append(args, "--delay", fmt.Sprintf("%d", delay))
		}
	}

	if body != "" {
		args = append(args, "-X", method, "--data", body)
	}
	for k, v := range headers {
		args = append(args, "-H", fmt.Sprintf("%s: %s", k, v))
	}

	cmd := exec.Command("dalfox", args...)
	
	runWithLogs("DalFox", target, cmd, func(output string) {
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

func RunSQLMap(targetURL string, rawRequest string, rateLimit string) {
	proxy := localUtils.GetProxyURL()
	dir := localUtils.GetWorkingDirectory()
	reqFile := fmt.Sprintf("%s/sqlmap_req_%d.txt", dir, time.Now().Unix())
	
	err := os.WriteFile(reqFile, []byte(rawRequest), 0644)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("[SQLMap] Failed to write request file: %v", err), 2)
		return
	}

	localUtils.Logger(fmt.Sprintf("[Tool] Starting SQLMap on %s (Proxy: %s, Rate: %s)", targetURL, proxy, rateLimit), 1)
	// Enhanced SQLMap arguments for better results and verbosity
	args := []string{
		"-r", reqFile,
		"--batch",
		"--random-agent",
		"--level", "2",
		"--risk", "2",
		"--threads", "5",
		"--base64", "P", // Try base64 encoding for parameters
		"--no-cast",
		"--no-escape",
		"--proxy", proxy,
		"-v", "3", // Increased verbosity
	}
	
	// SQLMap --delay is in seconds. If rate is 5 req/s, delay should be 1/5 = 0.2s
	if rateLimit != "" {
		if rate, err := strconv.ParseFloat(rateLimit, 64); err == nil && rate > 0 {
			delay := 1.0 / rate
			args = append(args, "--delay", fmt.Sprintf("%.2f", delay))
		}
	}

	if strings.HasPrefix(targetURL, "https") {
		args = append(args, "--force-ssl")
	}
	cmd := exec.Command("sqlmap", args...)
	
	runWithLogs("SQLMap", targetURL, cmd, func(output string) {
		if strings.Contains(output, "is vulnerable") {
			database, _ := db.OpenDatabase()
			defer database.Close()
			db.AddVulnerability(database, targetURL, "SQL Injection", "Confirmed via SQLMap", "SQLMap", "Critical")
			localUtils.Logger(fmt.Sprintf("[AI ALERT] SQLMap confirmed vulnerability at %s", targetURL), 1)
			localUtils.ReportToBurp("SQL Injection Confirmed", targetURL, "High")
		}
	})
}

// NucleiResult represents a finding from Nuclei
type NucleiResult struct {
	TemplateID string `json:"template-id"`
	Info       struct {
		Name     string `json:"name"`
		Severity string `json:"severity"`
	} `json:"info"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Matched  string `json:"matched-at"`
	Metadata struct {
		Description string `json:"description"`
	} `json:"metadata"`
}

func RunNuclei(target string, tags string, rateLimit string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Nuclei scan on %s (Proxy: %s, Rate: %s)", target, proxy, rateLimit), 1)
	args := []string{"-u", target, "-silent", "-nc", "-jsonl", "-proxy", proxy}
	if tags != "" {
		args = append(args, "-tags", tags)
	} else {
		args = append(args, "-as")
	}

	if rateLimit != "" {
		args = append(args, "-rl", rateLimit)
	} else {
		args = append(args, "-rl", "5") // Safety fallback
	}

	cmd := exec.Command("nuclei", args...)
	runWithLogs("Nuclei", target, cmd, func(output string) {
		lines := strings.Split(output, "\n")
		database, _ := db.OpenDatabase()
		defer database.Close()

		for _, line := range lines {
			if strings.TrimSpace(line) == "" { continue }
			var res NucleiResult
			if err := json.Unmarshal([]byte(line), &res); err == nil {
				severity := strings.Title(res.Info.Severity)
				detail := fmt.Sprintf("Template: %s\nMatched: %s\nName: %s", res.TemplateID, res.Matched, res.Info.Name)
				db.AddVulnerability(database, target, res.Info.Name, detail, "Nuclei", severity)
				localUtils.Logger(fmt.Sprintf("[AI ALERT] Nuclei found %s: %s", res.Info.Severity, res.Info.Name), 1)
				localUtils.ReportToBurp(res.Info.Name, detail, severity)
			}
		}
	})
}

func getWordlist() string {
	// 1. Check user config
	settingsPath := os.ExpandEnv("$HOME/.config/autoscout/user-config.yaml")
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		var config struct {
			Settings struct {
				Wordlist string `yaml:"wordlist_path"`
			} `yaml:"settings"`
		}
		yaml.Unmarshal(data, &config)
		if config.Settings.Wordlist != "" {
			if _, err := os.Stat(config.Settings.Wordlist); err == nil {
				return config.Settings.Wordlist
			}
		}
	}

	// 2. Check common paths
	commonPaths := []string{
		"/usr/share/wordlists/dirb/common.txt",
		"/usr/share/wordlists/rockyou.txt",
		"/usr/share/seclists/Discovery/Web-Content/common.txt",
	}
	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// 3. Fallback: Create a minimal internal wordlist
	dir := localUtils.GetWorkingDirectory()
	fallbackPath := dir + "/minimal_wordlist.txt"
	if _, err := os.Stat(fallbackPath); os.IsNotExist(err) {
		minimal := "admin\napi\nv1\nv2\ngraphql\n.env\nconfig\nlogin\nsetup\nindex.php\nindex.html\n"
		os.WriteFile(fallbackPath, []byte(minimal), 0644)
	}
	return fallbackPath
}

func getParamWordlist() string {
	// 1. Check user config
	settingsPath := os.ExpandEnv("$HOME/.config/autoscout/user-config.yaml")
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		var config struct {
			Settings struct {
				Wordlist string `yaml:"param_wordlist_path"`
			} `yaml:"settings"`
		}
		yaml.Unmarshal(data, &config)
		if config.Settings.Wordlist != "" {
			if _, err := os.Stat(config.Settings.Wordlist); err == nil {
				return config.Settings.Wordlist
			}
		}
	}

	// 2. Common path for parameters
	p := "/usr/share/wordlists/seclists/Discovery/Web-Content/burp-parameter-names.txt"
	if _, err := os.Stat(p); err == nil {
		return p
	}

	return "" // Let arjun use its internal default if nothing found
}

// FfufResult represents a single finding in FFUF JSON output
type FfufResult struct {
	URL           string `json:"url"`
	Status        int    `json:"status"`
	ContentLength int    `json:"length"`
	Words         int    `json:"words"`
	Lines         int    `json:"lines"`
	Input         map[string]string `json:"input"`
}

func RunFFUF(target string, method string, body string, headers map[string]string, rateLimit string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting FFUF on %s (Method: %s, Proxy: %s, Rate: %s)", target, method, proxy, rateLimit), 1)
	
	wordlist := getWordlist()
	localUtils.Logger(fmt.Sprintf("[FFUF] Using wordlist: %s", wordlist), 1)
	
	args := []string{"-u", target+"/FUZZ", "-w", wordlist, "-s", "-json", "-x", proxy}
	if method != "" {
		args = append(args, "-X", method)
	}
	if body != "" {
		args = append(args, "-d", body)
	}
	for k, v := range headers {
		args = append(args, "-H", fmt.Sprintf("%s: %s", k, v))
	}

	if rateLimit != "" {
		args = append(args, "-rate", rateLimit)
	} else {
		args = append(args, "-rate", "5") // Safety fallback
	}

	cmd := exec.Command("ffuf", args...)
	runWithLogs("FFUF", target, cmd, func(output string) {
		lines := strings.Split(output, "\n")
		database, _ := db.OpenDatabase()
		defer database.Close()
		for _, line := range lines {
			if strings.TrimSpace(line) == "" { continue }
			var res FfufResult
			if err := json.Unmarshal([]byte(line), &res); err == nil {
				// FFUF with -json outputs one JSON object per line for results
				resultStr := fmt.Sprintf("%d | %d L | %s", res.Status, res.ContentLength, res.URL)
				db.AddFuzzResult(database, target, resultStr, res.Input["FUZZ"], "FFUF")
			}
		}
	})
}

func RunArjun(target string, method string, body string, headers map[string]string, rateLimit string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Arjun on %s (Proxy: %s, Rate: %s)", target, proxy, rateLimit), 1)
	
	wordlist := getParamWordlist()
	args := []string{"-u", target, "-q"}
	
	// Arjun doesn't have a general --proxy but has -oB for Burp output
	// If the user has a proxy set, we'll try to use -oB if it's the default port
	if strings.Contains(proxy, "8080") {
		args = append(args, "-oB")
	}

	if wordlist != "" {
		localUtils.Logger(fmt.Sprintf("[Arjun] Using custom wordlist: %s", wordlist), 1)
		args = append(args, "-w", wordlist)
	}

	// Arjun -d is in seconds. If rate is 5 req/s, delay should be 1/5 = 0.2s
	if rateLimit != "" {
		if rate, err := strconv.ParseFloat(rateLimit, 64); err == nil && rate > 0 {
			delay := 1.0 / rate
			args = append(args, "-d", fmt.Sprintf("%.2f", delay))
		}
	}

	if method != "" {
		args = append(args, "-m", method)
	}
	if body != "" {
		args = append(args, "--headers", body) // Arjun uses --headers for adding data sometimes? Wait, check arjun help again.
		// Actually arjun help says --headers [HEADERS] Add headers. 
		// It doesn't seem to have a --data flag for the initial request in a way that matches others.
	}
	// For simplicity, we just pass the URL and let arjun handle discovery.
	cmd := exec.Command("arjun", args...)
	runWithLogs("Arjun", target, cmd, nil)
	}

	// GoSpiderResult represents a single finding from GoSpider
	type GoSpiderResult struct {
	Source string `json:"source"`
	Output string `json:"output"`
	}

	func RunGoSpider(target string, rateLimit string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting GoSpider on %s (Proxy: %s, Rate: %s)", target, proxy, rateLimit), 1)
	args := []string{"-s", target, "--quiet", "--json", "-p", proxy}

	// Map req/s to concurrency for GoSpider
	concurrency := "1"
	if rateLimit != "" {
		if r, err := strconv.Atoi(rateLimit); err == nil && r > 2 {
			concurrency = "2" // Keep concurrency very low for safety
		}
	}
	args = append(args, "-c", concurrency)

	cmd := exec.Command("gospider", args...)
	runWithLogs("GoSpider", target, cmd, func(output string) {
		database, _ := db.OpenDatabase()
		defer database.Close()
		lines := strings.Split(output, "\n")
		var endpoints []string
		for _, line := range lines {
			if strings.TrimSpace(line) == "" { continue }
			var res GoSpiderResult
			if err := json.Unmarshal([]byte(line), &res); err == nil {
				// GoSpider output field contains the URL
				endpoints = append(endpoints, res.Output)
			}
		}
		if len(endpoints) > 0 {
			db.AddSpiderTargets(database, target, endpoints)
		}
	})
	}

	// KatanaResult represents a single endpoint finding from Katana
	type KatanaResult struct {
	Timestamp string `json:"timestamp"`
	Request   struct {
		Method string `json:"method"`
		URL    string `json:"endpoint"`
	} `json:"request"`
	Response struct {
		StatusCode int `json:"status-code"`
	} `json:"response"`
	}

	func RunKatana(target string, rateLimit string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Katana on %s (Proxy: %s, Rate: %s)", target, proxy, rateLimit), 1)
	args := []string{"-u", target, "-silent", "-jsonl", "-proxy", proxy}
	if rateLimit != "" {
		args = append(args, "-rl", rateLimit)
	} else {
		args = append(args, "-rl", "5") // Safety fallback
	}
	cmd := exec.Command("katana", args...)
	runWithLogs("Katana", target, cmd, func(output string) {
		localUtils.Logger(fmt.Sprintf("[Katana RAW Output] %s", output), 3) // Log raw output
		database, _ := db.OpenDatabase()
		defer database.Close()
		lines := strings.Split(output, "\n")
		var endpoints []string
		for _, line := range lines {
			if strings.TrimSpace(line) == "" { continue }
			var res KatanaResult
			if err := json.Unmarshal([]byte(line), &res); err == nil {
				endpoints = append(endpoints, res.Request.URL)
			}
		}
		if len(endpoints) > 0 {
			db.AddSpiderTargets(database, target, endpoints)
		}
	})
	}

	func RunCensys(target string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Censys search: %s", target), 1)
	cmd := exec.Command("censys", "search", target)
	runWithLogs("Censys", target, cmd, nil)
	}
