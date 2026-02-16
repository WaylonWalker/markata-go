package config

import (
	"encoding/json"
	"fmt"
)

func getJSONValue(data []byte, path []KeySegment) (any, error) {
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	value, err := getValueByPath(doc, path)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func setJSONValue(data []byte, path []KeySegment, value any) ([]byte, error) {
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	updated, err := setValueByPath(doc, path, value)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(updated, "", "  ")
}

func getValueByPath(doc any, path []KeySegment) (any, error) {
	current := doc
	for _, segment := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("path not found")
		}
		value, exists := m[segment.Key]
		if !exists {
			return nil, fmt.Errorf("path not found")
		}
		current = value
		if segment.Index != nil {
			arr, ok := current.([]any)
			if !ok {
				return nil, fmt.Errorf("expected array at %s", segment.Key)
			}
			if *segment.Index < 0 || *segment.Index >= len(arr) {
				return nil, fmt.Errorf("index out of range")
			}
			current = arr[*segment.Index]
		}
	}
	return current, nil
}

func setValueByPath(doc any, path []KeySegment, value any) (any, error) {
	if len(path) == 0 {
		return value, nil
	}
	current, ok := doc.(map[string]any)
	if !ok {
		current = map[string]any{}
	}
	root := current
	for i, segment := range path {
		if i == len(path)-1 {
			if segment.Index == nil {
				current[segment.Key] = value
				return root, nil
			}
			arr, ok := current[segment.Key].([]any)
			if !ok {
				arr = []any{}
			}
			if *segment.Index < 0 {
				return nil, fmt.Errorf("index out of range")
			}
			for len(arr) <= *segment.Index {
				arr = append(arr, nil)
			}
			arr[*segment.Index] = value
			current[segment.Key] = arr
			return root, nil
		}

		next, ok := current[segment.Key]
		if !ok {
			current[segment.Key] = map[string]any{}
			next = current[segment.Key]
		}
		if segment.Index != nil {
			arr, ok := next.([]any)
			if !ok {
				arr = []any{}
			}
			for len(arr) <= *segment.Index {
				arr = append(arr, map[string]any{})
			}
			current[segment.Key] = arr
			next = arr[*segment.Index]
		}
		child, ok := next.(map[string]any)
		if !ok {
			child = map[string]any{}
		}
		current[segment.Key] = child
		current = child
	}

	return root, nil
}
