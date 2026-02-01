package criticalcss

import (
	"strings"
	"testing"
)

func TestExtractor_Extract_BasicSelectors(t *testing.T) {
	ext := NewExtractor()

	css := `
body {
	margin: 0;
	padding: 0;
}

.some-random-class {
	color: red;
}

header {
	background: blue;
}
`

	result, err := ext.Extract(css)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Body and header should be critical
	if !strings.Contains(result.Critical, "body{") && !strings.Contains(result.Critical, "body {") {
		t.Error("Expected 'body' to be in critical CSS")
	}
	if !strings.Contains(result.Critical, "header{") && !strings.Contains(result.Critical, "header {") {
		t.Error("Expected 'header' to be in critical CSS")
	}

	// Random class should not be critical
	if strings.Contains(result.Critical, "some-random-class") {
		t.Error("Expected '.some-random-class' to NOT be in critical CSS")
	}
	if !strings.Contains(result.NonCritical, "some-random-class") {
		t.Error("Expected '.some-random-class' to be in non-critical CSS")
	}
}

func TestExtractor_Extract_ClassSelectors(t *testing.T) {
	ext := NewExtractor()

	css := `
.site-header {
	display: flex;
}

.card {
	padding: 1rem;
}

.unusual-widget {
	animation: spin 1s;
}
`

	result, err := ext.Extract(css)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// .site-header and .card should be critical
	if !strings.Contains(result.Critical, "site-header") {
		t.Error("Expected '.site-header' to be in critical CSS")
	}
	if !strings.Contains(result.Critical, "card") {
		t.Error("Expected '.card' to be in critical CSS")
	}

	// .unusual-widget should not be critical
	if strings.Contains(result.Critical, "unusual-widget") {
		t.Error("Expected '.unusual-widget' to NOT be in critical CSS")
	}
}

func TestExtractor_Extract_MediaQueries(t *testing.T) {
	ext := NewExtractor()

	css := `
body {
	font-size: 16px;
}

@media (max-width: 768px) {
	body {
		font-size: 14px;
	}
	.unusual-widget {
		display: none;
	}
}
`

	result, err := ext.Extract(css)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// body in media query should be critical
	if !strings.Contains(result.Critical, "@media") {
		t.Error("Expected @media rule to be in critical CSS")
	}

	// The media query should be split - body is critical, .unusual-widget is not
	if strings.Contains(result.Critical, "unusual-widget") {
		t.Error("Expected '.unusual-widget' to NOT be in critical CSS media query")
	}
}

func TestExtractor_Extract_WithExtraSelectors(t *testing.T) {
	ext := NewExtractor().WithSelectors([]string{".my-custom-class"})

	css := `
.my-custom-class {
	color: blue;
}

.another-class {
	color: red;
}
`

	result, err := ext.Extract(css)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// .my-custom-class should now be critical
	if !strings.Contains(result.Critical, "my-custom-class") {
		t.Error("Expected '.my-custom-class' to be in critical CSS")
	}

	// .another-class should not be critical
	if strings.Contains(result.Critical, "another-class") {
		t.Error("Expected '.another-class' to NOT be in critical CSS")
	}
}

func TestExtractor_Extract_WithExcludeSelectors(t *testing.T) {
	ext := NewExtractor().WithExcludeSelectors([]string{"body"})

	css := `
body {
	margin: 0;
}

header {
	background: blue;
}
`

	result, err := ext.Extract(css)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// body should be excluded
	if strings.Contains(result.Critical, "body{") {
		t.Error("Expected 'body' to be EXCLUDED from critical CSS")
	}

	// header should still be critical
	if !strings.Contains(result.Critical, "header") {
		t.Error("Expected 'header' to be in critical CSS")
	}
}

func TestExtractor_Extract_Minification(t *testing.T) {
	ext := NewExtractor().WithMinify(true)

	css := `
body {
	margin: 0;
	padding: 0;
}
`

	result, err := ext.Extract(css)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Minified output should not have extra whitespace
	if strings.Contains(result.Critical, "\n") {
		t.Error("Expected minified CSS to not contain newlines")
	}
	if strings.Contains(result.Critical, "  ") {
		t.Error("Expected minified CSS to not contain double spaces")
	}
}

func TestExtractor_Extract_NoMinification(t *testing.T) {
	ext := NewExtractor().WithMinify(false)

	css := `body {
	margin: 0;
}`

	result, err := ext.Extract(css)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Non-minified output should preserve structure
	if !strings.Contains(result.Critical, "body {") && !strings.Contains(result.Critical, "body{") {
		t.Error("Expected CSS to contain body selector")
	}
}

func TestExtractor_Extract_CompoundSelectors(t *testing.T) {
	ext := NewExtractor()

	css := `
body.dark {
	background: black;
}

.card:hover {
	transform: scale(1.1);
}

.card .card-title {
	font-weight: bold;
}

header > nav {
	display: flex;
}
`

	result, err := ext.Extract(css)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// All these should match because they start with critical selectors
	if !strings.Contains(result.Critical, "body.dark") && !strings.Contains(result.Critical, "body") {
		t.Error("Expected 'body.dark' to be in critical CSS")
	}
	if !strings.Contains(result.Critical, "card:hover") && !strings.Contains(result.Critical, "card") {
		t.Error("Expected '.card:hover' to be in critical CSS")
	}
	if !strings.Contains(result.Critical, "header") {
		t.Error("Expected 'header > nav' to be in critical CSS")
	}
}

func TestExtractor_Extract_Comments(t *testing.T) {
	ext := NewExtractor()

	css := `
/* This is a comment */
body {
	margin: 0;
}
/* Another comment */
`

	result, err := ext.Extract(css)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Comments should be removed when minifying
	if strings.Contains(result.Critical, "comment") {
		t.Error("Expected comments to be removed from minified CSS")
	}
}

func TestExtractor_Extract_MultipleSelectors(t *testing.T) {
	ext := NewExtractor()

	css := `
h1, h2, h3 {
	font-weight: bold;
}

.card, .unusual-widget {
	padding: 1rem;
}
`

	result, err := ext.Extract(css)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// h1, h2, h3 should be critical
	if !strings.Contains(result.Critical, "h1") {
		t.Error("Expected 'h1, h2, h3' to be in critical CSS")
	}

	// .card, .unusual-widget should be critical because .card matches
	if !strings.Contains(result.Critical, "card") {
		t.Error("Expected '.card, .unusual-widget' to be in critical CSS because .card matches")
	}
}

func TestExtractor_selectorMatches(t *testing.T) {
	ext := NewExtractor()

	tests := []struct {
		ruleSelector     string
		criticalSelector string
		expected         bool
	}{
		{"body", "body", true},
		{".card", ".card", true},
		{"body.dark", "body", true},
		{".card:hover", ".card", true},
		{".card .card-title", ".card", true},
		{"header > nav", "header", true},
		{".cards", ".card", false}, // Should not match partial class names
		{".card-wrapper", ".card", false},
		{"h1, h2", "h1", true},
		{".unknown", ".card", false},
	}

	for _, tt := range tests {
		t.Run(tt.ruleSelector+"_"+tt.criticalSelector, func(t *testing.T) {
			result := ext.selectorMatches(tt.ruleSelector, tt.criticalSelector)
			if result != tt.expected {
				t.Errorf("selectorMatches(%q, %q) = %v, want %v",
					tt.ruleSelector, tt.criticalSelector, result, tt.expected)
			}
		})
	}
}

func TestExtractor_parseRules(t *testing.T) {
	ext := NewExtractor()

	css := `
body { margin: 0; }
.card { padding: 1rem; }
@media (max-width: 768px) {
	body { font-size: 14px; }
}
`

	rules := ext.parseRules(css)

	if len(rules) != 3 {
		t.Errorf("Expected 3 rules, got %d: %v", len(rules), rules)
	}
}

func TestExtractor_removeComments(t *testing.T) {
	ext := NewExtractor()

	tests := []struct {
		input    string
		expected string
	}{
		{"/* comment */ body { }", " body { }"},
		{"body { /* inline */ }", "body {  }"},
		{"body { } /* end */", "body { } "},
		{"/* multi\nline\ncomment */ body { }", " body { }"},
		{"body { }", "body { }"}, // No comment
	}

	for _, tt := range tests {
		result := ext.removeComments(tt.input)
		if result != tt.expected {
			t.Errorf("removeComments(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractor_minify(t *testing.T) {
	ext := NewExtractor()

	input := `
body {
	margin: 0;
	padding: 0;
}
`

	result := ext.minify(input)

	// Check that whitespace is minimized
	if strings.Contains(result, "\n") {
		t.Error("Minified CSS should not contain newlines")
	}
	if strings.Contains(result, "\t") {
		t.Error("Minified CSS should not contain tabs")
	}
	if strings.Contains(result, "  ") {
		t.Error("Minified CSS should not contain double spaces")
	}
}

func TestExtractor_Result_Sizes(t *testing.T) {
	ext := NewExtractor()

	css := `
body { margin: 0; }
.unknown { color: red; }
`

	result, err := ext.Extract(css)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Check that sizes are calculated
	if result.CriticalSize == 0 {
		t.Error("CriticalSize should not be 0")
	}
	if result.TotalSize == 0 {
		t.Error("TotalSize should not be 0")
	}
}

func TestExtractMultiple(t *testing.T) {
	ext := NewExtractor()

	files := map[string]string{
		"main.css":    "body { margin: 0; }",
		"widgets.css": ".unknown { color: red; }",
	}

	result, err := ext.ExtractMultiple(files)
	if err != nil {
		t.Fatalf("ExtractMultiple failed: %v", err)
	}

	if !strings.Contains(result.Critical, "body") {
		t.Error("Expected body to be in critical CSS")
	}
	if strings.Contains(result.Critical, "unknown") {
		t.Error("Expected .unknown to NOT be in critical CSS")
	}
}
