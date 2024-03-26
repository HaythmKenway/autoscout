package localUtils

import (
	"os"
	"testing"
)

func TestGetWorkingDirectory(t *testing.T) {
	// Test for the directory creation
	dirPath := GetWorkingDirectory()
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Errorf("Directory %s was not created", dirPath)
	}
}

func TestParseSubdomains(t *testing.T) {
	output := "subdomain1\nsubdomain2\nsubdomain3\n"

	expected := []string{"subdomain1", "subdomain2", "subdomain3"}

	subdomains := ParseSubdomains(output)

	if len(subdomains) != len(expected) {
		t.Errorf("Expected %d subdomains, but got %d", len(expected), len(subdomains))
	}

	for i := range subdomains {
		if subdomains[i] != expected[i] {
			t.Errorf("Expected subdomain '%s', but got '%s'", expected[i], subdomains[i])
		}
	}
}

func TestRemoveSpecialCharacters(t *testing.T) {
	input := "abc!@#123"
	expected := "abc123"

	output := RemoveSpecialCharacters(input)

	if output != expected {
		t.Errorf("Expected '%s' after removing special characters, but got '%s'", expected, output)
	}
}

func TestElementsOnlyInNow(t *testing.T) {
	prev := []string{"a", "b", "c"}
	now := []string{"b", "c", "d"}

	expected := []string{"d"}

	elements := ElementsOnlyInNow(prev, now)

	if len(elements) != len(expected) {
		t.Errorf("Expected %d elements only in 'now', but got %d", len(expected), len(elements))
	}

	for i := range elements {
		if elements[i] != expected[i] {
			t.Errorf("Expected element '%s' only in 'now', but got '%s'", expected[i], elements[i])
		}
	}
}
