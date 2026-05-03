package spider

import (
	"os/exec"
	"sort"
	"strings"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

func Spider(domain string) ([]string, error) {
	cmd := exec.Command("gau", domain)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	outputString := string(output)
	pipe := exec.Command("grep", "^h")
	pipe.Stdin = strings.NewReader(outputString)
	output1, err := pipe.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := ParseSpiderOutput(string(output1))
	return lines, nil
}

func ParseSpiderOutput(output string) []string {
	lines := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	lines = localUtils.RemoveDuplicates(lines)
	sort.Strings(lines)
	return lines
}
