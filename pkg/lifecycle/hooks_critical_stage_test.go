package lifecycle

import (
	"errors"
	"testing"
)

type criticalWriteTestPlugin struct {
	name     string
	critical bool
	called   *int
}

func (p *criticalWriteTestPlugin) Name() string { return p.name }
func (p *criticalWriteTestPlugin) Write(*Manager) error {
	if p.called != nil {
		(*p.called)++
	}
	return errors.New("write failed")
}
func (p *criticalWriteTestPlugin) CriticalStageErrors(stage Stage) bool {
	return p.critical && stage == StageWrite
}

func TestRunWriteHooksPluginCanMakeWriteFailureCritical(t *testing.T) {
	m := NewManager()
	firstCalls := 0
	laterCalls := 0
	m.RegisterPlugin(&criticalWriteTestPlugin{name: "publisher", critical: true, called: &firstCalls})
	m.RegisterPlugin(&criticalWriteTestPlugin{name: "later", called: &laterCalls})

	errs := runWriteHooks(m)
	if errs == nil || !errs.HasCritical() {
		t.Fatalf("write errors = %#v, want critical error", errs)
	}
	if firstCalls != 1 {
		t.Fatalf("critical plugin calls = %d, want 1", firstCalls)
	}
	if laterCalls != 0 {
		t.Fatalf("later plugin calls = %d, want 0 after critical failure", laterCalls)
	}
}

func TestRunWriteHooksKeepsOrdinaryWriteFailureNonCritical(t *testing.T) {
	m := NewManager()
	firstCalls := 0
	laterCalls := 0
	m.RegisterPlugin(&criticalWriteTestPlugin{name: "optional", called: &firstCalls})
	m.RegisterPlugin(&criticalWriteTestPlugin{name: "later", called: &laterCalls})

	errs := runWriteHooks(m)
	if errs == nil || errs.HasCritical() {
		t.Fatalf("write errors = %#v, want warning-only errors", errs)
	}
	if firstCalls != 1 || laterCalls != 1 {
		t.Fatalf("plugin calls = %d/%d, want both plugins to run", firstCalls, laterCalls)
	}
}
