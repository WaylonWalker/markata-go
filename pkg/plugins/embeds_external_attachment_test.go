package plugins

import (
	"strings"
	"testing"
)

func TestProcessAttachmentEmbedsLeavesExternalURLsForExternalParser(t *testing.T) {
	t.Parallel()

	p := NewEmbedsPlugin()
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "http html path",
			input: `![[http://example.com/article.html]]`,
		},
		{
			name:  "https html path",
			input: `![[https://halfwit.github.io/2017/05/08/keyboardblog.html]]`,
		},
		{
			name:  "query ending in dotted host",
			input: `![[https://ted-merz.com/2022/09/14/20-percent-time/?utm_source=chatgpt.com]]`,
		},
		{
			name:  "display title",
			input: `![[https://cleberg.net/blog/internet.html|Internet notes]]`,
		},
		{
			name:  "fragment",
			input: `![[https://example.com/article.html#section]]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := p.processAttachmentEmbeds(tt.input)
			if got != tt.input {
				t.Fatalf("processAttachmentEmbeds() = %q, want original external embed %q", got, tt.input)
			}
			if strings.Contains(got, p.config.AttachmentsPrefix+"http") {
				t.Fatalf("external URL was rewritten as a local attachment: %q", got)
			}
		})
	}
}

func TestProcessAttachmentEmbedsStillConvertsLocalAttachments(t *testing.T) {
	t.Parallel()

	p := NewEmbedsPlugin()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "image",
			in:   `![[photo.jpg]]`,
			want: `![photo.jpg](/static/photo.jpg)`,
		},
		{
			name: "nested pdf with title",
			in:   `![[docs/manual.pdf|Manual]]`,
			want: `![Manual](/static/docs/manual.pdf)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := p.processAttachmentEmbeds(tt.in); got != tt.want {
				t.Fatalf("processAttachmentEmbeds() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProcessAttachmentEmbedsPreservesFencedCode(t *testing.T) {
	t.Parallel()

	p := NewEmbedsPlugin()
	input := "```markdown\n![[photo.jpg]]\n![[https://example.com/article.html]]\n```"
	if got := p.processAttachmentEmbeds(input); got != input {
		t.Fatalf("processAttachmentEmbeds() changed fenced code: %q", got)
	}
}

func TestEmbedsCacheVersionInvalidatesExternalAttachmentFix(t *testing.T) {
	t.Parallel()

	if embedsCacheVersion != "v4" {
		t.Fatalf("embedsCacheVersion = %q, want v4 for the external attachment fix", embedsCacheVersion)
	}
}
