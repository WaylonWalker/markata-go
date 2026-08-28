package builddag

import (
	"context"
	"fmt"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

// NewLegacyTask adapts one lifecycle hook to a graph task.  Legacy hooks are
// conservative by design: they are exclusive and not marked parallel-safe
// because they can mutate Manager, Post, Config, or plugin-owned state.
// requires and provides are supplied by the graph compiler's caller so the
// adapter does not invent dependencies.
func NewLegacyTask(
	m *lifecycle.Manager,
	plugin lifecycle.Plugin,
	stage lifecycle.Stage,
	id TaskID,
	requires, provides []ArtifactID,
) TaskSpec {
	return TaskSpec{
		ID:        id,
		Group:     string(stage),
		Requires:  append([]ArtifactID(nil), requires...),
		Provides:  append([]ArtifactID(nil), provides...),
		Scope:     ScopeSite,
		Version:   "legacy-v1",
		Exclusive: true,
		Func: func(ctx context.Context, _ TaskContext) (TaskResult, error) {
			if err := ctx.Err(); err != nil {
				return TaskResult{}, err
			}
			if err := invokeLegacyHook(m, plugin, stage); err != nil {
				return TaskResult{}, err
			}
			artifacts := make([]Artifact, 0, len(provides))
			for _, artifactID := range provides {
				artifacts = append(artifacts, Artifact{ID: artifactID})
			}
			return TaskResult{Artifacts: artifacts, DynamicDeps: managerDynamicDependencies(m)}, nil
		},
	}
}

// LegacyTasks returns all hooks for stage in the exact order used by the
// lifecycle manager.  Completion artifacts form an explicit serial chain, so
// a future scheduler cannot reorder mutable legacy hooks accidentally.
func LegacyTasks(m *lifecycle.Manager, stage lifecycle.Stage) []TaskSpec {
	plugins := lifecycle.SortPluginsByPriority(m.Plugins(), stage)
	tasks := make([]TaskSpec, 0, len(plugins))
	var previous ArtifactID
	for index, plugin := range plugins {
		if !supportsStage(plugin, stage) {
			continue
		}
		id := TaskID(fmt.Sprintf("legacy.%s.%03d.%s", stage, index, plugin.Name()))
		completion := ArtifactID{Kind: "legacy-completion", Key: string(id)}
		var requires []ArtifactID
		if previous.Kind != "" {
			requires = []ArtifactID{previous}
		}
		tasks = append(tasks, NewLegacyTask(m, plugin, stage, id, requires, []ArtifactID{completion}))
		previous = completion
	}
	return tasks
}

func supportsStage(plugin lifecycle.Plugin, stage lifecycle.Stage) bool {
	switch stage {
	case lifecycle.StageConfigure:
		_, ok := plugin.(lifecycle.ConfigurePlugin)
		return ok
	case lifecycle.StageValidate:
		_, ok := plugin.(lifecycle.ValidatePlugin)
		return ok
	case lifecycle.StageGlob:
		_, ok := plugin.(lifecycle.GlobPlugin)
		return ok
	case lifecycle.StageLoad:
		_, ok := plugin.(lifecycle.LoadPlugin)
		return ok
	case lifecycle.StageTransform:
		_, ok := plugin.(lifecycle.TransformPlugin)
		return ok
	case lifecycle.StageRender:
		_, ok := plugin.(lifecycle.RenderPlugin)
		return ok
	case lifecycle.StageCollect:
		_, ok := plugin.(lifecycle.CollectPlugin)
		return ok
	case lifecycle.StageWrite:
		_, ok := plugin.(lifecycle.WritePlugin)
		return ok
	case lifecycle.StageCleanup:
		_, ok := plugin.(lifecycle.CleanupPlugin)
		return ok
	default:
		return false
	}
}

func invokeLegacyHook(m *lifecycle.Manager, plugin lifecycle.Plugin, stage lifecycle.Stage) error {
	return lifecycle.ExecutePluginHook(m, plugin, stage)
}
