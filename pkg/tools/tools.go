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
	
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	
	for scanner.Scan() {
		line := scanner.Text()
		fullOutput.WriteString(line + "\n")
		
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

func RunSubfinder(domain string, rateLimit string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Subfinder on %s", domain), 1)
	args := []string{"-d", domain, "-silent", "-nc", "-t", "10"}
	cmd := exec.Command("subfinder", args...)
	
	runWithLogs("Subfinder", domain, cmd, func(output string) {
		subs := localUtils.ParseSubdomains(output)
		database, _ := db.OpenDatabase()
		defer database.Close()
		
		for _, s := range subs {
			db.AddSubsManual(database, s, domain)
		}
	})
}

func RunHTTPX(domain string, rateLimit string) {
	localUtils.Logger(fmt.Sprintf("[Tool] Starting HTTPX on %s", domain), 1)
	args := []string{"-u", domain, "-title", "-status-code", "-ip", "-json", "-silent"}
	
	rate, _ := strconv.Atoi(rateLimit)
	if capability, exists := ToolRegistry["httpx"]; exists {
		if flag, val := capability.TranslateRateLimit(rate); flag != "" {
			args = append(args, flag, val)
		}
	} else {
		args = append(args, "-rl", rateLimit)
	}

	cmd := exec.Command("httpx", args...)
	runWithLogs("HTTPX", domain, cmd, func(output string) {
		database, _ := db.OpenDatabase()
		defer database.Close()
		
		scanner := bufio.NewScanner(strings.NewReader(output))
		for scanner.Scan() {
			var res map[string]interface{}
			if err := json.Unmarshal(scanner.Bytes(), &res); err == nil {
				db.AddUrl(database, domain, fmt.Sprint(res["title"]), fmt.Sprint(res["url"]), fmt.Sprint(res["host"]), fmt.Sprint(res["scheme"]), "", "", fmt.Sprint(res["tech"]), fmt.Sprint(res["ip"]), fmt.Sprint(res["port"]), fmt.Sprint(res["status_code"]), "")
			}
		}
	})
}

// DalfoxResult is a subset of DalFox's JSON output
type DalfoxResult struct {
	Type     string `json:"type"`
	Evidence string `json:"evidence"`
	PoC      string `json:"poc"`
}

func RunDalfox(target string, method string, body string, headers map[string]string, rateLimit string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting DalFox scan on %s (Method: %s, Proxy: %s, Rate: %s)", target, method, proxy, rateLimit), 1)
	
	args := []string{"url", target, "--format", "jsonl", "--no-color", "--no-spinner", "--proxy", proxy}
	
	rate, _ := strconv.Atoi(rateLimit)
	if capability, exists := ToolRegistry["dalfox"]; exists {
		if flag, val := capability.TranslateRateLimit(rate); flag != "" {
			args = append(args, flag, val)
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
	args := []string{
		"-r", reqFile,
		"--batch",
		"--random-agent",
		"--level", "2",
		"--risk", "2",
		"--threads", "5",
		"--base64", "P",
		"--no-cast",
		"--no-escape",
		"--proxy", proxy,
		"-v", "3",
	}
	
	rate, _ := strconv.Atoi(rateLimit)
	if capability, exists := ToolRegistry["sqlmap"]; exists {
		if flag, val := capability.TranslateRateLimit(rate); flag != "" {
			args = append(args, flag, val)
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

	rate, _ := strconv.Atoi(rateLimit)
	if capability, exists := ToolRegistry["nuclei"]; exists {
		if flag, val := capability.TranslateRateLimit(rate); flag != "" {
			args = append(args, flag, val)
		}
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

	dir := localUtils.GetWorkingDirectory()
	fallbackPath := dir + "/minimal_wordlist.txt"
	if _, err := os.Stat(fallbackPath); os.IsNotExist(err) {
		minimal := "admin\napi\nv1\nv2\ngraphql\n.env\nconfig\nlogin\nsetup\nindex.php\nindex.html\n"
		os.WriteFile(fallbackPath, []byte(minimal), 0644)
	}
	return fallbackPath
}

func getParamWordlist() string {
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

	p := "/usr/share/wordlists/seclists/Discovery/Web-Content/burp-parameter-names.txt"
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
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

	rate, _ := strconv.Atoi(rateLimit)
	if capability, exists := ToolRegistry["ffuf"]; exists {
		if flag, val := capability.TranslateRateLimit(rate); flag != "" {
			args = append(args, flag, val)
		}
	}

	cmd := exec.Command("ffuf", args...)
	runWithLogs("FFUF", target, cmd, func(output string) {
		lines := strings.Split(output, "\n")
		database, _ := db.OpenDatabase()
		defer database.Close()
		for _, line := range lines {
			if strings.TrimSpace(line) == "" { continue }
			var res struct {
				URL           string `json:"url"`
				Status        int    `json:"status"`
				ContentLength int    `json:"length"`
				Input         map[string]string `json:"input"`
			}
			if err := json.Unmarshal([]byte(line), &res); err == nil {
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
	
	if strings.Contains(proxy, "8080") {
		args = append(args, "-oB")
	}
	if wordlist != "" {
		args = append(args, "-w", wordlist)
	}

	rate, _ := strconv.Atoi(rateLimit)
	if capability, exists := ToolRegistry["arjun"]; exists {
		if flag, val := capability.TranslateRateLimit(rate); flag != "" {
			args = append(args, flag, val)
		}
	}

	if method != "" {
		args = append(args, "-m", method)
	}
	
	cmd := exec.Command("arjun", args...)
	runWithLogs("Arjun", target, cmd, nil)
}

func RunGoSpider(target string, rateLimit string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting GoSpider on %s", target), 1)
	args := []string{"-s", target, "--quiet", "--json", "-p", proxy, "-c", "2"}

	cmd := exec.Command("gospider", args...)
	runWithLogs("GoSpider", target, cmd, func(output string) {
		database, _ := db.OpenDatabase()
		defer database.Close()
		lines := strings.Split(output, "\n")
		var endpoints []string
		for _, line := range lines {
			if strings.TrimSpace(line) == "" { continue }
			var res struct {
				Output string `json:"output"`
			}
			if err := json.Unmarshal([]byte(line), &res); err == nil {
				endpoints = append(endpoints, res.Output)
			}
		}
		if len(endpoints) > 0 {
			db.AddSpiderTargets(database, target, endpoints)
		}
	})
}

func RunKatana(target string, rateLimit string) {
	proxy := localUtils.GetProxyURL()
	localUtils.Logger(fmt.Sprintf("[Tool] Starting Katana on %s (Proxy: %s, Rate: %s)", target, proxy, rateLimit), 1)
	args := []string{"-u", target, "-silent", "-jsonl", "-proxy", proxy}
	
	rate, _ := strconv.Atoi(rateLimit)
	if capability, exists := ToolRegistry["katana"]; exists {
		if flag, val := capability.TranslateRateLimit(rate); flag != "" {
			args = append(args, flag, val)
		}
	}

	cmd := exec.Command("katana", args...)
	runWithLogs("Katana", target, cmd, func(output string) {
		database, _ := db.OpenDatabase()
		defer database.Close()
		lines := strings.Split(output, "\n")
		var endpoints []string
		for _, line := range lines {
			if strings.TrimSpace(line) == "" { continue }
			var res struct {
				Request struct {
					URL string `json:"endpoint"`
				} `json:"request"`
			}
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
