package subdomain_test

import (
	"testing"

	"github.com/HaythmKenway/autoscout/pkg/subdomain"
)

func TestSubdomain(t *testing.T) {
	domain := "example.com"

	subdomains, err := subdomain.Subdomain(domain)
	if err != nil {
		t.Errorf("Error while enumerating subdomains: %v", err)
	}

	if len(subdomains) == 0 {
		t.Errorf("No subdomains found for domain %s", domain)
	}

	t.Logf("Subdomains for domain %s: %v", domain, subdomains)
}
