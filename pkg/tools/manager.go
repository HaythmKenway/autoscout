package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

var (
	toolAvailCache   = make(map[string]bool)
	toolAvailCacheMu sync.RWMutex
)

// IsToolAvailable checks whether a binary is on PATH, caching the result.
func IsToolAvailable(name string) bool {
	toolAvailCacheMu.RLock()
	if avail, ok := toolAvailCache[name]; ok {
		toolAvailCacheMu.RUnlock()
		return avail
	}
	toolAvailCacheMu.RUnlock()

	_, err := exec.LookPath(name)
	avail := err == nil

	toolAvailCacheMu.Lock()
	toolAvailCache[name] = avail
	toolAvailCacheMu.Unlock()

	return avail
}

type ToolParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Flag        string `json:"flag"`
	Default     string `json:"default,omitempty"`
}

type RateLimitConfig struct {
	Flag      string           `json:"flag"`
	Unit      string           `json:"unit"` // "req_s", "ms", "s", "concurrency"
	Calculate func(int) string `json:"-"`
}

type ToolCapability struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	HelpText    string          `json:"help_text"`
	Parameters  []ToolParameter `json:"parameters"`
	RateLimit   RateLimitConfig `json:"rate_limit"`
	Timeout     time.Duration   `json:"-"`
}

var ToolRegistry = make(map[string]ToolCapability)

func init() {
	registerTools()
}

func registerTools() {
	// 1. DalFox
	ToolRegistry["dalfox"] = ToolCapability{
		Name:        "DalFox",
		Description: "Parameter Analysis and XSS Scanning tool.",
		Parameters: []ToolParameter{
			{Name: "rate_limit", Type: "string", Description: "Max requests per second (e.g. '5')", Flag: "--delay", Default: "5"},
			{Name: "method", Type: "string", Description: "HTTP method (GET, POST, etc.)", Flag: "-X"},
		},
		RateLimit: RateLimitConfig{
			Flag: "--delay",
			Unit: "ms",
			Calculate: func(r int) string {
				if r <= 0 { r = 5 }
				return strconv.Itoa(1000 / r)
			},
		},
		Timeout: 10 * time.Minute,
	}

	// 2. SQLMap
	ToolRegistry["sqlmap"] = ToolCapability{
		Name:        "SQLMap",
		Description: "Automatic SQL injection tool.",
		Parameters: []ToolParameter{
			{Name: "rate_limit", Type: "string", Description: "Max requests per second (e.g. '5')", Flag: "--delay", Default: "5"},
			{Name: "level", Type: "string", Description: "Test level 1-5 (default 2)", Flag: "--level", Default: "2"},
			{Name: "risk", Type: "string", Description: "Risk level 1-3 (default 2)", Flag: "--risk", Default: "2"},
		},
		RateLimit: RateLimitConfig{
			Flag: "--delay",
			Unit: "s",
			Calculate: func(r int) string {
				if r <= 0 { r = 5 }
				return fmt.Sprintf("%.2f", 1.0/float64(r))
			},
		},
		Timeout: 20 * time.Minute,
	}

	// 3. Nuclei
	ToolRegistry["nuclei"] = ToolCapability{
		Name:        "Nuclei",
		Description: "Template-based vulnerability scanner.",
		Parameters: []ToolParameter{
			{Name: "tags", Type: "string", Description: "Comma-separated template tags (e.g. 'lfi,rce,sqli,ssrf')", Flag: "-tags"},
			{Name: "rate_limit", Type: "string", Description: "Max requests per second (e.g. '5')", Flag: "-rl", Default: "5"},
		},
		RateLimit: RateLimitConfig{
			Flag: "-rl",
			Unit: "req_s",
			Calculate: func(r int) string {
				if r <= 0 { r = 5 }
				return strconv.Itoa(r)
			},
		},
		Timeout: 15 * time.Minute,
	}

	// 4. FFUF
	ToolRegistry["ffuf"] = ToolCapability{
		Name:        "FFUF",
		Description: "Fast web fuzzer for directories and parameters.",
		Parameters: []ToolParameter{
			{Name: "rate_limit", Type: "string", Description: "Max requests per second (e.g. '5')", Flag: "-rate", Default: "5"},
			{Name: "method", Type: "string", Description: "HTTP method (GET, POST, etc.)", Flag: "-X"},
			{Name: "wordlist", Type: "string", Description: "Optional path to a custom wordlist file", Flag: "-w"},
		},
		RateLimit: RateLimitConfig{
			Flag: "-rate",
			Unit: "req_s",
			Calculate: func(r int) string {
				if r <= 0 { r = 5 }
				return strconv.Itoa(r)
			},
		},
		Timeout: 15 * time.Minute,
	}

	// 5. Katana
	ToolRegistry["katana"] = ToolCapability{
		Name:        "Katana",
		Description: "Next-gen web crawling framework.",
		Parameters: []ToolParameter{
			{Name: "rate_limit", Type: "string", Description: "Max requests per second (e.g. '5')", Flag: "-rl", Default: "5"},
		},
		RateLimit: RateLimitConfig{
			Flag: "-rl",
			Unit: "req_s",
			Calculate: func(r int) string {
				if r <= 0 { r = 5 }
				return strconv.Itoa(r)
			},
		},
		Timeout: 5 * time.Minute,
	}

	// 6. Arjun
	ToolRegistry["arjun"] = ToolCapability{
		Name:        "Arjun",
		Description: "HTTP parameter discovery suite.",
		Parameters: []ToolParameter{
			{Name: "rate_limit", Type: "string", Description: "Max requests per second (e.g. '5')", Flag: "-d", Default: "5"},
			{Name: "method", Type: "string", Description: "HTTP method to test (GET, POST, JSON, XML)", Flag: "-m"},
		},
		RateLimit: RateLimitConfig{
			Flag: "-d",
			Unit: "s",
			Calculate: func(r int) string {
				if r <= 0 { r = 5 }
				return fmt.Sprintf("%.2f", 1.0/float64(r))
			},
		},
		Timeout: 8 * time.Minute,
	}

	// 7. HTTPX
	ToolRegistry["httpx"] = ToolCapability{
		Name:        "HTTPX",
		Description: "HTTP probing toolkit for service discovery.",
		Parameters: []ToolParameter{
			{Name: "rate_limit", Type: "string", Description: "Max requests per second (e.g. '5')", Flag: "-rl", Default: "5"},
		},
		RateLimit: RateLimitConfig{
			Flag: "-rl",
			Unit: "req_s",
			Calculate: func(r int) string {
				if r <= 0 { r = 5 }
				return strconv.Itoa(r)
			},
		},
		Timeout: 3 * time.Minute,
	}

	// 8. Subfinder
	ToolRegistry["subfinder"] = ToolCapability{
		Name:        "Subfinder",
		Description: "Passive subdomain discovery tool.",
		Parameters: []ToolParameter{
			{Name: "rate_limit", Type: "string", Description: "Number of concurrent threads", Flag: "-t", Default: "10"},
		},
		RateLimit: RateLimitConfig{
			Flag: "-t",
			Unit: "threads",
			Calculate: func(r int) string {
				return "10"
			},
		},
		Timeout: 5 * time.Minute,
	}
}

func (tc ToolCapability) TranslateRateLimit(rate int) (string, string) {
	if tc.RateLimit.Flag == "" {
		return "", ""
	}
	return tc.RateLimit.Flag, tc.RateLimit.Calculate(rate)
}

type Job struct {
	ID        string
	Tool      string
	Target    string
	StartTime time.Time
	Cmd       *exec.Cmd
	IsKilling bool
}

type JobManager struct {
	mu   sync.RWMutex
	jobs map[string]*Job
	sem  chan struct{}
}

const (
	FuncSubfinder = "Subfinder"
	FuncHTTPX     = "HTTPX"
	FuncSpider    = "Spider"
)

var DefaultJobManager = &JobManager{
	jobs: make(map[string]*Job),
	sem:  make(chan struct{}, 10), 
}

func (m *JobManager) Register(tool, target string, cmd *exec.Cmd) string {
	m.sem <- struct{}{}
	m.mu.Lock()
	defer m.mu.Unlock()

	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	} else {
		cmd.SysProcAttr.Setpgid = true
	}

	id := fmt.Sprintf("%s-%d", tool, time.Now().Unix()%10000)
	job := &Job{
		ID:        id,
		Tool:      tool,
		Target:    target,
		StartTime: time.Now(),
		Cmd:       cmd,
	}
	m.jobs[id] = job
	return id
}

func (m *JobManager) Unregister(id string) {
	m.mu.Lock()
	delete(m.jobs, id)
	m.mu.Unlock()
	<-m.sem
}

func (m *JobManager) StopJob(id string) error {
	m.mu.Lock()
	job, ok := m.jobs[id]
	if ok {
		job.IsKilling = true
	}
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("job not found: %s", id)
	}

	if job.Cmd != nil && job.Cmd.Process != nil {
		pid := job.Cmd.Process.Pid
		err := syscall.Kill(-pid, syscall.SIGKILL)
		if err == nil {
			localUtils.Logger(fmt.Sprintf("[JobManager] Killed process group for job %s", id), 1)
			return nil
		}
		return job.Cmd.Process.Kill()
	}
	return nil
}

func (m *JobManager) StopAllJobs() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.jobs))
	for id := range m.jobs {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	for _, id := range ids {
		m.StopJob(id)
	}
}

func (m *JobManager) ListJobs() []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var jobs []*Job
	for _, job := range m.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

func GetToolCapabilitiesJSON() string {
	data, _ := json.MarshalIndent(ToolRegistry, "", "  ")
	return string(data)
}
