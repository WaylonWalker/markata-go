package cmd

import (
	"encoding/json"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

// MarshalJSON adds executor identity to the existing benchmark payload without
// changing the human build result or the benchmark writer call sites.
func (output benchmarkJSONOutput) MarshalJSON() ([]byte, error) {
	type benchmarkJSONAlias benchmarkJSONOutput
	return json.Marshal(struct {
		Executor lifecycle.BuildExecutor `json:"executor"`
		benchmarkJSONAlias
	}{
		Executor:           selectedBenchmarkExecutor(),
		benchmarkJSONAlias: benchmarkJSONAlias(output),
	})
}

func selectedBenchmarkExecutor() lifecycle.BuildExecutor {
	if dagBuildEnabled() {
		return lifecycle.BuildExecutorDAG
	}
	return lifecycle.BuildExecutorLegacy
}
