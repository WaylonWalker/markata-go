package lifecycle

import (
	"errors"
	"reflect"
	"testing"
)

type executionTestPlugin struct {
	name      string
	priority  int
	transform func(*Manager) error
	load      func(*Manager) error
}

func (p *executionTestPlugin) Name() string { return p.name }

func (p *executionTestPlugin) Priority(Stage) int { return p.priority }

func (p *executionTestPlugin) Transform(m *Manager) error {
	if p.transform == nil {
		return nil
	}
	return p.transform(m)
}

func (p *executionTestPlugin) Load(m *Manager) error {
	if p.load == nil {
		return nil
	}
	return p.load(m)
}

type executionNameOnlyPlugin struct{ name string }

func (p *executionNameOnlyPlugin) Name() string { return p.name }

func TestSortPluginsByPriorityStable(t *testing.T) {
	plugins := []Plugin{
		&executionTestPlugin{name: "default-a"},
		&executionTestPlugin{name: "early", priority: PriorityEarly},
		&executionTestPlugin{name: "default-b"},
	}
	gotPlugins := SortPluginsByPriority(plugins, StageTransform)
	got := make([]string, 0, len(gotPlugins))
	for _, plugin := range gotPlugins {
		got = append(got, plugin.Name())
	}
	want := []string{"early", "default-a", "default-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("priority order = %v, want %v", got, want)
	}
}

func TestExecutePluginHookSetsCurrentStageBeforeHook(t *testing.T) {
	manager := NewManager()
	plugin := &executionTestPlugin{name: "stage-aware", transform: func(m *Manager) error {
		if got := m.CurrentStage(); got != StageTransform {
			t.Fatalf("CurrentStage() inside hook = %q, want %q", got, StageTransform)
		}
		return nil
	}}

	if err := ExecutePluginHook(manager, plugin, StageTransform); err != nil {
		t.Fatalf("ExecutePluginHook() = %v", err)
	}
}

func TestExecutePluginHookCollectsNonCriticalWarning(t *testing.T) {
	manager := NewManager()
	pluginErr := errors.New("transform warning")
	plugin := &executionTestPlugin{name: "warn", transform: func(*Manager) error { return pluginErr }}

	if err := ExecutePluginHook(manager, plugin, StageTransform); err != nil {
		t.Fatalf("ExecutePluginHook() = %v, want non-critical warning", err)
	}
	warnings := manager.Warnings()
	if len(warnings) != 1 {
		t.Fatalf("warnings = %d, want 1", len(warnings))
	}
	if warnings[0].Plugin != "warn" || warnings[0].Stage != StageTransform || !errors.Is(warnings[0], pluginErr) {
		t.Fatalf("warning = %+v, want transform warning from warn", warnings[0])
	}
	if warnings[0].Critical {
		t.Fatal("transform warning unexpectedly marked critical")
	}
}

func TestExecutePluginHookReturnsCriticalError(t *testing.T) {
	manager := NewManager()
	pluginErr := errors.New("load failed")
	plugin := &executionTestPlugin{name: "loader", load: func(*Manager) error { return pluginErr }}

	err := ExecutePluginHook(manager, plugin, StageLoad)
	if err == nil {
		t.Fatal("ExecutePluginHook() = nil, want critical load error")
	}
	var hookErr *HookError
	if !errors.As(err, &hookErr) {
		t.Fatalf("error = %T %v, want *HookError", err, err)
	}
	if !hookErr.Critical || hookErr.Plugin != "loader" || hookErr.Stage != StageLoad || !errors.Is(err, pluginErr) {
		t.Fatalf("hook error = %+v, want critical load error", hookErr)
	}
	if len(manager.Warnings()) != 0 {
		t.Fatalf("critical error was also recorded as warning: %+v", manager.Warnings())
	}
}

func TestExecutePluginHookRejectsUnsupportedStage(t *testing.T) {
	manager := NewManager()
	plugin := &executionNameOnlyPlugin{name: "name-only"}
	if err := ExecutePluginHook(manager, plugin, StageTransform); err == nil {
		t.Fatal("unsupported plugin stage was accepted")
	}
}
