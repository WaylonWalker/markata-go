package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestDAGBuildEnabled(t *testing.T) {
	previous := buildDAG
	previousCommand := currentCmd
	buildDAG = false
	currentCmd = nil
	t.Cleanup(func() {
		buildDAG = previous
		currentCmd = previousCommand
	})

	t.Setenv(dagBuildEnv, envValueDisabled)
	if dagBuildEnabled() {
		t.Fatal("dagBuildEnabled() = true with flag=false and env=false")
	}

	t.Setenv(dagBuildEnv, boolStrTrue)
	if !dagBuildEnabled() {
		t.Fatal("dagBuildEnabled() = false with env=true")
	}

	t.Setenv(dagBuildEnv, "not-a-bool")
	if dagBuildEnabled() {
		t.Fatal("dagBuildEnabled() = true with invalid env value")
	}

	buildDAG = true
	t.Setenv(dagBuildEnv, envValueDisabled)
	if !dagBuildEnabled() {
		t.Fatal("explicit --dag flag did not take precedence over env=false")
	}
}

func TestDAGExplicitFlagOverridesEnvironment(t *testing.T) {
	previousCommand := currentCmd
	previous := buildDAG
	buildDAG = false
	t.Cleanup(func() {
		currentCmd = previousCommand
		buildDAG = previous
	})
	for _, test := range []struct {
		name string
		env  string
		flag string
		want bool
	}{
		{"disable inherited opt-in", "true", "false", false},
		{"explicit opt-in", "false", "true", true},
		{"disable invalid environment", "invalid", "false", false},
		{"enable invalid environment", "invalid", "true", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(dagBuildEnv, test.env)
			cmd := &cobra.Command{Use: "build"}
			cmd.Flags().Bool("dag", false, "")
			if err := cmd.Flags().Set("dag", test.flag); err != nil {
				t.Fatal(err)
			}
			currentCmd = cmd
			if err := validateDAGEnvironment(cmd); err != nil {
				t.Fatalf("explicit flag did not override environment: %v", err)
			}
			if got := dagBuildEnabled(); got != test.want {
				t.Fatalf("dagBuildEnabled() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestDAGEnvironmentValidation(t *testing.T) {
	for _, value := range []string{"", "false", " true ", "1", "0"} {
		t.Setenv(dagBuildEnv, value)
		if err := validateDAGEnvironment(nil); err != nil {
			t.Fatalf("valid environment %q rejected: %v", value, err)
		}
	}
	t.Setenv(dagBuildEnv, "invalid")
	if err := validateDAGEnvironment(nil); err == nil {
		t.Fatal("invalid environment was silently ignored")
	}
}
