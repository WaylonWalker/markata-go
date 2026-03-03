package plugins

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestResolveIndexName(t *testing.T) {
	sc := models.NewSearchcraftConfig()
	sc.IndexPerSite = true
	if got := sc.ResolveIndexName("My Site! v1"); got != "markata_my-site-v1" {
		t.Fatalf("unexpected index value %q", got)
	}
	sc.IndexName = "Custom Index"
	if got := sc.ResolveIndexName("ignored"); got != "custom-index" {
		t.Fatalf("unexpected explicit index value %q", got)
	}
}

func TestSearchcraftBatchSizeDefault(t *testing.T) {
	sc := models.NewSearchcraftConfig()
	sc.BatchSize = 0
	if got := sc.BatchSizeOrDefault(); got != 100 {
		t.Fatalf("batch size default wrong: %d", got)
	}
}

func TestShouldIndexPost(t *testing.T) {
	sc := models.NewSearchcraftConfig()
	post := &models.Post{Published: true}
	if !shouldIndexPost(post, sc) {
		t.Fatal("expected published post to be indexed")
	}
	post.Private = true
	if shouldIndexPost(post, sc) {
		t.Fatal("expected private post to be skipped")
	}
	sc.IncludePrivate = true
	if !shouldIndexPost(post, sc) {
		t.Fatal("expected private post to be indexed when include_private")
	}
	post.Private = false
	post.Draft = true
	if shouldIndexPost(post, sc) {
		t.Fatal("expected draft to be skipped")
	}
	sc.IncludeDrafts = true
	if !shouldIndexPost(post, sc) {
		t.Fatal("expected draft to be indexed when include_drafts")
	}
}

func TestComputeDocumentHashDeterminism(t *testing.T) {
	doc := searchcraftDocument{ID: "foo", Title: "title", Summary: "sum"}
	hash1 := computeDocumentHash(doc)
	hash2 := computeDocumentHash(doc)
	if hash1 != hash2 {
		t.Fatalf("hash should be deterministic")
	}
}

func TestBuildSearchcraftCardHTML_NilInputs(t *testing.T) {
	if got := buildSearchcraftCardHTML(nil, models.NewConfig(), nil); got != "" {
		t.Fatalf("expected empty html for nil post, got %q", got)
	}
	post := &models.Post{Slug: "example", Href: "/example/"}
	if got := buildSearchcraftCardHTML(post, nil, nil); got != "" {
		t.Fatalf("expected empty html for nil config/engine, got %q", got)
	}
}

func TestSearchcraftIndexHasField(t *testing.T) {
	payload := []byte(`{"status":200,"data":{"fields":{"title":{"type":"text"},"card_html":{"type":"text"}}}}`)
	if !searchcraftIndexHasField(payload, "card_html") {
		t.Fatalf("expected card_html field to be detected")
	}
	if searchcraftIndexHasField(payload, "missing") {
		t.Fatalf("did not expect missing field to be detected")
	}
}
