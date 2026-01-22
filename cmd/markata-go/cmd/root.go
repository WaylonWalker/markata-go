// Package cmd provides the CLI commands for markata-go.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// cfgFile is the path to the config file specified via --config flag.
	cfgFile string

	// outputDir is the output directory specified via --output flag.
	outputDir string

	// verbose enables verbose output.
	verbose bool
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "markata-go",
	Short: "A plugin-driven static site generator",
	Long: `Markata-go is a static site generator with a powerful feed system.

It processes markdown files with YAML frontmatter and generates a static website
with support for multiple feed formats (RSS, Atom, JSON Feed), automatic archives,
tag pages, and more.

Example usage:
  markata-go build           # Build the site
  markata-go serve           # Build and serve locally with live reload
  markata-go new "My Post"   # Create a new post
  markata-go config show     # Show resolved configuration`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path (default: auto-discover)")
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "", "output directory (overrides config)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Config initialization is handled by the core package when needed
	if verbose {
		fmt.Fprintln(os.Stderr, "Verbose mode enabled")
	}
}
