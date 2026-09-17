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
			{Src: "/z.png", Embed: true, Uses: []Use{{Href: "/z/", Caption: "Z caption", Embed: true}}},
			{Src: "/a.png", AddedAt: parseTime(t, timeValue), LastUsedAt: parseTime(t, timeValue), Uses: []Use{
				{Href: "/z/"},
				{Href: "/a/"},
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
	if len(parsed.Images) != 2 || parsed.Images[0].Uses[0].Href != "/a/" {
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

func TestMarshal_EmitsStableEmptyImageFields(t *testing.T) {
	data, err := Marshal(Index{
		Generator: Generator{Name: GeneratorName, Version: "test"},
		Images:    []Image{{Src: "/unused.png"}},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	text := string(data)
	for _, field := range []string{`"alt":""`, `"mime_type":""`, `"uses":[]`} {
		if !strings.Contains(text, field) {
			t.Fatalf("Marshal() missing stable empty field %q: %s", field, data)
		}
	}
	if strings.Contains(text, `"post"`) {
		t.Fatalf("Marshal() emitted a repository-relative use field: %s", data)
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
	data := `{"$schema":"markata://schemas/image-index/v1","schema":"markata.image-index","schema_version":1,"generator":{"name":"markata-go","version":"test"},"image_count":1,"images":[{"src":"/a.png","width":1,"height":2,"alt":"","mime_type":"image/png","cover":false,"uses":[],"future_field":"ignored"}],"future_top_level":true}`
	index, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if index.Images[0].Src != "/a.png" {
		t.Fatalf("Parse() image = %#v", index.Images[0])
	}
}

func TestParse_RejectsMissingRequiredV1ImageFields(t *testing.T) {
	base := `{"$schema":"markata://schemas/image-index/v1","schema":"markata.image-index","schema_version":1,"generator":{"name":"markata-go","version":"test"},"image_count":1,"images":[{"src":"/a.png","width":1,"height":2,"alt":"","mime_type":"image/png","cover":false,"uses":[]}]}`
	for name, fragment := range map[string]string{
		"src":       `"src":"/a.png",`,
		"width":     `"width":1,`,
		"height":    `"height":2,`,
		"alt":       `"alt":"",`,
		"mime_type": `"mime_type":"image/png",`,
		"cover":     `"cover":false,`,
		"uses":      `,"uses":[]`,
	} {
		t.Run(name, func(t *testing.T) {
			data := strings.Replace(base, fragment, "", 1)
			_, err := Parse([]byte(data))
			if err == nil || !strings.Contains(err.Error(), "images[0]."+name) {
				t.Fatalf("Parse() error = %v, want missing images[0].%s", err, name)
			}
		})
	}
}

func TestParse_RejectsMissingRequiredV1TopLevelFields(t *testing.T) {
	base := `{"$schema":"markata://schemas/image-index/v1","schema":"markata.image-index","schema_version":1,"generator":{"name":"markata-go","version":"test"},"image_count":0,"images":[]}`
	for name, fragment := range map[string]string{
		"$schema":        `"$schema":"markata://schemas/image-index/v1",`,
		"schema":         `"schema":"markata.image-index",`,
		"schema_version": `"schema_version":1,`,
		"generator":      `"generator":{"name":"markata-go","version":"test"},`,
		"image_count":    `"image_count":0,`,
		"images":         `"images":[]`,
	} {
		t.Run(name, func(t *testing.T) {
			data := strings.Replace(base, fragment, "", 1)
			if _, err := Parse([]byte(data)); err == nil {
				t.Fatalf("Parse() accepted missing top-level field %q", name)
			}
		})
	}
}

func TestParse_RejectsInvalidRequiredV1FieldTypes(t *testing.T) {
	base := `{"$schema":"markata://schemas/image-index/v1","schema":"markata.image-index","schema_version":1,"generator":{"name":"markata-go","version":"test"},"image_count":1,"images":[{"src":"/a.png","width":1,"height":2,"alt":"","mime_type":"image/png","cover":false,"uses":[]}]}`
	for name, replacement := range map[string][2]string{
		"width":     {`"width":1`, `"width":"1"`},
		"alt":       {`"alt":""`, `"alt":false`},
		"mime_type": {`"mime_type":"image/png"`, `"mime_type":false`},
		"cover":     {`"cover":false`, `"cover":"false"`},
		"uses":      {`"uses":[]`, `"uses":{}`},
	} {
		t.Run(name, func(t *testing.T) {
			data := strings.Replace(base, replacement[0], replacement[1], 1)
			_, err := Parse([]byte(data))
			if err == nil || !strings.Contains(err.Error(), "images[0]."+name) {
				t.Fatalf("Parse() error = %v, want invalid images[0].%s", err, name)
			}
		})
	}
}

func TestParse_RejectsMissingRequiredV1UseFields(t *testing.T) {
	base := `{"$schema":"markata://schemas/image-index/v1","schema":"markata.image-index","schema_version":1,"generator":{"name":"markata-go","version":"test"},"image_count":1,"images":[{"src":"/a.png","width":1,"height":2,"alt":"","mime_type":"image/png","cover":false,"uses":[{"href":"/post/","cover":false}]}]}`
	for name, replacement := range map[string][2]string{
		"href":  {`"href":"/post/",`, ``},
		"cover": {`"uses":[{"href":"/post/","cover":false}]`, `"uses":[{"href":"/post/"}]`},
	} {
		t.Run(name, func(t *testing.T) {
			data := strings.Replace(base, replacement[0], replacement[1], 1)
			_, err := Parse([]byte(data))
			if err == nil || !strings.Contains(err.Error(), "images[0].uses[0]."+name) {
				t.Fatalf("Parse() error = %v, want missing images[0].uses[0].%s", err, name)
			}
		})
	}
}
