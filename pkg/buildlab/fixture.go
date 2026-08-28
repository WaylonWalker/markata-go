package buildlab

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FixtureConfig controls the bounded, deterministic fixture generator.
type FixtureConfig struct {
	Seed               int64   `json:"seed"`
	Posts              int     `json:"posts"`
	Feeds              int     `json:"feeds"`
	Tags               int     `json:"tags"`
	Wikilinks          int     `json:"wikilinks"`
	Embeds             int     `json:"embeds"`
	WikilinkDensity    float64 `json:"wikilink_density,omitempty"`
	EmbedDensity       float64 `json:"embed_density,omitempty"`
	DependencyDepth    int     `json:"dependency_depth"`
	Assets             int     `json:"assets"`
	TemplateVariations int     `json:"template_variations"`
}
type FixtureState struct {
	Seed       int64         `json:"seed"`
	Config     FixtureConfig `json:"config"`
	Operations []Operation   `json:"operations"`
}

// Apply applies the generated mutation sequence to root.
func (s FixtureState) Apply(root string) error {
	return (Scenario{ID: "generated", Version: "1", Seed: s.Seed, Operations: s.Operations}).Apply(root)
}

// GenerateFixture creates a small site fixture and returns the state used to
// create it. The same seed and parameters produce identical bytes.
//
//nolint:gocyclo // The fixture schema maps each independently configurable section to files.
func GenerateFixture(root string, cfg FixtureConfig) (FixtureState, error) {
	if cfg.Posts < 0 || cfg.Feeds < 0 || cfg.Tags < 0 || cfg.Wikilinks < 0 || cfg.Embeds < 0 || cfg.WikilinkDensity < 0 || cfg.WikilinkDensity > 1 || cfg.EmbedDensity < 0 || cfg.EmbedDensity > 1 || cfg.DependencyDepth < 0 || cfg.Assets < 0 || cfg.TemplateVariations < 0 {
		return FixtureState{}, fmt.Errorf("fixture counts must be non-negative")
	}
	//nolint:gosec // A seeded PRNG is required for reproducible fixture bytes.
	r := rand.New(rand.NewSource(cfg.Seed))
	state := FixtureState{Seed: cfg.Seed, Config: cfg}
	write := func(name, data string) error {
		p, e := safePath(root, name)
		if e == nil {
			e = os.MkdirAll(filepath.Dir(p), 0o755)
		}
		if e == nil {
			e = os.WriteFile(p, []byte(data), 0o600)
		}
		return e
	}
	for i := 0; i < cfg.Posts; i++ {
		links := ""
		linkCount := cfg.Wikilinks
		if linkCount == 0 && cfg.WikilinkDensity > 0 && r.Float64() < cfg.WikilinkDensity {
			linkCount = 1
		}
		for j := 0; j < linkCount; j++ {
			links += fmt.Sprintf(" [[post-%d]]", (i+j+1)%maxInt(1, cfg.Posts))
		}
		embeds := ""
		embedCount := cfg.Embeds
		if embedCount == 0 && cfg.EmbedDensity > 0 && r.Float64() < cfg.EmbedDensity {
			embedCount = 1
		}
		for j := 0; j < embedCount; j++ {
			target := (i + j + 1) % maxInt(1, cfg.Posts)
			embeds += fmt.Sprintf("\n![[post-%d]]", target)
		}
		if e := write(fmt.Sprintf("content/post-%03d.md", i), fmt.Sprintf("---\ntitle: Post %d\n---\n# Post %d\n%s%s\n", i, i, links, embeds)); e != nil {
			return state, e
		}
	}
	if e := write("markata-go.toml", generatedConfig(cfg)); e != nil {
		return state, e
	}
	for i := 0; i < cfg.Feeds; i++ {
		if e := write(fmt.Sprintf("feeds/feed-%03d.toml", i), fmt.Sprintf("name = \"feed-%d\"\n", i)); e != nil {
			return state, e
		}
	}
	for i := 0; i < cfg.Tags; i++ {
		if e := write(fmt.Sprintf("tags/tag-%03d.txt", i), fmt.Sprintf("tag-%d\n", i)); e != nil {
			return state, e
		}
	}
	for i := 0; i < cfg.Assets; i++ {
		if e := write(fmt.Sprintf("assets/asset-%03d.bin", i), strconv.Itoa(r.Int())); e != nil {
			return state, e
		}
	}
	for i := 0; i < cfg.DependencyDepth; i++ {
		if e := write(fmt.Sprintf("templates/part-%03d.html", i), fmt.Sprintf("part-%d\n", i)); e != nil {
			return state, e
		}
	}
	for i := 0; i < cfg.TemplateVariations; i++ {
		if e := write(fmt.Sprintf("templates/variant-%03d.html", i), strings.Repeat("template\n", i+1)); e != nil {
			return state, e
		}
	}
	return state, nil
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GeneratedMutationScenario returns a deterministic stateful sequence. It
// includes a priming build and one incremental checkpoint after each mutation.
// Each operation is safe to apply in order to a fixture produced by
// GenerateFixture.
func GeneratedMutationScenario(cfg FixtureConfig, count int) FixtureState {
	//nolint:gosec // A seeded PRNG is required for reproducible mutations.
	r := rand.New(rand.NewSource(cfg.Seed))
	s := FixtureState{Seed: cfg.Seed, Config: cfg}
	if cfg.Posts <= 0 || count <= 0 {
		return s
	}
	s.Operations = append(s.Operations, Operation{Type: OpBuild})
	current := make([]string, cfg.Posts)
	for i := range current {
		current[i] = fmt.Sprintf("# Post %d", i)
	}
	for i := 0; i < count; i++ {
		n := r.Intn(cfg.Posts)
		p := fmt.Sprintf("content/post-%03d.md", n)
		old := current[n]
		newValue := fmt.Sprintf("# Post %d mutation-%d", n, i)
		s.Operations = append(s.Operations, Operation{Type: OpReplaceExact, Path: p, Old: old, New: newValue})
		current[n] = newValue
		s.Operations = append(s.Operations, Operation{Type: OpBuild})
	}
	return s
}

func generatedConfig(_ FixtureConfig) string {
	return `[markata-go]
title = "Generated Build Lab Fixture"
license = false
output_dir = "output"

[markata-go.glob]
patterns = ["content/**/*.md"]
use_gitignore = false

[markata-go.search]
enabled = false

[markata-go.assets]
mode = "cdn"

[markata-go.tailwind]
include = false
build = false
`
}

// GenerateMutationScenario is the descriptive-name alias for
// GeneratedMutationScenario.
func GenerateMutationScenario(cfg FixtureConfig, count int) FixtureState {
	return GeneratedMutationScenario(cfg, count)
}

// CopyFixture recursively copies regular files and symlinks without following
// links. Existing destination content is replaced, but source is never changed.
func CopyFixture(src, dst string) error {
	if filepath.Clean(src) == filepath.Clean(dst) {
		return fmt.Errorf("source and destination are identical")
	}
	return copyDir(src, dst, nil)
}

func copyDir(src, dst string, excluded []string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, e := filepath.Rel(src, path)
		if e != nil {
			return e
		}
		if rel == "." {
			return os.MkdirAll(dst, 0o755)
		}
		if shouldExcludeFixturePath(rel, excluded) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.Mode()&os.ModeSymlink != 0 {
			v, e := os.Readlink(path)
			if e == nil {
				e = validateFixtureSymlink(src, path, v)
			}
			if e == nil {
				v, e = rewriteFixtureSymlink(src, path, dst, rel, v)
			}
			if e == nil {
				e = os.MkdirAll(filepath.Dir(target), 0o755)
			}
			if e == nil {
				e = os.Symlink(v, target)
			}
			return e
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		b, e := os.ReadFile(path)
		if e == nil {
			e = os.MkdirAll(filepath.Dir(target), 0o755)
		}
		if e == nil {
			e = os.WriteFile(target, b, info.Mode().Perm())
		}
		return e
	})
}

func shouldExcludeFixturePath(path string, excluded []string) bool {
	cleanPath := filepath.Clean(filepath.FromSlash(path))
	for _, excludedPath := range excluded {
		cleanExcluded := filepath.Clean(filepath.FromSlash(excludedPath))
		if cleanExcluded == "." || cleanExcluded == "" {
			continue
		}
		if cleanPath == cleanExcluded || strings.HasPrefix(cleanPath, cleanExcluded+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func validateFixtureSymlink(root, path, target string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	resolved, err := resolveFixtureSymlink(path, target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("fixture symlink %q escapes fixture", path)
	}
	return nil
}

func rewriteFixtureSymlink(src, path, dst, relativePath, target string) (string, error) {
	if !filepath.IsAbs(target) {
		return target, nil
	}
	root, err := filepath.Abs(src)
	if err != nil {
		return "", err
	}
	resolved, err := resolveFixtureSymlink(path, target)
	if err != nil {
		return "", err
	}
	relativeTarget, err := filepath.Rel(root, resolved)
	if err != nil || relativeTarget == ".." || strings.HasPrefix(relativeTarget, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("fixture symlink %q escapes fixture", path)
	}
	linkPath := filepath.Join(dst, relativePath)
	return filepath.Rel(filepath.Dir(linkPath), filepath.Join(dst, relativeTarget))
}

func resolveFixtureSymlink(path, target string) (string, error) {
	resolved := target
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(filepath.Dir(path), resolved)
	}
	return filepath.Abs(resolved)
}
