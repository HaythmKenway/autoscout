package deps

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// InstallMethod describes how a missing dependency can be installed.
type InstallMethod int

const (
	MethodGo     InstallMethod = iota // go install <pkg>
	MethodPip3                        // pip3 install <pkg>
	MethodManual                      // user must install manually
)

// Dep describes a single external binary dependency.
type Dep struct {
	Name     string
	Binary   string
	Pkg      string        // package identifier for go install / pip3 install
	Method   InstallMethod
	Category string
	Optional bool // optional deps only warn, never block startup
	ManualURL string // shown for MethodManual deps
}

// Result is the outcome of checking one dependency.
type Result struct {
	Dep       Dep
	Installed bool
}

// All registered dependencies, in display order.
var All = []Dep{
	// ── Scanners ──────────────────────────────────────────────────────────────
	{Name: "nuclei", Binary: "nuclei",
		Pkg: "github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest",
		Method: MethodGo, Category: "scanner"},
	{Name: "dalfox", Binary: "dalfox",
		Pkg: "github.com/hahwul/dalfox/v2@latest",
		Method: MethodGo, Category: "scanner"},
	{Name: "ffuf", Binary: "ffuf",
		Pkg: "github.com/ffuf/ffuf/v2@latest",
		Method: MethodGo, Category: "scanner"},
	{Name: "httpx", Binary: "httpx",
		Pkg: "github.com/projectdiscovery/httpx/cmd/httpx@latest",
		Method: MethodGo, Category: "scanner"},
	{Name: "subfinder", Binary: "subfinder",
		Pkg: "github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest",
		Method: MethodGo, Category: "scanner"},
	{Name: "sqlmap", Binary: "sqlmap",
		Pkg: "sqlmap", Method: MethodPip3, Category: "scanner"},
	{Name: "arjun", Binary: "arjun",
		Pkg: "arjun", Method: MethodPip3, Category: "scanner"},
	// ── Crawlers ──────────────────────────────────────────────────────────────
	{Name: "katana", Binary: "katana",
		Pkg: "github.com/projectdiscovery/katana/cmd/katana@latest",
		Method: MethodGo, Category: "crawler"},
	{Name: "gospider", Binary: "gospider",
		Pkg: "github.com/jaeles-project/gospider@latest",
		Method: MethodGo, Category: "crawler"},
	{Name: "censys", Binary: "censys",
		Pkg: "censys", Method: MethodPip3, Category: "crawler", Optional: true},
	// ── AI Backends (all optional — only one is needed) ───────────────────────
	{Name: "claude", Binary: "claude",
		Method: MethodManual, Category: "ai", Optional: true,
		ManualURL: "https://claude.ai/download"},
	{Name: "ollama", Binary: "ollama",
		Method: MethodManual, Category: "ai", Optional: true,
		ManualURL: "https://ollama.ai"},
	{Name: "codex", Binary: "codex",
		Method: MethodManual, Category: "ai", Optional: true,
		ManualURL: "npm install -g @openai/codex"},
}

// Check returns the install status of every dependency.
func Check() []Result {
	results := make([]Result, len(All))
	for i, dep := range All {
		_, err := exec.LookPath(dep.Binary)
		results[i] = Result{Dep: dep, Installed: err == nil}
	}
	return results
}

// RunStartupCheck prints a dependency table, auto-installs missing Go tools,
// and prints install commands for Python/manual tools.
// Returns the count of required (non-optional) tools still missing after the run.
func RunStartupCheck(skipInstall bool) int {
	results := Check()

	// Fast path: everything present.
	missing := onlyMissing(results)
	if len(missing) == 0 {
		fmt.Printf("%s✓ All dependencies are installed.%s\n", green, reset)
		return 0
	}

	printTable(results)

	goMissing := byMethod(missing, MethodGo)
	pipMissing := byMethod(missing, MethodPip3)
	manualMissing := byMethod(missing, MethodManual)

	if !skipInstall && len(goMissing) > 0 {
		fmt.Printf("\n%s  Installing %d missing Go tool(s) in parallel...%s\n", yellow, len(goMissing), reset)
		installGoParallel(goMissing)

		// Invalidate the tools.IsToolAvailable cache by re-checking via LookPath directly here;
		// the orchestrator cache will self-populate on first use after tools land in PATH.
		fmt.Printf("\n  %sRe-verifying installs...%s\n", dim, reset)
		for _, r := range goMissing {
			if _, err := exec.LookPath(r.Dep.Binary); err != nil {
				// Check GOPATH/bin as fallback (binary installed but not yet in PATH)
				if goBin := goBinPath(r.Dep.Binary); goBin != "" {
					fmt.Printf("  %s⚠  %s installed to %s but not in PATH.%s\n",
						yellow, r.Dep.Binary, filepath.Dir(goBin), reset)
					fmt.Printf("     Add to shell profile: %sexport PATH=$PATH:%s%s\n",
						dim, filepath.Dir(goBin), reset)
				} else {
					fmt.Printf("  %s✗  %s install may have failed. Check 'go install' output above.%s\n",
						red, r.Dep.Binary, reset)
				}
			} else {
				fmt.Printf("  %s✓  %s%s\n", green, r.Dep.Binary, reset)
			}
		}
	} else if skipInstall && len(goMissing) > 0 {
		pkgs := make([]string, len(goMissing))
		for i, r := range goMissing {
			pkgs[i] = "go install " + r.Dep.Pkg
		}
		fmt.Printf("\n%s  Missing Go tools — install with:%s\n", yellow, reset)
		for _, cmd := range pkgs {
			fmt.Printf("    %s%s%s\n", dim, cmd, reset)
		}
	}

	if len(pipMissing) > 0 {
		pkgs := make([]string, len(pipMissing))
		for i, r := range pipMissing {
			pkgs[i] = r.Dep.Pkg
		}
		fmt.Printf("\n%s  Python tools — install with:%s\n", yellow, reset)
		fmt.Printf("    %spip3 install %s%s\n", dim, strings.Join(pkgs, " "), reset)
	}

	if len(manualMissing) > 0 {
		nonAI := []Result{}
		for _, r := range manualMissing {
			if !r.Dep.Optional {
				nonAI = append(nonAI, r)
			}
		}
		if len(nonAI) > 0 {
			fmt.Printf("\n%s  Manual installs required:%s\n", yellow, reset)
			for _, r := range nonAI {
				fmt.Printf("    %-12s → %s%s%s\n", r.Dep.Binary, dim, r.Dep.ManualURL, reset)
			}
		}
		// AI backends: only show if none of them are installed
		anyAIInstalled := false
		for _, r := range results {
			if r.Dep.Category == "ai" && r.Installed {
				anyAIInstalled = true
				break
			}
		}
		if !anyAIInstalled {
			fmt.Printf("\n%s  No AI backend found. Install at least one:%s\n", yellow, reset)
			for _, r := range manualMissing {
				if r.Dep.Category == "ai" {
					fmt.Printf("    %-12s → %s%s%s\n", r.Dep.Binary, dim, r.Dep.ManualURL, reset)
				}
			}
		}
	}

	// Re-count required missing after potential installs.
	finalResults := Check()
	missingRequired := 0
	for _, r := range finalResults {
		if !r.Installed && !r.Dep.Optional {
			missingRequired++
		}
	}
	return missingRequired
}

// ── internal helpers ──────────────────────────────────────────────────────────

func printTable(results []Result) {
	categories := []string{"scanner", "crawler", "ai"}
	labels := map[string]string{
		"scanner": "SCANNERS",
		"crawler": "CRAWLERS",
		"ai":      "AI BACKENDS  (pick one — all optional)",
	}

	fmt.Printf("\n%s%s  AUTOSCOUT — DEPENDENCY CHECK  %s%s\n\n", bold, cyan, reset, reset)

	for _, cat := range categories {
		printed := false
		for _, r := range results {
			if r.Dep.Category != cat {
				continue
			}
			if !printed {
				fmt.Printf("  %s%s%s%s\n", bold, dim, labels[cat], reset)
				printed = true
			}
			status := fmt.Sprintf("%s✓%s", green, reset)
			if !r.Installed {
				status = fmt.Sprintf("%s✗%s", red, reset)
			}
			optional := ""
			if r.Dep.Optional {
				optional = fmt.Sprintf("  %s(optional)%s", dim, reset)
			}
			fmt.Printf("    %s  %-14s%s\n", status, r.Dep.Binary, optional)
		}
		if printed {
			fmt.Println()
		}
	}
}

// installGoParallel runs `go install` for each missing tool concurrently,
// streaming per-tool results as they complete.
func installGoParallel(missing []Result) {
	type outcome struct {
		name    string
		elapsed time.Duration
		err     error
	}
	ch := make(chan outcome, len(missing))
	var wg sync.WaitGroup

	for _, r := range missing {
		wg.Add(1)
		go func(dep Dep) {
			defer wg.Done()
			start := time.Now()
			cmd := exec.Command("go", "install", dep.Pkg)
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			err := cmd.Run()
			ch <- outcome{name: dep.Binary, elapsed: time.Since(start), err: err}
		}(r.Dep)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for o := range ch {
		if o.err != nil {
			fmt.Printf("    %s✗  %s failed: %v%s\n", red, o.name, o.err, reset)
		} else {
			fmt.Printf("    %s✓  %-12s%s %s(%ds)%s\n",
				green, o.name, reset, dim, int(o.elapsed.Seconds()), reset)
		}
	}
}

// goBinPath returns the full path to binary in GOPATH/bin, or "" if not found.
func goBinPath(binary string) string {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	p := filepath.Join(gopath, "bin", binary)
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

func onlyMissing(results []Result) []Result {
	var out []Result
	for _, r := range results {
		if !r.Installed {
			out = append(out, r)
		}
	}
	return out
}

func byMethod(results []Result, m InstallMethod) []Result {
	var out []Result
	for _, r := range results {
		if r.Dep.Method == m {
			out = append(out, r)
		}
	}
	return out
}

// ANSI color codes — avoids a lipgloss import for pre-GUI terminal output.
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	cyan   = "\033[36m"
)
