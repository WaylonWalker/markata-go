package cmd

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/WaylonWalker/markata-go/pkg/config"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

// Settings sidebar endpoints. They exist only under `markata-go serve`;
// production builds never reference them.
const (
	serveSettingsEndpoint   = "/__markata/settings"
	serveSettingsScriptPath = "/__markata/settings.js"
	serveSettingsPreview    = "/__markata/settings/preview"
	maxSettingsChanges      = 200
)

//go:embed devui/settings.js
var serveSettingsScript []byte

type settingsFieldJSON struct {
	config.SettingField
	// Source is the config file that currently defines the setting.
	Source string `json:"source,omitempty"`
	// Target is the config file a change would be written to.
	Target string `json:"target"`
	// Env names an environment variable that overrides the setting.
	Env string `json:"env,omitempty"`
}

type settingsResponse struct {
	Fields []settingsFieldJSON `json:"fields,omitempty"`
	// Preview lists unsaved settings the dev server is currently applying.
	Preview  []settingsChange  `json:"preview"`
	Files    []string          `json:"files,omitempty"`
	Changed  []settingsChanged `json:"changed,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
	Error    string            `json:"error,omitempty"`
}

type settingsChanged struct {
	Key    string `json:"key"`
	Target string `json:"target"`
}

type settingsChange struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type settingsRequest struct {
	Changes []settingsChange `json:"changes"`
}

// Preview state: unsaved settings merged over the config files for every
// rebuild while `markata-go serve` runs. They live only in memory.
var (
	servePreviewMu       sync.RWMutex
	servePreviewSettings []plannedSetting
)

// servePreviewOverlay returns the raw overlay for the current preview, or nil.
func servePreviewOverlay() map[string]any {
	servePreviewMu.RLock()
	defer servePreviewMu.RUnlock()
	return previewOverlay(servePreviewSettings)
}

func previewOverlay(planned []plannedSetting) map[string]any {
	if len(planned) == 0 {
		return nil
	}
	settings := make([]config.BakeSetting, len(planned))
	for i, p := range planned {
		settings[i] = config.BakeSetting{Path: p.path, Value: p.value}
	}
	return config.SettingsOverlay(settings)
}

func servePreviewChanges() []settingsChange {
	servePreviewMu.RLock()
	defer servePreviewMu.RUnlock()
	out := make([]settingsChange, len(servePreviewSettings))
	for i, p := range servePreviewSettings {
		out[i] = settingsChange{Key: p.key, Value: p.value}
	}
	return out
}

func setServePreview(planned []plannedSetting) {
	sort.Slice(planned, func(i, j int) bool { return planned[i].key < planned[j].key })
	servePreviewMu.Lock()
	servePreviewSettings = planned
	servePreviewMu.Unlock()
}

// dropServePreviewKeys removes baked keys from the preview.
func dropServePreviewKeys(keys map[string]bool) {
	servePreviewMu.Lock()
	defer servePreviewMu.Unlock()
	kept := servePreviewSettings[:0:0]
	for _, p := range servePreviewSettings {
		if !keys[p.key] {
			kept = append(kept, p)
		}
	}
	servePreviewSettings = kept
}

// settingsState is the effective config plus the files that compose it.
type settingsState struct {
	cfg     *models.Config
	sources []string
	raws    []map[string]any
}

func loadSettingsState() (*settingsState, error) {
	cfg, sources, err := resolveServeConfigSources(cfgFile)
	if err != nil {
		return nil, err
	}
	state := &settingsState{cfg: cfg, sources: sources}
	for _, path := range sources {
		raw, err := config.LoadRawConfigFile(path)
		if err != nil {
			return nil, err
		}
		state.raws = append(state.raws, raw)
	}
	return state, nil
}

// source returns the highest-precedence file that defines path, or "".
func (s *settingsState) source(path []string) string {
	found := ""
	for i, raw := range s.raws {
		if config.SettingDefinedDepth(raw, path) == len(path) {
			found = s.sources[i]
		}
	}
	return found
}

// target picks the file that receives a change to path: the file defining
// the most specific part of the key path (later files win ties because they
// override earlier ones), else the root config. A config-less site gets a
// new markata-go.toml in the working directory.
func (s *settingsState) target(path []string) (string, error) {
	if len(s.sources) == 0 {
		return filepath.Abs("markata-go.toml")
	}
	best, bestDepth := s.sources[0], 0
	for i, raw := range s.raws {
		if depth := config.SettingDefinedDepth(raw, path); depth > 0 && depth >= bestDepth {
			best, bestDepth = s.sources[i], depth
		}
	}
	return best, nil
}

func settingEnvName(key string) string {
	return "MARKATA_GO_" + strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
}

func settingEnvOverride(key string) string {
	name := settingEnvName(key)
	if _, ok := os.LookupEnv(name); ok {
		return name
	}
	return ""
}

func (s *settingsState) describe() (settingsResponse, error) {
	resp := settingsResponse{Preview: servePreviewChanges()}
	for _, path := range s.sources {
		resp.Files = append(resp.Files, displayConfigPath(path))
	}
	for _, field := range config.Settings(s.cfg) {
		path := field.SettingPath()
		target, err := s.target(path)
		if err != nil {
			return resp, err
		}
		entry := settingsFieldJSON{SettingField: field, Target: displayConfigPath(target), Env: settingEnvOverride(field.Key)}
		if src := s.source(path); src != "" {
			entry.Source = displayConfigPath(src)
		}
		resp.Fields = append(resp.Fields, entry)
	}
	return resp, nil
}

func writeSettingsJSON(w http.ResponseWriter, status int, resp settingsResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	//nolint:errcheck // Best effort write to HTTP response
	json.NewEncoder(w).Encode(resp)
}

func handleSettingsScript(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	//nolint:errcheck // Best effort write to HTTP response
	w.Write(serveSettingsScript)
}

// handleSettings serves GET (describe every setting with its value, source,
// and target file) and POST (bake changes into the config files, verify that
// the reloaded config reflects them, and queue a full rebuild).
func handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Allow", "GET, POST")
		writeSettingsJSON(w, http.StatusMethodNotAllowed, settingsResponse{Error: "method not allowed"})
		return
	}
	if !sameOriginRequest(r) {
		writeSettingsJSON(w, http.StatusForbidden, settingsResponse{Error: "cross-origin requests are not allowed"})
		return
	}

	themeBakeMu.Lock()
	defer themeBakeMu.Unlock()

	state, err := loadSettingsState()
	if err != nil {
		writeSettingsJSON(w, http.StatusInternalServerError, settingsResponse{Error: err.Error()})
		return
	}
	if r.Method == http.MethodGet {
		resp, err := state.describe()
		if err != nil {
			writeSettingsJSON(w, http.StatusInternalServerError, settingsResponse{Error: err.Error()})
			return
		}
		writeSettingsJSON(w, http.StatusOK, resp)
		return
	}

	req, ok := decodeSettingsRequest(w, r)
	if !ok {
		return
	}
	resp, status := applySettingsChanges(state, req.Changes)
	writeSettingsJSON(w, status, resp)
}

func decodeSettingsRequest(w http.ResponseWriter, r *http.Request) (settingsRequest, bool) {
	var req settingsRequest
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeSettingsJSON(w, http.StatusUnsupportedMediaType, settingsResponse{Error: "Content-Type must be application/json"})
		return req, false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeSettingsJSON(w, http.StatusBadRequest, settingsResponse{Error: "invalid request: " + err.Error()})
		return req, false
	}
	return req, true
}

// handleSettingsPreview replaces the in-memory preview with the posted
// changes (an empty list resets it). The config files are not touched: the
// preview is checked like a bake (it must load, add no validation errors, and
// take effect), then a full rebuild applies it and live reload shows it.
func handleSettingsPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeSettingsJSON(w, http.StatusMethodNotAllowed, settingsResponse{Error: "method not allowed"})
		return
	}
	if !sameOriginRequest(r) {
		writeSettingsJSON(w, http.StatusForbidden, settingsResponse{Error: "cross-origin requests are not allowed"})
		return
	}
	req, ok := decodeSettingsRequest(w, r)
	if !ok {
		return
	}

	themeBakeMu.Lock()
	defer themeBakeMu.Unlock()

	state, err := loadSettingsState()
	if err != nil {
		writeSettingsJSON(w, http.StatusInternalServerError, settingsResponse{Error: err.Error()})
		return
	}
	resp, status := previewSettingsChanges(state, req.Changes)
	writeSettingsJSON(w, status, resp)
}

func previewSettingsChanges(state *settingsState, changes []settingsChange) (settingsResponse, int) {
	var resp settingsResponse
	planned, err := planSettingsChanges(state, changes, true)
	if err != nil {
		resp.Error = err.Error()
		resp.Preview = servePreviewChanges()
		return resp, http.StatusBadRequest
	}
	if len(planned) > 0 {
		previewed, _, _, err := loadManagerConfigWith(cfgFile, previewOverlay(planned))
		if err == nil {
			err = checkSettingsTookEffect(validationErrorSet(state.cfg), previewed, planned, &resp)
		}
		if err != nil {
			resp.Error = err.Error()
			resp.Preview = servePreviewChanges()
			return resp, http.StatusUnprocessableEntity
		}
	}
	setServePreview(planned)
	resp.Preview = servePreviewChanges()
	if len(planned) == 0 {
		infof("Settings preview reset")
	} else {
		keys := make([]string, len(planned))
		for i, p := range planned {
			keys[i] = p.key
		}
		infof("Previewing unsaved settings (%s)", strings.Join(keys, ", "))
	}
	if serveRequestFullRebuild != nil {
		serveRequestFullRebuild()
	}
	return resp, http.StatusOK
}

// checkSettingsTookEffect fails when cfg has validation errors not in before,
// or a planned value did not load back (env overrides only warn).
func checkSettingsTookEffect(before map[string]bool, cfg *models.Config, planned []plannedSetting, resp *settingsResponse) error {
	for msg := range validationErrorSet(cfg) {
		if !before[msg] {
			return errors.New(msg)
		}
	}
	for _, p := range planned {
		if p.env != "" {
			resp.Warnings = append(resp.Warnings, p.env+" is set and overrides "+p.key)
			continue
		}
		got, _ := config.SettingValue(cfg, p.key)
		if !config.SettingsEqual(got, p.value) {
			return fmt.Errorf("%s did not take effect (the loaded config reports %v)", p.key, got)
		}
	}
	return nil
}

type plannedSetting struct {
	key    string
	path   []string
	value  any
	target string
	env    string
}

func planSettingsChanges(state *settingsState, changes []settingsChange, allowEmpty bool) ([]plannedSetting, error) {
	if len(changes) == 0 && !allowEmpty {
		return nil, errors.New("no changes to bake")
	}
	if len(changes) > maxSettingsChanges {
		return nil, fmt.Errorf("too many changes (max %d)", maxSettingsChanges)
	}
	seen := map[string]bool{}
	planned := make([]plannedSetting, 0, len(changes))
	for _, change := range changes {
		field, ok := config.LookupSettingIn(state.cfg, change.Key)
		if !ok {
			return nil, fmt.Errorf("%w: %q", config.ErrUnknownSetting, change.Key)
		}
		if seen[change.Key] {
			return nil, fmt.Errorf("duplicate change for %s", change.Key)
		}
		seen[change.Key] = true
		value, err := config.CoerceSettingValue(field, change.Value)
		if err != nil {
			return nil, err
		}
		path := field.SettingPath()
		target, err := state.target(path)
		if err != nil {
			return nil, err
		}
		planned = append(planned, plannedSetting{
			key: change.Key, path: path, value: value, target: target, env: settingEnvOverride(change.Key),
		})
	}
	return planned, nil
}

type fileBackup struct {
	path    string
	data    []byte
	existed bool
	mode    os.FileMode
}

func backupFile(path string) (fileBackup, error) {
	b := fileBackup{path: path}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return b, nil
	}
	if err != nil {
		return b, err
	}
	b.data, err = os.ReadFile(path)
	if err != nil {
		return b, err
	}
	b.existed, b.mode = true, info.Mode().Perm()
	return b, nil
}

func (b fileBackup) restore() error {
	if !b.existed {
		if err := os.Remove(b.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	return os.WriteFile(b.path, b.data, b.mode)
}

func validationErrorSet(cfg *models.Config) map[string]bool {
	set := map[string]bool{}
	actual, _ := config.SplitErrorsAndWarnings(config.ValidateConfig(cfg))
	for _, err := range actual {
		set[err.Error()] = true
	}
	return set
}

// applySettingsChanges writes changes grouped by target file. If any write,
// reload, validation, or effect check fails, every touched file is restored.
func applySettingsChanges(state *settingsState, changes []settingsChange) (settingsResponse, int) {
	var resp settingsResponse
	planned, err := planSettingsChanges(state, changes, false)
	if err != nil {
		resp.Error = err.Error()
		return resp, http.StatusBadRequest
	}

	var order []string
	byFile := map[string][]config.BakeSetting{}
	for _, p := range planned {
		if _, ok := byFile[p.target]; !ok {
			order = append(order, p.target)
		}
		byFile[p.target] = append(byFile[p.target], config.BakeSetting{Path: p.path, Value: p.value})
	}

	var backups []fileBackup
	rollback := func() {
		for i := len(backups) - 1; i >= 0; i-- {
			if err := backups[i].restore(); err != nil {
				resp.Warnings = append(resp.Warnings, "could not restore "+displayConfigPath(backups[i].path)+": "+err.Error())
			}
		}
	}
	for _, path := range order {
		backup, err := backupFile(path)
		if err != nil {
			rollback()
			resp.Error = err.Error()
			return resp, http.StatusInternalServerError
		}
		backups = append(backups, backup)
		if err := config.BakeSettings(path, byFile[path]); err != nil {
			rollback()
			resp.Error = err.Error()
			if errors.Is(err, config.ErrBakeUnsupportedLayout) {
				return resp, http.StatusConflict
			}
			return resp, http.StatusInternalServerError
		}
	}

	before := validationErrorSet(state.cfg)
	reloaded, _, err := resolveServeConfigSources(cfgFile)
	if err != nil {
		rollback()
		resp.Error = "config no longer loads, changes rolled back: " + err.Error()
		return resp, http.StatusUnprocessableEntity
	}
	if err := checkSettingsTookEffect(before, reloaded, planned, &resp); err != nil {
		rollback()
		resp.Error = err.Error() + "; changes rolled back"
		return resp, http.StatusUnprocessableEntity
	}

	keysByFile := map[string][]string{}
	baked := map[string]bool{}
	for _, p := range planned {
		baked[p.key] = true
		target := displayConfigPath(p.target)
		resp.Changed = append(resp.Changed, settingsChanged{Key: p.key, Target: target})
		keysByFile[target] = append(keysByFile[target], p.key)
	}
	for _, path := range order {
		target := displayConfigPath(path)
		infof("Baked settings into %s (%s)", target, strings.Join(keysByFile[target], ", "))
	}
	dropServePreviewKeys(baked)
	resp.Preview = servePreviewChanges()
	if serveRequestFullRebuild != nil {
		serveRequestFullRebuild()
	}
	return resp, http.StatusOK
}
