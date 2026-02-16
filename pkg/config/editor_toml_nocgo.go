//go:build !cgo

package config

import (
	"strings"

	"github.com/BurntSushi/toml"
)

func getTomlValue(data []byte, path []KeySegment) (any, error) {
	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc == nil {
		doc = map[string]any{}
	}
	return getValueByPath(doc, path)
}

func setTomlValue(data []byte, path []KeySegment, value any) ([]byte, error) {
	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc == nil {
		doc = map[string]any{}
	}
	updated, err := setValueByPath(doc, path, value)
	if err != nil {
		return nil, err
	}

	var buf strings.Builder
	if err := toml.NewEncoder(&buf).Encode(updated); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}
