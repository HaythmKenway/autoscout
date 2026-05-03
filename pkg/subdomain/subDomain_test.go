package subdomain

import (
	"reflect"
	"testing"
)

func TestParseSubfinderOutput(t *testing.T) {
	input := " beta.example.com \n\nalpha.example.com\n"

	got := parseSubfinderOutput(input)
	want := []string{"alpha.example.com", "beta.example.com"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseSubfinderOutput() = %#v, want %#v", got, want)
	}
}
