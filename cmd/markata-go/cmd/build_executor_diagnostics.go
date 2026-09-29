package cmd

import "github.com/spf13/cobra"

const (
	legacyExecutorDiagnostic = "legacy lifecycle"
	dagExecutorDiagnostic    = "serial DAG (experimental)"
)

func init() {
	wrapExecutorDiagnostic(buildCmd, func() bool { return dagBuildEnabled() }, true)
	wrapExecutorDiagnostic(serveCmd, func() bool { return buildDAG }, false)
	wrapExecutorDiagnostic(builderAdminCmd, func() bool { return builderAdminDAG }, false)
}

func wrapExecutorDiagnostic(cmd *cobra.Command, dagEnabled func() bool, preserveJSONStdout bool) {
	previousPreRunE := cmd.PreRunE
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if previousPreRunE != nil {
			if err := previousPreRunE(cmd, args); err != nil {
				return err
			}
		}

		name, experimental := executorDiagnostic(dagEnabled())
		if experimental {
			// Keep diagnostics on stderr. This is required for
			// `build --benchmark-json -`, and is also the right stream for
			// long-lived serve/admin process startup notices.
			errlnf("Build executor: %s", name)
		} else if !preserveJSONStdout || verbose {
			verbosef("Build executor: %s", name)
		}
		return nil
	}
}

func selectedBuildExecutorDiagnostic() (name string, experimental bool) {
	return executorDiagnostic(dagBuildEnabled())
}

func executorDiagnostic(dagEnabled bool) (name string, experimental bool) {
	if dagEnabled {
		return dagExecutorDiagnostic, true
	}
	return legacyExecutorDiagnostic, false
}
