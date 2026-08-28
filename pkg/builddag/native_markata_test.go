package builddag

import (
	"context"
	"errors"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

type spineSourcePlugin struct{}

func (spineSourcePlugin) Name() string                       { return "source" }
func (spineSourcePlugin) Configure(*lifecycle.Manager) error { return nil }
func (spineSourcePlugin) Validate(*lifecycle.Manager) error  { return nil }
func (spineSourcePlugin) Glob(m *lifecycle.Manager) error {
	m.SetFiles([]string{"post.md"})
	return nil
}
func (spineSourcePlugin) Load(m *lifecycle.Manager) error {
	m.SetPosts([]*models.Post{{Path: "post.md", Slug: "post"}})
	return nil
}

type spineTransformPlugin struct {
	name     string
	mark     string
	err      error
	priority int
}

func (p spineTransformPlugin) Name() string                 { return p.name }
func (p spineTransformPlugin) Priority(lifecycle.Stage) int { return p.priority }
func (p spineTransformPlugin) Transform(m *lifecycle.Manager) error {
	for _, post := range m.Posts() {
		post.Content += p.mark
		if p.name == "wikilinks" {
			post.AddDependency("target")
		}
	}
	return p.err
}

type spineRenderPlugin struct{}

func (spineRenderPlugin) Name() string { return "render_markdown" }
func (spineRenderPlugin) Render(m *lifecycle.Manager) error {
	for _, post := range m.Posts() {
		post.ArticleHTML = "<p>rendered</p>"
	}
	return nil
}

type spineTemplateCacheProbe struct{}

func (spineTemplateCacheProbe) Name() string                 { return "template_cache_probe" }
func (spineTemplateCacheProbe) Priority(lifecycle.Stage) int { return lifecycle.PriorityFirst - 2 }
func (spineTemplateCacheProbe) Transform(m *lifecycle.Manager) error {
	templates.PostsToMaps(m.Posts())
	return nil
}

type spineTemplateMetadataProbe struct{}

func (spineTemplateMetadataProbe) Name() string                 { return "template_metadata_probe" }
func (spineTemplateMetadataProbe) Priority(lifecycle.Stage) int { return lifecycle.PriorityFirst - 1 }
func (spineTemplateMetadataProbe) Transform(m *lifecycle.Manager) error {
	for _, post := range m.Posts() {
		post.Set("fresh_template_metadata", true)
	}
	return nil
}

type spineTemplateCacheRenderProbe struct {
	metadataVisible bool
}

func (p *spineTemplateCacheRenderProbe) Name() string { return "render_markdown" }
func (p *spineTemplateCacheRenderProbe) Render(m *lifecycle.Manager) error {
	posts := templates.PostsToMaps(m.Posts())
	p.metadataVisible = len(posts) == 1 && posts[0]["fresh_template_metadata"] == true
	return nil
}

type spineTailPlugin struct {
	collect int
	write   int
	cleanup int
}

func (p *spineTailPlugin) Name() string { return "tail" }
func (p *spineTailPlugin) Collect(*lifecycle.Manager) error {
	p.collect++
	return nil
}
func (p *spineTailPlugin) Write(*lifecycle.Manager) error {
	p.write++
	return nil
}
func (p *spineTailPlugin) Cleanup(*lifecycle.Manager) error {
	p.cleanup++
	return nil
}

func TestMarkataSpine_RunsNativeSliceThenLegacyTail(t *testing.T) {
	m := lifecycle.NewManager()
	tail := &spineTailPlugin{}
	m.RegisterPlugins(
		spineSourcePlugin{},
		spineTransformPlugin{name: "before_titles", mark: " before", priority: lifecycle.PriorityFirst - 1},
		spineTransformPlugin{name: "auto_title", mark: " title"},
		spineTransformPlugin{name: "inline_titles", mark: " inline"},
		spineTransformPlugin{name: "wikilinks", mark: " links"},
		spineRenderPlugin{},
		tail,
	)

	spine, err := NewMarkataSpine(m, 7, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := spine.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !m.HasRun(lifecycle.StageTransform) || !m.HasRun(lifecycle.StageRender) {
		t.Fatal("native stages were not marked complete")
	}
	if !m.HasRun(lifecycle.StageCollect) || !m.HasRun(lifecycle.StageWrite) || !m.HasRun(lifecycle.StageCleanup) {
		t.Fatal("legacy tail stages were not completed")
	}
	if tail.collect != 1 || tail.write != 1 || tail.cleanup != 1 {
		t.Fatalf("tail calls = collect %d, write %d, cleanup %d", tail.collect, tail.write, tail.cleanup)
	}
	post := m.Posts()[0]
	if post.Content != " before title inline links" || post.ArticleHTML != "<p>rendered</p>" {
		t.Fatalf("native post projection = %#v", post)
	}
	order := spine.Graph.Order()
	beforeIndex, autoIndex := indexOfTask(order, "legacy.transform.000.before_titles"), indexOfTask(order, "native.post.auto_title")
	if beforeIndex < 0 || autoIndex < 0 || beforeIndex > autoIndex {
		t.Fatalf("priority order = %v", order)
	}
	if len(spine.State.Tasks) != len(spine.Graph.Order()) {
		t.Fatalf("completed tasks = %d, graph tasks = %d", len(spine.State.Tasks), len(spine.Graph.Order()))
	}
	wikilinkState := spine.State.Tasks[TaskID("native.post.wikilinks")]
	if len(wikilinkState.Result.DynamicDeps) != 1 || wikilinkState.Result.DynamicDeps[0] != (ArtifactID{Kind: "post", Key: "target"}) {
		t.Fatalf("wikilink dynamic dependencies = %v", wikilinkState.Result.DynamicDeps)
	}
}

func TestMarkataSpine_ClearsTemplateProjectionCacheAtRenderBoundary(t *testing.T) {
	m := lifecycle.NewManager()
	render := &spineTemplateCacheRenderProbe{}
	m.RegisterPlugins(
		spineSourcePlugin{},
		spineTemplateCacheProbe{},
		spineTemplateMetadataProbe{},
		spineTransformPlugin{name: "auto_title"},
		spineTransformPlugin{name: "inline_titles"},
		spineTransformPlugin{name: "wikilinks"},
		render,
	)

	spine, err := NewMarkataSpine(m, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := spine.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !render.metadataVisible {
		t.Fatal("render task observed a stale template post projection")
	}
}

func indexOfTask(tasks []TaskID, want TaskID) int {
	for i, task := range tasks {
		if task == want {
			return i
		}
	}
	return -1
}

func TestLegacyTask_NonCriticalHookErrorBecomesWarning(t *testing.T) {
	m := lifecycle.NewManager()
	warning := spineTransformPlugin{name: "warning", err: errors.New("recoverable")}
	success := spineTransformPlugin{name: "success", mark: "done"}
	first := ArtifactID{Kind: "step", Key: "first"}
	second := ArtifactID{Kind: "step", Key: "second"}
	b := NewBuilder()
	b.AddTask(NewLegacyTask(m, warning, lifecycle.StageTransform, "warning", nil, []ArtifactID{first}))
	b.AddTask(NewLegacyTask(m, success, lifecycle.StageTransform, "success", []ArtifactID{first}, []ArtifactID{second}))
	g, err := b.Compile()
	if err != nil {
		t.Fatal(err)
	}
	state, err := (&Executor{MaxParallel: 1}).Execute(context.Background(), g, NewMemoryStore(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Tasks) != 2 {
		t.Fatalf("completed tasks = %d", len(state.Tasks))
	}
	warnings := m.Warnings()
	if len(warnings) != 1 || warnings[0].Plugin != "warning" || warnings[0].Stage != lifecycle.StageTransform {
		t.Fatalf("warnings = %+v", warnings)
	}
}
