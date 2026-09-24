package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupThemeBakeSite(t *testing.T, files map[string]string) string {
	t.Helper()
	site := t.TempDir()
	t.Chdir(site)
	for name, content := range files {
		path := filepath.Join(site, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	prevCfg, prevMerge, prevRebuild := cfgFile, mergeConfigFiles, serveRequestFullRebuild
	cfgFile, mergeConfigFiles = "", nil
	setServePreview(nil)
	t.Cleanup(func() {
		cfgFile, mergeConfigFiles, serveRequestFullRebuild = prevCfg, prevMerge, prevRebuild
		setServePreview(nil)
	})
	return site
}

func doThemeBake(t *testing.T, method, body string, headers map[string]string) (*httptest.ResponseRecorder, themeBakeResponse) {
	t.Helper()
	req := httptest.NewRequest(method, "http://localhost:8000"+serveThemeBakeEndpoint, strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:52100"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	handleThemeBake(rec, req)
	var resp themeBakeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return rec, resp
}

func TestHandleThemeBake_WritesIncludedThemeFile(t *testing.T) {
	site := setupThemeBakeSite(t, map[string]string{
		"markata-go.toml":       "[markata-go]\ntitle = \"Site\"\ninclude = [\"config/*.toml\"]\n",
		"config/site.toml":      "[markata-go]\ndescription = \"d\"\n",
		"config/theme.toml":     "# my theme\n[markata-go.theme]\npalette = \"old\" # keep\nseasonal = true\n",
		"config/zz-extras.toml": "[markata-go.feed_defaults]\nitems_per_page = 5\n",
	})
	rebuilt := 0
	serveRequestFullRebuild = func() { rebuilt++ }

	rec, info := doThemeBake(t, http.MethodGet, "", nil)
	if rec.Code != http.StatusOK || info.Target != filepath.Join("config", "theme.toml") {
		t.Fatalf("GET = %d %+v, want config/theme.toml", rec.Code, info)
	}

	body := `{"palette":"nord-dark","palette_light":"nord-light","palette_dark":"nord-dark","fallback_mode":"dark","aesthetic":"brutal","fontpack":"typewriter","text_size":"large"}`
	rec, resp := doThemeBake(t, http.MethodPost, body, map[string]string{"Origin": "http://localhost:8000"})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST = %d %+v", rec.Code, resp)
	}
	if rebuilt != 1 {
		t.Errorf("rebuild requested %d times, want 1", rebuilt)
	}
	got, err := os.ReadFile(filepath.Join(site, "config", "theme.toml"))
	if err != nil {
		t.Fatal(err)
	}
	want := "# my theme\n[markata-go.theme]\npalette = \"nord-dark\" # keep\nseasonal = false\npalette_light = \"nord-light\"\npalette_dark = \"nord-dark\"\nfallback_mode = \"dark\"\naesthetic = \"brutal\"\nfontpack = \"typewriter\"\ntext_size = \"large\"\n"
	if string(got) != want {
		t.Errorf("config/theme.toml =\n%s\nwant:\n%s", got, want)
	}
	root, _ := os.ReadFile(filepath.Join(site, "markata-go.toml"))
	if strings.Contains(string(root), "theme") {
		t.Errorf("root config should not be edited:\n%s", root)
	}
}

func TestHandleThemeBake_SeasonalKeepsPalettes(t *testing.T) {
	site := setupThemeBakeSite(t, map[string]string{
		"markata-go.toml": "[markata-go.theme]\npalette = \"catppuccin\"\n",
	})
	body := `{"seasonal":true,"palette":"ignored","fallback_mode":"light"}`
	rec, resp := doThemeBake(t, http.MethodPost, body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST = %d %+v", rec.Code, resp)
	}
	got, _ := os.ReadFile(filepath.Join(site, "markata-go.toml"))
	want := "[markata-go.theme]\npalette = \"catppuccin\"\nseasonal = true\nfallback_mode = \"light\"\n"
	if string(got) != want {
		t.Errorf("markata-go.toml =\n%s\nwant:\n%s", got, want)
	}
}

func TestHandleThemeBake_CreatesConfigForConfigLessSite(t *testing.T) {
	site := setupThemeBakeSite(t, nil)
	if home, err := os.UserHomeDir(); err == nil {
		if _, err := os.Stat(filepath.Join(home, ".config", "markata-go", "config.toml")); err == nil {
			t.Skip("user-level markata-go config would be discovered")
		}
	}
	rec, resp := doThemeBake(t, http.MethodPost, `{"palette":"nord-dark","fallback_mode":"dark"}`, nil)
	if rec.Code != http.StatusOK || resp.Target != "markata-go.toml" {
		t.Fatalf("POST = %d %+v", rec.Code, resp)
	}
	got, _ := os.ReadFile(filepath.Join(site, "markata-go.toml"))
	if string(got) != "[markata-go.theme]\npalette = \"nord-dark\"\nfallback_mode = \"dark\"\n" {
		t.Errorf("markata-go.toml =\n%s", got)
	}
}

func TestHandleThemeBake_RejectsUnsafeRequests(t *testing.T) {
	site := setupThemeBakeSite(t, map[string]string{"markata-go.toml": "[markata-go]\ntitle = \"Site\"\n"})
	tests := []struct {
		name    string
		method  string
		body    string
		headers map[string]string
		status  int
	}{
		{"cross origin", http.MethodPost, `{"palette":"x"}`, map[string]string{"Origin": "https://evil.example"}, http.StatusForbidden},
		{"cross site fetch", http.MethodPost, `{"palette":"x"}`, map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"bad palette", http.MethodPost, `{"palette":"x\n[markata-go]"}`, nil, http.StatusBadRequest},
		{"bad mode", http.MethodPost, `{"fallback_mode":"purple"}`, nil, http.StatusBadRequest},
		{"unknown fontpack", http.MethodPost, `{"fontpack":"not-a-pack"}`, nil, http.StatusBadRequest},
		{"unknown field", http.MethodPost, `{"output_dir":"/"}`, nil, http.StatusBadRequest},
		{"wrong content type", http.MethodPost, `{"palette":"x"}`, map[string]string{"Content-Type": "text/plain"}, http.StatusUnsupportedMediaType},
		{"method", http.MethodDelete, "", nil, http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, resp := doThemeBake(t, tt.method, tt.body, tt.headers)
			if rec.Code != tt.status {
				t.Errorf("status = %d (%+v), want %d", rec.Code, resp, tt.status)
			}
		})
	}
	got, _ := os.ReadFile(filepath.Join(site, "markata-go.toml"))
	if string(got) != "[markata-go]\ntitle = \"Site\"\n" {
		t.Errorf("config changed by rejected requests:\n%s", got)
	}
}

func TestInjectDevScripts_ExposesThemeBakeEndpoint(t *testing.T) {
	html := injectDevScripts("<html><head></head><body></body></html>", BuildStatus{})
	if !strings.Contains(html, `window.__markataThemeBakeEndpoint = "`+serveThemeBakeEndpoint+`"`) {
		t.Errorf("dev scripts missing bake endpoint:\n%s", html)
	}
}
