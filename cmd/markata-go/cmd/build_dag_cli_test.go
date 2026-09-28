package cmd

import "testing"

func TestDAGBuildEnabled(t *testing.T) {
	previous := buildDAG
	buildDAG = false
	t.Cleanup(func() { buildDAG = previous })

	t.Setenv(dagBuildEnv, "false")
	if dagBuildEnabled() {
		t.Fatal("dagBuildEnabled() = true with flag=false and env=false")
	}

	t.Setenv(dagBuildEnv, "true")
	if !dagBuildEnabled() {
		t.Fatal("dagBuildEnabled() = false with env=true")
	}

	t.Setenv(dagBuildEnv, "not-a-bool")
	if dagBuildEnabled() {
		t.Fatal("dagBuildEnabled() = true with invalid env value")
	}

	buildDAG = true
	t.Setenv(dagBuildEnv, "false")
	if !dagBuildEnabled() {
		t.Fatal("explicit --dag flag did not take precedence over env=false")
	}
}
