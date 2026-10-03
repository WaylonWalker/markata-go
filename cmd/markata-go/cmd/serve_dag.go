package cmd

func init() {
	// Share the same opt-in bit as `build --dag`. Serve's initial build and all
	// rebuilds already flow through runBuildObserved, so the Control Center keeps
	// the same stage/job reporting while only the executor changes.
	serveCmd.Flags().BoolVar(&buildDAG, "dag", false, "use the experimental serial DAG executor for initial builds and rebuilds")
}
