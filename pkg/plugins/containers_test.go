// Package plugins provides lifecycle plugins for markata-go.
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
	var output bytes.Buffer
	if err := md.Convert([]byte(input), &output); err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	return output.String()
}

func assertContainerHTMLInOrder(t *testing.T, got string, wantInOrder ...string) {
	t.Helper()

	cursor := 0
	for _, want := range wantInOrder {
		idx := strings.Index(got[cursor:], want)
		if idx < 0 {
			t.Fatalf("rendered HTML missing %q after byte %d\nGot:\n%s", want, cursor, got)
		}
		cursor += idx + len(want)
	}
}

func TestContainerExtension_NestedContainersUseMatchingDelimiterDepth(t *testing.T) {
	got := renderContainerMarkdown(t, `::: outer
outer before

:::: inner
inner body
::::

outer after
:::`)

	assertContainerHTMLInOrder(t, got,
		`<div class="outer">`,
		`<p>outer before</p>`,
		`<div class="inner">`,
		`<p>inner body</p>`,
		`</div>`,
		`<p>outer after</p>`,
		`</div>`,
	)
}

func TestContainerExtension_CloseMarkerAllowsSurroundingWhitespace(t *testing.T) {
	got := renderContainerMarkdown(t, "::: outer\nbody\n   :::   \nafter")

	assertContainerHTMLInOrder(t, got,
		`<div class="outer">`,
		`<p>body</p>`,
		`</div>`,
		`<p>after</p>`,
	)
	if strings.Contains(got, ":::") {
		t.Fatalf("closing marker leaked into rendered HTML:\n%s", got)
	}
}

func TestContainerCloseMarkerMustBeColonOnly(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{name: "too short", line: "::", want: false},
		{name: "trailing whitespace is normalized by parser", line: "::: ", want: false},
		{name: "three colons exact", line: ":::", want: true},
		{name: "four colons exact", line: "::::", want: true},
		{name: "named container", line: "::: card", want: false},
		{name: "attribute container", line: `::: {.card}`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isContainerClose(tt.line); got != tt.want {
				t.Fatalf("isContainerClose(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}
