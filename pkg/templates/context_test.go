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

func TestConfigToMap_IncludesLicense(t *testing.T) {
	cfg := &models.Config{
		License: models.LicenseValue{Raw: models.DefaultLicenseKey},
	}
	mapped := configToMap(cfg)
	license, ok := mapped["license"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected license map, got %T", mapped["license"])
	}
	if license["name"] != "Creative Commons Attribution 4.0" {
		t.Errorf("unexpected license name %q", license["name"])
	}
	if license["url"] == "" {
		t.Error("expected license url to be set")
	}
	if license["key"] != models.DefaultLicenseKey {
		t.Errorf("unexpected license key %q", license["key"])
	}
}
