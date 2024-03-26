package localUtils

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
)

func GetWorkingDirectory() string {
	usr, err := user.Current()
	dirPath := filepath.Join(usr.HomeDir, ".autoscout")

	_, err = os.Stat(dirPath)
	if os.IsNotExist(err) {
		err = os.Mkdir(dirPath, 0755)
		if err != nil {
			os.Exit(1)
		}
		return dirPath
	} else if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	return dirPath
}

func ParseSubdomains(output string) []string {
	var subdomains []string

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		subdomain := strings.TrimSpace(line)
		if subdomain != "" {
			subdomains = append(subdomains, subdomain)
		}
	}

	return subdomains
}
func RemoveSpecialCharacters(input string) string {

	regex := regexp.MustCompile("[^a-zA-Z0-9\\s]+")
	processedString := regex.ReplaceAllString(input, "")
	return processedString
}
func ElementsOnlyInNow(prev []string, now []string) []string {
	elementsInPrev := make(map[string]struct{})

	for _, p := range prev {
		elementsInPrev[p] = struct{}{}
	}

	elementsOnlyInNow := []string{}
	for _, n := range now {
		if _, exists := elementsInPrev[n]; !exists {
			elementsOnlyInNow = append(elementsOnlyInNow, n)
		}
	}

	return elementsOnlyInNow

}
func Logger(str string, sc int) {
	f, err := os.OpenFile(GetWorkingDirectory()+"/go.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)
	switch sc {
	case 1:
		log.Info(str)
	case 2:
		log.Error(str)
	case 3:
		log.Debug(str)

	}

}
