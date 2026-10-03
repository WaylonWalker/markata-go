package lifecycle

import "testing"

func TestMarkStageComplete(t *testing.T) {
	manager := NewManager()
	if manager.HasRun(StageTransform) {
		t.Fatal("transform unexpectedly marked complete before scheduler")
	}

	if err := manager.MarkStageComplete(StageTransform); err != nil {
		t.Fatalf("MarkStageComplete() = %v", err)
	}
	if !manager.HasRun(StageTransform) {
		t.Fatal("transform was not marked complete")
	}
	if got := manager.CurrentStage(); got != StageTransform {
		t.Fatalf("CurrentStage() = %q, want %q", got, StageTransform)
	}
}

func TestMarkStageCompleteRejectsInvalidStage(t *testing.T) {
	manager := NewManager()
	if err := manager.MarkStageComplete(Stage("not-a-stage")); err == nil {
		t.Fatal("MarkStageComplete() accepted an invalid stage")
	}
}
