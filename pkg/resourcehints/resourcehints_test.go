package resourcehints

import (
	"testing"
)

func TestDetector_DetectExternalDomains(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected []string
	}{
		{
			name: "Google Fonts",
			html: `<link href="https://fonts.googleapis.com/css2?family=Inter" rel="stylesheet">
                   <link href="https://fonts.gstatic.com" rel="preconnect" crossorigin>`,
			expected: []string{"fonts.googleapis.com", "fonts.gstatic.com"},
		},
		{
			name:     "CDN script",
			html:     `<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>`,
			expected: []string{"cdn.jsdelivr.net"},
		},
		{
			name:     "CSS url()",
			html:     `<style>background: url("https://images.unsplash.com/photo-123.jpg");</style>`,
			expected: []string{"images.unsplash.com"},
		},
		{
			name:     "Protocol-relative URL",
			html:     `<script src="//cdn.example.com/script.js"></script>`,
			expected: []string{"cdn.example.com"},
		},
		{
			name:     "Skip relative URLs",
			html:     `<link href="/css/style.css" rel="stylesheet"><script src="./js/app.js"></script>`,
			expected: []string{},
		},
		{
			name:     "Skip localhost",
			html:     `<script src="http://localhost:3000/script.js"></script>`,
			expected: []string{},
		},
		{
			name: "Multiple domains deduplicated",
			html: `<link href="https://cdn.jsdelivr.net/a.css" rel="stylesheet">
                   <script src="https://cdn.jsdelivr.net/b.js"></script>`,
			expected: []string{"cdn.jsdelivr.net"},
		},
		{
			name:     "YouTube embed",
			html:     `<iframe src="https://www.youtube-nocookie.com/embed/abc123"></iframe>`,
			expected: []string{"www.youtube-nocookie.com"},
		},
	}

	detector := NewDetector()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domains := detector.DetectExternalDomains(tt.html)

			// Convert to map for easier checking
			gotDomains := make(map[string]bool)
			for _, d := range domains {
				gotDomains[d.Domain] = true
			}

			// Check expected domains are present
			for _, expected := range tt.expected {
				if !gotDomains[expected] {
					t.Errorf("Expected domain %q not found", expected)
				}
			}

			// Check no unexpected domains
			if len(domains) != len(tt.expected) {
				var got []string
				for _, d := range domains {
					got = append(got, d.Domain)
				}
				t.Errorf("Got %d domains %v, expected %d domains %v", len(domains), got, len(tt.expected), tt.expected)
			}
		})
	}
}

func TestDetector_ExcludeDomains(t *testing.T) {
	html := `<script src="https://cdn.jsdelivr.net/a.js"></script>
             <script src="https://unpkg.com/b.js"></script>`

	detector := NewDetector()
	detector.SetExcludeDomains([]string{"cdn.jsdelivr.net"})

	domains := detector.DetectExternalDomains(html)

	if len(domains) != 1 {
		t.Errorf("Expected 1 domain, got %d", len(domains))
	}

	if len(domains) > 0 && domains[0].Domain != "unpkg.com" {
		t.Errorf("Expected 'unpkg.com', got %q", domains[0].Domain)
	}
}

func TestDetector_SuggestHints(t *testing.T) {
	tests := []struct {
		name          string
		domain        string
		expectedTypes []HintType
		expectedCross string
	}{
		{
			name:          "Google Fonts API",
			domain:        "fonts.googleapis.com",
			expectedTypes: []HintType{HintTypePreconnect},
			expectedCross: "",
		},
		{
			name:          "Google Fonts Static",
			domain:        "fonts.gstatic.com",
			expectedTypes: []HintType{HintTypePreconnect},
			expectedCross: "anonymous",
		},
		{
			name:          "CDN domain",
			domain:        "cdn.jsdelivr.net",
			expectedTypes: []HintType{HintTypeDNSPrefetch},
			expectedCross: "",
		},
		{
			name:          "Unknown domain",
			domain:        "example.com",
			expectedTypes: []HintType{HintTypeDNSPrefetch},
			expectedCross: "",
		},
	}

	detector := NewDetector()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domains := []DetectedDomain{{Domain: tt.domain, Scheme: "https"}}
			hints := detector.SuggestHints(domains)

			if len(hints) != 1 {
				t.Fatalf("Expected 1 hint, got %d", len(hints))
			}

			hint := hints[0]

			if len(hint.HintTypes) != len(tt.expectedTypes) {
				t.Errorf("Expected %d hint types, got %d", len(tt.expectedTypes), len(hint.HintTypes))
			}

			for i, expected := range tt.expectedTypes {
				if i < len(hint.HintTypes) && hint.HintTypes[i] != expected {
					t.Errorf("Expected hint type %q, got %q", expected, hint.HintTypes[i])
				}
			}

			if hint.CrossOrigin != tt.expectedCross {
				t.Errorf("Expected crossorigin %q, got %q", tt.expectedCross, hint.CrossOrigin)
			}
		})
	}
}

func TestGenerator_GenerateHintTags(t *testing.T) {
	tests := []struct {
		name     string
		hints    []SuggestedHint
		contains []string
	}{
		{
			name: "Preconnect without crossorigin",
			hints: []SuggestedHint{
				{
					Domain:    "fonts.googleapis.com",
					Scheme:    "https",
					HintTypes: []HintType{HintTypePreconnect},
				},
			},
			contains: []string{
				`rel="preconnect"`,
				`href="https://fonts.googleapis.com"`,
			},
		},
		{
			name: "Preconnect with crossorigin",
			hints: []SuggestedHint{
				{
					Domain:      "fonts.gstatic.com",
					Scheme:      "https",
					HintTypes:   []HintType{HintTypePreconnect},
					CrossOrigin: "anonymous",
				},
			},
			contains: []string{
				`rel="preconnect"`,
				`href="https://fonts.gstatic.com"`,
				`crossorigin`,
			},
		},
		{
			name: "DNS-prefetch",
			hints: []SuggestedHint{
				{
					Domain:    "cdn.jsdelivr.net",
					Scheme:    "https",
					HintTypes: []HintType{HintTypeDNSPrefetch},
				},
			},
			contains: []string{
				`rel="dns-prefetch"`,
				`href="https://cdn.jsdelivr.net"`,
			},
		},
		{
			name:     "Empty hints",
			hints:    []SuggestedHint{},
			contains: []string{},
		},
	}

	generator := NewGenerator()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.GenerateHintTags(tt.hints)

			if len(tt.contains) == 0 {
				if result != "" {
					t.Errorf("Expected empty result, got %q", result)
				}
				return
			}

			for _, expected := range tt.contains {
				if expected != "" && !containsString(result, expected) {
					t.Errorf("Result should contain %q, got:\n%s", expected, result)
				}
			}
		})
	}
}

func TestGenerator_SortOrder(t *testing.T) {
	hints := []SuggestedHint{
		{Domain: "c.com", Scheme: "https", HintTypes: []HintType{HintTypePrefetch}},
		{Domain: "a.com", Scheme: "https", HintTypes: []HintType{HintTypePreconnect}},
		{Domain: "b.com", Scheme: "https", HintTypes: []HintType{HintTypeDNSPrefetch}},
	}

	generator := NewGenerator()
	result := generator.GenerateHintTags(hints)

	// Preconnect should come before dns-prefetch which should come before prefetch
	preconnectIdx := indexOf(result, "preconnect")
	dnsPrefetchIdx := indexOf(result, "dns-prefetch")
	prefetchIdx := lastIndexOf(result, "prefetch") // Use last to avoid matching dns-prefetch

	if preconnectIdx > dnsPrefetchIdx {
		t.Error("preconnect should come before dns-prefetch")
	}
	if dnsPrefetchIdx > prefetchIdx {
		t.Error("dns-prefetch should come before prefetch")
	}
}

func TestGenerateComment(t *testing.T) {
	content := `<link rel="preconnect" href="https://example.com">`
	result := GenerateComment(content)

	if !containsString(result, "<!-- Auto-generated resource hints -->") {
		t.Error("Should contain opening comment")
	}
	if !containsString(result, "<!-- End resource hints -->") {
		t.Error("Should contain closing comment")
	}
	if !containsString(result, content) {
		t.Error("Should contain original content")
	}

	// Empty content should return empty
	if GenerateComment("") != "" {
		t.Error("Empty content should return empty string")
	}
}

// Helper functions

func containsString(s, substr string) bool {
	return indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func lastIndexOf(s, substr string) int {
	last := -1
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			last = i
		}
	}
	return last
}
