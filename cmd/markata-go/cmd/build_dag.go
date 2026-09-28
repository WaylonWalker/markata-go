package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/builddag"
	"github.com/WaylonWalker/markata-go/pkg/buildstats"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
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

// runDAGBuild executes the existing lifecycle through an explicit serial task
// and artifact graph. It intentionally changes no plugin boundaries yet.
func runDAGBuild(m *lifecycle.Manager) (result *BuildResult, err error) {
	return runDAGBuildObserved(m, nil)
}

// runDAGBuildObserved is the feature-flagged DAG equivalent of
// runBuildObserved. The observer contract is kept identical so serve can opt
// into this executor without losing stage visibility.
func runDAGBuildObserved(m *lifecycle.Manager, observe func(lifecycle.Stage, bool, error)) (result *BuildResult, err error) {
	profile := buildstats.Start()
	defer func() {
		summary := profile.Stop()
		if result != nil {
			result.Benchmark = summary
		}
	}()

	builder := builddag.NewBuilder()
	var previous *builddag.ArtifactID
	for _, lifecycleStage := range dagLifecycleStages {
		stage := lifecycleStage
		provided := builddag.ArtifactID{Kind: "lifecycle-stage", Key: string(stage)}
		requires := []builddag.ArtifactID(nil)
		if previous != nil {
			requires = []builddag.ArtifactID{*previous}
		}
		builder.AddTask(builddag.TaskSpec{
			ID:        builddag.TaskID("lifecycle." + string(stage)),
			Group:     string(stage),
			Requires:  requires,
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
		current := provided
		previous = &current
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
