package subdomain

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"sort"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/gologger/levels" // Import levels
	"github.com/projectdiscovery/subfinder/v2/pkg/runner"
)

// 1. Create a custom adapter struct
type logAdapter struct {
	w io.Writer
}

// 2. Implement the specific Write method gologger demands
// It wants Write([]byte, levels.Level), not the standard Write([]byte) (int, error)
func (l *logAdapter) Write(data []byte, level levels.Level) {
	l.w.Write(data)
}

func Subdomain(domain string) ([]string, error) {
	// Standardize path using your utility so it matches the rest of the app
	logPath := localUtils.GetWorkingDirectory() + "/go.log"

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		// 3. Use the adapter to wrap the file
		adapter := &logAdapter{w: f}
		gologger.DefaultLogger.SetWriter(adapter)
	} else {
		// Wrap io.Discard if file fails
		adapter := &logAdapter{w: io.Discard}
		gologger.DefaultLogger.SetWriter(adapter)
	}

	localUtils.Logger("performing subdomain Enumeration for "+domain, 1)

	subfinderOpts := &runner.Options{
		Threads:            10,
		Timeout:            30,
		MaxEnumerationTime: 10,
		Silent:             true,
	}

	log.SetFlags(0)

	subfinder, err := runner.NewRunner(subfinderOpts)
	if err != nil {
		return nil, err
	}

	output := &bytes.Buffer{}
	if err := subfinder.EnumerateSingleDomainWithCtx(context.Background(), domain, []io.Writer{output}); err != nil {
		return nil, err
	}

	subdomains := localUtils.ParseSubdomains(output.String())
	sort.Strings(subdomains)
	localUtils.Logger("subdomain Enumeration for "+domain+" completed", 1)
	return subdomains, nil
}
