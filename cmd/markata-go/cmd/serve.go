package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

// HTTP server timeout constants.
const (
	serverReadHeaderTimeout = 10 * time.Second
)

// searchResult represents a search result for the no-JS fallback search.
type searchResult struct {
	Title       string
	Description string
	URL         string
	Score       int
}

var (
	// servePort is the port to serve on.
	servePort int

	// serveHost is the host to serve on.
	serveHost string

	// serveWatch enables file watching (default true).
	serveWatch bool

	// serveNoWatch disables file watching (legacy flag for backward compatibility).
	serveNoWatch bool

	// serveOutputPath is the output directory path for filtering watch events.
	serveOutputPath string

	// isRebuilding tracks whether a rebuild is in progress to avoid event loops.
	isRebuilding atomic.Bool

	// rebuildPending tracks whether changes happened during a rebuild.
	rebuildPending atomic.Bool
)

// serveCmd represents the serve command.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Build and serve locally with live reload",
	Long: `Serve builds the site and starts a local development server with file watching.

Features:
  - Initial build of the site
  - HTTP server serving the output directory
  - File watching for content, templates, and config
  - Automatic rebuild on file changes
  - Live reload support (injects reload script into HTML)

Example usage:
  markata-go serve              # Serve on localhost:8000 with file watching
  markata-go serve -p 3000      # Serve on localhost:3000
  markata-go serve --watch      # Explicitly enable file watching (default)
  markata-go serve --watch=false # Disable file watching
  markata-go serve --no-watch   # Serve without file watching (legacy flag)
  markata-go serve -v           # Serve with verbose output`,
	RunE: runServeCommand,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8000, "port to serve on")
	serveCmd.Flags().StringVar(&serveHost, "host", "localhost", "host to serve on")
	serveCmd.Flags().BoolVar(&serveWatch, "watch", true, "enable file watching")
	serveCmd.Flags().BoolVar(&serveNoWatch, "no-watch", false, "disable file watching (legacy, overrides --watch)")
}

func runServeCommand(_ *cobra.Command, _ []string) error {
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupt received - shutting down...")
		if isRebuilding.Load() {
			fmt.Println("Rebuild in progress - canceling...")
		}
		cancel()
	}()

	// Initial build
	fmt.Println("Running initial build...")
	m, err := createManager(cfgFile)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	result, err := runBuild(m)
	if err != nil {
		return fmt.Errorf("initial build failed: %w", err)
	}

	fmt.Printf("Built %d posts, %d feeds\n", result.PostsProcessed, result.FeedsGenerated)

	// Determine output directory
	outputPath := m.Config().OutputDir
	if outputPath == "" {
		outputPath = "output"
	}
	// Store the absolute output path for watch filtering
	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		absOutputPath = outputPath
	}
	serveOutputPath = absOutputPath

	// Start file watcher if enabled
	// Watch is enabled if: --watch is true (default) AND --no-watch is false
	// --no-watch takes precedence for backward compatibility
	shouldWatch := serveWatch && !serveNoWatch
	var watcher *fsnotify.Watcher
	var rebuildCh chan struct{}
	var wg sync.WaitGroup

	if shouldWatch {
		watcher, err = fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("failed to create file watcher: %w", err)
		}
		defer watcher.Close()

		rebuildCh = make(chan struct{}, 1)

		// Start watcher goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			watchFiles(ctx, watcher, rebuildCh)
		}()

		// Start rebuild goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			handleRebuilds(ctx, rebuildCh)
		}()

		// Add paths to watch
		if err := addWatchPaths(watcher, m); err != nil {
			return fmt.Errorf("failed to setup file watching: %w", err)
		}

		fmt.Println("Watching for file changes...")
	}

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", serveHost, servePort)
	handler := createHandler(outputPath)

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: serverReadHeaderTimeout,
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	serverStarted := make(chan struct{})
	go func() {
		fmt.Printf("\nServing at http://%s\n", addr)
		fmt.Println("Press Ctrl+C to stop")
		close(serverStarted) // Signal that server is ready
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for server to start before entering select
	<-serverStarted

	// Wait for shutdown or error
	select {
	case <-ctx.Done():
		// Graceful shutdown
		fmt.Println("Initiating graceful shutdown...")

		// Close live reload connections first
		closeAllLiveReloadConnections()

		// Shorter timeout for faster shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer shutdownCancel()

		fmt.Printf("Shutting down HTTP server (timeout: 2s)...\n")
		if err := server.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("Server shutdown error: %v\n", err)
		} else {
			fmt.Println("HTTP server shutdown completed")
		}
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}

	// Wait for goroutines to finish with timeout
	activeConnections := liveReloadCount.Load()
	if verbose || activeConnections > 0 {
		fmt.Printf("Waiting for goroutines to finish (active SSE connections: %d)...\n", activeConnections)
	} else {
		fmt.Println("Waiting for goroutines to finish...")
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("All tracked goroutines finished")
	case <-time.After(1 * time.Second):
		fmt.Printf("Shutdown timeout after 1s - forcing exit\n")
		if activeConnections > 0 {
			fmt.Printf("Note: Had %d active SSE connections that may not have closed cleanly\n", activeConnections)
		}
	}

	fmt.Println("Server stopped")
	return nil
}

// createHandler creates an HTTP handler that serves files with live reload injection.
func createHandler(outputDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(outputDir))
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		absOutputDir = outputDir
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log requests in verbose mode
		if verbose {
			fmt.Printf("[%s] %s %s\n", time.Now().Format("15:04:05"), r.Method, r.URL.Path)
		}

		// Handle live reload endpoint
		if r.URL.Path == "/__livereload" {
			handleLiveReload(w, r)
			return
		}

		// Handle search endpoint (no-JS fallback for 404 page search)
		if r.URL.Path == "/_search" && r.Method == http.MethodPost {
			handleSearchFallback(w, r, outputDir)
			return
		}

		// Determine the file path
		fullPath, requestPath, resolveErr := resolveRequestPath(absOutputDir, r.URL.Path)
		if resolveErr != nil {
			serve404Page(w, outputDir)
			return
		}

		// Check if file exists
		info, err := os.Stat(fullPath)
		if err == nil && info.IsDir() {
			// Try index.html in directory
			indexPath := filepath.Join(fullPath, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				requestPath = path.Join(requestPath, "index.html")
				fullPath = indexPath
			}
		}

		// Check if file exists - if not, serve 404 page
		if err != nil && os.IsNotExist(err) {
			serve404Page(w, outputDir)
			return
		}

		// Check if it's an HTML file and inject live reload script
		if strings.HasSuffix(requestPath, ".html") || (info != nil && !info.IsDir() && strings.HasSuffix(fullPath, ".html")) {
			serveHTMLWithLiveReload(w, fullPath, outputDir)
			return
		}

		// Serve with file server
		r.URL.Path = requestPath
		fileServer.ServeHTTP(w, r)
	})
}

// serveHTMLWithLiveReload reads an HTML file and injects the live reload script.
func serveHTMLWithLiveReload(w http.ResponseWriter, path, outputDir string) {
	content, err := os.ReadFile(path)
	if err != nil {
		// File not found - serve 404 page
		serve404Page(w, outputDir)
		return
	}

	// Inject live reload script before </body>
	liveReloadScript := `<script>
(function() {
    var source = new EventSource('/__livereload');
    source.onmessage = function(e) {
        if (e.data === 'reload') {
            location.reload();
        }
    };
    source.onerror = function() {
        source.close();
        setTimeout(function() {
            location.reload();
        }, 1000);
    };
})();
</script>`

	html := string(content)
	switch {
	case strings.Contains(html, "</body>"):
		html = strings.Replace(html, "</body>", liveReloadScript+"</body>", 1)
	case strings.Contains(html, "</html>"):
		html = strings.Replace(html, "</html>", liveReloadScript+"</html>", 1)
	default:
		html += liveReloadScript
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(html)); err != nil && verbose {
		fmt.Printf("Error writing response: %v\n", err)
	}
}

// serve404Page serves the static 404.html page with live reload injection.
// The 404 page uses client-side JavaScript for fuzzy search suggestions,
// so it works the same in dev server as in production.
func serve404Page(w http.ResponseWriter, outputDir string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)

	// Try to serve the static 404.html
	notFoundPath := filepath.Join(outputDir, "404.html")
	content, err := os.ReadFile(notFoundPath)
	if err != nil {
		// Fallback to simple error message if 404.html doesn't exist
		//nolint:errcheck // Best effort write to HTTP response
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>404 - Page Not Found</title></head>
<body>
<h1>404 - Page Not Found</h1>
<p>The requested page could not be found.</p>
<p><a href="/">Go to home page</a></p>
</body>
</html>`))
		return
	}

	// Inject live reload script
	liveReloadScript := `<script>
(function() {
    var source = new EventSource('/__livereload');
    source.onmessage = function(e) {
        if (e.data === 'reload') {
            location.reload();
        }
    };
    source.onerror = function() {
        source.close();
        setTimeout(function() {
            location.reload();
        }, 1000);
    };
})();
</script>`

	html := string(content)
	switch {
	case strings.Contains(html, "</body>"):
		html = strings.Replace(html, "</body>", liveReloadScript+"</body>", 1)
	case strings.Contains(html, "</html>"):
		html = strings.Replace(html, "</html>", liveReloadScript+"</html>", 1)
	default:
		html += liveReloadScript
	}

	//nolint:errcheck // Best effort write to HTTP response
	w.Write([]byte(html))
}

// handleSearchFallback handles POST requests to /_search for no-JS fallback search.
// It reads the posts index, performs fuzzy search, and renders results as HTML.
func handleSearchFallback(w http.ResponseWriter, r *http.Request, outputDir string) {
	// Parse the form to get the search query
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	query := strings.TrimSpace(r.FormValue("q"))
	if query == "" {
		// No query, redirect to home
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Load posts index
	indexPath := filepath.Join(outputDir, "_404-index.json")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		// Index not available, redirect to home with query param
		http.Redirect(w, r, "/?q="+query, http.StatusSeeOther)
		return
	}

	// Parse the index
	var posts []struct {
		Slug        string `json:"slug"`
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
	}
	if err := json.Unmarshal(indexData, &posts); err != nil {
		http.Redirect(w, r, "/?q="+query, http.StatusSeeOther)
		return
	}

	// Perform simple search (case-insensitive substring matching + basic scoring)
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)
	var results []searchResult

	for _, post := range posts {
		titleLower := strings.ToLower(post.Title)
		slugLower := strings.ToLower(post.Slug)
		descLower := strings.ToLower(post.Description)

		score := 0
		for _, word := range queryWords {
			if strings.Contains(titleLower, word) {
				score += 10
			}
			if strings.Contains(slugLower, word) {
				score += 8
			}
			if strings.Contains(descLower, word) {
				score += 5
			}
		}

		if score > 0 {
			results = append(results, searchResult{
				Title:       post.Title,
				Description: post.Description,
				URL:         post.URL,
				Score:       score,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if len(results) > 20 {
		results = results[:20]
	}

	// Render search results page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderSearchResultsPage(w, query, results)
}

// renderSearchResultsPage renders HTML for search results (no-JS fallback).
func renderSearchResultsPage(w http.ResponseWriter, query string, results []searchResult) {
	var html strings.Builder
	html.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Search Results</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 2rem 1rem;
            background: #f9fafb;
            color: #1f2937;
        }
        @media (prefers-color-scheme: dark) {
            body { background: #111827; color: #f3f4f6; }
            .search-input { background: #1f2937; border-color: #374151; color: #f3f4f6; }
            .result-item { background: #1f2937; border-color: #374151; }
            .result-item:hover { background: #374151; }
        }
        h1 { font-size: 1.5rem; margin-bottom: 1.5rem; }
        .search-form { display: flex; gap: 0.5rem; margin-bottom: 2rem; }
        .search-input {
            flex: 1;
            padding: 0.75rem 1rem;
            font-size: 1rem;
            border: 2px solid #e5e7eb;
            border-radius: 0.5rem;
        }
        .search-button {
            padding: 0.75rem 1.5rem;
            font-size: 1rem;
            font-weight: 600;
            color: white;
            background: #3b82f6;
            border: none;
            border-radius: 0.5rem;
            cursor: pointer;
        }
        .search-button:hover { background: #2563eb; }
        .results-count { color: #6b7280; margin-bottom: 1rem; }
        .result-item {
            display: block;
            padding: 1rem;
            margin-bottom: 0.5rem;
            background: white;
            border: 1px solid #e5e7eb;
            border-radius: 0.5rem;
            text-decoration: none;
            color: inherit;
            transition: background 0.2s, transform 0.2s;
        }
        .result-item:hover { background: #f3f4f6; transform: translateX(4px); }
        .result-title { display: block; font-weight: 600; color: #3b82f6; margin-bottom: 0.25rem; }
        .result-desc { display: block; font-size: 0.875rem; color: #6b7280; }
        .no-results { color: #6b7280; font-style: italic; }
        .back-link { display: inline-block; margin-top: 1.5rem; color: #3b82f6; }
    </style>
</head>
<body>
    <h1>Search Results</h1>
    <form class="search-form" action="/_search" method="POST">
        <input type="text" name="q" class="search-input" value="`)
	html.WriteString(template.HTMLEscapeString(query))
	html.WriteString(`" placeholder="Search..." autocomplete="off">
        <button type="submit" class="search-button">Search</button>
    </form>
`)

	if len(results) == 0 {
		html.WriteString(`    <p class="no-results">No results found for "`)
		html.WriteString(template.HTMLEscapeString(query))
		html.WriteString(`"</p>
`)
	} else {
		html.WriteString(fmt.Sprintf(`    <p class="results-count">Found %d result(s) for "%s"</p>
`, len(results), template.HTMLEscapeString(query)))

		for _, result := range results {
			html.WriteString(`    <a href="`)
			html.WriteString(template.HTMLEscapeString(result.URL))
			html.WriteString(`" class="result-item">
        <span class="result-title">`)
			html.WriteString(template.HTMLEscapeString(result.Title))
			html.WriteString(`</span>
`)
			if result.Description != "" {
				desc := result.Description
				if len(desc) > 150 {
					desc = desc[:150] + "..."
				}
				html.WriteString(`        <span class="result-desc">`)
				html.WriteString(template.HTMLEscapeString(desc))
				html.WriteString(`</span>
`)
			}
			html.WriteString(`    </a>
`)
		}
	}

	html.WriteString(`    <a href="/" class="back-link">&larr; Back to home</a>
</body>
</html>`)

	//nolint:errcheck // Best effort write to HTTP response
	w.Write([]byte(html.String()))
}

// Live reload clients
var (
	liveReloadClients    = make(map[chan string]struct{})
	liveReloadClientsMu  sync.RWMutex
	liveReloadCount      atomic.Int32
	liveReloadShutdown   = make(chan struct{})
	liveReloadShutdownMu sync.Mutex
)

// handleLiveReload handles Server-Sent Events for live reload.
func handleLiveReload(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create channel for this client
	ch := make(chan string, 1)

	// Register client
	liveReloadClientsMu.Lock()
	liveReloadClients[ch] = struct{}{}
	liveReloadCount.Add(1)
	liveReloadClientsMu.Unlock()

	if verbose {
		fmt.Printf("Live reload client connected (total: %d)\n", liveReloadCount.Load())
	}

	// Ensure client is removed on disconnect
	defer func() {
		liveReloadClientsMu.Lock()
		delete(liveReloadClients, ch)
		liveReloadCount.Add(-1)
		// Close channel if not already closed
		select {
		case <-ch:
			// Channel already closed
		default:
			close(ch)
		}
		liveReloadClientsMu.Unlock()

		if verbose {
			fmt.Printf("Live reload client disconnected (total: %d)\n", liveReloadCount.Load())
		}
	}()

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection message
	fmt.Fprintf(w, "data: connected\n\n")
	flusher.Flush()

	// Wait for messages or disconnect
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				// Channel closed by global shutdown
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		case <-liveReloadShutdown:
			if verbose {
				fmt.Printf("SSE handler received global shutdown signal\n")
			}
			return
		}
	}
}

// notifyLiveReload sends a reload message to all connected clients.
func notifyLiveReload() {
	liveReloadClientsMu.RLock()
	defer liveReloadClientsMu.RUnlock()

	for ch := range liveReloadClients {
		select {
		case ch <- "reload":
		case <-ch:
			// Channel closed, skip
		default:
			// Skip if channel is full
		}
	}
}

// closeAllLiveReloadConnections closes all SSE connections for shutdown.
func closeAllLiveReloadConnections() {
	liveReloadShutdownMu.Lock()
	defer liveReloadShutdownMu.Unlock()

	liveReloadClientsMu.Lock()
	defer liveReloadClientsMu.Unlock()

	count := len(liveReloadClients)
	if count > 0 {
		fmt.Printf("Closing %d live reload connection(s)...\n", count)

		// Signal global shutdown - this will cause all SSE handlers to exit
		select {
		case <-liveReloadShutdown:
			// Already closed
		default:
			close(liveReloadShutdown)
		}

		// Close all client channels to force immediate exit
		for ch := range liveReloadClients {
			select {
			case <-ch:
				// Channel already closed
			default:
				close(ch)
			}
		}

		// Clear the map
		clear(liveReloadClients)
		liveReloadCount.Store(0)
		fmt.Printf("Closed all live reload connections\n")
	}
}

// watchFiles handles file system events.
func watchFiles(ctx context.Context, watcher *fsnotify.Watcher, rebuildCh chan<- struct{}) {
	for {
		select {
		case <-ctx.Done():
			if verbose {
				fmt.Println("File watcher canceled")
			}
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Get absolute path for comparison
			absEventPath, err := filepath.Abs(event.Name)
			if err != nil {
				absEventPath = event.Name
			}

			// Ignore events for output directory
			if isPathWithinDir(absEventPath, serveOutputPath) {
				continue
			}

			// Ignore temporary/backup files and hidden files
			baseName := filepath.Base(event.Name)
			if strings.HasSuffix(event.Name, "~") ||
				strings.HasPrefix(baseName, ".") ||
				strings.HasSuffix(event.Name, ".swp") ||
				strings.HasSuffix(event.Name, ".swo") ||
				strings.HasSuffix(event.Name, ".tmp") {
				continue
			}

			// Trigger on write/create/remove/rename
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				if verbose {
					fmt.Printf("File changed: %s\n", event.Name)
				}

				if isRebuilding.Load() {
					rebuildPending.Store(true)
					continue
				}

				// Debounce rebuilds
				select {
				case rebuildCh <- struct{}{}:
				default:
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("Watcher error: %v\n", err)
		}
	}
}

// handleRebuilds processes rebuild requests with debouncing.
func handleRebuilds(ctx context.Context, rebuildCh chan struct{}) {
	// Debounce timer
	var timer *time.Timer
	debounceDelay := 300 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				if !timer.Stop() {
					drainTimer(timer)
				}
			}
			if verbose {
				fmt.Println("Rebuild handler canceled")
			}
			return
		case <-rebuildCh:
			// Reset debounce timer
			if timer != nil {
				if !timer.Stop() {
					drainTimer(timer)
				}
				timer.Reset(debounceDelay)
				continue
			}
			timer = time.NewTimer(debounceDelay)
		case <-timerChannel(timer):
			if timer != nil {
				timer.Stop()
				timer = nil
			}
			if ctx.Err() != nil {
				return
			}
			doRebuild(ctx, rebuildCh)
		}
	}
}

// doRebuild performs an incremental rebuild.
func doRebuild(ctx context.Context, rebuildCh chan<- struct{}) {
	// Set rebuilding flag to ignore events during build
	isRebuilding.Store(true)
	defer func() {
		isRebuilding.Store(false)
		if rebuildPending.Swap(false) {
			if ctx.Err() != nil {
				return
			}
			select {
			case rebuildCh <- struct{}{}:
			default:
			}
		}
	}()

	fmt.Println("\nRebuilding...")
	startTime := time.Now()

	// Check if context is canceled before starting rebuild
	select {
	case <-ctx.Done():
		fmt.Println("Rebuild canceled")
		return
	default:
	}

	m, err := createManager(cfgFile)
	if err != nil {
		fmt.Printf("Rebuild failed: %v\n", err)
		return
	}

	// Check for cancellation after creating manager
	select {
	case <-ctx.Done():
		fmt.Println("Rebuild canceled")
		return
	default:
	}

	result, err := runBuild(m)
	if err != nil {
		fmt.Printf("Rebuild failed: %v\n", err)
		return
	}

	// Check for cancellation after build
	select {
	case <-ctx.Done():
		fmt.Println("Rebuild canceled")
		return
	default:
	}

	duration := time.Since(startTime)
	fmt.Printf("Rebuilt in %.2fs (%d posts, %d feeds)\n",
		duration.Seconds(), result.PostsProcessed, result.FeedsGenerated)

	// Notify live reload clients
	notifyLiveReload()
}

// addWatchPaths adds paths to the file watcher.
func addWatchPaths(watcher *fsnotify.Watcher, m *lifecycle.Manager) error {
	config := m.Config()

	// Watch current directory for config files
	if err := watcher.Add("."); err != nil {
		return err
	}

	// Watch content directory
	contentDir := config.ContentDir
	if contentDir == "" {
		contentDir = "."
	}
	if err := addDirRecursive(watcher, contentDir); err != nil {
		return err
	}

	// Watch templates directory
	templatesDir := "templates"
	if td, ok := config.Extra["templates_dir"].(string); ok && td != "" {
		templatesDir = td
	}
	if _, err := os.Stat(templatesDir); err == nil {
		if err := addDirRecursive(watcher, templatesDir); err != nil {
			return err
		}
	}

	// Watch static/assets directory
	assetsDir := "static"
	if ad, ok := config.Extra["assets_dir"].(string); ok && ad != "" {
		assetsDir = ad
	}
	if _, err := os.Stat(assetsDir); err == nil {
		if err := addDirRecursive(watcher, assetsDir); err != nil {
			return err
		}
	}

	return nil
}

// addDirRecursive recursively adds directories to the watcher.
func addDirRecursive(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get absolute path for comparison
		absPath, pathErr := filepath.Abs(path)
		if pathErr != nil {
			absPath = path
		}

		// Skip output directory
		if isPathWithinDir(absPath, serveOutputPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden directories
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
			return filepath.SkipDir
		}

		// Add directories to watcher
		if d.IsDir() {
			if verbose {
				fmt.Printf("Watching: %s\n", path)
			}
			return watcher.Add(path)
		}

		return nil
	})
}

func resolveRequestPath(outputDir, requestPath string) (string, string, error) {
	if requestPath == "" || requestPath == "/" {
		requestPath = "/index.html"
	}

	cleanURLPath := path.Clean("/" + requestPath)
	relPath := strings.TrimPrefix(cleanURLPath, "/")
	if relPath == "" {
		relPath = "index.html"
		cleanURLPath = "/index.html"
	}

	fullPath := filepath.Join(outputDir, filepath.FromSlash(relPath))
	if !isPathWithinDir(fullPath, outputDir) {
		return "", "", errors.New("resolved path escapes output directory")
	}

	return fullPath, cleanURLPath, nil
}

func isPathWithinDir(pathname, dir string) bool {
	if dir == "" {
		return false
	}
	if filepath.Clean(pathname) == filepath.Clean(dir) {
		return true
	}
	rel, err := filepath.Rel(dir, pathname)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func timerChannel(timer *time.Timer) <-chan time.Time {
	if timer == nil {
		return nil
	}
	return timer.C
}

func drainTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	select {
	case <-timer.C:
	default:
	}
}
