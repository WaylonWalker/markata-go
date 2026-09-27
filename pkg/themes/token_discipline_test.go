package themes

import (
	"strings"
	"testing"
)

func TestComponentTypographyUsesSharedScale(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		required  []string
		forbidden []string
	}{
		{
			name: "encrypted content",
			path: "css/encryption.css",
			required: []string{
				"font-size: var(--text-2xl);",
				"font-size: var(--text-sm);",
			},
			forbidden: []string{
				"font-size: 1.5rem;",
				"font-size: 0.875rem;",
			},
		},
		{
			name: "mermaid controls",
			path: "css/mermaid.css",
			required: []string{
				"font-size: var(--text-sm);",
			},
			forbidden: []string{
				"font-size: 0.8125rem;",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := ReadStatic(tt.path)
			if err != nil {
				t.Fatalf("read %s: %v", tt.path, err)
			}
			css := string(data)
			for _, required := range tt.required {
				if !strings.Contains(css, required) {
					t.Errorf("%s missing shared type-scale declaration %q", tt.path, required)
				}
			}
			for _, forbidden := range tt.forbidden {
				if strings.Contains(css, forbidden) {
					t.Errorf("%s still contains one-off type size %q", tt.path, forbidden)
				}
			}
		})
	}
}

func TestDefaultThemeDefinesSecondaryTextSemanticToken(t *testing.T) {
	data, err := ReadStatic("css/variables.css")
	if err != nil {
		t.Fatalf("read variables.css: %v", err)
	}
	css := string(data)
	if !strings.Contains(css, "--color-text-secondary: var(--color-text-muted);") {
		t.Error("variables.css should define secondary text as a semantic fallback to muted text")
	}
}
