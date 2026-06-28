package builderadmin

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	defaultLogDirName   = "logs"
	defaultStateName    = "state.json"
	defaultOverrideName = "overrides"
	defaultListenHost   = "127.0.0.1"
	defaultListenPort   = 8080
	defaultReleaseKeep  = 10
)

type Config struct {
	Host                 string
	Port                 int
	SourceDir            string
	SiteDir              string
	ConfigPath           string
	CacheMount           string
	HistoryDir           string
	WatchEnabled         bool
	WatchDebounce        time.Duration
	Fast                 bool
	MermaidMode          string
	ReleasesKeep         int
	SuccessfulBuildsKeep int
	FailedBuildsKeep     int
	RefreshRunsKeep      int
	RefreshTasks         []RefreshTaskConfig
	BuildTimeout         time.Duration
}

type RefreshTaskConfig struct {
	Name                  string   `json:"name"`
	Every                 string   `json:"every"`
	EnqueueBuildOnSuccess bool     `json:"enqueue_build_on_success"`
	Args                  []string `json:"args"`

	interval time.Duration
}

type State struct {
	Queue   []QueuedOperation `json:"queue"`
	Running *RunningOperation `json:"running,omitempty"`
	Builds  []BuildRecord     `json:"builds"`
	Refresh []RefreshRecord   `json:"refresh"`
}

type QueuedOperation struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Label       string    `json:"label"`
	TriggerType string    `json:"trigger_type"`
	Detail      string    `json:"detail,omitempty"`
	Changed     []string  `json:"changed,omitempty"`
	EnqueuedAt  time.Time `json:"enqueued_at"`
	ReleaseID   string    `json:"release_id,omitempty"`
	TaskName    string    `json:"task_name,omitempty"`
}

type RunningOperation struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Label       string    `json:"label"`
	TriggerType string    `json:"trigger_type"`
	Detail      string    `json:"detail,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	Phase       string    `json:"phase"`
}

type BuildRecord struct {
	ID              string    `json:"id"`
	Kind            string    `json:"kind"`
	Status          string    `json:"status"`
	TriggerType     string    `json:"trigger_type"`
	TriggerDetail   string    `json:"trigger_detail,omitempty"`
	ChangedPaths    []string  `json:"changed_paths,omitempty"`
	EnqueuedAt      time.Time `json:"enqueued_at"`
	StartedAt       time.Time `json:"started_at"`
	FinishedAt      time.Time `json:"finished_at"`
	QueueWaitMS     int64     `json:"queue_wait_ms"`
	PrepareMS       int64     `json:"prepare_ms"`
	BuildMS         int64     `json:"build_ms"`
	PromoteMS       int64     `json:"promote_ms"`
	PruneMS         int64     `json:"prune_ms"`
	TotalMS         int64     `json:"total_ms"`
	ReleaseID       string    `json:"release_id,omitempty"`
	ReleasePath     string    `json:"release_path,omitempty"`
	BecameLive      bool      `json:"became_live"`
	LogPath         string    `json:"log_path,omitempty"`
	PerfSummary     []string  `json:"perf_summary,omitempty"`
	Error           string    `json:"error,omitempty"`
	RollbackRelease string    `json:"rollback_release,omitempty"`
}

type RefreshRecord struct {
	ID                    string    `json:"id"`
	TaskName              string    `json:"task_name"`
	Status                string    `json:"status"`
	TriggerType           string    `json:"trigger_type"`
	TriggerDetail         string    `json:"trigger_detail,omitempty"`
	EnqueuedAt            time.Time `json:"enqueued_at"`
	StartedAt             time.Time `json:"started_at"`
	FinishedAt            time.Time `json:"finished_at"`
	QueueWaitMS           int64     `json:"queue_wait_ms"`
	RunMS                 int64     `json:"run_ms"`
	TotalMS               int64     `json:"total_ms"`
	LogPath               string    `json:"log_path,omitempty"`
	EnqueuedBuildID       string    `json:"enqueued_build_id,omitempty"`
	EnqueueBuildOnSuccess bool      `json:"enqueue_build_on_success"`
	Command               []string  `json:"command,omitempty"`
	Error                 string    `json:"error,omitempty"`
}

type ReleaseView struct {
	ID           string    `json:"id"`
	Path         string    `json:"path"`
	CreatedAt    time.Time `json:"created_at"`
	Current      bool      `json:"current"`
	BuildID      string    `json:"build_id,omitempty"`
	RollbackOnly bool      `json:"rollback_only"`
}

type Service struct {
	cfg          Config
	executable   string
	statePath    string
	logDir       string
	overrideDir  string
	queueCh      chan queueRequest
	watchMu      sync.Mutex
	watchChanged map[string]struct{}
	watchTimer   *time.Timer
	stateMu      sync.Mutex
	state        State
	server       *http.Server
}

type queueRequest struct {
	QueuedOperation
	commandArgs []string
}

func New(cfg Config) (*Service, error) {
	if cfg.Host == "" {
		cfg.Host = defaultListenHost
	}
	if cfg.Port == 0 {
		cfg.Port = defaultListenPort
	}
	if cfg.SourceDir == "" {
		cfg.SourceDir = "."
	}
	if cfg.SiteDir == "" {
		cfg.SiteDir = "public"
	}
	if cfg.HistoryDir == "" {
		cfg.HistoryDir = filepath.Join(cfg.SiteDir, ".builder-admin")
	}
	if cfg.WatchDebounce <= 0 {
		cfg.WatchDebounce = 2 * time.Second
	}
	if cfg.ReleasesKeep <= 0 {
		cfg.ReleasesKeep = defaultReleaseKeep
	}
	if cfg.SuccessfulBuildsKeep <= 0 {
		cfg.SuccessfulBuildsKeep = 50
	}
	if cfg.FailedBuildsKeep <= 0 {
		cfg.FailedBuildsKeep = 100
	}
	if cfg.RefreshRunsKeep <= 0 {
		cfg.RefreshRunsKeep = 100
	}
	if cfg.BuildTimeout <= 0 {
		cfg.BuildTimeout = 2 * time.Hour
	}
	for i := range cfg.RefreshTasks {
		if cfg.RefreshTasks[i].Name == "" {
			return nil, fmt.Errorf("refresh task name is required")
		}
		if len(cfg.RefreshTasks[i].Args) == 0 {
			return nil, fmt.Errorf("refresh task %q must define args", cfg.RefreshTasks[i].Name)
		}
		d, err := time.ParseDuration(cfg.RefreshTasks[i].Every)
		if err != nil {
			return nil, fmt.Errorf("refresh task %q invalid every duration: %w", cfg.RefreshTasks[i].Name, err)
		}
		cfg.RefreshTasks[i].interval = d
	}
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable: %w", err)
	}
	s := &Service{
		cfg:          cfg,
		executable:   execPath,
		statePath:    filepath.Join(cfg.HistoryDir, defaultStateName),
		logDir:       filepath.Join(cfg.HistoryDir, defaultLogDirName),
		overrideDir:  filepath.Join(cfg.HistoryDir, defaultOverrideName),
		queueCh:      make(chan queueRequest, 128),
		watchChanged: make(map[string]struct{}),
	}
	if err := os.MkdirAll(s.logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	if err := os.MkdirAll(s.overrideDir, 0o755); err != nil {
		return nil, fmt.Errorf("create override dir: %w", err)
	}
	if err := s.loadState(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Service) Start(ctx context.Context) error {
	go s.worker(ctx)
	for i := range s.cfg.RefreshTasks {
		go s.runRefreshScheduler(ctx, s.cfg.RefreshTasks[i])
	}
	if s.cfg.WatchEnabled {
		go s.watchSource(ctx)
	}
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	s.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()
	err := s.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Service) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/builds", s.handleBuilds)
	mux.HandleFunc("/api/refresh/", s.handleRefreshRun)
	mux.HandleFunc("/api/releases/", s.handleReleaseAction)
	mux.HandleFunc("/logs/", s.handleLogs)
}

func (s *Service) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"queue":  len(s.snapshotState().Queue),
	})
}

func (s *Service) handleState(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	state := s.snapshotState()
	_ = json.NewEncoder(w).Encode(struct {
		State        State         `json:"state"`
		Releases     []ReleaseView `json:"releases"`
		CurrentID    string        `json:"current_release_id,omitempty"`
		CurrentPath  string        `json:"current_release_path,omitempty"`
		Config       Config        `json:"config"`
		RefreshTasks []string      `json:"refresh_tasks"`
	}{
		State:        state,
		Releases:     s.discoverReleases(),
		CurrentID:    s.currentReleaseID(),
		CurrentPath:  s.currentReleasePath(),
		Config:       s.cfg,
		RefreshTasks: s.refreshTaskNames(),
	})
}

func (s *Service) handleBuilds(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := s.enqueueBuild("manual-ui", "Manual build from admin UI", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Service) handleRefreshRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/refresh/")
	if name == "" {
		http.Error(w, "missing task name", http.StatusBadRequest)
		return
	}
	if err := s.enqueueRefresh(name, "manual-ui", "Manual refresh from admin UI"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Service) handleReleaseAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/releases/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) != 2 || parts[1] != "rollback" {
		http.Error(w, "unsupported release action", http.StatusBadRequest)
		return
	}
	if err := s.enqueueRollback(parts[0], "manual-ui", "Rollback from admin UI"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Service) handleLogs(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimPrefix(r.URL.Path, "/logs/")
	if rel == "" || strings.Contains(rel, "..") {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(s.logDir, rel)
	data, err := os.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Service) handleIndex(w http.ResponseWriter, _ *http.Request) {
	tmpl := template.Must(template.New("builder-admin").Funcs(template.FuncMap{
		"msToSeconds": func(ms int64) string {
			return fmt.Sprintf("%.2fs", float64(ms)/1000)
		},
		"since": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format(time.RFC3339)
		},
		"summaryPreview": func(lines []string) []string {
			if len(lines) <= 6 {
				return lines
			}
			return lines[len(lines)-6:]
		},
	}).Parse(indexHTML))
	state := s.snapshotState()
	data := struct {
		State        State
		Releases     []ReleaseView
		CurrentID    string
		CurrentPath  string
		RefreshTasks []RefreshTaskConfig
	}{
		State:        state,
		Releases:     s.discoverReleases(),
		CurrentID:    s.currentReleaseID(),
		CurrentPath:  s.currentReleasePath(),
		RefreshTasks: s.cfg.RefreshTasks,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

func (s *Service) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-s.queueCh:
			s.process(ctx, req)
		}
	}
}

func (s *Service) process(ctx context.Context, req queueRequest) {
	s.setRunning(req)
	defer s.clearRunning()
	s.removeQueued(req.ID)
	switch req.Kind {
	case "build":
		s.runBuild(ctx, req)
	case "refresh":
		s.runRefresh(ctx, req)
	case "rollback":
		s.runRollback(req)
	}
}

func (s *Service) enqueueBuild(triggerType, detail string, changed []string) error {
	queued := QueuedOperation{
		ID:          nextID("build"),
		Kind:        "build",
		Label:       "Build",
		TriggerType: triggerType,
		Detail:      detail,
		Changed:     append([]string(nil), changed...),
		EnqueuedAt:  time.Now().UTC(),
	}
	s.pushQueued(queued)
	s.queueCh <- queueRequest{QueuedOperation: queued}
	return nil
}

func (s *Service) enqueueRefresh(name, triggerType, detail string) error {
	task, ok := s.findRefreshTask(name)
	if !ok {
		return fmt.Errorf("unknown refresh task %q", name)
	}
	queued := QueuedOperation{
		ID:          nextID("refresh"),
		Kind:        "refresh",
		Label:       "Refresh " + task.Name,
		TriggerType: triggerType,
		Detail:      detail,
		EnqueuedAt:  time.Now().UTC(),
		TaskName:    task.Name,
	}
	s.pushQueued(queued)
	s.queueCh <- queueRequest{QueuedOperation: queued, commandArgs: append([]string(nil), task.Args...)}
	return nil
}

func (s *Service) enqueueRollback(releaseID, triggerType, detail string) error {
	if releaseID == "" {
		return fmt.Errorf("release id is required")
	}
	queued := QueuedOperation{
		ID:          nextID("rollback"),
		Kind:        "rollback",
		Label:       "Rollback " + releaseID,
		TriggerType: triggerType,
		Detail:      detail,
		EnqueuedAt:  time.Now().UTC(),
		ReleaseID:   releaseID,
	}
	s.pushQueued(queued)
	s.queueCh <- queueRequest{QueuedOperation: queued}
	return nil
}

func (s *Service) runBuild(ctx context.Context, req queueRequest) {
	started := time.Now().UTC()
	record := BuildRecord{
		ID:            req.ID,
		Kind:          req.Kind,
		Status:        "running",
		TriggerType:   req.TriggerType,
		TriggerDetail: req.Detail,
		ChangedPaths:  append([]string(nil), req.Changed...),
		EnqueuedAt:    req.EnqueuedAt,
		StartedAt:     started,
		QueueWaitMS:   started.Sub(req.EnqueuedAt).Milliseconds(),
	}
	logPath, logFile, err := s.createLogFile(req.ID)
	if err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		record.FinishedAt = time.Now().UTC()
		record.TotalMS = record.FinishedAt.Sub(started).Milliseconds()
		s.finishBuild(record)
		return
	}
	defer logFile.Close()
	record.LogPath = logPath
	ctx, cancel := context.WithTimeout(ctx, s.cfg.BuildTimeout)
	defer cancel()

	phaseStart := time.Now()
	s.updateRunningPhase("prepare")
	if err := s.prepareBuild(logFile); err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		record.PrepareMS = time.Since(phaseStart).Milliseconds()
		record.FinishedAt = time.Now().UTC()
		record.TotalMS = record.FinishedAt.Sub(started).Milliseconds()
		record.PerfSummary = extractPerfSummaryFromFile(filepath.Join(s.logDir, logPath))
		s.finishBuild(record)
		return
	}
	record.PrepareMS = time.Since(phaseStart).Milliseconds()

	buildWork := filepath.Join(s.cfg.SiteDir, ".build-work")
	phaseStart = time.Now()
	s.updateRunningPhase("build")
	cmdArgs, cleanup, err := s.buildCommandArgs(req.ID, buildWork)
	if err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		record.BuildMS = time.Since(phaseStart).Milliseconds()
		record.FinishedAt = time.Now().UTC()
		record.TotalMS = record.FinishedAt.Sub(started).Milliseconds()
		record.PerfSummary = extractPerfSummaryFromFile(filepath.Join(s.logDir, logPath))
		s.finishBuild(record)
		return
	}
	defer cleanup()
	if err := s.runLoggedCommand(ctx, logFile, s.cfg.SourceDir, nil, cmdArgs...); err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		record.BuildMS = time.Since(phaseStart).Milliseconds()
		record.FinishedAt = time.Now().UTC()
		record.TotalMS = record.FinishedAt.Sub(started).Milliseconds()
		record.PerfSummary = extractPerfSummaryFromFile(filepath.Join(s.logDir, logPath))
		s.finishBuild(record)
		return
	}
	record.BuildMS = time.Since(phaseStart).Milliseconds()

	phaseStart = time.Now()
	s.updateRunningPhase("promote")
	releaseID, releasePath, err := s.promoteBuild(buildWork)
	if err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		record.PromoteMS = time.Since(phaseStart).Milliseconds()
		record.FinishedAt = time.Now().UTC()
		record.TotalMS = record.FinishedAt.Sub(started).Milliseconds()
		record.PerfSummary = extractPerfSummaryFromFile(filepath.Join(s.logDir, logPath))
		s.finishBuild(record)
		return
	}
	record.PromoteMS = time.Since(phaseStart).Milliseconds()
	record.ReleaseID = releaseID
	record.ReleasePath = releasePath
	record.BecameLive = true

	phaseStart = time.Now()
	s.updateRunningPhase("prune")
	_ = s.pruneReleases()
	record.PruneMS = time.Since(phaseStart).Milliseconds()

	record.Status = "success"
	record.FinishedAt = time.Now().UTC()
	record.TotalMS = record.FinishedAt.Sub(started).Milliseconds()
	record.PerfSummary = extractPerfSummaryFromFile(filepath.Join(s.logDir, logPath))
	s.finishBuild(record)
}

func (s *Service) runRefresh(ctx context.Context, req queueRequest) {
	started := time.Now().UTC()
	record := RefreshRecord{
		ID:                    req.ID,
		TaskName:              req.TaskName,
		Status:                "running",
		TriggerType:           req.TriggerType,
		TriggerDetail:         req.Detail,
		EnqueuedAt:            req.EnqueuedAt,
		StartedAt:             started,
		QueueWaitMS:           started.Sub(req.EnqueuedAt).Milliseconds(),
		EnqueueBuildOnSuccess: false,
		Command:               append([]string(nil), req.commandArgs...),
	}
	task, ok := s.findRefreshTask(req.TaskName)
	if ok {
		record.EnqueueBuildOnSuccess = task.EnqueueBuildOnSuccess
	}
	logPath, logFile, err := s.createLogFile(req.ID)
	if err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		record.FinishedAt = time.Now().UTC()
		record.TotalMS = record.FinishedAt.Sub(started).Milliseconds()
		s.finishRefresh(record)
		return
	}
	defer logFile.Close()
	record.LogPath = logPath
	ctx, cancel := context.WithTimeout(ctx, s.cfg.BuildTimeout)
	defer cancel()
	s.updateRunningPhase("refresh")
	runStart := time.Now()
	if err := s.runLoggedCommand(ctx, logFile, s.cfg.SourceDir, nil, req.commandArgs...); err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		record.RunMS = time.Since(runStart).Milliseconds()
		record.FinishedAt = time.Now().UTC()
		record.TotalMS = record.FinishedAt.Sub(started).Milliseconds()
		s.finishRefresh(record)
		return
	}
	record.RunMS = time.Since(runStart).Milliseconds()
	record.Status = "success"
	if task.EnqueueBuildOnSuccess {
		buildID := nextID("build")
		queued := QueuedOperation{
			ID:          buildID,
			Kind:        "build",
			Label:       "Build",
			TriggerType: "scheduled-refresh",
			Detail:      "Build enqueued by refresh task " + task.Name,
			EnqueuedAt:  time.Now().UTC(),
		}
		record.EnqueuedBuildID = buildID
		s.pushQueued(queued)
		s.queueCh <- queueRequest{QueuedOperation: queued}
	}
	record.FinishedAt = time.Now().UTC()
	record.TotalMS = record.FinishedAt.Sub(started).Milliseconds()
	s.finishRefresh(record)
}

func (s *Service) runRollback(req queueRequest) {
	started := time.Now().UTC()
	record := BuildRecord{
		ID:              req.ID,
		Kind:            req.Kind,
		Status:          "running",
		TriggerType:     req.TriggerType,
		TriggerDetail:   req.Detail,
		EnqueuedAt:      req.EnqueuedAt,
		StartedAt:       started,
		QueueWaitMS:     started.Sub(req.EnqueuedAt).Milliseconds(),
		RollbackRelease: req.ReleaseID,
	}
	logPath, logFile, err := s.createLogFile(req.ID)
	if err != nil {
		record.Status = "failed"
		record.Error = err.Error()
		record.FinishedAt = time.Now().UTC()
		record.TotalMS = record.FinishedAt.Sub(started).Milliseconds()
		s.finishBuild(record)
		return
	}
	defer logFile.Close()
	record.LogPath = logPath
	s.updateRunningPhase("promote")
	releasePath := filepath.Join(s.cfg.SiteDir, "releases", req.ReleaseID)
	if _, err := os.Stat(releasePath); err != nil {
		record.Status = "failed"
		record.Error = fmt.Sprintf("release %q not found: %v", req.ReleaseID, err)
	} else if err := s.switchCurrentRelease(req.ReleaseID); err != nil {
		record.Status = "failed"
		record.Error = err.Error()
	} else {
		_, _ = fmt.Fprintf(logFile, "promoted release %s\n", req.ReleaseID)
		record.Status = "success"
		record.ReleaseID = req.ReleaseID
		record.ReleasePath = releasePath
		record.BecameLive = true
	}
	record.FinishedAt = time.Now().UTC()
	record.PromoteMS = record.FinishedAt.Sub(started).Milliseconds()
	record.TotalMS = record.PromoteMS
	record.PerfSummary = extractPerfSummaryFromFile(filepath.Join(s.logDir, logPath))
	s.finishBuild(record)
}

func (s *Service) prepareBuild(log io.Writer) error {
	if err := os.MkdirAll(filepath.Join(s.cfg.SiteDir, "releases"), 0o755); err != nil {
		return err
	}
	if s.cfg.CacheMount != "" {
		for _, part := range []string{"build", "plugin", "xdg"} {
			if err := os.MkdirAll(filepath.Join(s.cfg.CacheMount, part), 0o755); err != nil {
				return err
			}
		}
		for _, linkName := range []string{".markata", ".markata-cache"} {
			_ = os.RemoveAll(filepath.Join(s.cfg.SourceDir, linkName))
		}
		if err := os.Symlink(filepath.Join(s.cfg.CacheMount, "build"), filepath.Join(s.cfg.SourceDir, ".markata")); err != nil {
			return err
		}
		if err := os.Symlink(filepath.Join(s.cfg.CacheMount, "plugin"), filepath.Join(s.cfg.SourceDir, ".markata-cache")); err != nil {
			return err
		}
	}
	buildWork := filepath.Join(s.cfg.SiteDir, ".build-work")
	if err := os.RemoveAll(buildWork); err != nil {
		return err
	}
	if err := os.MkdirAll(buildWork, 0o755); err != nil {
		return err
	}
	current := filepath.Join(s.cfg.SiteDir, "current")
	if _, err := os.Stat(current); err == nil {
		_, _ = fmt.Fprintln(log, "seeding build work from current release")
		return s.runLoggedCommand(context.Background(), log, "", nil, "cp", "-al", current+"/.", buildWork+string(os.PathSeparator))
	}
	return nil
}

func (s *Service) buildCommandArgs(id, buildWork string) ([]string, func(), error) {
	args := make([]string, 0, 10)
	if s.cfg.ConfigPath != "" {
		args = append(args, "--config", s.cfg.ConfigPath)
	}
	cleanup := func() {}
	if s.cfg.MermaidMode != "" {
		overridePath := filepath.Join(s.overrideDir, "builder-admin.toml")
		contents := fmt.Sprintf("[markata-go.mermaid]\nmode = %q\n", s.cfg.MermaidMode)
		if err := os.WriteFile(overridePath, []byte(contents), 0o644); err != nil {
			return nil, cleanup, err
		}
		args = append(args, "-m", overridePath)
	}
	args = append(args, "build")
	if s.cfg.Fast {
		args = append(args, "--fast")
	}
	args = append(args, "--output", buildWork)
	return args, cleanup, nil
}

func (s *Service) promoteBuild(buildWork string) (string, string, error) {
	releaseID := time.Now().UTC().Format("20060102T150405Z") + "-" + hostSuffix()
	releasePath := filepath.Join(s.cfg.SiteDir, "releases", releaseID)
	if err := os.RemoveAll(releasePath); err != nil {
		return "", "", err
	}
	if err := os.Rename(buildWork, releasePath); err != nil {
		return "", "", err
	}
	if err := s.switchCurrentRelease(releaseID); err != nil {
		return "", "", err
	}
	return releaseID, releasePath, nil
}

func (s *Service) switchCurrentRelease(releaseID string) error {
	currentNext := filepath.Join(s.cfg.SiteDir, "current.next")
	_ = os.Remove(currentNext)
	if err := os.Symlink(filepath.Join("releases", releaseID), currentNext); err != nil {
		return err
	}
	return os.Rename(currentNext, filepath.Join(s.cfg.SiteDir, "current"))
}

func (s *Service) pruneReleases() error {
	releases := s.discoverReleases()
	if len(releases) <= s.cfg.ReleasesKeep {
		return nil
	}
	for _, release := range releases[s.cfg.ReleasesKeep:] {
		if release.Current {
			continue
		}
		_ = os.RemoveAll(release.Path)
	}
	return nil
}

func (s *Service) runLoggedCommand(ctx context.Context, log io.Writer, cwd string, env []string, args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("command is required")
	}
	cmdName := args[0]
	cmdArgs := args[1:]
	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)
	if strings.HasPrefix(cmdName, "-") || cmdName == "build" || strings.HasSuffix(cmdName, "markata-go") || filepath.Base(cmdName) == filepath.Base(s.executable) {
		if strings.HasSuffix(cmdName, "markata-go") || filepath.Base(cmdName) == filepath.Base(s.executable) {
			cmd = exec.CommandContext(ctx, s.executable, cmdArgs...)
		} else {
			cmd = exec.CommandContext(ctx, s.executable, args...)
		}
	}
	cmd.Stdout = log
	cmd.Stderr = log
	if cwd != "" {
		cmd.Dir = cwd
	}
	cmd.Env = os.Environ()
	if len(env) > 0 {
		cmd.Env = append(cmd.Env, env...)
	}
	_, _ = fmt.Fprintf(log, "$ %s\n", strings.Join(cmd.Args, " "))
	err := cmd.Run()
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return fmt.Errorf("command failed with exit code %d", ws.ExitStatus())
		}
	}
	return err
}

func (s *Service) runRefreshScheduler(ctx context.Context, task RefreshTaskConfig) {
	ticker := time.NewTicker(task.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.enqueueRefresh(task.Name, "schedule", "Scheduled refresh")
		}
	}
}

func (s *Service) watchSource(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()
	_ = addDirRecursive(watcher, s.cfg.SourceDir)
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if ignoreWatchPath(s.cfg.SourceDir, event.Name) {
				continue
			}
			if event.Op&fsnotify.Create != 0 {
				_ = addDirRecursiveIfDir(watcher, event.Name)
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			s.recordWatchPath(event.Name)
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func (s *Service) recordWatchPath(path string) {
	rel, err := filepath.Rel(s.cfg.SourceDir, path)
	if err != nil {
		rel = path
	}
	s.watchMu.Lock()
	defer s.watchMu.Unlock()
	s.watchChanged[filepath.ToSlash(rel)] = struct{}{}
	if s.watchTimer != nil {
		if !s.watchTimer.Stop() {
			select {
			case <-s.watchTimer.C:
			default:
			}
		}
	}
	s.watchTimer = time.AfterFunc(s.cfg.WatchDebounce, func() {
		s.flushWatchBuild()
	})
}

func (s *Service) flushWatchBuild() {
	s.watchMu.Lock()
	changed := make([]string, 0, len(s.watchChanged))
	for path := range s.watchChanged {
		changed = append(changed, path)
	}
	clear(s.watchChanged)
	s.watchMu.Unlock()
	if len(changed) == 0 {
		return
	}
	sort.Strings(changed)
	_ = s.enqueueBuild("file-watch", fmt.Sprintf("Debounced file-watch build (%d paths)", len(changed)), changed)
}

func (s *Service) findRefreshTask(name string) (RefreshTaskConfig, bool) {
	for _, task := range s.cfg.RefreshTasks {
		if task.Name == name {
			return task, true
		}
	}
	return RefreshTaskConfig{}, false
}

func (s *Service) refreshTaskNames() []string {
	names := make([]string, 0, len(s.cfg.RefreshTasks))
	for _, task := range s.cfg.RefreshTasks {
		names = append(names, task.Name)
	}
	return names
}

func (s *Service) createLogFile(id string) (string, *os.File, error) {
	rel := id + ".log"
	path := filepath.Join(s.logDir, rel)
	f, err := os.Create(path)
	if err != nil {
		return "", nil, err
	}
	return rel, f, nil
}

func (s *Service) snapshotState() State {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	clone := s.state
	clone.Queue = append([]QueuedOperation(nil), s.state.Queue...)
	clone.Builds = append([]BuildRecord(nil), s.state.Builds...)
	clone.Refresh = append([]RefreshRecord(nil), s.state.Refresh...)
	return clone
}

func (s *Service) pushQueued(queued QueuedOperation) {
	s.stateMu.Lock()
	s.state.Queue = append(s.state.Queue, queued)
	s.saveStateLocked()
	s.stateMu.Unlock()
}

func (s *Service) removeQueued(id string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	filtered := s.state.Queue[:0]
	for _, queued := range s.state.Queue {
		if queued.ID != id {
			filtered = append(filtered, queued)
		}
	}
	s.state.Queue = append([]QueuedOperation(nil), filtered...)
	s.saveStateLocked()
}

func (s *Service) setRunning(req queueRequest) {
	s.stateMu.Lock()
	s.state.Running = &RunningOperation{
		ID:          req.ID,
		Kind:        req.Kind,
		Label:       req.Label,
		TriggerType: req.TriggerType,
		Detail:      req.Detail,
		StartedAt:   time.Now().UTC(),
		Phase:       "starting",
	}
	s.saveStateLocked()
	s.stateMu.Unlock()
}

func (s *Service) updateRunningPhase(phase string) {
	s.stateMu.Lock()
	if s.state.Running != nil {
		s.state.Running.Phase = phase
		s.saveStateLocked()
	}
	s.stateMu.Unlock()
}

func (s *Service) clearRunning() {
	s.stateMu.Lock()
	s.state.Running = nil
	s.saveStateLocked()
	s.stateMu.Unlock()
}

func (s *Service) finishBuild(record BuildRecord) {
	s.stateMu.Lock()
	s.state.Builds = append([]BuildRecord{record}, s.state.Builds...)
	s.pruneBuildHistoryLocked()
	s.saveStateLocked()
	s.stateMu.Unlock()
}

func (s *Service) finishRefresh(record RefreshRecord) {
	s.stateMu.Lock()
	s.state.Refresh = append([]RefreshRecord{record}, s.state.Refresh...)
	s.pruneRefreshHistoryLocked()
	s.saveStateLocked()
	s.stateMu.Unlock()
}

func (s *Service) pruneBuildHistoryLocked() {
	kept := make([]BuildRecord, 0, len(s.state.Builds))
	successCount := 0
	failureCount := 0
	for _, record := range s.state.Builds {
		keep := false
		switch record.Status {
		case "success":
			if successCount < s.cfg.SuccessfulBuildsKeep {
				keep = true
				successCount++
			}
		default:
			if failureCount < s.cfg.FailedBuildsKeep {
				keep = true
				failureCount++
			}
		}
		if keep {
			kept = append(kept, record)
			continue
		}
		if record.LogPath != "" {
			_ = os.Remove(filepath.Join(s.logDir, record.LogPath))
		}
	}
	s.state.Builds = kept
}

func (s *Service) pruneRefreshHistoryLocked() {
	if len(s.state.Refresh) <= s.cfg.RefreshRunsKeep {
		return
	}
	for _, record := range s.state.Refresh[s.cfg.RefreshRunsKeep:] {
		if record.LogPath != "" {
			_ = os.Remove(filepath.Join(s.logDir, record.LogPath))
		}
	}
	s.state.Refresh = append([]RefreshRecord(nil), s.state.Refresh[:s.cfg.RefreshRunsKeep]...)
}

func (s *Service) loadState() error {
	data, err := os.ReadFile(s.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			s.state = State{}
			return nil
		}
		return fmt.Errorf("read state: %w", err)
	}
	if err := json.Unmarshal(data, &s.state); err != nil {
		return fmt.Errorf("decode state: %w", err)
	}
	// Queue and running state are in-memory worker concerns. If the pod restarted,
	// the old worker is gone, so clear transient state instead of showing ghosts.
	s.state.Queue = nil
	s.state.Running = nil
	s.stateMu.Lock()
	s.saveStateLocked()
	s.stateMu.Unlock()
	return nil
}

func (s *Service) saveStateLocked() {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return
	}
	tmp := s.statePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, s.statePath)
}

func (s *Service) discoverReleases() []ReleaseView {
	releasesDir := filepath.Join(s.cfg.SiteDir, "releases")
	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		return nil
	}
	current := s.currentReleaseID()
	buildByRelease := make(map[string]string)
	for _, record := range s.snapshotState().Builds {
		if record.ReleaseID != "" && buildByRelease[record.ReleaseID] == "" {
			buildByRelease[record.ReleaseID] = record.ID
		}
	}
	views := make([]ReleaseView, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		views = append(views, ReleaseView{
			ID:           entry.Name(),
			Path:         filepath.Join(releasesDir, entry.Name()),
			CreatedAt:    info.ModTime().UTC(),
			Current:      entry.Name() == current,
			BuildID:      buildByRelease[entry.Name()],
			RollbackOnly: true,
		})
	}
	sort.Slice(views, func(i, j int) bool {
		return views[i].CreatedAt.After(views[j].CreatedAt)
	})
	return views
}

func (s *Service) currentReleasePath() string {
	target, err := os.Readlink(filepath.Join(s.cfg.SiteDir, "current"))
	if err != nil {
		return ""
	}
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Join(s.cfg.SiteDir, target)
}

func (s *Service) currentReleaseID() string {
	path := s.currentReleasePath()
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

func addDirRecursive(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if ignoreWatchPath(root, path) && path != root {
			return filepath.SkipDir
		}
		return w.Add(path)
	})
}

func addDirRecursiveIfDir(w *fsnotify.Watcher, path string) error {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return nil
	}
	return addDirRecursive(w, path)
}

func ignoreWatchPath(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return false
	}
	parts := strings.Split(rel, "/")
	for _, part := range parts {
		switch part {
		case ".git", ".markata", ".markata-cache", ".builder-admin":
			return true
		}
	}
	return false
}

func extractPerfSummaryFromFile(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	summary := make([]string, 0, 24)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "] took ") || strings.Contains(line, "Hotspots:") || strings.Contains(line, "Duration:") {
			summary = append(summary, line)
		}
	}
	if len(summary) > 24 {
		return summary[len(summary)-24:]
	}
	return summary
}

func nextID(prefix string) string {
	return prefix + "-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
}

func hostSuffix() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "host"
	}
	return host
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Builder Admin</title>
  <meta http-equiv="refresh" content="5">
  <style>
    :root {
      color-scheme: dark;
      --bg: #09090b;
      --panel: rgba(24, 24, 27, 0.9);
      --panel-strong: rgba(9, 9, 11, 0.95);
      --line: rgba(82, 82, 91, 0.75);
      --line-soft: rgba(63, 63, 70, 0.55);
      --text: #f4f4f5;
      --muted: #a1a1aa;
      --accent: #fafafa;
      --shadow: 0 24px 60px rgba(0, 0, 0, 0.35);
    }
    * { box-sizing: border-box; }
    html { background: var(--bg); }
    body {
      margin: 0;
      color: var(--text);
      font-family: Inter, ui-sans-serif, system-ui, sans-serif;
      background:
        radial-gradient(circle at top left, rgba(255,255,255,0.04), transparent 30%),
        radial-gradient(circle at bottom right, rgba(255,255,255,0.03), transparent 35%),
        linear-gradient(rgba(255,255,255,0.018) 1px, transparent 1px),
        linear-gradient(90deg, rgba(255,255,255,0.018) 1px, transparent 1px),
        var(--bg);
      background-size: auto, auto, 11px 11px, 11px 11px, auto;
    }
    a { color: var(--accent); text-decoration: none; }
    a:hover { text-decoration: underline; }
    main {
      width: 100%;
      max-width: none;
      padding: 20px 24px 48px;
    }
    h1, h2, h3, p { margin: 0; }
    .topbar {
      display: flex;
      justify-content: space-between;
      gap: 20px;
      align-items: flex-start;
      margin-bottom: 20px;
    }
    .titleblock h1 {
      font-size: clamp(2rem, 4vw, 3.6rem);
      line-height: 0.95;
      letter-spacing: -0.06em;
      text-transform: uppercase;
    }
    .titleblock p {
      margin-top: 10px;
      color: var(--muted);
      max-width: 72ch;
    }
    .hero {
      display: grid;
      grid-template-columns: 1.5fr 1fr;
      gap: 18px;
      margin-bottom: 20px;
    }
    .grid {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 18px;
      margin-bottom: 20px;
    }
    .section-grid {
      display: grid;
      grid-template-columns: 1.3fr 1fr;
      gap: 18px;
      margin-bottom: 20px;
    }
    .card {
      background: var(--panel);
      border: 1px solid var(--line-soft);
      border-radius: 28px;
      padding: 18px 20px;
      box-shadow: var(--shadow);
      backdrop-filter: blur(10px);
    }
    .card strong, .muted-label {
      display: block;
      font-size: 0.72rem;
      letter-spacing: 0.14em;
      text-transform: uppercase;
      color: var(--muted);
      margin-bottom: 6px;
    }
    .value {
      font-size: 1.15rem;
      line-height: 1.25;
      word-break: break-word;
    }
    .actions {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      align-content: start;
    }
    .actions form { margin: 0; }
    button {
      background: var(--panel-strong);
      color: var(--text);
      border: 1px solid var(--line);
      border-radius: 999px;
      padding: 10px 16px;
      cursor: pointer;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      font-size: 0.75rem;
    }
    button.secondary { background: transparent; }
    button:hover { background: #18181b; }
    .stack { display: flex; flex-direction: column; gap: 10px; }
    .panel-head {
      display: flex;
      justify-content: space-between;
      align-items: baseline;
      gap: 12px;
      margin-bottom: 12px;
    }
    .panel-head h2 { font-size: 1rem; text-transform: uppercase; letter-spacing: 0.08em; }
    .panel-head span { color: var(--muted); font-size: 0.8rem; }
    table { width: 100%; border-collapse: collapse; table-layout: fixed; }
    th, td {
      text-align: left;
      padding: 10px 8px;
      border-top: 1px solid var(--line-soft);
      vertical-align: top;
      font-size: 0.9rem;
    }
    th {
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.08em;
      font-size: 0.72rem;
    }
    code {
      display: inline-block;
      background: rgba(255,255,255,0.04);
      border: 1px solid rgba(255,255,255,0.05);
      border-radius: 999px;
      padding: 3px 8px;
      white-space: nowrap;
      max-width: 100%;
      overflow: hidden;
      text-overflow: ellipsis;
      color: #fafafa;
    }
    pre {
      margin: 0;
      padding: 10px 12px;
      overflow: auto;
      white-space: pre-wrap;
      background: rgba(0,0,0,0.35);
      border: 1px solid var(--line-soft);
      border-radius: 18px;
      max-height: 11rem;
      line-height: 1.45;
      color: #e4e4e7;
      font-size: 0.82rem;
    }
    .pill {
      display: inline-block;
      padding: 4px 10px;
      border-radius: 999px;
      border: 1px solid var(--line);
      background: rgba(255,255,255,0.04);
      color: var(--text);
      text-transform: uppercase;
      letter-spacing: 0.08em;
      font-size: 0.7rem;
    }
    .summary-cell { min-width: 0; }
    .summary-meta { color: var(--muted); font-size: 0.76rem; margin-bottom: 6px; }
    .summary-list { display: grid; gap: 6px; }
    .summary-list div { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: #e4e4e7; }
    .mono { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }
    .wide { overflow-x: auto; }
    .muted { color: var(--muted); }
    .tabs {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin-bottom: 14px;
    }
    .tabs a {
      display: inline-flex;
      align-items: center;
      border: 1px solid var(--line-soft);
      border-radius: 999px;
      padding: 9px 14px;
      text-transform: uppercase;
      letter-spacing: 0.08em;
      font-size: 0.72rem;
      color: var(--muted);
      background: rgba(255,255,255,0.03);
    }
    .tabs a.active {
      color: var(--text);
      background: var(--panel-strong);
      border-color: var(--line);
    }
    .tab-panel { display: none; }
    .tab-panel:target { display: block; }
    .tab-panel.default-panel { display: block; }
    .tab-shell .tab-panel:target ~ .default-panel { display: none; }
    @media (max-width: 1200px) {
      .hero, .section-grid, .grid { grid-template-columns: 1fr 1fr; }
    }
    @media (max-width: 800px) {
      main { padding: 14px; }
      .topbar, .hero, .section-grid, .grid { grid-template-columns: 1fr; display: grid; }
      .topbar { gap: 14px; }
      table { min-width: 860px; }
    }
  </style>
</head>
<body>
<main>
  <div class="topbar">
    <div class="titleblock">
      <h1>Builder Admin</h1>
      <p>Warm queue-driven builds, release promotion, raw logs, and refresh scheduling for the live go.waylonwalker.com authoring loop.</p>
    </div>
  </div>

  <div class="hero">
    <section class="card">
      <div class="panel-head"><h2>Live State</h2><span>desktop-first control surface</span></div>
      <div class="grid" style="grid-template-columns: repeat(3, minmax(0, 1fr)); margin-bottom: 0;">
        <div>
          <strong>Current release</strong>
          <div class="value mono">{{ .CurrentID }}</div>
        </div>
        <div>
          <strong>Current path</strong>
          <div class="value mono">{{ .CurrentPath }}</div>
        </div>
        <div>
          <strong>Active work</strong>
          <div class="value">{{ if .State.Running }}{{ .State.Running.Kind }} <span class="pill">{{ .State.Running.Phase }}</span>{{ else }}idle{{ end }}</div>
        </div>
      </div>
    </section>
    <section class="card actions">
      <div class="panel-head"><h2>Actions</h2><span>manual triggers</span></div>
      <form method="post" action="/api/builds"><button type="submit">Enqueue Build</button></form>
      {{ range .RefreshTasks }}
      <form method="post" action="/api/refresh/{{ .Name }}"><button class="secondary" type="submit">Run {{ .Name }}</button></form>
      {{ end }}
    </section>
  </div>

  <div class="grid">
    <section class="card">
      <strong>Queue</strong>
      <div class="value">{{ len .State.Queue }}</div>
    </section>
    <section class="card">
      <strong>Build history</strong>
      <div class="value">{{ len .State.Builds }}</div>
    </section>
    <section class="card">
      <strong>Refresh history</strong>
      <div class="value">{{ len .State.Refresh }}</div>
    </section>
    <section class="card">
      <strong>Releases</strong>
      <div class="value">{{ len .Releases }}</div>
    </section>
  </div>

  <div class="section-grid">
  <section class="card wide">
    <div class="panel-head"><h2>Queue</h2><span>debounced watch + manual triggers</span></div>
    <table>
      <thead><tr><th>ID</th><th>Kind</th><th>Trigger</th><th>Detail</th><th>Changed</th><th>Queued</th></tr></thead>
      <tbody>
      {{ range .State.Queue }}
      <tr>
        <td><code>{{ .ID }}</code></td>
        <td>{{ .Kind }}</td>
        <td>{{ .TriggerType }}</td>
        <td>{{ .Detail }}</td>
        <td>{{ range .Changed }}<div><code>{{ . }}</code></div>{{ end }}</td>
        <td>{{ since .EnqueuedAt }}</td>
      </tr>
      {{ else }}
      <tr><td colspan="6">Queue is empty.</td></tr>
      {{ end }}
      </tbody>
    </table>
  </section>

  <section class="card">
    <div class="panel-head"><h2>Running</h2><span>live worker</span></div>
    <div class="stack">
      {{ if .State.Running }}
      <div><strong>ID</strong><div class="value mono">{{ .State.Running.ID }}</div></div>
      <div><strong>Kind</strong><div class="value">{{ .State.Running.Kind }}</div></div>
      <div><strong>Trigger</strong><div class="value">{{ .State.Running.TriggerType }}</div></div>
      <div><strong>Detail</strong><div class="value">{{ .State.Running.Detail }}</div></div>
      <div><strong>Started</strong><div class="value mono">{{ since .State.Running.StartedAt }}</div></div>
      <div><strong>Phase</strong><div class="value"><span class="pill">{{ .State.Running.Phase }}</span></div></div>
      {{ else }}
      <div class="muted">No build or refresh is running right now.</div>
      {{ end }}
    </div>
  </section>
  </div>

  <section class="card wide tab-shell">
    <div class="panel-head"><h2>Workspace</h2><span>switch between builds, refreshes, and releases</span></div>
    <nav class="tabs">
      <a href="#builds" class="active">Builds</a>
      <a href="#refresh-runs">Refresh Runs</a>
      <a href="#releases">Releases</a>
    </nav>

    <section id="builds" class="tab-panel default-panel">
      <table>
        <thead><tr><th>ID</th><th>Status</th><th>Trigger</th><th>Total</th><th>Build</th><th>Release</th><th>Logs</th><th>Summary</th></tr></thead>
        <tbody>
        {{ range .State.Builds }}
        <tr>
          <td><code>{{ .ID }}</code></td>
          <td>{{ .Status }}</td>
          <td>{{ .TriggerType }}</td>
          <td>{{ msToSeconds .TotalMS }}</td>
          <td>{{ msToSeconds .BuildMS }}</td>
          <td>{{ if .ReleaseID }}<code>{{ .ReleaseID }}</code>{{ end }}</td>
          <td>{{ if .LogPath }}<a href="/logs/{{ .LogPath }}">log</a>{{ end }}</td>
          <td class="summary-cell">{{ if .PerfSummary }}<div class="summary-meta">{{ len .PerfSummary }} perf lines</div><div class="summary-list mono">{{ range summaryPreview .PerfSummary }}<div>{{ . }}</div>{{ end }}</div>{{ end }}</td>
        </tr>
        {{ else }}
        <tr><td colspan="8">No builds yet.</td></tr>
        {{ end }}
        </tbody>
      </table>
    </section>

    <section id="refresh-runs" class="tab-panel">
      <table>
        <thead><tr><th>ID</th><th>Task</th><th>Status</th><th>Total</th><th>Logs</th><th>Build</th><th>Command</th></tr></thead>
        <tbody>
        {{ range .State.Refresh }}
        <tr>
          <td><code>{{ .ID }}</code></td>
          <td>{{ .TaskName }}</td>
          <td>{{ .Status }}</td>
          <td>{{ msToSeconds .TotalMS }}</td>
          <td>{{ if .LogPath }}<a href="/logs/{{ .LogPath }}">log</a>{{ end }}</td>
          <td>{{ if .EnqueuedBuildID }}<code>{{ .EnqueuedBuildID }}</code>{{ end }}</td>
          <td class="mono muted">{{ if .Command }}{{ index .Command 0 }} {{ end }}</td>
        </tr>
        {{ else }}
        <tr><td colspan="7">No refresh runs yet.</td></tr>
        {{ end }}
        </tbody>
      </table>
    </section>

    <section id="releases" class="tab-panel">
      <table>
        <thead><tr><th>ID</th><th>Current</th><th>Created</th><th>Build</th><th>Action</th></tr></thead>
        <tbody>
        {{ range .Releases }}
        <tr>
          <td><code>{{ .ID }}</code></td>
          <td>{{ if .Current }}live{{ end }}</td>
          <td>{{ since .CreatedAt }}</td>
          <td>{{ if .BuildID }}<code>{{ .BuildID }}</code>{{ end }}</td>
          <td>{{ if not .Current }}<form method="post" action="/api/releases/{{ .ID }}/rollback"><button class="secondary" type="submit">Promote</button></form>{{ end }}</td>
        </tr>
        {{ else }}
        <tr><td colspan="5">No releases found.</td></tr>
        {{ end }}
        </tbody>
      </table>
    </section>
  </section>
</main>
</body>
</html>`
