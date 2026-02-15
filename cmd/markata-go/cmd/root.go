// Package cmd provides the CLI commands for markata-go.
package cmd

import (
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/spf13/cobra"
)

var (
	// cfgFile is the path to the config file specified via --config flag.
	cfgFile string

	// mergeConfigFiles is a list of additional config files to merge with the base config.
	// These are applied in order, with later files taking precedence over earlier ones.
	mergeConfigFiles []string

	// outputDir is the output directory specified via --output flag.
	outputDir string

	// verbose enables verbose output.
	verbose bool

	// cpuProfile is the path to write CPU profile data.
	cpuProfile string

	// memProfile is the path to write memory profile data.
	memProfile string

	// cpuProfileFile holds the open CPU profile file for cleanup.
	cpuProfileFile *os.File
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
  markata-go config show     # Show resolved configuration

Profiling:
  markata-go build --cpuprofile cpu.prof   # Write CPU profile
  markata-go build --memprofile mem.prof   # Write memory profile

  # Analyze with:
  go tool pprof cpu.prof
  go tool pprof -http=:8080 cpu.prof`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       Version,
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		// Start CPU profiling if requested
		if cpuProfile != "" {
			f, err := os.Create(cpuProfile)
			if err != nil {
				return fmt.Errorf("failed to create CPU profile: %w", err)
			}
			cpuProfileFile = f
			if err := pprof.StartCPUProfile(f); err != nil {
				f.Close()
				return fmt.Errorf("failed to start CPU profile: %w", err)
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "CPU profiling enabled, writing to %s\n", cpuProfile)
			}
		}
		return nil
	},
	PersistentPostRunE: func(_ *cobra.Command, _ []string) error {
		// Stop CPU profiling
		if cpuProfileFile != nil {
			pprof.StopCPUProfile()
			cpuProfileFile.Close()
			if verbose {
				fmt.Fprintf(os.Stderr, "CPU profile written to %s\n", cpuProfile)
			}
		}

		// Write memory profile if requested
		if memProfile != "" {
			f, err := os.Create(memProfile)
			if err != nil {
				return fmt.Errorf("failed to create memory profile: %w", err)
			}
			defer f.Close()

			// Get the heap profile (most useful for memory analysis)
			if err := pprof.WriteHeapProfile(f); err != nil {
				return fmt.Errorf("failed to write memory profile: %w", err)
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "Memory profile written to %s\n", memProfile)
			}
		}
		return nil
	},
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
	rootCmd.PersistentFlags().StringSliceVarP(&mergeConfigFiles, "merge-config", "m", nil, "additional config file(s) to merge with base config (can be specified multiple times)")
	rootCmd.PersistentFlags().StringVarP(&outputDir, "output", "o", "", "output directory (overrides config)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Profiling flags
	rootCmd.PersistentFlags().StringVar(&cpuProfile, "cpuprofile", "", "write CPU profile to file")
	rootCmd.PersistentFlags().StringVar(&memProfile, "memprofile", "", "write memory profile to file")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Config initialization is handled by the core package when needed
	if verbose {
		fmt.Fprintln(os.Stderr, "Verbose mode enabled")
	}
}
