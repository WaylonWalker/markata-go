package buildlab

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateOperationCharacterization(t *testing.T) {
	valid := []Operation{
		{Type: OpBuild}, {Type: OpCleanCache}, {Type: opLegacyClearCache}, {Type: OpClearOutput},
		{Type: OpWriteFile, Path: "x"},
		{Type: OpReplaceExact, Path: "x", Old: "old"},
		{Type: OpDelete, Path: "x"}, {Type: OpTouch, Path: "x"},
		{Type: OpRename, Path: "x", Dest: "y"}, {Type: OpCopy, Path: "x", Dest: "y"},
		{Type: OpSetConfig, Path: "x", Key: "title"},
	}
	for _, operation := range valid {
		t.Run(string(operation.Type)+" valid", func(t *testing.T) {
			if err := validateOperation(operation); err != nil {
				t.Fatalf("validateOperation(%+v) = %v", operation, err)
			}
		})
	}

	tests := []struct {
		name string
		op   Operation
		want string
	}{
		{"build payload", Operation{Type: OpBuild, Path: "x"}, "operation does not accept payload fields"},
		{"clean cache payload", Operation{Type: OpCleanCache, Value: "x"}, "operation does not accept payload fields"},
		{"clear output payload", Operation{Type: OpClearOutput, Content: "x"}, "operation does not accept payload fields"},
		{"write missing path", Operation{Type: OpWriteFile}, "path is required"},
		{"replace missing path", Operation{Type: OpReplaceExact, Old: "x"}, "path is required"},
		{"replace missing old", Operation{Type: OpReplaceExact, Path: "x"}, "old text is required"},
		{"delete missing path", Operation{Type: OpDelete}, "path is required"},
		{"touch missing path", Operation{Type: OpTouch}, "path is required"},
		{"rename missing path", Operation{Type: OpRename, Dest: "y"}, "path and dest are required"},
		{"rename missing dest", Operation{Type: OpRename, Path: "x"}, "path and dest are required"},
		{"copy missing path", Operation{Type: OpCopy, Dest: "y"}, "path and dest are required"},
		{"copy missing dest", Operation{Type: OpCopy, Path: "x"}, "path and dest are required"},
		{"config missing path", Operation{Type: OpSetConfig, Key: "title"}, "path is required"},
		{"config missing key", Operation{Type: OpSetConfig, Path: "x"}, "key is required and must be a single config key"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateOperation(test.op)
			if err == nil || err.Error() != test.want {
				t.Fatalf("validateOperation(%+v) = %v, want %q", test.op, err, test.want)
			}
		})
	}

	for _, key := range []string{"title\n", "title\r", "title=other", "title with-space", "title\tother"} {
		t.Run(fmt.Sprintf("config key %q", key), func(t *testing.T) {
			err := validateOperation(Operation{Type: OpSetConfig, Path: "x", Key: key})
			if err == nil || err.Error() != "key is required and must be a single config key" {
				t.Fatalf("validateOperation key %q = %v", key, err)
			}
		})
	}
}

func TestGenerateFixtureCharacterization(t *testing.T) {
	configs := []FixtureConfig{
		{Seed: 1},
		{Seed: 42, Posts: 3, Feeds: 2, Tags: 2, Wikilinks: 1, Embeds: 1, DependencyDepth: 2, Assets: 2, TemplateVariations: 2},
		{Seed: -7, Posts: 4, WikilinkDensity: .5, EmbedDensity: 1, Assets: 3},
		{Seed: 99, Posts: 2, Feeds: 1, Tags: 1, Wikilinks: 1, Embeds: 1, WikilinkDensity: .2, EmbedDensity: .8, DependencyDepth: 1, TemplateVariations: 3},
	}
	for i, cfg := range configs {
		t.Run(fmt.Sprintf("config-%d", i), func(t *testing.T) {
			first, second := t.TempDir(), t.TempDir()
			stateA, err := GenerateFixture(first, cfg)
			if err != nil {
				t.Fatal(err)
			}
			stateB, err := GenerateFixture(second, cfg)
			if err != nil {
				t.Fatal(err)
			}
			if fmt.Sprintf("%#v", stateA) != fmt.Sprintf("%#v", stateB) {
				t.Fatalf("states differ:\n%#v\n%#v", stateA, stateB)
			}
			assertTreeEqual(t, first, second)
		})
	}
}

func TestGenerateFixtureCharacterizationRejectsInvalidCounts(t *testing.T) {
	fields := []struct {
		name string
		set  func(*FixtureConfig)
	}{
		{"posts", func(c *FixtureConfig) { c.Posts = -1 }}, {"feeds", func(c *FixtureConfig) { c.Feeds = -1 }},
		{"tags", func(c *FixtureConfig) { c.Tags = -1 }}, {"wikilinks", func(c *FixtureConfig) { c.Wikilinks = -1 }},
		{"embeds", func(c *FixtureConfig) { c.Embeds = -1 }}, {"wikilinks below zero", func(c *FixtureConfig) { c.WikilinkDensity = -.01 }},
		{"wikilinks above one", func(c *FixtureConfig) { c.WikilinkDensity = 1.01 }}, {"embeds below zero", func(c *FixtureConfig) { c.EmbedDensity = -.01 }},
		{"embeds above one", func(c *FixtureConfig) { c.EmbedDensity = 1.01 }}, {"dependency depth", func(c *FixtureConfig) { c.DependencyDepth = -1 }},
		{"assets", func(c *FixtureConfig) { c.Assets = -1 }}, {"template variations", func(c *FixtureConfig) { c.TemplateVariations = -1 }},
	}
	for _, field := range fields {
		t.Run(field.name, func(t *testing.T) {
			cfg := FixtureConfig{}
			field.set(&cfg)
			_, err := GenerateFixture(t.TempDir(), cfg)
			if err == nil || err.Error() != "fixture counts must be non-negative" {
				t.Fatalf("GenerateFixture error = %v", err)
			}
		})
	}
}

func assertTreeEqual(t *testing.T, first, second string) {
	t.Helper()
	left, right := treeSnapshot(t, first), treeSnapshot(t, second)
	if len(left) != len(right) {
		t.Fatalf("tree entries differ: %d != %d\nleft=%v\nright=%v", len(left), len(right), left, right)
	}
	for path, data := range left {
		if !bytes.Equal(data, right[path]) {
			t.Fatalf("tree file %q differs", path)
		}
	}
}

func treeSnapshot(t *testing.T, root string) map[string][]byte {
	t.Helper()
	result := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestValidateOperationCharacterizationUnknownShape(t *testing.T) {
	if err := (Scenario{ID: "x", Version: "1", Operations: []Operation{{Type: OperationType("unknown")}}}).Validate(); err == nil || !strings.Contains(err.Error(), `unknown operation "unknown"`) {
		t.Fatalf("Validate unknown operation = %v", err)
	}
}
