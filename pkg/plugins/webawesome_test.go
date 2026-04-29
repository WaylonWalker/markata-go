// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestWebAwesomePlugin_ProcessComparisonContainer(t *testing.T) {
	plugin := NewWebAwesomePlugin()
	post := &models.Post{
		ArticleHTML: `<div position="35" caption="Homepage redesign" class="webawesome comparison">
<p><img src="/before.webp" alt="Before image"> <img src="/after.webp" alt="After image"></p>
</div>`,
	}

	if err := plugin.processPost(post); err != nil {
		t.Fatalf("processPost() error = %v", err)
	}

	want := []string{
		`<figure class="markata-webawesome-figure">`,
		`<wa-comparison class="markata-webawesome-comparison" position="35">`,
		`<img slot="before" src="/before.webp" alt="Before image" loading="lazy">`,
		`<img slot="after" src="/after.webp" alt="After image" loading="lazy">`,
		`<figcaption>Homepage redesign</figcaption>`,
	}
	for _, expected := range want {
		if !strings.Contains(post.ArticleHTML, expected) {
			t.Fatalf("ArticleHTML missing %q\nGot: %s", expected, post.ArticleHTML)
		}
	}
}

func TestWebAwesomePlugin_ProcessComparisonContainerWithFigure(t *testing.T) {
	plugin := NewWebAwesomePlugin()
	post := &models.Post{
		ArticleHTML: `<div class="webawesome comparison" position="42" caption="Compare the same generated graphic before and after a color treatment.">
<figure>
<img src="/before.webp" alt="Before image">
<img src="/after.webp" alt="After image">
</figure>
</div>`,
	}

	if err := plugin.processPost(post); err != nil {
		t.Fatalf("processPost() error = %v", err)
	}

	if !strings.Contains(post.ArticleHTML, `<wa-comparison class="markata-webawesome-comparison" position="42">`) {
		t.Fatalf("ArticleHTML missing wa-comparison\nGot: %s", post.ArticleHTML)
	}
	if !strings.Contains(post.ArticleHTML, `<figcaption>Compare the same generated graphic before and after a color treatment.</figcaption>`) {
		t.Fatalf("ArticleHTML missing full caption\nGot: %s", post.ArticleHTML)
	}
}

func TestWebAwesomePlugin_ProcessUsefulContentContainers(t *testing.T) {
	plugin := NewWebAwesomePlugin()
	post := &models.Post{ArticleHTML: `<div class="webawesome details" summary="Install notes">
<p>Use the binary for your platform.</p>
</div>
<div class="wa-tabs">
<div class="wa-tab" label="macOS">
<pre><code>brew install markata-go</code></pre>
</div>
<div class="wa-tab" label="Linux">
<pre><code>curl -fsSL example.com</code></pre>
</div>
</div>
<div class="webawesome copy"><p>go test ./...</p></div>
<div class="webawesome qr"><p>https://example.com/post</p></div>
<div class="webawesome badge" variant="brand"><p>New</p></div>
<div class="webawesome tag" variant="success"><p>Stable</p></div>
<div class="webawesome tooltip" content="Static Site Generator"><p>SSG</p></div>
<div class="webawesome carousel" navigation="true">
<figure><img src="/one.webp" alt="One"><img src="/two.webp" alt="Two"></figure>
</div>
<div class="webawesome animated-image">
<p><img src="/demo.webp" alt="Animation demo"></p>
</div>`}

	if err := plugin.processPost(post); err != nil {
		t.Fatalf("processPost() error = %v", err)
	}

	want := []string{
		`<wa-details summary="Install notes">`,
		`<wa-tab-group>`,
		`<wa-tab slot="nav" panel="macos">macOS</wa-tab>`,
		`<wa-tab-panel name="linux">`,
		`<wa-copy-button value="go test ./..."></wa-copy-button>`,
		`<wa-qr-code value="https://example.com/post"></wa-qr-code>`,
		`<wa-badge variant="brand">New</wa-badge>`,
		`<wa-tag variant="success">Stable</wa-tag>`,
		`<span class="markata-wa-tooltip-anchor"`,
		`>SSG</span><wa-tooltip for="`,
		`>Static Site Generator</wa-tooltip>`,
		`<wa-carousel navigation="true">`,
		`<wa-carousel-item><img src="/one.webp" alt="One"></wa-carousel-item>`,
		`<wa-animated-image src="/demo.webp" alt="Animation demo"></wa-animated-image>`,
	}
	for _, expected := range want {
		if !strings.Contains(post.ArticleHTML, expected) {
			t.Fatalf("ArticleHTML missing %q\nGot: %s", expected, post.ArticleHTML)
		}
	}
}

func TestWebAwesomePlugin_RenderEnablesAssetsForRawComponent(t *testing.T) {
	plugin := NewWebAwesomePlugin()
	plugin.config.Source = "cdn"
	m := lifecycle.NewManager()
	m.SetConfig(&lifecycle.Config{Extra: map[string]interface{}{}})
	m.SetPosts([]*models.Post{{ArticleHTML: `<wa-button>Click</wa-button>`}})

	if err := plugin.Render(m); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if got, ok := m.Config().Extra["webawesome_enabled"].(bool); !ok || !got {
		t.Fatalf("webawesome_enabled = %v, want true", m.Config().Extra["webawesome_enabled"])
	}
	if got := m.Config().Extra["webawesome_css_url"]; got != webAwesomeDefaultCDNBase+"/styles/webawesome.css" {
		t.Fatalf("webawesome_css_url = %v", got)
	}
	if got := m.Config().Extra["webawesome_loader_url"]; got != webAwesomeDefaultCDNBase+"/webawesome.loader.js" {
		t.Fatalf("webawesome_loader_url = %v", got)
	}
	if needs, ok := m.Posts()[0].Extra["needs_webawesome"].(bool); !ok || !needs {
		t.Fatalf("needs_webawesome = %v, want true", m.Posts()[0].Extra["needs_webawesome"])
	}
}

func TestWebAwesomePlugin_RenderUsesSharedVendorAssetURL(t *testing.T) {
	plugin := NewWebAwesomePlugin()
	m := lifecycle.NewManager()
	m.SetConfig(&lifecycle.Config{Extra: map[string]interface{}{
		"asset_urls": map[string]string{
			"webawesome": "/assets/vendor/webawesome",
		},
	}})
	m.SetPosts([]*models.Post{{ArticleHTML: `<wa-button>Click</wa-button>`}})

	if err := plugin.Render(m); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if got := m.Config().Extra["webawesome_css_url"]; got != "/assets/vendor/webawesome/styles/webawesome.css" {
		t.Fatalf("webawesome_css_url = %v", got)
	}
	if got := m.Config().Extra["webawesome_loader_url"]; got != "/assets/vendor/webawesome/webawesome.loader.js" {
		t.Fatalf("webawesome_loader_url = %v", got)
	}
}

func TestWebAwesomePlugin_RenderDoesNotEnableAssetsWithoutComponents(t *testing.T) {
	plugin := NewWebAwesomePlugin()
	m := lifecycle.NewManager()
	m.SetConfig(&lifecycle.Config{Extra: map[string]interface{}{}})
	m.SetPosts([]*models.Post{{ArticleHTML: `<p>No Web Awesome here.</p>`}})

	if err := plugin.Render(m); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if _, ok := m.Config().Extra["webawesome_enabled"]; ok {
		t.Fatalf("webawesome_enabled was set for a page without Web Awesome components")
	}
	if m.Posts()[0].Extra != nil {
		if _, ok := m.Posts()[0].Extra["needs_webawesome"]; ok {
			t.Fatalf("needs_webawesome was set for a page without Web Awesome components")
		}
	}
}

func TestWebAwesomePlugin_ComponentModulesDeduplicatesDetectedComponents(t *testing.T) {
	plugin := NewWebAwesomePlugin()
	plugin.config.Source = "cdn"
	modules := plugin.componentModules(`<wa-comparison></wa-comparison><wa-button></wa-button><wa-button></wa-button>`)
	want := []string{
		webAwesomeDefaultCDNBase + "/components/comparison/comparison.js",
		webAwesomeDefaultCDNBase + "/components/button/button.js",
	}
	if len(modules) != len(want) {
		t.Fatalf("modules = %#v, want %#v", modules, want)
	}
	for i := range want {
		if modules[i] != want[i] {
			t.Fatalf("modules = %#v, want %#v", modules, want)
		}
	}
}

func TestWebAwesomePlugin_DefaultsToVendorSource(t *testing.T) {
	plugin := NewWebAwesomePlugin()
	if plugin.config.Source != "vendor" {
		t.Fatalf("default source = %q, want %q", plugin.config.Source, "vendor")
	}
	if plugin.config.OutputDir != "assets/vendor/webawesome" {
		t.Fatalf("default output dir = %q, want %q", plugin.config.OutputDir, "assets/vendor/webawesome")
	}
}
