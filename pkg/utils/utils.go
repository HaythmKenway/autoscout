package utils 

import (
	"os"
	"os/user"
	"fmt"
	"path/filepath"
	"strings"
	"regexp"
)

func GetWorkingDirectory() string{
	usr,err:=user.Current()
	dirPath := filepath.Join(usr.HomeDir, ".autoscout")

	_, err = os.Stat(dirPath)
	if os.IsNotExist(err) {
		err = os.Mkdir(dirPath, 0755) // 0755 is the permission mode
		if err != nil {
			fmt.Println("Error creating directory:", err)
			os.Exit(1)
		}
		return dirPath
	} else if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	} 
return dirPath}

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
    // Create a map to store elements from prev.
    elementsInPrev := make(map[string]struct{})

    // Iterate through prev and store elements in the map.
    for _, p := range prev {
        elementsInPrev[p] = struct{}{}
    }

    // Find elements that are in now but not in prev.
    elementsOnlyInNow := []string{}
    for _, n := range now {
        if _, exists := elementsInPrev[n]; !exists {
            elementsOnlyInNow = append(elementsOnlyInNow, n)
        }
    }

    return elementsOnlyInNow
}
