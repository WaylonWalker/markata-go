package plugins

import (
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestRenderMarkdownPlugin_Name(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	if p.Name() != "render_markdown" {
		t.Errorf("expected name 'render_markdown', got %q", p.Name())
	}
}

func TestRenderMarkdownPlugin_Configure(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	m := lifecycle.NewManager()
	if err := p.Configure(m); err != nil {
		t.Errorf("Configure returned error: %v", err)
	}
}

func TestRenderMarkdownPlugin_BasicParagraph(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "Hello world"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	expected := "<p>Hello world</p>"
	if !strings.Contains(post.ArticleHTML, expected) {
		t.Errorf("expected %q in output, got %q", expected, post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_Headings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"h1", "# Heading 1", "<h1"},
		{"h2", "## Heading 2", "<h2"},
		{"h3", "### Heading 3", "<h3"},
		{"h4", "#### Heading 4", "<h4"},
		{"h5", "##### Heading 5", "<h5"},
		{"h6", "###### Heading 6", "<h6"},
	}

	p := NewRenderMarkdownPlugin()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := &models.Post{Content: tt.input}
			err := p.renderPost(post)
			if err != nil {
				t.Fatalf("renderPost error: %v", err)
			}
			if !strings.Contains(post.ArticleHTML, tt.expected) {
				t.Errorf("expected %q in output, got %q", tt.expected, post.ArticleHTML)
			}
		})
	}
}

func TestRenderMarkdownPlugin_Emphasis(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "*italic* and **bold**"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, "<em>italic</em>") {
		t.Errorf("expected <em>italic</em> in output, got %q", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, "<strong>bold</strong>") {
		t.Errorf("expected <strong>bold</strong> in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_UnorderedList(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "- item 1\n- item 2\n- item 3"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, "<ul>") {
		t.Errorf("expected <ul> in output, got %q", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, "<li>") {
		t.Errorf("expected <li> in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_OrderedList(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "1. first\n2. second\n3. third"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, "<ol>") {
		t.Errorf("expected <ol> in output, got %q", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, "<li>") {
		t.Errorf("expected <li> in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_CodeBlock(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "```python\nprint('hello')\n```"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, "<pre") {
		t.Errorf("expected <pre> in output, got %q", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, "<code") {
		t.Errorf("expected <code> in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_InlineCode(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "Use `code` inline"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, "<code>code</code>") {
		t.Errorf("expected <code>code</code> in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_Links(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "[link text](https://example.com)"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, `<a href="https://example.com"`) {
		t.Errorf("expected link in output, got %q", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, "link text") {
		t.Errorf("expected link text in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_Images(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "![alt text](image.png)"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, `<img src="image.png"`) {
		t.Errorf("expected img tag in output, got %q", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, `alt="alt text"`) {
		t.Errorf("expected alt text in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_Blockquote(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "> This is a quote"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, "<blockquote>") {
		t.Errorf("expected <blockquote> in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_Table(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, "<table>") {
		t.Errorf("expected <table> in output, got %q", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, "<th>") {
		t.Errorf("expected <th> in output, got %q", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, "<td>") {
		t.Errorf("expected <td> in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_Strikethrough(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "~~strikethrough~~"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, "<del>strikethrough</del>") {
		t.Errorf("expected <del>strikethrough</del> in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_TaskList(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "- [ ] unchecked\n- [x] checked"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, `type="checkbox"`) {
		t.Errorf("expected checkbox input in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_Autolinks(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "Visit https://example.com for more"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, `<a href="https://example.com"`) {
		t.Errorf("expected autolinked URL in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_SkipPost(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{
		Content: "# Should not render",
		Skip:    true,
	}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if post.ArticleHTML != "" {
		t.Errorf("expected empty ArticleHTML for skipped post, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_EmptyContent(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: ""}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if post.ArticleHTML != "" {
		t.Errorf("expected empty ArticleHTML for empty content, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_RawHTML(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "<div class='custom'>content</div>"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, `<div class='custom'>content</div>`) {
		t.Errorf("expected raw HTML in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_AutoHeadingIDs(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "# My Heading"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, `id="`) {
		t.Errorf("expected heading ID in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_Render(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	m := lifecycle.NewManager()

	posts := []*models.Post{
		{Content: "# Post 1"},
		{Content: "# Post 2"},
		{Content: "# Post 3", Skip: true},
	}
	m.SetPosts(posts)

	err := p.Render(m)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	resultPosts := m.Posts()

	if !strings.Contains(resultPosts[0].ArticleHTML, "<h1") {
		t.Errorf("Post 1 not rendered correctly: %q", resultPosts[0].ArticleHTML)
	}
	if !strings.Contains(resultPosts[1].ArticleHTML, "<h1") {
		t.Errorf("Post 2 not rendered correctly: %q", resultPosts[1].ArticleHTML)
	}
	if resultPosts[2].ArticleHTML != "" {
		t.Errorf("Skipped post should have empty ArticleHTML: %q", resultPosts[2].ArticleHTML)
	}
}

// Admonition tests

func TestAdmonition_BasicNote(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "!!! note \"Important\"\n    This is important information"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	if !strings.Contains(post.ArticleHTML, `class="admonition note"`) {
		t.Errorf("expected admonition note class in output, got %q", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, `class="admonition-title"`) {
		t.Errorf("expected admonition-title class in output, got %q", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, "Important") {
		t.Errorf("expected title 'Important' in output, got %q", post.ArticleHTML)
	}
}

func TestAdmonition_Types(t *testing.T) {
	types := []string{"note", "warning", "tip", "important", "danger", "caution"}

	p := NewRenderMarkdownPlugin()

	for _, adType := range types {
		t.Run(adType, func(t *testing.T) {
			post := &models.Post{Content: "!!! " + adType + "\n    Content"}

			err := p.renderPost(post)
			if err != nil {
				t.Fatalf("renderPost error: %v", err)
			}

			expected := `class="admonition ` + adType + `"`
			if !strings.Contains(post.ArticleHTML, expected) {
				t.Errorf("expected %q in output, got %q", expected, post.ArticleHTML)
			}
		})
	}
}

func TestRenderMarkdownPlugin_AdmonitionDefaultTitle(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "!!! warning\n    Warning content"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	// Should use capitalized type as default title
	if !strings.Contains(post.ArticleHTML, "Warning") {
		t.Errorf("expected default title 'Warning' in output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_AdmonitionInvalidType(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	// Using an invalid type should not create an admonition
	post := &models.Post{Content: "!!! invalid\n    Content"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	// Should not be parsed as admonition
	if strings.Contains(post.ArticleHTML, `class="admonition`) {
		t.Errorf("invalid type should not create admonition, got %q", post.ArticleHTML)
	}
}

// Interface compliance tests

func TestRenderMarkdownPlugin_Interfaces(_ *testing.T) {
	p := NewRenderMarkdownPlugin()

	// Verify interface compliance
	var _ lifecycle.Plugin = p
	var _ lifecycle.ConfigurePlugin = p
	var _ lifecycle.RenderPlugin = p
}

// Highlight configuration tests

func TestRenderMarkdownPlugin_ConfigureWithPalette(t *testing.T) {
	tests := []struct {
		name        string
		extra       map[string]interface{}
		wantContain string // string that should be in rendered code
	}{
		{
			name: "catppuccin-mocha palette",
			extra: map[string]interface{}{
				"theme": map[string]interface{}{
					"palette": "catppuccin-mocha",
				},
			},
			wantContain: `class="chroma"`, // Uses CSS classes, not inline styles
		},
		{
			name: "explicit theme override",
			extra: map[string]interface{}{
				"theme": map[string]interface{}{
					"palette": "catppuccin-mocha",
				},
				"markdown": map[string]interface{}{
					"highlight": map[string]interface{}{
						"theme": "dracula",
					},
				},
			},
			wantContain: `class="kd"`, // keyword declaration class
		},
		{
			name:        "no config uses default",
			extra:       map[string]interface{}{},
			wantContain: `class="chroma"`, // Uses CSS classes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewRenderMarkdownPlugin()
			m := lifecycle.NewManager()
			m.Config().Extra = tt.extra

			err := p.Configure(m)
			if err != nil {
				t.Fatalf("Configure error: %v", err)
			}

			// Verify rendering works with the configured theme
			post := &models.Post{Content: "```go\nfunc main() {}\n```"}
			err = p.renderPost(post)
			if err != nil {
				t.Fatalf("renderPost error: %v", err)
			}

			if !strings.Contains(post.ArticleHTML, tt.wantContain) {
				t.Errorf("expected %q in output, got %q", tt.wantContain, post.ArticleHTML)
			}
		})
	}
}

func TestRenderMarkdownPlugin_ResolveHighlightConfig(t *testing.T) {
	tests := []struct {
		name      string
		extra     map[string]interface{}
		wantTheme string
		wantLN    bool
	}{
		{
			name:      "empty config - default dark theme",
			extra:     map[string]interface{}{},
			wantTheme: "github-dark", // Default for dark variant
			wantLN:    false,
		},
		{
			name: "explicit theme",
			extra: map[string]interface{}{
				"markdown": map[string]interface{}{
					"highlight": map[string]interface{}{
						"theme": "monokai",
					},
				},
			},
			wantTheme: "monokai",
			wantLN:    false,
		},
		{
			name: "line numbers enabled",
			extra: map[string]interface{}{
				"markdown": map[string]interface{}{
					"highlight": map[string]interface{}{
						"theme":        "dracula",
						"line_numbers": true,
					},
				},
			},
			wantTheme: "dracula",
			wantLN:    true,
		},
		{
			name: "palette-derived theme - catppuccin-mocha",
			extra: map[string]interface{}{
				"theme": map[string]interface{}{
					"palette": "catppuccin-mocha",
				},
			},
			wantTheme: "catppuccin-mocha",
			wantLN:    false,
		},
		{
			name: "palette-derived theme - gruvbox-light",
			extra: map[string]interface{}{
				"theme": map[string]interface{}{
					"palette": "gruvbox-light",
				},
			},
			wantTheme: "gruvbox-light",
			wantLN:    false,
		},
		{
			name: "explicit theme overrides palette",
			extra: map[string]interface{}{
				"theme": map[string]interface{}{
					"palette": "catppuccin-mocha",
				},
				"markdown": map[string]interface{}{
					"highlight": map[string]interface{}{
						"theme": "nord",
					},
				},
			},
			wantTheme: "nord",
			wantLN:    false,
		},
		{
			name: "unknown palette uses variant default - light",
			extra: map[string]interface{}{
				"theme": map[string]interface{}{
					"palette": "custom-light-theme",
				},
			},
			wantTheme: "github", // Light variant default
			wantLN:    false,
		},
		{
			name: "unknown palette uses variant default - dark",
			extra: map[string]interface{}{
				"theme": map[string]interface{}{
					"palette": "custom-dark-theme",
				},
			},
			wantTheme: "github-dark", // Dark variant default
			wantLN:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewRenderMarkdownPlugin()
			gotTheme, gotLN := p.resolveHighlightConfig(tt.extra)

			if gotTheme != tt.wantTheme {
				t.Errorf("theme = %q, want %q", gotTheme, tt.wantTheme)
			}
			if gotLN != tt.wantLN {
				t.Errorf("lineNumbers = %v, want %v", gotLN, tt.wantLN)
			}
		})
	}
}

func TestRenderMarkdownPlugin_AttributeSyntax(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "image with class",
			input:    "![alt text](image.webp){.more-cinematic}",
			expected: `class="more-cinematic"`,
		},
		{
			name:     "image with multiple classes",
			input:    "![photo](photo.jpg){.shadow .bordered}",
			expected: `class="shadow bordered"`,
		},
		{
			name:     "image with id",
			input:    "![hero](hero.png){#hero-image}",
			expected: `id="hero-image"`,
		},
		{
			name:     "image with class and id",
			input:    "![banner](banner.jpg){#main-banner .full-width}",
			expected: `id="main-banner"`,
		},
		{
			name:     "heading with class",
			input:    "## Section Title {.highlighted}",
			expected: `class="highlighted"`,
		},
		{
			name:     "heading with custom id",
			input:    "## Installation {#install}",
			expected: `id="install"`,
		},
		{
			name:     "link with class",
			input:    "[Click here](https://example.com){.external}",
			expected: `class="external"`,
		},
		{
			name:     "link with id",
			input:    "[Main link](https://example.com){#main-link}",
			expected: `id="main-link"`,
		},
	}

	p := NewRenderMarkdownPlugin()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := &models.Post{Content: tt.input}
			err := p.renderPost(post)
			if err != nil {
				t.Fatalf("renderPost error: %v", err)
			}
			if !strings.Contains(post.ArticleHTML, tt.expected) {
				t.Errorf("expected %q in output, got %q", tt.expected, post.ArticleHTML)
			}
		})
	}
}

func TestRenderMarkdownPlugin_AttributeSyntax_ImageClassInOutput(t *testing.T) {
	// Specific test for the issue #404 use case
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "![alt text](image.webp){.more-cinematic}"}

	err := p.renderPost(post)
	if err != nil {
		t.Fatalf("renderPost error: %v", err)
	}

	// Verify the img tag has the class attribute
	if !strings.Contains(post.ArticleHTML, `<img`) {
		t.Errorf("expected <img> tag in output, got %q", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, `class="more-cinematic"`) {
		t.Errorf("expected class=\"more-cinematic\" in output, got %q", post.ArticleHTML)
	}
	// Verify the attribute syntax {.more-cinematic} is NOT in the output
	if strings.Contains(post.ArticleHTML, "{.more-cinematic}") {
		t.Errorf("attribute syntax should be removed from output, got %q", post.ArticleHTML)
	}
}

func TestRenderMarkdownPlugin_GetPaletteVariant(t *testing.T) {
	p := NewRenderMarkdownPlugin()

	tests := []struct {
		palette string
		want    string
	}{
		// Light variants
		{"default-light", "light"},
		{"gruvbox-light", "light"},
		{"catppuccin-latte", "light"},
		{"rose-pine-dawn", "light"},
		{"tokyo-night-day", "light"},
		{"kanagawa-lotus", "light"},

		// Dark variants
		{"default-dark", "dark"},
		{"gruvbox-dark", "dark"},
		{"catppuccin-mocha", "dark"},
		{"rose-pine", "dark"},
		{"tokyo-night", "dark"},
		{"dracula", "dark"},
		{"matte-black", "dark"},
	}

	for _, tt := range tests {
		t.Run(tt.palette, func(t *testing.T) {
			got := p.getPaletteVariant(tt.palette)
			if string(got) != tt.want {
				t.Errorf("getPaletteVariant(%q) = %q, want %q", tt.palette, got, tt.want)
			}
		})
	}
}

func TestRenderMarkdownPlugin_DetectCSSRequirements(t *testing.T) {
	tests := []struct {
		name               string
		content            string
		wantAdmonitionsCSS bool
		wantCodeCSS        bool
	}{
		{
			name:               "plain text - no CSS needed",
			content:            "Hello world, this is plain text.",
			wantAdmonitionsCSS: false,
			wantCodeCSS:        false,
		},
		{
			name:               "fenced code block - needs code CSS",
			content:            "```go\nfunc main() {}\n```",
			wantAdmonitionsCSS: false,
			wantCodeCSS:        true,
		},
		{
			name:               "inline code - no special code CSS needed",
			content:            "Use `fmt.Println()` to print.",
			wantAdmonitionsCSS: false,
			wantCodeCSS:        false, // inline code doesn't need syntax highlighting CSS
		},
		{
			name:               "headings and links - no special CSS",
			content:            "# Heading\n\n[Link](https://example.com)",
			wantAdmonitionsCSS: false,
			wantCodeCSS:        false,
		},
		{
			name:               "code block without language - still needs code CSS",
			content:            "```\nplain code block\n```",
			wantAdmonitionsCSS: false,
			wantCodeCSS:        true,
		},
		{
			name:               "admonition - needs admonitions CSS (and code CSS due to indented content)",
			content:            "!!! note \"Note\"\n    This is a note admonition.",
			wantAdmonitionsCSS: true,
			wantCodeCSS:        true, // Current behavior: 4-space indented content becomes code block
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewRenderMarkdownPlugin()
			post := &models.Post{Content: tt.content}

			err := p.renderPost(post)
			if err != nil {
				t.Fatalf("renderPost error: %v", err)
			}

			// Check admonitions CSS flag
			gotAdmonitions := false
			if post.Extra != nil {
				if v, ok := post.Extra["needs_admonitions_css"].(bool); ok {
					gotAdmonitions = v
				}
			}
			if gotAdmonitions != tt.wantAdmonitionsCSS {
				t.Errorf("needs_admonitions_css = %v, want %v", gotAdmonitions, tt.wantAdmonitionsCSS)
			}

			// Check code CSS flag
			gotCode := false
			if post.Extra != nil {
				if v, ok := post.Extra["needs_code_css"].(bool); ok {
					gotCode = v
				}
			}
			if gotCode != tt.wantCodeCSS {
				t.Errorf("needs_code_css = %v, want %v", gotCode, tt.wantCodeCSS)
			}
		})
	}
}
