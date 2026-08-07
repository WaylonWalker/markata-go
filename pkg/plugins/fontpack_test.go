package plugins

import (
	"strings"
	"testing"
)

func TestMarkHTMLFontpackReplacesExactlyOneAttribute(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"exact", "<html>", `<html data-fontpack="field-notebook">`},
		{"attributes", `<html lang="en">`, `<html lang="en" data-fontpack="field-notebook">`},
		{"uppercase", `<HTML class="site">`, `<HTML class="site" data-fontpack="field-notebook">`},
		{"replace", `<html data-fontpack="old" lang="en">`, `<html data-fontpack="field-notebook" lang="en">`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := markHTMLFontpack(tt.in, "field-notebook")
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
			if strings.Count(strings.ToLower(got), "data-fontpack=") != 1 {
				t.Fatalf("duplicate data-fontpack attribute: %q", got)
			}
		})
	}
}
