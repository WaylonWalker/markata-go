package htmltotext

import (
	"strings"
	"testing"
)

func TestConvert_Empty(t *testing.T) {
	if got := Convert(""); got != "" {
		t.Errorf("Convert(\"\") = %q, want \"\"", got)
	}
}

func TestConvert_PlainText(t *testing.T) {
	input := "Hello world"
	want := "Hello world"
	if got := Convert(input); got != want {
		t.Errorf("Convert(%q) = %q, want %q", input, got, want)
	}
}

func TestConvert_HTMLEntities(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"ampersand", "Tom &amp; Jerry", "Tom & Jerry"},
		{"less than", "1 &lt; 2", "1 < 2"},
		{"greater than", "2 &gt; 1", "2 > 1"},
		{"quote", "&quot;hello&quot;", `"hello"`},
		{"apostrophe", "it&#39;s", "it's"},
		{"apos entity", "it&apos;s", "it's"},
		{"nbsp", "hello&nbsp;world", "hello\u00a0world"},
		{"numeric entity", "&#169; 2024", "\u00a9 2024"},
		{"multiple entities", "&lt;div&gt; &amp; &quot;test&quot;", `<div> & "test"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Convert(tt.input)
			if got != tt.want {
				t.Errorf("Convert(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvert_StripTags(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"bold", "<b>bold</b>", "bold"},
		{"italic", "<em>italic</em>", "italic"},
		{"span", "<span class='x'>text</span>", "text"},
		{"nested", "<div><p><b>deep</b></p></div>", "deep"},
		{"self closing", "line<br/>break", "line\nbreak"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Convert(tt.input)
			if got != tt.want {
				t.Errorf("Convert(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvert_BlockStructure(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"paragraphs",
			"<p>First paragraph.</p><p>Second paragraph.</p>",
			"First paragraph.\n\nSecond paragraph.",
		},
		{
			"headings",
			"<h1>Title</h1><p>Body text.</p>",
			"Title\n\nBody text.",
		},
		{
			"br tags",
			"Line one<br>Line two<br/>Line three",
			"Line one\nLine two\nLine three",
		},
		{
			"hr tags",
			"<p>Above</p><hr><p>Below</p>",
			"Above\n\n---\n\nBelow",
		},
		{
			"unordered list",
			"<ul><li>First</li><li>Second</li><li>Third</li></ul>",
			"- First\n\n- Second\n\n- Third",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Convert(tt.input)
			if got != tt.want {
				t.Errorf("Convert(%q) =\n%q\nwant:\n%q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvert_Links_Footnotes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"single link",
			`<a href="https://go.dev">Go</a>`,
			"Go [1]\n\nReferences:\n[1]: https://go.dev",
		},
		{
			"link in paragraph",
			`<p>Visit <a href="https://go.dev">Go</a> for more.</p>`,
			"Visit Go [1] for more.\n\nReferences:\n[1]: https://go.dev",
		},
		{
			"multiple different links",
			`<a href="https://go.dev">Go</a> and <a href="https://rust-lang.org">Rust</a>`,
			"Go [1] and Rust [2]\n\nReferences:\n[1]: https://go.dev\n[2]: https://rust-lang.org",
		},
		{
			"duplicate URLs reuse reference",
			`<a href="https://go.dev">Go</a> and <a href="https://go.dev">Go language</a>`,
			"Go [1] and Go language [1]\n\nReferences:\n[1]: https://go.dev",
		},
		{
			"bare link - text matches URL",
			`<a href="https://go.dev">https://go.dev</a>`,
			"https://go.dev",
		},
		{
			"bare link with trailing slash mismatch",
			`<a href="https://go.dev/">https://go.dev</a>`,
			"https://go.dev",
		},
		{
			"mixed bare and named links",
			`Visit <a href="https://go.dev">https://go.dev</a> or <a href="https://go.dev/doc">docs</a>`,
			"Visit https://go.dev or docs [1]\n\nReferences:\n[1]: https://go.dev/doc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Convert(tt.input)
			if got != tt.want {
				t.Errorf("Convert(%q) =\n%s\nwant:\n%s", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvert_LinksWithNestedTags(t *testing.T) {
	input := `<a href="https://go.dev"><strong>Go</strong></a>`
	want := "Go [1]\n\nReferences:\n[1]: https://go.dev"
	got := Convert(input)
	if got != want {
		t.Errorf("Convert(%q) =\n%s\nwant:\n%s", input, got, want)
	}
}

func TestConvert_EntitiesInLinkText(t *testing.T) {
	input := `<a href="https://example.com">Tom &amp; Jerry</a>`
	want := "Tom & Jerry [1]\n\nReferences:\n[1]: https://example.com"
	got := Convert(input)
	if got != want {
		t.Errorf("Convert(%q) =\n%s\nwant:\n%s", input, got, want)
	}
}

func TestConvert_ComplexDocument(t *testing.T) {
	input := `<h1>My Post</h1>
<p>This is about <a href="https://go.dev">Go</a> and &amp; more.</p>
<p>See the <a href="https://go.dev/doc">documentation</a> for details.</p>
<ul>
<li>Item one</li>
<li>Item two with <a href="https://go.dev">Go</a> link</li>
</ul>
<p>Copyright &copy; 2024</p>`

	got := Convert(input)

	// Verify key properties
	if strings.Contains(got, "<") && !strings.Contains(got, "< ") {
		// Allow literal < from entity decoding like "1 < 2" but not HTML tags
		for _, line := range strings.Split(got, "\n") {
			if strings.Contains(line, "<") && strings.Contains(line, ">") {
				t.Errorf("Output contains potential HTML tags: %q", line)
			}
		}
	}
	if strings.Contains(got, "&amp;") {
		t.Error("Output contains &amp; entity")
	}
	if strings.Contains(got, "&lt;") {
		t.Error("Output contains &lt; entity")
	}
	if !strings.Contains(got, "Go [1]") {
		t.Error("Expected footnote reference [1] for Go link")
	}
	if !strings.Contains(got, "documentation [2]") {
		t.Error("Expected footnote reference [2] for documentation link")
	}
	if !strings.Contains(got, "References:") {
		t.Error("Expected References section")
	}
	if !strings.Contains(got, "[1]: https://go.dev") {
		t.Error("Expected reference [1] for https://go.dev")
	}
	if !strings.Contains(got, "[2]: https://go.dev/doc") {
		t.Error("Expected reference [2] for https://go.dev/doc")
	}
	// The duplicate Go link in the list should reuse [1]
	lines := strings.Split(got, "\n")
	refCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "[") && strings.Contains(line, "]:") {
			refCount++
		}
	}
	if refCount != 2 {
		t.Errorf("Expected 2 unique references, got %d", refCount)
	}
}

func TestConvert_WhitespaceNormalization(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"collapses multiple spaces",
			"hello    world",
			"hello world",
		},
		{
			"trims trailing whitespace on lines",
			"<p>hello   </p><p>world</p>",
			"hello\n\nworld",
		},
		{
			"no more than 2 consecutive newlines",
			"<p>one</p>\n\n\n<p>two</p>",
			"one\n\ntwo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Convert(tt.input)
			if got != tt.want {
				t.Errorf("Convert(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvert_NoHTMLEntitiesInOutput(t *testing.T) {
	// This is the core requirement: no HTML entities should appear in text output
	entities := []string{"&amp;", "&lt;", "&gt;", "&quot;", "&#39;", "&apos;"}
	inputs := []string{
		`<p>Tom &amp; Jerry &lt;3</p>`,
		`<p>She said &quot;hello&quot; &amp; &apos;goodbye&apos;</p>`,
		`<a href="https://example.com">Link &amp; Text</a>`,
		`<h1>Title &lt;with&gt; entities</h1>`,
	}

	for _, input := range inputs {
		got := Convert(input)
		for _, entity := range entities {
			if strings.Contains(got, entity) {
				t.Errorf("Convert(%q) output contains HTML entity %q:\n%s", input, entity, got)
			}
		}
	}
}

func TestConvert_NoRawHTMLTags(t *testing.T) {
	inputs := []string{
		`<p>paragraph</p>`,
		`<div class="wrapper"><span>text</span></div>`,
		`<a href="https://go.dev">Go</a>`,
		`<img src="photo.jpg" alt="photo">`,
		`<script>alert('xss')</script>`,
	}

	tagPattern := strings.NewReader("")
	_ = tagPattern

	for _, input := range inputs {
		got := Convert(input)
		// Check for any HTML tag patterns (< followed by a letter or /)
		for i := 0; i < len(got)-1; i++ {
			if got[i] == '<' {
				next := got[i+1]
				if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || next == '/' {
					t.Errorf("Convert(%q) output contains HTML tag at position %d: ...%s...",
						input, i, got[max(0, i):min(len(got), i+20)])
				}
			}
		}
	}
}
