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
	want := Index{Schema: Schema, SchemaVersion: 1, Generator: Generator{Name: GeneratorName, Version: "test"}, Source: Source{Commit: "abc"}, Documents: []Document{{Path: "posts/hello.md", Slug: "hello", Href: "/hello/", Title: &title, TitleText: &title, Date: &when, Published: true, Tags: []string{"go"}, Feeds: []string{"blog"}}}, DocumentCount: 1}
	data, err := Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.DocumentCount != 1 || got.Documents[0].Path != want.Documents[0].Path || got.Source.Commit != "abc" || got.Documents[0].Date == nil || !got.Documents[0].Date.Equal(when) {
		t.Fatalf("round trip lost data: %#v", got)
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
	data := []byte(`{"$schema":"markata://schemas/content-index/v1","schema":"markata.content-index","schema_version":1,"generator":{"name":"markata-go","version":"test"},"source":{},"document_count":1,"documents":[{"path":"x.md","slug":"x","href":"/x/","published":true,"draft":false,"private":false,"future_optional":true}]}`)
	if _, err := Parse(data); err != nil {
		t.Fatal(err)
	}
}

func TestParseRequiresCanonicalSchemaURL(t *testing.T) {
	data := []byte(`{"$schema":"other","schema":"markata.content-index","schema_version":1,"generator":{"name":"markata-go","version":"test"},"source":{},"document_count":0,"documents":[]}`)
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "$schema") {
		t.Fatalf("error = %v", err)
	}
}

func TestParseRequiresDocumentCount(t *testing.T) {
	data := []byte(`{"$schema":"markata://schemas/content-index/v1","schema":"markata.content-index","schema_version":1,"generator":{"name":"markata-go","version":"test"},"source":{},"documents":[]}`)
	if _, err := Parse(data); err == nil {
		t.Fatal("expected missing document_count error")
	}
}

func TestParseRejectsNullRequiredTypes(t *testing.T) {
	data := []byte(`{"$schema":"markata://schemas/content-index/v1","schema":"markata.content-index","schema_version":1,"generator":{"name":"markata-go","version":"test"},"source":{},"document_count":1,"documents":[{"path":"x.md","slug":"x","href":"/x/","published":null,"draft":false,"private":false}]}`)
	if _, err := Parse(data); err == nil {
		t.Fatal("expected invalid boolean error")
	}
}

func TestParseRejectsNullOptionalWireTypes(t *testing.T) {
	data := []byte(`{"$schema":"markata://schemas/content-index/v1","schema":"markata.content-index","schema_version":1,"generator":{"name":"markata-go","version":"test"},"source":{"commit":null},"document_count":1,"documents":[{"path":"x.md","slug":"x","href":"/x/","published":true,"draft":false,"private":false,"date":null}]}`)
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
