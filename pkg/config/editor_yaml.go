package config

import (
	"bytes"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	tsyaml "github.com/smacker/go-tree-sitter/yaml"
	"gopkg.in/yaml.v3"
)

func getYamlValue(data []byte, path []KeySegment) (any, error) {
	valueNode, err := findYamlValueNode(data, path)
	if err != nil {
		return nil, err
	}
	valueText := strings.TrimSpace(nodeText(data, valueNode))
	var value any
	if err := yaml.Unmarshal([]byte(valueText), &value); err != nil {
		return nil, err
	}
	return value, nil
}

func setYamlValue(data []byte, path []KeySegment, value any) ([]byte, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(tsyaml.GetLanguage())
	tree := parser.Parse(nil, data)
	root := tree.RootNode()

	if valueNode, err := findYamlValueNodeWithRoot(data, root, path); err == nil {
		indent := lineIndent(data, int(valueNode.StartByte()))
		encoded, err := encodeYamlValue(value, indent)
		if err != nil {
			return nil, err
		}
		ed := Edit{Start: int(valueNode.StartByte()), End: int(valueNode.EndByte()), NewText: encoded}
		return applyEdit(data, ed), nil
	}

	insertNode := findYamlMappingNode(root)
	if insertNode == nil {
		insertText, err := buildYamlInsertText(path, value, "")
		if err != nil {
			return nil, err
		}
		insertText = strings.TrimPrefix(insertText, "\n")
		if !strings.HasSuffix(insertText, "\n") {
			insertText += "\n"
		}
		return append([]byte(insertText), data...), nil
	}

	indent := lineIndent(data, int(insertNode.StartByte()))
	insertText, err := buildYamlInsertText(path, value, indent)
	if err != nil {
		return nil, err
	}
	pos := int(insertNode.EndByte())
	return applyEdit(data, Edit{Start: pos, End: pos, NewText: insertText}), nil
}

func findYamlValueNode(data []byte, path []KeySegment) (*sitter.Node, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(tsyaml.GetLanguage())
	tree := parser.Parse(nil, data)
	root := tree.RootNode()
	return findYamlValueNodeWithRoot(data, root, path)
}

func findYamlValueNodeWithRoot(data []byte, root *sitter.Node, path []KeySegment) (*sitter.Node, error) {
	current := findYamlMappingNode(root)
	if current == nil {
		return nil, fmt.Errorf("no mapping found")
	}
	for i, segment := range path {
		pair := findYamlPairForKey(data, current, segment.Key)
		if pair == nil {
			return nil, fmt.Errorf("key not found")
		}
		valueNode := pair.ChildByFieldName("value")
		if valueNode == nil {
			return nil, fmt.Errorf("key not found")
		}
		if segment.Index != nil {
			item := yamlSequenceItem(valueNode, *segment.Index)
			if item == nil {
				return nil, fmt.Errorf("index out of range")
			}
			valueNode = item
		}
		if i == len(path)-1 {
			return valueNode, nil
		}
		current = valueNode
	}
	return nil, fmt.Errorf("key not found")
}

func findYamlMappingNode(root *sitter.Node) *sitter.Node {
	if root == nil {
		return nil
	}
	if isYamlMappingNode(root) {
		return root
	}
	for i := 0; i < int(root.NamedChildCount()); i++ {
		if node := findYamlMappingNode(root.NamedChild(i)); node != nil {
			return node
		}
	}
	return nil
}

func findYamlPairForKey(data []byte, mapping *sitter.Node, key string) *sitter.Node {
	if mapping == nil {
		return nil
	}
	for i := 0; i < int(mapping.NamedChildCount()); i++ {
		child := mapping.NamedChild(i)
		keyNode := child.ChildByFieldName("key")
		valueNode := child.ChildByFieldName("value")
		if keyNode == nil || valueNode == nil {
			continue
		}
		if normalizeKey(nodeText(data, keyNode)) == key {
			return child
		}
	}
	return nil
}

func yamlSequenceItem(sequence *sitter.Node, index int) *sitter.Node {
	if sequence == nil || index < 0 {
		return nil
	}
	items := make([]*sitter.Node, 0)
	for i := 0; i < int(sequence.NamedChildCount()); i++ {
		child := sequence.NamedChild(i)
		valueNode := child.ChildByFieldName("value")
		if valueNode != nil {
			items = append(items, valueNode)
			continue
		}
		items = append(items, child)
	}
	if index >= len(items) {
		return nil
	}
	return items[index]
}

func isYamlMappingNode(node *sitter.Node) bool {
	return node != nil && strings.Contains(node.Type(), "mapping")
}

func normalizeKey(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return value
}

func lineIndent(data []byte, pos int) string {
	if pos <= 0 {
		return ""
	}
	start := pos
	for start > 0 && data[start-1] != '\n' {
		start--
	}
	return string(bytes.TrimRight(data[start:pos], "\t "))
}

func buildYamlInsertText(path []KeySegment, value any, indent string) (string, error) {
	indentUnit := "  "
	lines := make([]string, 0)
	currentIndent := indent
	for i, segment := range path {
		key := segment.Key
		if i == len(path)-1 {
			encoded, err := encodeYamlValue(value, currentIndent+indentUnit)
			if err != nil {
				return "", err
			}
			lines = append(lines, fmt.Sprintf("%s%s: %s", currentIndent, key, encoded))
			break
		}
		lines = append(lines, fmt.Sprintf("%s%s:", currentIndent, key))
		currentIndent += indentUnit
	}
	return "\n" + strings.Join(lines, "\n"), nil
}

func encodeYamlValue(value any, indent string) (string, error) {
	data, err := yaml.Marshal(value)
	if err != nil {
		return "", err
	}
	text := strings.TrimRight(string(data), "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		return lines[0], nil
	}
	for i := range lines {
		lines[i] = indent + lines[i]
	}
	return "\n" + strings.Join(lines, "\n"), nil
}
