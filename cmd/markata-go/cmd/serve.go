package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
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
		fmt.Println("\nShutting down...")
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
	go func() {
		fmt.Printf("\nServing at http://%s\n", addr)
		fmt.Println("Press Ctrl+C to stop")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for shutdown or error
	select {
	case <-ctx.Done():
		// Graceful shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("Server shutdown error: %v\n", err)
		}
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	}

	// Wait for goroutines to finish
	wg.Wait()

	fmt.Println("Server stopped")
	return nil
}

// createHandler creates an HTTP handler that serves files with live reload injection.
func createHandler(outputDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(outputDir))

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

		// Determine the file path
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists
		fullPath := filepath.Join(outputDir, path)
		info, err := os.Stat(fullPath)
		if err == nil && info.IsDir() {
			// Try index.html in directory
			indexPath := filepath.Join(fullPath, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				path = filepath.Join(path, "index.html")
				fullPath = indexPath
			}
		}

		// Check if it's an HTML file and inject live reload script
		if strings.HasSuffix(path, ".html") || (info != nil && !info.IsDir() && strings.HasSuffix(fullPath, ".html")) {
			serveHTMLWithLiveReload(w, fullPath)
			return
		}

		// Serve with file server
		fileServer.ServeHTTP(w, r)
	})
}

// serveHTMLWithLiveReload reads an HTML file and injects the live reload script.
func serveHTMLWithLiveReload(w http.ResponseWriter, path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
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

// Live reload clients
var (
	liveReloadClients   = make(map[chan string]struct{})
	liveReloadClientsMu sync.RWMutex
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
	liveReloadClientsMu.Unlock()

	// Ensure client is removed on disconnect
	defer func() {
		liveReloadClientsMu.Lock()
		delete(liveReloadClients, ch)
		close(ch)
		liveReloadClientsMu.Unlock()
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
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
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
		default:
			// Skip if channel is full
		}
	}
}

// watchFiles handles file system events.
func watchFiles(ctx context.Context, watcher *fsnotify.Watcher, rebuildCh chan<- struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Skip events during rebuild to avoid infinite loops
			if isRebuilding.Load() {
				continue
			}

			// Get absolute path for comparison
			absEventPath, err := filepath.Abs(event.Name)
			if err != nil {
				absEventPath = event.Name
			}

			// Ignore events for output directory
			if serveOutputPath != "" && strings.HasPrefix(absEventPath, serveOutputPath) {
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

			// Only trigger on write and create events
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				if verbose {
					fmt.Printf("File changed: %s\n", event.Name)
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
func handleRebuilds(ctx context.Context, rebuildCh <-chan struct{}) {
	// Debounce timer
	var timer *time.Timer
	debounceDelay := 300 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return
		case <-rebuildCh:
			// Reset debounce timer
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounceDelay, func() {
				doRebuild()
			})
		}
	}
}

// doRebuild performs an incremental rebuild.
func doRebuild() {
	// Set rebuilding flag to ignore events during build
	isRebuilding.Store(true)
	defer isRebuilding.Store(false)

	fmt.Println("\nRebuilding...")
	startTime := time.Now()

	m, err := createManager(cfgFile)
	if err != nil {
		fmt.Printf("Rebuild failed: %v\n", err)
		return
	}

	result, err := runBuild(m)
	if err != nil {
		fmt.Printf("Rebuild failed: %v\n", err)
		return
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
		if serveOutputPath != "" && strings.HasPrefix(absPath, serveOutputPath) {
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
