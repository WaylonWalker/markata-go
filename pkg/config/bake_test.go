package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var bakeTestValues = []BakeValue{
	{Key: "palette", Value: "nord-dark"},
	{Key: "seasonal", Value: false},
	{Key: "fontpack", Value: "st.-patrick's \"x\""},
}

func writeBakeFixture(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if content != "" {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func TestBakeValues_Formats(t *testing.T) {
	tests := []struct {
		name  string
		file  string
		input string
		want  string
	}{
		{
			name: "toml replaces in place and keeps comments",
			file: "markata-go.toml",
			input: `# site config
[markata-go]
title = "Site" # the title

[markata-go.theme]
# chosen palette
palette = "old" # keep me
aesthetic = "brutal"

[markata-go.theme.switcher]
enabled = true
`,
			want: `# site config
[markata-go]
title = "Site" # the title

[markata-go.theme]
# chosen palette
palette = "nord-dark" # keep me
aesthetic = "brutal"
seasonal = false
fontpack = "st.-patrick's \"x\""

[markata-go.theme.switcher]
enabled = true
`,
		},
		{
			name:  "toml appends missing table",
			file:  "markata-go.toml",
			input: "[markata-go]\ntitle = \"Site\"\n",
			want:  "[markata-go]\ntitle = \"Site\"\n\n[markata-go.theme]\npalette = \"nord-dark\"\nseasonal = false\nfontpack = \"st.-patrick's \\\"x\\\"\"\n",
		},
		{
			name:  "toml implicit group from subtable",
			file:  "markata-go.toml",
			input: "[markata-go.theme.switcher]\nenabled = true\n",
			want:  "[markata-go.theme.switcher]\nenabled = true\n\n[markata-go.theme]\npalette = \"nord-dark\"\nseasonal = false\nfontpack = \"st.-patrick's \\\"x\\\"\"\n",
		},
		{
			name:  "toml dotted keys",
			file:  "markata-go.toml",
			input: "[markata-go]\ntitle = \"Site\"\ntheme.palette = \"old\"\n\n[markata-go.feeds]\nx = 1\n",
			want:  "[markata-go]\ntitle = \"Site\"\ntheme.palette = \"nord-dark\"\ntheme.seasonal = false\ntheme.fontpack = \"st.-patrick's \\\"x\\\"\"\n\n[markata-go.feeds]\nx = 1\n",
		},
		{
			name:  "toml new file",
			file:  "markata-go.toml",
			input: "",
			want:  "[markata-go.theme]\npalette = \"nord-dark\"\nseasonal = false\nfontpack = \"st.-patrick's \\\"x\\\"\"\n",
		},
		{
			name: "yaml replaces and inserts",
			file: "markata-go.yaml",
			input: `markata-go:
    title: Site
    theme:
        palette: old # keep
        switcher:
            enabled: true

    output_dir: public
`,
			want: `markata-go:
    title: Site
    theme:
        seasonal: false
        fontpack: "st.-patrick's \"x\""
        palette: "nord-dark" # keep
        switcher:
            enabled: true

    output_dir: public
`,
		},
		{
			name:  "yaml adds group",
			file:  "markata-go.yml",
			input: "# c\nmarkata-go:\n  title: Site\n",
			want:  "# c\nmarkata-go:\n  theme:\n    palette: \"nord-dark\"\n    seasonal: false\n    fontpack: \"st.-patrick's \\\"x\\\"\"\n  title: Site\n",
		},
		{
			name:  "yaml empty group",
			file:  "markata-go.yaml",
			input: "markata-go:\n  theme:\n  title: Site\n",
			want:  "markata-go:\n  theme:\n    palette: \"nord-dark\"\n    seasonal: false\n    fontpack: \"st.-patrick's \\\"x\\\"\"\n  title: Site\n",
		},
		{
			name:  "json keeps key order",
			file:  "markata-go.json",
			input: "{\n    \"markata-go\": {\n        \"title\": \"Site\",\n        \"theme\": {\"palette\": \"old\", \"aesthetic\": \"brutal\"},\n        \"concurrency\": 4\n    }\n}\n",
			want:  "{\n    \"markata-go\": {\n        \"title\": \"Site\",\n        \"theme\": {\n            \"palette\": \"nord-dark\",\n            \"aesthetic\": \"brutal\",\n            \"seasonal\": false,\n            \"fontpack\": \"st.-patrick's \\\"x\\\"\"\n        },\n        \"concurrency\": 4\n    }\n}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeBakeFixture(t, tt.file, tt.input)
			if err := BakeValues(path, "theme", bakeTestValues); err != nil {
				t.Fatalf("BakeValues() error = %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("BakeValues() wrote:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestBakeValues_UnsupportedLayoutsLeaveFileUntouched(t *testing.T) {
	tests := []struct {
		name, file, input string
	}{
		{"toml inline table", "markata-go.toml", "[markata-go]\ntheme = { palette = \"old\" }\n"},
		{"toml multiline value", "markata-go.toml", "[markata-go.theme]\npalette = \"\"\"\nold\"\"\"\n"},
		{"yaml flow mapping", "markata-go.yaml", "markata-go:\n  theme: {palette: old}\n"},
		{"yaml block scalar", "markata-go.yaml", "markata-go:\n  theme:\n    palette: |\n      old\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeBakeFixture(t, tt.file, tt.input)
			err := BakeValues(path, "theme", []BakeValue{{Key: "palette", Value: "nord-dark"}})
			if !errors.Is(err, ErrBakeUnsupportedLayout) {
				t.Fatalf("error = %v, want ErrBakeUnsupportedLayout", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.input {
				t.Errorf("file changed after failed bake:\n%s", got)
			}
		})
	}
}

func TestBakeValues_RejectsInvalidKeys(t *testing.T) {
	path := writeBakeFixture(t, "markata-go.toml", "")
	if err := BakeValues(path, "theme", []BakeValue{{Key: "bad key", Value: "x"}}); err == nil {
		t.Fatal("expected error for invalid key")
	}
	if err := BakeValues(path, "theme", []BakeValue{{Key: "n", Value: map[string]any{}}}); err == nil {
		t.Fatal("expected error for unsupported value type")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("invalid bake should not create the file")
	}
}

func TestFindGroupConfigFile_PicksHighestPrecedenceFileWithGroup(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	root := write("markata-go.toml", "[markata-go]\ninclude = [\"config/*.toml\"]\n[markata-go.theme]\npalette = \"a\"\n")
	write("config/a-site.toml", "[markata-go]\ntitle = \"x\"\n")
	themeFile := write("config/b-theme.toml", "[markata-go.theme]\naesthetic = \"brutal\"\n")
	write("config/c-feeds.toml", "[markata-go.feed_defaults]\nitems_per_page = 3\n")

	paths, err := DiscoverIncludedConfigPaths(root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := FindGroupConfigFile(paths, "theme")
	if err != nil {
		t.Fatal(err)
	}
	if got != themeFile {
		t.Errorf("FindGroupConfigFile() = %q, want %q (paths %v)", got, themeFile, paths)
	}

	none, err := FindGroupConfigFile(paths, "blogroll")
	if err != nil || none != "" {
		t.Errorf("FindGroupConfigFile(blogroll) = %q, %v; want empty", none, err)
	}
	if !strings.HasSuffix(paths[0], "markata-go.toml") {
		t.Errorf("root config should come first: %v", paths)
	}
}

var bakeNestedSettings = []BakeSetting{
	{Path: []string{"title"}, Value: "New Title"},
	{Path: []string{"concurrency"}, Value: 8},
	{Path: []string{"theme", "switcher", "enabled"}, Value: false},
	{Path: []string{"hooks"}, Value: []string{"default", "extra"}},
	{Path: []string{"seo", "ratio"}, Value: 1.5},
}

func TestBakeSettings_NestedTypesAndLists(t *testing.T) {
	tests := []struct {
		name  string
		file  string
		input string
		want  string
	}{
		{
			name:  "toml tables and multi-line array",
			file:  "markata-go.toml",
			input: "# site\n[markata-go]\ntitle = \"Old\" # the title\nhooks = [\n  \"default\", # core\n]\n\n[markata-go.theme]\npalette = \"x\"\n",
			want: "# site\n[markata-go]\ntitle = \"New Title\" # the title\nhooks = [\"default\", \"extra\"]\nconcurrency = 8\n\n[markata-go.theme]\npalette = \"x\"\n\n" +
				"[markata-go.theme.switcher]\nenabled = false\n\n[markata-go.seo]\nratio = 1.5\n",
		},
		{
			name:  "toml dotted keys",
			file:  "markata-go.toml",
			input: "[markata-go]\ntitle = \"Old\"\ntheme.switcher.enabled = true\nhooks = [\"a\"]\n",
			want:  "[markata-go]\ntitle = \"New Title\"\ntheme.switcher.enabled = false\nhooks = [\"default\", \"extra\"]\nconcurrency = 8\n\n[markata-go.seo]\nratio = 1.5\n",
		},
		{
			name:  "toml new file",
			file:  "markata-go.toml",
			input: "",
			want:  "[markata-go]\ntitle = \"New Title\"\nconcurrency = 8\nhooks = [\"default\", \"extra\"]\n\n[markata-go.theme.switcher]\nenabled = false\n\n[markata-go.seo]\nratio = 1.5\n",
		},
		{
			name:  "yaml block sequence and nested mapping",
			file:  "markata-go.yaml",
			input: "markata-go:\n  title: Old # the title\n  hooks:\n    - default\n  theme:\n    palette: x\n",
			want:  "markata-go:\n  seo:\n    ratio: 1.5\n  concurrency: 8\n  title: \"New Title\" # the title\n  hooks: [\"default\", \"extra\"]\n  theme:\n    switcher:\n      enabled: false\n    palette: x\n",
		},
		{
			name:  "json",
			file:  "markata-go.json",
			input: "{\n  \"markata-go\": {\n    \"title\": \"Old\"\n  }\n}\n",
			want: "{\n  \"markata-go\": {\n    \"title\": \"New Title\",\n    \"concurrency\": 8,\n    \"hooks\": [\n      \"default\",\n      \"extra\"\n    ],\n" +
				"    \"theme\": {\n      \"switcher\": {\n        \"enabled\": false\n      }\n    },\n    \"seo\": {\n      \"ratio\": 1.5\n    }\n  }\n}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeBakeFixture(t, tt.file, tt.input)
			if err := BakeSettings(path, bakeNestedSettings); err != nil {
				t.Fatalf("BakeSettings() error = %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("BakeSettings() wrote:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestBakeSettings_RejectsNonTableAncestor(t *testing.T) {
	input := "[markata-go]\ntheme = { palette = \"x\" }\n"
	path := writeBakeFixture(t, "markata-go.toml", input)
	err := BakeSettings(path, []BakeSetting{{Path: []string{"theme", "switcher", "enabled"}, Value: true}})
	if !errors.Is(err, ErrBakeUnsupportedLayout) {
		t.Fatalf("error = %v, want ErrBakeUnsupportedLayout", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != input {
		t.Errorf("file changed:\n%s", got)
	}
}

func TestSettingDefinedDepth(t *testing.T) {
	raw := map[string]any{"markata-go": map[string]any{"theme": map[string]any{"palette": "x"}, "title": "t"}}
	tests := []struct {
		path []string
		want int
	}{
		{[]string{"theme", "palette"}, 2},
		{[]string{"theme", "switcher", "enabled"}, 1},
		{[]string{"title"}, 1},
		{[]string{"seo", "x"}, 0},
		{[]string{"title", "x"}, 1},
	}
	for _, tt := range tests {
		if got := SettingDefinedDepth(raw, tt.path); got != tt.want {
			t.Errorf("SettingDefinedDepth(%v) = %d, want %d", tt.path, got, tt.want)
		}
	}
}

func TestBakeSettings_RemovesKeys(t *testing.T) {
	remove := []BakeSetting{
		{Path: []string{"title"}, Value: BakeRemove},
		{Path: []string{"theme", "palette"}, Value: BakeRemove},
		{Path: []string{"seo", "missing"}, Value: BakeRemove},
	}
	tests := []struct {
		name  string
		file  string
		input string
		want  string
	}{
		{
			name:  "toml",
			file:  "markata-go.toml",
			input: "[markata-go]\ntitle = \"Old\" # the title\nlicense = false\n\n[markata-go.theme]\npalette = \"x\"\naesthetic = \"y\"\n",
			want:  "[markata-go]\nlicense = false\n\n[markata-go.theme]\naesthetic = \"y\"\n",
		},
		{
			name:  "yaml",
			file:  "markata-go.yaml",
			input: "markata-go:\n  title: Old\n  license: false\n  theme:\n    palette: x\n    aesthetic: y\n",
			want:  "markata-go:\n  license: false\n  theme:\n    aesthetic: y\n",
		},
		{
			name:  "json",
			file:  "markata-go.json",
			input: "{\n  \"markata-go\": {\n    \"title\": \"Old\",\n    \"license\": false,\n    \"theme\": {\n      \"palette\": \"x\",\n      \"aesthetic\": \"y\"\n    }\n  }\n}\n",
			want:  "{\n  \"markata-go\": {\n    \"license\": false,\n    \"theme\": {\n      \"aesthetic\": \"y\"\n    }\n  }\n}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeBakeFixture(t, tt.file, tt.input)
			if err := BakeSettings(path, remove); err != nil {
				t.Fatalf("BakeSettings() error = %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("BakeSettings() wrote:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestPlanBake_DoesNotWriteUntilAsked(t *testing.T) {
	input := "[markata-go]\ntitle = \"Old\"\n"
	path := writeBakeFixture(t, "markata-go.toml", input)
	plan, err := PlanBake(path, []BakeSetting{{Path: []string{"title"}, Value: "New"}})
	if err != nil {
		t.Fatalf("PlanBake() error = %v", err)
	}
	if !plan.Exists || !plan.Changed() || string(plan.Before) != input || !strings.Contains(string(plan.After), "title = \"New\"") {
		t.Fatalf("plan = %+v", plan)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != input {
		t.Fatalf("PlanBake wrote the file:\n%s", got)
	}
	if err := plan.Write(); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plan.After) {
		t.Fatalf("Write() wrote:\n%s", got)
	}

	noop, err := PlanBake(path, []BakeSetting{{Path: []string{"nav"}, Value: BakeRemove}})
	if err != nil || noop.Changed() {
		t.Fatalf("removing an absent key should be a no-op: %v %+v", err, noop)
	}
}
