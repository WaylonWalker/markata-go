package imageindex

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMarshal_SortsImagesAndUses(t *testing.T) {
	timeValue := "2026-01-15T12:00:00Z"
	index := Index{
		Generator: Generator{Name: GeneratorName, Version: "test"},
		Images: []Image{
			{Src: "/z.png", Embed: true, Uses: []Use{{Post: "z.md", Href: "/z/", Caption: "Z caption", Embed: true}}},
			{Src: "/a.png", AddedAt: parseTime(t, timeValue), LastUsedAt: parseTime(t, timeValue), Uses: []Use{
				{Post: "z.md", Href: "/z/"},
				{Post: "a.md", Href: "/a/"},
			}},
		},
	}

	data, err := Marshal(index)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantOrder := []string{`"src":"/a.png"`, `"src":"/z.png"`}
	if first, second := bytes.Index(data, []byte(wantOrder[0])), bytes.Index(data, []byte(wantOrder[1])); first < 0 || second < 0 || first > second {
		t.Fatalf("Marshal() image order = %s", data)
	}
	if !strings.Contains(string(data), `"schema_version":1`) || !strings.Contains(string(data), `"image_count":2`) {
		t.Fatalf("Marshal() missing stable fields: %s", data)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(parsed.Images) != 2 || parsed.Images[0].Uses[0].Post != "a.md" {
		t.Fatalf("Parse() = %#v", parsed)
	}
	if !parsed.Images[1].Embed || !parsed.Images[1].Uses[0].Embed {
		t.Fatalf("Parse() lost embed metadata: %#v", parsed.Images[1])
	}
	if parsed.Images[1].Uses[0].Caption != "Z caption" {
		t.Fatalf("Parse() lost caption metadata: %#v", parsed.Images[1].Uses[0])
	}
	if parsed.Images[0].LastUsedAt == nil || parsed.Images[1].LastUsedAt != nil || parsed.Images[0].LastUsedAt.Format(time.RFC3339) != timeValue {
		t.Fatalf("Parse() lost last-used metadata: %#v", parsed.Images)
	}
}

func parseTime(t *testing.T, value string) *time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return &parsed
}

func TestParse_RejectsUnsupportedVersionAndCountMismatch(t *testing.T) {
	base := `{"$schema":"markata://schemas/image-index/v1","schema":"markata.image-index","schema_version":1,"generator":{"name":"markata-go","version":"test"},"image_count":0,"images":[]}`
	for name, data := range map[string]string{
		"version": strings.Replace(base, `"schema_version":1`, `"schema_version":2`, 1),
		"count":   strings.Replace(base, `"image_count":0`, `"image_count":1`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(data))
			if err == nil {
				t.Fatal("Parse() error = nil")
			}
			if name == "version" && !errors.Is(err, ErrUnsupportedVersion) {
				t.Fatalf("Parse() error = %v, want ErrUnsupportedVersion", err)
			}
		})
	}
}

func TestParse_IgnoresUnknownFields(t *testing.T) {
	data := `{"$schema":"markata://schemas/image-index/v1","schema":"markata.image-index","schema_version":1,"generator":{"name":"markata-go","version":"test"},"image_count":1,"images":[{"src":"/a.png","width":1,"height":2,"future_field":"ignored"}],"future_top_level":true}`
	index, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if index.Images[0].Src != "/a.png" {
		t.Fatalf("Parse() image = %#v", index.Images[0])
	}
}
