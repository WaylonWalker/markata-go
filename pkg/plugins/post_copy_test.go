package plugins

import (
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestBuildPostCopyPayloads(t *testing.T) {
	post := &models.Post{
		Slug:        "hello-world",
		Href:        "/hello-world/",
		Title:       stringPtr("Hello World"),
		Description: stringPtr("A friendly introduction"),
		Content:     "## Intro\n\n```python\nprint('hello')\n```\n",
		ArticleHTML: "<h2 id=\"intro\">Intro <a href=\"#intro\" class=\"heading-anchor\">#</a></h2><pre><code class=\"language-python\">print('hello')\n</code></pre>",
	}

	payloads := buildPostCopyPayloads(post, nil, "https://example.com")
	if payloads.URL != "https://example.com/hello-world/" {
		t.Fatalf("unexpected url payload: %q", payloads.URL)
	}
	if !strings.Contains(payloads.Markdown, "# Hello World") {
		t.Fatalf("expected markdown payload to contain title, got %q", payloads.Markdown)
	}
	if !strings.Contains(payloads.Markdown, "Source: https://example.com/hello-world/") {
		t.Fatalf("expected markdown payload to contain source url, got %q", payloads.Markdown)
	}
	if !strings.Contains(payloads.Markdown, "```python") {
		t.Fatalf("expected markdown payload to preserve code fences, got %q", payloads.Markdown)
	}
	if !strings.Contains(payloads.Text, "Source: https://example.com/hello-world/") {
		t.Fatalf("expected text payload to contain source url, got %q", payloads.Text)
	}
	if strings.Contains(payloads.Text, "# <#intro>") {
		t.Fatalf("expected text payload to omit heading anchors, got %q", payloads.Text)
	}
	if !strings.Contains(payloads.Text, "```python") {
		t.Fatalf("expected text payload to use fenced code blocks, got %q", payloads.Text)
	}
}

func stringPtr(value string) *string {
	return &value
}
