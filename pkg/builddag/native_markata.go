package builddag

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

var (
	artifactSourceManifest = ArtifactID{Kind: "SourceManifest", Key: "site"}
	artifactParsedPosts    = ArtifactID{Kind: "ParsedPost", Key: "site"}
	artifactPostTitles     = ArtifactID{Kind: "PostMetadata", Key: "titles"}
	artifactPostIdentity   = ArtifactID{Kind: "PostIdentity", Key: "site"}
	artifactPostIndex      = ArtifactID{Kind: "PostIndex", Key: "site"}
	artifactWikilinks      = ArtifactID{Kind: "DynamicDependencies", Key: "wikilinks"}
	artifactArticleHTML    = ArtifactID{Kind: "ArticleHTML", Key: "site"}
)

const nativeRenderMarkdownPlugin = "render_markdown"

// MarkataSpine is the opt-in serial migration boundary for the first native
// Markata-Go slice.  The graph is compiled before execution and the manager is
// projected into the later legacy lifecycle only after the graph succeeds.
type MarkataSpine struct {
	Manager  *lifecycle.Manager
	Graph    *Graph
	Store    *MemoryStore
	State    *ExecutionState
	Executor *Executor
}

// NewMarkataSpine builds the first native graph for an already-configured
// manager.  Configuration and validation remain lifecycle-compatible setup;
// the graph owns glob/load/title/index/wikilink/Markdown ordering.
//
//nolint:gocyclo // The native spine is an explicit compatibility map of legacy hook order.
func NewMarkataSpine(m *lifecycle.Manager, seed int64, randomizeReady bool) (*MarkataSpine, error) {
	if m == nil {
		return nil, fmt.Errorf("builddag: manager is nil")
	}
	m.SetSerialBuild(true)
	m.SetConcurrency(1)
	if err := m.RunTo(lifecycle.StageValidate); err != nil {
		return nil, fmt.Errorf("native setup: %w", err)
	}

	builder := NewBuilder()
	add := func(task TaskSpec) { builder.AddTask(task) }

	add(nativeManagerTask(m, "native.source.glob", "glob", nil,
		[]ArtifactID{artifactSourceManifest}, func() error {
			return m.RunTo(lifecycle.StageGlob)
		}))
	add(nativeManagerTask(m, "native.source.load", "load", []ArtifactID{artifactSourceManifest},
		[]ArtifactID{artifactParsedPosts}, func() error {
			if err := m.RunTo(lifecycle.StageLoad); err != nil {
				return err
			}
			m.ResetPostDependencies()
			return nil
		}))

	autoTitle, err := findPlugin(m, "auto_title")
	if err != nil {
		return nil, err
	}
	inlineTitles, err := findPlugin(m, "inline_titles")
	if err != nil {
		return nil, err
	}
	wikilinks, err := findPlugin(m, "wikilinks")
	if err != nil {
		return nil, err
	}
	renderMarkdown, err := findPlugin(m, "render_markdown")
	if err != nil {
		return nil, err
	}

	// Preserve the old transform order around the migrated boundaries. Every
	// compatibility hook is an explicit exclusive node; the post index is
	// inserted immediately before wikilinks so that link resolution has a
	// declared input without dropping the metadata hooks that precede it.
	transformPlugins := lifecycle.SortPluginsByPriority(m.Plugins(), lifecycle.StageTransform)
	previous := artifactParsedPosts
	seenWikilinks := false
	for index, plugin := range transformPlugins {
		if _, ok := plugin.(lifecycle.TransformPlugin); !ok {
			continue
		}
		switch plugin.Name() {
		case "auto_title":
			add(nativePluginTask(m, autoTitle, lifecycle.StageTransform, "native.post.auto_title",
				[]ArtifactID{previous}, []ArtifactID{artifactPostTitles}))
			previous = artifactPostTitles
			continue
		case "inline_titles":
			add(nativePluginTask(m, inlineTitles, lifecycle.StageTransform, "native.post.inline_titles",
				[]ArtifactID{previous}, []ArtifactID{artifactPostIdentity}))
			previous = artifactPostIdentity
			continue
		case "wikilinks":
			add(nativeManagerTask(m, "native.post.index", "post-index", []ArtifactID{previous},
				[]ArtifactID{artifactPostIndex}, func() error {
					m.PostIndex().Refresh(m)
					m.Cache().Set("native_post_index_finalized", true)
					return nil
				}))
			add(nativePluginTask(m, wikilinks, lifecycle.StageTransform, "native.post.wikilinks",
				[]ArtifactID{artifactPostIndex}, []ArtifactID{artifactWikilinks}))
			previous = artifactWikilinks
			seenWikilinks = true
			continue
		case "build_cache":
			completion := ArtifactID{Kind: "legacy-transform", Key: "legacy.transform.build_cache"}
			add(NewLegacyTask(m, plugin, lifecycle.StageTransform, "legacy.transform.build_cache", []ArtifactID{previous}, []ArtifactID{completion}))
			previous = completion
			continue
		}
		if plugin.Name() == "" {
			continue
		}
		id := TaskID(fmt.Sprintf("legacy.transform.after.%03d.%s", index, plugin.Name()))
		if !seenWikilinks {
			id = TaskID(fmt.Sprintf("legacy.transform.%03d.%s", index, plugin.Name()))
		}
		completion := ArtifactID{Kind: "legacy-transform", Key: string(id)}
		add(NewLegacyTask(m, plugin, lifecycle.StageTransform, id, []ArtifactID{previous}, []ArtifactID{completion}))
		previous = completion
	}
	if !seenWikilinks {
		return nil, fmt.Errorf("native graph requires wikilinks transform boundary")
	}
	// Legacy builds clear the global template projection cache at the render
	// stage boundary.  Keep that boundary explicit here because the graph runs
	// transform and render tasks without calling Manager.RunTo between them.
	artifactRenderBoundary := ArtifactID{Kind: "RenderBoundary", Key: "site"}
	add(nativeCacheBoundaryTask("native.render.prepare", previous, artifactRenderBoundary))
	previous = artifactRenderBoundary
	// Replace render_markdown at its original priority position. Render hooks
	// before it are retained as compatibility nodes as well.
	seenRender := false
	for index, plugin := range lifecycle.SortPluginsByPriority(m.Plugins(), lifecycle.StageRender) {
		if _, ok := plugin.(lifecycle.RenderPlugin); !ok {
			continue
		}
		if plugin.Name() == nativeRenderMarkdownPlugin {
			add(nativePluginTask(m, renderMarkdown, lifecycle.StageRender, "native.render.markdown",
				[]ArtifactID{previous}, []ArtifactID{artifactArticleHTML}))
			previous = artifactArticleHTML
			seenRender = true
			continue
		}
		idPrefix := "legacy.render.after"
		if !seenRender {
			idPrefix = "legacy.render.before"
		}
		id := TaskID(fmt.Sprintf("%s.%03d.%s", idPrefix, index, plugin.Name()))
		completion := ArtifactID{Kind: "legacy-render", Key: string(id)}
		add(NewLegacyTask(m, plugin, lifecycle.StageRender, id, []ArtifactID{previous}, []ArtifactID{completion}))
		previous = completion
	}
	if !seenRender {
		return nil, fmt.Errorf("native graph requires render_markdown render boundary")
	}

	graph, err := builder.Compile()
	if err != nil {
		return nil, fmt.Errorf("compile native graph: %w", err)
	}
	executor, err := NewExecutor(1)
	if err != nil {
		return nil, err
	}
	executor.Seed = seed
	executor.RandomizeReady = randomizeReady
	return &MarkataSpine{Manager: m, Graph: graph, Store: NewMemoryStore(), State: &ExecutionState{SchemaVersion: 1, Tasks: make(map[TaskID]TaskState)}, Executor: executor}, nil
}

func nativeCacheBoundaryTask(id TaskID, requires, provides ArtifactID) TaskSpec {
	return TaskSpec{
		ID: id, Group: string(lifecycle.StageRender), Requires: []ArtifactID{requires},
		Provides: []ArtifactID{provides}, Scope: ScopeSite,
		Version: "markata-spine-v1", Exclusive: true,
		Func: func(ctx context.Context, _ TaskContext) (TaskResult, error) {
			if err := ctx.Err(); err != nil {
				return TaskResult{}, err
			}
			templates.ClearAllCaches()
			return markerResult([]ArtifactID{provides}), nil
		},
	}
}

// Run executes the compiled spine serially, then resumes the legacy collect,
// write, and cleanup lifecycle.  The migrated transform and render stages are
// marked complete only after every graph task succeeds.
func (s *MarkataSpine) Run(ctx context.Context) error {
	if s == nil || s.Manager == nil || s.Graph == nil || s.Executor == nil {
		return fmt.Errorf("builddag: incomplete Markata spine")
	}
	if _, err := s.Executor.Execute(ctx, s.Graph, s.Store, s.State); err != nil {
		return err
	}
	if err := s.Manager.MarkStageComplete(lifecycle.StageTransform); err != nil {
		return err
	}
	if err := s.Manager.MarkStageComplete(lifecycle.StageRender); err != nil {
		return err
	}
	return s.Manager.RunTo(lifecycle.StageCleanup)
}

func nativeManagerTask(m *lifecycle.Manager, id TaskID, group string, requires, provides []ArtifactID, run func() error) TaskSpec {
	return TaskSpec{
		ID: id, Group: group, Requires: requires, Provides: provides, Scope: ScopeSite,
		Version: "markata-spine-v1", Exclusive: true,
		// This is a serial compatibility adapter. It uses the legacy manager as
		// its execution state and snapshots the declared projection afterward.
		Func: func(ctx context.Context, _ TaskContext) (TaskResult, error) {
			if err := ctx.Err(); err != nil {
				return TaskResult{}, err
			}
			if err := run(); err != nil {
				return TaskResult{}, err
			}
			result, err := managerResult(m, provides)
			if err != nil {
				return TaskResult{}, err
			}
			return result, nil
		},
	}
}

func nativePluginTask(m *lifecycle.Manager, plugin lifecycle.Plugin, stage lifecycle.Stage, id TaskID, requires, provides []ArtifactID) TaskSpec {
	return TaskSpec{
		ID: id, Group: string(stage), Requires: requires, Provides: provides, Scope: ScopeSite,
		Version: "markata-spine-v1", Exclusive: true,
		// This is a serial compatibility adapter. It invokes the legacy plugin
		// and snapshots manager state into the declared graph projection.
		Func: func(ctx context.Context, _ TaskContext) (TaskResult, error) {
			if err := ctx.Err(); err != nil {
				return TaskResult{}, err
			}
			if err := invokeLegacyHook(m, plugin, stage); err != nil {
				return TaskResult{}, err
			}
			result, err := managerResult(m, provides)
			if err != nil {
				return TaskResult{}, err
			}
			result.DynamicDeps = managerDynamicDependencies(m)
			return result, nil
		},
	}
}

func markerResult(ids []ArtifactID) TaskResult {
	result := TaskResult{Artifacts: make([]Artifact, 0, len(ids))}
	for _, id := range ids {
		result.Artifacts = append(result.Artifacts, Artifact{ID: id})
	}
	return result
}

func managerResult(m *lifecycle.Manager, ids []ArtifactID) (TaskResult, error) {
	data, err := snapshotManager(m)
	if err != nil {
		return TaskResult{}, err
	}
	result := TaskResult{Artifacts: make([]Artifact, 0, len(ids))}
	for _, id := range ids {
		result.Artifacts = append(result.Artifacts, Artifact{ID: id, Data: append([]byte(nil), data...)})
	}
	return result, nil
}

func managerDynamicDependencies(m *lifecycle.Manager) []ArtifactID {
	seen := make(map[ArtifactID]bool)
	for _, post := range m.Posts() {
		if post == nil {
			continue
		}
		for _, dependency := range post.Dependencies {
			if dependency == "" {
				continue
			}
			seen[ArtifactID{Kind: "post", Key: dependency}] = true
		}
	}
	result := make([]ArtifactID, 0, len(seen))
	for dependency := range seen {
		result = append(result, dependency)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}

func findPlugin(m *lifecycle.Manager, name string) (lifecycle.Plugin, error) {
	if plugin, ok := findPluginOptional(m, name); ok {
		return plugin, nil
	}
	return nil, fmt.Errorf("native graph requires plugin %q", name)
}

func findPluginOptional(m *lifecycle.Manager, name string) (lifecycle.Plugin, bool) {
	for _, plugin := range m.Plugins() {
		if plugin.Name() == name {
			return plugin, true
		}
	}
	return nil, false
}

// NativeSnapshot is a stable projection used for diagnostics and future
// artifact persistence.  It intentionally excludes mutable pointer graphs.
type NativeSnapshot struct {
	Files []string          `json:"files,omitempty"`
	Posts []NativePostState `json:"posts,omitempty"`
}

type NativePostState struct {
	Path        string `json:"path"`
	Slug        string `json:"slug"`
	Title       string `json:"title,omitempty"`
	Content     string `json:"content,omitempty"`
	ArticleHTML string `json:"article_html,omitempty"`
}

func snapshotManager(m *lifecycle.Manager) ([]byte, error) {
	base := m.Config().ContentDir
	snapshot := NativeSnapshot{}
	for _, file := range m.Files() {
		rel, err := filepath.Rel(base, file)
		if err != nil {
			rel = file
		}
		snapshot.Files = append(snapshot.Files, filepath.ToSlash(rel))
	}
	sort.Strings(snapshot.Files)
	for _, post := range m.Posts() {
		state := NativePostState{Path: post.Path, Slug: post.Slug, Content: post.Content, ArticleHTML: post.ArticleHTML}
		state.Title = post.PlainTitle()
		snapshot.Posts = append(snapshot.Posts, state)
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("marshal native snapshot: %w", err)
	}
	return data, nil
}
