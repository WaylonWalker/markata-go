package lifecycle

import "testing"

func TestBuildExecutorDefaultsToLegacy(t *testing.T) {
	manager := NewManager()
	if got := manager.BuildExecutor(); got != BuildExecutorLegacy {
		t.Fatalf("BuildExecutor() = %q, want %q", got, BuildExecutorLegacy)
	}
}

func TestSetBuildExecutor(t *testing.T) {
	manager := NewManager()
	if err := manager.SetBuildExecutor(BuildExecutorDAG); err != nil {
		t.Fatalf("SetBuildExecutor() = %v", err)
	}
	if got := manager.BuildExecutor(); got != BuildExecutorDAG {
		t.Fatalf("BuildExecutor() = %q, want %q", got, BuildExecutorDAG)
	}
}

func TestSetBuildExecutorRejectsUnknownIdentity(t *testing.T) {
	manager := NewManager()
	if err := manager.SetBuildExecutor(BuildExecutor("parallel-magic")); err == nil {
		t.Fatal("SetBuildExecutor() accepted an unknown executor")
	}
	if got := manager.BuildExecutor(); got != BuildExecutorLegacy {
		t.Fatalf("BuildExecutor() after rejected value = %q, want %q", got, BuildExecutorLegacy)
	}
}
