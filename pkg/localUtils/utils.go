package localUtils

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func GetWorkingDirectory() string {
	usr, err := user.Current()
	dirPath := filepath.Join(usr.HomeDir, ".autoscout")

	_, err = os.Stat(dirPath)
	if os.IsNotExist(err) {
		err = os.Mkdir(dirPath, 0755)
		CheckError(err)
		return dirPath
	}
	CheckError(err)
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
	return regex.ReplaceAllString(input, "")
}

func ElementsOnlyInNow(prev []string, now []string) []string {
	elementsInPrev := make(map[string]struct{})

	for _, p := range prev {
		elementsInPrev[p] = struct{}{}
	}

	var elementsOnlyInNow []string
	for _, n := range now {
		if _, exists := elementsInPrev[n]; !exists {
			elementsOnlyInNow = append(elementsOnlyInNow, n)
		}
	}

	return elementsOnlyInNow
}

func CheckError(err error) {
	if err != nil {
		Logger(err.Error(), 2)
	}
}

func Logger(str string, sc int) {
	dir := GetWorkingDirectory()
	f, err := os.OpenFile(dir+"/go.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	timestamp := time.Now().Format("2006/01/02 15:04:05")
	var prefix string
	switch sc {
	case 1:
		prefix = "INFO"
	case 2:
		prefix = "ERRO"
	case 3:
		prefix = "DEBU"
	default:
		prefix = "INFO"
	}

	line := fmt.Sprintf("%s %s %s\n", prefix, timestamp, str)
	f.WriteString(line)
	f.Sync()
}
func RemoveDuplicates(arr []string) []string {
	uniqueMap := make(map[string]struct{})
	for _, elem := range arr {
		uniqueMap[elem] = struct{}{}
	}
	unique := make([]string, 0, len(uniqueMap))
	for key := range uniqueMap {
		unique = append(unique, key)
	}
	return unique
}
