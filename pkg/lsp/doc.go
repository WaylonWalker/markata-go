// Package lsp provides a Language Server Protocol (LSP) implementation for markata-go.
//
// # Overview
//
// The LSP server enables IDE features for markdown files with wikilink support:
//   - Autocomplete: Type [[ to get suggestions for post slugs
//   - Diagnostics: Warnings for broken [[wikilinks]] that don't resolve to a post
//   - Hover: Show post title and description when hovering over a wikilink
//   - Go to Definition: Navigate to the target post file (Ctrl+click)
//
// # Server
//
// The Server type is the main LSP server implementation. It communicates over
// stdin/stdout using the Language Server Protocol (JSON-RPC 2.0).
//
// Example usage:
//
//	srv := lsp.New(logger)
//	srv.Run(ctx, os.Stdin, os.Stdout)
//
// # Index
//
// The Index type maintains an in-memory index of all markdown posts in the workspace.
// It tracks:
//   - Post slugs and file paths
//   - Titles and descriptions
//   - Wikilinks contained in each file
//
// The index is built when the server initializes and updated incrementally
// as files change.
//
// # Protocol Support
//
// The server implements the following LSP methods:
//
// Lifecycle:
//   - initialize
//   - initialized
//   - shutdown
//   - exit
//
// Document Synchronization:
//   - textDocument/didOpen
//   - textDocument/didChange
//   - textDocument/didClose
//   - textDocument/didSave
//   - workspace/didChangeWatchedFiles
//
// Language Features:
//   - textDocument/completion
//   - textDocument/hover
//   - textDocument/definition
//   - textDocument/publishDiagnostics (server->client notification)
//
// # Editor Integration
//
// For VS Code, create an extension that starts "markata-go lsp".
//
// For Neovim with nvim-lspconfig:
//
//	require('lspconfig').markata.setup{
//	  cmd = { "markata-go", "lsp" },
//	  filetypes = { "markdown" },
//	}
//
// For other editors, configure the LSP client to run "markata-go lsp"
// for markdown files.
package lsp
