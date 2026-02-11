// ABOUTME: Tests for embedded hook content
// ABOUTME: Verifies all hook files are embedded and non-empty

package hooks

import (
	"testing"
)

func TestEmbeddedFilesExist(t *testing.T) {
	files := Files()

	expected := []string{
		"relay-start.sh",
		"relay-poll.sh",
		"relay-end.sh",
		"relay-resolve.sh",
	}

	for _, name := range expected {
		content, ok := files[name]
		if !ok {
			t.Errorf("missing embedded file: %s", name)
			continue
		}
		if len(content) == 0 {
			t.Errorf("empty embedded file: %s", name)
		}
	}
}

func TestSettingsContent(t *testing.T) {
	content := Settings()
	if len(content) == 0 {
		t.Error("settings content is empty")
	}
}
