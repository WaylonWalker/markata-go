package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetValueFromFile reads a config file and returns the value for a key path.
func GetValueFromFile(path, key string) (any, error) {
	segments, err := ParseKeyPath(key)
	if err != nil {
		return nil, err
	}
	segments = normalizeConfigPath(segments)

	format := detectConfigFormat(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	switch format {
	case FormatTOML:
		return getTomlValue(data, segments)
	case FormatYAML:
		return getYamlValue(data, segments)
	case FormatJSON:
		return getJSONValue(data, segments)
	default:
		return nil, fmt.Errorf("unsupported config format: %s", format)
	}
}

// SetValueInFile updates a config file in place with the provided key path/value.
func SetValueInFile(path, key string, value any) error {
	segments, err := ParseKeyPath(key)
	if err != nil {
		return err
	}
	segments = normalizeConfigPath(segments)

	format := detectConfigFormat(path)

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat config file: %w", err)
	}
	mode := info.Mode().Perm()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var updated []byte
	switch format {
	case FormatTOML:
		updated, err = setTomlValue(data, segments, value)
	case FormatYAML:
		updated, err = setYamlValue(data, segments, value)
	case FormatJSON:
		updated, err = setJSONValue(data, segments, value)
	default:
		return fmt.Errorf("unsupported config format: %s", format)
	}
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, updated, mode); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

type Edit struct {
	Start   int
	End     int
	NewText string
}

func applyEdit(source []byte, edit Edit) []byte {
	if edit.Start < 0 {
		edit.Start = 0
	}
	if edit.End < edit.Start {
		edit.End = edit.Start
	}
	updated := make([]byte, 0, len(source)+(len(edit.NewText)))
	updated = append(updated, source[:edit.Start]...)
	updated = append(updated, []byte(edit.NewText)...)
	updated = append(updated, source[edit.End:]...)
	return updated
}

func detectConfigFormat(path string) Format {
	switch filepath.Ext(path) {
	case ".toml":
		return FormatTOML
	case ".yaml", ".yml":
		return FormatYAML
	case ".json":
		return FormatJSON
	default:
		return FormatTOML
	}
}

func jsonValueString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

func normalizeConfigPath(path []KeySegment) []KeySegment {
	if len(path) == 0 {
		return path
	}
	if strings.EqualFold(path[0].Key, "markata-go") {
		return path
	}
	return append([]KeySegment{{Key: "markata-go"}}, path...)
}
