package plugins

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/buildcache"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestConfigFilesHash_ChangesWhenOverlayChanges(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "markata-go.toml")
	overlay := filepath.Join(dir, "tailwind.toml")

	if err := os.WriteFile(base, []byte("title = 'base'\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(base) error = %v", err)
	}
	if err := os.WriteFile(overlay, []byte("[markata-go.tailwind]\nbuild = true\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(overlay) error = %v", err)
	}

	first := buildcache.ContentHash(configFilesHash([]string{base, overlay}))

	if err := os.WriteFile(overlay, []byte("[markata-go.tailwind]\nbuild = false\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(overlay update) error = %v", err)
	}

	second := buildcache.ContentHash(configFilesHash([]string{base, overlay}))
	if first == second {
		t.Fatal("expected config hash to change when overlay config changes")
	}
}

func TestConfigFilesHash_IsStableAcrossPathFormsAndOrder(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "markata-go.toml")
	overlay := filepath.Join(dir, "tailwind.toml")

	if err := os.WriteFile(base, []byte("title = 'base'\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(base) error = %v", err)
	}
	if err := os.WriteFile(overlay, []byte("[markata-go.tailwind]\nbuild = true\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(overlay) error = %v", err)
	}

	first := buildcache.ContentHash(configFilesHash([]string{overlay, base, ""}))
	second := buildcache.ContentHash(configFilesHash([]string{filepath.Clean(base), filepath.Clean(overlay)}))
	third := buildcache.ContentHash(configFilesHash([]string{base, overlay}))

	if first != second || second != third {
		t.Fatalf("expected config hash to be stable across path forms and ordering: %q %q %q", first, second, third)
	}
}

func TestConfigHashInput_IsStableForEquivalentConfig(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "markata-go.toml")
	if err := os.WriteFile(base, []byte("title = 'base'\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(base) error = %v", err)
	}

	config := &lifecycle.Config{
		ContentDir:   ".",
		OutputDir:    "output",
		GlobPatterns: []string{"**/*.md"},
		Extra: map[string]interface{}{
			"title":       "Test Site",
			"url":         "https://example.com",
			"config_path": base,
			"config_paths": []string{
				base,
			},
		},
	}

	first := buildcache.ContentHash(configHashInput(config, []string{base}))
	second := buildcache.ContentHash(configHashInput(config, []string{filepath.Clean(base)}))
	if first != second {
		t.Fatalf("expected effective config hash to be stable for equivalent config: %q != %q", first, second)
	}
}

func TestConfigHashInput_IgnoresOutputDir(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "markata-go.toml")
	if err := os.WriteFile(base, []byte("title = 'base'\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(base) error = %v", err)
	}

	configA := &lifecycle.Config{
		ContentDir:   ".",
		OutputDir:    "output",
		GlobPatterns: []string{"**/*.md"},
		Extra: map[string]interface{}{
			"config_path":  base,
			"config_paths": []string{base},
		},
	}
	configB := &lifecycle.Config{
		ContentDir:   ".",
		OutputDir:    "/data/site/releases/20260614T000000Z-site.tmp",
		GlobPatterns: []string{"**/*.md"},
		Extra: map[string]interface{}{
			"config_path":  base,
			"config_paths": []string{base},
		},
	}

	first := buildcache.ContentHash(configHashInput(configA, []string{base}))
	second := buildcache.ContentHash(configHashInput(configB, []string{base}))
	if first != second {
		t.Fatalf("expected config hash to ignore output dir changes: %q != %q", first, second)
	}
}

func TestBuildCacheConfigure_DefaultsCacheDirToContentDir(t *testing.T) {
	contentDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "build", "site")

	config := &lifecycle.Config{
		ContentDir: contentDir,
		OutputDir:  outputDir,
		Extra: map[string]interface{}{
			"config_path": filepath.Join(contentDir, "markata-go.toml"),
		},
	}

	plugin := NewBuildCachePlugin()
	manager := lifecycle.NewManager()
	cfg := manager.Config()
	*cfg = *config

	if err := plugin.Configure(manager); err != nil {
		t.Fatalf("Configure() error = %v", err)
	}

	cache := GetBuildCache(manager)
	if cache == nil {
		t.Fatal("expected build cache to be stored on manager")
	}

	want := filepath.Join(contentDir, ".markata", buildcache.CacheFileName)
	if cachePath := cachePathForTest(cache); cachePath != want {
		t.Fatalf("cache path = %q, want %q", cachePath, want)
	}
}

func TestBuildCachePlugin_TransformClearsRemovedDependencies(t *testing.T) {
	cache := buildcache.New(t.TempDir())
	plugin := NewBuildCachePlugin()
	plugin.cache = cache
	m := lifecycle.NewManager()
	m.SetPosts([]*models.Post{{Path: "a.md", Slug: "a", Dependencies: []string{"b"}}})

	if err := plugin.Transform(m); err != nil {
		t.Fatal(err)
	}
	if got := cache.Graph.GetDependencies("a.md"); len(got) != 1 || got[0] != "b" {
		t.Fatalf("initial dependencies = %v", got)
	}

	m.Posts()[0].Dependencies = nil
	if err := plugin.Transform(m); err != nil {
		t.Fatal(err)
	}
	if got := cache.Graph.GetDependencies("a.md"); len(got) != 0 {
		t.Fatalf("dependencies after removal = %v, want empty", got)
	}
	if cache.Graph.HasDependents("b") {
		t.Fatal("reverse dependency was not removed")
	}
}

func TestBuildCachePlugin_LoadRefreshesAffectedPathsAfterNewSlug(t *testing.T) {
	cache := buildcache.New(t.TempDir())
	cache.SetDependencies("source.md", "source", []string{"future-target"})
	cache.MarkSlugChanged("future-target")

	plugin := NewBuildCachePlugin()
	plugin.cache = cache
	m := lifecycle.NewManager()
	targetTitle := "Future Target"
	source := &models.Post{Path: "source.md", Slug: "source", Content: "See [[future target]]"}
	target := &models.Post{Path: "future-target.md", Slug: "future-target", Title: &targetTitle, Href: "/future-target/"}
	m.SetPosts([]*models.Post{target, source})
	lifecycle.SetServeIncremental(m, true)
	lifecycle.SetServeChangedPaths(m, []string{"future-target.md"})
	m.SetFiles([]string{"source.md", "future-target.md"})

	if err := plugin.Load(m); err != nil {
		t.Fatal(err)
	}
	if affected := lifecycle.GetServeAffectedPaths(m); !affected["source.md"] {
		t.Fatalf("affected paths = %v, want source.md", affected)
	}
	if err := NewWikilinksPlugin().Transform(m); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(source.Content, `<a href="/future-target/" class="wikilink"`) {
		t.Fatalf("newly resolvable wikilink was not rebuilt: %q", source.Content)
	}
}

func TestBuildCachePlugin_LoadMatchesNormalizedExplicitSlug(t *testing.T) {
	cache := buildcache.New(t.TempDir())
	cache.SetDependencies("source.md", "source", []string{"my post", "my-post"})
	cache.SetPostSlug("my-post.md", "My Post")

	plugin := NewBuildCachePlugin()
	plugin.cache = cache
	m := lifecycle.NewManager()
	m.SetPosts([]*models.Post{
		{Path: "source.md", Slug: "source"},
		{Path: "my-post.md", Slug: "My Post"},
	})
	lifecycle.SetServeIncremental(m, true)
	lifecycle.SetServeChangedPaths(m, []string{"my-post.md"})
	m.SetFiles([]string{"source.md", "my-post.md"})

	if err := plugin.Load(m); err != nil {
		t.Fatal(err)
	}
	if affected := lifecycle.GetServeAffectedPaths(m); !affected["source.md"] {
		t.Fatalf("affected paths = %v, want source.md", affected)
	}
}

func TestShouldCleanupCacheAsync_DisablesSharedServeCacheCleanup(t *testing.T) {
	tests := []struct {
		name        string
		fast        bool
		incremental bool
		serial      bool
		want        bool
	}{
		{name: "ordinary build", want: true},
		{name: "fast serve", fast: true},
		{name: "incremental serve", incremental: true},
		{name: "serial DAG", serial: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := lifecycle.NewManager()
			m.Config().Extra["cache_cleanup_async"] = true
			if tt.fast {
				m.Config().Extra["fast_mode"] = true
			}
			if tt.incremental {
				lifecycle.SetServeIncremental(m, true)
			}
			m.SetSerialBuild(tt.serial)
			if got := shouldCleanupCacheAsync(m); got != tt.want {
				t.Fatalf("shouldCleanupCacheAsync() = %t, want %t", got, tt.want)
			}
		})
	}
}

func cachePathForTest(c *buildcache.Cache) string {
	return filepath.Clean(reflect.ValueOf(c).Elem().FieldByName("path").String())
}
