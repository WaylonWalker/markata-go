package cmd

import (
	"os"
	"testing"
)

func TestBuilderAdminRegistersDAGFlag(t *testing.T) {
	flag := builderAdminCmd.Flags().Lookup("dag")
	if flag == nil {
		t.Fatal("builder-admin --dag flag is not registered")
	}
	if flag.DefValue != envValueDisabled {
		t.Fatalf("builder-admin --dag default = %q, want false", flag.DefValue)
	}
}

func TestEnableBuilderAdminDAGEnvironment(t *testing.T) {
	previousCommand := currentCmd
	currentCmd = nil
	t.Cleanup(func() { currentCmd = previousCommand })
	t.Setenv(dagBuildEnv, envValueDisabled)
	restore, err := enableBuilderAdminDAGEnvironment()
	if err != nil {
		t.Fatalf("enableBuilderAdminDAGEnvironment() = %v", err)
	}

	if got := os.Getenv(dagBuildEnv); got != boolStrTrue {
		t.Fatalf("%s = %q, want true", dagBuildEnv, got)
	}
	if !dagBuildEnabled() {
		t.Fatal("dagBuildEnabled() = false after builder-admin DAG opt-in")
	}
	restore()
	if got := os.Getenv(dagBuildEnv); got != envValueDisabled {
		t.Fatalf("restored %s = %q, want false", dagBuildEnv, got)
	}
}

func TestBuilderAdminCanDisableInheritedDAG(t *testing.T) {
	t.Setenv(dagBuildEnv, boolStrTrue)
	restore, err := setBuilderAdminDAGEnvironment(false)
	if err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv(dagBuildEnv); got != envValueDisabled {
		t.Fatalf("child build environment = %q, want false", got)
	}
	restore()
	if got := os.Getenv(dagBuildEnv); got != boolStrTrue {
		t.Fatalf("restored environment = %q, want true", got)
	}
}
