// Package main provides the entry point for the markata-go CLI.
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/WaylonWalker/markata-go/cmd/markata-go/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		exitCode := 1
		var exitErr interface{ ExitCode() int }
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		if message := strings.TrimSpace(err.Error()); message != "" {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(exitCode)
	}
}
