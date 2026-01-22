// Package main provides the entry point for the markata-go CLI.
package main

import (
	"fmt"
	"os"

	"github.com/WaylonWalker/markata-go/cmd/markata-go/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
