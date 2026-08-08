package plugins

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

func TestRenderInline_ProducesRichAndPlainRepresentations(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	result, err := p.renderInline("Good themes make **good places** to ==think.==")
	if err != nil {
		t.Fatal(err)
	}
	if result.HTML != `Good themes make <strong>good places</strong> to <mark>think.</mark>` {
		t.Fatalf("HTML = %q", result.HTML)
	}
	if result.Text != "Good themes make good places to think." {
		t.Fatalf("Text = %q", result.Text)
	}
}

func TestRenderInline_ParsesSuperscriptAndSubscript(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	result, err := p.renderInline("H~2~O and x^2^")
	if err != nil {
		t.Fatal(err)
	}
	if result.HTML != "H<sub>2</sub>O and x<sup>2</sup>" {
		t.Fatalf("HTML = %q", result.HTML)
	}
	if result.Text != "H2O and x2" {
		t.Fatalf("Text = %q", result.Text)
	}
}

func TestRenderInline_SemanticsAndSafety(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	result, err := p.renderInline("<script>alert(1)</script> _quiet_ ==**loud**== [bad](javascript:alert(1)) ~~old~~ `literal ==mark==`")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<em>quiet</em>", "<mark><strong>loud</strong></mark>", "<del>old</del>"} {
		if !strings.Contains(result.HTML, want) {
			t.Errorf("HTML %q does not contain %q", result.HTML, want)
		}
	}
	if strings.Contains(result.HTML, "<script>") || strings.Contains(result.HTML, "javascript:") {
		t.Fatalf("unsafe HTML or link leaked: %q", result.HTML)
	}
	if strings.Contains(result.Text, "**") || strings.Contains(result.Text, "<script>") {
		t.Fatalf("syntax leaked into text: %q", result.Text)
	}
}

func TestMarkParser_BuildsNestedInlineAST(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	source := []byte("==**strong**== ==_emphasis_== ==`code`== ==[link](/test)== ==~~deleted~~==")
	document := p.md.Parser().Parse(text.NewReader(source))
	paragraph := document.FirstChild()
	if paragraph == nil {
		t.Fatal("expected paragraph")
	}
	wantKinds := []ast.NodeKind{KindMark, ast.KindEmphasis, KindMark, ast.KindEmphasis, KindMark, ast.KindCodeSpan, KindMark, ast.KindLink, KindMark, extast.KindStrikethrough}
	var kinds []ast.NodeKind
	if err := ast.Walk(paragraph, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && node.Kind() != ast.KindParagraph && node.Kind() != ast.KindText {
			kinds = append(kinds, node.Kind())
		}
		return ast.WalkContinue, nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(kinds) != len(wantKinds) {
		t.Fatalf("node kinds = %v, want %v", kinds, wantKinds)
	}
	for i := range wantKinds {
		if kinds[i] != wantKinds[i] {
			t.Fatalf("node kinds = %v, want %v", kinds, wantKinds)
		}
	}
	var emphasisLevels []int
	if err := ast.Walk(paragraph, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if emphasis, ok := node.(*ast.Emphasis); ok {
				emphasisLevels = append(emphasisLevels, emphasis.Level)
			}
		}
		return ast.WalkContinue, nil
	}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(emphasisLevels, []int{2, 1}) {
		t.Fatalf("emphasis levels = %v", emphasisLevels)
	}
}

func TestModelsPost_PlainTitleAndSlugUseSemanticTitle(t *testing.T) {
	title := "Good themes make **good places** to ==think.=="
	post := &models.Post{Path: "posts/.md", Title: &title, TitleText: "Good themes make good places to think."}
	post.GenerateSlug()
	if post.Slug != "good-themes-make-good-places-to-think" {
		t.Fatalf("slug = %q", post.Slug)
	}
}

func TestInlineTitlesPlugin_PopulatesExplicitRepresentations(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	m := lifecycle.NewManager()
	if err := p.Configure(m); err != nil {
		t.Fatal(err)
	}
	title := "Good themes make **good places** to ==think.=="
	post := models.NewPost("post.md")
	post.Title = &title
	m.AddPost(post)
	if err := NewInlineTitlesPlugin().Transform(m); err != nil {
		t.Fatal(err)
	}
	if post.TitleHTML != `Good themes make <strong>good places</strong> to <span class="heading-highlight"><mark>think.</mark></span>` {
		t.Fatalf("TitleHTML = %q", post.TitleHTML)
	}
	if post.TitleText != "Good themes make good places to think." {
		t.Fatalf("TitleText = %q", post.TitleText)
	}
}

func TestMinimalPlugins_DerivesTitleAndTitleFallbackSlugThroughLifecycle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".md")
	source := "---\ntitle: Good **places** to ==think==\n---\nBody\n"
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}

	m := lifecycle.NewManager()
	m.Config().ContentDir = dir
	m.SetFiles([]string{".md"})
	m.RegisterPlugins(MinimalPlugins()...)
	if err := m.RunTo(lifecycle.StageTransform); err != nil {
		t.Fatal(err)
	}
	posts := m.Posts()
	if len(posts) != 1 {
		t.Fatalf("posts = %d, want 1", len(posts))
	}
	post := posts[0]
	if *post.Title != "Good **places** to ==think==" {
		t.Fatalf("source title changed: %q", *post.Title)
	}
	if post.TitleText != "Good places to think" || post.TitleTextDerived != true {
		t.Fatalf("semantic title = %q, derived = %v", post.TitleText, post.TitleTextDerived)
	}
	if post.TitleHTML != `Good <strong>places</strong> to <span class="heading-highlight"><mark>think</mark></span>` {
		t.Fatalf("title HTML = %q", post.TitleHTML)
	}
	if post.Slug != "good-places-to-think" {
		t.Fatalf("slug = %q", post.Slug)
	}
	if got := templates.GetPostMap(post)["title"]; got != "Good places to think" {
		t.Fatalf("template title = %#v", got)
	}
}

func TestInlineTitles_EmptySemanticTitleDoesNotFallBackToSource(t *testing.T) {
	p := NewRenderMarkdownPlugin()
	m := lifecycle.NewManager()
	if err := p.Configure(m); err != nil {
		t.Fatal(err)
	}
	title := "<!-- hidden -->"
	post := models.NewPost("hidden.md")
	post.Title = &title
	m.AddPost(post)
	if err := NewInlineTitlesPlugin().Transform(m); err != nil {
		t.Fatal(err)
	}
	if !post.TitleTextDerived || post.PlainTitle() != "" {
		t.Fatalf("plain title = %q, derived = %v", post.PlainTitle(), post.TitleTextDerived)
	}
	if got := templates.GetPostMap(post)["title"]; got != "" {
		t.Fatalf("template title leaked source markup: %#v", got)
	}
}

func TestPluginRegistry_ResolvesInlineTitles(t *testing.T) {
	plugin, ok := PluginByName("inline_titles")
	if !ok || plugin.Name() != "inline_titles" {
		t.Fatalf("inline_titles registry result = %#v, %v", plugin, ok)
	}
	plugins, warnings := ByNames([]string{"inline_titles"})
	if len(warnings) != 0 || len(plugins) != 1 {
		t.Fatalf("ByNames() = %d plugins, warnings %v", len(plugins), warnings)
	}
}

func TestParsePostFromContent_DerivesTitleFallbackSlug(t *testing.T) {
	post, err := ParsePostFromContent(".md", "---\ntitle: Good **places** to ==think==\n---\nBody\n")
	if err != nil {
		t.Fatal(err)
	}
	if post.Slug != "good-places-to-think" || post.Href != "/good-places-to-think/" {
		t.Fatalf("slug/href = %q/%q", post.Slug, post.Href)
	}
	if post.PlainTitle() != "Good places to think" {
		t.Fatalf("plain title = %q", post.PlainTitle())
	}
}

func TestInlineTitles_InitializesSharedRendererWhenNotRegistered(t *testing.T) {
	m := lifecycle.NewManager()
	title := "Good **places**"
	post := models.NewPost("post.md")
	post.Title = &title
	m.AddPost(post)
	if err := NewInlineTitlesPlugin().Transform(m); err != nil {
		t.Fatal(err)
	}
	if post.TitleText != "Good places" || post.TitleHTML != "Good <strong>places</strong>" {
		t.Fatalf("derived title = %q / %q", post.TitleText, post.TitleHTML)
	}
}
