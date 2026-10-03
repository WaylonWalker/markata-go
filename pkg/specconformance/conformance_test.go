package specconformance

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const requirementsPath = "../../spec/conformance/requirements.yaml"

type manifest struct {
	Version      int           `yaml:"version"`
	Requirements []requirement `yaml:"requirements"`
}

type requirement struct {
	ID             string   `yaml:"id"`
	Level          string   `yaml:"level"`
	Scope          string   `yaml:"scope"`
	Summary        string   `yaml:"summary"`
	Spec           string   `yaml:"spec"`
	Implementation []string `yaml:"implementation"`
	Tests          []string `yaml:"tests"`
	Status         string   `yaml:"status"`
	Notes          string   `yaml:"notes"`
}

var requirementIDPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*-[0-9]{3}$`)

func TestConformanceManifest(t *testing.T) {
	data, err := os.ReadFile(requirementsPath)
	if err != nil {
		t.Fatalf("read conformance requirements: %v", err)
	}

	var m manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse conformance requirements: %v", err)
	}
	if m.Version != 1 {
		t.Fatalf("manifest version = %d, want 1", m.Version)
	}
	if len(m.Requirements) == 0 {
		t.Fatal("conformance manifest must contain at least one requirement")
	}

	seen := make(map[string]struct{}, len(m.Requirements))
	for i, req := range m.Requirements {
		req := req
		t.Run(req.ID, func(t *testing.T) {
			if !requirementIDPattern.MatchString(req.ID) {
				t.Errorf("requirement %d has invalid id %q; want DOMAIN-NNN", i, req.ID)
			}
			if _, ok := seen[req.ID]; ok {
				t.Errorf("duplicate requirement id %q", req.ID)
			}
			seen[req.ID] = struct{}{}

			checkEnum(t, "level", req.Level, "MUST", "SHOULD", "MAY")
			checkEnum(t, "scope", req.Scope, "portable", "markata-go")
			checkEnum(t, "status", req.Status, "implemented", "partial", "planned", "not-applicable")

			if req.Summary == "" {
				t.Error("summary is required")
			}
			if req.Spec == "" {
				t.Error("spec path is required")
			} else {
				checkRepoPath(t, req.Spec)
			}
			for _, path := range req.Implementation {
				checkRepoPath(t, path)
			}
			for _, path := range req.Tests {
				checkRepoPath(t, path)
			}

			if req.Status == "implemented" {
				if len(req.Implementation) == 0 {
					t.Error("implemented requirement must reference implementation")
				}
				if len(req.Tests) == 0 {
					t.Error("implemented requirement must reference tests")
				}
			}
		})
	}
}

func checkEnum(t *testing.T, field, value string, allowed ...string) {
	t.Helper()
	for _, candidate := range allowed {
		if value == candidate {
			return
		}
	}
	t.Errorf("%s = %q; want one of %v", field, value, allowed)
}

func checkRepoPath(t *testing.T, path string) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join(filepath.Dir(requirementsPath), "../.."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	candidate := filepath.Clean(filepath.Join(root, filepath.FromSlash(path)))
	rel, err := filepath.Rel(root, candidate)
	if err != nil || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Errorf("path %q escapes repository root", path)
		return
	}
	info, err := os.Stat(candidate)
	if err != nil {
		t.Errorf("referenced path %q: %v", path, err)
		return
	}
	if info.IsDir() {
		t.Errorf("referenced path %q is a directory; reference a concrete file", path)
	}
}

func TestRequirementIDFormatExamples(t *testing.T) {
	for _, tc := range []struct {
		id   string
		want bool
	}{
		{"CORE-001", true},
		{"PLUGIN-123", true},
		{"BUILD_DAG-007", true},
		{"core-001", false},
		{"CORE-1", false},
		{"CORE001", false},
	} {
		t.Run(fmt.Sprintf("%s_%t", tc.id, tc.want), func(t *testing.T) {
			if got := requirementIDPattern.MatchString(tc.id); got != tc.want {
				t.Fatalf("MatchString(%q) = %t, want %t", tc.id, got, tc.want)
			}
		})
	}
}
