package builddag

import (
	"context"
	"fmt"
)

// ExecutionResult describes one serial graph execution.
type ExecutionResult struct {
	TaskCount int      `json:"task_count"`
	Order     []TaskID `json:"order"`
}

// Executor runs compiled graphs. The initial implementation intentionally
// permits exactly one task at a time so DAG adoption is separated from
// concurrency changes.
type Executor struct {
	MaxParallel int
}

// NewExecutor creates a serial executor. Values other than one are rejected.
func NewExecutor(maxParallel int) (*Executor, error) {
	if maxParallel == 0 {
		maxParallel = 1
	}
	if maxParallel != 1 {
		return nil, fmt.Errorf("builddag: MaxParallel must be 1, got %d", maxParallel)
	}
	return &Executor{MaxParallel: 1}, nil
}

// Execute runs every ready task in the graph's deterministic topological order.
func (e *Executor) Execute(ctx context.Context, graph *Graph) (ExecutionResult, error) {
	if e == nil || e.MaxParallel != 1 {
		return ExecutionResult{}, fmt.Errorf("builddag: serial executor requires MaxParallel=1")
	}
	if graph == nil {
		return ExecutionResult{}, fmt.Errorf("builddag: graph is nil")
	}

	available := make(map[ArtifactID]bool, len(graph.external))
	for id := range graph.external {
		available[id] = true
	}
	order := graph.Order()
	for _, id := range order {
		if err := ctx.Err(); err != nil {
			return ExecutionResult{}, err
		}
		task, ok := graph.Task(id)
		if !ok {
			return ExecutionResult{}, fmt.Errorf("builddag: task %q missing from compiled graph", id)
		}
		for _, artifact := range task.Requires {
			if !available[artifact] {
				return ExecutionResult{}, fmt.Errorf("builddag: task %q requires unavailable artifact %s", id, artifact.String())
			}
		}
		if task.Func == nil {
			return ExecutionResult{}, fmt.Errorf("builddag: task %q has no function", id)
		}
		if err := task.Func(ctx); err != nil {
			return ExecutionResult{}, fmt.Errorf("builddag: task %q: %w", id, err)
		}
		for _, artifact := range task.Provides {
			available[artifact] = true
		}
	}
	return ExecutionResult{TaskCount: len(order), Order: order}, nil
}
