package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

func TestBenchmarkJSONIncludesExecutor(t *testing.T) {
	previous := buildDAG
	t.Cleanup(func() { buildDAG = previous })
	t.Setenv(dagBuildEnv, envValueDisabled)

	for _, test := range []struct {
		name string
		dag  bool
		want lifecycle.BuildExecutor
	}{
		{name: "legacy", want: lifecycle.BuildExecutorLegacy},
		{name: "dag", dag: true, want: lifecycle.BuildExecutorDAG},
	} {
		t.Run(test.name, func(t *testing.T) {
			buildDAG = test.dag
			var output bytes.Buffer
			if err := writeBenchmarkJSON(&output, &BuildResult{}); err != nil {
				t.Fatalf("writeBenchmarkJSON() = %v", err)
			}
			var decoded struct {
				Executor lifecycle.BuildExecutor `json:"executor"`
			}
			if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
				t.Fatalf("benchmark JSON = %v", err)
			}
			if decoded.Executor != test.want {
				t.Fatalf("executor = %q, want %q", decoded.Executor, test.want)
			}
		})
	}
}

func TestRunDAGBuildRecordsExecutorOnManager(t *testing.T) {
	manager := lifecycle.NewManager()
	if _, err := runDAGBuildObserved(manager, nil); err != nil {
		t.Fatalf("runDAGBuildObserved() = %v", err)
	}
	if got := manager.BuildExecutor(); got != lifecycle.BuildExecutorDAG {
		t.Fatalf("BuildExecutor() = %q, want %q", got, lifecycle.BuildExecutorDAG)
	}
}
