package servecontrol

import (
	"context"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

func TestBrowserKeyboardNavigationAndHistory(t *testing.T) {
	browserPath := ""
	for _, candidate := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable"} {
		if path, err := exec.LookPath(candidate); err == nil {
			browserPath = path
			break
		}
	}
	if browserPath == "" {
		t.Skip("Chromium is unavailable")
	}

	runtime := NewRuntime()
	firstJob := runtime.QueueJob(JobSpec{Name: "First build", Type: "build"})
	runtime.StartJob(firstJob)
	runtime.FinishJob(firstJob, StateSuccess)
	secondJob := runtime.QueueJob(JobSpec{Name: "Second build", Type: "build"})
	runtime.StartJob(secondJob)
	runtime.FinishJob(secondJob, StateFailed)
	runtime.SetPages([]Page{{Path: "posts/first.md"}, {Path: "posts/second.md"}})
	actions := make(chan ActionRequest, 1)
	runtime.SetActionHandler(func(request ActionRequest) error {
		actions <- request
		return nil
	})
	server := httptest.NewServer(NewWebHandler(runtime))
	defer server.Close()

	allocator, cancelAllocator := chromedp.NewExecAllocator(context.Background(), append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(browserPath), chromedp.Headless, chromedp.NoFirstRun,
		chromedp.Flag("no-sandbox", true), chromedp.Flag("disable-dev-shm-usage", true),
	)...)
	defer cancelAllocator()
	browser, cancelBrowser := chromedp.NewContext(allocator)
	defer cancelBrowser()
	browser, cancelTimeout := context.WithTimeout(browser, 25*time.Second)
	defer cancelTimeout()

	var hash, section, filter string
	if err := chromedp.Run(browser,
		chromedp.EmulateViewport(1280, 900),
		chromedp.Navigate(server.URL+"/_markata/"),
		chromedp.WaitVisible(`#list [data-key]`),
		chromedp.KeyEvent("p"),
		chromedp.Evaluate(`location.hash`, &hash),
	); err != nil {
		t.Fatal(err)
	}
	if hash != "#/pages?item=page%3Aposts%2Ffirst.md" {
		t.Fatalf("p shortcut hash = %q", hash)
	}
	if err := chromedp.Run(browser,
		chromedp.KeyEvent("j"),
		chromedp.Evaluate(`location.hash`, &hash),
	); err != nil {
		t.Fatal(err)
	}
	if hash != "#/pages?item=page%3Aposts%2Fsecond.md" {
		t.Fatalf("j selection hash = %q", hash)
	}
	if err := chromedp.Run(browser,
		chromedp.KeyEvent(kb.Escape),
		chromedp.Evaluate(`location.hash`, &hash),
		chromedp.Evaluate(`document.querySelector('#title').textContent`, &section),
	); err != nil {
		t.Fatal(err)
	}
	if hash != "#/pages?item=page%3Aposts%2Ffirst.md" || section != "Pages" {
		t.Fatalf("Escape did not return to the prior selection: hash=%q title=%q", hash, section)
	}
	if err := chromedp.Run(browser,
		chromedp.Evaluate(`history.forward()`, nil),
		chromedp.Evaluate(`location.hash`, &hash),
	); err != nil {
		t.Fatal(err)
	}
	if hash != "#/pages?item=page%3Aposts%2Fsecond.md" {
		t.Fatalf("browser Forward did not restore selected item: %q", hash)
	}
	if err := chromedp.Run(browser,
		chromedp.KeyEvent(kb.Escape),
		chromedp.Evaluate(`location.hash`, &hash),
	); err != nil {
		t.Fatal(err)
	}
	if hash != "#/pages?item=page%3Aposts%2Ffirst.md" {
		t.Fatalf("second Escape did not return to prior route: %q", hash)
	}
	if err := chromedp.Run(browser,
		chromedp.Focus(`#filter`),
		chromedp.KeyEvent("p"),
		chromedp.Evaluate(`document.querySelector('#filter').value`, &filter),
		chromedp.Evaluate(`document.querySelector('#title').textContent`, &section),
	); err != nil {
		t.Fatal(err)
	}
	if filter != "p" || section != "Pages" {
		t.Fatalf("typing in filter was intercepted: filter=%q section=%q", filter, section)
	}
	if err := chromedp.Run(browser,
		chromedp.Focus(`#desktop-nav [data-section="feeds"]`),
		chromedp.KeyEvent(kb.Enter),
		chromedp.Evaluate(`document.querySelector('#title').textContent`, &section),
	); err != nil {
		t.Fatal(err)
	}
	if section != "Feeds" {
		t.Fatalf("Enter did not activate focused nav button: title=%q", section)
	}
	if err := chromedp.Run(browser,
		chromedp.Click(`#desktop-nav [data-section="jobs"]`),
		chromedp.KeyEvent("j"),
		chromedp.KeyEvent("r"),
	); err != nil {
		t.Fatal(err)
	}
	select {
	case action := <-actions:
		if action.Kind != "rerun" || action.JobID != secondJob {
			t.Fatalf("r shortcut action = %+v, want rerun for %s", action, secondJob)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("r shortcut did not rerun the selected completed job")
	}
	if err := chromedp.Run(browser, chromedp.Evaluate(`location.hash`, &hash)); err != nil {
		t.Fatal(err)
	}
	if hash != "#/jobs?item=job%3Ajob-2" {
		t.Fatalf("section navigation after browser interaction = %q", hash)
	}
	var focused string
	if err := chromedp.Run(browser,
		chromedp.Evaluate(`document.querySelector('#list [data-key="job:job-2"]').focus()`, nil),
		chromedp.Sleep(2300*time.Millisecond),
		chromedp.Evaluate(`document.activeElement?.dataset?.key || ''`, &focused),
	); err != nil {
		t.Fatal(err)
	}
	if focused != "job:job-2" {
		t.Fatalf("poll replaced focused row: focus = %q", focused)
	}
	runtime.QueueJob(JobSpec{Name: "Third build", Type: "build"})
	if err := chromedp.Run(browser,
		chromedp.Sleep(2300*time.Millisecond),
		chromedp.Evaluate(`document.activeElement?.dataset?.key || ''`, &focused),
	); err != nil {
		t.Fatal(err)
	}
	if focused != "job:job-2" {
		t.Fatalf("state update lost focused row: focus = %q", focused)
	}
}
