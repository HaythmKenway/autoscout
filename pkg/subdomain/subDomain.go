package subdomain

import (
	"bytes"
	"os/exec"
	"sort"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

func Subdomain(domain string, rateLimit string) ([]string, error) {
	if rateLimit == "" {
		rateLimit = localUtils.GetRateLimit()
	}

	args := []string{"-d", domain, "-silent", "-nc"}
	
	// Subfinder doesn't have a direct rate limit flag, but we use threads
	args = append(args, "-t", "10")

	cmd := exec.Command("subfinder", args...)
	
	output := &bytes.Buffer{}
	// We'll use a local buffer here for the scheduler, 
	// but the RunSubfinder wrapper in tools.go will handle the JobManager.
	cmd.Stdout = output
	
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	subdomains := parseSubfinderOutput(output.String())
	return subdomains, nil
}

func parseSubfinderOutput(output string) []string {
	subdomains := localUtils.ParseSubdomains(output)
	sort.Strings(subdomains)
	return subdomains
}
