package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/builderadmin"
	"github.com/spf13/cobra"
)

var (
	builderAdminHost                 string
	builderAdminPort                 int
	builderAdminSourceDir            string
	builderAdminSiteDir              string
	builderAdminCacheMount           string
	builderAdminHistoryDir           string
	builderAdminWatch                bool
	builderAdminWatchDebounce        time.Duration
	builderAdminFast                 bool
	builderAdminMermaidMode          string
	builderAdminReleasesKeep         int
	builderAdminSuccessfulBuildsKeep int
	builderAdminFailedBuildsKeep     int
	builderAdminRefreshRunsKeep      int
	builderAdminBuildTimeout         time.Duration
	builderAdminRefreshTaskSpecs     []string
)

var builderAdminCmd = &cobra.Command{
	Use:   "builder-admin",
	Short: "Run the long-lived builder admin HTTP service",
	Long: `Run the long-lived builder admin HTTP service.

The service keeps a warm build worker running for hostPath and Kubernetes authoring loops.
It exposes a queue-driven UI/API for builds, logs, releases, rollback, and scheduled refresh tasks.`,
	RunE: runBuilderAdmin,
}

func init() {
	rootCmd.AddCommand(builderAdminCmd)
	builderAdminCmd.Flags().StringVar(&builderAdminHost, "host", "127.0.0.1", "host to bind to")
	builderAdminCmd.Flags().IntVar(&builderAdminPort, "port", 8080, "port to listen on")
	builderAdminCmd.Flags().StringVar(&builderAdminSourceDir, "source-dir", ".", "source directory to watch and build from")
	builderAdminCmd.Flags().StringVar(&builderAdminSiteDir, "site-dir", "public", "site root that contains releases/ and current")
	builderAdminCmd.Flags().StringVar(&builderAdminCacheMount, "cache-mount", "", "optional dedicated cache mount for .markata symlinks")
	builderAdminCmd.Flags().StringVar(&builderAdminHistoryDir, "history-dir", "", "directory for persisted builder-admin state and logs")
	builderAdminCmd.Flags().BoolVar(&builderAdminWatch, "watch", true, "enable recursive file watching")
	builderAdminCmd.Flags().DurationVar(&builderAdminWatchDebounce, "watch-debounce", 2*time.Second, "debounce window for file-watch rebuilds")
	builderAdminCmd.Flags().BoolVar(&builderAdminFast, "fast", false, "run queued builds with --fast")
	builderAdminCmd.Flags().StringVar(&builderAdminMermaidMode, "mermaid-mode", "", "override [markata-go.mermaid].mode for queued builds")
	builderAdminCmd.Flags().IntVar(&builderAdminReleasesKeep, "releases-keep", 10, "number of rendered releases to keep")
	builderAdminCmd.Flags().IntVar(&builderAdminSuccessfulBuildsKeep, "successful-builds-keep", 50, "number of successful build records to keep")
	builderAdminCmd.Flags().IntVar(&builderAdminFailedBuildsKeep, "failed-builds-keep", 100, "number of failed build records to keep")
	builderAdminCmd.Flags().IntVar(&builderAdminRefreshRunsKeep, "refresh-runs-keep", 100, "number of refresh run records to keep")
	builderAdminCmd.Flags().DurationVar(&builderAdminBuildTimeout, "build-timeout", 2*time.Hour, "maximum runtime for a queued build or refresh task")
	builderAdminCmd.Flags().StringArrayVar(&builderAdminRefreshTaskSpecs, "refresh-task", nil, "repeatable task spec: name|every|enqueue|arg1|arg2...")
}

func runBuilderAdmin(_ *cobra.Command, _ []string) error {
	refreshTasks, err := parseRefreshTasks(builderAdminRefreshTaskSpecs)
	if err != nil {
		return err
	}
	svc, err := builderadmin.New(builderadmin.Config{
		Host:                 builderAdminHost,
		Port:                 builderAdminPort,
		SourceDir:            builderAdminSourceDir,
		SiteDir:              builderAdminSiteDir,
		ConfigPath:           cfgFile,
		CacheMount:           builderAdminCacheMount,
		HistoryDir:           builderAdminHistoryDir,
		WatchEnabled:         builderAdminWatch,
		WatchDebounce:        builderAdminWatchDebounce,
		Fast:                 builderAdminFast,
		MermaidMode:          builderAdminMermaidMode,
		ReleasesKeep:         builderAdminReleasesKeep,
		SuccessfulBuildsKeep: builderAdminSuccessfulBuildsKeep,
		FailedBuildsKeep:     builderAdminFailedBuildsKeep,
		RefreshRunsKeep:      builderAdminRefreshRunsKeep,
		RefreshTasks:         refreshTasks,
		BuildTimeout:         builderAdminBuildTimeout,
	})
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return svc.Start(ctx)
}

func parseRefreshTasks(specs []string) ([]builderadmin.RefreshTaskConfig, error) {
	tasks := make([]builderadmin.RefreshTaskConfig, 0, len(specs))
	for _, spec := range specs {
		parts := strings.Split(spec, "|")
		if len(parts) < 4 {
			return nil, fmt.Errorf("invalid --refresh-task %q: expected name|every|enqueue|arg1|arg2...", spec)
		}
		enqueue := strings.EqualFold(parts[2], "true") || parts[2] == "1" || strings.EqualFold(parts[2], "yes")
		tasks = append(tasks, builderadmin.RefreshTaskConfig{
			Name:                  parts[0],
			Every:                 parts[1],
			EnqueueBuildOnSuccess: enqueue,
			Args:                  append([]string(nil), parts[3:]...),
		})
	}
	return tasks, nil
}
