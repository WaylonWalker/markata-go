package cmd

import "testing"

func TestSelectedBuildExecutorDiagnostic(t *testing.T) {
	previousDAG := buildDAG
	t.Cleanup(func() { buildDAG = previousDAG })

	t.Setenv(dagBuildEnv, "false")
	buildDAG = false
	if got, experimental := selectedBuildExecutorDiagnostic(); got != legacyExecutorDiagnostic || experimental {
		t.Fatalf("legacy diagnostic = (%q, %v), want (%q, false)", got, experimental, legacyExecutorDiagnostic)
	}

	buildDAG = true
	if got, experimental := selectedBuildExecutorDiagnostic(); got != dagExecutorDiagnostic || !experimental {
		t.Fatalf("flag DAG diagnostic = (%q, %v), want (%q, true)", got, experimental, dagExecutorDiagnostic)
	}

	buildDAG = false
	t.Setenv(dagBuildEnv, "true")
	if got, experimental := selectedBuildExecutorDiagnostic(); got != dagExecutorDiagnostic || !experimental {
		t.Fatalf("env DAG diagnostic = (%q, %v), want (%q, true)", got, experimental, dagExecutorDiagnostic)
	}
}

func TestExecutorDiagnosticLabels(t *testing.T) {
	if got, experimental := executorDiagnostic(false); got != legacyExecutorDiagnostic || experimental {
		t.Fatalf("legacy executorDiagnostic = (%q, %v), want (%q, false)", got, experimental, legacyExecutorDiagnostic)
	}
	if got, experimental := executorDiagnostic(true); got != dagExecutorDiagnostic || !experimental {
		t.Fatalf("DAG executorDiagnostic = (%q, %v), want (%q, true)", got, experimental, dagExecutorDiagnostic)
	}
}
