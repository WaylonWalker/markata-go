package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const dagBuildEnv = "MARKATA_GO_DAG"

var buildDAG bool

func init() {
	buildCmd.Flags().BoolVar(&buildDAG, "dag", false, "use the experimental serial DAG executor")
}

// dagBuildEnabled lets long-lived commands such as builder-admin pass the
// experimental executor choice to child build processes without changing the
// normal default. An explicit build --dag flag and MARKATA_GO_DAG=true are
// equivalent opt-ins.
func dagBuildEnabled() bool {
	if enabled, explicit := explicitDAGSelection(currentCmd); explicit {
		return enabled
	}
	if buildDAG {
		return true
	}
	value := strings.TrimSpace(os.Getenv(dagBuildEnv))
	if value == "" {
		return false
	}
	enabled, err := strconv.ParseBool(value)
	return err == nil && enabled
}

func explicitDAGSelection(cmd *cobra.Command) (enabled, explicit bool) {
	if cmd == nil {
		return false, false
	}
	flag := cmd.Flags().Lookup("dag")
	if flag == nil || !flag.Changed {
		return false, false
	}
	return flag.Value.String() == boolStrTrue, true
}

func validateDAGEnvironment(cmd *cobra.Command) error {
	if _, explicit := explicitDAGSelection(cmd); explicit {
		return nil
	}
	value := strings.TrimSpace(os.Getenv(dagBuildEnv))
	if value == "" {
		return nil
	}
	if _, err := strconv.ParseBool(value); err != nil {
		return fmt.Errorf("%s must be a boolean: %w", dagBuildEnv, err)
	}
	return nil
}
