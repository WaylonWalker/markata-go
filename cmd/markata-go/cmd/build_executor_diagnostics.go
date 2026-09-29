package cmd

import "github.com/spf13/cobra"

const (
	legacyExecutorDiagnostic = "legacy lifecycle"
	dagExecutorDiagnostic    = "serial DAG (experimental)"
)

func init() {
	previousPreRunE := buildCmd.PreRunE
	buildCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if previousPreRunE != nil {
			if err := previousPreRunE(cmd, args); err != nil {
				return err
			}
		}

		name, experimental := selectedBuildExecutorDiagnostic()
		if experimental {
			// Use stderr so `build --benchmark-json -` remains valid JSON on stdout.
			errlnf("Build executor: %s", name)
		} else {
			verbosef("Build executor: %s", name)
		}
		return nil
	}
}

func selectedBuildExecutorDiagnostic() (name string, experimental bool) {
	if dagBuildEnabled() {
		return dagExecutorDiagnostic, true
	}
	return legacyExecutorDiagnostic, false
}
