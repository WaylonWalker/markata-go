package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const cleanPublishAssetGrace = 2 * time.Minute

var (
	errAtomicExchangeUnsupported = errors.New("atomic directory exchange unsupported")
	cleanPublishHTMLAssetPattern = regexp.MustCompile(`(?i)(?:href|src)\s*=\s*["']([^"']+)["']`)
	cleanPublishCSSURLPattern    = regexp.MustCompile(`(?i)url\(\s*["']?([^"')]+)`)
	cleanPublishCSSImportPattern = regexp.MustCompile(`(?i)@import\s+["']([^"']+)["']`)
)

func init() {
	originalRunE := buildCmd.RunE
	buildCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if (!buildClean && !buildCleanAll) || buildDryRun {
			return originalRunE(cmd, args)
		}
		return runStagedCleanBuild(cmd, args, originalRunE)
	}
}

func runStagedCleanBuild(cmd *cobra.Command, args []string, run func(*cobra.Command, []string) error) error {
	finalOutput, err := resolveCleanBuildOutput()
	if err != nil {
		return err
	}

	finalOutput, err = filepath.Abs(finalOutput)
	if err != nil {
		return fmt.Errorf("resolve clean build output path: %w", err)
	}
	parent := filepath.Dir(finalOutput)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create clean build output parent: %w", err)
	}

	stage, err := os.MkdirTemp(parent, "."+filepath.Base(finalOutput)+".markata-build-*")
	if err != nil {
		return fmt.Errorf("create clean build staging directory: %w", err)
	}
	defer os.RemoveAll(stage)

	stage, err = filepath.Abs(stage)
	if err != nil {
		return fmt.Errorf("resolve clean build staging path: %w", err)
	}

	originalOutputOverride := outputDir
	outputDir = stage
	defer func() { outputDir = originalOutputOverride }()

	if verbose {
		verbosef("Clean build staging output in %s", stage)
	}
	if err := run(cmd, args); err != nil {
		return err
	}

	// Keep in-flight browser assets outside the published output tree. This
	// preserves clean-build determinism while allowing `serve` to satisfy a CSS,
	// JS, or font request from HTML that left the server just before the swap.
	if err := refreshBrowserAssetGrace(finalOutput); err != nil {
		return fmt.Errorf("refresh previous generation browser assets: %w", err)
	}
	if err := publishStagedOutput(stage, finalOutput); err != nil {
		return fmt.Errorf("publish staged clean build: %w", err)
	}
	if verbose {
		verbosef("Published clean build to %s", finalOutput)
	}
	return nil
}

func resolveCleanBuildOutput() (string, error) {
	cfg, configPathUsed, _, err := loadManagerConfig(cfgFile)
	if err != nil {
		return "", fmt.Errorf("resolve clean build output: %w", err)
	}

	configuredOutput := cfg.OutputDir
	if outputDir != "" {
		configuredOutput = outputDir
	}
	if configuredOutput == "" {
		configuredOutput = defaultOutputDir
	}
	configuredOutput = resolveConfigRelativePath(resolveConfigBaseDir(configPathUsed), configuredOutput)
	return filepath.Clean(configuredOutput), nil
}

func browserAssetGraceDir(output string) string {
	clean := filepath.Clean(output)
	return filepath.Join(filepath.Dir(clean), "."+filepath.Base(clean)+".markata-browser-grace")
}

// refreshBrowserAssetGrace snapshots browser-critical assets reachable from the
// currently published HTML into a hidden sibling directory immediately before
// the generation switch. The files are timestamped when they retire. Older
// retired assets remain available for a bounded grace window without becoming
// part of the new output generation or its deterministic manifest.
func refreshBrowserAssetGrace(previousOutput string) error {
	info, err := os.Stat(previousOutput)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	graceDir := browserAssetGraceDir(previousOutput)
	if err := os.MkdirAll(graceDir, 0o755); err != nil {
		return err
	}

	retiredAt := time.Now()
	if err := pruneBrowserAssetGrace(graceDir, retiredAt.Add(-cleanPublishAssetGrace)); err != nil {
		return err
	}

	visited := make(map[string]struct{})
	return filepath.WalkDir(previousOutput, func(htmlPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".html") {
			return nil
		}

		content, err := os.ReadFile(htmlPath)
		if err != nil {
			return err
		}
		relHTML, err := filepath.Rel(previousOutput, htmlPath)
		if err != nil {
			return err
		}
		for _, match := range cleanPublishHTMLAssetPattern.FindAllSubmatch(content, -1) {
			if len(match) != 2 {
				continue
			}
			if err := retireReferencedBrowserAsset(previousOutput, graceDir, relHTML, string(match[1]), retiredAt, visited); err != nil {
				return err
			}
		}
		return nil
	})
}

func pruneBrowserAssetGrace(graceDir string, cutoff time.Time) error {
	return filepath.WalkDir(graceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		return nil
	})
}

func retireReferencedBrowserAsset(previousOutput, graceDir, baseFile, rawRef string, retiredAt time.Time, visited map[string]struct{}) error {
	relative, ok := resolveBrowserAssetPath(rawRef, baseFile)
	if !ok {
		return nil
	}
	if _, seen := visited[relative]; seen {
		return nil
	}
	visited[relative] = struct{}{}

	source := filepath.Join(previousOutput, relative)
	info, err := os.Stat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}

	destination := filepath.Join(graceDir, relative)
	if err := copyRetiredBrowserAsset(source, destination, retiredAt); err != nil {
		return err
	}

	if !strings.EqualFold(filepath.Ext(relative), ".css") {
		return nil
	}
	return retireCSSDependencies(previousOutput, graceDir, relative, source, retiredAt, visited)
}

func retireCSSDependencies(previousOutput, graceDir, relativeCSS, source string, retiredAt time.Time, visited map[string]struct{}) error {
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	for _, pattern := range []*regexp.Regexp{cleanPublishCSSURLPattern, cleanPublishCSSImportPattern} {
		for _, match := range pattern.FindAllSubmatch(content, -1) {
			if len(match) != 2 {
				continue
			}
			if err := retireReferencedBrowserAsset(previousOutput, graceDir, relativeCSS, string(match[1]), retiredAt, visited); err != nil {
				return err
			}
		}
	}
	return nil
}

func resolveBrowserAssetPath(rawRef, baseFile string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawRef))
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.Path == "" {
		return "", false
	}

	switch strings.ToLower(filepath.Ext(parsed.Path)) {
	case ".css", ".js", ".mjs", ".woff", ".woff2", ".ttf", ".otf":
	default:
		return "", false
	}

	var relative string
	if strings.HasPrefix(parsed.Path, "/") {
		relative = filepath.FromSlash(strings.TrimPrefix(parsed.Path, "/"))
	} else {
		relative = filepath.Join(filepath.Dir(baseFile), filepath.FromSlash(parsed.Path))
	}
	relative = filepath.Clean(relative)
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", false
	}
	return relative, true
}

func browserGraceAssetPath(outputDir, requestPath string, now time.Time) (string, bool) {
	relative, ok := resolveBrowserAssetPath(requestPath, "index.html")
	if !ok {
		return "", false
	}
	candidate := filepath.Join(browserAssetGraceDir(outputDir), relative)
	info, err := os.Stat(candidate)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	if info.ModTime().Before(now.Add(-cleanPublishAssetGrace)) {
		return "", false
	}
	return candidate, true
}

func copyRetiredBrowserAsset(source, destination string, retiredAt time.Time) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	temp, err := os.CreateTemp(filepath.Dir(destination), ".markata-grace-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	removeTemp := true
	defer func() {
		_ = temp.Close()
		if removeTemp {
			_ = os.Remove(tempName)
		}
	}()
	if err := temp.Chmod(info.Mode().Perm()); err != nil {
		return err
	}

	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	if _, err := io.Copy(temp, in); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Chtimes(tempName, retiredAt, retiredAt); err != nil {
		return err
	}
	if err := os.Rename(tempName, destination); err != nil {
		return err
	}
	removeTemp = false
	return nil
}

func publishStagedOutput(stagedOutput, finalOutput string) error {
	if _, err := os.Lstat(finalOutput); errors.Is(err, os.ErrNotExist) {
		return os.Rename(stagedOutput, finalOutput)
	} else if err != nil {
		return err
	}

	if err := exchangeOutputDirectories(stagedOutput, finalOutput); err == nil {
		if removeErr := os.RemoveAll(stagedOutput); removeErr != nil {
			warnf("published clean build but could not remove previous output %s: %v", stagedOutput, removeErr)
		}
		return nil
	} else if !errors.Is(err, errAtomicExchangeUnsupported) {
		return fmt.Errorf("atomically exchange clean build output: %w", err)
	}

	return publishStagedOutputFallback(stagedOutput, finalOutput)
}

func publishStagedOutputFallback(stagedOutput, finalOutput string) error {
	backup := fmt.Sprintf("%s.markata-previous-%d-%d", finalOutput, os.Getpid(), time.Now().UnixNano())
	if err := os.Rename(finalOutput, backup); err != nil {
		return fmt.Errorf("move current output aside: %w", err)
	}

	if err := os.Rename(stagedOutput, finalOutput); err != nil {
		restoreErr := os.Rename(backup, finalOutput)
		if restoreErr != nil {
			return errors.Join(
				fmt.Errorf("publish staged output: %w", err),
				fmt.Errorf("restore previous output: %w", restoreErr),
			)
		}
		return fmt.Errorf("publish staged output: %w", err)
	}

	if err := os.RemoveAll(backup); err != nil {
		warnf("published clean build but could not remove previous output %s: %v", backup, err)
	}
	return nil
}
