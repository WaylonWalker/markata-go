package cmd

import "github.com/spf13/cobra"

// adminCmd opens the local Serve Control Center. Production Builder Admin
// remains available through the builder-admin command and its proxy auth.
var adminCmd = &cobra.Command{
	Use:     "admin",
	Aliases: []string{"a"},
	Short:   "Start the local Builder Admin control center",
	Long:    "Build and serve the current site with its local jobs, logs, and page diagnostics dashboard at /_markata/.",
	Example: "markata-go admin\nmarkata-go admin --port 3000",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		serveAdmin = true
		serveNoTUI = true
		return runServeCommand(cmd, nil)
	},
}

func init() {
	rootCmd.AddCommand(adminCmd)
	adminCmd.Flags().IntVarP(&servePort, "port", "p", 8000, "port to serve on")
	adminCmd.Flags().StringVar(&serveHost, "host", "localhost", "loopback host to serve on")
	adminCmd.Flags().BoolVar(&serveWatch, "watch", true, "enable file watching")
	adminCmd.Flags().BoolVar(&serveFast, "fast", false, "skip minification and CSS purging")
	adminCmd.Flags().BoolVar(&serveIncremental, "incremental", false, "reuse unchanged posts")
}
