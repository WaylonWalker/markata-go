package terminalpage

import (
	"strings"
	"testing"
)

func TestRenderHTML_PlainPreservesStructure(t *testing.T) {
	input := `<h1>Title</h1><p>Visit <a href="https://go.dev">Go</a>.</p><blockquote><p>Quoted text</p></blockquote><table><thead><tr><th>Name</th><th>Value</th></tr></thead><tbody><tr><td>alpha</td><td>1</td></tr></tbody></table>`

	got := RenderHTML(input, Options{})

	checks := []string{
		"Title",
		"━━━━━",
		"Go <https://go.dev>",
		"│ Quoted text",
		"| Name",
		"| alpha",
	}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("plain output should not contain ANSI escapes: %q", got)
	}
}

func TestRenderHTML_StripsHeadingAnchorLinks(t *testing.T) {
	input := `<h2 id="intro">Intro <a href="#intro" class="heading-anchor">#</a></h2><p>Body</p>`

	got := RenderHTML(input, Options{})

	if strings.Contains(got, "# <#intro>") || strings.Contains(got, `href="#intro"`) {
		t.Fatalf("expected heading anchor link removed, got:\n%s", got)
	}
	if !strings.Contains(got, "Intro") {
		t.Fatalf("expected heading text preserved, got:\n%s", got)
	}
}

func TestRenderHTML_PlainCodeBlocksUseFences(t *testing.T) {
	input := `<pre><code class="language-python">from datetime import datetime

def test_copy():
    now = datetime.now()
    print(now)
</code></pre>`

	got := RenderHTML(input, Options{})

	if !strings.Contains(got, "```python") {
		t.Fatalf("expected fenced code block with language, got:\n%s", got)
	}
	if !strings.Contains(got, "def test_copy():") {
		t.Fatalf("expected code content preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "\n    now = datetime.now()\n") {
		t.Fatalf("expected code newlines preserved, got:\n%s", got)
	}
}

func TestRenderHTML_ANSIHighlightsCodeAndAdmonitions(t *testing.T) {
	input := `<div class="admonition warning"><p class="admonition-title">Careful</p><p>Read this first.</p></div><pre><code class="language-go">fmt.Println("hi")</code></pre>`

	got := RenderHTML(input, Options{ANSI: true, Palette: "default-dark", ChromaStyle: "github-dark"})

	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("ANSI output should contain escape sequences: %q", got)
	}
	if !strings.Contains(StripANSI(got), "WARNING Careful") {
		t.Fatalf("expected admonition heading in output, got:\n%s", StripANSI(got))
	}
	if !strings.Contains(StripANSI(got), "fmt.Println(\"hi\")") {
		t.Fatalf("expected code block in output, got:\n%s", StripANSI(got))
	}
}

func TestRenderHTML_PreservesMediaLinks(t *testing.T) {
	input := `<p><img src="https://example.com/diagram.png" alt="Diagram"></p><p><video controls><source src="https://example.com/demo.mp4" type="video/mp4"></video></p>`

	plain := RenderHTML(input, Options{})
	ansi := StripANSI(RenderHTML(input, Options{ANSI: true, Palette: "default-dark", ChromaStyle: "github-dark"}))

	for _, got := range []string{plain, ansi} {
		if !strings.Contains(got, "Image: Diagram <https://example.com/diagram.png>") {
			t.Fatalf("expected image link in output, got:\n%s", got)
		}
		if !strings.Contains(got, "Video: <https://example.com/demo.mp4>") {
			t.Fatalf("expected video link in output, got:\n%s", got)
		}
	}
}

func TestStripANSI_RemovesEscapeSequences(t *testing.T) {
	input := "\x1b[1mhello\x1b[0m \x1b[38;2;255;0;0mworld\x1b[0m"
	if got := StripANSI(input); got != "hello world" {
		t.Fatalf("StripANSI() = %q, want %q", got, "hello world")
	}
}
