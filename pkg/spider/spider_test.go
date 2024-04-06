package spider
import (
	"testing"
)
func TestSpider(t *testing.T) {
	domain := "https://example.com"
	spider,err := Spider(domain)
	if err != nil {
		t.Error(err)
	}
	if spider == nil {
		t.Error("spider is nil")
	}
}


