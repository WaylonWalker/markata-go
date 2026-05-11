// Package serveadmin provides admin CMS functionality for markata-go serve.
package serveadmin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/WaylonWalker/markata-go/pkg/contentedit"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/palettes"
	"github.com/WaylonWalker/markata-go/pkg/plugins"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v3"
)

var adminTemplates = template.Must(template.New("pages").Parse(authTemplate + dashboardTemplate + editorTemplate + settingsTemplate))

const (
	adminThemeDark  = "dark"
	adminThemeLight = "light"
)

type PageData struct {
	Title          string
	ThemeCSS       template.CSS
	NeedsSetup     bool
	NeedsLogin     bool
	IsDashboard    bool
	IsEditor       bool
	IsSettings     bool
	Posts          []PostInfo
	KnownTags      []string
	KnownPalettes  []string
	KnownDirs      []string
	KnownAuthors   []adminAuthorOption
	NewPostContext template.JS
	Post           PostEditData
	Settings       SettingsEditData
	Error          string
}

type PostInfo struct {
	Path      string `json:"path"`
	Title     string `json:"title"`
	Slug      string `json:"slug"`
	Date      string `json:"date"`
	Type      string `json:"type,omitempty"`
	Published bool   `json:"published"`
	Modified  string `json:"modified,omitempty"`
}

type PostEditData struct {
	Path        string `json:"path"`
	Title       string `json:"title"`
	Frontmatter string `json:"frontmatter"`
	Body        string `json:"body"`
	PreviewURL  string `json:"preview_url"`
	Slug        string `json:"slug"`
	GitStatus   string `json:"git_status,omitempty"`
	Hash        string `json:"base_hash"`
	Exists      bool   `json:"exists"`
}

type SettingsEditData struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Hash    string `json:"base_hash"`
	Exists  bool   `json:"exists"`
}

type savePostRequest struct {
	Path        string `json:"path"`
	Frontmatter string `json:"frontmatter"`
	Body        string `json:"body"`
	BaseHash    string `json:"base_hash"`
}

type adminKeyValueField struct {
	Key   string `json:"key"`
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type adminFrontmatterForm struct {
	Title       string               `json:"title"`
	Slug        string               `json:"slug"`
	Date        string               `json:"date"`
	Modified    string               `json:"modified"`
	Description string               `json:"description"`
	Published   bool                 `json:"published"`
	TemplateKey string               `json:"template_key"`
	Author      string               `json:"author"`
	Authors     []string             `json:"authors"`
	Tags        []string             `json:"tags"`
	Extras      []adminKeyValueField `json:"extras"`
}

type adminSettingsForm struct {
	Title                  string `json:"title"`
	Author                 string `json:"author"`
	URL                    string `json:"url"`
	Description            string `json:"description"`
	OutputDir              string `json:"output_dir"`
	TemplatesDir           string `json:"templates_dir"`
	AssetsDir              string `json:"assets_dir"`
	ThemePalette           string `json:"theme_palette"`
	ThemeLight             string `json:"theme_light"`
	ThemeDark              string `json:"theme_dark"`
	ThemeMode              string `json:"theme_mode"`
	SearchEnabled          bool   `json:"search_enabled"`
	SearchPosition         string `json:"search_position"`
	SearchPlaceholder      string `json:"search_placeholder"`
	PagefindBundleDir      string `json:"pagefind_bundle_dir"`
	PagefindVersion        string `json:"pagefind_version"`
	PagefindAutoInstall    bool   `json:"pagefind_auto_install"`
	ThemeSwitcherEnabled   bool   `json:"theme_switcher_enabled"`
	ThemeSwitcherPosition  string `json:"theme_switcher_position"`
	ThemeModeToggleEnabled bool   `json:"theme_mode_toggle_enabled"`
	ThemeIncludeAll        bool   `json:"theme_include_all"`
	FontFamily             string `json:"font_family"`
	FontHeadingFamily      string `json:"font_heading_family"`
	FontCodeFamily         string `json:"font_code_family"`
	FontSize               string `json:"font_size"`
	FontLineHeight         string `json:"font_line_height"`
}

type adminTemplateDefinition struct {
	Name        string                 `json:"name"`
	Label       string                 `json:"label"`
	Directory   string                 `json:"directory"`
	Frontmatter map[string]interface{} `json:"frontmatter"`
	Body        string                 `json:"body"`
	Source      string                 `json:"source"`
}

type adminAuthorOption struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Default bool   `json:"default"`
}

type adminConfigFile struct {
	MarkataGo adminMarkataConfig `toml:"markata-go"`
}

type adminMarkataConfig struct {
	Title        string             `toml:"title"`
	Author       string             `toml:"author"`
	URL          string             `toml:"url"`
	Description  string             `toml:"description"`
	OutputDir    string             `toml:"output_dir"`
	TemplatesDir string             `toml:"templates_dir"`
	AssetsDir    string             `toml:"assets_dir"`
	Search       adminSearchSection `toml:"search"`
	Theme        adminThemeSection  `toml:"theme"`
}

type adminSearchSection struct {
	Enabled     *bool                `toml:"enabled"`
	Position    string               `toml:"position"`
	Placeholder string               `toml:"placeholder"`
	Pagefind    adminPagefindSection `toml:"pagefind"`
}

type adminPagefindSection struct {
	BundleDir   string `toml:"bundle_dir"`
	Version     string `toml:"version"`
	AutoInstall *bool  `toml:"auto_install"`
}

type adminThemeSection struct {
	Palette      string                    `toml:"palette"`
	PaletteLight string                    `toml:"palette_light"`
	PaletteDark  string                    `toml:"palette_dark"`
	FallbackMode string                    `toml:"fallback_mode"`
	Switcher     adminThemeSwitcherSection `toml:"switcher"`
	Font         adminFontSection          `toml:"font"`
}

type adminThemeSwitcherSection struct {
	Enabled    *bool  `toml:"enabled"`
	ModeToggle *bool  `toml:"mode_toggle"`
	IncludeAll *bool  `toml:"include_all"`
	Position   string `toml:"position"`
}

type adminFontSection struct {
	Family        string `toml:"family"`
	HeadingFamily string `toml:"heading_family"`
	CodeFamily    string `toml:"code_family"`
	Size          string `toml:"size"`
	LineHeight    string `toml:"line_height"`
}

func Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /__admin/", handleAdminRoot)
	mux.HandleFunc("GET /__admin/dashboard", withAuth(handleDashboard))
	mux.HandleFunc("GET /__admin/editor", withAuth(handleEditor))
	mux.HandleFunc("GET /__admin/settings", withAuth(handleSettings))

	mux.HandleFunc("GET /__admin/login", handleLoginPage)
	mux.HandleFunc("POST /__admin/login", handleLogin)
	mux.HandleFunc("POST /__admin/logout", handleLogout)
	mux.HandleFunc("GET /__admin/logout", handleLogout)
	mux.HandleFunc("POST /__admin/setup", handleSetup)
	mux.HandleFunc("GET /__admin/setup", handleSetupPage)

	mux.HandleFunc("GET /__admin/api/posts", withAuth(handleListPosts))
	mux.HandleFunc("GET /__admin/api/post", withAuth(handleGetPost))
	mux.HandleFunc("POST /__admin/api/new/scaffold", withAuth(handleNewPostScaffold))
	mux.HandleFunc("POST /__admin/api/frontmatter/parse", withAuth(handleParseFrontmatter))
	mux.HandleFunc("POST /__admin/api/frontmatter/render", withAuth(handleRenderFrontmatter))
	mux.HandleFunc("POST /__admin/api/preview", withAuth(handlePreviewPost))
	mux.HandleFunc("POST /__admin/api/post", withAuth(handleCreatePost))
	mux.HandleFunc("PUT /__admin/api/post", withAuth(handleSavePost))
	mux.HandleFunc("DELETE /__admin/api/post", withAuth(handleDeletePost))
	mux.HandleFunc("GET /__admin/api/settings", withAuth(handleGetSettings))
	mux.HandleFunc("POST /__admin/api/settings/parse", withAuth(handleParseSettings))
	mux.HandleFunc("POST /__admin/api/settings/render", withAuth(handleRenderSettings))
	mux.HandleFunc("PUT /__admin/api/settings", withAuth(handleSaveSettings))
	mux.HandleFunc("POST /__admin/api/build-trigger", withAuth(handleBuildTrigger))
	mux.HandleFunc("GET /__admin/api/build-status", withAuth(handleBuildStatus))

	mux.HandleFunc("GET /__admin/api/git/status", withAuth(handleGitStatus))
	mux.HandleFunc("GET /__admin/api/git/diff", withAuth(handleGitDiff))
	mux.HandleFunc("POST /__admin/api/git/stage", withAuth(handleGitStage))
	mux.HandleFunc("POST /__admin/api/git/commit", withAuth(handleGitCommit))
	mux.HandleFunc("POST /__admin/api/git/push", withAuth(handleGitPush))

	return mux
}

func handleAdminRoot(w http.ResponseWriter, r *http.Request) {
	if !HasSecrets() {
		http.Redirect(w, r, "/__admin/setup", http.StatusFound)
		return
	}
	if !isAuthenticated(r) {
		http.Redirect(w, r, "/__admin/login", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/__admin/dashboard", http.StatusFound)
}

func handleLoginPage(w http.ResponseWriter, _ *http.Request) {
	renderPage(w, "auth", PageData{Title: "Login", NeedsLogin: true})
}

func handleSetupPage(w http.ResponseWriter, r *http.Request) {
	if HasSecrets() {
		http.Redirect(w, r, "/__admin/dashboard", http.StatusFound)
		return
	}
	renderPage(w, "auth", PageData{Title: "Setup", NeedsSetup: true})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	if !validatePassword(username, password) {
		renderPage(w, "auth", PageData{Title: "Login", NeedsLogin: true, Error: "Invalid username or password"})
		return
	}
	setSession(w, username)
	http.Redirect(w, r, "/__admin/dashboard", http.StatusFound)
}

func handleSetup(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	if len(username) < 3 || len(password) < 8 {
		renderPage(w, "auth", PageData{Title: "Setup", NeedsSetup: true, Error: "Username must be 3+ chars and password must be 8+ chars"})
		return
	}
	if err := createUser(username, password); err != nil {
		renderPage(w, "auth", PageData{Title: "Setup", NeedsSetup: true, Error: err.Error()})
		return
	}
	setSession(w, username)
	http.Redirect(w, r, "/__admin/dashboard", http.StatusFound)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	clearSession(w)
	http.Redirect(w, r, "/__admin/login", http.StatusFound)
}

func handleDashboard(w http.ResponseWriter, _ *http.Request) {
	posts, err := listPostInfos()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	renderPage(w, "dashboard", PageData{Title: "Dashboard", IsDashboard: true, Posts: posts})
}

func handleEditor(w http.ResponseWriter, r *http.Request) {
	postData, err := loadPostEditData(r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	renderPage(w, "editor", PageData{Title: "Editor", IsEditor: true, Post: postData, KnownTags: collectKnownTags(), KnownPalettes: collectKnownPalettes(), KnownDirs: collectKnownDirs(), KnownAuthors: discoverAuthorOptions(), NewPostContext: buildNewPostContextJSON()})
}

func handleSettings(w http.ResponseWriter, _ *http.Request) {
	settings, err := loadSettingsEditData()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	renderPage(w, "settings", PageData{Title: "Settings", IsSettings: true, Settings: settings, KnownPalettes: collectKnownPalettes()})
}

func handleListPosts(w http.ResponseWriter, _ *http.Request) {
	posts, err := listPostInfos()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"posts": posts})
}

func handleGetPost(w http.ResponseWriter, r *http.Request) {
	postData, err := loadPostEditData(r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respondJSON(w, http.StatusOK, postData)
}

func handleCreatePost(w http.ResponseWriter, r *http.Request) {
	result, err := savePostFromRequest(r, false)
	if err != nil {
		writeSaveError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, result)
}

func handleNewPostScaffold(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Title     string            `json:"title"`
		Template  string            `json:"template"`
		Directory string            `json:"directory"`
		Tags      []string          `json:"tags"`
		Private   bool              `json:"private"`
		Authors   []string          `json:"authors"`
		Extra     map[string]string `json:"extra"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := generateAdminNewPostScaffold(payload.Title, payload.Template, payload.Directory, payload.Tags, payload.Private, payload.Authors, payload.Extra)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func handleParseFrontmatter(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Frontmatter string `json:"frontmatter"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	form, err := parseFrontmatterForm(payload.Frontmatter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respondJSON(w, http.StatusOK, form)
}

func handleRenderFrontmatter(w http.ResponseWriter, r *http.Request) {
	var form adminFrontmatterForm
	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	raw, err := renderFrontmatterForm(form)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"frontmatter": raw})
}

func handlePreviewPost(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Frontmatter string `json:"frontmatter"`
		Body        string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	html, err := renderLivePreview(payload.Frontmatter, payload.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(html)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handleParseSettings(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respondJSON(w, http.StatusOK, parseSettingsForm(payload.Content))
}

func handleRenderSettings(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Content string            `json:"content"`
		Form    adminSettingsForm `json:"form"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	updated, err := renderSettingsForm(payload.Content, payload.Form)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateSettingsContent(updated); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"content": updated})
}

func handleSavePost(w http.ResponseWriter, r *http.Request) {
	result, err := savePostFromRequest(r, true)
	if err != nil {
		writeSaveError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func handleDeletePost(w http.ResponseWriter, r *http.Request) {
	fullPath, _, err := resolveContentFilePath(r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := os.Remove(fullPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	triggerRebuildIfNeeded()
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

func handleGetSettings(w http.ResponseWriter, _ *http.Request) {
	settings, err := loadSettingsEditData()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respondJSON(w, http.StatusOK, settings)
}

func handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Content  string `json:"content"`
		BaseHash string `json:"base_hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	configPath := GetConfigPath()
	if strings.TrimSpace(configPath) == "" {
		http.Error(w, "config file is not available for editing", http.StatusBadRequest)
		return
	}
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		absPath = filepath.Clean(configPath)
	}
	if payload.BaseHash != "" {
		if existing, readErr := os.ReadFile(absPath); readErr == nil && contentedit.ContentHash(string(existing)) != payload.BaseHash {
			respondJSON(w, http.StatusConflict, map[string]any{"success": false, "error": "conflict", "message": "Config file was modified externally"})
			return
		}
	}
	if err := validateSettingsContent(payload.Content); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := atomicWriteFile(absPath, payload.Content, 0o644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	triggerRebuildIfNeeded()
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "new_hash": contentedit.ContentHash(payload.Content), "build_triggered": !IsWatchEnabled()})
}

func handleBuildTrigger(w http.ResponseWriter, _ *http.Request) {
	TriggerRebuild()
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

func handleBuildStatus(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, GetBuildStatus())
}

func handleGitStatus(w http.ResponseWriter, r *http.Request) {
	fullPath, _, err := resolveContentFilePath(r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// #nosec G204 -- fullPath is resolved inside the configured content directory.
	cmd := exec.Command("git", "-C", getRepoDir(fullPath), "status", "--porcelain", fullPath)
	out, err := cmd.Output()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"status": strings.TrimSpace(string(out))})
}

func handleGitDiff(w http.ResponseWriter, r *http.Request) {
	fullPath, _, err := resolveContentFilePath(r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// #nosec G204 -- fullPath is resolved inside the configured content directory.
	cmd := exec.Command("git", "-C", getRepoDir(fullPath), "diff", fullPath)
	out, err := cmd.Output()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"diff": string(out)})
}

func handleGitStage(w http.ResponseWriter, r *http.Request) {
	fullPath, _, err := resolveContentFilePath(r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// #nosec G204 -- fullPath is resolved inside the configured content directory.
	cmd := exec.Command("git", "-C", getRepoDir(fullPath), "add", fullPath)
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

func handleGitCommit(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Path    string `json:"path"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	repoDir := "."
	if strings.TrimSpace(payload.Path) != "" {
		if fullPath, _, err := resolveContentFilePath(payload.Path); err == nil {
			repoDir = getRepoDir(fullPath)
		}
	}
	// #nosec G204 -- repoDir is derived from the configured content directory.
	cmd := exec.Command("git", "-C", repoDir, "commit", "-m", payload.Message)
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

func handleGitPush(w http.ResponseWriter, _ *http.Request) {
	cmd := exec.Command("git", "push")
	if err := cmd.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"success": true})
}

func validatePassword(username, password string) bool {
	secrets, err := LoadSecrets(GetSecretsDir())
	if err != nil {
		return false
	}
	if username != secrets.AdminUsername {
		return false
	}
	return CheckPassword(password, secrets.AdminPassword)
}

func createUser(username, password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	sessionKey, err := generateCSRF()
	if err != nil {
		return err
	}
	return CreateSecrets(GetSecretsDir(), username, hash, sessionKey)
}

func isAuthenticated(r *http.Request) bool {
	_, err := getSession(r)
	return err == nil
}

func renderPage(w http.ResponseWriter, name string, data PageData) {
	data.ThemeCSS = buildThemeCSS(GetSiteConfig())
	if err := adminTemplates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func buildNewPostContextJSON() template.JS {
	payload := map[string]interface{}{
		"templates":   discoverAdminTemplates(),
		"aliases":     discoverAdminTemplateAliases(),
		"directories": collectKnownDirs(),
		"tags":        collectKnownTags(),
		"authors":     discoverAuthorOptions(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		// #nosec G203 -- fallback JavaScript is a static JSON literal.
		return template.JS(`{"templates":[],"aliases":{},"directories":[],"tags":[],"authors":[]}`)
	}
	// #nosec G203 -- payload is JSON encoded by encoding/json.
	return template.JS(data)
}

func listPostInfos() ([]PostInfo, error) {
	sitePosts := GetSitePosts()
	if len(sitePosts) == 0 {
		posts, err := loadEditorPostsFromBuildGlob()
		if err != nil {
			return nil, err
		}
		infos := make([]PostInfo, 0, len(posts))
		for _, post := range posts {
			modified := ""
			if info, err := os.Stat(post.Path); err == nil {
				modified = info.ModTime().UTC().Format(time.RFC3339)
			}
			infos = append(infos, PostInfo{Path: toDisplayPath(post.Path), Title: valueOr(post.GetTitle(), filepath.Base(post.Path)), Slug: post.Slug, Date: post.GetDate(), Type: inferPostType(post.Frontmatter), Published: post.IsPublished(), Modified: modified})
		}
		return infos, nil
	}

	infos := make([]PostInfo, 0, len(sitePosts))
	for _, post := range sitePosts {
		if post == nil {
			continue
		}
		modified := ""
		if info, err := os.Stat(post.Path); err == nil {
			modified = info.ModTime().UTC().Format(time.RFC3339)
		}
		title := filepath.Base(post.Path)
		if post.Title != nil && strings.TrimSpace(*post.Title) != "" {
			title = *post.Title
		}
		date := ""
		if post.Date != nil {
			date = post.Date.UTC().Format(time.RFC3339)
		}
		infos = append(infos, PostInfo{Path: toDisplayPath(post.Path), Title: title, Slug: post.Slug, Date: date, Type: inferPostTypeFromModel(post), Published: post.Published && !post.Draft, Modified: modified})
	}
	return infos, nil
}

func loadEditorPostsFromBuildGlob() ([]*contentedit.Post, error) {
	cfg := GetSiteConfig()
	patterns := []string{"**/*.md"}
	useGitignore := true
	if cfg != nil {
		if len(cfg.GlobConfig.Patterns) > 0 {
			patterns = append([]string(nil), cfg.GlobConfig.Patterns...)
		}
		useGitignore = cfg.GlobConfig.UseGitignore
	}
	files, err := plugins.DiscoverFiles(GetContentDir(), patterns, useGitignore)
	if err != nil {
		return nil, err
	}
	posts := make([]*contentedit.Post, 0, len(files))
	for _, relPath := range files {
		post, loadErr := contentedit.LoadPost(ResolveContentPath(relPath))
		if loadErr != nil {
			continue
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func inferPostType(frontmatter string) string {
	var data map[string]any
	if err := yaml.Unmarshal([]byte(frontmatter), &data); err != nil {
		return ""
	}
	if value, ok := data["templateKey"].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if value, ok := data["template"].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if value, ok := data["layout"].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return ""
}

func inferPostTypeFromModel(post *models.Post) string {
	if post == nil {
		return ""
	}
	if value, ok := post.Get("templateKey").(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if strings.TrimSpace(post.Template) != "" {
		return strings.TrimSpace(post.Template)
	}
	return ""
}

func collectKnownTags() []string {
	if sitePosts := GetSitePosts(); len(sitePosts) > 0 {
		seen := make(map[string]struct{})
		for _, post := range sitePosts {
			if post == nil {
				continue
			}
			for _, tag := range post.Tags {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					seen[tag] = struct{}{}
				}
			}
		}
		tags := make([]string, 0, len(seen))
		for tag := range seen {
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		return tags
	}

	posts, err := loadEditorPostsFromBuildGlob()
	if err != nil {
		return nil
	}
	seen := make(map[string]struct{})
	for _, post := range posts {
		var data map[string]any
		if strings.TrimSpace(post.Frontmatter) == "" {
			continue
		}
		if err := yaml.Unmarshal([]byte(post.Frontmatter), &data); err != nil {
			continue
		}
		for _, tag := range interfaceSliceToStrings(data["tags"]) {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				seen[tag] = struct{}{}
			}
		}
	}
	tags := make([]string, 0, len(seen))
	for tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

func collectKnownPalettes() []string {
	paletteDir := discoverPalettesDir()
	entries, err := os.ReadDir(paletteDir)
	if err != nil {
		return []string{"default-light", "default-dark"}
	}
	palettes := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".toml" {
			continue
		}
		palettes = append(palettes, strings.TrimSuffix(name, filepath.Ext(name)))
	}
	sort.Strings(palettes)
	return palettes
}

func collectKnownDirs() []string {
	seen := make(map[string]struct{})
	if sitePosts := GetSitePosts(); len(sitePosts) > 0 {
		for _, post := range sitePosts {
			if post == nil {
				continue
			}
			dir := filepath.ToSlash(filepath.Dir(toDisplayPath(post.Path)))
			if dir != "." && dir != "" {
				seen[dir] = struct{}{}
			}
		}
	}
	for _, tmpl := range discoverAdminTemplates() {
		if strings.TrimSpace(tmpl.Directory) != "" {
			seen[filepath.ToSlash(tmpl.Directory)] = struct{}{}
		}
	}
	dirs := make([]string, 0, len(seen))
	for dir := range seen {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	return dirs
}

func discoverAuthorOptions() []adminAuthorOption {
	cfg := GetSiteConfig()
	if cfg == nil || len(cfg.Authors.Authors) == 0 {
		return nil
	}
	options := make([]adminAuthorOption, 0, len(cfg.Authors.Authors))
	for id := range cfg.Authors.Authors {
		author := cfg.Authors.Authors[id]
		if !author.Active && !author.Default {
			continue
		}
		name := author.Name
		if strings.TrimSpace(name) == "" {
			name = id
		}
		options = append(options, adminAuthorOption{ID: id, Name: name, Default: author.Default})
	}
	sort.Slice(options, func(i, j int) bool { return options[i].Name < options[j].Name })
	return options
}

func discoverAdminTemplates() map[string]adminTemplateDefinition {
	templates := map[string]adminTemplateDefinition{
		"post":    {Name: "post", Label: "post", Directory: "pages/post", Frontmatter: map[string]interface{}{"template": "post"}, Body: "Write your content here...", Source: "builtin"},
		"page":    {Name: "page", Label: "page", Directory: "pages", Frontmatter: map[string]interface{}{"template": "page"}, Body: "Write your page content here...", Source: "builtin"},
		"docs":    {Name: "docs", Label: "docs", Directory: "docs", Frontmatter: map[string]interface{}{"template": "docs"}, Body: "Write your documentation here...", Source: "builtin"},
		"article": {Name: "article", Label: "article (aka: blog-post, essay, tutorial)", Directory: "pages/article", Frontmatter: map[string]interface{}{"template": "article"}, Body: "Write your article here...", Source: "builtin"},
		"note":    {Name: "note", Label: "note (aka: ping, thought, status, tweet)", Directory: "pages/note", Frontmatter: map[string]interface{}{"template": "note"}, Body: "A quick note...", Source: "builtin"},
		"photo":   {Name: "photo", Label: "photo (aka: shot, shots, image, gallery)", Directory: "pages/photo", Frontmatter: map[string]interface{}{"template": "photo", "image": ""}, Body: "Photo caption...", Source: "builtin"},
		"video":   {Name: "video", Label: "video (aka: clip, cast, stream)", Directory: "pages/video", Frontmatter: map[string]interface{}{"template": "video", "video": "", "image": "", "duration": ""}, Body: "Video description...", Source: "builtin"},
		"link":    {Name: "link", Label: "link (aka: bookmark, til, stars)", Directory: "pages/link", Frontmatter: map[string]interface{}{"template": "link", "url": "", "image": ""}, Body: "Why I'm sharing this link...", Source: "builtin"},
		"quote":   {Name: "quote", Label: "quote (aka: quotation)", Directory: "pages/quote", Frontmatter: map[string]interface{}{"template": "quote", "quote": "", "source": ""}, Body: "Additional commentary on this quote...", Source: "builtin"},
		"guide":   {Name: "guide", Label: "guide (aka: series, step, chapter)", Directory: "pages/guide", Frontmatter: map[string]interface{}{"template": "guide"}, Body: "## Introduction\n\nWrite your guide here...", Source: "builtin"},
		"inline":  {Name: "inline", Label: "inline (aka: gratitude, micro)", Directory: "pages/inline", Frontmatter: map[string]interface{}{"template": "inline"}, Body: "Inline content...", Source: "builtin"},
		"contact": {Name: "contact", Label: "contact (aka: character, person)", Directory: "pages/contact", Frontmatter: map[string]interface{}{"template": "contact", "handle": "", "url": ""}, Body: "Bio or contact details...", Source: "builtin"},
		"author":  {Name: "author", Label: "author", Directory: "pages/author", Frontmatter: map[string]interface{}{"template": "author", "name": "", "bio": "", "role": "", "avatar": "", "url": "", "email": ""}, Body: "Extended author bio...", Source: "builtin"},
	}
	applyAdminTemplateOverrides(templates)
	return templates
}

func discoverAdminTemplateAliases() map[string][]string {
	return map[string][]string{
		"post":    {"blog-post", "essay", "tutorial"},
		"note":    {"ping", "thought", "status", "tweet"},
		"photo":   {"shot", "shots", "image", "gallery"},
		"video":   {"clip", "cast", "stream"},
		"link":    {"bookmark", "til", "stars"},
		"quote":   {"quotation"},
		"guide":   {"series", "step", "chapter"},
		"inline":  {"gratitude", "micro"},
		"contact": {"character", "person"},
	}
}

func isKnownAdminTemplateType(templateKey string) bool {
	trimmed := strings.TrimSpace(templateKey)
	if trimmed == "" {
		return true
	}
	if _, ok := discoverAdminTemplates()[trimmed]; ok {
		return true
	}
	for _, aliases := range discoverAdminTemplateAliases() {
		for _, alias := range aliases {
			if trimmed == alias {
				return true
			}
		}
	}
	return false
}

func applyAdminTemplateOverrides(templates map[string]adminTemplateDefinition) {
	configPath := GetConfigPath()
	if strings.TrimSpace(configPath) == "" {
		loadAdminTemplatesFromDir("content-templates", templates)
		return
	}
	configDir := filepath.Dir(configPath)
	if cfg, err := loadAdminTemplateConfig(configPath); err == nil && cfg != nil {
		for name, dir := range cfg.ContentTemplates.Placement {
			if tmpl, ok := templates[name]; ok {
				tmpl.Directory = dir
				templates[name] = tmpl
			}
		}
		for _, tmpl := range cfg.ContentTemplates.Templates {
			templates[tmpl.Name] = adminTemplateDefinition{Name: tmpl.Name, Label: tmpl.Name, Directory: tmpl.Directory, Frontmatter: tmpl.Frontmatter, Body: tmpl.Body, Source: "config"}
		}
		templatesDir := cfg.ContentTemplates.Directory
		if templatesDir == "" {
			templatesDir = "content-templates"
		}
		loadAdminTemplatesFromDir(filepath.Join(configDir, templatesDir), templates)
		return
	}
	loadAdminTemplatesFromDir(filepath.Join(configDir, "content-templates"), templates)
}

type adminTemplateConfigWrapper struct {
	ContentTemplates struct {
		Directory string            `yaml:"directory" toml:"directory" json:"directory"`
		Placement map[string]string `yaml:"placement" toml:"placement" json:"placement"`
		Templates []struct {
			Name        string                 `yaml:"name" toml:"name" json:"name"`
			Directory   string                 `yaml:"directory" toml:"directory" json:"directory"`
			Frontmatter map[string]interface{} `yaml:"frontmatter" toml:"frontmatter" json:"frontmatter"`
			Body        string                 `yaml:"body" toml:"body" json:"body"`
		} `yaml:"templates" toml:"templates" json:"templates"`
	} `yaml:"content_templates" toml:"content_templates" json:"content_templates"`
}

func loadAdminTemplateConfig(path string) (*adminTemplateConfigWrapper, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg adminTemplateConfigWrapper
	switch filepath.Ext(path) {
	case ".toml":
		if err := toml.Unmarshal(content, &cfg); err != nil {
			return nil, err
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(content, &cfg); err != nil {
			return nil, err
		}
	case ".json":
		if err := json.Unmarshal(content, &cfg); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported config format")
	}
	return &cfg, nil
}

func loadAdminTemplatesFromDir(dir string, templates map[string]adminTemplateDefinition) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".md")
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		templates[name] = parseAdminTemplateFile(name, string(content))
	}
}

func parseAdminTemplateFile(name, content string) adminTemplateDefinition {
	tmpl := adminTemplateDefinition{Name: name, Label: name, Directory: name, Frontmatter: make(map[string]interface{}), Body: "", Source: "file"}
	if !strings.HasPrefix(content, "---") {
		tmpl.Body = strings.TrimSpace(content)
		return tmpl
	}
	parts := strings.SplitN(content[3:], "---", 2)
	if len(parts) < 2 {
		tmpl.Body = strings.TrimSpace(content)
		return tmpl
	}
	frontmatterYAML := strings.TrimSpace(parts[0])
	if err := yaml.Unmarshal([]byte(frontmatterYAML), &tmpl.Frontmatter); err == nil {
		if dir, ok := tmpl.Frontmatter["_directory"].(string); ok {
			tmpl.Directory = dir
			delete(tmpl.Frontmatter, "_directory")
		}
	}
	tmpl.Body = strings.TrimSpace(parts[1])
	return tmpl
}

func generateAdminNewPostScaffold(title, templateName, directory string, tags []string, private bool, authors []string, extra map[string]string) (map[string]interface{}, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	templates := discoverAdminTemplates()
	tmpl, ok := templates[templateName]
	if !ok {
		tmpl = templates["post"]
		templateName = "post"
	}
	if strings.TrimSpace(directory) == "" {
		directory = tmpl.Directory
	}
	slug := adminSlugify(title)
	frontmatter := make(map[string]interface{}, len(tmpl.Frontmatter)+8)
	for k, v := range tmpl.Frontmatter {
		frontmatter[k] = v
	}
	frontmatter["title"] = title
	frontmatter["slug"] = slug
	frontmatter["date"] = time.Now().UTC().Format(time.RFC3339)
	frontmatter["published"] = true
	frontmatter["draft"] = false
	frontmatter["private"] = private
	frontmatter["description"] = ""
	frontmatter["templateKey"] = templateName
	if len(authors) > 0 {
		frontmatter["authors"] = authors
	}
	if len(tags) > 0 {
		frontmatter["tags"] = tags
	} else {
		frontmatter["tags"] = []string{}
	}
	for key, value := range extra {
		if strings.TrimSpace(key) != "" {
			frontmatter[key] = value
		}
	}
	raw, err := yaml.Marshal(frontmatter)
	if err != nil {
		return nil, err
	}
	formatted, err := contentedit.FormatFrontmatter(string(raw))
	if err != nil {
		return nil, err
	}
	path := filepath.ToSlash(filepath.Join(directory, slug+".md"))
	return map[string]interface{}{
		"path":        path,
		"frontmatter": formatted,
		"body":        tmpl.Body,
		"template":    templateName,
	}, nil
}

func discoverPalettesDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return filepath.Join(".", "palettes")
	}
	for {
		candidate := filepath.Join(dir, "palettes")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate
		}
		next := filepath.Dir(dir)
		if next == dir {
			return filepath.Join(".", "palettes")
		}
		dir = next
	}
}

func loadPostEditData(path string) (PostEditData, error) {
	if strings.TrimSpace(path) == "" {
		post := contentedit.NewPost(ResolveContentPath(defaultNewPostPath()), defaultFrontmatter(), "")
		return postToEditData(post), nil
	}
	fullPath, _, err := resolveContentFilePath(path)
	if err != nil {
		return PostEditData{}, err
	}
	post, err := contentedit.LoadPost(fullPath)
	if err != nil {
		return PostEditData{}, err
	}
	return postToEditData(post), nil
}

func loadSettingsEditData() (SettingsEditData, error) {
	path := GetConfigPath()
	if strings.TrimSpace(path) == "" {
		return SettingsEditData{}, fmt.Errorf("config file is not available for editing")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return SettingsEditData{}, err
	}
	content := string(data)
	return SettingsEditData{Path: path, Content: content, Hash: contentedit.ContentHash(content), Exists: true}, nil
}

func savePostFromRequest(r *http.Request, requireExisting bool) (map[string]any, error) {
	var payload savePostRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, err
	}
	fullPath, displayPath, err := preparePostPath(payload.Path, payload.Frontmatter)
	if err != nil {
		return nil, err
	}
	_, statErr := os.Stat(fullPath)
	exists := statErr == nil
	if requireExisting && !exists {
		return nil, contentedit.ErrFileNotFound
	}
	if !requireExisting && exists {
		return nil, fmt.Errorf("post already exists: %s", displayPath)
	}
	if exists {
		if form, parseErr := parseFrontmatterForm(payload.Frontmatter); parseErr == nil {
			form.Modified = time.Now().UTC().Format(time.RFC3339)
			if rendered, renderErr := renderFrontmatterForm(form); renderErr == nil {
				payload.Frontmatter = rendered
			}
		}
	}
	post := contentedit.NewPost(fullPath, payload.Frontmatter, payload.Body)
	if exists {
		post.Exists = true
	}
	if err := contentedit.SavePost(post, &contentedit.SaveOptions{BaseHash: payload.BaseHash}); err != nil {
		return nil, err
	}
	triggerRebuildIfNeeded()
	return map[string]any{"success": true, "path": toDisplayPath(post.Path), "preview_url": post.PreviewURL, "new_hash": post.Hash, "build_triggered": !IsWatchEnabled()}, nil
}

func writeSaveError(w http.ResponseWriter, err error) {
	if errors.Is(err, contentedit.ErrConflict) {
		respondJSON(w, http.StatusConflict, map[string]any{"success": false, "error": "conflict", "message": "File was modified externally"})
		return
	}
	if errors.Is(err, contentedit.ErrFileNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if errors.Is(err, contentedit.ErrInvalidFrontmatter) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.Contains(err.Error(), "already exists") {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func triggerRebuildIfNeeded() {
	if !IsWatchEnabled() {
		TriggerRebuild()
	}
}

func preparePostPath(path, frontmatter string) (fullPath, displayPath string, err error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = defaultPathFromFrontmatter(frontmatter)
	}
	return resolveContentFilePath(trimmed)
}

func resolveContentFilePath(input string) (fullPath, displayPath string, err error) {
	cleanInput := filepath.Clean(filepath.FromSlash(strings.TrimSpace(input)))
	if cleanInput == "." || cleanInput == "" {
		return "", "", fmt.Errorf("path is required")
	}
	if strings.HasPrefix(cleanInput, "..") || filepath.IsAbs(cleanInput) {
		return "", "", fmt.Errorf("invalid path")
	}
	fullPath = ResolveContentPath(cleanInput)
	contentRoot := ResolveContentPath(".")
	if !isWithinDir(fullPath, contentRoot) {
		return "", "", fmt.Errorf("path must stay within %s", contentRoot)
	}
	return fullPath, filepath.ToSlash(cleanInput), nil
}

func toDisplayPath(path string) string {
	contentRoot := ResolveContentPath(".")
	rel, err := filepath.Rel(contentRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func isWithinDir(path, dir string) bool {
	if filepath.Clean(path) == filepath.Clean(dir) {
		return true
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func postToEditData(post *contentedit.Post) PostEditData {
	return PostEditData{Path: toDisplayPath(post.Path), Title: post.GetTitle(), Frontmatter: post.Frontmatter, Body: post.Body, PreviewURL: post.PreviewURL, Slug: post.Slug, GitStatus: gitStatus(post.Path), Hash: post.Hash, Exists: post.Exists}
}

func defaultNewPostPath() string {
	contentRoot := ResolveContentPath(".")
	for _, candidate := range []string{"pages", "content", "docs"} {
		if info, err := os.Stat(filepath.Join(contentRoot, candidate)); err == nil && info.IsDir() {
			return filepath.ToSlash(filepath.Join(candidate, "new-post.md"))
		}
	}
	return "new-post.md"
}

func defaultPathFromFrontmatter(frontmatter string) string {
	post := contentedit.NewPost(defaultNewPostPath(), frontmatter, "")
	slug := strings.TrimSpace(post.Slug)
	if slug == "" {
		slug = "untitled"
	}
	dir := filepath.Dir(defaultNewPostPath())
	if dir == "." {
		return slug + ".md"
	}
	return filepath.ToSlash(filepath.Join(dir, slug+".md"))
}

func defaultFrontmatter() string {
	return strings.TrimSpace(fmt.Sprintf("title: Untitled\ndate: %s\npublished: false", time.Now().Format("2006-01-02")))
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return
	}
}

func atomicWriteFile(path, content string, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(dir, ".tmp-config-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func gitStatus(path string) string {
	// #nosec G204 -- path is resolved inside the configured content directory.
	cmd := exec.Command("git", "-C", getRepoDir(path), "status", "--porcelain", path)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	status := strings.TrimSpace(string(out))
	if status == "" {
		return "tracked"
	}
	if strings.HasPrefix(status, "??") {
		return "untracked"
	}
	if status != "" && status[0] != ' ' {
		return "staged"
	}
	return "modified"
}

func getRepoDir(path string) string {
	probe := path
	if info, err := os.Stat(probe); err == nil && !info.IsDir() {
		probe = filepath.Dir(probe)
	}
	for {
		cmd := exec.Command("git", "-C", probe, "rev-parse", "--show-toplevel")
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
		next := filepath.Dir(probe)
		if next == probe {
			return "."
		}
		probe = next
	}
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func adminSlugify(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	input = strings.ReplaceAll(input, "_", "-")
	input = strings.ReplaceAll(input, " ", "-")
	for strings.Contains(input, "--") {
		input = strings.ReplaceAll(input, "--", "-")
	}
	return strings.Trim(input, "-")
}

func buildThemeCSS(cfg *models.Config) template.CSS {
	lightVars := defaultThemeVars(false)
	darkVars := defaultThemeVars(true)
	typographyVars := adminTypographyVars(cfg)
	if cfg == nil {
		// #nosec G203 -- CSS is generated from internal palette variables.
		return template.CSS(renderThemeCSS(lightVars, darkVars, typographyVars, adminThemeLight))
	}
	loader := palettes.NewLoader()
	lightName, darkName, fallbackMode := resolvePaletteNames(cfg)
	if palette, err := loader.Load(lightName); err == nil {
		lightVars = themeVarsFromPalette(palette, false)
	}
	if palette, err := loader.Load(darkName); err == nil {
		darkVars = themeVarsFromPalette(palette, true)
	}
	defaultVars := lightVars
	otherVars := darkVars
	if fallbackMode == adminThemeDark {
		defaultVars = darkVars
		otherVars = lightVars
	}
	// #nosec G203 -- CSS is generated from internal palette variables.
	return template.CSS(renderThemeCSS(defaultVars, otherVars, typographyVars, fallbackMode))
}

func adminTypographyVars(cfg *models.Config) map[string]string {
	font := models.NewFontConfig()
	if cfg != nil {
		if cfg.Theme.Font.Family != "" {
			font.Family = cfg.Theme.Font.Family
		}
		if cfg.Theme.Font.HeadingFamily != "" {
			font.HeadingFamily = cfg.Theme.Font.HeadingFamily
		}
		if cfg.Theme.Font.CodeFamily != "" {
			font.CodeFamily = cfg.Theme.Font.CodeFamily
		}
		if cfg.Theme.Font.Size != "" {
			font.Size = cfg.Theme.Font.Size
		}
		if cfg.Theme.Font.LineHeight != "" {
			font.LineHeight = cfg.Theme.Font.LineHeight
		}
	}

	return map[string]string{
		"font-body":    font.Family,
		"font-heading": font.GetHeadingFamily(),
		"font-code":    font.CodeFamily,
		"font-size":    font.Size,
		"line-height":  font.LineHeight,
	}
}

func resolvePaletteNames(cfg *models.Config) (lightName, darkName, fallbackMode string) {
	lightName = "default-light"
	darkName = "default-dark"
	fallbackMode = adminThemeLight
	if cfg == nil {
		return
	}
	if cfg.Theme.FallbackMode != "" {
		fallbackMode = cfg.Theme.FallbackMode
	}
	if cfg.Theme.PaletteLight != "" {
		lightName = cfg.Theme.PaletteLight
	}
	if cfg.Theme.PaletteDark != "" {
		darkName = cfg.Theme.PaletteDark
	}
	if cfg.Theme.Palette != "" {
		name := cfg.Theme.Palette
		if strings.Contains(name, "light") || strings.Contains(name, "latte") || strings.Contains(name, "dawn") || strings.Contains(name, "day") {
			lightName = name
		} else {
			darkName = name
		}
	}
	return
}

func defaultThemeVars(dark bool) map[string]string {
	if dark {
		return map[string]string{"bg": "#111827", "surface": "#1f2937", "surfaceAlt": "#374151", "text": "#f9fafb", "muted": "#9ca3af", "border": "#374151", "accent": "#60a5fa", "accentHover": "#93c5fd", "accentContrast": "#0f172a", "success": "#34d399", "warning": "#fbbf24", "error": "#f87171", "shadow": "rgba(15, 23, 42, 0.35)"}
	}
	return map[string]string{"bg": "#f9fafb", "surface": "#ffffff", "surfaceAlt": "#f3f4f6", "text": "#111827", "muted": "#6b7280", "border": "#d1d5db", "accent": "#2563eb", "accentHover": "#1d4ed8", "accentContrast": "#ffffff", "success": "#047857", "warning": "#b45309", "error": "#dc2626", "shadow": "rgba(15, 23, 42, 0.08)"}
}

func themeVarsFromPalette(p *palettes.Palette, dark bool) map[string]string {
	vars := defaultThemeVars(dark)
	setIf := func(key, colorName string) {
		if hex := p.Resolve(colorName); hex != "" {
			vars[key] = hex
		}
	}
	setIf("bg", "bg-primary")
	setIf("surface", "card-bg")
	setIf("surfaceAlt", "bg-secondary")
	setIf("text", "text-primary")
	setIf("muted", "text-muted")
	setIf("border", "border")
	setIf("accent", "accent")
	setIf("accentHover", "accent-hover")
	setIf("success", "success")
	setIf("warning", "warning")
	setIf("error", "error")
	if hex := p.Resolve("button-primary-text"); hex != "" {
		vars["accentContrast"] = hex
	}
	return vars
}

func renderThemeCSS(primaryVars, alternateVars, typographyVars map[string]string, fallbackMode string) string {
	primaryMode := adminThemeLight
	alternateMode := adminThemeDark
	mediaMode := adminThemeDark
	if fallbackMode == adminThemeDark {
		primaryMode = adminThemeDark
		alternateMode = adminThemeLight
		mediaMode = adminThemeLight
	}
	return fmt.Sprintf(`:root { %s %s --admin-fallback-mode: %s; }
@media (prefers-color-scheme: %s) { :root:not([data-theme]) { %s } }
:root[data-theme="%s"] { %s }
:root[data-theme="%s"] { %s }`, cssVarBlock(primaryVars), typographyVarBlock(typographyVars), primaryMode, mediaMode, cssVarBlock(alternateVars), primaryMode, cssVarBlock(primaryVars), alternateMode, cssVarBlock(alternateVars))
}

func cssVarBlock(vars map[string]string) string {
	keys := []string{"bg", "surface", "surfaceAlt", "text", "muted", "border", "accent", "accentHover", "accentContrast", "success", "warning", "error", "shadow"}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("--admin-%s: %s;", key, vars[key]))
	}
	return strings.Join(parts, " ")
}

func typographyVarBlock(vars map[string]string) string {
	keys := []string{"font-body", "font-heading", "font-code", "font-size", "line-height"}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		if vars[key] == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("--admin-%s: %s;", key, vars[key]))
	}
	return strings.Join(parts, " ")
}

func renderLivePreview(frontmatter, body string) (string, error) {
	var data map[string]any
	if strings.TrimSpace(frontmatter) != "" {
		if err := yaml.Unmarshal([]byte(frontmatter), &data); err != nil {
			return "", fmt.Errorf("frontmatter validation failed: %w", err)
		}
	}
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
	)
	var rendered bytes.Buffer
	if err := md.Convert([]byte(body), &rendered); err != nil {
		return "", err
	}
	title := "Untitled"
	if rawTitle, ok := data["title"].(string); ok && strings.TrimSpace(rawTitle) != "" {
		title = strings.TrimSpace(rawTitle)
	}
	description := ""
	if rawDescription, ok := data["description"].(string); ok {
		description = strings.TrimSpace(rawDescription)
	}
	themeCSS := string(buildThemeCSS(GetSiteConfig()))
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
  <style>
    %s
    :root { color-scheme: light dark; }
    html, body { margin: 0; min-height: 100%%; background: radial-gradient(circle at top, color-mix(in srgb, var(--admin-accent) 12%%, transparent) 0%%, transparent 36%%), linear-gradient(180deg, color-mix(in srgb, var(--admin-bg) 90%%, var(--admin-surfaceAlt)) 0%%, var(--admin-bg) 100%%); color: var(--admin-text); }
    body { padding: clamp(1rem, 3vw, 2rem); font: var(--admin-font-size, 16px)/calc(var(--admin-line-height, 1.6) + 0.15) var(--admin-font-body, "Iowan Old Style", "Palatino Linotype", serif); }
    main { max-width: 78ch; margin: 0 auto; background: color-mix(in srgb, var(--admin-surface) 92%%, transparent); border: 1px solid color-mix(in srgb, var(--admin-border) 70%%, transparent); border-radius: 24px; padding: clamp(1.1rem, 2vw, 2rem); box-shadow: 0 28px 60px -44px var(--admin-shadow); }
    h1, h2, h3 { line-height: 1.15; font-family: var(--admin-font-heading, "Avenir Next Condensed", "Franklin Gothic Medium", "Arial Narrow", sans-serif); }
    h1 { font-size: clamp(2rem, 4vw, 3rem); margin: 0 0 0.35rem; }
    h2 { margin-top: 2.2rem; font-size: clamp(1.4rem, 2vw, 1.9rem); }
    h3 { margin-top: 1.7rem; font-size: 1.15rem; }
    a { color: var(--admin-accent, #2563eb); }
    p, li { max-width: 72ch; }
    pre, code { font-family: var(--admin-font-code, "SFMono-Regular", "JetBrains Mono", monospace); }
    pre { overflow-x: auto; padding: 1rem; border-radius: 14px; background: color-mix(in srgb, var(--admin-surfaceAlt, #f3f4f6) 85%%, white); border: 1px solid color-mix(in srgb, var(--admin-border) 70%%, transparent); }
    blockquote { border-left: 4px solid var(--admin-accent); margin: 1.5rem 0; padding: 0.2rem 0 0.2rem 1rem; color: var(--admin-muted, #6b7280); }
    img { max-width: 100%%; height: auto; border-radius: 16px; }
    header { margin-bottom: 2rem; }
    hr { border: 0; border-top: 1px solid color-mix(in srgb, var(--admin-border) 72%%, transparent); margin: 2rem 0; }
    table { width: 100%%; border-collapse: collapse; margin: 1.5rem 0; }
    th, td { padding: 0.7rem 0.85rem; border-bottom: 1px solid color-mix(in srgb, var(--admin-border) 72%%, transparent); text-align: left; }
    .description { color: var(--admin-muted, #6b7280); }
  </style>
</head>
<body>
  <main>
    <header>
      <h1>%s</h1>
      %s
    </header>
    %s
  </main>
</body>
</html>`, template.HTMLEscapeString(title), themeCSS, template.HTMLEscapeString(title), renderDescription(description), rendered.String()), nil
}

func renderDescription(description string) string {
	if description == "" {
		return ""
	}
	return fmt.Sprintf(`<p class="description">%s</p>`, template.HTMLEscapeString(description))
}

func parseFrontmatterForm(frontmatter string) (adminFrontmatterForm, error) {
	result := adminFrontmatterForm{Tags: []string{}, Authors: []string{}, Extras: []adminKeyValueField{}}
	if strings.TrimSpace(frontmatter) == "" {
		return result, nil
	}
	var data map[string]any
	if err := yaml.Unmarshal([]byte(frontmatter), &data); err != nil {
		return result, fmt.Errorf("frontmatter validation failed: %w", err)
	}
	if value, ok := data["title"].(string); ok {
		result.Title = value
	}
	if value, ok := data["slug"].(string); ok {
		result.Slug = value
	}
	result.Date = stringifyFrontmatterScalar(data["date"])
	result.Modified = stringifyFrontmatterScalar(data["modified"])
	if value, ok := data["description"].(string); ok {
		result.Description = value
	}
	if value, ok := data["published"].(bool); ok {
		result.Published = value
	}
	if value, ok := data["templateKey"].(string); ok {
		result.TemplateKey = value
	} else if value, ok := data["template"].(string); ok {
		result.TemplateKey = value
	} else if value, ok := data["layout"].(string); ok {
		result.TemplateKey = value
	}
	if value, ok := data["author"].(string); ok {
		result.Author = strings.TrimSpace(value)
		if result.Author != "" && len(result.Authors) == 0 {
			result.Authors = []string{result.Author}
		}
	}
	if rawAuthors, ok := data["authors"]; ok {
		result.Authors = interfaceSliceToStrings(rawAuthors)
	}
	if rawTags, ok := data["tags"]; ok {
		result.Tags = interfaceSliceToStrings(rawTags)
	}
	known := map[string]bool{"title": true, "slug": true, "date": true, "modified": true, "description": true, "published": true, "layout": true, "template": true, "templateKey": true, "author": true, "authors": true, "tags": true}
	for key, value := range data {
		if known[key] {
			continue
		}
		result.Extras = append(result.Extras, adminKeyValueField{Key: key, Kind: detectValueKind(value), Value: interfaceToScalarString(value)})
	}
	return result, nil
}

//nolint:gocyclo // Frontmatter rendering maps the editable form fields one-to-one.
func renderFrontmatterForm(form adminFrontmatterForm) (string, error) {
	if err := validateFrontmatterForm(form); err != nil {
		return "", err
	}
	data := make(map[string]any)
	if strings.TrimSpace(form.Title) != "" {
		data["title"] = strings.TrimSpace(form.Title)
	}
	if strings.TrimSpace(form.Slug) != "" {
		data["slug"] = strings.TrimSpace(form.Slug)
	}
	if strings.TrimSpace(form.Date) != "" {
		data["date"] = strings.TrimSpace(form.Date)
	}
	if strings.TrimSpace(form.Modified) != "" {
		data["modified"] = strings.TrimSpace(form.Modified)
	}
	if strings.TrimSpace(form.Description) != "" {
		data["description"] = strings.TrimSpace(form.Description)
	}
	data["published"] = form.Published
	if strings.TrimSpace(form.TemplateKey) != "" {
		data["templateKey"] = strings.TrimSpace(form.TemplateKey)
	}
	if strings.TrimSpace(form.Author) != "" {
		data["author"] = strings.TrimSpace(form.Author)
	}
	if len(form.Authors) > 0 {
		authors := make([]string, 0, len(form.Authors))
		for _, author := range form.Authors {
			author = strings.TrimSpace(author)
			if author != "" {
				authors = append(authors, author)
			}
		}
		if len(authors) > 0 {
			data["authors"] = authors
		}
	}
	if len(form.Tags) > 0 {
		tags := make([]string, 0, len(form.Tags))
		for _, tag := range form.Tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		if len(tags) > 0 {
			data["tags"] = tags
		}
	}
	for _, extra := range form.Extras {
		key := strings.TrimSpace(extra.Key)
		value := strings.TrimSpace(extra.Value)
		if key == "" || value == "" {
			continue
		}
		data[key] = parseStringValue(extra.Kind, value)
	}
	raw, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return contentedit.FormatFrontmatter(string(raw))
}

func stringifyFrontmatterScalar(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case time.Time:
		return typed.UTC().Format(time.RFC3339)
	default:
		return fmt.Sprint(value)
	}
}

func validateFrontmatterForm(form adminFrontmatterForm) error {
	if strings.TrimSpace(form.Title) == "" {
		return fmt.Errorf("title is required")
	}
	if slug := strings.TrimSpace(form.Slug); slug != "" && slug != adminSlugify(slug) {
		return fmt.Errorf("slug must be lowercase, URL-safe text")
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "date", value: form.Date},
		{name: "modified", value: form.Modified},
	} {
		if strings.TrimSpace(field.value) == "" {
			continue
		}
		if _, err := parseFrontmatterDate(field.value); err != nil {
			return fmt.Errorf("%s must be YYYY-MM-DD or RFC3339", field.name)
		}
	}
	if templateKey := strings.TrimSpace(form.TemplateKey); templateKey != "" {
		if !isKnownAdminTemplateType(templateKey) {
			return fmt.Errorf("type must be one of the known templates")
		}
	}
	return nil
}

func parseFrontmatterDate(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	for _, layout := range []string{time.RFC3339, "2006-01-02", "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date")
}

func parseSettingsForm(content string) adminSettingsForm {
	var cfg adminConfigFile
	if _, err := toml.Decode(content, &cfg); err != nil {
		return adminSettingsForm{}
	}
	return adminSettingsForm{
		Title:                  cfg.MarkataGo.Title,
		Author:                 cfg.MarkataGo.Author,
		URL:                    cfg.MarkataGo.URL,
		Description:            cfg.MarkataGo.Description,
		OutputDir:              cfg.MarkataGo.OutputDir,
		TemplatesDir:           cfg.MarkataGo.TemplatesDir,
		AssetsDir:              cfg.MarkataGo.AssetsDir,
		ThemePalette:           cfg.MarkataGo.Theme.Palette,
		ThemeLight:             cfg.MarkataGo.Theme.PaletteLight,
		ThemeDark:              cfg.MarkataGo.Theme.PaletteDark,
		ThemeMode:              cfg.MarkataGo.Theme.FallbackMode,
		SearchEnabled:          boolPtrValue(cfg.MarkataGo.Search.Enabled, true),
		SearchPosition:         cfg.MarkataGo.Search.Position,
		SearchPlaceholder:      cfg.MarkataGo.Search.Placeholder,
		PagefindBundleDir:      cfg.MarkataGo.Search.Pagefind.BundleDir,
		PagefindVersion:        cfg.MarkataGo.Search.Pagefind.Version,
		PagefindAutoInstall:    boolPtrValue(cfg.MarkataGo.Search.Pagefind.AutoInstall, true),
		ThemeSwitcherEnabled:   boolPtrValue(cfg.MarkataGo.Theme.Switcher.Enabled, false),
		ThemeSwitcherPosition:  cfg.MarkataGo.Theme.Switcher.Position,
		ThemeModeToggleEnabled: boolPtrValue(cfg.MarkataGo.Theme.Switcher.ModeToggle, true),
		ThemeIncludeAll:        boolPtrValue(cfg.MarkataGo.Theme.Switcher.IncludeAll, true),
		FontFamily:             cfg.MarkataGo.Theme.Font.Family,
		FontHeadingFamily:      cfg.MarkataGo.Theme.Font.HeadingFamily,
		FontCodeFamily:         cfg.MarkataGo.Theme.Font.CodeFamily,
		FontSize:               cfg.MarkataGo.Theme.Font.Size,
		FontLineHeight:         cfg.MarkataGo.Theme.Font.LineHeight,
	}
}

func renderSettingsForm(content string, form adminSettingsForm) (string, error) {
	if err := validateSettingsForm(form); err != nil {
		return "", err
	}
	content = setTOMLValue(content, "markata-go", "title", form.Title)
	content = setTOMLValue(content, "markata-go", "author", form.Author)
	content = setTOMLValue(content, "markata-go", "url", form.URL)
	content = setTOMLValue(content, "markata-go", "description", form.Description)
	content = setTOMLValue(content, "markata-go", "output_dir", form.OutputDir)
	content = setTOMLValue(content, "markata-go", "templates_dir", form.TemplatesDir)
	content = setTOMLValue(content, "markata-go", "assets_dir", form.AssetsDir)
	content = setTOMLValue(content, "markata-go.theme", "palette", form.ThemePalette)
	content = setTOMLValue(content, "markata-go.theme", "palette_light", form.ThemeLight)
	content = setTOMLValue(content, "markata-go.theme", "palette_dark", form.ThemeDark)
	content = setTOMLValue(content, "markata-go.theme", "fallback_mode", form.ThemeMode)
	content = setTOMLBoolValue(content, "markata-go.search", "enabled", form.SearchEnabled)
	content = setTOMLValue(content, "markata-go.search", "position", form.SearchPosition)
	content = setTOMLValue(content, "markata-go.search", "placeholder", form.SearchPlaceholder)
	content = setTOMLValue(content, "markata-go.search.pagefind", "bundle_dir", form.PagefindBundleDir)
	content = setTOMLValue(content, "markata-go.search.pagefind", "version", form.PagefindVersion)
	content = setTOMLBoolValue(content, "markata-go.search.pagefind", "auto_install", form.PagefindAutoInstall)
	content = setTOMLBoolValue(content, "markata-go.theme.switcher", "enabled", form.ThemeSwitcherEnabled)
	content = setTOMLBoolValue(content, "markata-go.theme.switcher", "mode_toggle", form.ThemeModeToggleEnabled)
	content = setTOMLBoolValue(content, "markata-go.theme.switcher", "include_all", form.ThemeIncludeAll)
	content = setTOMLValue(content, "markata-go.theme.switcher", "position", form.ThemeSwitcherPosition)
	content = setTOMLValue(content, "markata-go.theme.font", "family", form.FontFamily)
	content = setTOMLValue(content, "markata-go.theme.font", "heading_family", form.FontHeadingFamily)
	content = setTOMLValue(content, "markata-go.theme.font", "code_family", form.FontCodeFamily)
	content = setTOMLValue(content, "markata-go.theme.font", "size", form.FontSize)
	content = setTOMLValue(content, "markata-go.theme.font", "line_height", form.FontLineHeight)
	return content, nil
}

func boolPtrValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func validateSettingsContent(content string) error {
	form := parseSettingsForm(content)
	return validateSettingsForm(form)
}

func validateSettingsForm(form adminSettingsForm) error {
	loader := palettes.NewLoader()
	for _, palette := range []struct {
		name  string
		value string
	}{
		{name: "theme.palette", value: form.ThemePalette},
		{name: "theme.palette_light", value: form.ThemeLight},
		{name: "theme.palette_dark", value: form.ThemeDark},
	} {
		if strings.TrimSpace(palette.value) == "" {
			continue
		}
		if _, err := loader.Load(strings.TrimSpace(palette.value)); err != nil {
			return fmt.Errorf("%s must be one of the known palettes", palette.name)
		}
	}
	if form.ThemeMode != "" && form.ThemeMode != adminThemeLight && form.ThemeMode != adminThemeDark {
		return fmt.Errorf("theme.fallback_mode must be light or dark")
	}
	if form.SearchPosition != "" && !stringInSlice(form.SearchPosition, []string{"navbar", "sidebar", "footer", "custom"}) {
		return fmt.Errorf("search.position must be navbar, sidebar, footer, or custom")
	}
	if form.PagefindVersion != "" && strings.TrimSpace(form.PagefindVersion) == "" {
		return fmt.Errorf("search.pagefind.version cannot be blank")
	}
	if form.ThemeSwitcherPosition != "" && !stringInSlice(form.ThemeSwitcherPosition, []string{"header", "footer"}) {
		return fmt.Errorf("theme.switcher.position must be header or footer")
	}
	if form.URL != "" && !strings.HasPrefix(form.URL, "http://") && !strings.HasPrefix(form.URL, "https://") {
		return fmt.Errorf("url must start with http:// or https://")
	}
	return nil
}

func stringInSlice(value string, values []string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func interfaceSliceToStrings(value any) []string {
	switch typed := value.(type) {
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			result = append(result, fmt.Sprint(item))
		}
		return result
	case []string:
		return typed
	default:
		if value == nil {
			return nil
		}
		return []string{fmt.Sprint(value)}
	}
}

func interfaceToScalarString(value any) string {
	switch typed := value.(type) {
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, fmt.Sprint(item))
		}
		return strings.Join(parts, ", ")
	case map[string]any:
		raw, err := yaml.Marshal(typed)
		if err != nil {
			return fmt.Sprint(value)
		}
		return strings.TrimSpace(string(raw))
	default:
		return fmt.Sprint(value)
	}
}

func detectValueKind(value any) string {
	switch value.(type) {
	case bool:
		return "bool"
	case []any, []string:
		return "list"
	case map[string]any:
		return "object"
	default:
		return "string"
	}
}

func parseStringValue(kind, value string) any {
	trimmed := strings.TrimSpace(value)
	switch kind {
	case "bool":
		return trimmed == "true"
	case "list":
		parts := strings.Split(trimmed, ",")
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				values = append(values, part)
			}
		}
		return values
	case "object":
		var parsed map[string]any
		if err := yaml.Unmarshal([]byte(trimmed), &parsed); err == nil && len(parsed) > 0 {
			return parsed
		}
		return trimmed
	}
	if trimmed == "true" {
		return true
	}
	if trimmed == "false" {
		return false
	}
	if strings.Contains(trimmed, ",") {
		parts := strings.Split(trimmed, ",")
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				values = append(values, part)
			}
		}
		if len(values) > 1 {
			return values
		}
	}
	return trimmed
}

func setTOMLValue(content, section, key, value string) string {
	lines := strings.Split(content, "\n")
	sectionIndex := -1
	insertAt := len(lines)
	current := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if current == section && insertAt == len(lines) {
				insertAt = i
			}
			current = strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")
			if current == section {
				sectionIndex = i
			}
			continue
		}
		if current == section && strings.HasPrefix(strings.TrimSpace(line), key+" =") {
			if strings.TrimSpace(value) == "" {
				return strings.Join(append(lines[:i], lines[i+1:]...), "\n")
			}
			lines[i] = key + " = " + fmt.Sprintf("%q", strings.TrimSpace(value))
			return strings.Join(lines, "\n")
		}
	}
	if strings.TrimSpace(value) == "" {
		return content
	}
	line := key + " = " + fmt.Sprintf("%q", strings.TrimSpace(value))
	if sectionIndex == -1 {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "["+section+"]", line)
		return strings.Join(lines, "\n")
	}
	lines = append(lines[:insertAt], append([]string{line}, lines[insertAt:]...)...)
	return strings.Join(lines, "\n")
}

func setTOMLBoolValue(content, section, key string, value bool) string {
	lines := strings.Split(content, "\n")
	sectionIndex := -1
	insertAt := len(lines)
	current := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if current == section && insertAt == len(lines) {
				insertAt = i
			}
			current = strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")
			if current == section {
				sectionIndex = i
			}
			continue
		}
		if current == section && strings.HasPrefix(strings.TrimSpace(line), key+" =") {
			lines[i] = fmt.Sprintf("%s = %t", key, value)
			return strings.Join(lines, "\n")
		}
	}
	line := fmt.Sprintf("%s = %t", key, value)
	if sectionIndex == -1 {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "["+section+"]", line)
		return strings.Join(lines, "\n")
	}
	lines = append(lines[:insertAt], append([]string{line}, lines[insertAt:]...)...)
	return strings.Join(lines, "\n")
}

const pageHeadTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} - markata-go Admin</title>
  <style>
    {{.ThemeCSS}}
    :root {
      color-scheme: light dark;
      --admin-shell-max: 2100px;
      --admin-header-height: 4.5rem;
      --admin-gap: clamp(0.85rem, 1vw, 1.2rem);
      --admin-radius: 18px;
      --admin-radius-sm: 12px;
      --admin-line: color-mix(in srgb, var(--admin-border) 72%, transparent);
      --admin-panel: color-mix(in srgb, var(--admin-surface) 92%, var(--admin-bg));
      --admin-panel-strong: color-mix(in srgb, var(--admin-surfaceAlt) 86%, var(--admin-surface));
      --admin-glow: color-mix(in srgb, var(--admin-accent) 18%, transparent);
      --admin-editor-page: color-mix(in srgb, var(--admin-surface) 94%, white);
      --admin-editor-rule: color-mix(in srgb, var(--admin-border) 48%, transparent);
      --admin-editor-ink: color-mix(in srgb, var(--admin-text) 94%, transparent);
      font-family: var(--admin-font-body, Georgia, serif);
    }
    * { box-sizing: border-box; }
    html, body { margin: 0; min-height: 100%; background: radial-gradient(circle at top, color-mix(in srgb, var(--admin-glow) 45%, transparent) 0%, transparent 35%), linear-gradient(180deg, color-mix(in srgb, var(--admin-bg) 92%, var(--admin-surfaceAlt)) 0%, var(--admin-bg) 100%); color: var(--admin-text); }
    body { font-family: var(--admin-font-body, "Iowan Old Style", "Palatino Linotype", "Book Antiqua", Palatino, Georgia, serif); font-size: var(--admin-font-size, 16px); line-height: var(--admin-line-height, 1.6); }
    a { color: var(--admin-accent); }
    a:hover { color: var(--admin-accentHover); }
    button, input, textarea, select { font: inherit; }
    .shell { width: min(calc(100vw - 1.5rem), var(--admin-shell-max)); margin: 0 auto; padding: clamp(0.75rem, 1vw, 1.2rem); }
    .card { background: linear-gradient(180deg, color-mix(in srgb, var(--admin-surface) 96%, white) 0%, var(--admin-panel) 100%); border: 1px solid var(--admin-line); border-radius: var(--admin-radius); box-shadow: 0 32px 60px -42px var(--admin-shadow); }
    .nav { display: flex; min-height: var(--admin-header-height); justify-content: space-between; align-items: center; gap: 1rem; padding: 0.95rem 1.1rem; margin-bottom: var(--admin-gap); backdrop-filter: blur(16px); }
    .brand { display: flex; flex-direction: column; gap: 0.1rem; text-decoration: none; color: var(--admin-text); }
    .brand strong { font-family: var(--admin-font-heading, "Avenir Next Condensed", "Franklin Gothic Medium", "Arial Narrow", sans-serif); letter-spacing: 0.08em; font-size: 0.82rem; text-transform: uppercase; color: var(--admin-muted); }
    .brand span { font-size: 1.05rem; font-weight: 700; }
    .nav nav { display: flex; gap: 0.65rem; align-items: center; flex-wrap: wrap; }
    .nav nav a { text-decoration: none; padding: 0.55rem 0.75rem; border-radius: 999px; color: var(--admin-muted); }
    .nav nav a:hover { background: color-mix(in srgb, var(--admin-surfaceAlt) 88%, transparent); color: var(--admin-text); }
    .btn, button { appearance: none; border: 1px solid transparent; border-radius: 999px; cursor: pointer; padding: 0.72rem 1rem; line-height: 1; }
    .btn-primary { background: linear-gradient(135deg, var(--admin-accent) 0%, color-mix(in srgb, var(--admin-accentHover) 85%, var(--admin-accent)) 100%); color: var(--admin-accentContrast); box-shadow: 0 18px 32px -26px var(--admin-accent); }
    .btn-secondary { background: color-mix(in srgb, var(--admin-surfaceAlt) 90%, transparent); color: var(--admin-text); border-color: var(--admin-line); }
    .btn-ghost { background: transparent; color: var(--admin-muted); border-color: var(--admin-line); }
    .btn-link { text-decoration: none; display: inline-flex; align-items: center; }
    .btn:disabled, button:disabled { opacity: 0.68; cursor: wait; }
    .hero { padding: 1.1rem 1.2rem; margin-bottom: var(--admin-gap); }
    .hero h1 { margin: 0 0 0.35rem; font-family: var(--admin-font-heading, "Avenir Next Condensed", "Franklin Gothic Medium", "Arial Narrow", sans-serif); font-size: clamp(1.75rem, 2vw, 2.7rem); line-height: 0.96; letter-spacing: 0.01em; }
    .hero p { margin: 0; max-width: 68ch; color: var(--admin-muted); }
    .stats { display: grid; gap: var(--admin-gap); grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); margin-bottom: var(--admin-gap); }
    .stat { padding: 1rem 1.1rem; }
    .stat-label { display: block; color: var(--admin-muted); font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.08em; }
    .stat-value { display: block; margin-top: 0.35rem; font-size: 1.15rem; font-weight: 700; }
    .panel { padding: 1rem; }
    .stack { display: grid; gap: var(--admin-gap); }
    .workspace { display: grid; gap: var(--admin-gap); align-items: start; grid-template-columns: 1fr; }
    .workspace[data-center-collapsed="false"][data-right-collapsed="false"] { grid-template-columns: minmax(280px, 22vw) minmax(0, 1fr) minmax(320px, 28vw); }
    .workspace[data-center-collapsed="false"][data-right-collapsed="true"] { grid-template-columns: minmax(280px, 28vw) minmax(0, 1fr); }
    .workspace[data-center-collapsed="true"][data-right-collapsed="false"] { grid-template-columns: minmax(0, 1fr) minmax(320px, 32vw); }
    .workspace-two { display: grid; gap: var(--admin-gap); align-items: start; grid-template-columns: minmax(320px, 24vw) minmax(540px, 1fr); }
    .resize-handle { display: none; }
    .pane { min-width: 0; }
    .pane-sticky { position: sticky; top: 1rem; }
    .pane-head { display: flex; justify-content: space-between; gap: 0.75rem; align-items: center; margin-bottom: 0.9rem; }
    .pane-head h2, .pane-head h3 { margin: 0; font-family: var(--admin-font-heading, "Avenir Next Condensed", "Franklin Gothic Medium", "Arial Narrow", sans-serif); font-size: 1.02rem; letter-spacing: 0.05em; text-transform: uppercase; }
    .pane-subtitle { color: var(--admin-muted); font-size: 0.9rem; }
    .toolbar, .toolbar-actions, .toolbar-group, .segmented { display: flex; gap: 0.65rem; flex-wrap: wrap; align-items: center; }
    .toolbar { justify-content: space-between; }
    .shell-actions { display: flex; gap: 0.55rem; flex-wrap: wrap; align-items: center; }
    .segment { padding: 0.52rem 0.8rem; border-radius: 999px; border: 1px solid var(--admin-line); background: transparent; color: var(--admin-muted); }
    .segment.active { background: var(--admin-accent); border-color: transparent; color: var(--admin-accentContrast); }
    .icon-btn { width: 2.2rem; height: 2.2rem; display: inline-flex; align-items: center; justify-content: center; border-radius: 999px; border: 1px solid var(--admin-line); background: transparent; color: var(--admin-muted); }
    .icon-btn:hover { color: var(--admin-text); background: color-mix(in srgb, var(--admin-surfaceAlt) 80%, transparent); }
    .pill { display: inline-flex; align-items: center; gap: 0.35rem; border-radius: 999px; padding: 0.33rem 0.68rem; background: color-mix(in srgb, var(--admin-surfaceAlt) 86%, transparent); color: var(--admin-muted); font-size: 0.85rem; }
    .pill.active { background: color-mix(in srgb, var(--admin-accent) 88%, transparent); color: var(--admin-accentContrast); }
    .status { min-height: 1.3rem; padding: 0.9rem 1rem; border: 1px solid var(--admin-line); border-radius: var(--admin-radius-sm); background: color-mix(in srgb, var(--admin-surfaceAlt) 72%, transparent); }
    .status[data-state="success"] { color: var(--admin-success); }
    .status[data-state="error"] { color: var(--admin-error); }
    .status[data-state="building"] { color: var(--admin-warning); }
    .status[data-state="dirty"] { color: var(--admin-accent); }
    .status-bar { display: flex; justify-content: space-between; gap: 0.75rem; align-items: center; flex-wrap: wrap; padding-top: 0.85rem; border-top: 1px solid var(--admin-line); color: var(--admin-muted); font-size: 0.88rem; }
    .field-grid { display: grid; gap: 0.85rem; grid-template-columns: repeat(2, minmax(0, 1fr)); align-items: start; }
    .field-grid-3 { display: grid; gap: 0.85rem; grid-template-columns: repeat(3, minmax(0, 1fr)); }
    .field-span-2 { grid-column: span 2; }
    .field-span-3 { grid-column: span 3; }
    .field { display: grid; gap: 0.45rem; min-width: 0; align-content: start; }
    .field label, label { display: block; margin-bottom: 0.42rem; color: var(--admin-muted); font-size: 0.82rem; text-transform: uppercase; letter-spacing: 0.08em; font-family: var(--admin-font-heading, "Avenir Next Condensed", "Franklin Gothic Medium", "Arial Narrow", sans-serif); }
    input, textarea, select { width: 100%; border-radius: var(--admin-radius-sm); border: 1px solid var(--admin-line); background: color-mix(in srgb, var(--admin-bg) 76%, var(--admin-surface)); color: var(--admin-text); padding: 0.8rem 0.92rem; }
    input:hover, textarea:hover, select:hover { border-color: color-mix(in srgb, var(--admin-accent) 35%, var(--admin-line)); }
    input:focus, textarea:focus, select:focus { outline: 2px solid color-mix(in srgb, var(--admin-accent) 28%, transparent); border-color: var(--admin-accent); }
    textarea { min-height: 14rem; resize: vertical; }
    .mono { font-family: var(--admin-font-code, "SFMono-Regular", "JetBrains Mono", Consolas, monospace); }
    .editor-body { min-height: 62vh; border: 0; border-radius: calc(var(--admin-radius) - 6px); background: linear-gradient(180deg, color-mix(in srgb, var(--admin-editor-page) 96%, white) 0%, var(--admin-editor-page) 100%); color: var(--admin-editor-ink); padding: 1.6rem clamp(1rem, 2vw, 2rem); line-height: calc(var(--admin-line-height, 1.6) + 0.18); font-size: clamp(1.04rem, 0.95rem + 0.25vw, 1.15rem); font-family: var(--admin-font-body, Georgia, serif); box-shadow: inset 0 1px 0 rgba(255,255,255,0.5), 0 24px 50px -42px var(--admin-shadow); }
    .editor-body::placeholder { color: color-mix(in srgb, var(--admin-muted) 75%, transparent); }
    .preview-frame { width: 100%; min-height: 62vh; border: 1px solid var(--admin-line); border-radius: var(--admin-radius); background: white; }
    .code-panel textarea { min-height: 48vh; }
    .meta-section { display: grid; gap: 0.85rem; }
    .meta-card { padding: 1rem; border: 1px solid var(--admin-line); border-radius: var(--admin-radius-sm); background: color-mix(in srgb, var(--admin-surfaceAlt) 75%, transparent); }
    .meta-card h3 { margin: 0 0 0.8rem; font-size: 0.95rem; }
    .properties-card { overflow: visible; padding: 1.15rem; }
    .properties-layout { display: grid; gap: 1rem; }
    .properties-section { display: grid; gap: 0.85rem; padding-top: 0.9rem; border-top: 1px solid color-mix(in srgb, var(--admin-border) 28%, transparent); }
    .properties-section:first-of-type { padding-top: 0; border-top: 0; }
    .properties-section-head { display: flex; justify-content: space-between; gap: 0.75rem; align-items: center; }
    .properties-section-title { margin: 0; font-family: var(--admin-font-heading, sans-serif); font-size: 0.96rem; letter-spacing: 0.06em; text-transform: uppercase; }
    .properties-section-copy { margin: 0.2rem 0 0; color: var(--admin-muted); font-size: 0.9rem; }
    .field-label { display: flex; align-items: center; gap: 0.35rem; margin: 0; flex-wrap: wrap; }
    .field[data-invalid="true"] input, .field[data-invalid="true"] textarea, .field[data-invalid="true"] select, .field[data-invalid="true"] .checkbox-field { border-color: var(--admin-error); outline-color: color-mix(in srgb, var(--admin-error) 22%, transparent); }
    .field-help { position: relative; display: inline-flex; align-items: center; justify-content: center; width: 1rem; height: 1rem; border-radius: 999px; border: 1px solid var(--admin-line); background: transparent; color: var(--admin-muted); font-size: 0.72rem; cursor: help; }
    .field-help::after { content: attr(data-help); position: absolute; left: 50%; top: calc(100% + 0.45rem); transform: translateX(-50%); width: min(18rem, 70vw); padding: 0.65rem 0.75rem; border-radius: 12px; border: 1px solid var(--admin-line); background: color-mix(in srgb, var(--admin-surface) 97%, white); box-shadow: 0 18px 32px -24px var(--admin-shadow); color: var(--admin-text); font-family: var(--admin-font-body, serif); font-size: 0.88rem; line-height: 1.45; text-transform: none; letter-spacing: 0; opacity: 0; pointer-events: none; z-index: 5; }
    .field-help:hover::after, .field-help:focus-visible::after { opacity: 1; }
    .field-inline-help { margin-top: 0.38rem; color: var(--admin-muted); font-size: 0.85rem; }
    .field-default { color: color-mix(in srgb, var(--admin-muted) 88%, transparent); font-style: italic; }
    .field-error { min-height: 1rem; color: var(--admin-error); font-size: 0.82rem; }
    .field-with-action { display: grid; gap: 0.55rem; grid-template-columns: minmax(0, 1fr) auto; align-items: center; }
    .field-with-action input, .field-with-action select { grid-column: 1; }
    .field-with-action .btn { grid-column: 2; }
    .field-with-action .field-default, .field-with-action .field-error { grid-column: 1 / -1; }
    .dashboard-filters { display: flex; gap: 0.75rem; flex-wrap: wrap; align-items: center; margin-bottom: 1rem; }
    .dashboard-filters input, .dashboard-filters select { max-width: 18rem; }
    .posts-table-empty { padding: 1rem 0; color: var(--admin-muted); }
    .checkbox-field { display: flex; align-items: center; gap: 0.7rem; min-height: 3.1rem; padding: 0.8rem 0.92rem; border-radius: var(--admin-radius-sm); border: 1px solid var(--admin-line); background: color-mix(in srgb, var(--admin-bg) 76%, var(--admin-surface)); }
    .checkbox-field input[type="checkbox"] { width: 1rem; height: 1rem; margin: 0; accent-color: var(--admin-accent); }
    .checkbox-copy { display: grid; gap: 0.18rem; }
    .checkbox-copy strong { font-size: 0.95rem; }
    .checkbox-copy span { color: var(--admin-muted); font-size: 0.85rem; }
    .author-picker { display: flex; flex-wrap: wrap; gap: 0.55rem; }
    .author-search { width: 100%; margin-bottom: 0.15rem; }
    .author-chip { display: inline-flex; align-items: center; gap: 0.4rem; border-radius: 999px; border: 1px solid var(--admin-line); background: color-mix(in srgb, var(--admin-surface) 76%, transparent); color: var(--admin-text); padding: 0.52rem 0.78rem; }
    .author-chip.active { background: color-mix(in srgb, var(--admin-accent) 90%, transparent); color: var(--admin-accentContrast); border-color: transparent; }
    .author-chip small { color: inherit; opacity: 0.78; }
    .properties-footer { display: flex; justify-content: space-between; gap: 0.75rem; flex-wrap: wrap; padding-top: 0.85rem; border-top: 1px solid color-mix(in srgb, var(--admin-border) 28%, transparent); color: var(--admin-muted); font-size: 0.84rem; }
    .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }
    .editor-note { overflow: hidden; background: linear-gradient(180deg, color-mix(in srgb, var(--admin-surface) 94%, white) 0%, color-mix(in srgb, var(--admin-surfaceAlt) 65%, var(--admin-surface)) 100%); }
    .editor-note .pane-head { margin-bottom: 0.6rem; }
    .note-kicker { display: inline-block; margin-bottom: 0.45rem; color: var(--admin-muted); font-size: 0.76rem; text-transform: uppercase; letter-spacing: 0.12em; font-family: var(--admin-font-heading, sans-serif); }
    .note-title { margin: 0; font-family: var(--admin-font-heading, sans-serif); font-size: clamp(1.6rem, 1.1rem + 1vw, 2.4rem); line-height: 0.98; }
    .note-subtitle { margin: 0.4rem 0 0; max-width: 48rem; color: var(--admin-muted); }
    .editor-meta-strip { display: flex; flex-wrap: wrap; gap: 0.55rem; margin: 0.75rem 0 1rem; }
    .editor-meta-strip .pill { background: color-mix(in srgb, var(--admin-surface) 55%, transparent); }
    .editor-topbar { display: flex; justify-content: space-between; gap: 0.75rem; align-items: center; flex-wrap: wrap; margin-bottom: 0.85rem; }
    .shortcut-cluster { display: flex; gap: 0.55rem; align-items: center; flex-wrap: wrap; }
    .shortcut-btn { display: inline-flex; align-items: center; gap: 0.45rem; border-radius: 999px; border: 1px solid var(--admin-line); background: color-mix(in srgb, var(--admin-surface) 70%, transparent); color: var(--admin-text); padding: 0.55rem 0.8rem; }
    .shortcut-btn.active { background: color-mix(in srgb, var(--admin-accent) 92%, transparent); color: var(--admin-accentContrast); border-color: transparent; }
    .shortcut-btn kbd, .command-empty kbd { font-family: var(--admin-font-code, monospace); font-size: 0.78rem; padding: 0.16rem 0.38rem; border-radius: 999px; background: color-mix(in srgb, var(--admin-bg) 55%, var(--admin-surface)); border: 1px solid color-mix(in srgb, var(--admin-border) 40%, transparent); }
    .editor-sheet { position: relative; padding: 0.6rem; border-radius: var(--admin-radius); background: linear-gradient(180deg, color-mix(in srgb, var(--admin-surface) 84%, transparent) 0%, color-mix(in srgb, var(--admin-bg) 55%, var(--admin-surface)) 100%); border: 1px solid color-mix(in srgb, var(--admin-border) 45%, transparent); }
    .editor-sheet::before { content: ""; position: absolute; inset: 0.75rem; border-radius: calc(var(--admin-radius) - 10px); border: 1px solid color-mix(in srgb, var(--admin-editor-rule) 55%, transparent); pointer-events: none; }
    .command-palette { position: absolute; left: 1rem; right: 1rem; bottom: 1rem; z-index: 3; border-radius: calc(var(--admin-radius) - 4px); border: 1px solid color-mix(in srgb, var(--admin-border) 50%, transparent); background: color-mix(in srgb, var(--admin-surface) 97%, white); box-shadow: 0 28px 48px -34px var(--admin-shadow); overflow: hidden; }
    .command-palette[hidden] { display: none; }
    .command-header { display: flex; justify-content: space-between; gap: 0.75rem; align-items: center; padding: 0.85rem 1rem 0.65rem; border-bottom: 1px solid var(--admin-line); }
    .command-header strong { font-family: var(--admin-font-heading, sans-serif); font-size: 0.9rem; letter-spacing: 0.04em; text-transform: uppercase; }
    .command-header span { color: var(--admin-muted); font-size: 0.9rem; }
    .command-list { display: grid; gap: 0; max-height: 18rem; overflow: auto; }
    .command-item { width: 100%; border: 0; border-bottom: 1px solid color-mix(in srgb, var(--admin-border) 28%, transparent); border-radius: 0; padding: 0.85rem 1rem; background: transparent; text-align: left; color: var(--admin-text); }
    .command-item:last-child { border-bottom: 0; }
    .command-item strong { display: block; margin-bottom: 0.14rem; }
    .command-item span { display: block; color: var(--admin-muted); font-size: 0.9rem; }
    .command-item.active, .command-item:hover { background: color-mix(in srgb, var(--admin-accent) 10%, var(--admin-surfaceAlt)); }
    .command-empty { padding: 1rem; color: var(--admin-muted); }
    .preview-card .preview-frame { min-height: 66vh; background: color-mix(in srgb, var(--admin-surface) 80%, white); }
    .preview-frame-shell { padding: 0.45rem; border-radius: var(--admin-radius); background: linear-gradient(180deg, color-mix(in srgb, var(--admin-surface) 82%, transparent) 0%, color-mix(in srgb, var(--admin-bg) 60%, var(--admin-surfaceAlt)) 100%); }
    .overlay-panel { position: relative; width: 100%; padding: 0.7rem; background: transparent; box-shadow: none; overflow-y: visible; transform: none; transition: none; }
    .overlay-panel[hidden] { display: none; }
    .overlay-panel-left { border-right: 1px solid var(--admin-line); }
    .overlay-panel-right { border-left: 1px solid var(--admin-line); }
    .overlay-panel-card { min-height: calc(100vh - 1.4rem); }
    .overlay-close { display: none; }
    body.admin-focus-mode .nav, body.admin-focus-mode .hero, body.admin-focus-mode #post-status, body.admin-focus-mode .toolbar .toolbar-group { display: none; }
    body.admin-focus-mode .shell { width: min(calc(100vw - 0.9rem), 1600px); }
    body.admin-focus-mode .workspace { grid-template-columns: 1fr !important; }
    body.admin-focus-mode .pane-editor { max-width: 1040px; margin: 0 auto; }
    body.typewriter-mode .editor-body { padding-top: 28vh; padding-bottom: 42vh; }
    .list-editor { display: grid; gap: 0.6rem; }
    .list-row { display: grid; gap: 0.55rem; grid-template-columns: minmax(0, 1fr) auto; align-items: center; }
    .key-value-row { display: grid; gap: 0.55rem; grid-template-columns: minmax(0, 0.7fr) 120px minmax(0, 1.3fr) auto; align-items: start; }
    .key-value-row > * { min-width: 0; }
    .tag-editor { display: grid; gap: 0.5rem; }
    .tag-pills { display: flex; flex-wrap: wrap; gap: 0.45rem; min-height: 2.2rem; padding: 0.2rem 0; }
    .tag-pill { display: inline-flex; align-items: center; gap: 0.4rem; padding: 0.4rem 0.7rem; border-radius: 999px; background: color-mix(in srgb, var(--admin-accent) 14%, var(--admin-surfaceAlt)); color: var(--admin-text); border: 1px solid color-mix(in srgb, var(--admin-accent) 22%, var(--admin-line)); }
    .tag-pill button { padding: 0; width: 1.1rem; height: 1.1rem; min-width: 1.1rem; background: transparent; color: var(--admin-muted); border: 0; }
    .tag-input-row { display: grid; gap: 0.55rem; grid-template-columns: minmax(0, 1fr) auto; }
    .outline-list { display: grid; gap: 0.4rem; margin: 0; padding: 0; list-style: none; }
    .outline-list a { text-decoration: none; color: var(--admin-muted); }
    .outline-list a:hover { color: var(--admin-text); }
    .muted { color: var(--admin-muted); }
    .tab-panel[hidden] { display: none; }
    .tabs { display: flex; gap: 0.4rem; margin-bottom: 0.85rem; flex-wrap: wrap; }
    .tab { padding: 0.58rem 0.82rem; border-radius: 999px; border: 1px solid var(--admin-line); background: transparent; color: var(--admin-muted); }
    .tab.active { background: color-mix(in srgb, var(--admin-accent) 90%, transparent); border-color: transparent; color: var(--admin-accentContrast); }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 0.92rem 1rem; border-bottom: 1px solid var(--admin-line); text-align: left; vertical-align: top; }
    th { color: var(--admin-muted); font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.08em; }
    tr:hover td { background: color-mix(in srgb, var(--admin-surfaceAlt) 55%, transparent); }
    .auth { max-width: 30rem; margin: 10vh auto; padding: 1.5rem; }
    .auth h1 { margin-top: 0; }
    .error { color: var(--admin-error); margin: 0 0 1rem; }
    .hide-desktop { display: none; }
    @media (max-width: 1279px) {
      .workspace-two { grid-template-columns: 1fr; }
    }
    @media (max-width: 959px) {
      .shell { width: min(calc(100vw - 0.75rem), var(--admin-shell-max)); padding: 0.4rem; }
      .nav { border-radius: 16px; }
      .workspace, .workspace-two, .field-grid, .field-grid-3 { grid-template-columns: 1fr; }
      .workspace[data-center-collapsed="false"], .workspace[data-right-collapsed="false"] { grid-template-columns: 1fr; }
      .field-span-2, .field-span-3 { grid-column: span 1; }
      .key-value-row { grid-template-columns: 1fr; }
      .pane-sticky { position: static; }
      .preview-frame { min-height: 50vh; }
      .hide-desktop { display: inline-flex; }
      .overlay-panel { position: fixed; top: 0; bottom: 0; z-index: 30; width: min(34rem, calc(100vw - 1rem)); padding: 0.7rem; background: color-mix(in srgb, var(--admin-bg) 78%, rgba(5, 8, 15, 0.82)); backdrop-filter: blur(18px); box-shadow: 0 34px 80px -48px var(--admin-shadow); overflow-y: auto; transform: translateX(0); transition: transform 160ms ease, opacity 160ms ease; }
      .overlay-panel[hidden] { display: block; pointer-events: none; opacity: 0; }
      .overlay-panel-left { left: 0; border-right: 1px solid var(--admin-line); }
      .overlay-panel-left[hidden] { transform: translateX(calc(-100% - 1rem)); }
      .overlay-panel-right { right: 0; border-left: 1px solid var(--admin-line); }
      .overlay-panel-right[hidden] { transform: translateX(calc(100% + 1rem)); }
      .overlay-panel-card { min-height: calc(100vh - 1.4rem); }
      .overlay-close { display: inline-flex; min-width: 6.5rem; }
    }
    @media (min-width: 1800px) {
      .workspace-two { grid-template-columns: 420px minmax(920px, 1fr); }
      .editor-body { min-height: 62vh; }
    }
  </style>
</head>
<body>
<script>
(() => {
  const stored = localStorage.getItem('theme') || localStorage.getItem('color-mode');
  if (stored === 'light' || stored === 'dark') {
    document.documentElement.dataset.theme = stored;
  }
})();
</script>`

const pageFootTemplate = `
</body>
</html>`

const authTemplate = `{{define "auth"}}` + pageHeadTemplate + `
<main class="shell">
  <section class="card auth">
    <h1>{{if .NeedsSetup}}Set up admin access{{else}}Admin login{{end}}</h1>
    <p class="muted">{{if .NeedsSetup}}Create the first local admin account for this site.{{else}}Sign in to edit posts and settings for the local dev server.{{end}}</p>
    {{if .Error}}<p class="error">{{.Error}}</p>{{end}}
    <form method="post" action="{{if .NeedsSetup}}/__admin/setup{{else}}/__admin/login{{end}}" class="stack">
      <div><label for="username">Username</label><input id="username" name="username" required minlength="3"></div>
      <div><label for="password">Password</label><input id="password" name="password" type="password" required minlength="8"></div>
      <button class="btn btn-primary" type="submit">{{if .NeedsSetup}}Create admin{{else}}Login{{end}}</button>
    </form>
  </section>
</main>` + pageFootTemplate + `{{end}}`

const dashboardTemplate = `{{define "dashboard"}}` + pageHeadTemplate + `
<main class="shell">
  <header class="card nav">
    <a class="brand" href="/__admin/dashboard"><strong>Markata local admin</strong><span>Content workspace</span></a>
    <nav><a href="/__admin/dashboard">Posts</a><a href="/__admin/settings">Settings</a><a href="/__admin/logout">Logout</a></nav>
  </header>
  <section class="card hero">
    <h1>Content admin</h1>
    <p>A local writing cockpit for markdown, frontmatter, settings, and preview. Edit safely in forms or raw text, then save to the real source files.</p>
  </section>
  <section class="stats">
    <div class="card stat"><span class="stat-label">Posts</span><span class="stat-value">{{len .Posts}}</span></div>
    <div class="card stat"><span class="stat-label">Preview model</span><span class="stat-value">Save + build</span></div>
    <div class="card stat"><span class="stat-label">Settings editor</span><span class="stat-value">Config file</span></div>
  </section>
  <section class="card panel">
    <div class="toolbar" style="margin-bottom: 1rem;">
      <div><h2 style="margin:0;">Posts</h2><p class="muted" style="margin:0.25rem 0 0;">Open an existing file or start a new one.</p></div>
      <a class="btn btn-primary btn-link" href="/__admin/editor">New post</a>
    </div>
    <div class="dashboard-filters">
      <input id="post-search" type="search" placeholder="Search title, path, slug, or tag type">
      <select id="post-status-filter"><option value="all">All statuses</option><option value="published">Published</option><option value="draft">Draft</option></select>
      <select id="post-type-filter"><option value="all">All types</option>{{range .Posts}}{{if .Type}}<option value="{{.Type}}">{{.Type}}</option>{{end}}{{end}}</select>
    </div>
    <table>
      <thead><tr><th>Title</th><th>Path</th><th>Type</th><th>Date</th><th>Status</th></tr></thead>
      <tbody id="posts-table-body">{{range .Posts}}<tr data-status="{{if .Published}}published{{else}}draft{{end}}" data-type="{{.Type}}"><td><a href="/__admin/editor?path={{.Path}}">{{.Title}}</a></td><td class="mono">{{.Path}}</td><td>{{.Type}}</td><td>{{.Date}}</td><td>{{if .Published}}Published{{else}}Draft{{end}}</td></tr>{{end}}</tbody>
    </table>
    <div id="posts-empty" class="posts-table-empty" hidden>No posts match the current filters.</div>
  </section>
</main>
<script>
(() => {
  const search = document.getElementById('post-search');
  const status = document.getElementById('post-status-filter');
  const type = document.getElementById('post-type-filter');
  const rows = Array.from(document.querySelectorAll('#posts-table-body tr'));
  const empty = document.getElementById('posts-empty');
  const seen = new Set();
  Array.from(type.options).forEach((option) => {
    if (!option.value || option.value === 'all') { return; }
    if (seen.has(option.value)) { option.remove(); return; }
    seen.add(option.value);
  });
  function applyFilters() {
    const query = search.value.trim().toLowerCase();
    const statusValue = status.value;
    const typeValue = type.value;
    let visible = 0;
    rows.forEach((row) => {
      const haystack = row.textContent.toLowerCase();
      const matchesQuery = !query || haystack.includes(query);
      const matchesStatus = statusValue === 'all' || row.dataset.status === statusValue;
      const matchesType = typeValue === 'all' || row.dataset.type === typeValue;
      const show = matchesQuery && matchesStatus && matchesType;
      row.hidden = !show;
      if (show) { visible += 1; }
    });
    empty.hidden = visible !== 0;
  }
  [search, status, type].forEach((element) => element.addEventListener('input', applyFilters));
})();
</script>` + pageFootTemplate + `{{end}}`

const editorTemplate = `{{define "editor"}}` + pageHeadTemplate + `
<main class="shell">
  <header class="card nav">
    <a class="brand" href="/__admin/dashboard"><strong>Markata local admin</strong><span>Document workspace</span></a>
    <nav><a href="/__admin/dashboard">Posts</a><a href="/__admin/settings">Settings</a><a href="/__admin/logout">Logout</a></nav>
  </header>
  <section class="card hero">
    <h1>{{if .Post.Exists}}{{if .Post.Title}}{{.Post.Title}}{{else}}{{.Post.Path}}{{end}}{{else}}New post{{end}}</h1>
    <p>Writing-first markdown workspace with content front and center, plus optional properties and preview when you need them.</p>
  </section>
  {{if not .Post.Exists}}
  <section class="card panel stack">
    <div class="pane-head"><div><h3>New Content Wizard</h3><span class="pane-subtitle">Mirror the markata-go new flow before you start writing</span></div></div>
    <div class="field-grid">
      <div class="field"><label for="new-title">Title</label><input id="new-title" placeholder="My new post"></div>
      <div class="field"><label for="new-template">Type</label><select id="new-template"></select></div>
      <div class="field"><label for="new-directory">Directory</label><select id="new-directory"></select></div>
      <div class="field"><label for="new-custom-directory">Custom directory</label><input id="new-custom-directory" class="mono" placeholder="pages/custom"></div>
      <div class="field field-span-2 tag-editor"><label for="new-tag-input">Tags</label><div id="new-tag-pills" class="tag-pills"></div><div class="tag-input-row"><input id="new-tag-input" list="known-tags" placeholder="Add tags from the site or create new ones"><button id="new-tag-add" class="btn btn-secondary" type="button">Add</button></div></div>
      <div class="field"><label><input id="new-private" type="checkbox" style="width:auto; margin-right:0.5rem;"> Private post</label></div>
      <div id="new-authors-field" class="field field-span-2" hidden><label for="new-authors">Authors</label><select id="new-authors" multiple style="min-height:7rem;"></select></div>
      <div id="new-template-fields" class="field field-span-2 stack"></div>
      <div class="field field-span-2"><button id="generate-scaffold" class="btn btn-primary" type="button">Generate post scaffold</button></div>
    </div>
  </section>
  {{end}}
  <section class="card panel stack">
    <div class="toolbar">
      <div class="toolbar-actions">
        <button id="save-post" class="btn btn-primary" type="button">{{if .Post.Exists}}Save changes{{else}}Create post{{end}}</button>
        <a id="preview-link" class="btn btn-secondary btn-link" href="{{.Post.PreviewURL}}" target="_blank" rel="noreferrer">Open built page</a>
        <button id="toggle-editor-fullscreen" class="btn btn-secondary" type="button">Fullscreen</button>
      </div>
      <div class="shell-actions">
        <button id="open-properties-panel" class="btn btn-secondary" type="button">Properties</button>
        <button id="open-preview-panel" class="btn btn-secondary" type="button">Live</button>
        <span class="pill">{{if .Post.Exists}}Existing file{{else}}New file{{end}}</span><span class="pill" id="dirty-indicator">Saved</span>
      </div>
    </div>
    <div id="post-status" class="status muted">Ready.</div>
    <div id="editor-workspace" class="workspace" data-center-collapsed="true" data-right-collapsed="true">
      <aside id="properties-panel" class="overlay-panel overlay-panel-left" hidden>
        <section class="meta-card properties-card overlay-panel-card">
          <div class="pane-head"><div><h3>Properties</h3><span class="pane-subtitle">Publishing fields and page metadata</span></div><button id="close-properties-panel" class="btn btn-ghost overlay-close" type="button">Close</button></div>
          <div class="tabs" data-tabs="frontmatter-tabs">
            <button class="tab active" type="button" data-tab="frontmatter-form">Properties</button>
            <button class="tab" type="button" data-tab="frontmatter-raw">Raw YAML</button>
          </div>
          <div id="frontmatter-form" class="tab-panel properties-layout">
            <section class="properties-section">
              <div class="field"><label class="field-label" for="post-path">Path <span class="field-help" tabindex="0" data-help="The source markdown file path inside your content directory. Renaming this moves the file on save.">?</span></label><input id="post-path" class="mono" value="{{.Post.Path}}"><div class="field-inline-help">Source file location for this page.</div></div>
            </section>
            <section class="properties-section">
              <div>
                <h4 class="properties-section-title">Core Metadata</h4>
                <p class="properties-section-copy">The fields readers and templates depend on most.</p>
              </div>
              <div class="field-grid">
                <div class="field" id="fm-title-field"><label class="field-label" for="fm-title">Title <span class="field-help" tabindex="0" data-help="Human-facing page title used in templates, lists, and feeds.">?</span></label><input id="fm-title"><div id="fm-title-error" class="field-error"></div></div>
                <div class="field" id="fm-slug-field"><label class="field-label" for="fm-slug">Slug <span class="field-help" tabindex="0" data-help="URL path segment for the page. Use lowercase letters, numbers, and hyphens.">?</span></label><div class="field-with-action"><input id="fm-slug"><button id="fm-slug-reset" class="btn btn-ghost" type="button">Reset</button><div id="fm-slug-default" class="field-inline-help field-default"></div><div id="fm-slug-error" class="field-error"></div></div></div>
                <div class="field" id="fm-date-field"><label class="field-label" for="fm-date">Published Date <span class="field-help" tabindex="0" data-help="Primary publish date for sorting and feeds. Accepts YYYY-MM-DD or a full RFC3339 timestamp.">?</span></label><input id="fm-date" class="mono" placeholder="YYYY-MM-DD or RFC3339"><div id="fm-date-error" class="field-error"></div></div>
                <div class="field" id="fm-modified-field"><label class="field-label" for="fm-modified">Updated Date <span class="field-help" tabindex="0" data-help="Optional last-updated timestamp. Existing posts are auto-filled on save.">?</span></label><input id="fm-modified" class="mono"><div id="fm-modified-default" class="field-inline-help field-default"></div><div id="fm-modified-error" class="field-error"></div></div>
                <div class="field" id="fm-template-key-field"><label class="field-label" for="fm-template-key">Type <span class="field-help" tabindex="0" data-help="The content type or template family for this page, such as post, note, photo, or guide.">?</span></label><select id="fm-template-key"></select><div id="fm-template-key-error" class="field-error"></div></div>
                <div class="field"><label class="field-label" for="fm-published">Visibility <span class="field-help" tabindex="0" data-help="Published pages appear in the built site and feeds unless another field like private excludes them.">?</span></label><div class="checkbox-field"><input id="fm-published" type="checkbox"><div class="checkbox-copy"><strong>Published</strong><span>Include this page in the normal published output.</span></div></div></div>
              </div>
            </section>
            <section class="properties-section">
              <div>
                <h4 class="properties-section-title">Summary And Attribution</h4>
                <p class="properties-section-copy">Describe the page and connect it to people and taxonomy.</p>
              </div>
              <div class="field-grid">
                <div class="field field-span-2"><label class="field-label" for="fm-description">Description <span class="field-help" tabindex="0" data-help="Short summary used for SEO, previews, and feed descriptions.">?</span></label><textarea id="fm-description" style="min-height: 7rem;"></textarea></div>
                <div class="field field-span-2"><label class="field-label" for="fm-authors">Authors <span class="field-help" tabindex="0" data-help="Select one or more configured site authors for this post.">?</span></label><input id="fm-author-search" class="author-search" type="search" placeholder="Search authors"><div id="fm-author-picker" class="author-picker">{{range .KnownAuthors}}<button class="author-chip" type="button" data-author-id="{{.ID}}" data-author-name="{{.Name}}">{{.Name}}{{if .Default}} <small>default</small>{{end}}</button>{{end}}</div><select id="fm-authors" multiple hidden>{{range .KnownAuthors}}<option value="{{.ID}}">{{.Name}}{{if .Default}} (default){{end}}</option>{{end}}</select><div class="field-inline-help">Choose one or more authors for attribution.</div></div>
                <div class="field field-span-2 tag-editor"><label class="field-label" for="fm-tag-input">Tags <span class="field-help" tabindex="0" data-help="Tags group related posts and can power feeds, filters, and taxonomy pages.">?</span></label><div id="fm-tag-pills" class="tag-pills"></div><div class="tag-input-row"><input id="fm-tag-input" list="known-tags" placeholder="Add a tag and press Enter"><button id="fm-tag-add" class="btn btn-secondary" type="button">Add</button></div><datalist id="known-tags">{{range .KnownTags}}<option value="{{.}}"></option>{{end}}</datalist><input id="fm-tags" type="hidden"></div>
              </div>
            </section>
            <section class="properties-section">
              <div class="properties-section-head"><div><h4 class="properties-section-title">Extra Fields</h4><p class="properties-section-copy">Keep uncommon or template-specific frontmatter here.</p></div><button id="fm-add-extra" class="btn btn-ghost" type="button">Add field</button></div>
              <div id="fm-extra-fields" class="list-editor"></div>
            </section>
            <div class="properties-footer"><span id="frontmatter-state">Frontmatter matches the saved file.</span><span id="last-autosaved">No autosave yet.</span></div>
          </div>
          <div id="frontmatter-raw" class="tab-panel" hidden>
            <div class="code-panel"><label for="frontmatter">Frontmatter</label><textarea id="frontmatter" class="mono">{{.Post.Frontmatter}}</textarea></div>
          </div>
        </section>
      </aside>
      <section class="pane pane-editor stack">
        <section class="meta-card editor-note">
          <div class="pane-head"><div><span class="note-kicker">Markdown draft</span><h2 class="note-title">{{if .Post.Title}}{{.Post.Title}}{{else}}{{if .Post.Exists}}Untitled draft{{else}}Start a new note{{end}}{{end}}</h2><p class="note-subtitle">Draft in the site&apos;s own type and palette system, then open properties or live preview only when you need them.</p></div></div>
          <div class="editor-meta-strip"><span class="pill">{{if .Post.Exists}}Existing file{{else}}New file{{end}}</span><span class="pill mono">{{.Post.Path}}</span><span class="pill" id="reading-time">0 min read</span></div>
          <div class="editor-topbar">
            <div class="shortcut-cluster">
              <button id="open-command-palette" class="shortcut-btn" type="button"><span>/ commands</span><kbd>/</kbd></button>
              <span class="muted">Start a new line with <kbd>/</kbd> or press <kbd>Ctrl</kbd>/<kbd>Cmd</kbd> + <kbd>K</kbd></span>
            </div>
            <div class="shortcut-cluster">
              <button id="toggle-focus-mode" class="shortcut-btn" type="button">Focus</button>
              <button id="toggle-typewriter-mode" class="shortcut-btn" type="button">Typewriter</button>
            </div>
          </div>
          <div class="editor-sheet">
            <textarea id="body" class="editor-body" placeholder="Start with a title, a scene, a list, or just a single sentence.">{{.Post.Body}}</textarea>
            <div id="command-palette" class="command-palette" hidden>
              <div class="command-header"><div><strong>Commands</strong><span id="command-context">Markdown inserts</span></div><span id="command-query" class="mono">/</span></div>
              <div id="command-list" class="command-list"></div>
            </div>
          </div>
          <div class="status-bar"><span id="body-stats">0 words</span><span>Cmd/Ctrl+S saves to file</span></div>
        </section>
      </section>
      <aside id="preview-panel" class="overlay-panel overlay-panel-right" hidden>
        <section class="meta-card preview-card overlay-panel-card">
          <div class="pane-head"><div><h3 id="preview-label">Live Preview</h3><div id="preview-help" class="pane-subtitle">Draft rendering updates as you type</div></div><div class="toolbar-group"><div class="segmented"><button id="live-preview-toggle" class="segment active" type="button">Live</button><button id="built-preview-toggle" class="segment" type="button">Built</button></div><button id="close-preview-panel" class="btn btn-ghost overlay-close" type="button">Close</button></div></div>
          <div class="preview-frame-shell"><iframe id="preview-frame" class="preview-frame" src="{{.Post.PreviewURL}}"></iframe></div>
        </section>
      </aside>
    </div>
    <input id="post-base-hash" type="hidden" value="{{.Post.Hash}}">
    <input id="post-exists" type="hidden" value="{{if .Post.Exists}}true{{else}}false{{end}}">
  </section>
</main>
<script>
(() => {
  const NEW_POST_CONTEXT = {{.NewPostContext}};
  function enableTabs(groupName) {
    const root = document.querySelector('[data-tabs="' + groupName + '"]');
    if (!root) { return; }
    root.querySelectorAll('[data-tab]').forEach((button) => {
      button.addEventListener('click', () => {
        const target = button.getAttribute('data-tab');
        root.querySelectorAll('[data-tab]').forEach((tabButton) => tabButton.classList.toggle('active', tabButton === button));
        document.querySelectorAll('#frontmatter-form, #frontmatter-raw').forEach((panel) => { panel.hidden = panel.id !== target; });
      });
    });
  }
  function escapeHTML(value) {
    return String(value || '').replace(/[&<>"']/g, (char) => ({'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'}[char]));
  }
  const slashCommands = [
    {id: 'h1', label: 'Heading 1', description: 'Insert a top-level heading', snippet: '# '},
    {id: 'h2', label: 'Heading 2', description: 'Insert a section heading', snippet: '## '},
    {id: 'todo', label: 'Task List', description: 'Insert a markdown checklist item', snippet: '- [ ] '},
    {id: 'quote', label: 'Blockquote', description: 'Insert a quoted passage', snippet: '> '},
    {id: 'code', label: 'Code Block', description: 'Insert a fenced code block', snippet: '~~~text\n\n~~~', cursorOffset: 8},
    {id: 'link', label: 'Link', description: 'Insert a markdown link', snippet: '[title](https://example.com)', cursorOffset: 1},
    {id: 'image', label: 'Image', description: 'Insert an image embed', snippet: '![alt text](/images/example.png)', cursorOffset: 2},
    {id: 'divider', label: 'Divider', description: 'Insert a horizontal rule', snippet: '---'},
  ];
  async function postJSON(url, payload) {
    const response = await fetch(url, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }
  const tagState = [];
  const newTagState = [];
  function normalizeTag(tag) { return tag.trim().replace(/^#/, ''); }
  function syncTagsInput() { document.getElementById('fm-tags').value = tagState.join(', '); }
  function renderTagPills() {
    const container = document.getElementById('fm-tag-pills');
    container.innerHTML = '';
    tagState.forEach((tag, index) => {
      const pill = document.createElement('span');
      pill.className = 'tag-pill';
      pill.innerHTML = '<span>#' + tag + '</span><button type="button" aria-label="Remove tag">&times;</button>';
      pill.querySelector('button').addEventListener('click', () => {
        tagState.splice(index, 1);
        syncTagsInput();
        renderTagPills();
        syncFrontmatterFromForm();
        markDirty();
      });
      container.appendChild(pill);
    });
  }
  function setTags(tags) {
    tagState.splice(0, tagState.length, ...tags.map(normalizeTag).filter(Boolean));
    syncTagsInput();
    renderTagPills();
  }
  function addTag(tag) {
    tag = normalizeTag(tag || '');
    if (!tag || tagState.includes(tag)) { return; }
    tagState.push(tag);
    syncTagsInput();
    renderTagPills();
    syncFrontmatterFromForm();
    markDirty();
  }
  function renderNewTagPills() {
    const container = document.getElementById('new-tag-pills');
    if (!container) { return; }
    container.innerHTML = '';
    newTagState.forEach((tag, index) => {
      const pill = document.createElement('span');
      pill.className = 'tag-pill';
      pill.innerHTML = '<span>#' + tag + '</span><button type="button" aria-label="Remove tag">&times;</button>';
      pill.querySelector('button').addEventListener('click', () => {
        newTagState.splice(index, 1);
        renderNewTagPills();
      });
      container.appendChild(pill);
    });
  }
  function addNewTag(tag) {
    tag = normalizeTag(tag || '');
    if (!tag || newTagState.includes(tag)) { return; }
    newTagState.push(tag);
    renderNewTagPills();
  }
  function renderNewAuthorOptions() {
    const select = document.getElementById('new-authors');
    const wrapper = document.getElementById('new-authors-field');
    if (!select || !wrapper) { return; }
    const authors = (NEW_POST_CONTEXT && NEW_POST_CONTEXT.authors) || [];
    select.innerHTML = '';
    if (!authors.length) {
      wrapper.hidden = true;
      return;
    }
    wrapper.hidden = false;
    authors.forEach((author) => {
      const option = document.createElement('option');
      option.value = author.id;
      option.textContent = author.default ? author.name + ' (default)' : author.name;
      option.selected = !!author.default;
      select.appendChild(option);
    });
  }
  function ensureSelectOption(select, value) {
    if (!select || !value) { return; }
    const exists = Array.from(select.options).some((option) => option.value === value);
    if (exists) { return; }
    const option = document.createElement('option');
    option.value = value;
    option.textContent = value;
    select.insertBefore(option, select.lastElementChild);
  }
  function populateNewTemplateFields(templateName) {
    const templates = (NEW_POST_CONTEXT && NEW_POST_CONTEXT.templates) || {};
    const templateDef = templates[templateName] || templates.post || {};
    const container = document.getElementById('new-template-fields');
    if (!container) { return; }
    container.innerHTML = '';
    const extraKeys = Object.keys(templateDef.frontmatter || {}).filter((key) => !['template', 'templateKey', 'title', 'slug', 'date', 'published', 'draft', 'description', 'tags', 'private', 'authors'].includes(key));
    extraKeys.forEach((key) => {
      const field = document.createElement('div');
      field.className = 'field';
      const value = (templateDef.frontmatter || {})[key];
      field.innerHTML = '<label for="new-field-' + key + '">' + key + '</label><input id="new-field-' + key + '" data-field-key="' + key + '" value="' + String(value == null ? '' : value).replace(/"/g, '&quot;') + '">';
      container.appendChild(field);
    });
  }
  async function generateNewPostScaffold() {
    const directorySelect = document.getElementById('new-directory');
    const selectedDir = directorySelect.value === '__custom__' ? document.getElementById('new-custom-directory').value : directorySelect.value;
    const extra = {};
    document.querySelectorAll('#new-template-fields [data-field-key]').forEach((field) => {
      if (field.dataset.fieldKey) {
        extra[field.dataset.fieldKey] = field.value;
      }
    });
    const authorSelect = document.getElementById('new-authors');
    const authors = authorSelect ? Array.from(authorSelect.selectedOptions).map((option) => option.value) : [];
    const result = await postJSON('/__admin/api/new/scaffold', {
      title: document.getElementById('new-title').value,
      template: document.getElementById('new-template').value,
      directory: selectedDir,
      tags: newTagState.slice(),
      private: document.getElementById('new-private').checked,
      authors: authors,
      extra: extra
    });
    document.getElementById('post-path').value = result.path || '';
    document.getElementById('frontmatter').value = result.frontmatter || '';
    body.value = result.body || '';
    await loadFrontmatterForm();
    updateWordCount();
    updateOutline();
    queueLivePreview();
    setStatus('Scaffold generated. Review the fields, then start writing.', 'success');
  }
  function addExtraField(key, value, kind) {
    const container = document.getElementById('fm-extra-fields');
    const row = document.createElement('div');
    row.className = 'key-value-row fm-extra-row';
    row.innerHTML = '<input data-role="key" placeholder="custom_key" value="' + (key || '') + '"><select data-role="kind"><option value="string">string</option><option value="bool">bool</option><option value="list">list</option><option value="object">object</option></select><textarea data-role="value" placeholder="value" style="min-height:4.5rem;">' + (value || '') + '</textarea><button class="btn btn-ghost" type="button">Remove</button>';
    row.querySelector('[data-role="kind"]').value = kind || 'string';
    row.querySelector('button').addEventListener('click', () => { row.remove(); syncFrontmatterFromForm(); markDirty(); });
    row.querySelectorAll('input, textarea, select').forEach((input) => {
      input.addEventListener('input', () => { syncFrontmatterFromForm(); markDirty(); });
      input.addEventListener('change', () => { syncFrontmatterFromForm(); markDirty(); });
    });
    container.appendChild(row);
  }
  async function loadFrontmatterForm() {
    const parsed = await postJSON('/__admin/api/frontmatter/parse', { frontmatter: document.getElementById('frontmatter').value });
    document.getElementById('fm-title').value = parsed.title;
    document.getElementById('fm-slug').value = parsed.slug;
    document.getElementById('fm-date').value = parsed.date;
    document.getElementById('fm-modified').value = parsed.modified;
    document.getElementById('fm-description').value = parsed.description;
    document.getElementById('fm-published').checked = parsed.published;
    document.getElementById('fm-template-key').value = parsed.template_key;
    populateTemplateTypeOptions();
    setSelectedValues(document.getElementById('fm-authors'), parsed.authors || []);
    setTags(parsed.tags || []);
    document.getElementById('fm-extra-fields').innerHTML = '';
    parsed.extras.forEach((field) => addExtraField(field.key, field.value, field.kind));
    syncNoteTitle();
    validateEditorFields();
    updatePropertiesStatus();
  }
  async function syncFrontmatterFromForm() {
    try {
      validateEditorFields();
      const fieldIDs = ['fm-title', 'fm-slug', 'fm-date', 'fm-modified', 'fm-template-key'];
      if (fieldIDs.some((id) => !document.getElementById(id).checkValidity())) {
        return;
      }
      const extras = Array.from(document.querySelectorAll('.fm-extra-row')).map((row) => ({ key: row.querySelector('[data-role="key"]').value, kind: row.querySelector('[data-role="kind"]').value, value: row.querySelector('[data-role="value"]').value }));
      const result = await postJSON('/__admin/api/frontmatter/render', {
        title: document.getElementById('fm-title').value,
        slug: document.getElementById('fm-slug').value,
        date: document.getElementById('fm-date').value,
        modified: document.getElementById('fm-modified').value,
        description: document.getElementById('fm-description').value,
        published: document.getElementById('fm-published').checked,
        template_key: document.getElementById('fm-template-key').value,
        authors: selectedValues(document.getElementById('fm-authors')),
        tags: tagState.slice(),
        extras: extras
      });
      document.getElementById('frontmatter').value = result.frontmatter || '';
      queueLivePreview();
    } catch (error) {
      setStatus(error.message, 'error');
    }
  }
  function applyWorkspaceState() {
    workspace.dataset.centerCollapsed = String(centerCollapsed);
    workspace.dataset.rightCollapsed = String(rightCollapsed);
    propertiesPanel.hidden = centerCollapsed;
    previewPanel.hidden = rightCollapsed;
    propertiesButton.classList.toggle('active', !centerCollapsed);
    previewButton.classList.toggle('active', !rightCollapsed);
    localStorage.setItem('admin-workspace-center', String(centerCollapsed));
    localStorage.setItem('admin-workspace-right', String(rightCollapsed));
  }
  const statusEl = document.getElementById('post-status');
  const workspace = document.getElementById('editor-workspace');
  const saveButton = document.getElementById('save-post');
  const previewFrame = document.getElementById('preview-frame');
  const previewLink = document.getElementById('preview-link');
  const liveToggle = document.getElementById('live-preview-toggle');
  const builtToggle = document.getElementById('built-preview-toggle');
  const previewLabel = document.getElementById('preview-label');
  const previewHelp = document.getElementById('preview-help');
  const baseHash = document.getElementById('post-base-hash');
  const existsInput = document.getElementById('post-exists');
  const body = document.getElementById('body');
  const noteTitle = document.querySelector('.note-title');
  const commandPalette = document.getElementById('command-palette');
  const commandList = document.getElementById('command-list');
  const commandQuery = document.getElementById('command-query');
  const commandContext = document.getElementById('command-context');
  const commandButton = document.getElementById('open-command-palette');
  const focusButton = document.getElementById('toggle-focus-mode');
  const typewriterButton = document.getElementById('toggle-typewriter-mode');
  const fullscreenButton = document.getElementById('toggle-editor-fullscreen');
  const propertiesPanel = document.getElementById('properties-panel');
  const previewPanel = document.getElementById('preview-panel');
  const propertiesButton = document.getElementById('open-properties-panel');
  const previewButton = document.getElementById('open-preview-panel');
  const dirtyIndicator = document.getElementById('dirty-indicator');
  let previewMode = 'live';
  let livePreviewTimer = null;
  let centerCollapsed = localStorage.getItem('admin-workspace-center') === 'true';
  let rightCollapsed = localStorage.getItem('admin-workspace-right');
  rightCollapsed = rightCollapsed == null ? true : rightCollapsed === 'true';
  let focusMode = localStorage.getItem('admin-editor-focus') === 'true';
  let typewriterMode = localStorage.getItem('admin-editor-typewriter') === 'true';
  let commandState = {open: false, query: '', selected: 0, fromKeyboard: false};
  let dirty = false;
  let isSaving = false;
  let autosaveTimer = null;
  let lastSavedFrontmatter = document.getElementById('frontmatter').value || '';
  let lastSavedAt = null;
  function setStatus(message, state) { statusEl.textContent = message; statusEl.dataset.state = state || ''; }
  function setPreviewMode(mode) {
    previewMode = mode;
    const isLive = mode === 'live';
    liveToggle.classList.toggle('btn-primary', isLive);
    liveToggle.classList.toggle('btn-secondary', !isLive);
    builtToggle.classList.toggle('btn-primary', !isLive);
    builtToggle.classList.toggle('btn-secondary', isLive);
    previewLabel.textContent = isLive ? 'Live preview' : 'Built preview';
    previewHelp.textContent = isLive ? 'Updates while you type' : 'Shows the saved site after build';
  }
  async function refreshBuiltPreview() { const url = previewLink.getAttribute('href'); previewFrame.removeAttribute('srcdoc'); previewFrame.src = url + (url.includes('?') ? '&' : '?') + 'admin_ts=' + Date.now(); }
  async function refreshLivePreview() {
    const response = await fetch('/__admin/api/preview', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({ frontmatter: document.getElementById('frontmatter').value, body: document.getElementById('body').value })
    });
    if (!response.ok) {
      setStatus(await response.text(), 'error');
      return;
    }
    previewFrame.srcdoc = await response.text();
  }
  function queueLivePreview() {
    if (previewMode !== 'live') { return; }
    if (livePreviewTimer) { clearTimeout(livePreviewTimer); }
    livePreviewTimer = setTimeout(() => {
      refreshLivePreview().catch((error) => setStatus(error.message, 'error'));
    }, 250);
  }
  function updateWordCount() {
    const words = body.value.trim() ? body.value.trim().split(/\s+/).length : 0;
    document.getElementById('body-stats').textContent = words + ' words';
    document.getElementById('reading-time').textContent = Math.max(1, Math.ceil(words / 220)) + ' min read';
  }
  function filteredCommands() {
    const query = commandState.query.trim().toLowerCase();
    if (!query) { return slashCommands; }
    return slashCommands.filter((command) => command.label.toLowerCase().includes(query) || command.description.toLowerCase().includes(query) || command.id.includes(query));
  }
  function renderCommandPalette() {
    const commands = filteredCommands();
    commandQuery.textContent = '/' + commandState.query;
    commandContext.textContent = commandState.fromKeyboard ? 'Quick inserts from Ctrl/Cmd+K' : 'Markdown inserts for the current line';
    commandList.innerHTML = '';
    if (!commands.length) {
      commandList.innerHTML = '<div class="command-empty">No command matches <kbd>/' + escapeHTML(commandState.query) + '</kbd></div>';
      return;
    }
    commandState.selected = Math.max(0, Math.min(commandState.selected, commands.length - 1));
    commands.forEach((command, index) => {
      const item = document.createElement('button');
      item.type = 'button';
      item.className = 'command-item' + (index === commandState.selected ? ' active' : '');
      item.innerHTML = '<strong>' + command.label + '</strong><span>' + command.description + '</span>';
      item.addEventListener('click', () => applySlashCommand(command));
      commandList.appendChild(item);
    });
  }
  function openCommandPalette(fromKeyboard) {
    commandState.open = true;
    commandState.fromKeyboard = !!fromKeyboard;
    commandPalette.hidden = false;
    commandButton.classList.add('active');
    renderCommandPalette();
  }
  function closeCommandPalette() {
    commandState.open = false;
    commandState.query = '';
    commandState.selected = 0;
    commandState.fromKeyboard = false;
    commandPalette.hidden = true;
    commandButton.classList.remove('active');
  }
  function currentLineRange() {
    const value = body.value;
    const cursor = body.selectionStart;
    const lineStart = value.lastIndexOf('\n', Math.max(0, cursor - 1)) + 1;
    let lineEnd = value.indexOf('\n', cursor);
    if (lineEnd === -1) { lineEnd = value.length; }
    return {start: lineStart, end: lineEnd, text: value.slice(lineStart, lineEnd)};
  }
  function refreshSlashCommandState() {
    const line = currentLineRange().text;
    const match = line.match(/^\/([^\s]*)$/);
    if (match) {
      commandState.query = match[1] || '';
      openCommandPalette(false);
      return true;
    }
    if (!commandState.fromKeyboard) {
      closeCommandPalette();
    }
    return false;
  }
  function insertSnippet(command) {
    const value = body.value;
    const cursor = body.selectionStart;
    const line = currentLineRange();
    const slashMatch = line.text.match(/^\/([^\s]*)$/);
    let start = cursor;
    let end = cursor;
    if (slashMatch) {
      start = line.start;
      end = line.end;
    }
    const before = value.slice(0, start);
    const after = value.slice(end);
    const prefix = before && !before.endsWith('\n') ? '\n' : '';
    const suffix = after && !after.startsWith('\n') && command.snippet.includes('\n') ? '\n' : '';
    body.value = before + prefix + command.snippet + suffix + after;
    const caret = before.length + prefix.length + (command.cursorOffset == null ? command.snippet.length : command.cursorOffset);
    body.focus();
    body.setSelectionRange(caret, caret);
  }
  function applySlashCommand(command) {
    insertSnippet(command);
    closeCommandPalette();
    updateWordCount();
    updateOutline();
    queueLivePreview();
  }
  function applyEditorModes() {
    document.body.classList.toggle('admin-focus-mode', focusMode);
    document.body.classList.toggle('typewriter-mode', typewriterMode);
    focusButton.classList.toggle('active', focusMode);
    typewriterButton.classList.toggle('active', typewriterMode);
    fullscreenButton.classList.toggle('active', focusMode);
    localStorage.setItem('admin-editor-focus', String(focusMode));
    localStorage.setItem('admin-editor-typewriter', String(typewriterMode));
  }
  function updateDirtyIndicator() {
    dirtyIndicator.textContent = dirty ? 'Unsaved changes' : 'Saved';
    dirtyIndicator.classList.toggle('active', dirty);
  }
  function updatePropertiesStatus() {
    const frontmatterValue = document.getElementById('frontmatter').value || '';
    const frontmatterState = document.getElementById('frontmatter-state');
    const lastAutosaved = document.getElementById('last-autosaved');
    frontmatterState.textContent = frontmatterValue === lastSavedFrontmatter ? 'Frontmatter matches the saved file.' : 'Frontmatter has unsaved changes.';
    if (!lastSavedAt) {
      lastAutosaved.textContent = 'No autosave yet.';
      return;
    }
    lastAutosaved.textContent = 'Last saved ' + lastSavedAt;
  }
  function setDirty(next) {
    dirty = next;
    updateDirtyIndicator();
    updatePropertiesStatus();
    if (dirty) {
      setStatus('Unsaved changes. Autosave starts after a short pause.', 'dirty');
    }
  }
  function scheduleAutosave() {
    if (autosaveTimer) { clearTimeout(autosaveTimer); }
    autosaveTimer = setTimeout(() => {
      if (dirty) {
        savePost(true);
      }
    }, 1500);
  }
  function markDirty() {
    setDirty(true);
    scheduleAutosave();
  }
  function setSelectedValues(select, values) {
    if (!select) { return; }
    const selected = new Set((values || []).map((value) => String(value).trim()).filter(Boolean));
    Array.from(select.options).forEach((option) => {
      option.selected = selected.has(option.value);
    });
    syncAuthorChips();
  }
  function selectedValues(select) {
    if (!select) { return []; }
    return Array.from(select.selectedOptions).map((option) => option.value).filter(Boolean);
  }
  function syncAuthorChips() {
    const selected = new Set(selectedValues(document.getElementById('fm-authors')));
    const visibleButtons = Array.from(document.querySelectorAll('[data-author-id]')).filter((button) => !button.hidden);
    document.querySelectorAll('[data-author-id]').forEach((button) => {
      button.classList.toggle('active', selected.has(button.dataset.authorId));
      button.setAttribute('aria-pressed', selected.has(button.dataset.authorId) ? 'true' : 'false');
      button.tabIndex = button.hidden ? -1 : 0;
    });
    visibleButtons.forEach((button, index) => {
      button.dataset.authorIndex = String(index);
    });
  }
  function filterAuthorChips() {
    const query = document.getElementById('fm-author-search').value.trim().toLowerCase();
    document.querySelectorAll('[data-author-id]').forEach((button) => {
      const haystack = (button.dataset.authorName || button.textContent || '').toLowerCase();
      button.hidden = !!query && !haystack.includes(query);
    });
    syncAuthorChips();
  }
  function slugifyDraftValue(value) {
    return String(value || '').trim().toLowerCase().replace(/\//g, '-').replace(/_/g, '-').replace(/\s+/g, '-').replace(/[^a-z0-9-]+/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '');
  }
  function basenameWithoutExt(value) {
    const cleaned = String(value || '').split('/').pop() || '';
    return cleaned.replace(/\.[^.]+$/, '');
  }
  function refreshDefaultHints() {
    const titleValue = document.getElementById('fm-title').value.trim();
    const pathValue = document.getElementById('post-path').value.trim();
    const slugField = document.getElementById('fm-slug');
    const modifiedField = document.getElementById('fm-modified');
    const defaultSlug = slugifyDraftValue(titleValue || basenameWithoutExt(pathValue) || 'untitled');
    slugField.placeholder = defaultSlug;
    document.getElementById('fm-slug-default').textContent = slugField.value.trim() ? 'Saved slug: ' + slugField.value.trim() : 'Default on save: ' + defaultSlug;
    const defaultModified = new Date().toISOString().replace(/\.\d{3}Z$/, 'Z');
    modifiedField.placeholder = defaultModified;
    document.getElementById('fm-modified-default').textContent = modifiedField.value.trim() ? '' : 'Default on save: ' + defaultModified;
  }
  function setFieldError(id, message) {
    const field = document.getElementById(id);
    const wrapper = document.getElementById(id + '-field');
    const error = document.getElementById(id + '-error');
    if (wrapper) {
      wrapper.dataset.invalid = message ? 'true' : 'false';
    }
    if (field) {
      field.setAttribute('aria-invalid', message ? 'true' : 'false');
    }
    if (error) {
      error.textContent = message || '';
    }
  }
  function populateTemplateTypeOptions() {
    const select = document.getElementById('fm-template-key');
    if (!select) { return; }
    const templates = (NEW_POST_CONTEXT && NEW_POST_CONTEXT.templates) || {};
    const aliases = (NEW_POST_CONTEXT && NEW_POST_CONTEXT.aliases) || {};
    const currentValue = select.value;
    select.innerHTML = '<option value="">Default (post)</option>';
    Object.keys(templates).sort().forEach((name) => {
      const option = document.createElement('option');
      option.value = name;
      option.textContent = name;
      select.appendChild(option);
    });
    Object.keys(aliases).sort().forEach((name) => {
      if (!(aliases[name] || []).includes(currentValue)) { return; }
      if (Array.from(select.options).some((option) => option.value === currentValue)) { return; }
      const option = document.createElement('option');
      option.value = currentValue;
      option.textContent = currentValue + ' -> ' + name;
      select.appendChild(option);
    });
    select.value = currentValue;
    if (currentValue && select.value !== currentValue) {
      const fallback = Object.entries(aliases).find(([, values]) => (values || []).includes(currentValue));
      if (fallback) {
        const option = document.createElement('option');
        option.value = currentValue;
        option.textContent = currentValue + ' -> ' + fallback[0];
        select.appendChild(option);
        select.value = currentValue;
      }
    }
  }
  function validateEditorFields() {
    const title = document.getElementById('fm-title');
    const slug = document.getElementById('fm-slug');
    const date = document.getElementById('fm-date');
    const modified = document.getElementById('fm-modified');
    const type = document.getElementById('fm-template-key');
    const titleError = title.value.trim() ? '' : 'Title is required.';
    title.setCustomValidity(titleError);
    setFieldError('fm-title', titleError);
    const slugError = !slug.value.trim() || /^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(slug.value.trim()) ? '' : 'Slug must use lowercase letters, numbers, and hyphens.';
    slug.setCustomValidity(slugError);
    setFieldError('fm-slug', slugError);
    [date, modified].forEach((field) => {
      const value = field.value.trim();
      const valid = !value || /^\d{4}-\d{2}-\d{2}$/.test(value) || /^\d{4}-\d{2}-\d{2}T/.test(value);
      const message = valid ? '' : 'Use YYYY-MM-DD or RFC3339 timestamp.';
      field.setCustomValidity(message);
      setFieldError(field.id, message);
    });
    const knownTypes = new Set(Array.from(type.options).map((option) => option.value).filter(Boolean));
    const typeError = !type.value.trim() || knownTypes.has(type.value.trim()) ? '' : 'Choose a known content type.';
    type.setCustomValidity(typeError);
    setFieldError('fm-template-key', typeError);
    refreshDefaultHints();
  }
  function maybeCenterTypewriter() {
    if (!typewriterMode) { return; }
    const ratio = body.selectionStart / Math.max(body.value.length, 1);
    const target = Math.max(0, body.scrollHeight * ratio - body.clientHeight * 0.42);
    body.scrollTop = target;
  }
  function syncNoteTitle() {
    if (!noteTitle) { return; }
    const title = document.getElementById('fm-title').value.trim();
    noteTitle.textContent = title || (existsInput.value === 'true' ? 'Untitled draft' : 'Start a new note');
  }
  function updateOutline() {
    const outline = document.getElementById('editor-outline');
    if (!outline) { return; }
    const headings = body.value.split(/\r?\n/).map((line) => line.match(/^(#{1,6})\s+(.+)$/)).filter(Boolean);
    outline.innerHTML = '';
    if (!headings.length) {
      outline.innerHTML = '<li class="muted">Add markdown headings to build an outline.</li>';
      return;
    }
    headings.forEach((match) => {
      const depth = match[1].length;
      const text = match[2].trim();
      const item = document.createElement('li');
      item.style.paddingLeft = ((depth - 1) * 0.8) + 'rem';
      item.innerHTML = '<a href="#">' + text + '</a>';
      item.querySelector('a').addEventListener('click', (event) => {
        event.preventDefault();
        body.focus();
        const index = body.value.indexOf(match[0]);
        if (index >= 0) {
          body.setSelectionRange(index, index + match[0].length);
          body.scrollTop = (body.scrollHeight / Math.max(body.value.length, 1)) * index;
        }
      });
      outline.appendChild(item);
    });
  }
  async function waitForBuild() {
    let sawBuilding = false;
    for (let i = 0; i < 45; i += 1) {
      await new Promise((resolve) => setTimeout(resolve, 500));
      const response = await fetch('/__admin/api/build-status');
      const status = await response.json();
      if (status.status === 'building') { sawBuilding = true; setStatus(status.message || 'Building preview...', 'building'); continue; }
      if (status.status === 'error') { setStatus(status.message || 'Build failed.', 'error'); return; }
      if (status.status === 'success' && sawBuilding) { setStatus('Build complete. Preview refreshed.', 'success'); await refreshBuiltPreview(); return; }
    }
    await refreshBuiltPreview();
    setStatus('Saved. Preview refreshed with current output.', 'success');
  }
  async function savePost(isAutoSave) {
    if (isSaving) { return; }
    validateEditorFields();
    const fieldIDs = ['fm-title', 'fm-slug', 'fm-date', 'fm-modified', 'fm-template-key'];
    const invalidField = fieldIDs.map((id) => document.getElementById(id)).find((field) => !field.checkValidity());
    if (invalidField) {
      invalidField.reportValidity();
      return;
    }
    isSaving = true;
    saveButton.disabled = true;
    setStatus(isAutoSave ? 'Autosaving source file...' : 'Saving source file...', 'building');
    const response = await fetch('/__admin/api/post', {
      method: existsInput.value === 'true' ? 'PUT' : 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({ path: document.getElementById('post-path').value, frontmatter: document.getElementById('frontmatter').value, body: body.value, base_hash: baseHash.value })
    });
    if (!response.ok) { setStatus(await response.text(), 'error'); saveButton.disabled = false; isSaving = false; return; }
    const result = await response.json();
    baseHash.value = result.new_hash || '';
    existsInput.value = 'true';
    document.getElementById('post-path').value = result.path || document.getElementById('post-path').value;
    if (result.preview_url) { previewLink.href = result.preview_url; }
    saveButton.textContent = 'Save changes';
    lastSavedFrontmatter = document.getElementById('frontmatter').value || '';
    lastSavedAt = new Date().toLocaleTimeString([], {hour: 'numeric', minute: '2-digit', second: '2-digit'});
    setDirty(false);
    if (previewMode === 'built') {
      setStatus(isAutoSave ? 'Autosaved. Waiting for preview build...' : 'Saved. Waiting for preview build...', 'building');
      await waitForBuild();
    } else {
      setStatus(isAutoSave ? 'Autosaved. Live preview shows current draft.' : 'Saved source file. Live preview shows current draft.', 'success');
      await refreshLivePreview();
    }
    saveButton.disabled = false;
    isSaving = false;
  }
  enableTabs('frontmatter-tabs');
  applyWorkspaceState();
  applyEditorModes();
  populateTemplateTypeOptions();
  loadFrontmatterForm().catch((error) => setStatus(error.message, 'error'));
  ['fm-title', 'fm-slug', 'fm-date', 'fm-modified', 'fm-description', 'fm-template-key'].forEach((id) => document.getElementById(id).addEventListener('input', () => { syncFrontmatterFromForm(); markDirty(); }));
  document.getElementById('fm-title').addEventListener('input', syncNoteTitle);
  document.getElementById('post-path').addEventListener('input', refreshDefaultHints);
  document.getElementById('fm-slug-reset').addEventListener('click', () => {
    document.getElementById('fm-slug').value = '';
    validateEditorFields();
    refreshDefaultHints();
    syncFrontmatterFromForm();
    markDirty();
  });
  document.getElementById('fm-published').addEventListener('change', () => { syncFrontmatterFromForm(); markDirty(); });
  document.getElementById('fm-authors').addEventListener('change', () => { syncFrontmatterFromForm(); markDirty(); });
  document.getElementById('fm-author-search').addEventListener('input', filterAuthorChips);
  document.querySelectorAll('[data-author-id]').forEach((button) => {
    button.addEventListener('click', () => {
      const select = document.getElementById('fm-authors');
      const option = Array.from(select.options).find((candidate) => candidate.value === button.dataset.authorId);
      if (!option) { return; }
      option.selected = !option.selected;
      syncAuthorChips();
      syncFrontmatterFromForm();
      markDirty();
    });
    button.addEventListener('keydown', (event) => {
      const visibleButtons = Array.from(document.querySelectorAll('[data-author-id]')).filter((candidate) => !candidate.hidden);
      const currentIndex = visibleButtons.indexOf(button);
      if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
        event.preventDefault();
        const next = visibleButtons[currentIndex + 1] || visibleButtons[0];
        if (next) { next.focus(); }
      }
      if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') {
        event.preventDefault();
        const prev = visibleButtons[currentIndex - 1] || visibleButtons[visibleButtons.length - 1];
        if (prev) { prev.focus(); }
      }
      if (event.key === ' ' || event.key === 'Enter') {
        event.preventDefault();
        button.click();
      }
    });
  });
  document.getElementById('fm-add-extra').addEventListener('click', () => { addExtraField('', '', 'string'); syncFrontmatterFromForm(); markDirty(); });
  document.getElementById('fm-tag-add').addEventListener('click', () => { addTag(document.getElementById('fm-tag-input').value); document.getElementById('fm-tag-input').value = ''; markDirty(); });
  document.getElementById('fm-tag-input').addEventListener('keydown', (event) => {
    if (event.key === 'Enter' || event.key === ',') {
      event.preventDefault();
      addTag(event.currentTarget.value);
      event.currentTarget.value = '';
      markDirty();
    } else if (event.key === 'Backspace' && !event.currentTarget.value && tagState.length) {
      tagState.pop();
      syncTagsInput();
      renderTagPills();
      syncFrontmatterFromForm();
      markDirty();
    }
  });
  document.getElementById('frontmatter').addEventListener('input', () => { loadFrontmatterForm().catch((error) => setStatus(error.message, 'error')); queueLivePreview(); markDirty(); });
  body.addEventListener('input', () => { updateWordCount(); updateOutline(); queueLivePreview(); refreshSlashCommandState(); maybeCenterTypewriter(); markDirty(); });
  body.addEventListener('click', () => { if (!commandState.fromKeyboard) { refreshSlashCommandState(); } });
  liveToggle.addEventListener('click', async () => { setPreviewMode('live'); await refreshLivePreview(); });
  builtToggle.addEventListener('click', async () => { setPreviewMode('built'); await refreshBuiltPreview(); });
  propertiesButton.addEventListener('click', () => { centerCollapsed = !centerCollapsed; applyWorkspaceState(); });
  previewButton.addEventListener('click', () => { rightCollapsed = !rightCollapsed; applyWorkspaceState(); if (!rightCollapsed) { setPreviewMode('live'); refreshLivePreview().catch((error) => setStatus(error.message, 'error')); } });
  document.getElementById('close-preview-panel').addEventListener('click', () => { rightCollapsed = true; applyWorkspaceState(); });
  document.getElementById('close-properties-panel').addEventListener('click', () => { centerCollapsed = true; applyWorkspaceState(); });
  commandButton.addEventListener('click', () => {
    if (commandState.open) {
      closeCommandPalette();
      body.focus();
      return;
    }
    openCommandPalette(true);
    body.focus();
  });
  focusButton.addEventListener('click', () => { focusMode = !focusMode; applyEditorModes(); body.focus(); });
  typewriterButton.addEventListener('click', () => { typewriterMode = !typewriterMode; applyEditorModes(); maybeCenterTypewriter(); body.focus(); });
  fullscreenButton.addEventListener('click', () => { focusMode = !focusMode; applyEditorModes(); body.focus(); });
  if (document.getElementById('new-template')) {
    const templateSelect = document.getElementById('new-template');
    const dirSelect = document.getElementById('new-directory');
    const templates = (NEW_POST_CONTEXT && NEW_POST_CONTEXT.templates) || {};
    Object.keys(templates).sort().forEach((name) => {
      const option = document.createElement('option');
      option.value = name;
      option.textContent = (templates[name].label || name) + ' -> ' + (templates[name].directory || '') + ' (' + (templates[name].source || 'builtin') + ')';
      templateSelect.appendChild(option);
    });
    ((NEW_POST_CONTEXT && NEW_POST_CONTEXT.directories) || []).forEach((dir) => {
      const option = document.createElement('option');
      option.value = dir;
      option.textContent = dir;
      dirSelect.appendChild(option);
    });
    const customDirOption = document.createElement('option');
    customDirOption.value = '__custom__';
    customDirOption.textContent = 'Custom...';
    dirSelect.appendChild(customDirOption);
    templateSelect.value = 'post';
    dirSelect.value = (templates.post && templates.post.directory) || ((NEW_POST_CONTEXT && NEW_POST_CONTEXT.directories && NEW_POST_CONTEXT.directories[0]) || 'pages/post');
    renderNewAuthorOptions();
    populateNewTemplateFields(templateSelect.value);
    templateSelect.addEventListener('change', () => {
      const templateDef = templates[templateSelect.value] || {};
      if (templateDef.directory) {
        ensureSelectOption(dirSelect, templateDef.directory);
        dirSelect.value = templateDef.directory;
      }
      populateNewTemplateFields(templateSelect.value);
    });
    document.getElementById('new-tag-add').addEventListener('click', () => { addNewTag(document.getElementById('new-tag-input').value); document.getElementById('new-tag-input').value = ''; });
    document.getElementById('new-tag-input').addEventListener('keydown', (event) => {
      if (event.key === 'Enter' || event.key === ',') {
        event.preventDefault();
        addNewTag(event.currentTarget.value);
        event.currentTarget.value = '';
      }
    });
    document.getElementById('generate-scaffold').addEventListener('click', () => { generateNewPostScaffold().catch((error) => setStatus(error.message, 'error')); });
  }
  saveButton.addEventListener('click', () => savePost(false));
  document.addEventListener('click', (event) => {
    if (!commandPalette.contains(event.target) && event.target !== commandButton && !commandButton.contains(event.target) && event.target !== body && !body.contains(event.target)) {
      closeCommandPalette();
    }
  });
  document.addEventListener('keydown', (event) => {
    if ((event.ctrlKey || event.metaKey) && event.key === 's') { event.preventDefault(); savePost(false); return; }
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') { event.preventDefault(); openCommandPalette(true); body.focus(); return; }
    if ((event.ctrlKey || event.metaKey) && event.key === '.') { event.preventDefault(); focusMode = !focusMode; applyEditorModes(); body.focus(); return; }
    if (event.key === 'Escape' && commandState.open) { closeCommandPalette(); return; }
    if (commandState.open) {
      const commands = filteredCommands();
      if (event.key === 'ArrowDown') { event.preventDefault(); commandState.selected = Math.min(commandState.selected + 1, Math.max(0, commands.length - 1)); renderCommandPalette(); return; }
      if (event.key === 'ArrowUp') { event.preventDefault(); commandState.selected = Math.max(commandState.selected - 1, 0); renderCommandPalette(); return; }
      if (event.key === 'Enter' && commands.length) { event.preventDefault(); applySlashCommand(commands[commandState.selected]); return; }
    }
  });
  setPreviewMode('live');
  updateDirtyIndicator();
  updatePropertiesStatus();
  updateWordCount();
  syncNoteTitle();
  refreshDefaultHints();
  syncAuthorChips();
  filterAuthorChips();
  updateOutline();
  renderCommandPalette();
  refreshLivePreview().catch((error) => setStatus(error.message, 'error'));
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && !centerCollapsed && !commandState.open) { centerCollapsed = true; applyWorkspaceState(); }
    if (event.key === 'Escape' && !rightCollapsed && !commandState.open) { rightCollapsed = true; applyWorkspaceState(); }
  });
  window.addEventListener('beforeunload', (event) => {
    if (!dirty) { return; }
    event.preventDefault();
    event.returnValue = '';
  });
})();
</script>` + pageFootTemplate + `{{end}}`

const settingsTemplate = `{{define "settings"}}` + pageHeadTemplate + `
<main class="shell">
  <header class="card nav">
    <a class="brand" href="/__admin/dashboard"><strong>Markata local admin</strong><span>Site settings</span></a>
    <nav><a href="/__admin/dashboard">Posts</a><a href="/__admin/settings">Settings</a><a href="/__admin/logout">Logout</a></nav>
  </header>
  <section class="card hero"><h1>Settings editor</h1><p>Use the object form for common site settings and fall back to raw config when you need exact control.</p></section>
  <section class="card panel stack">
    <div class="toolbar"><div><strong>Config file</strong><div class="muted mono">{{.Settings.Path}}</div></div><button id="save-settings" class="btn btn-primary" type="button">Save settings</button></div>
    <div id="settings-status" class="status muted">Ready.</div>
    <div class="workspace-two">
      <aside class="pane pane-sticky stack">
        <section class="meta-card">
          <div class="tabs" data-tabs="settings-tabs">
            <button class="tab active" type="button" data-tab="settings-form">Settings</button>
            <button class="tab" type="button" data-tab="settings-raw">Raw config</button>
          </div>
          <div id="settings-form" class="tab-panel stack">
            <div class="field-grid">
              <div class="field"><label for="cfg-title">Site title</label><input id="cfg-title"></div>
              <div class="field"><label for="cfg-author">Author</label><input id="cfg-author"></div>
              <div class="field field-span-2"><label for="cfg-url">Site URL</label><input id="cfg-url" class="mono"></div>
              <div class="field field-span-2"><label for="cfg-description">Description</label><textarea id="cfg-description" style="min-height: 7rem;"></textarea></div>
            </div>
            <div class="meta-section">
              <div class="pane-head"><h3>Build</h3><span class="pane-subtitle">Core paths</span></div>
              <div class="field-grid">
                <div class="field"><label for="cfg-output-dir">Output dir</label><input id="cfg-output-dir" class="mono"></div>
                <div class="field"><label for="cfg-templates-dir">Templates dir</label><input id="cfg-templates-dir" class="mono"></div>
                <div class="field field-span-2"><label for="cfg-assets-dir">Assets dir</label><input id="cfg-assets-dir" class="mono"></div>
              </div>
            </div>
            <div class="meta-section">
              <div class="pane-head"><h3>Theme</h3><span class="pane-subtitle">Palette-driven look</span></div>
              <div class="field-grid">
                <div class="field"><label for="cfg-theme-palette">Base palette</label><input id="cfg-theme-palette" list="known-palettes"></div>
                <div class="field"><label for="cfg-theme-fallback">Fallback mode</label><select id="cfg-theme-fallback"><option value="">Default</option><option value="light">light</option><option value="dark">dark</option></select></div>
                <div class="field"><label for="cfg-theme-light">Light palette</label><input id="cfg-theme-light" list="known-palettes"></div>
                <div class="field"><label for="cfg-theme-dark">Dark palette</label><input id="cfg-theme-dark" list="known-palettes"></div>
              </div>
              <datalist id="known-palettes">{{range .KnownPalettes}}<option value="{{.}}"></option>{{end}}</datalist>
              <p class="muted" style="margin:0;">Theme palette values are validated against the built-in palette set.</p>
            </div>
            <div class="meta-section">
              <div class="pane-head"><h3>Search</h3><span class="pane-subtitle">Built-in search UI</span></div>
              <div class="field-grid">
                <div class="field"><label><input id="cfg-search-enabled" type="checkbox" style="width:auto; margin-right:0.5rem;"> Search enabled</label></div>
                <div class="field"><label for="cfg-search-position">Search position</label><select id="cfg-search-position"><option value="">Default</option><option value="navbar">navbar</option><option value="sidebar">sidebar</option><option value="footer">footer</option><option value="custom">custom</option></select></div>
                <div class="field field-span-2"><label for="cfg-search-placeholder">Search placeholder</label><input id="cfg-search-placeholder"></div>
                <div class="field"><label><input id="cfg-pagefind-auto-install" type="checkbox" style="width:auto; margin-right:0.5rem;"> Auto-install Pagefind</label></div>
                <div class="field"><label for="cfg-pagefind-version">Pagefind version</label><input id="cfg-pagefind-version" placeholder="latest"></div>
                <div class="field field-span-2"><label for="cfg-pagefind-bundle-dir">Pagefind bundle dir</label><input id="cfg-pagefind-bundle-dir" class="mono" placeholder="_pagefind"></div>
              </div>
            </div>
            <div class="meta-section">
              <div class="pane-head"><h3>Theme Switcher</h3><span class="pane-subtitle">Palette switcher behavior</span></div>
              <div class="field-grid">
                <div class="field"><label><input id="cfg-theme-switcher-enabled" type="checkbox" style="width:auto; margin-right:0.5rem;"> Switcher enabled</label></div>
                <div class="field"><label for="cfg-theme-switcher-position">Switcher position</label><select id="cfg-theme-switcher-position"><option value="">Default</option><option value="header">header</option><option value="footer">footer</option></select></div>
                <div class="field"><label><input id="cfg-theme-mode-toggle-enabled" type="checkbox" style="width:auto; margin-right:0.5rem;"> Show mode toggle</label></div>
                <div class="field"><label><input id="cfg-theme-include-all" type="checkbox" style="width:auto; margin-right:0.5rem;"> Include all palettes</label></div>
              </div>
            </div>
            <div class="meta-section">
              <div class="pane-head"><h3>Typography</h3><span class="pane-subtitle">Theme font controls</span></div>
              <div class="field-grid">
                <div class="field field-span-2"><label for="cfg-font-family">Body font family</label><input id="cfg-font-family"></div>
                <div class="field"><label for="cfg-font-heading-family">Heading family</label><input id="cfg-font-heading-family"></div>
                <div class="field"><label for="cfg-font-code-family">Code family</label><input id="cfg-font-code-family"></div>
                <div class="field"><label for="cfg-font-size">Base size</label><input id="cfg-font-size" placeholder="16px"></div>
                <div class="field"><label for="cfg-font-line-height">Line height</label><input id="cfg-font-line-height" placeholder="1.6"></div>
              </div>
            </div>
          </div>
          <div id="settings-raw" class="tab-panel" hidden>
            <div class="code-panel"><label for="settings-content">Config</label><textarea id="settings-content" class="mono" style="min-height: 56vh;">{{.Settings.Content}}</textarea></div>
          </div>
        </section>
      </aside>
      <section class="pane stack">
        <section class="meta-card">
          <div class="pane-head"><h3>How This Works</h3><span class="pane-subtitle">Safe defaults with escape hatch</span></div>
          <div class="stack muted">
            <p style="margin:0;">The form editor patches the raw config text for the common settings people change most often.</p>
            <p style="margin:0;">Anything custom stays available in the raw config tab, so you do not lose advanced configuration.</p>
            <p style="margin:0;">Save writes the active config file, then the dev server rebuilds using the normal config path.</p>
          </div>
        </section>
      </section>
    </div>
    <input id="settings-base-hash" type="hidden" value="{{.Settings.Hash}}">
  </section>
</main>
<script>
(() => {
  function enableTabs(groupName, panelIds) {
    const root = document.querySelector('[data-tabs="' + groupName + '"]');
    if (!root) { return; }
    root.querySelectorAll('[data-tab]').forEach((button) => {
      button.addEventListener('click', () => {
        const target = button.getAttribute('data-tab');
        root.querySelectorAll('[data-tab]').forEach((tabButton) => tabButton.classList.toggle('active', tabButton === button));
        panelIds.forEach((panelId) => { document.getElementById(panelId).hidden = panelId !== target; });
      });
    });
  }
  async function postJSON(url, payload) {
    const response = await fetch(url, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    return response.json();
  }
  async function loadSettingsForm() {
    const parsed = await postJSON('/__admin/api/settings/parse', { content: document.getElementById('settings-content').value });
    document.getElementById('cfg-title').value = parsed.title || '';
    document.getElementById('cfg-author').value = parsed.author || '';
    document.getElementById('cfg-url').value = parsed.url || '';
    document.getElementById('cfg-description').value = parsed.description || '';
    document.getElementById('cfg-output-dir').value = parsed.output_dir || '';
    document.getElementById('cfg-templates-dir').value = parsed.templates_dir || '';
    document.getElementById('cfg-assets-dir').value = parsed.assets_dir || '';
    document.getElementById('cfg-theme-palette').value = parsed.theme_palette || '';
    document.getElementById('cfg-theme-light').value = parsed.theme_light || '';
    document.getElementById('cfg-theme-dark').value = parsed.theme_dark || '';
    document.getElementById('cfg-theme-fallback').value = parsed.theme_mode || '';
    document.getElementById('cfg-search-enabled').checked = !!parsed.search_enabled;
    document.getElementById('cfg-search-position').value = parsed.search_position || '';
    document.getElementById('cfg-search-placeholder').value = parsed.search_placeholder || '';
    document.getElementById('cfg-pagefind-bundle-dir').value = parsed.pagefind_bundle_dir || '';
    document.getElementById('cfg-pagefind-version').value = parsed.pagefind_version || '';
    document.getElementById('cfg-pagefind-auto-install').checked = !!parsed.pagefind_auto_install;
    document.getElementById('cfg-theme-switcher-enabled').checked = !!parsed.theme_switcher_enabled;
    document.getElementById('cfg-theme-switcher-position').value = parsed.theme_switcher_position || '';
    document.getElementById('cfg-theme-mode-toggle-enabled').checked = !!parsed.theme_mode_toggle_enabled;
    document.getElementById('cfg-theme-include-all').checked = !!parsed.theme_include_all;
    document.getElementById('cfg-font-family').value = parsed.font_family || '';
    document.getElementById('cfg-font-heading-family').value = parsed.font_heading_family || '';
    document.getElementById('cfg-font-code-family').value = parsed.font_code_family || '';
    document.getElementById('cfg-font-size').value = parsed.font_size || '';
    document.getElementById('cfg-font-line-height').value = parsed.font_line_height || '';
  }
  async function syncSettingsRawFromForm() {
    try {
      const result = await postJSON('/__admin/api/settings/render', {
        content: document.getElementById('settings-content').value,
        form: {
          title: document.getElementById('cfg-title').value,
          author: document.getElementById('cfg-author').value,
          url: document.getElementById('cfg-url').value,
          description: document.getElementById('cfg-description').value,
          output_dir: document.getElementById('cfg-output-dir').value,
          templates_dir: document.getElementById('cfg-templates-dir').value,
          assets_dir: document.getElementById('cfg-assets-dir').value,
          theme_palette: document.getElementById('cfg-theme-palette').value,
          theme_light: document.getElementById('cfg-theme-light').value,
          theme_dark: document.getElementById('cfg-theme-dark').value,
          theme_mode: document.getElementById('cfg-theme-fallback').value,
          search_enabled: document.getElementById('cfg-search-enabled').checked,
          search_position: document.getElementById('cfg-search-position').value,
          search_placeholder: document.getElementById('cfg-search-placeholder').value,
          pagefind_bundle_dir: document.getElementById('cfg-pagefind-bundle-dir').value,
          pagefind_version: document.getElementById('cfg-pagefind-version').value,
          pagefind_auto_install: document.getElementById('cfg-pagefind-auto-install').checked,
          theme_switcher_enabled: document.getElementById('cfg-theme-switcher-enabled').checked,
          theme_switcher_position: document.getElementById('cfg-theme-switcher-position').value,
          theme_mode_toggle_enabled: document.getElementById('cfg-theme-mode-toggle-enabled').checked,
          theme_include_all: document.getElementById('cfg-theme-include-all').checked,
          font_family: document.getElementById('cfg-font-family').value,
          font_heading_family: document.getElementById('cfg-font-heading-family').value,
          font_code_family: document.getElementById('cfg-font-code-family').value,
          font_size: document.getElementById('cfg-font-size').value,
          font_line_height: document.getElementById('cfg-font-line-height').value
        }
      });
      document.getElementById('settings-content').value = result.content || '';
    } catch (error) {
      setStatus(error.message, 'error');
    }
  }
  const saveButton = document.getElementById('save-settings');
  const statusEl = document.getElementById('settings-status');
  function setStatus(message, state) { statusEl.textContent = message; statusEl.dataset.state = state || ''; }
  async function waitForBuild() {
    let sawBuilding = false;
    for (let i = 0; i < 45; i += 1) {
      await new Promise((resolve) => setTimeout(resolve, 500));
      const response = await fetch('/__admin/api/build-status');
      const status = await response.json();
      if (status.status === 'building') { sawBuilding = true; setStatus(status.message || 'Building site...', 'building'); continue; }
      if (status.status === 'error') { setStatus(status.message || 'Build failed.', 'error'); return; }
      if (status.status === 'success' && sawBuilding) { setStatus('Settings saved and site rebuilt.', 'success'); return; }
    }
    setStatus('Settings saved.', 'success');
  }
  async function saveSettings() {
    saveButton.disabled = true;
    setStatus('Saving config file...', 'building');
    const response = await fetch('/__admin/api/settings', {
      method: 'PUT',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({ content: document.getElementById('settings-content').value, base_hash: document.getElementById('settings-base-hash').value })
    });
    if (!response.ok) { setStatus(await response.text(), 'error'); saveButton.disabled = false; return; }
    const result = await response.json();
    document.getElementById('settings-base-hash').value = result.new_hash || '';
    setStatus('Saved. Waiting for rebuild...', 'building');
    await waitForBuild();
    saveButton.disabled = false;
  }
  enableTabs('settings-tabs', ['settings-form', 'settings-raw']);
  loadSettingsForm().catch((error) => setStatus(error.message, 'error'));
  ['cfg-title', 'cfg-author', 'cfg-url', 'cfg-description', 'cfg-output-dir', 'cfg-templates-dir', 'cfg-assets-dir', 'cfg-theme-palette', 'cfg-theme-light', 'cfg-theme-dark', 'cfg-theme-fallback', 'cfg-search-position', 'cfg-search-placeholder', 'cfg-pagefind-bundle-dir', 'cfg-pagefind-version', 'cfg-theme-switcher-position', 'cfg-font-family', 'cfg-font-heading-family', 'cfg-font-code-family', 'cfg-font-size', 'cfg-font-line-height'].forEach((id) => {
    document.getElementById(id).addEventListener('input', syncSettingsRawFromForm);
    document.getElementById(id).addEventListener('change', syncSettingsRawFromForm);
  });
  ['cfg-search-enabled', 'cfg-pagefind-auto-install', 'cfg-theme-switcher-enabled', 'cfg-theme-mode-toggle-enabled', 'cfg-theme-include-all'].forEach((id) => document.getElementById(id).addEventListener('change', syncSettingsRawFromForm));
  document.getElementById('settings-content').addEventListener('input', () => { loadSettingsForm().catch((error) => setStatus(error.message, 'error')); });
  saveButton.addEventListener('click', saveSettings);
  document.addEventListener('keydown', (event) => { if ((event.ctrlKey || event.metaKey) && event.key === 's') { event.preventDefault(); saveSettings(); } });
})();
</script>` + pageFootTemplate + `{{end}}`
