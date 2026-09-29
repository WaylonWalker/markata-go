package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var builderAdminDAG bool

func init() {
	builderAdminCmd.Flags().BoolVar(&builderAdminDAG, "dag", false, "run queued builds with the experimental serial DAG executor")
	legacyRun := builderAdminCmd.RunE
	builderAdminCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if !builderAdminDAG {
			return legacyRun(cmd, args)
		}
		restore, err := enableBuilderAdminDAGEnvironment()
		if err != nil {
			return err
		}
		defer restore()
		return legacyRun(cmd, args)
	}
}

// enableBuilderAdminDAGEnvironment keeps the executor selection in the
// builder-admin process environment for the service lifetime. Builder Admin's
// queued build subprocesses inherit os.Environ(), so each `markata-go build`
// sees the same explicit opt-in without altering queue or release semantics.
func enableBuilderAdminDAGEnvironment() (func(), error) {
	previous, existed := os.LookupEnv(dagBuildEnv)
	if err := os.Setenv(dagBuildEnv, boolStrTrue); err != nil {
		return nil, fmt.Errorf("enable builder-admin DAG executor: %w", err)
	}
	return func() {
		if existed {
			_ = os.Setenv(dagBuildEnv, previous)
			return
		}
		_ = os.Unsetenv(dagBuildEnv)
	}, nil
}
