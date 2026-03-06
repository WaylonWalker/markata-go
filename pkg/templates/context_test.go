package templates

import (
	"reflect"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestPostToMap_InlinksOutlinks(t *testing.T) {
	sourceTitle := "Source"
	targetTitle := "Target"

	sourcePost := &models.Post{Slug: "source", Href: "/source/", Title: &sourceTitle}
	targetPost := &models.Post{Slug: "target", Href: "/target/", Title: &targetTitle}

	link := &models.Link{
		SourceURL:    "https://example.com/source/",
		SourcePost:   sourcePost,
		TargetPost:   targetPost,
		RawTarget:    "/target/",
		TargetURL:    "https://example.com/target/",
		TargetDomain: "example.com",
		IsInternal:   true,
		SourceText:   "Source",
		TargetText:   "Target",
	}

	post := &models.Post{
		Slug:     "target",
		Href:     "/target/",
		Hrefs:    []string{"/target/"},
		Inlinks:  []*models.Link{link},
		Outlinks: []*models.Link{link},
	}

	mapped := postToMapUncached(post)
	if mapped["hrefs"] == nil {
		t.Fatalf("expected hrefs to be set")
	}

	if !reflect.DeepEqual(mapped["hrefs"], []string{"/target/"}) {
		t.Errorf("unexpected hrefs: %#v", mapped["hrefs"])
	}

	inlinks, ok := mapped["inlinks"].([]map[string]interface{})
	if !ok || len(inlinks) != 1 {
		t.Fatalf("expected inlinks map slice, got %#v", mapped["inlinks"])
	}

	linkMap := inlinks[0]
	if linkMap["source_url"] != "https://example.com/source/" {
		t.Errorf("unexpected source_url: %#v", linkMap["source_url"])
	}
	if linkMap["target_domain"] != "example.com" {
		t.Errorf("unexpected target_domain: %#v", linkMap["target_domain"])
	}
	if linkMap["is_internal"] != true {
		t.Errorf("unexpected is_internal: %#v", linkMap["is_internal"])
	}

	sourceMap, ok := linkMap["source_post"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected source_post map, got %#v", linkMap["source_post"])
	}
	if sourceMap["href"] != "/source/" {
		t.Errorf("unexpected source_post href: %#v", sourceMap["href"])
	}
	if sourceMap["title"] != "Source" {
		t.Errorf("unexpected source_post title: %#v", sourceMap["title"])
	}
}

func TestSwitcherToMap_Defaults(t *testing.T) {
	m := SwitcherToMap(nil)

	if got, ok := m["enabled"].(bool); !ok || got {
		t.Fatalf("enabled = %#v, want false", m["enabled"])
	}
	if got, ok := m["mode_toggle"].(bool); !ok || !got {
		t.Fatalf("mode_toggle = %#v, want true", m["mode_toggle"])
	}
}

func TestSwitcherToMap_ModeToggleFromConfig(t *testing.T) {
	modeToggle := false
	enabled := true

	m := SwitcherToMap(&models.ThemeSwitcherConfig{
		Enabled:    &enabled,
		ModeToggle: &modeToggle,
	})

	if got, ok := m["enabled"].(bool); !ok || !got {
		t.Fatalf("enabled = %#v, want true", m["enabled"])
	}
	if got, ok := m["mode_toggle"].(bool); !ok || got {
		t.Fatalf("mode_toggle = %#v, want false", m["mode_toggle"])
	}
}

func TestComponentsToMap_PostConnectionsDefaults(t *testing.T) {
	components := models.NewComponentsConfig()
	m := componentsToMap(&components)

	postConnections, ok := m["post_connections"].(map[string]interface{})
	if !ok {
		t.Fatalf("post_connections map missing: %#v", m["post_connections"])
	}

	if got, ok := postConnections["display_graph"].(bool); !ok || !got {
		t.Fatalf("display_graph = %#v, want true", postConnections["display_graph"])
	}
	if got, ok := postConnections["display_list"].(bool); !ok || got {
		t.Fatalf("display_list = %#v, want false", postConnections["display_list"])
	}
	if got, ok := postConnections["graph_min_links"].(int); !ok || got != 3 {
		t.Fatalf("graph_min_links = %#v, want 3", postConnections["graph_min_links"])
	}
}
