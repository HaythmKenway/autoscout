package notifier_test

import (
	"testing"

	"github.com/HaythmKenway/autoscout/pkg/notifier"
)

func TestClassifyNotification(t *testing.T) {
	notifier.ClassifyNotification([]string{})

	urls := []string{"https://example.com", "https://test.com"}
	notifier.ClassifyNotification(urls)
}
