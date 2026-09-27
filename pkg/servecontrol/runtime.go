// Package servecontrol holds the live state shared by serve clients.
package servecontrol

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxJobs              = 256
	maxLogs              = 5000
	maxJobLogs           = 1000
	logBroadcastInterval = 50 * time.Millisecond
)

// State describes a server, job, step, or page lifecycle state.
type State string

const (
	StateQueued    State = "queued"
	StateRunning   State = "running"
	StateSuccess   State = "success"
	StateWarning   State = "warning"
	StateFailed    State = "failed"
	StateCancelled State = "canceled"
)

type ServerState struct {
	Status   State  `json:"status"`
	Address  string `json:"address,omitempty"`
	AdminURL string `json:"admin_url,omitempty"`
	Message  string `json:"message,omitempty"`
}

// SiteState describes the latest build independently of HTTP listener state.
type SiteState struct {
	Status    State  `json:"status"`
	Message   string `json:"message,omitempty"`
	PageCount int    `json:"page_count"`
}

type Step struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	State     State     `json:"state"`
	StartedAt time.Time `json:"started_at,omitempty"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
}

type Diagnostic struct {
	Code         string `json:"code,omitempty"`
	Severity     string `json:"severity"`
	Message      string `json:"message"`
	Explanation  string `json:"explanation,omitempty"`
	File         string `json:"file,omitempty"`
	Line         int    `json:"line,omitempty"`
	Column       int    `json:"column,omitempty"`
	Page         string `json:"page,omitempty"`
	JobID        string `json:"job_id,omitempty"`
	StepID       string `json:"step_id,omitempty"`
	SuggestedFix string `json:"suggested_fix,omitempty"`
}

type LogEntry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level,omitempty"`
	Message string    `json:"message"`
	JobID   string    `json:"job_id,omitempty"`
	StepID  string    `json:"step_id,omitempty"`
}

type Page struct {
	Path        string       `json:"path"`
	URL         string       `json:"url,omitempty"`
	Status      State        `json:"status,omitempty"`
	JobID       string       `json:"job_id,omitempty"`
	LastJobID   string       `json:"last_job_id,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

// Feed and FeedEntry project lifecycle feed metadata for session clients.
type Feed struct {
	Name    string      `json:"name"`
	Title   string      `json:"title,omitempty"`
	Path    string      `json:"path,omitempty"`
	Status  State       `json:"status,omitempty"`
	Entries []FeedEntry `json:"entries"`
}

type FeedEntry struct {
	Path  string    `json:"path"`
	Title string    `json:"title,omitempty"`
	URL   string    `json:"url,omitempty"`
	Date  time.Time `json:"date,omitempty"`
}

type JobSpec struct {
	Name    string   `json:"name"`
	Type    string   `json:"type,omitempty"`
	Trigger string   `json:"trigger,omitempty"`
	Pages   []string `json:"pages,omitempty"`
}

type Job struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        string       `json:"type,omitempty"`
	Trigger     string       `json:"trigger,omitempty"`
	State       State        `json:"state"`
	Steps       []Step       `json:"steps,omitempty"`
	Logs        []LogEntry   `json:"logs,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
	Pages       []string     `json:"pages,omitempty"`
	QueuedAt    time.Time    `json:"queued_at"`
	StartedAt   time.Time    `json:"started_at,omitempty"`
	EndedAt     time.Time    `json:"ended_at,omitempty"`
}

type Snapshot struct {
	Server      ServerState       `json:"server"`
	Site        SiteState         `json:"site"`
	Theme       map[string]string `json:"theme,omitempty"`
	Jobs        []Job             `json:"jobs"`
	Diagnostics []Diagnostic      `json:"diagnostics"`
	Pages       []Page            `json:"pages"`
	Feeds       []Feed            `json:"feeds"`
	Logs        []LogEntry        `json:"logs"`
}

type ActionRequest struct {
	Kind  string `json:"kind"`
	JobID string `json:"job_id,omitempty"`
	Page  string `json:"page,omitempty"`
}

var ErrNoActionHandler = errors.New("serve action handler is unavailable")

// Runtime stores one serve session's state. All client snapshots are detached
// from its internal slices, including nested job and page data.
type Runtime struct {
	mu                  sync.RWMutex
	state               Snapshot
	nextID              uint64
	subscribers         map[chan Snapshot]struct{}
	action              func(ActionRequest) error
	lastBroadcast       time.Time
	logBroadcastTimer   *time.Timer
	pendingLogBroadcast bool
}

func NewRuntime() *Runtime {
	return &Runtime{
		state:       Snapshot{Jobs: []Job{}, Diagnostics: []Diagnostic{}, Pages: []Page{}, Feeds: []Feed{}, Logs: []LogEntry{}},
		subscribers: make(map[chan Snapshot]struct{}),
	}
}

func (r *Runtime) Snapshot() Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneSnapshot(r.state)
}

// ReplaceSnapshot imports an authoritative state, such as Builder Admin's
// persisted queue and history. Existing subscribers and action wiring remain
// attached to the runtime. The input is cloned so later caller edits cannot
// change the live state.
func (r *Runtime) ReplaceSnapshot(snapshot Snapshot) {
	r.mu.Lock()
	r.state = cloneSnapshot(snapshot)
	for i := range r.state.Jobs {
		job := &r.state.Jobs[i]
		r.advanceIDLocked(job.ID, "job-")
		for j := range job.Steps {
			r.advanceIDLocked(job.Steps[j].ID, "step-")
		}
	}
	r.broadcastLocked()
	r.mu.Unlock()
}

func (r *Runtime) advanceIDLocked(id, prefix string) {
	number, err := strconv.ParseUint(strings.TrimPrefix(id, prefix), 10, 64)
	if err == nil && strings.HasPrefix(id, prefix) && number > r.nextID {
		r.nextID = number
	}
}

// Subscribe returns an immediate snapshot and subsequent latest-state updates.
// A slow subscriber skips intermediate versions without blocking a build.
func (r *Runtime) Subscribe() (updates <-chan Snapshot, unsubscribe func()) {
	ch := make(chan Snapshot, 1)
	r.mu.Lock()
	if r.subscribers == nil {
		r.subscribers = make(map[chan Snapshot]struct{})
	}
	r.subscribers[ch] = struct{}{}
	ch <- cloneSnapshot(r.state)
	r.lastBroadcast = time.Now()
	r.mu.Unlock()
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			r.mu.Lock()
			delete(r.subscribers, ch)
			if len(r.subscribers) == 0 {
				r.cancelLogBroadcastLocked()
			}
			close(ch)
			r.mu.Unlock()
		})
	}
}

func (r *Runtime) SetActionHandler(handler func(ActionRequest) error) {
	r.mu.Lock()
	r.action = handler
	r.mu.Unlock()
}

func (r *Runtime) Trigger(request ActionRequest) error {
	if request.Kind == "" {
		return errors.New("serve action kind is required")
	}
	r.mu.RLock()
	handler := r.action
	r.mu.RUnlock()
	if handler == nil {
		return ErrNoActionHandler
	}
	return handler(request)
}

func (r *Runtime) SetServer(state ServerState) {
	r.mu.Lock()
	r.state.Server = state
	r.broadcastLocked()
	r.mu.Unlock()
}

func (r *Runtime) SetSite(state SiteState) {
	r.mu.Lock()
	r.state.Site = state
	r.broadcastLocked()
	r.mu.Unlock()
}

// SetTheme publishes resolved semantic colors for all session clients.
func (r *Runtime) SetTheme(theme map[string]string) {
	r.mu.Lock()
	r.state.Theme = make(map[string]string, len(theme))
	for key, value := range theme {
		r.state.Theme[key] = value
	}
	r.broadcastLocked()
	r.mu.Unlock()
}

func (r *Runtime) QueueJob(spec JobSpec) string {
	r.mu.Lock()
	r.nextID++
	id := fmt.Sprintf("job-%d", r.nextID)
	r.state.Jobs = append(r.state.Jobs, Job{
		ID: id, Name: spec.Name, Type: spec.Type, Trigger: spec.Trigger,
		State: StateQueued, QueuedAt: time.Now().UTC(), Pages: append([]string(nil), spec.Pages...),
	})
	if len(r.state.Jobs) > maxJobs {
		r.state.Jobs = append([]Job(nil), r.state.Jobs[len(r.state.Jobs)-maxJobs:]...)
	}
	r.broadcastLocked()
	r.mu.Unlock()
	return id
}

func (r *Runtime) StartJob(id string) {
	r.mu.Lock()
	if job := r.jobLocked(id); job != nil && job.State == StateQueued {
		job.State = StateRunning
		job.StartedAt = time.Now().UTC()
		r.broadcastLocked()
	}
	r.mu.Unlock()
}

func (r *Runtime) StartStep(jobID, name string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	job := r.jobLocked(jobID)
	if job == nil || job.State != StateRunning {
		return ""
	}
	r.nextID++
	id := fmt.Sprintf("step-%d", r.nextID)
	job.Steps = append(job.Steps, Step{ID: id, Name: name, State: StateRunning, StartedAt: time.Now().UTC()})
	r.broadcastLocked()
	return id
}

func (r *Runtime) FinishStep(jobID, stepID string, state State) {
	r.mu.Lock()
	if job := r.jobLocked(jobID); job != nil {
		for i := range job.Steps {
			step := &job.Steps[i]
			if step.ID == stepID && step.State == StateRunning {
				step.State = state
				step.EndedAt = time.Now().UTC()
				r.broadcastLocked()
				break
			}
		}
	}
	r.mu.Unlock()
}

func (r *Runtime) FinishJob(id string, state State) {
	r.mu.Lock()
	if job := r.jobLocked(id); job != nil && (job.State == StateQueued || job.State == StateRunning) {
		job.State = state
		job.EndedAt = time.Now().UTC()
		r.broadcastLocked()
	}
	r.mu.Unlock()
}

// SetJobPages attaches the pages affected by a completed build to its history.
func (r *Runtime) SetJobPages(id string, pages []string) {
	r.mu.Lock()
	if job := r.jobLocked(id); job != nil {
		job.Pages = append([]string(nil), pages...)
		r.broadcastLocked()
	}
	r.mu.Unlock()
}

func (r *Runtime) AddLog(entry LogEntry) {
	r.mu.Lock()
	if entry.Time.IsZero() {
		entry.Time = time.Now().UTC()
	}
	r.state.Logs = appendBounded(r.state.Logs, entry, maxLogs)
	if job := r.jobLocked(entry.JobID); job != nil {
		job.Logs = appendBounded(job.Logs, entry, maxJobLogs)
	}
	r.broadcastLogLocked()
	r.mu.Unlock()
}

func (r *Runtime) AddDiagnostic(diagnostic Diagnostic) {
	r.mu.Lock()
	r.state.Diagnostics = append(r.state.Diagnostics, diagnostic)
	if job := r.jobLocked(diagnostic.JobID); job != nil {
		job.Diagnostics = append(job.Diagnostics, diagnostic)
	}
	if diagnostic.Page != "" || diagnostic.File != "" {
		path := diagnostic.Page
		if path == "" {
			path = diagnostic.File
		}
		page := r.pageLocked(path)
		page.Diagnostics = append(page.Diagnostics, diagnostic)
		if diagnostic.JobID != "" {
			page.JobID = diagnostic.JobID
			page.LastJobID = diagnostic.JobID
		}
		if diagnostic.Severity == "error" {
			page.Status = StateFailed
		} else if diagnostic.Severity == "warning" && page.Status != StateFailed {
			page.Status = StateWarning
		}
	}
	r.broadcastLocked()
	r.mu.Unlock()
}

// SetPages replaces current page state. Job and session diagnostics retain
// history; page diagnostics describe the latest build that touched each page.
func (r *Runtime) SetPages(pages []Page) {
	r.mu.Lock()
	r.state.Pages = make([]Page, len(pages))
	for i, page := range pages {
		r.state.Pages[i] = clonePage(page)
	}
	r.broadcastLocked()
	r.mu.Unlock()
}

// SetFeeds replaces feed inventory using projected lifecycle metadata.
func (r *Runtime) SetFeeds(feeds []Feed) {
	r.mu.Lock()
	r.state.Feeds = cloneFeeds(feeds)
	r.broadcastLocked()
	r.mu.Unlock()
}

func (r *Runtime) jobLocked(id string) *Job {
	for i := len(r.state.Jobs) - 1; i >= 0; i-- {
		if r.state.Jobs[i].ID == id {
			return &r.state.Jobs[i]
		}
	}
	return nil
}

func (r *Runtime) pageLocked(path string) *Page {
	for i := range r.state.Pages {
		if r.state.Pages[i].Path == path {
			return &r.state.Pages[i]
		}
	}
	r.state.Pages = append(r.state.Pages, Page{Path: path})
	return &r.state.Pages[len(r.state.Pages)-1]
}

func (r *Runtime) broadcastLocked() {
	r.cancelLogBroadcastLocked()
	if len(r.subscribers) == 0 {
		return
	}
	r.lastBroadcast = time.Now()
	for ch := range r.subscribers {
		snapshot := cloneSnapshot(r.state)
		select {
		case ch <- snapshot:
		default:
			select {
			case <-ch:
			default:
			}
			ch <- snapshot
		}
	}
}

// Log updates are coalesced while subscribers are active. The timer ensures
// the last line is eventually delivered even when logging stops abruptly.
func (r *Runtime) broadcastLogLocked() {
	if len(r.subscribers) == 0 {
		return
	}
	delay := logBroadcastInterval - time.Since(r.lastBroadcast)
	if delay <= 0 {
		r.broadcastLocked()
		return
	}
	if r.pendingLogBroadcast {
		return
	}
	r.pendingLogBroadcast = true
	r.logBroadcastTimer = time.AfterFunc(delay, func() {
		r.mu.Lock()
		if r.pendingLogBroadcast {
			r.broadcastLocked()
		}
		r.mu.Unlock()
	})
}

func (r *Runtime) cancelLogBroadcastLocked() {
	r.pendingLogBroadcast = false
	if r.logBroadcastTimer != nil {
		r.logBroadcastTimer.Stop()
		r.logBroadcastTimer = nil
	}
}

func appendBounded[T any](items []T, item T, limit int) []T {
	if len(items) == limit {
		copy(items, items[1:])
		items[len(items)-1] = item
		return items
	}
	return append(items, item)
}

func cloneSnapshot(source Snapshot) Snapshot {
	copyOf := source
	if source.Theme != nil {
		copyOf.Theme = make(map[string]string, len(source.Theme))
		for key, value := range source.Theme {
			copyOf.Theme[key] = value
		}
	}
	copyOf.Jobs = make([]Job, len(source.Jobs))
	for i := range source.Jobs {
		job := &source.Jobs[i]
		copyOf.Jobs[i] = *job
		copyOf.Jobs[i].Steps = append([]Step(nil), job.Steps...)
		copyOf.Jobs[i].Logs = append([]LogEntry(nil), job.Logs...)
		copyOf.Jobs[i].Diagnostics = append([]Diagnostic(nil), job.Diagnostics...)
		copyOf.Jobs[i].Pages = append([]string(nil), job.Pages...)
	}
	copyOf.Diagnostics = append([]Diagnostic(nil), source.Diagnostics...)
	copyOf.Pages = make([]Page, len(source.Pages))
	for i, page := range source.Pages {
		copyOf.Pages[i] = clonePage(page)
	}
	copyOf.Feeds = cloneFeeds(source.Feeds)
	copyOf.Logs = append([]LogEntry(nil), source.Logs...)
	return copyOf
}

func cloneFeeds(feeds []Feed) []Feed {
	cloned := make([]Feed, len(feeds))
	for i, feed := range feeds {
		cloned[i] = feed
		cloned[i].Entries = append([]FeedEntry(nil), feed.Entries...)
	}
	return cloned
}

func clonePage(page Page) Page {
	page.Diagnostics = append([]Diagnostic(nil), page.Diagnostics...)
	return page
}
