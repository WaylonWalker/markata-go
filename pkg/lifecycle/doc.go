// Package lifecycle provides the lifecycle management system for markata-go.
//
// The lifecycle system orchestrates the build process through 9 stages:
//
//   - configure: Load configuration and initialize plugins
//   - validate: Validate configuration before processing
//   - glob: Discover content files
//   - load: Parse files into posts
//   - transform: Pre-render processing (jinja-md, etc.)
//   - render: Convert markdown to HTML
//   - collect: Build feeds, navigation, and aggregated content
//   - write: Write output files
//   - cleanup: Release resources
//
// # Plugin System
//
// Plugins implement optional interfaces corresponding to each stage:
//
//	type MyPlugin struct{}
//
//	func (p *MyPlugin) Name() string { return "my-plugin" }
//
//	func (p *MyPlugin) Load(m *Manager) error {
//	    // Process files into posts
//	    return nil
//	}
//
//	func (p *MyPlugin) Render(m *Manager) error {
//	    // Render posts to HTML
//	    return nil
//	}
//
// # Priority Ordering
//
// Plugins can implement PriorityPlugin to control execution order:
//
//	func (p *MyPlugin) Priority(stage Stage) int {
//	    if stage == StageRender {
//	        return PriorityLast // Run after other render plugins
//	    }
//	    return PriorityDefault
//	}
//
// # Usage
//
// Basic usage:
//
//	m := lifecycle.NewManager()
//	m.RegisterPlugin(&MyGlobPlugin{})
//	m.RegisterPlugin(&MyLoadPlugin{})
//	m.RegisterPlugin(&MyRenderPlugin{})
//
//	if err := m.Run(); err != nil {
//	    log.Fatal(err)
//	}
//
// Running up to a specific stage:
//
//	// Run only up to load stage (useful for preview/watch mode)
//	if err := m.RunTo(lifecycle.StageLoad); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Later, continue from where we left off
//	if err := m.Run(); err != nil {
//	    log.Fatal(err)
//	}
//
// Filtering and mapping posts:
//
//	// Get published posts
//	posts, _ := m.Filter("published==true")
//
//	// Get all titles of posts with tag "golang", sorted by date
//	titles, _ := m.Map("title", "tags contains golang", "date", true)
package lifecycle
