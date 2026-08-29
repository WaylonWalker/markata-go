package contentindex

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMarshalParseRoundTrip(t *testing.T) {
	title := "Hello"
	when := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	dirty := false
	image, video, author, category := "https://example.test/image.webp", "https://example.test/video.mp4", "waylon", "notes"
	want := Index{Schema: Schema, SchemaVersion: 1, Scope: "public", Generator: Generator{Name: GeneratorName, Version: "test"}, Source: Source{Commit: "abc", Dirty: &dirty}, Documents: []Document{{Path: "posts/hello.md", Slug: "hello", Href: "/hello/", Title: &title, TitleText: &title, Date: &when, Published: true, Tags: []string{"go"}, Feeds: []string{"blog"}, Image: &image, Video: &video, Author: &author, Authors: []string{"waylon", "guest"}, Category: &category, Categories: []string{"notes", "writing"}}}, DocumentCount: 1}
	data, err := Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.DocumentCount != 1 || got.Scope != "public" || got.Documents[0].Path != want.Documents[0].Path || got.Source.Commit != "abc" || got.Source.Dirty == nil || *got.Source.Dirty || got.Documents[0].Date == nil || !got.Documents[0].Date.Equal(when) {
		t.Fatalf("round trip lost data: %#v", got)
	}
	if got.Documents[0].Image == nil || *got.Documents[0].Image != image || got.Documents[0].Video == nil || *got.Documents[0].Video != video || got.Documents[0].Author == nil || *got.Documents[0].Author != author || len(got.Documents[0].Authors) != 2 || got.Documents[0].Category == nil || *got.Documents[0].Category != category {
		t.Fatalf("optional media and author metadata was lost: %#v", got.Documents[0])
	}
	if !strings.Contains(string(data), `"$schema":"`+SchemaURL+`"`) {
		t.Fatalf("missing canonical schema identity: %s", data)
	}
}

func TestParseErrors(t *testing.T) {
	for name, input := range map[string]string{"malformed": "{", "missing schema": `{ "schema_version": 1 }`, "missing version": `{ "schema": "` + Schema + `" }`, "future": `{ "schema": "` + Schema + `", "schema_version": 999 }`, "wrong schema": `{ "schema": "other", "schema_version": 1 }`} {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(input))
			if err == nil {
				t.Fatal("expected error")
			}
			if name == "future" && !errors.Is(err, ErrUnsupportedVersion) {
				t.Errorf("error = %v", err)
			}
		})
	}
}

func TestParseIgnoresUnknownFields(t *testing.T) {
	data := []byte(`{"$schema":"markata://schemas/content-index/v1","schema":"markata.content-index","schema_version":1,"scope":"public","generator":{"name":"markata-go","version":"test"},"source":{},"document_count":1,"documents":[{"path":"x.md","slug":"x","href":"/x/","published":true,"draft":false,"private":false,"future_optional":true}]}`)
	if _, err := Parse(data); err != nil {
		t.Fatal(err)
	}
}

func TestParseRequiresCanonicalSchemaURL(t *testing.T) {
	data := []byte(`{"$schema":"other","schema":"markata.content-index","schema_version":1,"scope":"public","generator":{"name":"markata-go","version":"test"},"source":{},"document_count":0,"documents":[]}`)
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "$schema") {
		t.Fatalf("error = %v", err)
	}
}

func TestParseRequiresDocumentCount(t *testing.T) {
	data := []byte(`{"$schema":"markata://schemas/content-index/v1","schema":"markata.content-index","schema_version":1,"scope":"public","generator":{"name":"markata-go","version":"test"},"source":{},"documents":[]}`)
	if _, err := Parse(data); err == nil {
		t.Fatal("expected missing document_count error")
	}
}

func TestParseRejectsNullRequiredTypes(t *testing.T) {
	data := []byte(`{"$schema":"markata://schemas/content-index/v1","schema":"markata.content-index","schema_version":1,"scope":"public","generator":{"name":"markata-go","version":"test"},"source":{},"document_count":1,"documents":[{"path":"x.md","slug":"x","href":"/x/","published":null,"draft":false,"private":false}]}`)
	if _, err := Parse(data); err == nil {
		t.Fatal("expected invalid boolean error")
	}
}

func TestParseRejectsNullOptionalWireTypes(t *testing.T) {
	data := []byte(`{"$schema":"markata://schemas/content-index/v1","schema":"markata.content-index","schema_version":1,"scope":"public","generator":{"name":"markata-go","version":"test"},"source":{"commit":null},"document_count":1,"documents":[{"path":"x.md","slug":"x","href":"/x/","published":true,"draft":false,"private":false,"date":null}]}`)
	if _, err := Parse(data); err == nil {
		t.Fatal("expected invalid optional type error")
	}
}

func TestMarshalIsDeterministicAndNormalizesOrdering(t *testing.T) {
	a := Index{Generator: Generator{Name: GeneratorName, Version: "test"}, Documents: []Document{{Path: "z.md", Slug: "z", Href: "/z/", Tags: []string{"z", "a"}}, {Path: "a.md", Slug: "a", Href: "/a/", Feeds: []string{"z", "a"}}}}
	b := Index{Generator: a.Generator, Documents: []Document{{Path: "a.md", Slug: "a", Href: "/a/", Feeds: []string{"a", "z"}}, {Path: "z.md", Slug: "z", Href: "/z/", Tags: []string{"a", "z"}}}}
	first, err := Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("equivalent indexes differ:\n%s\n%s", first, second)
	}
}

func TestMarshalRejectsUnsupportedScope(t *testing.T) {
	_, err := Marshal(Index{Scope: "workspace", Generator: Generator{Name: GeneratorName, Version: "test"}})
	if err == nil {
		t.Fatal("expected unsupported scope error")
	}
}

func TestMarshalAcceptsPublicMetadataScope(t *testing.T) {
	data, err := Marshal(Index{SchemaVersion: 2, Scope: PublicMetadataScope, Generator: Generator{Name: GeneratorName, Version: "test"}, Documents: []Document{{Path: "private.md", Slug: "private", Href: "/private/", Private: true}}})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scope != PublicMetadataScope {
		t.Fatalf("scope = %q, want %q", parsed.Scope, PublicMetadataScope)
	}
}

func TestV1CompatibilityAllowsPrivateRecords(t *testing.T) {
	data, err := os.ReadFile("fixtures/v1-private-compat.json")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() rejected schema-valid v1 private record: %v", err)
	}
	if !parsed.Documents[0].Private {
		t.Fatal("Parse() lost v1 private record")
	}

	roundTrip, err := Marshal(parsed)
	if err != nil {
		t.Fatalf("Marshal() rejected schema-valid v1 private record: %v", err)
	}
	if _, err := Parse(roundTrip); err != nil {
		t.Fatalf("Parse() rejected v1 round trip: %v", err)
	}
	if !strings.Contains(string(roundTrip), `"private":true`) {
		t.Fatalf("Marshal() dropped the v1 private record: %s", roundTrip)
	}
}

func TestMarshalV2RejectsForbiddenPrivateMetadata(t *testing.T) {
	fields := []struct {
		name string
		set  func(*Document, *string)
	}{
		{name: "image", set: func(document *Document, value *string) { document.Image = value }},
		{name: "video", set: func(document *Document, value *string) { document.Video = value }},
		{name: "bio", set: func(document *Document, value *string) { document.Bio = value }},
		{name: "thumbnail", set: func(document *Document, value *string) { document.Thumbnail = value }},
		{name: "cover", set: func(document *Document, value *string) { document.Cover = value }},
		{name: "og_image", set: func(document *Document, value *string) { document.OGImage = value }},
	}

	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			value := "private value"
			document := Document{Path: "private.md", Slug: "private", Href: "/private/", Private: true}
			field.set(&document, &value)
			data, err := Marshal(Index{SchemaVersion: 2, Scope: PublicMetadataScope, Generator: Generator{Name: GeneratorName, Version: "test"}, Documents: []Document{document}})
			if err == nil || !strings.Contains(err.Error(), "documents[0]."+field.name) {
				t.Fatalf("Marshal() error = %v, want forbidden %s error", err, field.name)
			}
			if data != nil {
				t.Fatalf("Marshal() returned data on error: %s", data)
			}
		})
	}
}

func TestMarshalV2AllowsMediaOnPublicDocuments(t *testing.T) {
	value := "public value"
	document := Document{
		Path: "public.md", Slug: "public", Href: "/public/",
		Image: &value, Video: &value, Bio: &value, Thumbnail: &value,
		Cover: &value, OGImage: &value,
	}
	if _, err := Marshal(Index{SchemaVersion: 2, Scope: PublicScope, Generator: Generator{Name: GeneratorName, Version: "test"}, Documents: []Document{document}}); err != nil {
		t.Fatalf("Marshal() rejected public media metadata: %v", err)
	}
}

func TestMarshalRejectsDirtySourceWithoutCommit(t *testing.T) {
	dirty := true
	_, err := Marshal(Index{Scope: PublicScope, Generator: Generator{Name: GeneratorName, Version: "test"}, Source: Source{Dirty: &dirty}})
	if err == nil {
		t.Fatal("expected dirty source without commit error")
	}
}

func TestReleasedFixturesParse(t *testing.T) {
	fixtures, err := filepath.Glob("fixtures/v1-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) < 2 {
		t.Fatal("expected immutable v1 fixtures")
	}
	for _, fixture := range fixtures {
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			data, err := os.ReadFile(fixture)
			if err != nil {
				t.Fatal(err)
			}
			index, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if index.DocumentCount != len(index.Documents) {
				t.Fatalf("count mismatch")
			}
		})
	}
}

func TestV2FixturesParse(t *testing.T) {
	fixtures, err := filepath.Glob("fixtures/v2-*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("expected v2 fixtures")
	}
	for _, fixture := range fixtures {
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			data, err := os.ReadFile(fixture)
			if err != nil {
				t.Fatal(err)
			}
			index, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if index.SchemaVersion != 2 || index.DocumentCount != len(index.Documents) {
				t.Fatalf("invalid v2 fixture: %#v", index)
			}
		})
	}
}

func TestParseExternalArtifact(t *testing.T) {
	path := os.Getenv("CONTENT_INDEX_ARTIFACT")
	if path == "" {
		t.Skip("CONTENT_INDEX_ARTIFACT not set")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	index, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if index.DocumentCount == 0 {
		t.Fatal("expected real artifact to contain documents")
	}
}

func TestV1SchemaMetadataAndFixtureShape(t *testing.T) {
	data, err := os.ReadFile("v1_schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		ID       string   `json:"$id"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if schema.ID != SchemaURL {
		t.Fatalf("schema id = %q", schema.ID)
	}
	for _, field := range []string{"$schema", "schema", "schema_version", "generator", "source", "document_count", "documents"} {
		found := false
		for _, required := range schema.Required {
			if required == field {
				found = true
			}
		}
		if !found {
			t.Errorf("schema does not require %q", field)
		}
	}
	for _, fixture := range []string{"fixtures/v1-minimal.json", "fixtures/v1-full.json", "fixtures/v1-no-feeds.json", "fixtures/v1-private-draft.json"} {
		fixtureData, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]json.RawMessage
		if err := json.Unmarshal(fixtureData, &document); err != nil {
			t.Fatal(err)
		}
		for _, field := range schema.Required {
			if _, ok := document[field]; !ok {
				t.Errorf("%s lacks required field %q", fixture, field)
			}
		}
	}
}
