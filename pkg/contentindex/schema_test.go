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
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			data := readJSON(t, fixture)
			validateJSONSchema(t, schema, data)
			if _, err := Parse(data); err != nil {
				t.Fatalf("parser rejected schema-valid fixture: %v", err)
			}
		})
	}
	data, err := Marshal(Index{SchemaVersion: 1, Scope: "public", Generator: Generator{Name: GeneratorName, Version: "test"}, Documents: []Document{{Path: "post.md", Slug: "post", Href: "/post/", Published: true}}})
	if err != nil {
		t.Fatal(err)
	}
	validateJSONSchema(t, schema, data)
	if _, err := Parse(data); err != nil {
		t.Fatalf("parser rejected schema-valid generated artifact: %v", err)
	}
}

func TestV2JSONSchemaValidatesGeneratedPrivateArtifact(t *testing.T) {
	schema := loadSchema(t, "v2_schema.json")
	data, err := Marshal(Index{SchemaVersion: 2, Scope: PublicMetadataScope, Generator: Generator{Name: GeneratorName, Version: "test"}, Documents: []Document{{Path: "private.md", Slug: "private", Href: "/private/", Private: true}}})
	if err != nil {
		t.Fatal(err)
	}
	validateJSONSchema(t, schema, data)
	index, err := Parse(data)
	if err != nil {
		t.Fatalf("parser rejected generated v2 artifact: %v", err)
	}
	if index.SchemaVersion != 2 || index.Scope != PublicMetadataScope || len(index.Documents) != 1 || !index.Documents[0].Private {
		t.Fatalf("unexpected v2 artifact: %#v", index)
	}
}

func TestV2JSONSchemaValidatesFixtures(t *testing.T) {
	schema := loadSchema(t, "v2_schema.json")
	fixtures, err := filepath.Glob("fixtures/v2-*.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			data := readJSON(t, fixture)
			validateJSONSchema(t, schema, data)
			if _, err := Parse(data); err != nil {
				t.Fatalf("parser rejected schema-valid fixture: %v", err)
			}
		})
	}
}

func TestV2RejectsPrivateDocumentsInPublicScope(t *testing.T) {
	schema := loadSchema(t, "v2_schema.json")
	for _, scope := range []string{PublicScope, "workspace"} {
		t.Run(scope, func(t *testing.T) {
			data, err := Marshal(Index{SchemaVersion: 2, Scope: PublicMetadataScope, Generator: Generator{Name: GeneratorName, Version: "test"}, Documents: []Document{{Path: "private.md", Slug: "private", Href: "/private/", Private: true}}})
			if err != nil {
				t.Fatal(err)
			}
			data = []byte(strings.Replace(string(data), `"scope":"public-metadata"`, `"scope":"`+scope+`"`, 1))
			if err := schema.Validate(mustJSON(t, data)); err == nil {
				t.Fatalf("v2 schema accepted private documents in %q scope", scope)
			}
			if _, err := Parse(data); err == nil {
				t.Fatalf("v2 parser accepted private documents in %q scope", scope)
			}
		})
	}
}

func TestV2RejectsUnsupportedScope(t *testing.T) {
	schema := loadSchema(t, "v2_schema.json")
	data := v2TestArtifact(`"title":null`)
	data = []byte(strings.Replace(string(data), `"scope":"public-metadata"`, `"scope":"workspace"`, 1))
	if err := schema.Validate(mustJSON(t, data)); err == nil {
		t.Fatal("v2 schema accepted unsupported scope")
	}
	if _, err := Parse(data); err == nil {
		t.Fatal("v2 parser accepted unsupported scope")
	}
}

func TestV2SchemaAndParserRejectForbiddenPrivateMetadata(t *testing.T) {
	schema := loadSchema(t, "v2_schema.json")
	for _, field := range []string{"image", "video", "bio", "thumbnail", "cover", "og_image"} {
		t.Run(field, func(t *testing.T) {
			data := v2TestArtifact(`"` + field + `":"private value"`)
			if err := schema.Validate(mustJSON(t, data)); err == nil {
				t.Fatalf("v2 schema accepted private %s metadata", field)
			}
			if _, err := Parse(data); err == nil {
				t.Fatalf("v2 parser accepted private %s metadata", field)
			}

			data = v2TestArtifact(`"` + field + `":null`)
			if err := schema.Validate(mustJSON(t, data)); err == nil {
				t.Fatalf("v2 schema accepted null private %s metadata", field)
			}
			if _, err := Parse(data); err == nil {
				t.Fatalf("v2 parser accepted null private %s metadata", field)
			}
		})
	}
}

func TestV2SchemaAndParserAcceptNullableTextMetadata(t *testing.T) {
	schema := loadSchema(t, "v2_schema.json")
	for _, field := range []string{"title", "title_text", "description"} {
		t.Run(field, func(t *testing.T) {
			data := v2TestArtifact(`"` + field + `":null`)
			validateJSONSchema(t, schema, data)
			index, err := Parse(data)
			if err != nil {
				t.Fatalf("parser rejected nullable %s: %v", field, err)
			}
			document := index.Documents[0]
			switch field {
			case "title":
				if document.Title != nil {
					t.Fatalf("title = %v, want nil", document.Title)
				}
			case "title_text":
				if document.TitleText != nil {
					t.Fatalf("title_text = %v, want nil", document.TitleText)
				}
			case "description":
				if document.Description != nil {
					t.Fatalf("description = %v, want nil", document.Description)
				}
			}
		})
	}
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
		"document needs an identifier":     {`"slug":"post","href":"/post/"`, `"slug":"","href":""`},
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
			if _, err := Parse(data); err == nil {
				t.Fatal("parser accepted invalid artifact")
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

func TestV1JSONSchemaAndParserAcceptIntegralJSONNumbers(t *testing.T) {
	schema := loadV1Schema(t)
	data := []byte(`{"$schema":"markata://schemas/content-index/v1","schema":"markata.content-index","schema_version":1.0,"scope":"public","generator":{"name":"markata-go","version":"test"},"source":{},"document_count":1.0,"documents":[{"path":"post.md","slug":"post","href":"/post/","published":true,"draft":false,"private":false}]}`)
	validateJSONSchema(t, schema, data)
	if _, err := Parse(data); err != nil {
		t.Fatalf("parser rejected schema-valid integral numbers: %v", err)
	}
}

func TestV1JSONSchemaAndParserAcceptExponentIntegralNumbers(t *testing.T) {
	schema := loadV1Schema(t)
	data := []byte(`{"$schema":"markata://schemas/content-index/v1","schema":"markata.content-index","schema_version":1e0,"scope":"public","generator":{"name":"markata-go","version":"test"},"source":{},"document_count":1e0,"documents":[{"path":"post.md","slug":"post","href":"/post/","published":true,"draft":false,"private":false}]}`)
	validateJSONSchema(t, schema, data)
	if _, err := Parse(data); err != nil {
		t.Fatalf("parser rejected schema-valid exponent numbers: %v", err)
	}
}

func TestV1JSONSchemaAndParserRejectNonIntegralJSONNumbers(t *testing.T) {
	schema := loadV1Schema(t)
	data := []byte(`{"$schema":"markata://schemas/content-index/v1","schema":"markata.content-index","schema_version":1.1,"scope":"public","generator":{"name":"markata-go","version":"test"},"source":{},"document_count":1,"documents":[{"path":"post.md","slug":"post","href":"/post/","published":true,"draft":false,"private":false}]}`)
	if err := schema.Validate(mustJSON(t, data)); err == nil {
		t.Fatal("schema accepted a non-integral schema version")
	}
	if _, err := Parse(data); err == nil {
		t.Fatal("parser accepted a non-integral schema version")
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
	return loadSchema(t, "v1_schema.json")
}

func loadSchema(t *testing.T, filename string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	data := readJSON(t, filename)
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	expectedID := SchemaURL
	if filename == "v2_schema.json" {
		expectedID = V2SchemaURL
	}
	if document["$id"] != expectedID || document["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
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

func v2TestArtifact(documentFields string) []byte {
	return []byte(`{"$schema":"markata://schemas/content-index/v2","schema":"markata.content-index","schema_version":2,"scope":"public-metadata","generator":{"name":"markata-go","version":"test"},"source":{},"document_count":1,"documents":[{"path":"private.md","slug":"private","href":"/private/","published":true,"draft":false,"private":true,` + documentFields + `}]}`)
}
