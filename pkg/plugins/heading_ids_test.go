package plugins

import (
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestHeadingIDTransformer_LinkDestinationDoesNotLeakIntoID(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	post := &models.Post{Content: "## Recent [TIL](/til/)\n\n## Plain `code` heading\n\n## Custom {#custom-id}\n\n## Recent TIL\n"}
	if err := p.renderPost(post); err != nil {
		t.Fatalf("renderPost: %v", err)
	}
	for _, want := range []string{`id="recent-til"`, `id="plain-code-heading"`, `id="custom-id"`, `id="recent-til-1"`} {
		if !strings.Contains(post.ArticleHTML, want) {
			t.Errorf("expected %s in %s", want, post.ArticleHTML)
		}
	}
	if strings.Contains(post.ArticleHTML, "tiltil") {
		t.Errorf("link destination leaked into heading id: %s", post.ArticleHTML)
	}
}
