package config

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	sitter "github.com/smacker/go-tree-sitter"
	tstoml "github.com/smacker/go-tree-sitter/toml"
)

func getTomlValue(data []byte, path []KeySegment) (any, error) {
	valueNode, err := findTomlValueNode(data, path)
	if err != nil {
		return nil, err
	}
	valueText := strings.TrimSpace(nodeText(data, valueNode))
	return decodeTomlValue(valueText)
}

func setTomlValue(data []byte, path []KeySegment, value any) ([]byte, error) {
	valueText, err := encodeTomlValue(value)
	if err != nil {
		return nil, err
	}

	if valueNode, err := findTomlValueNode(data, path); err == nil {
		ed := Edit{Start: int(valueNode.StartByte()), End: int(valueNode.EndByte()), NewText: valueText}
		return applyEdit(data, ed), nil
	}

	if len(path) == 0 {
		return data, fmt.Errorf("invalid key path")
	}
	insertPos, tablePath, found := findTomlInsertPosition(data, path[:len(path)-1])
	keyName := path[len(path)-1].Key
	insertText := buildTomlInsertText(tablePath, keyName, valueText, found)
	if insertPos == 0 {
		insertText = strings.TrimLeft(insertText, "\n")
	}
	return applyEdit(data, Edit{Start: insertPos, End: insertPos, NewText: insertText}), nil
}

func findTomlValueNode(data []byte, path []KeySegment) (*sitter.Node, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(tstoml.GetLanguage())
	tree := parser.Parse(nil, data)
	root := tree.RootNode()

	currentTable := []KeySegment{}
	arrayIndices := make(map[string]int)

	for i := 0; i < int(root.NamedChildCount()); i++ {
		node := root.NamedChild(i)
		if isTomlTableNode(node) {
			currentTable = parseTomlTablePath(data, node, arrayIndices)
			continue
		}
		keyNode := node.ChildByFieldName("key")
		valueNode := node.ChildByFieldName("value")
		if keyNode == nil || valueNode == nil {
			continue
		}
		segments := parseTomlKeyText(nodeText(data, keyNode))
		fullPath := append(append([]KeySegment{}, currentTable...), segments...)
		if keySegmentsEqual(fullPath, path) {
			return valueNode, nil
		}
	}

	return nil, fmt.Errorf("key not found")
}

func findTomlInsertPosition(data []byte, tablePath []KeySegment) (int, []KeySegment, bool) {
	parser := sitter.NewParser()
	parser.SetLanguage(tstoml.GetLanguage())
	tree := parser.Parse(nil, data)
	root := tree.RootNode()

	currentTable := []KeySegment{}
	arrayIndices := make(map[string]int)
	insertPos := len(data)
	found := len(tablePath) == 0

	for i := 0; i < int(root.NamedChildCount()); i++ {
		node := root.NamedChild(i)
		if isTomlTableNode(node) {
			if found {
				return insertPos, tablePath, true
			}
			currentTable = parseTomlTablePath(data, node, arrayIndices)
			if keySegmentsEqual(currentTable, tablePath) {
				found = true
				insertPos = int(node.EndByte())
			} else {
				insertPos = int(node.EndByte())
			}
			continue
		}
		if found {
			insertPos = int(node.EndByte())
		}
	}

	return insertPos, tablePath, found
}

func buildTomlInsertText(tablePath []KeySegment, keyName, valueText string, tableFound bool) string {
	keyLine := fmt.Sprintf("%s = %s", keyName, valueText)
	if len(tablePath) == 0 {
		return "\n" + keyLine
	}
	if tableFound {
		return "\n" + keyLine
	}
	return fmt.Sprintf("\n\n[%s]\n%s", tomlTablePathString(tablePath), keyLine)
}

func tomlTablePathString(path []KeySegment) string {
	parts := make([]string, 0, len(path))
	for _, seg := range path {
		parts = append(parts, seg.Key)
	}
	return strings.Join(parts, ".")
}

func parseTomlTablePath(data []byte, node *sitter.Node, arrayIndices map[string]int) []KeySegment {
	text := strings.TrimSpace(nodeText(data, node))
	if strings.HasPrefix(text, "[[") {
		text = strings.TrimPrefix(text, "[[")
		text = strings.TrimSuffix(text, "]]")
		segments := parseTomlKeyText(text)
		key := tomlTablePathString(segments)
		idx := arrayIndices[key]
		arrayIndices[key] = idx + 1
		segments[len(segments)-1].Index = &idx
		return segments
	}
	text = strings.TrimPrefix(text, "[")
	text = strings.TrimSuffix(text, "]")
	return parseTomlKeyText(text)
}

func parseTomlKeyText(text string) []KeySegment {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	parts := strings.Split(text, ".")
	segments := make([]KeySegment, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		if part == "" {
			continue
		}
		segments = append(segments, KeySegment{Key: part})
	}
	return segments
}

func isTomlTableNode(node *sitter.Node) bool {
	return strings.Contains(node.Type(), "table")
}

func decodeTomlValue(valueText string) (any, error) {
	var wrapper map[string]any
	_, err := toml.Decode("value = "+valueText, &wrapper)
	if err != nil {
		return nil, err
	}
	return wrapper["value"], nil
}

func encodeTomlValue(value any) (string, error) {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%q", v), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case int:
		return fmt.Sprintf("%d", v), nil
	case int64:
		return fmt.Sprintf("%d", v), nil
	case float64:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", v), "0"), "."), nil
	case []string:
		items := make([]string, 0, len(v))
		for _, item := range v {
			items = append(items, fmt.Sprintf("%q", item))
		}
		return "[" + strings.Join(items, ", ") + "]", nil
	case []any:
		items := make([]string, 0, len(v))
		for _, item := range v {
			text, err := encodeTomlValue(item)
			if err != nil {
				return "", err
			}
			items = append(items, text)
		}
		return "[" + strings.Join(items, ", ") + "]", nil
	case map[string]any:
		parts := make([]string, 0, len(v))
		for key, item := range v {
			text, err := encodeTomlValue(item)
			if err != nil {
				return "", err
			}
			parts = append(parts, fmt.Sprintf("%s = %s", key, text))
		}
		return "{" + strings.Join(parts, ", ") + "}", nil
	default:
		return jsonValueString(value), nil
	}
}

func nodeText(data []byte, node *sitter.Node) string {
	if node == nil {
		return ""
	}
	return string(data[node.StartByte():node.EndByte()])
}
