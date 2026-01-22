package lifecycle

// Plugin is the base interface that all plugins must implement.
type Plugin interface {
	// Name returns the unique name of the plugin.
	Name() string
}

// ConfigurePlugin is implemented by plugins that participate in the configure stage.
// This stage is used to load configuration and initialize plugin state.
type ConfigurePlugin interface {
	Plugin
	Configure(m *Manager) error
}

// ValidatePlugin is implemented by plugins that participate in the validate stage.
// This stage is used to validate configuration before processing begins.
type ValidatePlugin interface {
	Plugin
	Validate(m *Manager) error
}

// GlobPlugin is implemented by plugins that participate in the glob stage.
// This stage is used to discover content files.
type GlobPlugin interface {
	Plugin
	Glob(m *Manager) error
}

// LoadPlugin is implemented by plugins that participate in the load stage.
// This stage is used to parse files into posts.
type LoadPlugin interface {
	Plugin
	Load(m *Manager) error
}

// TransformPlugin is implemented by plugins that participate in the transform stage.
// This stage is used for pre-render processing (jinja-md, etc.).
type TransformPlugin interface {
	Plugin
	Transform(m *Manager) error
}

// RenderPlugin is implemented by plugins that participate in the render stage.
// This stage is used to convert markdown to HTML.
type RenderPlugin interface {
	Plugin
	Render(m *Manager) error
}

// CollectPlugin is implemented by plugins that participate in the collect stage.
// This stage is used to build feeds, navigation, and other aggregated content.
type CollectPlugin interface {
	Plugin
	Collect(m *Manager) error
}

// WritePlugin is implemented by plugins that participate in the write stage.
// This stage is used to write output files.
type WritePlugin interface {
	Plugin
	Write(m *Manager) error
}

// CleanupPlugin is implemented by plugins that participate in the cleanup stage.
// This stage is used to release resources and perform cleanup tasks.
type CleanupPlugin interface {
	Plugin
	Cleanup(m *Manager) error
}

// PriorityPlugin can be implemented by plugins to control execution order within a stage.
// Plugins with lower priority values run first.
type PriorityPlugin interface {
	Plugin
	// Priority returns the plugin's priority for a given stage.
	// Lower values run first. Default priority is 0.
	// Use negative values for "tryfirst" behavior.
	// Use positive values for "trylast" behavior.
	Priority(stage Stage) int
}

// Priority constants for common ordering scenarios.
const (
	// PriorityFirst ensures a plugin runs before most others.
	PriorityFirst = -1000

	// PriorityEarly ensures a plugin runs early in the stage.
	PriorityEarly = -100

	// PriorityDefault is the default priority.
	PriorityDefault = 0

	// PriorityLate ensures a plugin runs late in the stage.
	PriorityLate = 100

	// PriorityLast ensures a plugin runs after most others.
	PriorityLast = 1000
)
