package spider

import (
	"os/exec"
	"strings"
	"sort"
	
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

func Spider(domain string) ([]string,error) {
	cmd := exec.Command("gau", domain)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil,err	}
	outputString := string(output)
	pipe:=exec.Command("grep","^h")
	pipe.Stdin= strings.NewReader(outputString)
	output1, err := pipe.CombinedOutput()
	if err != nil {
		return nil,err
	}

	lines := strings.Split(string(output1),"\n")
	sort.Strings(lines)
	lines = localUtils.RemoveDuplicates(lines)
	return lines,nil
}
