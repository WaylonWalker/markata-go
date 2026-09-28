package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPlanBake_CharacterizationCorpus(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		input    string
		settings []BakeSetting
		want     string
	}{
		{
			name:     "toml existing nested table",
			file:     "markata-go.toml",
			input:    "# keep\n[markata-go]\ntitle = \"Old\"\n\n[markata-go.theme]\npalette = \"old\" # keep\n",
			settings: []BakeSetting{{Path: []string{"title"}, Value: "New"}, {Path: []string{"theme", "palette"}, Value: "nord"}, {Path: []string{"theme", "seasonal"}, Value: false}},
			want:     "# keep\n[markata-go]\ntitle = \"New\"\n\n[markata-go.theme]\npalette = \"nord\" # keep\nseasonal = false\n",
		},
		{
			name:     "yaml creates nested table",
			file:     "markata-go.yaml",
			input:    "markata-go:\n  title: Old\n",
			settings: []BakeSetting{{Path: []string{"theme", "palette"}, Value: "nord"}, {Path: []string{"theme", "switcher", "enabled"}, Value: true}},
			want:     "markata-go:\n  theme:\n    switcher:\n      enabled: true\n    palette: \"nord\"\n  title: Old\n",
		},
		{
			name:     "json preserves structure",
			file:     "markata-go.json",
			input:    "{\n  \"markata-go\": {\n    \"title\": \"Old\",\n    \"theme\": {\"palette\": \"old\"}\n  }\n}\n",
			settings: []BakeSetting{{Path: []string{"theme", "palette"}, Value: "nord"}, {Path: []string{"concurrency"}, Value: 4}},
			want:     "{\n  \"markata-go\": {\n    \"title\": \"Old\",\n    \"theme\": {\n      \"palette\": \"nord\"\n    },\n    \"concurrency\": 4\n  }\n}\n",
		},
		{
			name:     "new toml",
			file:     "markata-go.toml",
			settings: []BakeSetting{{Path: []string{"title"}, Value: "New"}},
			want:     "[markata-go]\ntitle = \"New\"\n",
		},
		{
			name:     "remove and absent remove",
			file:     "markata-go.yaml",
			input:    "markata-go:\n  title: Old\n  theme:\n    palette: old\n",
			settings: []BakeSetting{{Path: []string{"title"}, Value: BakeRemove}, {Path: []string{"theme", "palette"}, Value: BakeRemove}, {Path: []string{"missing"}, Value: BakeRemove}},
			want:     "markata-go:\n  theme:\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.file)
			if tt.input != "" {
				if err := os.WriteFile(path, []byte(tt.input), 0o640); err != nil {
					t.Fatal(err)
				}
			}
			before := []byte(tt.input)
			plan, err := PlanBake(path, tt.settings)
			if err != nil {
				t.Fatalf("PlanBake() error = %v", err)
			}
			if plan.Path != path || plan.Exists != (tt.input != "") || !bytes.Equal(plan.Before, before) || !bytes.Equal(plan.After, []byte(tt.want)) {
				t.Fatalf("PlanBake() = %+v, want exists=%t before=%q after=%q", plan, tt.input != "", before, tt.want)
			}
			if !plan.Changed() {
				t.Fatal("PlanBake() says expected change is unchanged")
			}
		})
	}
}

func TestPlanBake_CharacterizationErrors(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		input    string
		settings []BakeSetting
		want     string
		is       error
	}{
		{name: "no settings", file: "markata-go.toml", want: "no settings to bake"},
		{name: "empty path", file: "markata-go.toml", settings: []BakeSetting{{Value: "x"}}, want: "empty config key"},
		{name: "invalid key", file: "markata-go.toml", settings: []BakeSetting{{Path: []string{"bad key"}, Value: "x"}}, want: `invalid config key "bad key"`},
		{name: "unsupported value", file: "markata-go.toml", settings: []BakeSetting{{Path: []string{"title"}, Value: map[string]any{}}}, want: "title: unsupported value type map[string]interface {}"},
		{name: "invalid existing toml", file: "markata-go.toml", input: "[markata-go\n", settings: []BakeSetting{{Path: []string{"title"}, Value: "x"}}, want: "parse config file"},
		{name: "unsupported inline table", file: "markata-go.toml", input: "[markata-go]\ntheme = { palette = \"old\" }\n", settings: []BakeSetting{{Path: []string{"theme", "palette"}, Value: "x"}}, want: "config group layout cannot be edited safely", is: ErrBakeUnsupportedLayout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.file)
			if tt.input != "" {
				if err := os.WriteFile(path, []byte(tt.input), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			plan, err := PlanBake(path, tt.settings)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("PlanBake() error = %v, want substring %q", err, tt.want)
			}
			if tt.is != nil && !errors.Is(err, tt.is) {
				t.Fatalf("error = %v, want %v", err, tt.is)
			}
			if plan.Exists && !reflect.DeepEqual(plan.Before, []byte(tt.input)) {
				t.Fatalf("plan mutated Before: %+v", plan)
			}
			got, readErr := os.ReadFile(path)
			if readErr == nil && !bytes.Equal(got, []byte(tt.input)) {
				t.Fatalf("failed plan changed file: %q", got)
			}
		})
	}
}
