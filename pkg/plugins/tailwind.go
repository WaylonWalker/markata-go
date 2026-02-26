// Package plugins provides lifecycle plugins for markata-go.
package plugins

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
)

const (
	tailwindIncludeCSS = "css"
	tailwindIncludeJS  = "js"
)

// TailwindPlugin runs the Tailwind standalone CLI and wires inclusion into the head.
type TailwindPlugin struct {
	config    models.TailwindConfig
	assetURLs map[string]string
}

// NewTailwindPlugin creates a new TailwindPlugin with default settings.
func NewTailwindPlugin() *TailwindPlugin {
	return &TailwindPlugin{
		config: models.NewTailwindConfig(),
	}
}

// Name returns the unique name of the plugin.
func (p *TailwindPlugin) Name() string {
	return "tailwind"
}

// Priority ensures Tailwind runs early in Configure so output CSS exists before asset copying.
func (p *TailwindPlugin) Priority(stage lifecycle.Stage) int {
	if stage == lifecycle.StageConfigure {
		return lifecycle.PriorityEarly
	}
	return lifecycle.PriorityDefault
}

// Configure reads Tailwind config, runs the CLI, and injects CSS/JS includes.
func (p *TailwindPlugin) Configure(m *lifecycle.Manager) error {
	config := m.Config()
	if config == nil {
		return nil
	}

	if config.Extra == nil {
		config.Extra = make(map[string]interface{})
	}

	if assetURLs, ok := config.Extra["asset_urls"].(map[string]string); ok {
		p.assetURLs = assetURLs
	}

	p.config = p.parseTailwindConfig(config.Extra)

	includeMode := normalizeTailwindInclude(p.config.IncludeMode())

	if p.config.IsBuildEnabled() {
		if err := p.runTailwindBuild(config); err != nil {
			return err
		}
	}

	if includeMode != "" {
		if err := p.injectIncludes(config, includeMode); err != nil {
			return err
		}
	}

	return nil
}

func (p *TailwindPlugin) parseTailwindConfig(extra map[string]interface{}) models.TailwindConfig {
	result := models.NewTailwindConfig()
	if extra == nil {
		return result
	}

	if cfg, ok := extra["tailwind"].(models.TailwindConfig); ok {
		return cfg
	}

	raw, ok := extra["tailwind"].(map[string]interface{})
	if !ok {
		return result
	}

	if include, ok := raw["include"]; ok {
		if v := tailwindConfigString(include); v != "" {
			result.Include = &v
		} else if b, ok := include.(bool); ok {
			if b {
				value := "css"
				result.Include = &value
			} else {
				value := "false"
				result.Include = &value
			}
		}
	}

	if v := tailwindConfigString(raw["input"]); v != "" {
		result.Input = v
	}
	if v := tailwindConfigString(raw["output"]); v != "" {
		result.Output = v
	}
	if v := tailwindConfigString(raw["config_file"]); v != "" {
		result.ConfigFile = v
	}
	if v := tailwindConfigString(raw["config"]); v != "" {
		result.ConfigFile = v
	}
	if rawBuild, ok := raw["build"]; ok {
		value := tailwindConfigBoolean(rawBuild, true)
		result.Build = &value
	}
	if rawMinify, ok := raw["minify"]; ok {
		value := tailwindConfigBoolean(rawMinify, true)
		result.Minify = &value
	}
	if rawAuto, ok := raw["auto_install"]; ok {
		value := tailwindConfigBoolean(rawAuto, true)
		result.AutoInstall = &value
	}
	if v := tailwindConfigString(raw["version"]); v != "" {
		if normalized, err := tailwindNormalizeVersion(v); err == nil {
			result.Version = normalized
		}
	}
	if v := tailwindConfigString(raw["cache_dir"]); v != "" {
		result.CacheDir = v
	}
	if v := tailwindConfigString(raw["binary"]); v != "" {
		result.Binary = v
	}
	if rawExtra := raw["extra_args"]; rawExtra != nil {
		result.ExtraArgs = tailwindConfigStringSlice(rawExtra)
	}
	if rawVerbose, ok := raw["verbose"]; ok {
		value := tailwindConfigBoolean(rawVerbose, false)
		result.Verbose = &value
	}

	return result
}

func (p *TailwindPlugin) runTailwindBuild(config *lifecycle.Config) error {
	inputPath := p.resolveAssetPath(config, p.config.Input)
	outputPath := p.resolveAssetPath(config, p.config.Output)

	if inputPath == "" || outputPath == "" {
		return fmt.Errorf("tailwind: input and output paths must be set")
	}

	if _, err := os.Stat(inputPath); err != nil {
		fmt.Printf("[tailwind] WARNING: input file not found: %s (skipping build)\n", inputPath)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("tailwind: creating output directory: %w", err)
	}

	cliPath, err := p.findOrInstallTailwind(config)
	if err != nil {
		return err
	}
	if cliPath == "" {
		return nil
	}

	args := []string{"-i", inputPath, "-o", outputPath}
	if p.config.ConfigFile != "" {
		args = append(args, "--config", p.config.ConfigFile)
	}
	if p.config.IsMinifyEnabled() {
		args = append(args, "--minify")
	}
	if len(p.config.ExtraArgs) > 0 {
		args = append(args, p.config.ExtraArgs...)
	}

	cmd := exec.Command(cliPath, args...)
	cmd.Dir = "."
	cmd.Env = os.Environ()

	if p.config.IsVerbose() {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		var stderr bytes.Buffer
		cmd.Stdout = nil
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			if stderr.Len() > 0 {
				fmt.Fprintf(os.Stderr, "%s", stderr.String())
			}
			return fmt.Errorf("tailwind build failed: %w", err)
		}
		return nil
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tailwind build failed: %w", err)
	}

	return nil
}

func (p *TailwindPlugin) findOrInstallTailwind(config *lifecycle.Config) (string, error) {
	if p.config.Binary != "" {
		if _, err := os.Stat(p.config.Binary); err == nil {
			return p.config.Binary, nil
		}
	}

	if path, err := exec.LookPath(getTailwindBinaryName()); err == nil {
		return path, nil
	}

	if !p.config.IsAutoInstallEnabled() {
		fmt.Printf("[tailwind] WARNING: tailwindcss not found in PATH, skipping build\n")
		fmt.Printf("[tailwind] Install it or set [markata-go.tailwind].auto_install = true\n")
		return "", nil
	}

	version := p.config.Version
	if normalized, err := tailwindNormalizeVersion(version); err == nil {
		version = normalized
	}

	installer := NewTailwindInstallerWithConfig(TailwindInstallerConfig{
		Version:  version,
		CacheDir: p.config.CacheDir,
	})
	installer.Verbose = p.config.IsVerbose()

	if p.config.IsVerbose() {
		fmt.Printf("[tailwind] tailwindcss not found in PATH, attempting auto-install...\n")
	}

	installedPath, err := installer.Install()
	if err != nil {
		return "", fmt.Errorf("tailwind auto-install failed: %w", err)
	}

	return installedPath, nil
}

func (p *TailwindPlugin) injectIncludes(config *lifecycle.Config, includeMode string) error {
	modelsConfig, ok := config.Extra["models_config"].(*models.Config)
	if !ok || modelsConfig == nil {
		return nil
	}

	if includeMode == tailwindIncludeCSS {
		if strings.TrimSpace(modelsConfig.Theme.CustomCSS) == "" {
			outputPath := p.resolveAssetPath(config, p.config.Output)
			if outputPath != "" {
				if _, err := os.Stat(outputPath); err == nil {
					modelsConfig.Theme.CustomCSS = p.relativeAssetPath(config, p.config.Output)
				} else {
					fmt.Printf("[tailwind] WARNING: output CSS not found: %s (skipping include)\n", outputPath)
					return nil
				}
			}
		}
		config.Extra["theme"] = modelsConfig.Theme
		config.Extra["models_config"] = modelsConfig
		return nil
	}

	if includeMode == tailwindIncludeJS {
		scriptSrc := "https://cdn.tailwindcss.com"
		if url, ok := p.assetURLs["tailwindcss-js"]; ok && url != "" {
			scriptSrc = url
		}
		if !headHasScript(modelsConfig.Head.Script, scriptSrc) {
			modelsConfig.Head.Script = append(modelsConfig.Head.Script, models.ScriptTag{Src: scriptSrc})
		}
		config.Extra["head"] = modelsConfig.Head
		config.Extra["models_config"] = modelsConfig
	}

	return nil
}

func (p *TailwindPlugin) resolveAssetPath(config *lifecycle.Config, path string) string {
	if path == "" {
		return ""
	}

	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}

	if filepath.IsAbs(trimmed) {
		return trimmed
	}

	assetsDir := ""
	if config.Extra != nil {
		if v, ok := config.Extra["assets_dir"].(string); ok && v != "" {
			assetsDir = v
		}
	}
	if assetsDir == "" {
		assetsDir = "static"
	}

	if strings.HasPrefix(trimmed, assetsDir+string(filepath.Separator)) || strings.HasPrefix(trimmed, assetsDir+"/") {
		return trimmed
	}

	return filepath.Join(assetsDir, trimmed)
}

func (p *TailwindPlugin) relativeAssetPath(config *lifecycle.Config, path string) string {
	if path == "" {
		return ""
	}

	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}

	assetsDir := ""
	if config.Extra != nil {
		if v, ok := config.Extra["assets_dir"].(string); ok && v != "" {
			assetsDir = v
		}
	}
	if assetsDir == "" {
		assetsDir = "static"
	}

	if filepath.IsAbs(trimmed) {
		rel, err := filepath.Rel(assetsDir, trimmed)
		if err != nil || strings.HasPrefix(rel, "..") {
			return trimmed
		}
		return filepath.ToSlash(rel)
	}

	if strings.HasPrefix(trimmed, assetsDir+string(filepath.Separator)) || strings.HasPrefix(trimmed, assetsDir+"/") {
		rel := strings.TrimPrefix(trimmed, assetsDir)
		rel = strings.TrimPrefix(rel, string(filepath.Separator))
		rel = strings.TrimPrefix(rel, "/")
		return filepath.ToSlash(rel)
	}

	return filepath.ToSlash(trimmed)
}

func normalizeTailwindInclude(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "false", "off", "0", "none":
		return ""
	case "css", "js":
		return value
	default:
		return ""
	}
}

func headHasScript(scripts []models.ScriptTag, src string) bool {
	if src == "" {
		return false
	}
	for _, script := range scripts {
		if strings.TrimSpace(script.Src) == src {
			return true
		}
	}
	return false
}

// Ensure TailwindPlugin implements the required interfaces.
var (
	_ lifecycle.Plugin          = (*TailwindPlugin)(nil)
	_ lifecycle.ConfigurePlugin = (*TailwindPlugin)(nil)
	_ lifecycle.PriorityPlugin  = (*TailwindPlugin)(nil)
)

// No additional helpers.
