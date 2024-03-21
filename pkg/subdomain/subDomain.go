package subdomain

import (
	"bytes"
	"context"
	"io"
	"log"
	"sort"

	"github.com/HaythmKenway/autoscout/pkg/utils"
	"github.com/projectdiscovery/subfinder/v2/pkg/runner"
)

func Subdomain(domain string) ([]string, error) {
	subfinderOpts := &runner.Options{
		Threads:            10, // Thread controls the number of threads to use for active enumerations
		Timeout:            30, // Timeout is the seconds to wait for sources to respond
		MaxEnumerationTime: 10, // MaxEnumerationTime is the maximum amount of time in mins to wait for enumeration
	}

	log.SetFlags(0)

	subfinder, err := runner.NewRunner(subfinderOpts)
	if err != nil {
		log.Fatalf("failed to create subfinder runner: %v", err)
	}

	output := &bytes.Buffer{}
	if err = subfinder.EnumerateSingleDomainWithCtx(context.Background(), domain, []io.Writer{output}); err != nil {
		log.Fatalf("failed to enumerate single domain: %v", err)
	}

	subdomains := utils.ParseSubdomains(output.String())

	sort.Strings(subdomains)

	return subdomains, nil
}
