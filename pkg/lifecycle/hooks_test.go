package lifecycle

import (
	"errors"
	"testing"
)

type adapterTestPlugin struct {
	err error
}

func (p adapterTestPlugin) Name() string             { return "adapter-test" }
func (p adapterTestPlugin) Transform(*Manager) error { return p.err }

type adapterCriticalError struct{}

func (adapterCriticalError) Error() string    { return "critical adapter error" }
func (adapterCriticalError) IsCritical() bool { return true }

func TestExecutePluginHook_PreservesWarningPolicy(t *testing.T) {
	m := NewManager()
	if err := ExecutePluginHook(m, adapterTestPlugin{err: errors.New("warning")}, StageTransform); err != nil {
		t.Fatalf("non-critical error = %v", err)
	}
	warnings := m.Warnings()
	if len(warnings) != 1 || warnings[0].Plugin != "adapter-test" || warnings[0].Critical {
		t.Fatalf("warnings = %+v", warnings)
	}
	if m.CurrentStage() != StageTransform {
		t.Fatalf("current stage = %s", m.CurrentStage())
	}
}

func TestExecutePluginHook_StopsOnCriticalError(t *testing.T) {
	m := NewManager()
	err := ExecutePluginHook(m, adapterTestPlugin{err: adapterCriticalError{}}, StageTransform)
	var hookErrors *HookErrors
	if !errors.As(err, &hookErrors) || !hookErrors.HasCritical() {
		t.Fatalf("critical error = %v", err)
	}
	if len(m.Warnings()) != 0 {
		t.Fatalf("critical error became warning: %+v", m.Warnings())
	}
}
