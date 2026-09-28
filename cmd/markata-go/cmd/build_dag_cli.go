package cmd

import (
	"os"
	"strconv"
	"strings"
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
