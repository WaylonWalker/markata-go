package cmd

import (
	"context"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/builddag"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

// runSerialDAGBuild executes the first native graph slice and resumes the
// legacy collect/write/cleanup stages.  MaxParallel is deliberately fixed at
// one by builddag.NewExecutor.
func runSerialDAGBuild(m *lifecycle.Manager) (*BuildResult, error) {
	start := time.Now()
	spine, err := builddag.NewMarkataSpine(m, buildDAGSeed, buildDAGRandom)
	if err != nil {
		return nil, err
	}
	if digest, digestErr := spine.Graph.Digest(); digestErr == nil {
		m.Cache().Set("builddag_graph_digest", digest)
	}
	if err := spine.Run(context.Background()); err != nil {
		return nil, err
	}

	result := &BuildResult{
		PostsProcessed: len(m.Posts()),
		FeedsGenerated: len(m.Feeds()),
		DAG: &DAGBuildInfo{
			Enabled:         true,
			MaxParallel:     spine.Executor.MaxParallel,
			Randomized:      buildDAGRandom,
			Seed:            buildDAGSeed,
			TaskCount:       len(spine.Graph.Order()),
			GraphDigest:     graphDigest(spine.Graph),
			DurationSeconds: time.Since(start).Seconds(),
		},
	}
	result.BlogrollStatus = getBlogrollStatus(m)
	for _, warning := range m.Warnings() {
		result.Warnings = append(result.Warnings, warning.Error())
	}
	return result, nil
}

func graphDigest(graph *builddag.Graph) string {
	if graph == nil {
		return ""
	}
	digest, digestErr := graph.Digest()
	if digestErr != nil {
		return ""
	}
	return digest
}
