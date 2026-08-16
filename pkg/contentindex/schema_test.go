package contentindex

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestV1JSONSchemaValidatesFixturesAndWriter(t *testing.T) {
	schema := loadV1Schema(t)
	fixtures, err := filepath.Glob("fixtures/v1-*.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(filepath.Base(fixture), func(t *testing.T) { validateJSONSchema(t, schema, readJSON(t, fixture)) })
	}
	data, err := Marshal(Index{Scope: "public", Generator: Generator{Name: GeneratorName, Version: "test"}, Documents: []Document{{Path: "post.md", Slug: "post", Href: "/post/", Published: true}}})
	if err != nil {
		t.Fatal(err)
	}
	validateJSONSchema(t, schema, data)
}

func TestV1JSONSchemaRejectsInvalidArtifacts(t *testing.T) {
	schema := loadV1Schema(t)
	valid := `{"$schema":"markata://schemas/content-index/v1","schema":"markata.content-index","schema_version":1,"scope":"public","generator":{"name":"markata-go","version":"test"},"source":{},"document_count":1,"documents":[{"path":"post.md","slug":"post","href":"/post/","published":true,"draft":false,"private":false}]}`
	for name, replacement := range map[string][2]string{
		"missing required top-level field": {`"scope":"public",`, ``},
		"missing generator name":           {`"name":"markata-go",`, ``},
		"missing generator version":        {`,"version":"test"`, ``},
		"missing document path":            {`"path":"post.md",`, ``},
		"missing document published":       {`"published":true,`, ``},
		"wrong schema identity":            {`"schema":"markata.content-index"`, `"schema":"other"`},
		"wrong schema version":             {`"schema_version":1`, `"schema_version":2`},
		"wrong scope type":                 {`"scope":"public"`, `"scope":false`},
		"invalid document type":            {`"published":true`, `"published":"yes"`},
		"invalid date":                     {`"href":"/post/"`, `"href":"/post/","date":"not-a-date"`},
		"invalid source type":              {`"source":{}`, `"source":{"dirty":"no"}`},
		"dirty without commit":             {`"source":{}`, `"source":{"dirty":true}`},
		"commit without dirty":             {`"source":{}`, `"source":{"commit":"abc"}`},
		"empty commit":                     {`"source":{}`, `"source":{"commit":"","dirty":false}`},
		"forbidden null":                   {`"private":false`, `"private":null`},
	} {
		t.Run(name, func(t *testing.T) {
			data := []byte(valid)
			data = []byte(strings.Replace(string(data), replacement[0], replacement[1], 1))
			if err := schema.Validate(mustJSON(t, data)); err == nil {
				t.Fatal("schema accepted invalid artifact")
			}
		})
	}
}

func TestV1JSONSchemaAllowsUnknownOptionalFields(t *testing.T) {
	schema := loadV1Schema(t)
	data := []byte(`{"$schema":"markata://schemas/content-index/v1","schema":"markata.content-index","schema_version":1,"scope":"future-scope","generator":{"name":"markata-go","version":"test","future":true},"source":{"future":true},"document_count":1,"documents":[{"path":"post.md","slug":"post","href":"/post/","published":true,"draft":false,"private":false,"future":true}]}`)
	validateJSONSchema(t, schema, data)
	if _, err := Parse(data); err != nil {
		t.Fatal(err)
	}
}

func TestV1JSONSchemaValidatesExternalArtifact(t *testing.T) {
	path := os.Getenv("CONTENT_INDEX_ARTIFACT")
	if path == "" {
		t.Skip("CONTENT_INDEX_ARTIFACT not set")
	}
	validateJSONSchema(t, loadV1Schema(t), readJSON(t, path))
}

func loadV1Schema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	data := readJSON(t, "v1_schema.json")
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document["$id"] != SchemaURL || document["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		t.Fatalf("published schema metadata is not canonical: %#v", document)
	}
	// The validator cannot resolve Markata's intentionally internal markata://
	// URI. Remove only identity metadata; the published validation keywords are
	// compiled and executed unchanged below.
	delete(document, "$id") // The library cannot resolve the internal markata:// URI; validation rules are unchanged.
	delete(document, "$schema")
	data, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	resource, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if err := compiler.AddResource("content-index-v1.json", resource); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("content-index-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

func validateJSONSchema(t *testing.T, schema *jsonschema.Schema, data []byte) {
	t.Helper()
	if err := schema.Validate(mustJSON(t, data)); err != nil {
		t.Fatal(err)
	}
}

func readJSON(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func mustJSON(t *testing.T, data []byte) any {
	t.Helper()
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
