package plugins

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/imageindex"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestParseImageLibraryConfig_DefaultsAndExplicitFalse(t *testing.T) {
	defaults, err := parseImageLibraryConfig(nil)
	if err != nil {
		t.Fatalf("parseImageLibraryConfig(nil) error = %v", err)
	}
	if !defaults.IsEnabled() || !defaults.ShouldExportJSON() || !defaults.ShouldIncludeUnreferenced() {
		t.Fatalf("defaults = %#v", defaults)
	}

	configured, err := parseImageLibraryConfig(map[string]interface{}{
		"images": map[string]interface{}{
			"enabled":              false,
			"path":                 "media",
			"template":             "media.html",
			"export_json":          false,
			"include_unreferenced": false,
		},
	})
	if err != nil {
		t.Fatalf("parseImageLibraryConfig() error = %v", err)
	}
	if configured.IsEnabled() || configured.ShouldExportJSON() || configured.ShouldIncludeUnreferenced() || configured.Path != "media" || configured.Template != "media.html" {
		t.Fatalf("configured = %#v", configured)
	}
}

func TestImageLibraryPage_UsesCanonicalDropperDerivatives(t *testing.T) {
	index := imageIndexForTest()
	page := newImageLibraryPage(index)
	if len(page.Images) != 2 {
		t.Fatalf("page images = %#v", page.Images)
	}

	remote := page.Images[1]
	if !strings.Contains(remote.PreviewSrc, "w=640") || !strings.Contains(remote.Srcset, "w=320") || !strings.Contains(remote.Srcset, "w=1280") {
		t.Fatalf("remote presentation URLs = preview %q, srcset %q", remote.PreviewSrc, remote.Srcset)
	}
	if remote.Src != "https://dropper.wayl.one/file/photo.webp?token=keep" {
		t.Fatalf("canonical source changed = %q", remote.Src)
	}
	if !strings.Contains(remote.Markdown, remote.Src) {
		t.Fatalf("copy Markdown does not use canonical source: %q", remote.Markdown)
	}
	if page.UsedCount != 1 || page.UnusedCount != 1 || page.CoverCount != 1 {
		t.Fatalf("page counts = %#v", page)
	}
}

func TestImageLibraryPage_RendersVideoMetadataAndEmbedLabel(t *testing.T) {
	index := imageindex.Index{Images: []imageindex.Image{{
		Src:       "https://dropper.wayl.one/file/clip.mp4?token=keep",
		MIMEType:  "video/mp4",
		PosterSrc: "http://dropper.wayl.one/file/clip.webp",
		Embed:     true,
		Uses:      []imageindex.Use{{Post: "posts/clip.md", Href: "/clip/", Embed: true}},
	}}}

	page := newImageLibraryPage(index)
	if len(page.Images) != 1 {
		t.Fatalf("page images = %#v", page.Images)
	}
	card := page.Images[0]
	if !card.IsVideo || !card.Embed {
		t.Fatalf("video card metadata = %#v", card)
	}
	if !strings.Contains(card.PosterSrc, "clip.webp") || !strings.Contains(card.PosterSrc, "w=640") {
		t.Fatalf("video poster = %q", card.PosterSrc)
	}
	if card.Srcset != "" {
		t.Fatalf("video srcset = %q, want empty", card.Srcset)
	}
	if !strings.Contains(card.SearchText, "video") || !strings.Contains(card.SearchText, "embed") {
		t.Fatalf("video search text = %q", card.SearchText)
	}
}

func TestImageLibraryPage_FormatsAddedAtInUTC(t *testing.T) {
	addedAt := time.Date(2026, time.January, 2, 23, 30, 0, 0, time.FixedZone("west", -8*60*60))
	page := newImageLibraryPage(imageindex.Index{Images: []imageindex.Image{{Src: "/images/photo.png", AddedAt: &addedAt}}})
	if got := page.Images[0].AddedAt; got != "Jan 3, 2026" {
		t.Fatalf("AddedAt = %q, want UTC date", got)
	}
}

func TestImageLibraryPage_FormatsLastUsedAtInUTC(t *testing.T) {
	lastUsedAt := time.Date(2026, time.January, 2, 23, 30, 0, 0, time.FixedZone("west", -8*60*60))
	page := newImageLibraryPage(imageindex.Index{Images: []imageindex.Image{{Src: "/images/photo.png", LastUsedAt: &lastUsedAt}}})
	if got := page.Images[0].LastUsedAt; got != "Jan 3, 2026" {
		t.Fatalf("LastUsedAt = %q, want UTC date", got)
	}
	if page.Images[0].LastUsedAtUnix != lastUsedAt.Unix() {
		t.Fatalf("LastUsedAtUnix = %d, want %d", page.Images[0].LastUsedAtUnix, lastUsedAt.Unix())
	}
}

func TestImageLibraryPlugin_WriteOutputsPageAndJSON(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	assets := filepath.Join(root, "static")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}

	post := models.NewPost("posts/hello.md")
	post.Slug = "hello"
	post.Href = "/hello/"
	post.Published = true
	post.Title = stringPointerImages("Hello")
	post.Content = "![Hello image](https://dropper.wayl.one/file/hello.webp)"
	videoPost := models.NewPost("posts/video.md")
	videoPost.Slug = "video"
	videoPost.Href = "/video/"
	videoPost.Published = true
	videoPost.Title = stringPointerImages("Video")
	videoDate := time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC)
	videoPost.Date = &videoDate
	videoPost.Extra["image"] = "https://dropper.wayl.one/file/video.mp4"
	videoPost.Extra["poster_image"] = "https://dropper.wayl.one/file/video.webp"

	modelsConfig := models.NewConfig()
	modelsConfig.OutputDir = output
	modelsConfig.AssetsDir = assets
	modelsConfig.Images = models.NewImagesConfig()
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{
		ContentDir: root,
		OutputDir:  output,
		Extra: map[string]interface{}{
			"assets_dir":      assets,
			"markata_version": "test",
			"models_config":   modelsConfig,
		},
	})
	manager.SetPosts([]*models.Post{post, videoPost})
	manager.RegisterPlugin(NewImageLibraryPlugin())

	if err := manager.RunTo(lifecycle.StageWrite); err != nil {
		t.Fatalf("RunTo(write) error = %v", err)
	}

	pagePath := filepath.Join(output, "images", "index.html")
	jsonPath := filepath.Join(output, "images", "index.json")
	flatJSONPath := filepath.Join(output, "images.json")
	page, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("read generated page: %v", err)
	}
	if !strings.Contains(string(page), "Copy Markdown") || !strings.Contains(string(page), "hello.webp") || !strings.Contains(string(page), "data-image-filter=\"used\"") {
		t.Fatalf("generated page missing image-library controls/content: %s", page)
	}
	if !strings.Contains(string(page), "<video") || !strings.Contains(string(page), `<source src="https://dropper.wayl.one/file/video.mp4?w=640" type="video/mp4">`) || !strings.Contains(string(page), `poster="https://dropper.wayl.one/file/video.webp?w=640"`) || !strings.Contains(string(page), ">Video</span>") || !strings.Contains(string(page), `data-last-used="1768046400"`) || !strings.Contains(string(page), ">Latest used</option>") || strings.Contains(string(page), `<img src="https://dropper.wayl.one/file/video.mp4`) {
		t.Fatalf("generated page missing video presentation: %s", page)
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read generated JSON: %v", err)
	}
	var artifact map[string]interface{}
	if err := json.Unmarshal(data, &artifact); err != nil {
		t.Fatalf("generated JSON is invalid: %v", err)
	}
	if artifact["image_count"] != float64(2) {
		t.Fatalf("generated image_count = %#v", artifact["image_count"])
	}
	flatData, err := os.ReadFile(flatJSONPath)
	if err != nil {
		t.Fatalf("read root image index: %v", err)
	}
	if !bytes.Equal(flatData, data) {
		t.Fatalf("root image index differs from nested artifact")
	}
}

func TestImageLibraryPlugin_ExportJSONFalseRemovesStaleArtifact(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	assets := filepath.Join(root, "static")
	if err := os.MkdirAll(filepath.Join(output, "images"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "images", "index.json"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "images.json"), []byte("stale-root"), 0o600); err != nil {
		t.Fatal(err)
	}

	modelsConfig := models.NewConfig()
	modelsConfig.OutputDir = output
	modelsConfig.AssetsDir = assets
	modelsConfig.Images = models.NewImagesConfig()
	noJSON := false
	modelsConfig.Images.ExportJSON = &noJSON
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{
		ContentDir: root,
		OutputDir:  output,
		Extra: map[string]interface{}{
			"assets_dir":    assets,
			"models_config": modelsConfig,
		},
	})
	cache := buildcache.New(filepath.Join(root, "cache"))
	cache.SetImageLibraryOutputs(root, output, map[string]string{
		filepath.Join(output, "images", "index.html"): "stale-page",
		filepath.Join(output, "images", "index.json"): buildcache.ContentHash("stale"),
		filepath.Join(output, "images.json"):          buildcache.ContentHash("stale-root"),
	})
	// The stale JSON is the only file that exists. The page hash is retained to
	// verify that missing plugin-owned files do not block cleanup.
	manager.Cache().Set("build_cache", cache)
	manager.RegisterPlugin(NewImageLibraryPlugin())

	if err := manager.RunTo(lifecycle.StageWrite); err != nil {
		t.Fatalf("RunTo(write) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(output, "images", "index.json")); !os.IsNotExist(err) {
		t.Fatalf("stale JSON artifact still exists, stat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(output, "images.json")); !os.IsNotExist(err) {
		t.Fatalf("stale root JSON artifact still exists, stat error = %v", err)
	}
	page, err := os.ReadFile(filepath.Join(output, "images", "index.html"))
	if err != nil {
		t.Fatalf("read generated page: %v", err)
	}
	if strings.Contains(string(page), "Download JSON index") {
		t.Fatal("page links to JSON after export_json=false")
	}
}

func TestImageLibraryPlugin_ExportJSONFalseReportsUnownedArtifact(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	assets := filepath.Join(root, "static")
	jsonPath := filepath.Join(output, "images", "index.json")
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, []byte("site-owned"), 0o600); err != nil {
		t.Fatal(err)
	}

	modelsConfig := models.NewConfig()
	modelsConfig.OutputDir = output
	modelsConfig.AssetsDir = assets
	noJSON := false
	modelsConfig.Images.ExportJSON = &noJSON
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{
		ContentDir: root,
		OutputDir:  output,
		Extra: map[string]interface{}{
			"assets_dir":    assets,
			"models_config": modelsConfig,
		},
	})

	err := NewImageLibraryPlugin().Write(manager)
	if err == nil || !strings.Contains(err.Error(), "not owned") {
		t.Fatalf("Write() error = %v, want unowned-output conflict", err)
	}
	if data, readErr := os.ReadFile(jsonPath); readErr != nil || string(data) != "site-owned" {
		t.Fatalf("unowned JSON changed: data=%q error=%v", data, readErr)
	}
}

func TestImageLibraryPlugin_RejectsUnownedRootJSONArtifact(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	assets := filepath.Join(root, "static")
	flatJSONPath := filepath.Join(output, "images.json")
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(flatJSONPath, []byte("site-owned"), 0o600); err != nil {
		t.Fatal(err)
	}

	modelsConfig := models.NewConfig()
	modelsConfig.OutputDir = output
	modelsConfig.AssetsDir = assets
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{
		ContentDir: root,
		OutputDir:  output,
		Extra: map[string]interface{}{
			"assets_dir":    assets,
			"models_config": modelsConfig,
		},
	})

	err := NewImageLibraryPlugin().Write(manager)
	if err == nil || !strings.Contains(err.Error(), "not owned") {
		t.Fatalf("Write() error = %v, want unowned root output conflict", err)
	}
	if data, readErr := os.ReadFile(flatJSONPath); readErr != nil || string(data) != "site-owned" {
		t.Fatalf("unowned root JSON changed: data=%q error=%v", data, readErr)
	}
}

func TestImageLibraryPlugin_RejectsUserReplacementOfCachedOutput(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	assets := filepath.Join(root, "static")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}

	modelsConfig := models.NewConfig()
	modelsConfig.OutputDir = output
	modelsConfig.AssetsDir = assets
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{
		ContentDir: root,
		OutputDir:  output,
		Extra: map[string]interface{}{
			"assets_dir":      assets,
			"markata_version": "test",
			"models_config":   modelsConfig,
		},
	})
	manager.Cache().Set("build_cache", buildcache.New(filepath.Join(root, "cache")))
	plugin := NewImageLibraryPlugin()
	manager.RegisterPlugin(plugin)

	if err := manager.RunTo(lifecycle.StageWrite); err != nil {
		t.Fatalf("first RunTo(write) error = %v", err)
	}
	pagePath := filepath.Join(output, "images", "index.html")
	before, err := os.Stat(pagePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := plugin.Write(manager); err != nil {
		t.Fatalf("cached Write() error = %v", err)
	}
	after, err := os.Stat(pagePath)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("cached image-library page was rewritten")
	}
	if err := os.WriteFile(pagePath, []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := plugin.Write(manager); err == nil || !strings.Contains(err.Error(), "was modified") {
		t.Fatalf("second Write() error = %v, want modified-output error", err)
	}
	page, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(page) != "sentinel" {
		t.Fatalf("cached page was rewritten: %q", page)
	}
}

func TestImageLibraryPlugin_RebuildsWhenTemplateChanges(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	assets := filepath.Join(root, "static")
	templatesDir := filepath.Join(root, "templates")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	templatePath := filepath.Join(templatesDir, "images.html")
	if err := os.WriteFile(templatePath, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}

	modelsConfig := models.NewConfig()
	modelsConfig.OutputDir = output
	modelsConfig.AssetsDir = assets
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{
		ContentDir: root,
		OutputDir:  output,
		Extra: map[string]interface{}{
			"assets_dir":    assets,
			"models_config": modelsConfig,
			"templates_dir": templatesDir,
		},
	})
	manager.Cache().Set("build_cache", buildcache.New(filepath.Join(root, "cache")))
	plugin := NewImageLibraryPlugin()
	manager.RegisterPlugin(plugin)

	if err := manager.RunTo(lifecycle.StageWrite); err != nil {
		t.Fatalf("first RunTo(write) error = %v", err)
	}
	pagePath := filepath.Join(output, "images", "index.html")
	first, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != "first" {
		t.Fatalf("first page = %q", first)
	}

	if err := os.WriteFile(templatePath, []byte("second"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := plugin.Write(manager); err != nil {
		t.Fatalf("second Write() error = %v", err)
	}
	second, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(second) != "second" {
		t.Fatalf("template change was not rendered: %q", second)
	}
}

func TestImageLibraryPlugin_RebuildsWhenRenderConfigChanges(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	assets := filepath.Join(root, "static")
	templatesDir := filepath.Join(root, "templates")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	templatePath := filepath.Join(templatesDir, "images.html")
	if err := os.WriteFile(templatePath, []byte("{{ site_title }}"), 0o600); err != nil {
		t.Fatal(err)
	}

	modelsConfig := models.NewConfig()
	modelsConfig.OutputDir = output
	modelsConfig.AssetsDir = assets
	modelsConfig.Title = "First title"
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{
		ContentDir: root,
		OutputDir:  output,
		Extra: map[string]interface{}{
			"assets_dir":    assets,
			"models_config": modelsConfig,
			"templates_dir": templatesDir,
		},
	})
	manager.Cache().Set("build_cache", buildcache.New(filepath.Join(root, "cache")))
	plugin := NewImageLibraryPlugin()
	manager.RegisterPlugin(plugin)

	if err := manager.RunTo(lifecycle.StageWrite); err != nil {
		t.Fatalf("first RunTo(write) error = %v", err)
	}
	pagePath := filepath.Join(output, "images", "index.html")
	first, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != "First title" {
		t.Fatalf("first page = %q", first)
	}

	modelsConfig.Title = "Second title"
	if err := plugin.Write(manager); err != nil {
		t.Fatalf("second Write() error = %v", err)
	}
	second, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(second) != "Second title" {
		t.Fatalf("configuration change was not rendered: %q", second)
	}
}

func TestImageLibraryPlugin_RebuildsWhenReferencedContentImageChanges(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	assets := filepath.Join(root, "static")
	postsDir := filepath.Join(root, "posts")
	if err := os.MkdirAll(postsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(postsDir, "local.png")
	if err := os.WriteFile(imagePath, []byte("first image"), 0o600); err != nil {
		t.Fatal(err)
	}

	modelsConfig := models.NewConfig()
	modelsConfig.OutputDir = output
	modelsConfig.AssetsDir = assets
	post := models.NewPost("posts/example.md")
	post.Slug = "example"
	post.Href = "/example/"
	post.Published = true
	post.Content = "![Local image](local.png)"
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{
		ContentDir: root,
		OutputDir:  output,
		Extra: map[string]interface{}{
			"assets_dir":    assets,
			"models_config": modelsConfig,
		},
	})
	manager.SetPosts([]*models.Post{post})
	cache := buildcache.New(filepath.Join(root, "cache"))
	manager.Cache().Set("build_cache", cache)
	plugin := NewImageLibraryPlugin()

	if err := plugin.Write(manager); err != nil {
		t.Fatalf("first Write() error = %v", err)
	}
	firstHash := cache.GetImageLibraryHash()
	if firstHash == "" {
		t.Fatal("first image-library hash is empty")
	}

	if err := os.WriteFile(imagePath, []byte("second image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := plugin.Write(manager); err != nil {
		t.Fatalf("second Write() error = %v", err)
	}
	if secondHash := cache.GetImageLibraryHash(); secondHash == firstHash {
		t.Fatalf("image-library hash did not change after referenced content image changed: %q", secondHash)
	}
}

func TestHashImageDirectory_NormalizesMediaExtensionCase(t *testing.T) {
	root := t.TempDir()
	mediaPath := filepath.Join(root, "clip.Mp4")
	if err := os.WriteFile(mediaPath, []byte("first video"), 0o600); err != nil {
		t.Fatal(err)
	}

	first, _, err := hashImageDirectory(root, imageHashExtensions())
	if err != nil {
		t.Fatalf("first hashImageDirectory() error = %v", err)
	}
	if err := os.WriteFile(mediaPath, []byte("second video"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, _, err := hashImageDirectory(root, imageHashExtensions())
	if err != nil {
		t.Fatalf("second hashImageDirectory() error = %v", err)
	}
	if first == second {
		t.Fatalf("mixed-case media content hash did not change: %q", first)
	}
}

func TestImageLibraryPlugin_IgnoresDanglingImageSymlinkInContentHash(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	assets := filepath.Join(root, "static")
	link := filepath.Join(root, "ignored.png")
	if err := os.Symlink(filepath.Join(root, "missing.png"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	modelsConfig := models.NewConfig()
	modelsConfig.OutputDir = output
	modelsConfig.AssetsDir = assets
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{
		ContentDir: root,
		OutputDir:  output,
		Extra: map[string]interface{}{
			"assets_dir":    assets,
			"models_config": modelsConfig,
		},
	})

	if err := NewImageLibraryPlugin().Write(manager); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
}

func TestImageLibraryPlugin_RejectsSymlinkedOutputRoot(t *testing.T) {
	root := t.TempDir()
	realOutput := filepath.Join(root, "real-output")
	output := filepath.Join(root, "output")
	if err := os.MkdirAll(realOutput, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realOutput, output); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	assets := filepath.Join(root, "static")
	modelsConfig := models.NewConfig()
	modelsConfig.OutputDir = output
	modelsConfig.AssetsDir = assets
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{
		ContentDir: root,
		OutputDir:  output,
		Extra: map[string]interface{}{
			"assets_dir":    assets,
			"models_config": modelsConfig,
		},
	})

	err := NewImageLibraryPlugin().Write(manager)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("Write() error = %v, want symlink-output-root error", err)
	}
	if _, err := os.Stat(filepath.Join(realOutput, "images", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("symlink target was written, stat error = %v", err)
	}
}

func TestImageLibraryPlugin_PathChangePreservesModifiedOldOutput(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	assets := filepath.Join(root, "static")
	oldPage := filepath.Join(output, "images", "index.html")
	if err := os.MkdirAll(filepath.Dir(oldPage), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldPage, []byte("user content"), 0o600); err != nil {
		t.Fatal(err)
	}

	modelsConfig := models.NewConfig()
	modelsConfig.OutputDir = output
	modelsConfig.AssetsDir = assets
	modelsConfig.Images.Path = "gallery"
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{
		ContentDir: root,
		OutputDir:  output,
		Extra: map[string]interface{}{
			"assets_dir":    assets,
			"models_config": modelsConfig,
		},
	})
	cache := buildcache.New(filepath.Join(root, "cache"))
	cache.SetImageLibraryOutputs(root, output, map[string]string{oldPage: buildcache.ContentHash("generated content")})
	manager.Cache().Set("build_cache", cache)
	plugin := NewImageLibraryPlugin()

	if err := plugin.Write(manager); err == nil || !strings.Contains(err.Error(), "was modified") {
		t.Fatalf("Write() error = %v, want modified-output error", err)
	}
	page, err := os.ReadFile(oldPage)
	if err != nil {
		t.Fatal(err)
	}
	if string(page) != "user content" {
		t.Fatalf("modified old output changed: %q", page)
	}
	if _, err := os.Stat(filepath.Join(output, "gallery", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("new output was written after collision, stat error = %v", err)
	}
}

func TestImageLibraryPlugin_PathChangeCleansPreviousOutputs(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	assets := filepath.Join(root, "static")
	modelsConfig := models.NewConfig()
	modelsConfig.OutputDir = output
	modelsConfig.AssetsDir = assets
	manager := lifecycle.NewManager()
	manager.SetConfig(&lifecycle.Config{
		ContentDir: root,
		OutputDir:  output,
		Extra: map[string]interface{}{
			"assets_dir":    assets,
			"models_config": modelsConfig,
		},
	})
	manager.Cache().Set("build_cache", buildcache.New(filepath.Join(root, "cache")))
	plugin := NewImageLibraryPlugin()

	if err := plugin.Write(manager); err != nil {
		t.Fatalf("first Write() error = %v", err)
	}
	oldPage := filepath.Join(output, "images", "index.html")
	if _, err := os.Stat(oldPage); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(output, "images.json")); err != nil {
		t.Fatalf("root image index missing: %v", err)
	}
	modelsConfig.Images.Path = "gallery"
	if err := plugin.Write(manager); err != nil {
		t.Fatalf("second Write() error = %v", err)
	}
	if _, err := os.Stat(oldPage); !os.IsNotExist(err) {
		t.Fatalf("old image-library page was not removed, stat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(output, "gallery", "index.html")); err != nil {
		t.Fatalf("new image-library page missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(output, "images.json")); err != nil {
		t.Fatalf("root image index was not retained: %v", err)
	}
}

func TestImageLibraryPlugin_RejectsOutputCollisions(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, root string)
		posts      []*models.Post
		wantPhrase string
	}{
		{
			name:       "post",
			posts:      []*models.Post{{Path: "posts/images.md", Slug: "images", Published: true}},
			wantPhrase: "conflicts with post",
		},
		{
			name: "static file",
			setup: func(t *testing.T, root string) {
				t.Helper()
				path := filepath.Join(root, "static", "images", "index.html")
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("owned by site"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantPhrase: "conflicts with static file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, root)
			}
			modelsConfig := models.NewConfig()
			modelsConfig.OutputDir = filepath.Join(root, "output")
			modelsConfig.AssetsDir = filepath.Join(root, "static")
			manager := lifecycle.NewManager()
			manager.SetConfig(&lifecycle.Config{
				ContentDir: root,
				OutputDir:  modelsConfig.OutputDir,
				Extra: map[string]interface{}{
					"assets_dir":    modelsConfig.AssetsDir,
					"models_config": modelsConfig,
				},
			})
			manager.SetPosts(tt.posts)
			manager.RegisterPlugin(NewImageLibraryPlugin())

			if err := manager.RunTo(lifecycle.StageWrite); err != nil {
				t.Fatalf("RunTo(write) error = %v", err)
			}
			warnings := manager.Warnings()
			if len(warnings) != 1 || !strings.Contains(warnings[0].Error(), tt.wantPhrase) {
				t.Fatalf("warnings = %v, want %q", warnings, tt.wantPhrase)
			}
		})
	}
}

func imageIndexForTest() imageindex.Index {
	return imageindex.Index{
		Generator: imageindex.Generator{Name: imageindex.GeneratorName, Version: "test"},
		Images: []imageindex.Image{
			{Src: "/static/unused.png", Alt: "unused"},
			{Src: "https://dropper.wayl.one/file/photo.webp?token=keep", Alt: "photo", Cover: true, Uses: []imageindex.Use{{Post: "posts/photo.md", Href: "/photo/", Title: "Photo", Cover: true}}},
		},
	}
}

func stringPointerImages(value string) *string {
	return &value
}
