package plugins

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
)

func TestDefaultPlugins_HaveUniqueNames(t *testing.T) {
	seen := make(map[string]struct{})

	for _, plugin := range DefaultPlugins() {
		name := plugin.Name()
		if _, exists := seen[name]; exists {
			t.Fatalf("duplicate default plugin registration: %q", name)
		}
		seen[name] = struct{}{}
	}
}

func TestDefaultPlugins_MultiStageCoverage(t *testing.T) {
	manager := lifecycle.NewManager()
	manager.RegisterPlugins(DefaultPlugins()...)

	stats := defaultPluginByName(t, manager, "stats")
	if _, ok := stats.(*StatsPlugin); !ok {
		t.Fatalf("stats registered as %T, want *StatsPlugin", stats)
	}
	if _, ok := stats.(lifecycle.ConfigurePlugin); !ok {
		t.Error("stats does not implement configure stage")
	}
	if _, ok := stats.(lifecycle.TransformPlugin); !ok {
		t.Error("stats does not implement transform stage")
	}
	if _, ok := stats.(lifecycle.CollectPlugin); !ok {
		t.Error("stats does not implement collect stage")
	}
	assertPluginPriority(t, stats, lifecycle.StageTransform, lifecycle.PriorityEarly)
	assertPluginPriority(t, stats, lifecycle.StageCollect, lifecycle.PriorityLate)

	blogroll := defaultPluginByName(t, manager, "blogroll")
	if _, ok := blogroll.(*BlogrollPlugin); !ok {
		t.Fatalf("blogroll registered as %T, want *BlogrollPlugin", blogroll)
	}
	if _, ok := blogroll.(lifecycle.ConfigurePlugin); !ok {
		t.Error("blogroll does not implement configure stage")
	}
	if _, ok := blogroll.(lifecycle.CollectPlugin); !ok {
		t.Error("blogroll does not implement collect stage")
	}
	if _, ok := blogroll.(lifecycle.WritePlugin); !ok {
		t.Error("blogroll does not implement write stage")
	}
	assertPluginPriority(t, blogroll, lifecycle.StageConfigure, lifecycle.PriorityDefault)
	assertPluginPriority(t, blogroll, lifecycle.StageCollect, lifecycle.PriorityLate+10)
	assertPluginPriority(t, blogroll, lifecycle.StageWrite, lifecycle.PriorityLate+20)
}

func defaultPluginByName(t *testing.T, manager *lifecycle.Manager, name string) lifecycle.Plugin {
	t.Helper()
	var found lifecycle.Plugin
	for _, plugin := range manager.Plugins() {
		if plugin.Name() != name {
			continue
		}
		if found != nil {
			t.Fatalf("registered %q more than once", name)
		}
		found = plugin
	}
	if found == nil {
		t.Fatalf("default plugin %q not registered", name)
	}
	return found
}

func assertPluginPriority(t *testing.T, plugin lifecycle.Plugin, stage lifecycle.Stage, want int) {
	t.Helper()
	priorityPlugin, ok := plugin.(lifecycle.PriorityPlugin)
	if !ok {
		t.Fatalf("%q does not implement lifecycle.PriorityPlugin", plugin.Name())
	}
	if got := priorityPlugin.Priority(stage); got != want {
		t.Errorf("%q %s priority = %d, want %d", plugin.Name(), stage, got, want)
	}
}
