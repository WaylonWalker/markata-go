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
	"net/http/httputil"
	"net/url"
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
	defaultLeaderName   = "leader.json"
	defaultLockName     = "leader.lock"
	defaultListenHost   = "127.0.0.1"
	defaultListenPort   = 8080
	defaultReleaseKeep  = 25
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
	BuildStatus  string    `json:"build_status,omitempty"`
	RollbackOnly bool      `json:"rollback_only"`
}

type Service struct {
	cfg          Config
	executable   string
	statePath    string
	logDir       string
	overrideDir  string
	leaderPath   string
	queueCh      chan queueRequest
	watchMu      sync.Mutex
	watchChanged map[string]struct{}
	watchTimer   *time.Timer
	stateMu      sync.Mutex
	state        State
	leaderMu     sync.RWMutex
	leader       bool
	leaderCancel context.CancelFunc
	leaderLock   *os.File
	instanceID   string
	instanceAddr string
	server       *http.Server
}

type leaderRecord struct {
	InstanceID string    `json:"instance_id"`
	Addr       string    `json:"addr"`
	AcquiredAt time.Time `json:"acquired_at"`
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
		leaderPath:   filepath.Join(cfg.HistoryDir, defaultLeaderName),
		queueCh:      make(chan queueRequest, 128),
		watchChanged: make(map[string]struct{}),
		instanceID:   os.Getenv("POD_NAME"),
	}
	if s.instanceID == "" {
		s.instanceID = hostSuffix()
	}
	instanceHost := os.Getenv("POD_IP")
	if instanceHost == "" {
		instanceHost = cfg.Host
	}
	s.instanceAddr = fmt.Sprintf("%s:%d", instanceHost, cfg.Port)
	if err := os.MkdirAll(s.logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}
	if err := os.MkdirAll(s.overrideDir, 0o755); err != nil {
		return nil, fmt.Errorf("create override dir: %w", err)
	}
	lockPath := filepath.Join(cfg.HistoryDir, defaultLockName)
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open leader lock: %w", err)
	}
	s.leaderLock = lockFile
	if err := s.loadState(); err != nil {
		_ = lockFile.Close()
		return nil, err
	}
	return s, nil
}

func (s *Service) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	s.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go s.runLeadershipLoop(ctx)
	go func() {
		<-ctx.Done()
		s.releaseLeadership()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()
	err := s.server.ListenAndServe()
	if s.leaderLock != nil {
		_ = s.leaderLock.Close()
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Service) runLeadershipLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		s.tryBecomeLeader(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) tryBecomeLeader(ctx context.Context) {
	if s.isLeader() || s.leaderLock == nil {
		return
	}
	if err := syscall.Flock(int(s.leaderLock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return
	}
	if err := s.writeLeaderRecord(); err != nil {
		_ = syscall.Flock(int(s.leaderLock.Fd()), syscall.LOCK_UN)
		return
	}
	leaderCtx, cancel := context.WithCancel(ctx)
	s.leaderMu.Lock()
	s.leader = true
	s.leaderCancel = cancel
	s.leaderMu.Unlock()
	s.resumeLeaderSession()
	go s.worker(leaderCtx)
	for i := range s.cfg.RefreshTasks {
		go s.runRefreshScheduler(leaderCtx, s.cfg.RefreshTasks[i])
	}
	if s.cfg.WatchEnabled {
		go s.watchSource(leaderCtx)
	}
}

func (s *Service) releaseLeadership() {
	s.leaderMu.Lock()
	if !s.leader {
		s.leaderMu.Unlock()
		return
	}
	cancel := s.leaderCancel
	s.leader = false
	s.leaderCancel = nil
	s.leaderMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if s.leaderLock != nil {
		_ = syscall.Flock(int(s.leaderLock.Fd()), syscall.LOCK_UN)
	}
}

func (s *Service) isLeader() bool {
	s.leaderMu.RLock()
	defer s.leaderMu.RUnlock()
	return s.leader
}

func (s *Service) writeLeaderRecord() error {
	record := leaderRecord{
		InstanceID: s.instanceID,
		Addr:       s.instanceAddr,
		AcquiredAt: time.Now().UTC(),
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.leaderPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.leaderPath)
}

func (s *Service) readLeaderRecord() (leaderRecord, error) {
	data, err := os.ReadFile(s.leaderPath)
	if err != nil {
		return leaderRecord{}, err
	}
	var record leaderRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return leaderRecord{}, err
	}
	return record, nil
}

func (s *Service) resumeLeaderSession() {
	queued := s.recoverQueuedState()
	for _, item := range queued {
		if req, ok := s.queueRequestFromQueued(item); ok {
			s.queueCh <- req
		}
	}
}

func (s *Service) recoverQueuedState() []QueuedOperation {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	queued := append([]QueuedOperation(nil), s.state.Queue...)
	if s.state.Running != nil {
		s.state.Running = nil
		s.saveStateLocked()
	}
	return queued
}

func (s *Service) queueRequestFromQueued(queued QueuedOperation) (queueRequest, bool) {
	req := queueRequest{QueuedOperation: queued}
	switch queued.Kind {
	case "build", "rollback":
		return req, true
	case "refresh":
		task, ok := s.findRefreshTask(queued.TaskName)
		if !ok {
			return queueRequest{}, false
		}
		req.commandArgs = append([]string(nil), task.Args...)
		return req, true
	default:
		return queueRequest{}, false
	}
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
	state := s.viewState()
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"queue":  len(state.Queue),
		"leader": s.isLeader(),
	})
}

func (s *Service) handleState(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	state := s.viewState()
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
		if s.handleStandbyMutation(w, r) {
			return
		}
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
	if s.handleStandbyMutation(w, r) {
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
	if s.handleStandbyMutation(w, r) {
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
			return formatUITimestamp(t, time.Now().UTC())
		},
		"summaryPreview": func(lines []string) []string {
			if len(lines) <= 6 {
				return lines
			}
			return lines[len(lines)-6:]
		},
		"statusClass": uiStatusClass,
	}).Parse(indexHTML))
	state := s.viewState()
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

func (s *Service) handleStandbyMutation(w http.ResponseWriter, r *http.Request) bool {
	if s.isLeader() {
		return false
	}
	record, err := s.readLeaderRecord()
	if err != nil || record.Addr == "" || record.InstanceID == s.instanceID {
		http.Error(w, "builder-admin standby is waiting for the active leader", http.StatusServiceUnavailable)
		return true
	}
	target, err := url.Parse("http://" + record.Addr)
	if err != nil {
		http.Error(w, "builder-admin leader address is unavailable", http.StatusServiceUnavailable)
		return true
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(rw http.ResponseWriter, _ *http.Request, proxyErr error) {
		http.Error(rw, fmt.Sprintf("builder-admin leader proxy failed: %v", proxyErr), http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
	return true
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

func (s *Service) readPersistedState() (State, error) {
	data, err := os.ReadFile(s.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

func (s *Service) viewState() State {
	if s.isLeader() {
		return s.snapshotState()
	}
	state, err := s.readPersistedState()
	if err == nil {
		return state
	}
	return s.snapshotState()
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
	state, err := s.readPersistedState()
	if err != nil {
		return fmt.Errorf("read state: %w", err)
	}
	s.state = state
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
	type releaseBuildMeta struct {
		id       string
		status   string
		finished time.Time
	}
	buildByRelease := make(map[string]releaseBuildMeta)
	for _, record := range s.viewState().Builds {
		if record.ReleaseID != "" && buildByRelease[record.ReleaseID].id == "" {
			buildByRelease[record.ReleaseID] = releaseBuildMeta{
				id:       record.ID,
				status:   record.Status,
				finished: record.FinishedAt,
			}
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
		meta := buildByRelease[entry.Name()]
		createdAt := info.ModTime().UTC()
		if !meta.finished.IsZero() {
			createdAt = meta.finished.UTC()
		} else if parsed, ok := releaseTimestampFromID(entry.Name()); ok {
			createdAt = parsed
		}
		views = append(views, ReleaseView{
			ID:           entry.Name(),
			Path:         filepath.Join(releasesDir, entry.Name()),
			CreatedAt:    createdAt,
			Current:      entry.Name() == current,
			BuildID:      meta.id,
			BuildStatus:  meta.status,
			RollbackOnly: true,
		})
	}
	sort.Slice(views, func(i, j int) bool {
		if views[i].Current != views[j].Current {
			return views[i].Current
		}
		if !views[i].CreatedAt.Equal(views[j].CreatedAt) {
			return views[i].CreatedAt.After(views[j].CreatedAt)
		}
		return views[i].ID > views[j].ID
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

func formatUITimestamp(ts, now time.Time) string {
	if ts.IsZero() {
		return ""
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return fmt.Sprintf("%s (%s ago)", ts.UTC().Format(time.RFC3339), humanizeAge(now.Sub(ts)))
}

func humanizeAge(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Round(time.Second)/time.Second))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Round(time.Minute)/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Round(time.Hour)/time.Hour))
	default:
		return fmt.Sprintf("%dd", int(d.Round(24*time.Hour)/(24*time.Hour)))
	}
}

func uiStatusClass(value string) string {
	switch value {
	case "success", "live", "ready", "idle":
		return "status-success"
	case "running", "build", "refresh", "promote", "prepare", "prune":
		return "status-running"
	case "queued", "pending", "starting":
		return "status-queued"
	case "failed", "error", "cancelled":
		return "status-failed"
	default:
		return "status-neutral"
	}
}

func releaseTimestampFromID(id string) (time.Time, bool) {
	formats := []struct {
		layout string
		length int
	}{
		{layout: "20060102T150405Z", length: len("20060102T150405Z")},
		{layout: "20060102150405", length: len("20060102150405")},
	}
	for _, format := range formats {
		if len(id) < format.length {
			continue
		}
		candidate := id[:format.length]
		ts, err := time.Parse(format.layout, candidate)
		if err == nil {
			return ts.UTC(), true
		}
	}
	return time.Time{}, false
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Builder Admin</title>
  <link id="app-favicon" rel="icon" href="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 64 64'%3E%3Crect width='64' height='64' rx='16' fill='%23525562'/%3E%3Ccircle cx='32' cy='32' r='13' fill='none' stroke='white' stroke-width='6' stroke-linecap='round' stroke-dasharray='0.01 82'/%3E%3C/svg%3E">
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
      align-items: flex-end;
      margin-bottom: 20px;
      padding-bottom: 14px;
      border-bottom: 1px solid var(--line-soft);
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
    .title-meta {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 14px;
    }
    .meta-chip {
      display: inline-flex;
      gap: 8px;
      align-items: center;
      padding: 6px 10px;
      border: 1px solid var(--line-soft);
      border-radius: 999px;
      color: var(--muted);
      font-size: 0.78rem;
    }
    .meta-chip strong {
      margin: 0;
      color: var(--text);
      letter-spacing: 0;
      font-size: 0.78rem;
    }
    .hero {
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto;
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
      background: transparent;
      border: 1px solid var(--line-soft);
      border-radius: 16px;
      padding: 16px 18px;
      box-shadow: none;
      backdrop-filter: none;
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
      align-content: center;
      justify-content: flex-end;
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
    .workspace-head {
      margin-bottom: 10px;
    }
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
    .status-success { border-color: rgba(34,197,94,0.45); background: rgba(34,197,94,0.14); color: #bbf7d0; }
    .status-running { border-color: rgba(59,130,246,0.45); background: rgba(59,130,246,0.14); color: #bfdbfe; }
    .status-queued { border-color: rgba(245,158,11,0.45); background: rgba(245,158,11,0.14); color: #fde68a; }
    .status-failed { border-color: rgba(239,68,68,0.45); background: rgba(239,68,68,0.16); color: #fecaca; }
    .status-neutral { border-color: var(--line); background: rgba(255,255,255,0.04); color: var(--text); }
    .summary-cell { min-width: 0; }
    .summary-meta { color: var(--muted); font-size: 0.76rem; margin-bottom: 6px; }
    .summary-list { display: grid; gap: 6px; }
    .summary-list div { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: #e4e4e7; }
    .mono { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace; }
    .wide { overflow-x: auto; }
    .muted { color: var(--muted); }
    .time-stamp { white-space: nowrap; }
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
    .run-list { display: grid; gap: 8px; }
    .run {
      display: grid;
      grid-template-columns: auto minmax(15rem, 1fr) auto auto auto;
      gap: 14px;
      align-items: center;
      padding: 12px 14px;
      border: 1px solid var(--line-soft);
      border-radius: 10px;
      background: rgba(255,255,255,0.02);
    }
    .run:hover { border-color: var(--line); background: rgba(255,255,255,0.04); }
    .run-status { width: 10px; height: 10px; border-radius: 50%; background: #71717a; }
    .run-status.status-success { background: #3fb950; box-shadow: 0 0 0 3px rgba(63,185,80,0.12); }
    .run-status.status-running { background: #58a6ff; box-shadow: 0 0 0 3px rgba(88,166,255,0.12); }
    .run-status.status-queued { background: #d29922; box-shadow: 0 0 0 3px rgba(210,153,34,0.12); }
    .run-status.status-failed { background: #f85149; box-shadow: 0 0 0 3px rgba(248,81,73,0.12); }
    .run-title { min-width: 0; font-weight: 650; }
    .run-title span { color: var(--muted); font-weight: 400; }
    .run-meta { color: var(--muted); font-size: 0.82rem; white-space: nowrap; }
    .run-action { font-size: 0.82rem; font-weight: 600; white-space: nowrap; }
    details.run-details { grid-column: 2 / -1; color: var(--muted); font-size: 0.82rem; }
    details.run-details summary { cursor: pointer; width: fit-content; color: var(--muted); }
    details.run-details summary:hover { color: var(--text); }
    .detail-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; margin: 12px 0; }
    .detail-grid div { min-width: 0; }
    .detail-grid strong { margin-bottom: 3px; }
    .detail-error { color: #fecaca; }
    .detail-perf { max-height: 14rem; margin-top: 10px; }
    .sync-status { color: var(--muted); font-size: 0.78rem; }
    .tab-panel { display: none; }
    .tab-panel.is-active { display: block; }
    @media (max-width: 1200px) {
      .hero, .section-grid { grid-template-columns: 1fr 1fr; }
    }
    @media (max-width: 800px) {
      main { padding: 14px; }
      .topbar, .hero, .section-grid { grid-template-columns: 1fr; display: grid; }
      .topbar { gap: 14px; }
      table { min-width: 860px; }
      .run { grid-template-columns: auto minmax(0, 1fr) auto; gap: 8px 12px; }
      .run-meta { grid-column: 2; white-space: normal; }
      .run-action { grid-column: 3; grid-row: 1 / span 2; }
      details.run-details { grid-column: 2 / -1; }
      .detail-grid { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
<main>
  <div class="topbar">
    <div class="titleblock">
      <h1>Builder Admin</h1>
      <p>Queue-driven builds, release promotion, refresh scheduling, and search/build runtime controls for the live go.waylonwalker.com authoring loop.</p>
      <div class="title-meta">
        <div class="meta-chip">Current <strong id="current-release">{{ .CurrentID }}</strong></div>
        <div class="meta-chip">Queue <strong id="queue-count">{{ len .State.Queue }}</strong></div>
        <div class="meta-chip">Builds <strong id="build-count">{{ len .State.Builds }}</strong></div>
        <div class="meta-chip">Refreshes <strong id="refresh-count">{{ len .State.Refresh }}</strong></div>
        <div class="meta-chip">Releases <strong id="release-count">{{ len .Releases }}</strong></div>
      </div>
    </div>
    <div class="sync-status" id="sync-status">Live polling every 2s</div>
  </div>

  <div class="hero">
    <section class="card">
      <div class="panel-head"><h2>Live State</h2><span>current release and active worker</span></div>
      <div style="display:grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 18px;">
        <div>
          <strong>Live release</strong>
          <div class="value mono">{{ .CurrentID }}</div>
        </div>
        <div>
          <strong>Current path</strong>
          <div class="value mono" id="current-path">{{ .CurrentPath }}</div>
        </div>
        <div>
          <strong>Active work</strong>
          <div class="value" id="active-work">{{ if .State.Running }}{{ .State.Running.Kind }} <span class="pill {{ statusClass .State.Running.Phase }}">{{ .State.Running.Phase }}</span>{{ else }}<span class="pill {{ statusClass "idle" }}">idle</span>{{ end }}</div>
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

  <div class="section-grid">
  <section class="card wide">
    <div class="panel-head"><h2>Queue</h2><span>debounced watch + manual triggers</span></div>
    <table>
      <thead><tr><th>ID</th><th>Kind</th><th>Trigger</th><th>Detail</th><th>Changed</th><th>Queued</th></tr></thead>
      <tbody id="queue-body">
      {{ range .State.Queue }}
      <tr>
        <td><code>{{ .ID }}</code></td>
        <td>{{ .Kind }}</td>
        <td>{{ .TriggerType }}</td>
        <td>{{ .Detail }}</td>
        <td>{{ range .Changed }}<div><code>{{ . }}</code></div>{{ end }}</td>
          <td class="time-stamp">{{ since .EnqueuedAt }}</td>
      </tr>
      {{ else }}
      <tr><td colspan="6">Queue is empty.</td></tr>
      {{ end }}
      </tbody>
    </table>
  </section>

  <section class="card">
    <div class="panel-head"><h2>Running</h2><span>live worker</span></div>
    <div class="stack" id="running-panel">
      {{ if .State.Running }}
      <div><strong>ID</strong><div class="value mono">{{ .State.Running.ID }}</div></div>
      <div><strong>Kind</strong><div class="value">{{ .State.Running.Kind }}</div></div>
      <div><strong>Trigger</strong><div class="value">{{ .State.Running.TriggerType }}</div></div>
      <div><strong>Detail</strong><div class="value">{{ .State.Running.Detail }}</div></div>
      <div><strong>Started</strong><div class="value mono time-stamp">{{ since .State.Running.StartedAt }}</div></div>
      <div><strong>Phase</strong><div class="value"><span class="pill {{ statusClass .State.Running.Phase }}">{{ .State.Running.Phase }}</span></div></div>
      {{ else }}
      <div class="muted">No build or refresh is running right now.</div>
      {{ end }}
    </div>
  </section>
  </div>

  <section class="card wide tab-shell">
    <div class="panel-head workspace-head"><h2>Workspace</h2><span>switch between builds, refreshes, and releases</span></div>
    <nav class="tabs">
      <a href="#builds" data-tab-link="builds">Builds</a>
      <a href="#refresh-runs" data-tab-link="refresh-runs">Refresh Runs</a>
      <a href="#releases" data-tab-link="releases">Releases</a>
    </nav>

    <section id="builds" class="tab-panel" data-tab-panel="builds">
      <div class="run-list" id="builds-body">
        {{ range .State.Builds }}
        <article class="run">
          <span class="run-status {{ statusClass .Status }}" aria-label="{{ .Status }}"></span>
          <div class="run-title">{{ .Status }} <span>via {{ .TriggerType }}</span></div>
          <div class="run-meta">{{ since .FinishedAt }} · {{ msToSeconds .TotalMS }}</div>
          <div class="run-meta">{{ if .ReleaseID }}release {{ .ReleaseID }}{{ else }}no release{{ end }}</div>
          <div class="run-action">{{ if .LogPath }}<a href="/logs/{{ .LogPath }}">View log</a>{{ end }}</div>
          <details class="run-details">
            <summary>Details</summary>
            <div class="detail-grid">
              <div><strong>Build ID</strong><code>{{ .ID }}</code></div>
              <div><strong>Total</strong><span>{{ msToSeconds .TotalMS }}</span></div>
              <div><strong>Release</strong>{{ if .ReleaseID }}<code>{{ .ReleaseID }}</code>{{ else }}<span>Not published</span>{{ end }}</div>
              <div><strong>Queue wait</strong><span>{{ msToSeconds .QueueWaitMS }}</span></div>
              <div><strong>Prepare</strong><span>{{ msToSeconds .PrepareMS }}</span></div>
              <div><strong>Build</strong><span>{{ msToSeconds .BuildMS }}</span></div>
              <div><strong>Promote</strong><span>{{ msToSeconds .PromoteMS }}</span></div>
              <div><strong>Prune</strong><span>{{ msToSeconds .PruneMS }}</span></div>
            </div>
            {{ if .ChangedPaths }}<div><strong>Changed paths</strong>{{ range .ChangedPaths }}<code>{{ . }}</code> {{ end }}</div>{{ end }}
            {{ if .Error }}<div class="detail-error"><strong>Error</strong>{{ .Error }}</div>{{ end }}
            {{ if .PerfSummary }}<pre class="detail-perf">{{ range .PerfSummary }}{{ . }}
{{ end }}</pre>{{ end }}
          </details>
        </article>
        {{ else }}
        <div class="muted">No builds yet.</div>
        {{ end }}
      </div>
    </section>

    <section id="refresh-runs" class="tab-panel" data-tab-panel="refresh-runs">
      <table>
        <thead><tr><th>ID</th><th>Task</th><th>Status</th><th>Total</th><th>Logs</th><th>Build</th><th>Command</th></tr></thead>
        <tbody id="refresh-body">
        {{ range .State.Refresh }}
        <tr>
          <td><code>{{ .ID }}</code></td>
          <td>{{ .TaskName }}</td>
          <td><span class="pill {{ statusClass .Status }}">{{ .Status }}</span></td>
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

    <section id="releases" class="tab-panel" data-tab-panel="releases">
      <table>
        <thead><tr><th>ID</th><th>Current</th><th>Created</th><th>Build</th><th>Status</th><th>Action</th></tr></thead>
        <tbody id="releases-body">
        {{ range .Releases }}
        <tr>
          <td><code>{{ .ID }}</code></td>
          <td>{{ if .Current }}<span class="pill {{ statusClass "live" }}">live</span>{{ end }}</td>
          <td class="time-stamp">{{ since .CreatedAt }}</td>
          <td>{{ if .BuildID }}<code>{{ .BuildID }}</code>{{ end }}</td>
          <td>{{ if .BuildStatus }}<span class="pill {{ statusClass .BuildStatus }}">{{ .BuildStatus }}</span>{{ end }}</td>
          <td>{{ if not .Current }}<form method="post" action="/api/releases/{{ .ID }}/rollback"><button class="secondary" type="submit">Promote</button></form>{{ end }}</td>
        </tr>
        {{ else }}
        <tr><td colspan="6">No releases found.</td></tr>
        {{ end }}
        </tbody>
      </table>
    </section>
  </section>
</main>
<script>
  const favicon = document.getElementById('app-favicon');
  const syncStatus = document.getElementById('sync-status');
  const currentRelease = document.getElementById('current-release');
  const currentPath = document.getElementById('current-path');
  const activeWork = document.getElementById('active-work');
  const queueCount = document.getElementById('queue-count');
  const buildCount = document.getElementById('build-count');
  const refreshCount = document.getElementById('refresh-count');
  const releaseCount = document.getElementById('release-count');
  const queueBody = document.getElementById('queue-body');
  const runningPanel = document.getElementById('running-panel');
  const buildsBody = document.getElementById('builds-body');
  const refreshBody = document.getElementById('refresh-body');
  const releasesBody = document.getElementById('releases-body');

  function escapeHtml(value) {
    return String(value ?? '')
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll("'", '&#39;');
  }

  function fmtTime(value) {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toISOString().replace('.000', '') + ' (' + timeAgo(date) + ' ago)';
  }

  function fmtSeconds(ms) {
    return ((ms || 0) / 1000).toFixed(2) + 's';
  }

  function timeAgo(date) {
    const delta = Math.max(0, Date.now() - date.getTime());
    const seconds = Math.round(delta / 1000);
    if (seconds < 60) return seconds + 's';
    const minutes = Math.round(seconds / 60);
    if (minutes < 60) return minutes + 'm';
    const hours = Math.round(minutes / 60);
    if (hours < 24) return hours + 'h';
    return Math.round(hours / 24) + 'd';
  }

  function statusClass(value) {
    switch (value) {
      case 'success':
      case 'live':
      case 'ready':
      case 'idle':
        return 'status-success';
      case 'running':
      case 'build':
      case 'refresh':
      case 'promote':
      case 'prepare':
      case 'prune':
        return 'status-running';
      case 'queued':
      case 'pending':
      case 'starting':
        return 'status-queued';
      case 'failed':
      case 'error':
      case 'cancelled':
        return 'status-failed';
      default:
        return 'status-neutral';
    }
  }

  function statusPill(value) {
    return '<span class="pill ' + statusClass(value) + '">' + escapeHtml(value) + '</span>';
  }

  function faviconDataURL(svg) {
    return 'data:image/svg+xml,' + encodeURIComponent(svg);
  }

  function buildFaviconSVG(stateName) {
    const base = {
      idle: '#525562',
      queued: '#d97706',
      build: '#16a34a',
      refresh: '#2563eb',
      error: '#dc2626'
    }[stateName] || '#7c3aed';
    const icon = {
      idle: '<path d="M18 33l9 9 19-20" fill="none" stroke="white" stroke-width="6" stroke-linecap="round" stroke-linejoin="round"/>',
      queued: '<circle cx="22" cy="32" r="4" fill="white"/><circle cx="32" cy="32" r="4" fill="white" opacity="0.85"/><circle cx="42" cy="32" r="4" fill="white" opacity="0.7"/>',
      build: '<circle cx="32" cy="32" r="12" fill="none" stroke="white" stroke-width="6"/><path d="M32 12v9M32 43v9M12 32h9M43 32h9M18 18l6 6M40 40l6 6M46 18l-6 6M18 46l6-6" fill="none" stroke="white" stroke-width="4" stroke-linecap="round"/>',
      refresh: '<path d="M18 28a14 14 0 0 1 24-8" fill="none" stroke="white" stroke-width="6" stroke-linecap="round"/><path d="M43 13v12H31" fill="none" stroke="white" stroke-width="6" stroke-linecap="round" stroke-linejoin="round"/><path d="M46 36a14 14 0 0 1-24 8" fill="none" stroke="white" stroke-width="6" stroke-linecap="round"/><path d="M21 51V39h12" fill="none" stroke="white" stroke-width="6" stroke-linecap="round" stroke-linejoin="round"/>',
      error: '<path d="M32 18v18" fill="none" stroke="white" stroke-width="6" stroke-linecap="round"/><circle cx="32" cy="44" r="3.5" fill="white"/>'
    }[stateName] || '<path d="M20 20l24 24M44 20L20 44" fill="none" stroke="white" stroke-width="6" stroke-linecap="round"/>';
    const svg = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">' +
      '<rect width="64" height="64" rx="16" fill="' + base + '" />' +
      icon +
      '</svg>';
    return svg;
  }

  function updateFavicon(stateName) {
    if (!favicon) {
      return;
    }
    favicon.href = faviconDataURL(buildFaviconSVG(stateName));
  }

  function faviconState(state) {
    if (state && state.running) {
      return state.running.kind === 'refresh' ? 'refresh' : 'build';
    }
    if (state && Array.isArray(state.queue) && state.queue.length > 0) {
      return 'queued';
    }
    return 'idle';
  }

  function summaryPreview(lines) {
    if (!Array.isArray(lines)) return [];
    return lines.slice(-6);
  }

  function activateTabs() {
    const active = (location.hash || '#builds').slice(1);
    document.querySelectorAll('[data-tab-link]').forEach((link) => {
      link.classList.toggle('active', link.dataset.tabLink === active);
    });
    document.querySelectorAll('[data-tab-panel]').forEach((panel) => {
      panel.classList.toggle('is-active', panel.dataset.tabPanel === active);
    });
  }

  function renderQueue(items) {
    if (!items || !items.length) {
      queueBody.innerHTML = '<tr><td colspan="6">Queue is empty.</td></tr>';
      return;
    }
    queueBody.innerHTML = items.map((item) => {
      const changed = (item.changed || []).map((path) => '<div><code>' + escapeHtml(path) + '</code></div>').join('');
      return '<tr>' +
        '<td><code>' + escapeHtml(item.id) + '</code></td>' +
        '<td>' + escapeHtml(item.kind) + '</td>' +
        '<td>' + escapeHtml(item.trigger_type) + '</td>' +
        '<td>' + escapeHtml(item.detail) + '</td>' +
        '<td>' + changed + '</td>' +
        '<td class="time-stamp">' + escapeHtml(fmtTime(item.enqueued_at)) + '</td>' +
      '</tr>';
    }).join('');
  }

  function renderRunning(running) {
    if (!running) {
      runningPanel.innerHTML = '<div class="muted">No build or refresh is running right now.</div>';
      activeWork.innerHTML = statusPill('idle');
      return;
    }
    activeWork.innerHTML = escapeHtml(running.kind) + ' ' + statusPill(running.phase);
    runningPanel.innerHTML = [
      ['ID', '<div class="value mono">' + escapeHtml(running.id) + '</div>'],
      ['Kind', '<div class="value">' + escapeHtml(running.kind) + '</div>'],
      ['Trigger', '<div class="value">' + escapeHtml(running.trigger_type) + '</div>'],
      ['Detail', '<div class="value">' + escapeHtml(running.detail) + '</div>'],
      ['Started', '<div class="value mono time-stamp">' + escapeHtml(fmtTime(running.started_at)) + '</div>'],
      ['Phase', '<div class="value">' + statusPill(running.phase) + '</div>']
    ].map(([label, value]) => '<div><strong>' + label + '</strong>' + value + '</div>').join('');
  }

  function renderBuilds(items) {
    if (!items || !items.length) {
      buildsBody.innerHTML = '<div class="muted">No builds yet.</div>';
      return;
    }
    buildsBody.innerHTML = items.map((item) => {
      const changed = (item.changed_paths || []).map((path) => '<code>' + escapeHtml(path) + '</code>').join(' ');
      const error = item.error ? '<div class="detail-error"><strong>Error</strong>' + escapeHtml(item.error) + '</div>' : '';
      const perf = Array.isArray(item.perf_summary) && item.perf_summary.length ? '<pre class="detail-perf">' + escapeHtml(item.perf_summary.join('\n')) + '</pre>' : '';
      const release = item.release_id ? '<code>' + escapeHtml(item.release_id) + '</code>' : '<span>Not published</span>';
      const releaseMeta = item.release_id ? 'release ' + escapeHtml(item.release_id) : 'no release';
      const phaseTiming = (label, value) => '<div><strong>' + label + '</strong><span>' + escapeHtml(fmtSeconds(value)) + '</span></div>';
      return '<article class="run">' +
        '<span class="run-status ' + statusClass(item.status) + '" aria-label="' + escapeHtml(item.status) + '"></span>' +
        '<div class="run-title">' + escapeHtml(item.status) + ' <span>via ' + escapeHtml(item.trigger_type) + '</span></div>' +
        '<div class="run-meta">' + escapeHtml(fmtTime(item.finished_at)) + ' · ' + escapeHtml(fmtSeconds(item.total_ms)) + '</div>' +
        '<div class="run-meta">' + releaseMeta + '</div>' +
        '<div class="run-action">' + (item.log_path ? '<a href="/logs/' + encodeURIComponent(item.log_path) + '">View log</a>' : '') + '</div>' +
        '<details class="run-details"><summary>Details</summary>' +
          '<div class="detail-grid">' +
            '<div><strong>Build ID</strong><code>' + escapeHtml(item.id) + '</code></div>' +
            '<div><strong>Release</strong>' + release + '</div>' +
            phaseTiming('Total', item.total_ms) +
            phaseTiming('Queue wait', item.queue_wait_ms) +
            phaseTiming('Prepare', item.prepare_ms) +
            phaseTiming('Build', item.build_ms) +
            phaseTiming('Promote', item.promote_ms) +
            phaseTiming('Prune', item.prune_ms) +
          '</div>' +
          (changed ? '<div><strong>Changed paths</strong>' + changed + '</div>' : '') + error + perf +
        '</details>' +
      '</article>';
    }).join('');
  }

  function renderRefresh(items) {
    if (!items || !items.length) {
      refreshBody.innerHTML = '<tr><td colspan="7">No refresh runs yet.</td></tr>';
      return;
    }
    refreshBody.innerHTML = items.map((item) => {
      const command = Array.isArray(item.command) && item.command.length ? item.command.join(' ') : '';
      return '<tr>' +
        '<td><code>' + escapeHtml(item.id) + '</code></td>' +
        '<td>' + escapeHtml(item.task_name) + '</td>' +
        '<td>' + statusPill(item.status) + '</td>' +
        '<td>' + escapeHtml(fmtSeconds(item.total_ms)) + '</td>' +
        '<td>' + (item.log_path ? '<a href="/logs/' + encodeURIComponent(item.log_path) + '">log</a>' : '') + '</td>' +
        '<td>' + (item.enqueued_build_id ? '<code>' + escapeHtml(item.enqueued_build_id) + '</code>' : '') + '</td>' +
        '<td class="mono muted">' + escapeHtml(command) + '</td>' +
      '</tr>';
    }).join('');
  }

  function renderReleases(items) {
    if (!items || !items.length) {
      releasesBody.innerHTML = '<tr><td colspan="5">No releases found.</td></tr>';
      return;
    }
    releasesBody.innerHTML = items.map((item) => {
      const action = item.current ? '' : '<form method="post" action="/api/releases/' + encodeURIComponent(item.id) + '/rollback"><button class="secondary" type="submit">Promote</button></form>';
      return '<tr>' +
        '<td><code>' + escapeHtml(item.id) + '</code></td>' +
        '<td>' + (item.current ? statusPill('live') : '') + '</td>' +
        '<td class="time-stamp">' + escapeHtml(fmtTime(item.created_at)) + '</td>' +
        '<td>' + (item.build_id ? '<code>' + escapeHtml(item.build_id) + '</code>' : '') + '</td>' +
        '<td>' + (item.build_status ? statusPill(item.build_status) : '') + '</td>' +
        '<td>' + action + '</td>' +
      '</tr>';
    }).join('');
  }

  function renderState(payload) {
    const state = payload.state || {};
    currentRelease.textContent = payload.current_release_id || '';
    currentPath.textContent = payload.current_release_path || '';
    queueCount.textContent = (state.queue || []).length;
    buildCount.textContent = (state.builds || []).length;
    refreshCount.textContent = (state.refresh || []).length;
    releaseCount.textContent = (payload.releases || []).length;
    renderQueue(state.queue || []);
    renderRunning(state.running || null);
    renderBuilds(state.builds || []);
    renderRefresh(state.refresh || []);
    renderReleases(payload.releases || []);
    syncStatus.textContent = 'Live polling every 2s';
    updateFavicon(faviconState(state));
  }

  async function pollState() {
    try {
      const response = await fetch('/api/state', { headers: { 'Accept': 'application/json' }, cache: 'no-store' });
      if (!response.ok) {
        throw new Error('HTTP ' + response.status);
      }
      const payload = await response.json();
      renderState(payload);
    } catch (error) {
      syncStatus.textContent = 'Sync stalled: ' + error.message;
      updateFavicon('error');
    }
  }

  window.addEventListener('hashchange', activateTabs);
  activateTabs();
  updateFavicon('idle');
  pollState();
  window.setInterval(pollState, 2000);
</script>
</body>
</html>`
