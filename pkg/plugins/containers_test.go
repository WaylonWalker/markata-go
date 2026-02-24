package plugins

import (
	"strings"
	"testing"
)

func renderMarkdownForTest(t *testing.T, content string) string {
	t.Helper()
	plugin := NewRenderMarkdownPlugin()

	html, err := plugin.doRender(content)
	if err != nil {
		t.Fatalf("rendering markdown: %v", err)
	}

	return html
}

func TestNestedContainersMaintainStructure(t *testing.T) {
	content := `:::card
Outer header

:::card
Inner details
:::

After inner close
:::`
	html := renderMarkdownForTest(t, content)
	outerStart := strings.Index(html, `<div class="card">`)
	if outerStart == -1 {
		t.Fatalf("outer card not rendered: %s", html)
	}
	innerStart := strings.Index(html[outerStart+len(`<div class="card">`):], `<div class="card">`)
	if innerStart == -1 {
		t.Fatalf("inner card not rendered inside outer card: %s", html)
	}
	innerStart += outerStart + len(`<div class="card">`)
	innerCloseRel := strings.Index(html[innerStart:], "</div>")
	if innerCloseRel == -1 {
		t.Fatalf("inner card missing closing tag: %s", html)
	}
	innerClose := innerStart + innerCloseRel
	innerContentAfter := strings.Index(html[innerClose:], "After inner close")
	if innerContentAfter == -1 {
		t.Fatalf("outer content missing after inner close: %s", html)
	}
	outerClose := strings.LastIndex(html, "</div>")
	if outerClose == -1 || outerClose <= innerClose {
		t.Fatalf("outer card closing tag is misplaced: %s", html)
	}
	if outerClose <= outerStart {
		t.Fatalf("outer closing tag appears before outer start: %s", html)
	}
	if innerClose >= outerClose {
		t.Fatalf("inner closing tag should be inside outer card: %s", html)
	}
}

func TestContainerWithHeaderAndNestedContent(t *testing.T) {
	content := `:::card {#summary .highlight}
### Summary

Leading paragraph before nested card.

:::card {#popover}
Nested info block.
:::

Trailing paragraph after nested card.

:::`
	html := renderMarkdownForTest(t, content)
	if !strings.Contains(html, `<h3`) || !strings.Contains(html, `Summary`) {
		t.Fatalf("header not preserved inside card: %s", html)
	}
	nestedIdx := strings.Index(html, `id="popover"`)
	if nestedIdx == -1 {
		t.Fatalf("nested card id attribute not rendered: %s", html)
	}
	trailingIdx := strings.Index(html[nestedIdx:], "Trailing paragraph after nested card")
	if trailingIdx == -1 {
		t.Fatalf("content after nested card missing: %s", html)
	}
	outerClose := strings.LastIndex(html, "</div>")
	if outerClose == -1 {
		t.Fatalf("outer card never closes: %s", html)
	}
	if trailingIdx >= outerClose {
		t.Fatalf("trailing content should appear before outer card closes: %s", html)
	}
}

func TestNestedContainersWithConsecutiveOpenersAndTable(t *testing.T) {
	content := `::: card
::: header
## Wicket{.center}

100 hp

:::

![A pickture of wicket, the wise old bird](/wicket.webp)
figcaption

| hp | 5 |
| --- | --- |
| ac | 15 |
| speed | 30 |

:::`

	html := renderMarkdownForTest(t, content)
	outerStart := strings.Index(html, `<div class="card">`)
	if outerStart == -1 {
		t.Fatalf("outer card not rendered: %s", html)
	}
	if !strings.Contains(html, `<div class="header">`) {
		t.Fatalf("inner header container not rendered: %s", html)
	}
	figureIdx := strings.Index(html, "<figure>")
	if figureIdx == -1 {
		t.Fatalf("figure should be inside container: %s", html)
	}
	outerClose := strings.LastIndex(html, "</div>")
	if outerClose == -1 {
		t.Fatalf("outer container did not close: %s", html)
	}
	if figureIdx > outerClose {
		t.Fatalf("outer card closed before figure/table content: %s", html)
	}
	if strings.Contains(html, "<p>:::</p>") {
		t.Fatalf("closing marker leaked into output: %s", html)
	}
}
