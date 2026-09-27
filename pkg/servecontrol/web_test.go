package servecontrol

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/servefix"
)

func TestWebHandler_SharedRuntimeStateAndActions(t *testing.T) {
	runtime := NewRuntime()
	runtime.SetServer(ServerState{Status: StateSuccess, Address: "127.0.0.1:8000"})
	runtime.SetSite(SiteState{Status: StateSuccess, PageCount: 42})
	runtime.SetTheme(map[string]string{"background": "#102030", "primary": "#f08040"})
	jobID := runtime.QueueJob(JobSpec{Name: "Initial build", Type: "build", Trigger: "serve"})
	runtime.StartJob(jobID)
	runtime.AddDiagnostic(Diagnostic{Code: "MARKATA-W014", Severity: "warning", Message: "Check frontmatter", Page: "posts/foo.md", JobID: jobID})
	runtime.SetPages([]Page{{Path: "posts/foo.md", URL: "/foo/", Status: StateSuccess, LastJobID: jobID}})
	var called ActionRequest
	runtime.SetActionHandler(func(request ActionRequest) error { called = request; return nil })
	handler := NewWebHandler(runtime)

	request := httptest.NewRequest(http.MethodGet, "/_markata/api/state", http.NoBody)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("state status = %d, body = %q", response.Code, response.Body.String())
	}
	var got Snapshot
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Jobs) != 1 || got.Jobs[0].ID != jobID || len(got.Diagnostics) != 1 || got.Diagnostics[0].JobID != jobID {
		t.Fatalf("web state did not mirror runtime: %+v", got)
	}
	if len(got.Pages) != 1 || got.Pages[0].URL != "/foo/" || len(got.Pages[0].Diagnostics) != 0 {
		t.Fatalf("web page did not reflect current page state: %+v", got.Pages)
	}
	if got.Site.PageCount != 42 || got.Theme["background"] != "#102030" {
		t.Fatalf("site or theme state did not reach web client: site=%+v theme=%+v", got.Site, got.Theme)
	}
	actionBody := bytes.NewBufferString(`{"kind":"rerun","job_id":"` + jobID + `"}`)
	request = httptest.NewRequest(http.MethodPost, "http://localhost/_markata/api/actions", actionBody)
	request.RemoteAddr = "127.0.0.1:45678"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || called.Kind != "rerun" || called.JobID != jobID {
		t.Fatalf("action status = %d, called = %+v", response.Code, called)
	}
}

func TestWebHTML_UsesOnlyLocalFonts(t *testing.T) {
	if strings.Contains(webHTML, "fonts.googleapis.com") || strings.Contains(webHTML, "@import") {
		t.Fatal("local dashboard must not load external font stylesheets")
	}
	if !strings.Contains(webHTML, "system-ui") || !strings.Contains(webHTML, "ui-monospace") {
		t.Fatal("local dashboard should use system sans and monospace font stacks")
	}
}

func TestWebHandler_FixPreviewApplyAndStaleProtection(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "post.md")
	original := "---\ndate: 09/26/2026\n---\n# Heading\n[link](//example.com)\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	handler := NewWebHandlerWithSourceRoot(NewRuntime(), root)
	request := fixRequest{Path: "post.md", Selection: servefix.Selection{All: true}}
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	response := serveLocalPost(handler, "/_markata/api/fixes/preview", body)
	if response.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", response.Code, response.Body.String())
	}
	var preview fixPreview
	if err := json.Unmarshal(response.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.After, "## Heading") || !strings.Contains(preview.After, "https://example.com") {
		t.Fatalf("preview did not contain safe fixes: %q", preview.After)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != original {
		t.Fatalf("preview changed source: %q err=%v", got, err)
	}
	if err := os.WriteFile(path, []byte(original+"changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	applyBody, err := json.Marshal(fixApplyRequest{Path: "post.md", Digest: preview.Digest, Selection: preview.Selection})
	if err != nil {
		t.Fatal(err)
	}
	response = serveLocalPost(handler, "/_markata/api/fixes/apply", applyBody)
	if response.Code != http.StatusConflict {
		t.Fatalf("stale apply status=%d body=%s", response.Code, response.Body.String())
	}
	updatedPlan, err := servefix.PlanFile(root, "post.md")
	if err != nil {
		t.Fatal(err)
	}
	applyBody, err = json.Marshal(fixApplyRequest{Path: "post.md", Digest: updatedPlan.Digest, Selection: servefix.Selection{All: true}})
	if err != nil {
		t.Fatal(err)
	}
	response = serveLocalPost(handler, "/_markata/api/fixes/apply", applyBody)
	if response.Code != http.StatusOK {
		t.Fatalf("apply status=%d body=%s", response.Code, response.Body.String())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "## Heading") || !strings.Contains(string(got), "https://example.com") {
		t.Fatalf("applied source = %q", got)
	}
}

func serveLocalPost(handler http.Handler, path string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "http://localhost"+path, bytes.NewReader(body))
	request.RemoteAddr = "127.0.0.1:45678"
	request.Header.Set("Origin", "http://localhost")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestWebHandler_RejectsDNSRebindingActionHost(t *testing.T) {
	runtime := NewRuntime()
	called := false
	runtime.SetActionHandler(func(ActionRequest) error { called = true; return nil })
	handler := NewWebHandler(runtime)
	request := httptest.NewRequest(http.MethodPost, "http://attacker.example/_markata/api/actions", strings.NewReader(`{"kind":"build"}`))
	request.RemoteAddr = "127.0.0.1:45678"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://attacker.example")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || called {
		t.Fatalf("DNS rebinding action status = %d, called = %v", response.Code, called)
	}
}

func TestWebHandler_OnlyServesItsPrefixAndEscapesDynamicState(t *testing.T) {
	runtime := NewRuntime()
	runtime.QueueJob(JobSpec{Name: "<script>alert(1)</script>", Type: "build"})
	handler := NewWebHandler(runtime)
	for _, path := range []string{"/", "/_markata/other"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, http.NoBody))
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d", path, response.Code)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/_markata/", http.NoBody))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Serve Control Center") {
		t.Fatalf("dashboard status = %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "<script>alert(1)</script>") {
		t.Fatal("runtime text appeared in initial HTML")
	}
	if !strings.Contains(response.Body.String(), "--markata-background:#09090b") || !strings.Contains(response.Body.String(), "--markata-focus:#93c5fd") {
		t.Fatal("local dashboard did not receive the shared semantic token stylesheet")
	}
	if !strings.Contains(response.Body.String(), "arr(item.diagnostics)") || !strings.Contains(response.Body.String(), "Open page") {
		t.Fatal("dashboard lacks current page diagnostics or page preview link")
	}
	for _, affordance := range []string{
		"applyTheme(state.snapshot.theme)", "data-page", "matches.slice(-200)", "site.page_count", "Session history",
		"--markata-text-primary", "routeHash()", "history.pushState", "history.replaceState",
		"window.addEventListener('popstate'", "window.addEventListener('hashchange'", "editableTarget(e.target)",
		"moveSelection(1)", "moveSelection(-1)", "keyboard-help", "const capabilities=Object.freeze",
	} {
		if !strings.Contains(response.Body.String(), affordance) {
			t.Errorf("dashboard lacks %q", affordance)
		}
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/_markata/api/state", http.NoBody))
	if strings.Contains(response.Body.String(), "<script>alert(1)</script>") {
		t.Fatal("JSON did not HTML-escape runtime text")
	}
}

func TestWebHandler_RejectsInvalidActionRequests(t *testing.T) {
	handler := NewWebHandler(NewRuntime())
	tests := []struct {
		method, contentType, body string
		status                    int
	}{
		{http.MethodGet, "", "", http.StatusMethodNotAllowed},
		{http.MethodPost, "text/plain", "{}", http.StatusUnsupportedMediaType},
		{http.MethodPost, "application/json", "{", http.StatusBadRequest},
		{http.MethodPost, "application/json", "{}", http.StatusBadRequest},
	}
	for _, tt := range tests {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(tt.method, "http://localhost/_markata/api/actions", strings.NewReader(tt.body))
		request.RemoteAddr = "127.0.0.1:45678"
		request.Header.Set("Content-Type", tt.contentType)
		request.Header.Set("Origin", "http://localhost")
		handler.ServeHTTP(response, request)
		if response.Code != tt.status {
			t.Errorf("%s %q: status = %d, want %d", tt.method, tt.body, response.Code, tt.status)
		}
	}
}

func TestWebHandler_RejectsCrossOriginActions(t *testing.T) {
	runtime := NewRuntime()
	called := false
	runtime.SetActionHandler(func(ActionRequest) error { called = true; return nil })
	handler := NewWebHandler(runtime)
	for _, tt := range []struct {
		origin, fetchSite string
	}{
		{"", ""},
		{"http://attacker.example", ""},
		{"http://example.com", "cross-site"},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/_markata/api/actions", strings.NewReader(`{"kind":"build"}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", tt.origin)
		request.Header.Set("Sec-Fetch-Site", tt.fetchSite)
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Errorf("origin %q, fetch site %q: status = %d", tt.origin, tt.fetchSite, response.Code)
		}
	}
	if called {
		t.Fatal("cross-origin action reached runtime")
	}
}
