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

	if err := retainPreviousGenerationBrowserAssets(finalOutput, stage); err != nil {
		return fmt.Errorf("retain previous generation browser assets: %w", err)
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

// retainPreviousGenerationBrowserAssets keeps browser-critical assets that may
// still be requested by a page served immediately before a clean publish. The
// current generation's HTML references are always retained, including CSS
// dependencies. Assets becoming retired are timestamped when they enter the
// grace set; already-retired assets preserve that retirement timestamp while
// they are carried forward. This keeps the grace window bounded without
// dropping an old-but-current asset immediately after it is replaced.
func retainPreviousGenerationBrowserAssets(previousOutput, stagedOutput string) error {
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

	visited := make(map[string]struct{})
	if err := retainReferencedBrowserAssets(previousOutput, stagedOutput, visited); err != nil {
		return err
	}
	return retainRecentBrowserAssets(previousOutput, stagedOutput, time.Now().Add(-cleanPublishAssetGrace), visited)
}

func retainReferencedBrowserAssets(previousOutput, stagedOutput string, visited map[string]struct{}) error {
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
			if err := retainReferencedBrowserAsset(previousOutput, stagedOutput, relHTML, string(match[1]), visited); err != nil {
				return err
			}
		}
		return nil
	})
}

func retainRecentBrowserAssets(previousOutput, stagedOutput string, cutoff time.Time, visited map[string]struct{}) error {
	return filepath.WalkDir(previousOutput, func(source string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isBrowserCriticalAsset(entry.Name()) {
			return nil
		}

		relative, err := filepath.Rel(previousOutput, source)
		if err != nil {
			return err
		}
		if _, seen := visited[relative]; seen {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.ModTime().Before(cutoff) {
			return nil
		}

		visited[relative] = struct{}{}
		destination := filepath.Join(stagedOutput, relative)
		if _, err := os.Stat(destination); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return copyPublishedAsset(source, destination)
	})
}

func isBrowserCriticalAsset(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".css", ".js", ".mjs", ".woff", ".woff2", ".ttf", ".otf":
		return true
	default:
		return false
	}
}

func retainReferencedBrowserAsset(previousOutput, stagedOutput, baseFile, rawRef string, visited map[string]struct{}) error {
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

	destination := filepath.Join(stagedOutput, relative)
	if _, err := os.Stat(destination); errors.Is(err, os.ErrNotExist) {
		if err := copyPublishedAsset(source, destination); err != nil {
			return err
		}
		retiredAt := time.Now()
		if err := os.Chtimes(destination, retiredAt, retiredAt); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if !strings.EqualFold(filepath.Ext(relative), ".css") {
		return nil
	}
	return retainCSSDependencies(previousOutput, stagedOutput, relative, source, visited)
}

func retainCSSDependencies(previousOutput, stagedOutput, relativeCSS, source string, visited map[string]struct{}) error {
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	for _, pattern := range []*regexp.Regexp{cleanPublishCSSURLPattern, cleanPublishCSSImportPattern} {
		for _, match := range pattern.FindAllSubmatch(content, -1) {
			if len(match) != 2 {
				continue
			}
			if err := retainReferencedBrowserAsset(previousOutput, stagedOutput, relativeCSS, string(match[1]), visited); err != nil {
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

func copyPublishedAsset(source, destination string) error {
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

	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return err
	}
	removePartial := true
	defer func() {
		_ = out.Close()
		if removePartial {
			_ = os.Remove(destination)
		}
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Chtimes(destination, info.ModTime(), info.ModTime()); err != nil {
		return err
	}
	removePartial = false
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
