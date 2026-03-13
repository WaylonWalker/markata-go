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
		"=====",
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

func TestStripANSI_RemovesEscapeSequences(t *testing.T) {
	input := "\x1b[1mhello\x1b[0m \x1b[38;2;255;0;0mworld\x1b[0m"
	if got := StripANSI(input); got != "hello world" {
		t.Fatalf("StripANSI() = %q, want %q", got, "hello world")
	}
}
