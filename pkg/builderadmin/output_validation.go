package builderadmin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/WaylonWalker/markata-go/pkg/config"
)

const defaultContentIndexOutput = "content-index.json"

func (s *Service) validateCandidateOutput(candidate string) error {
	configPath := s.cfg.ConfigPath
	if configPath != "" && !filepath.IsAbs(configPath) {
		configPath = filepath.Join(s.cfg.SourceDir, configPath)
	}
	return validateBuildOutput(candidate, configPath)
}

// validateBuildOutput checks the minimum filesystem contract required before a
// rendered candidate can replace the live release.
func validateBuildOutput(candidate, configPath string) error {
	if err := validateNonEmptyRegularFile(filepath.Join(candidate, "index.html"), "homepage"); err != nil {
		return err
	}

	contentIndex, required, err := configuredContentIndex(configPath)
	if err != nil {
		return err
	}
	if contentIndex == "" {
		contentIndex = defaultContentIndexOutput
	}
	path, err := candidateArtifactPath(candidate, contentIndex)
	if err != nil {
		return err
	}
	if required {
		return validateJSONArtifact(path, "content index")
	}
	if _, err := os.Lstat(path); err == nil {
		return validateJSONArtifact(path, "content index")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect optional content index %q: %w", path, err)
	}
	return nil
}

func validateNonEmptyRegularFile(path, label string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("candidate %s is missing: %w", label, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("candidate %s is not a regular file: %s", label, path)
	}
	if info.Size() == 0 {
		return fmt.Errorf("candidate %s is empty: %s", label, path)
	}
	return nil
}

func validateJSONArtifact(path, label string) error {
	if err := validateNonEmptyRegularFile(path, label); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read candidate %s: %w", label, err)
	}
	if !json.Valid(data) {
		return fmt.Errorf("candidate %s is not valid JSON: %s", label, path)
	}
	return nil
}

func configuredContentIndex(configPath string) (output string, required bool, err error) {
	if strings.TrimSpace(configPath) == "" {
		return "", false, nil
	}
	siteConfig, err := config.Load(configPath)
	if err != nil {
		return "", false, fmt.Errorf("load build configuration for candidate validation: %w", err)
	}
	raw, ok := siteConfig.Extra["content_index"]
	if !ok {
		return "", false, nil
	}
	values, ok := raw.(map[string]interface{})
	if !ok {
		return "", false, fmt.Errorf("content_index configuration has unexpected type %T", raw)
	}
	enabled, _ := values["enabled"].(bool)
	output, _ = values["output"].(string)
	return output, enabled, nil
}

func candidateArtifactPath(candidate, relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("candidate artifact path must be relative: %q", relative)
	}
	clean := filepath.Clean(relative)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("candidate artifact path escapes candidate: %q", relative)
	}
	return filepath.Join(candidate, clean), nil
}
