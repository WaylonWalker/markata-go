// Package main provides the entry point for the markata-go CLI.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/WaylonWalker/markata-go/cmd/markata-go/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		exitCode := cmd.ExitCodeForError(err)
		if message := strings.TrimSpace(err.Error()); message != "" {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(exitCode)
	}
}
