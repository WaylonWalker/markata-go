package csspurge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanHTMLContent(t *testing.T) {
	tests := []struct {
		name           string
		html           string
		wantClasses    []string
		wantIDs        []string
		wantElements   []string
		wantAttributes []string
	}{
		{
			name: "basic elements",
			html: `<!DOCTYPE html>
<html><head><title>Test</title></head>
<body><div><p>Hello</p></div></body></html>`,
			wantElements: []string{"html", "head", "title", "body", "div", "p"},
		},
		{
			name:        "classes",
			html:        `<div class="container main-content"><p class="text-red bold">Hello</p></div>`,
			wantClasses: []string{"container", "main-content", "text-red", "bold"},
		},
		{
			name:    "IDs",
			html:    `<div id="header"><nav id="main-nav">Nav</nav></div>`,
			wantIDs: []string{"header", "main-nav"},
		},
		{
			name:           "attributes",
			html:           `<input type="text" name="email" data-validate="email">`,
			wantAttributes: []string{"type", "name", "data-validate"},
		},
		{
			name:         "mixed content",
			html:         `<article class="post" id="article-1" data-category="tech"><h1 class="title">Title</h1></article>`,
			wantClasses:  []string{"post", "title"},
			wantIDs:      []string{"article-1"},
			wantElements: []string{"article", "h1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			used := NewUsedSelectors()
			if err := ScanHTMLContent(tt.html, used); err != nil {
				t.Fatalf("ScanHTMLContent() error = %v", err)
			}

			for _, class := range tt.wantClasses {
				if !used.Classes[class] {
					t.Errorf("expected class %q not found", class)
				}
			}

			for _, id := range tt.wantIDs {
				if !used.IDs[id] {
					t.Errorf("expected ID %q not found", id)
				}
			}

			for _, elem := range tt.wantElements {
				if !used.Elements[elem] {
					t.Errorf("expected element %q not found", elem)
				}
			}

			for _, attr := range tt.wantAttributes {
				if !used.Attributes[attr] {
					t.Errorf("expected attribute %q not found", attr)
				}
			}
		})
	}
}

func TestScanHTML(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	htmlPath := filepath.Join(tmpDir, "test.html")
	content := `<div class="container"><span class="text">Hello</span></div>`
	if err := os.WriteFile(htmlPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	used := NewUsedSelectors()
	if err := ScanHTML(htmlPath, used); err != nil {
		t.Fatalf("ScanHTML() error = %v", err)
	}

	if !used.Classes["container"] {
		t.Error("expected class 'container' not found")
	}
	if !used.Classes["text"] {
		t.Error("expected class 'text' not found")
	}
}

func TestUsedSelectorsMerge(t *testing.T) {
	u1 := NewUsedSelectors()
	u1.Classes["a"] = true
	u1.IDs["id1"] = true

	u2 := NewUsedSelectors()
	u2.Classes["b"] = true
	u2.IDs["id2"] = true

	u1.Merge(u2)

	if !u1.Classes["a"] || !u1.Classes["b"] {
		t.Error("classes not merged correctly")
	}
	if !u1.IDs["id1"] || !u1.IDs["id2"] {
		t.Error("IDs not merged correctly")
	}
}
