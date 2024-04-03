package subdomain

import (
	"bytes"
	"context"
	"io"
	"log"
	"sort"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/projectdiscovery/subfinder/v2/pkg/runner"
)

func Subdomain(domain string) ([]string, error) {
	localUtils.Logger("performing subdomain Enumeration for "+domain, 1)

	subfinderOpts := &runner.Options{
		Threads:            10,
		Timeout:            30,
		MaxEnumerationTime: 10,
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
