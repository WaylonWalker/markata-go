package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/WaylonWalker/markata-go/pkg/lsp"
	"github.com/spf13/cobra"
)

// lspCmd represents the lsp command.
var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start the Language Server Protocol server",
	Long: `Start the markata-go LSP server for IDE integration.

The LSP server provides IDE features for markdown files with wikilink support:
  - Autocomplete for [[wikilinks]] - type [[ to get suggestions
  - Diagnostics for broken wikilinks - warnings for links to missing posts
  - Hover information - see post title and description on hover
  - Go to definition - Ctrl+click to navigate to linked posts

The server communicates over stdin/stdout using the Language Server Protocol.

Example usage with VS Code:
  1. Install the markata-go extension
  2. The extension will automatically start this server

Example usage with Neovim (nvim-lspconfig):
  require('lspconfig').markata.setup{
    cmd = { "markata-go", "lsp" },
    filetypes = { "markdown" },
  }

Example usage with other editors:
  Configure your editor's LSP client to run "markata-go lsp" for markdown files.`,
	RunE: runLSPCommand,
}

func init() {
	rootCmd.AddCommand(lspCmd)
}

func runLSPCommand(_ *cobra.Command, _ []string) error {
	// Setup logging to stderr (stdout is used for LSP communication)
	var logger *log.Logger
	if verbose {
		logger = log.New(os.Stderr, "[markata-lsp] ", log.LstdFlags|log.Lshortfile)
	} else {
		logger = log.New(os.Stderr, "[markata-lsp] ", log.LstdFlags)
	}

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Println("Received shutdown signal")
		cancel()
	}()

	// Create and run the LSP server
	srv := lsp.New(logger)
	if err := srv.Run(ctx, os.Stdin, os.Stdout); err != nil {
		return fmt.Errorf("LSP server error: %w", err)
	}

	return nil
}
