package lifecycle

import "fmt"

// MarkStageComplete records that stage was completed by an alternate executor.
// It intentionally does not execute hooks: schedulers must only call it after
// they have run the stage's hooks with the same lifecycle semantics.
func (m *Manager) MarkStageComplete(stage Stage) error {
	if m == nil {
		return fmt.Errorf("lifecycle: manager is nil")
	}
	if !IsValidStage(stage) {
		return fmt.Errorf("lifecycle: invalid stage %q", stage)
	}

	m.mu.Lock()
	m.currentStage = stage
	m.stagesRun[stage] = true
	m.mu.Unlock()
	return nil
}
