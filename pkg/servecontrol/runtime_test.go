package servecontrol

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRuntime_JobLifecycleAndHistory(t *testing.T) {
	r := NewRuntime()
	first := r.QueueJob(JobSpec{Name: "initial build", Type: "build", Trigger: "startup", Pages: []string{"posts/a.md"}})
	r.StartJob(first)
	step := r.StartStep(first, "load")
	if step == "" {
		t.Fatal("running job did not accept a step")
	}
	r.AddLog(LogEntry{JobID: first, StepID: step, Message: "loading"})
	r.FinishStep(first, step, StateFailed)
	r.FinishJob(first, StateFailed)
	second := r.QueueJob(JobSpec{Name: "rebuild", Trigger: "watch"})
	r.StartJob(second)
	r.FinishJob(second, StateSuccess)

	snapshot := r.Snapshot()
	if len(snapshot.Jobs) != 2 || snapshot.Jobs[0].State != StateFailed || snapshot.Jobs[1].State != StateSuccess {
		t.Fatalf("job history = %+v", snapshot.Jobs)
	}
	if snapshot.Jobs[0].Steps[0].State != StateFailed || snapshot.Jobs[0].StartedAt.IsZero() || snapshot.Jobs[0].EndedAt.IsZero() {
		t.Fatalf("first job lifecycle = %+v", snapshot.Jobs[0])
	}
	if len(snapshot.Jobs[0].Logs) != 1 || snapshot.Jobs[0].Logs[0].StepID != step {
		t.Fatalf("first job logs = %+v", snapshot.Jobs[0].Logs)
	}
	if len(snapshot.Jobs[0].Pages) != 1 || snapshot.Jobs[0].Pages[0] != "posts/a.md" {
		t.Fatalf("first job pages = %+v", snapshot.Jobs[0].Pages)
	}
}

func TestRuntime_DiagnosticsRemainAvailableAfterLogsAndRebuilds(t *testing.T) {
	r := NewRuntime()
	id := r.QueueJob(JobSpec{Name: "initial build"})
	r.StartJob(id)
	r.SetPages([]Page{{Path: "posts/a.md", Status: StateSuccess}})
	r.AddDiagnostic(Diagnostic{Code: "missing-alt-text", Severity: "warning", Message: "image needs alt text", File: "posts/a.md", Line: 7, JobID: id})
	for i := 0; i < maxLogs+10; i++ {
		r.AddLog(LogEntry{Message: fmt.Sprintf("line %d", i)})
	}
	r.FinishJob(id, StateWarning)
	next := r.QueueJob(JobSpec{Name: "rebuild"})
	r.StartJob(next)
	r.FinishJob(next, StateSuccess)

	snapshot := r.Snapshot()
	if len(snapshot.Logs) != maxLogs || snapshot.Logs[0].Message != "line 10" {
		t.Fatalf("bounded logs = %d, first = %+v", len(snapshot.Logs), snapshot.Logs[0])
	}
	if len(snapshot.Diagnostics) != 1 || snapshot.Diagnostics[0].JobID != id {
		t.Fatalf("diagnostics = %+v", snapshot.Diagnostics)
	}
	if len(snapshot.Jobs[0].Diagnostics) != 1 || len(snapshot.Pages[0].Diagnostics) != 1 {
		t.Fatalf("diagnostic relationships missing: job=%+v page=%+v", snapshot.Jobs[0], snapshot.Pages[0])
	}
	if snapshot.Pages[0].JobID != id || snapshot.Pages[0].Status != StateWarning {
		t.Fatalf("page state = %+v", snapshot.Pages[0])
	}
}

func TestRuntime_SetPagesClearsCurrentDiagnosticsButKeepsHistory(t *testing.T) {
	r := NewRuntime()
	id := r.QueueJob(JobSpec{Name: "build"})
	r.AddDiagnostic(Diagnostic{Code: "missing-alt-text", Severity: "warning", File: "posts/a.md", JobID: id})
	r.SetPages([]Page{{Path: "posts/a.md", Status: StateSuccess, JobID: "job-new"}})
	snapshot := r.Snapshot()
	if len(snapshot.Pages) != 1 || len(snapshot.Pages[0].Diagnostics) != 0 || snapshot.Pages[0].Status != StateSuccess {
		t.Fatalf("current page still has stale diagnostic: %+v", snapshot.Pages)
	}
	if len(snapshot.Diagnostics) != 1 || len(snapshot.Jobs[0].Diagnostics) != 1 {
		t.Fatalf("diagnostic history lost: %+v", snapshot)
	}
}

func TestRuntime_SnapshotsAndSubscribersAreDetached(t *testing.T) {
	r := NewRuntime()
	r.SetPages([]Page{{Path: "posts/a.md"}})
	r.SetFeeds([]Feed{{Name: "thoughts", Entries: []FeedEntry{{Path: "posts/a.md", Title: "A"}}}})
	id := r.QueueJob(JobSpec{Name: "build", Pages: []string{"posts/a.md"}})
	first, stopFirst := r.Subscribe()
	defer stopFirst()
	second, stopSecond := r.Subscribe()
	defer stopSecond()
	initial := <-first
	other := <-second
	initial.Jobs[0].Pages[0] = "changed"
	initial.Pages[0].Path = "changed"
	initial.Feeds[0].Entries[0].Title = "changed"
	if other.Jobs[0].Pages[0] != "posts/a.md" || other.Pages[0].Path != "posts/a.md" {
		t.Fatal("subscriber snapshots share mutable slices")
	}
	if other.Feeds[0].Entries[0].Title != "A" {
		t.Fatal("subscriber feed entries share mutable slices")
	}
	if actual := r.Snapshot(); actual.Jobs[0].ID != id || actual.Jobs[0].Pages[0] != "posts/a.md" || actual.Pages[0].Path != "posts/a.md" || actual.Feeds[0].Entries[0].Title != "A" {
		t.Fatalf("runtime state mutated by subscriber: %+v", actual)
	}

	for i := 0; i < 20; i++ {
		r.SetServer(ServerState{Status: StateRunning, Message: fmt.Sprintf("version %d", i)})
	}
	latest := <-first
	if latest.Server.Message != "version 19" {
		t.Fatalf("slow subscriber saw %q", latest.Server.Message)
	}
}

func TestRuntime_LogBroadcastEventuallyDeliversLatestState(t *testing.T) {
	r := NewRuntime()
	updates, unsubscribe := r.Subscribe()
	defer unsubscribe()
	<-updates

	for i := 0; i < 100; i++ {
		r.AddLog(LogEntry{Message: fmt.Sprintf("line %d", i)})
	}
	deadline := time.After(time.Second)
	for {
		select {
		case snapshot := <-updates:
			if len(snapshot.Logs) == 100 {
				if snapshot.Logs[99].Message != "line 99" {
					t.Fatalf("latest log = %+v", snapshot.Logs[99])
				}
				return
			}
		case <-deadline:
			t.Fatal("latest log state was not delivered")
		}
	}
}

func TestRuntime_StateChangeFlushesPendingLogs(t *testing.T) {
	r := NewRuntime()
	updates, unsubscribe := r.Subscribe()
	defer unsubscribe()
	<-updates
	r.AddLog(LogEntry{Message: "build failed"})
	r.SetServer(ServerState{Status: StateFailed, Message: "listen failed"})
	select {
	case snapshot := <-updates:
		if snapshot.Server.Status != StateFailed || len(snapshot.Logs) != 1 {
			t.Fatalf("state change did not flush logs: %+v", snapshot)
		}
	case <-time.After(time.Second):
		t.Fatal("state change was not broadcast immediately")
	}
}

func TestRuntime_ActionsAndConcurrentIngestion(t *testing.T) {
	r := NewRuntime()
	if err := r.Trigger(ActionRequest{Kind: "build"}); !errors.Is(err, ErrNoActionHandler) {
		t.Fatalf("missing handler error = %v", err)
	}
	var got ActionRequest
	r.SetActionHandler(func(request ActionRequest) error { got = request; return nil })
	request := ActionRequest{Kind: "rebuild-page", Page: "posts/a.md"}
	if err := r.Trigger(request); err != nil || got != request {
		t.Fatalf("trigger = %v, request = %+v", err, got)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := r.QueueJob(JobSpec{Name: "parallel"})
			r.StartJob(id)
			r.AddLog(LogEntry{JobID: id, Message: fmt.Sprint(i)})
			r.FinishJob(id, StateSuccess)
		}(i)
	}
	wg.Wait()
	if len(r.Snapshot().Jobs) != 8 {
		t.Fatalf("jobs = %d", len(r.Snapshot().Jobs))
	}
}

func TestRuntime_ReplaceSnapshotPreservesWiringAndCopiesInput(t *testing.T) {
	r := NewRuntime()
	updates, unsubscribe := r.Subscribe()
	defer unsubscribe()
	<-updates
	called := false
	r.SetActionHandler(func(ActionRequest) error { called = true; return nil })
	imported := Snapshot{
		Server: ServerState{Status: StateSuccess, Address: "127.0.0.1:8080"},
		Jobs: []Job{{ID: "job-42", Name: "persisted build", State: StateFailed,
			Steps: []Step{{ID: "step-43", Name: "render", State: StateFailed}},
			Pages: []string{"posts/a.md"}}},
		Pages: []Page{{Path: "posts/a.md", Diagnostics: []Diagnostic{{Code: "old-error"}}}},
	}
	r.ReplaceSnapshot(imported)
	imported.Jobs[0].Pages[0] = "changed"
	imported.Pages[0].Diagnostics[0].Code = "changed"

	select {
	case snapshot := <-updates:
		if snapshot.Jobs[0].Pages[0] != "posts/a.md" || snapshot.Pages[0].Diagnostics[0].Code != "old-error" {
			t.Fatalf("import aliased input: %+v", snapshot)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive imported snapshot")
	}
	if err := r.Trigger(ActionRequest{Kind: "build"}); err != nil || !called {
		t.Fatalf("action handler lost after import: called=%t err=%v", called, err)
	}
	if id := r.QueueJob(JobSpec{Name: "next"}); id != "job-44" {
		t.Fatalf("generated ID after import = %q", id)
	}
}

func TestRuntime_ThemeAndSiteSnapshotsAreDetached(t *testing.T) {
	r := NewRuntime()
	colors := map[string]string{"primary": "#123456"}
	r.SetTheme(colors)
	r.SetSite(SiteState{Status: StateFailed, Message: "render failed", PageCount: 3})
	colors["primary"] = "changed"
	snapshot := r.Snapshot()
	if snapshot.Theme["primary"] != "#123456" || snapshot.Site.Status != StateFailed {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	snapshot.Theme["primary"] = "mutated"
	if r.Snapshot().Theme["primary"] != "#123456" {
		t.Fatal("client modified runtime theme")
	}
}
