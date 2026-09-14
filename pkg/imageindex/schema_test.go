package imageindex

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestV1SchemaValidatesGeneratedArtifact(t *testing.T) {
	schema := loadSchema(t)
	data, err := Marshal(Index{
		Generator: Generator{Name: GeneratorName, Version: "test"},
		Images: []Image{{
			Src:        "/images/photo.png",
			Width:      3,
			Height:     2,
			Alt:        "Photo",
			PosterSrc:  "/images/poster.webp",
			LastUsedAt: parseTime(t, "2026-01-15T12:00:00Z"),
			Cover:      true,
			Embed:      true,
			Uses:       []Use{{Post: "posts/photo.md", Href: "/photo/", Title: "Photo", Cover: true, Embed: true}},
		}},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := schema.Validate(mustJSON(t, data)); err != nil {
		t.Fatalf("generated artifact does not match schema: %v", err)
	}
}

func TestParseRejectsNullImagesArray(t *testing.T) {
	data := []byte(`{"$schema":"markata://schemas/image-index/v1","schema":"markata.image-index","schema_version":1,"generator":{"name":"markata-go","version":"test"},"image_count":0,"images":null}`)
	if _, err := Parse(data); err == nil {
		t.Fatal("Parse() accepted a null images array")
	}
}

func loadSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	data, err := os.ReadFile("v1_schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	delete(document, "$id")
	delete(document, "$schema")
	data, err = json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	resource, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	if err := compiler.AddResource("image-index-v1.json", resource); err != nil {
		t.Fatalf("add schema resource: %v", err)
	}
	schema, err := compiler.Compile("image-index-v1.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return schema
}

func mustJSON(t *testing.T, data []byte) any {
	t.Helper()
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	return value
}
