package lifecycle

import (
	"fmt"
	"sort"
)

// HookError represents an error that occurred during hook execution.
type HookError struct {
	Stage    Stage
	Plugin   string
	Err      error
	Critical bool
}

func (e *HookError) Error() string {
	severity := "warning"
	if e.Critical {
		severity = "error"
	}
	return fmt.Sprintf("[%s] %s plugin %q: %v", severity, e.Stage, e.Plugin, e.Err)
}

func (e *HookError) Unwrap() error {
	return e.Err
}

// HookErrors is a collection of errors from hook execution.
type HookErrors struct {
	Errors []*HookError
}

func (e *HookErrors) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("%d errors occurred during hook execution; first: %v", len(e.Errors), e.Errors[0])
}

// HasCritical returns true if any error is marked as critical.
func (e *HookErrors) HasCritical() bool {
	for _, err := range e.Errors {
		if err.Critical {
			return true
		}
	}
	return false
}

// Add adds an error to the collection.
func (e *HookErrors) Add(stage Stage, plugin string, err error, critical bool) {
	e.Errors = append(e.Errors, &HookError{
		Stage:    stage,
		Plugin:   plugin,
		Err:      err,
		Critical: critical,
	})
}

// pluginWithPriority wraps a plugin with its computed priority for sorting.
type pluginWithPriority struct {
	plugin   Plugin
	priority int
}

// sortPluginsByPriority returns plugins sorted by their priority for the given stage.
// Plugins implementing PriorityPlugin have their priority queried; others use PriorityDefault.
func sortPluginsByPriority(plugins []Plugin, stage Stage) []Plugin {
	wrapped := make([]pluginWithPriority, len(plugins))
	for i, p := range plugins {
		priority := PriorityDefault
		if pp, ok := p.(PriorityPlugin); ok {
			priority = pp.Priority(stage)
		}
		wrapped[i] = pluginWithPriority{plugin: p, priority: priority}
	}

	sort.SliceStable(wrapped, func(i, j int) bool {
		return wrapped[i].priority < wrapped[j].priority
	})

	sorted := make([]Plugin, len(plugins))
	for i, w := range wrapped {
		sorted[i] = w.plugin
	}
	return sorted
}

// isCriticalStage returns true if errors in the given stage should halt execution.
func isCriticalStage(stage Stage) bool {
	switch stage {
	case StageConfigure, StageValidate, StageGlob, StageLoad:
		// Early stages are critical - can't continue without them
		return true
	case StageTransform, StageRender, StageCollect, StageWrite:
		// Later stages can potentially continue on partial failures
		return false
	case StageCleanup:
		// Cleanup errors are warnings only
		return false
	default:
		return true
	}
}

// executeHooks runs all plugins that implement the given stage interface.
// Returns collected errors. If any critical error occurs, execution stops.
func executeHooks[T Plugin](
	m *Manager,
	stage Stage,
	plugins []Plugin,
	check func(Plugin) (T, bool),
	execute func(T) error,
) *HookErrors {
	errors := &HookErrors{}
	critical := isCriticalStage(stage)

	// Sort plugins by priority
	sorted := sortPluginsByPriority(plugins, stage)

	for _, p := range sorted {
		if typed, ok := check(p); ok {
			if err := execute(typed); err != nil {
				errors.Add(stage, p.Name(), err, critical)
				if critical {
					// Stop on first critical error
					return errors
				}
			}
		}
	}

	return errors
}

// runConfigureHooks executes all ConfigurePlugin hooks.
func runConfigureHooks(m *Manager) *HookErrors {
	return executeHooks(m, StageConfigure, m.plugins,
		func(p Plugin) (ConfigurePlugin, bool) {
			cp, ok := p.(ConfigurePlugin)
			return cp, ok
		},
		func(cp ConfigurePlugin) error {
			return cp.Configure(m)
		},
	)
}

// runValidateHooks executes all ValidatePlugin hooks.
func runValidateHooks(m *Manager) *HookErrors {
	return executeHooks(m, StageValidate, m.plugins,
		func(p Plugin) (ValidatePlugin, bool) {
			vp, ok := p.(ValidatePlugin)
			return vp, ok
		},
		func(vp ValidatePlugin) error {
			return vp.Validate(m)
		},
	)
}

// runGlobHooks executes all GlobPlugin hooks.
func runGlobHooks(m *Manager) *HookErrors {
	return executeHooks(m, StageGlob, m.plugins,
		func(p Plugin) (GlobPlugin, bool) {
			gp, ok := p.(GlobPlugin)
			return gp, ok
		},
		func(gp GlobPlugin) error {
			return gp.Glob(m)
		},
	)
}

// runLoadHooks executes all LoadPlugin hooks.
func runLoadHooks(m *Manager) *HookErrors {
	return executeHooks(m, StageLoad, m.plugins,
		func(p Plugin) (LoadPlugin, bool) {
			lp, ok := p.(LoadPlugin)
			return lp, ok
		},
		func(lp LoadPlugin) error {
			return lp.Load(m)
		},
	)
}

// runTransformHooks executes all TransformPlugin hooks.
func runTransformHooks(m *Manager) *HookErrors {
	return executeHooks(m, StageTransform, m.plugins,
		func(p Plugin) (TransformPlugin, bool) {
			tp, ok := p.(TransformPlugin)
			return tp, ok
		},
		func(tp TransformPlugin) error {
			return tp.Transform(m)
		},
	)
}

// runRenderHooks executes all RenderPlugin hooks.
func runRenderHooks(m *Manager) *HookErrors {
	return executeHooks(m, StageRender, m.plugins,
		func(p Plugin) (RenderPlugin, bool) {
			rp, ok := p.(RenderPlugin)
			return rp, ok
		},
		func(rp RenderPlugin) error {
			return rp.Render(m)
		},
	)
}

// runCollectHooks executes all CollectPlugin hooks.
func runCollectHooks(m *Manager) *HookErrors {
	return executeHooks(m, StageCollect, m.plugins,
		func(p Plugin) (CollectPlugin, bool) {
			cp, ok := p.(CollectPlugin)
			return cp, ok
		},
		func(cp CollectPlugin) error {
			return cp.Collect(m)
		},
	)
}

// runWriteHooks executes all WritePlugin hooks.
func runWriteHooks(m *Manager) *HookErrors {
	return executeHooks(m, StageWrite, m.plugins,
		func(p Plugin) (WritePlugin, bool) {
			wp, ok := p.(WritePlugin)
			return wp, ok
		},
		func(wp WritePlugin) error {
			return wp.Write(m)
		},
	)
}

// runCleanupHooks executes all CleanupPlugin hooks.
func runCleanupHooks(m *Manager) *HookErrors {
	return executeHooks(m, StageCleanup, m.plugins,
		func(p Plugin) (CleanupPlugin, bool) {
			cp, ok := p.(CleanupPlugin)
			return cp, ok
		},
		func(cp CleanupPlugin) error {
			return cp.Cleanup(m)
		},
	)
}
