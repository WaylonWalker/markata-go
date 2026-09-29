package cmd

import (
	"context"
	"reflect"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/builddag"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

type dagPluginStageTestPlugin struct {
	name     string
	priority int
	calls    *[]string
}

func (p *dagPluginStageTestPlugin) Name() string { return p.name }

func (p *dagPluginStageTestPlugin) Priority(lifecycle.Stage) int { return p.priority }

func (p *dagPluginStageTestPlugin) Transform(*lifecycle.Manager) error {
	*p.calls = append(*p.calls, p.name)
	return nil
}

func (p *dagPluginStageTestPlugin) Collect(*lifecycle.Manager) error {
	*p.calls = append(*p.calls, p.name)
	return nil
}

func TestDAGUsesPluginTasks(t *testing.T) {
	tests := []struct {
		stage lifecycle.Stage
		want  bool
	}{
		{stage: lifecycle.StageConfigure},
		{stage: lifecycle.StageValidate},
		{stage: lifecycle.StageGlob},
		{stage: lifecycle.StageLoad},
		{stage: lifecycle.StageTransform, want: true},
		{stage: lifecycle.StageRender, want: true},
		{stage: lifecycle.StageCollect, want: true},
		{stage: lifecycle.StageWrite},
		{stage: lifecycle.StageCleanup},
	}

	for _, test := range tests {
		if got := dagUsesPluginTasks(test.stage); got != test.want {
			t.Errorf("dagUsesPluginTasks(%s) = %v, want %v", test.stage, got, test.want)
		}
	}
}

func TestDAGLegacyPluginStagePreservesOrderAndObserver(t *testing.T) {
	for _, stage := range []lifecycle.Stage{lifecycle.StageTransform, lifecycle.StageCollect} {
		t.Run(string(stage), func(t *testing.T) {
			manager := lifecycle.NewManager()
			calls := []string{}
			manager.RegisterPlugins(
				&dagPluginStageTestPlugin{name: "late", priority: lifecycle.PriorityLate, calls: &calls},
				&dagPluginStageTestPlugin{name: "early", priority: lifecycle.PriorityEarly, calls: &calls},
			)

			observerEvents := []string{}
			observe := func(observedStage lifecycle.Stage, starting bool, err error) {
				if err != nil {
					t.Fatalf("observer error for %s: %v", observedStage, err)
				}
				if observedStage != stage {
					t.Fatalf("observer stage = %s, want %s", observedStage, stage)
				}
				if starting {
					observerEvents = append(observerEvents, "start")
					return
				}
				observerEvents = append(observerEvents, "finish")
			}

			builder := builddag.NewBuilder()
			provided := builddag.ArtifactID{Kind: "lifecycle-stage", Key: string(stage)}
			addDAGLegacyPluginStage(builder, manager, stage, nil, provided, observe)
			graph, err := builder.Compile()
			if err != nil {
				t.Fatalf("Compile() = %v", err)
			}
			executor, err := builddag.NewExecutor(1)
			if err != nil {
				t.Fatalf("NewExecutor() = %v", err)
			}
			result, err := executor.Execute(context.Background(), graph)
			if err != nil {
				t.Fatalf("Execute() = %v", err)
			}

			if got, want := calls, []string{"early", "late"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("plugin calls = %v, want %v", got, want)
			}
			if got, want := observerEvents, []string{"start", "finish"}; !reflect.DeepEqual(got, want) {
				t.Fatalf("observer events = %v, want %v", got, want)
			}
			if !manager.HasRun(stage) {
				t.Fatalf("%s stage was not marked complete", stage)
			}
			if got, want := result.TaskCount, 4; got != want {
				t.Fatalf("task count = %d, want %d", got, want)
			}
		})
	}
}
