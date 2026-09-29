package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/builddag"
	"github.com/WaylonWalker/markata-go/pkg/buildstats"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/templates"
)

var dagLifecycleStages = []lifecycle.Stage{
	lifecycle.StageConfigure,
	lifecycle.StageValidate,
	lifecycle.StageGlob,
	lifecycle.StageLoad,
	lifecycle.StageTransform,
	lifecycle.StageRender,
	lifecycle.StageCollect,
	lifecycle.StageWrite,
	lifecycle.StageCleanup,
}

// runDAGBuildObserved is the feature-flagged DAG equivalent of
// runBuildObserved. The observer contract is kept identical so serve can opt
// into this executor without losing stage visibility.
func runDAGBuildObserved(m *lifecycle.Manager, observe func(lifecycle.Stage, bool, error)) (result *BuildResult, err error) {
	if err := m.SetBuildExecutor(lifecycle.BuildExecutorDAG); err != nil {
		return nil, fmt.Errorf("select DAG build executor: %w", err)
	}

	profile := buildstats.Start()
	defer func() {
		summary := profile.Stop()
		if result != nil {
			result.Benchmark = summary
		}
	}()

	builder := builddag.NewBuilder()
	var requires []builddag.ArtifactID
	for _, lifecycleStage := range dagLifecycleStages {
		stage := lifecycleStage
		provided := builddag.ArtifactID{Kind: "lifecycle-stage", Key: string(stage)}

		if stage == lifecycle.StageTransform || stage == lifecycle.StageRender || stage == lifecycle.StageCollect {
			addDAGLegacyPluginStage(builder, m, stage, requires, provided, observe)
		} else {
			addDAGLifecycleStage(builder, m, stage, requires, provided, observe)
		}
		requires = []builddag.ArtifactID{provided}
	}

	graph, err := builder.Compile()
	if err != nil {
		return nil, fmt.Errorf("compile lifecycle DAG: %w", err)
	}
	digest, err := graph.Digest()
	if err != nil {
		return nil, fmt.Errorf("digest lifecycle DAG: %w", err)
	}
	executor, err := builddag.NewExecutor(1)
	if err != nil {
		return nil, err
	}
	execution, err := executor.Execute(context.Background(), graph)
	if err != nil {
		return nil, err
	}
	verbosef("  [dag] serial executor completed %d tasks (graph=%s)", execution.TaskCount, digest)

	result = &BuildResult{
		PostsProcessed: len(m.Posts()),
		FeedsGenerated: len(m.Feeds()),
		Content:        m.ContentDiagnostics(),
	}
	result.BlogrollStatus = getBlogrollStatus(m)
	for _, warning := range m.Warnings() {
		result.Warnings = append(result.Warnings, warning.Error())
	}
	return result, nil
}

func addDAGLifecycleStage(
	builder *builddag.Builder,
	m *lifecycle.Manager,
	stage lifecycle.Stage,
	requires []builddag.ArtifactID,
	provided builddag.ArtifactID,
	observe func(lifecycle.Stage, bool, error),
) {
	builder.AddTask(builddag.TaskSpec{
		ID:        builddag.TaskID("lifecycle." + string(stage)),
		Group:     string(stage),
		Requires:  append([]builddag.ArtifactID(nil), requires...),
		Provides:  []builddag.ArtifactID{provided},
		Scope:     builddag.ScopeSite,
		Version:   "lifecycle-v1",
		Exclusive: true,
		Func: func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return runDAGLifecycleStage(m, stage, observe)
		},
	})
}

// addDAGLegacyPluginStage expands one lifecycle stage into explicit plugin
// tasks while preserving the lifecycle's serial compatibility semantics. The
// boundary tasks keep stage-level timing and Serve observer behavior identical
// to the coarse executor.
func addDAGLegacyPluginStage(
	builder *builddag.Builder,
	m *lifecycle.Manager,
	stage lifecycle.Stage,
	requires []builddag.ArtifactID,
	provided builddag.ArtifactID,
	observe func(lifecycle.Stage, bool, error),
) {
	startedArtifact := builddag.ArtifactID{Kind: "lifecycle-stage-start", Key: string(stage)}
	var stageStart time.Time
	var skipStage bool

	builder.AddTask(builddag.TaskSpec{
		ID:        builddag.TaskID("lifecycle." + string(stage) + ".start"),
		Group:     string(stage),
		Requires:  append([]builddag.ArtifactID(nil), requires...),
		Provides:  []builddag.ArtifactID{startedArtifact},
		Scope:     builddag.ScopeSite,
		Version:   "legacy-plugin-stage-v1",
		Exclusive: true,
		Func: func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			stageStart = time.Now()
			if observe != nil {
				observe(stage, true, nil)
			}
			buildstats.SetActiveStage(string(stage))
			verbosef("  [%s] running as serial plugin DAG...", stage)

			// Manager.RunTo clears template caches before every requested stage,
			// including an already-completed stage. Preserve that boundary while
			// executing the stage hooks explicitly.
			templates.ClearAllCaches()
			skipStage = m.HasRun(stage)
			return nil
		},
	})

	lastRequires := []builddag.ArtifactID{startedArtifact}
	pluginTasks := builddag.LegacyTasks(m, stage, lastRequires)
	for i := range pluginTasks {
		task := pluginTasks[i]
		original := task.Func
		task.Func = func(ctx context.Context) error {
			if skipStage {
				return nil
			}
			if err := original(ctx); err != nil {
				finishDAGPluginStageError(stage, err, observe)
				return err
			}
			return nil
		}
		builder.AddTask(task)
		lastRequires = append([]builddag.ArtifactID(nil), task.Provides...)
	}

	builder.AddTask(builddag.TaskSpec{
		ID:        builddag.TaskID("lifecycle." + string(stage) + ".complete"),
		Group:     string(stage),
		Requires:  lastRequires,
		Provides:  []builddag.ArtifactID{provided},
		Scope:     builddag.ScopeSite,
		Version:   "legacy-plugin-stage-v1",
		Exclusive: true,
		Func: func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				finishDAGPluginStageError(stage, err, observe)
				return err
			}
			if !skipStage {
				if err := m.MarkStageComplete(stage); err != nil {
					finishDAGPluginStageError(stage, err, observe)
					return err
				}
			}

			buildstats.SetActiveStage("")
			if observe != nil {
				observe(stage, false, nil)
			}
			stageElapsed := time.Since(stageStart)
			buildstats.RecordStage(string(stage), stageElapsed)
			if verbose {
				verbosef("  [%s] done in %s", stage, stageElapsed.Truncate(100*time.Microsecond))
			}
			return nil
		},
	})
}

func finishDAGPluginStageError(stage lifecycle.Stage, err error, observe func(lifecycle.Stage, bool, error)) {
	buildstats.SetActiveStage("")
	if observe != nil {
		observe(stage, false, err)
	}
}

func runDAGLifecycleStage(m *lifecycle.Manager, stage lifecycle.Stage, observe func(lifecycle.Stage, bool, error)) error {
	stageStart := time.Now()
	if observe != nil {
		observe(stage, true, nil)
	}
	buildstats.SetActiveStage(string(stage))
	defer buildstats.SetActiveStage("")
	verbosef("  [%s] running...", stage)
	if err := m.RunTo(stage); err != nil {
		if observe != nil {
			observe(stage, false, err)
		}
		return fmt.Errorf("stage %s: %w", stage, err)
	}
	if observe != nil {
		observe(stage, false, nil)
	}
	stageElapsed := time.Since(stageStart)
	buildstats.RecordStage(string(stage), stageElapsed)
	if verbose {
		verbosef("  [%s] done in %s", stage, stageElapsed.Truncate(100*time.Microsecond))
		switch stage {
		case lifecycle.StageGlob:
			verbosef("  [%s] discovered %d files", stage, len(m.Files()))
		case lifecycle.StageLoad:
			verbosef("  [%s] loaded %d posts", stage, len(m.Posts()))
		case lifecycle.StageCollect:
			verbosef("  [%s] collected %d feeds", stage, len(m.Feeds()))
		case lifecycle.StageConfigure, lifecycle.StageValidate, lifecycle.StageTransform,
			lifecycle.StageRender, lifecycle.StageWrite, lifecycle.StageCleanup:
		}
	}
	return nil
}
