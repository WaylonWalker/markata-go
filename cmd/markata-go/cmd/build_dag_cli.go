package cmd

import (
	"fmt"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/spf13/cobra"
)

var buildDAG bool

func init() {
	buildCmd.Flags().BoolVar(&buildDAG, "dag", false, "use the experimental serial DAG executor")
	legacyRun := buildCmd.RunE
	buildCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if !buildDAG {
			return legacyRun(cmd, args)
		}
		return runDAGBuildCommand(args)
	}
}

// runDAGBuildCommand mirrors the existing build command boundary while routing
// only the lifecycle execution step through the feature-flagged DAG executor.
// Keeping this path separate makes removal of the experiment trivial while the
// graph is still being proven by Build Lab.
func runDAGBuildCommand(args []string) error {
	startTime := time.Now()
	verbosef("Starting build with experimental serial DAG executor...")

	var (
		m   *lifecycle.Manager
		err error
	)
	if len(args) == 1 {
		m, err = createSinglePageManager(cfgFile, args[0])
	} else {
		m, err = createManager(cfgFile)
	}
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}
	configureLoggerForManager(m)

	if buildFast {
		applyFastMode(m)
	}
	verbosef("Configuration loaded (output: %s, patterns: %v)", m.Config().OutputDir, m.Config().GlobPatterns)

	if buildCleanAll || buildClean {
		if err := cleanBuildDirs(m); err != nil {
			return err
		}
	}
	if buildDryRun {
		return runDryBuild(m)
	}

	result, err := runDAGBuild(m)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	result.Duration = time.Since(startTime).Seconds()

	if buildBenchmarkJSON == "-" {
		if err := writeBenchmarkJSON(outWriter(), result); err != nil {
			return fmt.Errorf("writing benchmark json: %w", err)
		}
	} else {
		printBuildResult(result)
		outlnf("  %s", buildLabel("Executor: serial DAG (experimental)"))
	}
	if buildBenchmarkJSON != "" && buildBenchmarkJSON != "-" {
		if err := writeBenchmarkJSONFile(buildBenchmarkJSON, result); err != nil {
			return fmt.Errorf("writing benchmark json file: %w", err)
		}
	}
	if len(result.Warnings) > 0 && verbose {
		errln("\nWarnings:")
		for _, warning := range result.Warnings {
			errlnf("  - %s", warning)
		}
	}
	return nil
}
