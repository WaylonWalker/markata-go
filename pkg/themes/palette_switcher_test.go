package themes

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestThemePicker_ResetButtonIsLabeledInBothTemplates(t *testing.T) {
	resetButton := regexp.MustCompile(`(?s)<button[^>]*data-picker-reset[^>]*>\s*<svg.*?</svg>\s*<span>Reset</span>\s*</button>`)

	for _, tc := range []struct {
		name string
		read func() ([]byte, error)
	}{
		{"embedded theme", func() ([]byte, error) {
			return fs.ReadFile(DefaultTemplates(), "partials/palette-switcher.html")
		}},
		{"root templates", func() ([]byte, error) {
			return os.ReadFile(filepath.Join("..", "..", "templates", "partials", "palette-switcher.html"))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content, err := tc.read()
			if err != nil {
				t.Fatal(err)
			}
			if !resetButton.Match(content) {
				t.Error("theme picker reset control must show a visible Reset label")
			}
		})
	}
}
