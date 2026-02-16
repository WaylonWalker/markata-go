//go:build !cgo

package config

import "gopkg.in/yaml.v3"

func getYamlValue(data []byte, path []KeySegment) (any, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc == nil {
		doc = map[string]any{}
	}
	return getValueByPath(doc, path)
}

func setYamlValue(data []byte, path []KeySegment, value any) ([]byte, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc == nil {
		doc = map[string]any{}
	}
	updated, err := setValueByPath(doc, path, value)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(updated)
}
