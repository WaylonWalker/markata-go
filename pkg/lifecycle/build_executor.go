package lifecycle

import "fmt"

const buildExecutorCacheKey = "lifecycle.build_executor"

// BuildExecutor identifies the engine responsible for a build.
type BuildExecutor string

const (
	// BuildExecutorLegacy is the normal lifecycle executor and remains the default.
	BuildExecutorLegacy BuildExecutor = "legacy"
	// BuildExecutorDAG is the feature-flagged task-graph executor.
	BuildExecutorDAG BuildExecutor = "dag"
)

// Valid reports whether the executor is a supported build engine identity.
func (executor BuildExecutor) Valid() bool {
	switch executor {
	case BuildExecutorLegacy, BuildExecutorDAG:
		return true
	default:
		return false
	}
}

// SetBuildExecutor records runtime executor identity on the manager without
// mixing execution metadata into user configuration or cache fingerprints.
func (m *Manager) SetBuildExecutor(executor BuildExecutor) error {
	if m == nil {
		return fmt.Errorf("lifecycle: manager is nil")
	}
	if !executor.Valid() {
		return fmt.Errorf("lifecycle: unsupported build executor %q", executor)
	}
	cache := m.Cache()
	if cache == nil {
		return fmt.Errorf("lifecycle: manager cache is nil")
	}
	cache.Set(buildExecutorCacheKey, executor)
	return nil
}

// BuildExecutor returns the selected runtime executor. Managers default to the
// legacy lifecycle executor until an alternate executor explicitly opts in.
func (m *Manager) BuildExecutor() BuildExecutor {
	if m == nil {
		return BuildExecutorLegacy
	}
	cache := m.Cache()
	if cache == nil {
		return BuildExecutorLegacy
	}
	value, ok := cache.Get(buildExecutorCacheKey)
	if !ok {
		return BuildExecutorLegacy
	}
	switch executor := value.(type) {
	case BuildExecutor:
		if executor.Valid() {
			return executor
		}
	case string:
		parsed := BuildExecutor(executor)
		if parsed.Valid() {
			return parsed
		}
	}
	return BuildExecutorLegacy
}
