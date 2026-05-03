package spider

import (
	"reflect"
	"testing"
)

func TestParseSpiderOutput(t *testing.T) {
	input := "https://example.com/b\nftp://example.com/file\nhttps://example.com/a\nhttps://example.com/a\n"

	got := ParseSpiderOutput(input)
	want := []string{"ftp://example.com/file", "https://example.com/a", "https://example.com/b"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseSpiderOutput() = %#v, want %#v", got, want)
	}
}
