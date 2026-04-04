package serveadmin

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestResolveContentFilePath_RejectsTraversal(t *testing.T) {
	t.Helper()
	SetContentDir(t.TempDir())

	if _, _, err := resolveContentFilePath("../secrets.txt"); err == nil {
		t.Fatal("resolveContentFilePath() should reject traversal")
	}
}

func TestLoadPostEditData_NewPostDraft(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	pagesDir := filepath.Join(root, "pages")
	if err := os.MkdirAll(pagesDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	SetContentDir(root)

	post, err := loadPostEditData("")
	if err != nil {
		t.Fatalf("loadPostEditData() error = %v", err)
	}
	if post.Exists {
		t.Fatal("new draft should not exist yet")
	}
	if post.Path != "pages/new-post.md" {
		t.Fatalf("Path = %q, want %q", post.Path, "pages/new-post.md")
	}
}

func TestRenderPage_UsesThemeCSS(t *testing.T) {
	t.Helper()
	SetSiteConfig(&models.Config{Theme: models.ThemeConfig{PaletteLight: "default-light", PaletteDark: "default-dark", FallbackMode: "light"}})
	recorder := httptest.NewRecorder()

	renderPage(recorder, "auth", PageData{Title: "Login", NeedsLogin: true})

	body := recorder.Body.String()
	if recorder.Code != 200 {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(body, "--admin-bg") {
		t.Fatal("rendered page should include admin theme css variables")
	}
	if !strings.Contains(body, "Admin login") {
		t.Fatal("rendered auth page should include login heading")
	}
}

func TestRenderPage_UsesConfiguredThemeFonts(t *testing.T) {
	t.Helper()
	SetSiteConfig(&models.Config{Theme: models.ThemeConfig{Font: models.FontConfig{Family: "'IBM Plex Serif', serif", HeadingFamily: "'Oswald', sans-serif", CodeFamily: "'Fira Code', monospace", Size: "18px", LineHeight: "1.8"}}})
	recorder := httptest.NewRecorder()

	renderPage(recorder, "auth", PageData{Title: "Login", NeedsLogin: true})

	body := recorder.Body.String()
	if !strings.Contains(body, "--admin-font-body: 'IBM Plex Serif', serif;") {
		t.Fatal("rendered page should include configured body font")
	}
	if !strings.Contains(body, "--admin-font-heading: 'Oswald', sans-serif;") {
		t.Fatal("rendered page should include configured heading font")
	}
	if !strings.Contains(body, "--admin-font-code: 'Fira Code', monospace;") {
		t.Fatal("rendered page should include configured code font")
	}
	if !strings.Contains(body, "--admin-font-size: 18px;") || !strings.Contains(body, "--admin-line-height: 1.8;") {
		t.Fatal("rendered page should include configured font size and line height")
	}
}

func TestSavePostFromRequest_Create(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pages"), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	SetContentDir(root)
	SetWatchEnabled(true)

	req := httptest.NewRequest("POST", "/__admin/api/post", strings.NewReader(`{"frontmatter":"title: Hello Admin","body":"Body"}`))
	result, err := savePostFromRequest(req, false)
	if err != nil {
		t.Fatalf("savePostFromRequest() error = %v", err)
	}
	if result["preview_url"] != "/hello-admin/" {
		t.Fatalf("preview_url = %v, want %v", result["preview_url"], "/hello-admin/")
	}
	if _, err := os.Stat(filepath.Join(root, "pages", "hello-admin.md")); err != nil {
		t.Fatalf("saved file not found: %v", err)
	}
}

func TestListPostInfos_PrefersSitePostsOverFilesystemWalk(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	SetContentDir(root)
	venvDir := filepath.Join(root, ".venv")
	if err := os.MkdirAll(venvDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(venvDir, "README.md"), []byte("# not content"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	title := "Real Post"
	now := time.Now().UTC()
	SetSitePosts([]*models.Post{{Path: filepath.Join(root, "posts", "real.md"), Slug: "real", Title: &title, Date: &now, Published: true, Draft: false, Tags: []string{"go"}}})

	posts, err := listPostInfos()
	if err != nil {
		t.Fatalf("listPostInfos() error = %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("len(posts) = %d, want 1", len(posts))
	}
	if posts[0].Slug != "real" {
		t.Fatalf("Slug = %q, want real", posts[0].Slug)
	}
}

func TestListPostInfos_FallbackUsesBuildGlobRules(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	SetContentDir(root)
	SetSitePosts(nil)
	SetSiteConfig(&models.Config{GlobConfig: models.GlobConfig{Patterns: []string{"posts/**/*.md"}, UseGitignore: true}})
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".venv/\noutput/\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.gitignore) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "posts"), 0o755); err != nil {
		t.Fatalf("MkdirAll(posts) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".venv"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.venv) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "output"), 0o755); err != nil {
		t.Fatalf("MkdirAll(output) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "posts", "real.md"), []byte("---\ntitle: Real\n---\nbody"), 0o644); err != nil {
		t.Fatalf("WriteFile(real.md) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".venv", "ignore.md"), []byte("---\ntitle: Ignore\n---\nbody"), 0o644); err != nil {
		t.Fatalf("WriteFile(ignore.md) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "output", "also-ignore.md"), []byte("---\ntitle: Ignore\n---\nbody"), 0o644); err != nil {
		t.Fatalf("WriteFile(also-ignore.md) error = %v", err)
	}

	posts, err := listPostInfos()
	if err != nil {
		t.Fatalf("listPostInfos() error = %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("len(posts) = %d, want 1", len(posts))
	}
	if posts[0].Path != "posts/real.md" {
		t.Fatalf("Path = %q, want posts/real.md", posts[0].Path)
	}
}

func TestGenerateAdminNewPostScaffold(t *testing.T) {
	t.Helper()
	result, err := generateAdminNewPostScaffold("My New Note", "note", "pages/note", []string{"one", "two"}, true, []string{"waylon"}, map[string]string{})
	if err != nil {
		t.Fatalf("generateAdminNewPostScaffold() error = %v", err)
	}
	if result["path"] != "pages/note/my-new-note.md" {
		t.Fatalf("path = %v, want %v", result["path"], "pages/note/my-new-note.md")
	}
	frontmatter, _ := result["frontmatter"].(string)
	if !strings.Contains(frontmatter, "template: note") || !strings.Contains(frontmatter, "private: true") || !strings.Contains(frontmatter, "authors:") {
		t.Fatalf("frontmatter missing expected values: %s", frontmatter)
	}
}

func TestRenderLivePreview_RendersMarkdown(t *testing.T) {
	t.Helper()
	html, err := renderLivePreview("title: Preview Title", "# Heading\n\nBody")
	if err != nil {
		t.Fatalf("renderLivePreview() error = %v", err)
	}
	if !strings.Contains(html, "Preview Title") {
		t.Fatal("live preview should include title")
	}
	if !strings.Contains(html, "<h1>Heading</h1>") {
		t.Fatal("live preview should render markdown heading")
	}
}

func TestParseAndRenderFrontmatterForm_RoundTrip(t *testing.T) {
	t.Helper()
	form, err := parseFrontmatterForm("title: Hello\npublished: true\ntags:\n  - one\ncustom: value\n")
	if err != nil {
		t.Fatalf("parseFrontmatterForm() error = %v", err)
	}
	if form.Title != "Hello" || !form.Published {
		t.Fatalf("unexpected parsed form: %+v", form)
	}
	if len(form.Extras) != 1 || form.Extras[0].Key != "custom" {
		t.Fatalf("extras = %+v, want custom field", form.Extras)
	}
	if form.Extras[0].Kind != "string" {
		t.Fatalf("extra kind = %q, want string", form.Extras[0].Kind)
	}
	raw, err := renderFrontmatterForm(form)
	if err != nil {
		t.Fatalf("renderFrontmatterForm() error = %v", err)
	}
	if !strings.Contains(raw, "title: Hello") || !strings.Contains(raw, "published: true") {
		t.Fatalf("rendered frontmatter missing expected content: %s", raw)
	}
}

func TestParseFrontmatterForm_ParsesTimestampDateAndAuthors(t *testing.T) {
	t.Helper()
	form, err := parseFrontmatterForm("title: Hello\ndate: 2025-01-12T21:07:12Z\nmodified: 2025-01-13T08:00:00Z\nauthors:\n  - waylon\ntemplateKey: blog-post\npublished: true\n")
	if err != nil {
		t.Fatalf("parseFrontmatterForm() error = %v", err)
	}
	if form.Date != "2025-01-12T21:07:12Z" {
		t.Fatalf("Date = %q, want timestamp", form.Date)
	}
	if form.Modified != "2025-01-13T08:00:00Z" {
		t.Fatalf("Modified = %q, want timestamp", form.Modified)
	}
	if form.TemplateKey != "blog-post" {
		t.Fatalf("TemplateKey = %q, want blog-post", form.TemplateKey)
	}
	if len(form.Authors) != 1 || form.Authors[0] != "waylon" {
		t.Fatalf("Authors = %+v, want [waylon]", form.Authors)
	}
}

func TestValidateFrontmatterForm_AllowsTemplateAliases(t *testing.T) {
	t.Helper()
	if err := validateFrontmatterForm(adminFrontmatterForm{Title: "Hello", TemplateKey: "blog-post"}); err != nil {
		t.Fatalf("validateFrontmatterForm() error = %v, want alias accepted", err)
	}
}

func TestRenderFrontmatterForm_ObjectExtra(t *testing.T) {
	t.Helper()
	raw, err := renderFrontmatterForm(adminFrontmatterForm{
		Title:  "Hello",
		Extras: []adminKeyValueField{{Key: "seo", Kind: "object", Value: "description: Better\nimage: /cover.png"}},
	})
	if err != nil {
		t.Fatalf("renderFrontmatterForm() error = %v", err)
	}
	if !strings.Contains(raw, "seo:") || !strings.Contains(raw, "image: /cover.png") {
		t.Fatalf("expected nested object in frontmatter, got: %s", raw)
	}
}

func TestParseAndRenderSettingsForm(t *testing.T) {
	t.Helper()
	raw := "[markata-go]\ntitle = \"Site\"\n[markata-go.theme]\npalette = \"nord-dark\"\n"
	form := parseSettingsForm(raw)
	if form.Title != "Site" || form.ThemePalette != "nord-dark" {
		t.Fatalf("unexpected settings form: %+v", form)
	}
	form.Author = "Waylon"
	form.SearchPosition = "navbar"
	form.SearchEnabled = true
	form.PagefindBundleDir = "_pagefind"
	form.PagefindVersion = "latest"
	form.PagefindAutoInstall = true
	form.ThemeSwitcherEnabled = true
	form.ThemeSwitcherPosition = "header"
	form.ThemeModeToggleEnabled = true
	form.ThemeIncludeAll = true
	form.FontFamily = "IBM Plex Serif"
	updated, err := renderSettingsForm(raw, form)
	if err != nil {
		t.Fatalf("renderSettingsForm() error = %v", err)
	}
	if !strings.Contains(updated, "author = \"Waylon\"") {
		t.Fatalf("updated config missing author: %s", updated)
	}
	if !strings.Contains(updated, "[markata-go.search]") || !strings.Contains(updated, "enabled = true") {
		t.Fatalf("updated config missing search settings: %s", updated)
	}
	if !strings.Contains(updated, "[markata-go.search.pagefind]") || !strings.Contains(updated, "bundle_dir = \"_pagefind\"") {
		t.Fatalf("updated config missing pagefind settings: %s", updated)
	}
	if !strings.Contains(updated, "[markata-go.theme.font]") || !strings.Contains(updated, "family = \"IBM Plex Serif\"") {
		t.Fatalf("updated config missing font settings: %s", updated)
	}
}

func TestValidateSettingsForm_RejectsUnknownPalette(t *testing.T) {
	t.Helper()
	err := validateSettingsForm(adminSettingsForm{ThemePalette: "not-a-palette"})
	if err == nil {
		t.Fatal("validateSettingsForm() should reject unknown palette")
	}
}

func TestHandlePreviewPost_ReturnsHTML(t *testing.T) {
	t.Helper()
	req := httptest.NewRequest("POST", "/__admin/api/preview", strings.NewReader(`{"frontmatter":"title: Preview","body":"**bold**"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handlePreviewPost(recorder, req)

	if recorder.Code != 200 {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "<strong>bold</strong>") {
		t.Fatal("preview route should render markdown html")
	}
}

func TestRenderPage_EditorIncludesCommandAndModeControls(t *testing.T) {
	t.Helper()
	recorder := httptest.NewRecorder()
	renderPage(recorder, "editor", PageData{Title: "Editor", IsEditor: true, Post: PostEditData{Path: "pages/post/example.md", Exists: true}})
	body := recorder.Body.String()
	if !strings.Contains(body, "/ commands") {
		t.Fatal("editor should include slash command controls")
	}
	if !strings.Contains(body, "toggle-focus-mode") || !strings.Contains(body, "toggle-typewriter-mode") {
		t.Fatal("editor should include focus and typewriter mode controls")
	}
	if !strings.Contains(body, "command-palette") {
		t.Fatal("editor should render the command palette container")
	}
	if !strings.Contains(body, "fm-author-picker") || !strings.Contains(body, "fm-author-search") || !strings.Contains(body, "fm-slug-default") || !strings.Contains(body, "fm-slug-reset") || !strings.Contains(body, "last-autosaved") {
		t.Fatal("editor should render upgraded properties controls")
	}
	if !strings.Contains(body, "<select id=\"fm-template-key\"") {
		t.Fatal("editor should render type as a select control")
	}
	if !strings.Contains(body, "open-properties-panel") || !strings.Contains(body, "open-preview-panel") {
		t.Fatal("editor should render explicit panel buttons")
	}
	if strings.Contains(body, "id=\"toggle-preview-pane\"") || strings.Contains(body, "id=\"toggle-meta\"") {
		t.Fatal("editor should not render cryptic P/I icon buttons")
	}
}
