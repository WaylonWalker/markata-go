package plugins

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func renderContainerMarkdown(t *testing.T, input string) string {
	t.Helper()
	md := goldmark.New(goldmark.WithExtensions(&ContainerExtension{}))
	var buf bytes.Buffer
	if err := md.Convert([]byte(input), &buf); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	return buf.String()
}

func TestContainerExtension_NestedShallowerClosersStayInsideOuter(t *testing.T) {
	input := `::::: cards
:::: card
### Card one
Content of card one.
::::
:::: card
### Card two
Content of card two.
::::
:::::

After.`

	output := renderContainerMarkdown(t, input)

	if strings.Contains(output, ":::") {
		t.Fatalf("closer leaked into output: %s", output)
	}
	if strings.Count(output, `class="card"`) != 2 {
		t.Fatalf("expected two card containers: %s", output)
	}
	cardsIdx := strings.Index(output, `class="cards"`)
	afterIdx := strings.Index(output, "<p>After.</p>")
	lastCard := strings.LastIndex(output, "Content of card two.")
	if cardsIdx < 0 || afterIdx < 0 || lastCard < 0 || lastCard > afterIdx {
		t.Fatalf("cards did not nest inside outer container: %s", output)
	}
}

func TestContainerExtension_InnerCloserAfterCodeBlock(t *testing.T) {
	input := ":::: outer\n::: inner\n```python\nimport this\n```\n:::\n\nsecond inner block\n::::\n"

	output := renderContainerMarkdown(t, input)

	if strings.Contains(output, ":::") {
		t.Fatalf("closer leaked into output: %s", output)
	}
	if !strings.Contains(output, `<div class="outer">`) || !strings.Contains(output, `<div class="inner">`) {
		t.Fatalf("missing containers: %s", output)
	}
	if !strings.Contains(output, "<p>second inner block</p>") {
		t.Fatalf("trailing paragraph in outer container was mangled: %s", output)
	}
}

func TestContainerExtension_DeeperCloserDoesNotCloseOuter(t *testing.T) {
	input := "::: outer\n:::: inner\nnested\n::::\nstill outer\n:::\n"

	output := renderContainerMarkdown(t, input)

	if strings.Contains(output, ":::") {
		t.Fatalf("closer leaked into output: %s", output)
	}
	if !strings.Contains(output, "<p>still outer</p>") {
		t.Fatalf("outer closed early: %s", output)
	}
}
