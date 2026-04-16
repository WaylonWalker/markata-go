package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/searchapi"
	"github.com/spf13/cobra"
)

var (
	searchServerPort int
	searchServerHost string
)

var searchServerCmd = &cobra.Command{
	Use:   "search-server",
	Short: "Start a standalone search API server",
	Long: `Start a read-only HTTP search API server powered by bleve full-text search.

The server provides a single GET endpoint that returns JSON search results.
Private post content is never indexed — only metadata (title, description, tags).

Example:
  markata-go search-server
  markata-go search-server --port 8081
  curl "http://localhost:3001/api/search?q=golang&fuzzy=true&limit=10"`,
	RunE: runSearchServer,
}

func init() {
	rootCmd.AddCommand(searchServerCmd)
	searchServerCmd.Flags().IntVar(&searchServerPort, "port", 3001, "Port to listen on")
	searchServerCmd.Flags().StringVar(&searchServerHost, "host", "localhost", "Host to bind to")
}

func runSearchServer(cmd *cobra.Command, _ []string) error {
	app, err := loadListApp(cmd.Context())
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	posts := app.Manager.Posts()

	// Configure search API
	apiCfg := searchapi.DefaultConfig()
	if mc := getModelsConfig(app.Manager); mc != nil {
		apiCfg.DefaultLimit = mc.Search.Bleve.DefaultLimit()
		apiCfg.MaxLimit = mc.Search.Bleve.GetMaxLimit()
		apiCfg.DefaultFuzzy = mc.Search.Bleve.IsFuzzy()
		if len(mc.Search.Bleve.CORSOrigins) > 0 {
			apiCfg.CORSOrigins = mc.Search.Bleve.CORSOrigins
		}
	}

	cacheDir := filepath.Join(".markata", "cache")
	handler := searchapi.NewHandler(posts, cacheDir, apiCfg)
	defer handler.Close()

	endpoint := "/api/search"
	if mc := getModelsConfig(app.Manager); mc != nil {
		endpoint = mc.Search.SearchEndpoint()
	}

	mux := http.NewServeMux()
	mux.Handle(endpoint, handler)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","posts":%d}`, len(posts))
	})

	addr := fmt.Sprintf("%s:%d", searchServerHost, searchServerPort)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Fprintf(os.Stderr, "Search API listening on http://%s%s\n", addr, endpoint)
		fmt.Fprintf(os.Stderr, "Indexed %d posts (%d published)\n", len(posts), countPublished(posts))
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-stop
	fmt.Fprintln(os.Stderr, "\nShutting down...")
	return server.Close()
}

func countPublished(posts []*models.Post) int {
	n := 0
	for _, p := range posts {
		if p.Published && !p.Draft && !p.Skip {
			n++
		}
	}
	return n
}
