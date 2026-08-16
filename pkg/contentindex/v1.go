package contentindex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

type v1Index struct {
	SchemaURL     string       `json:"$schema"`
	Schema        string       `json:"schema"`
	SchemaVersion int          `json:"schema_version"`
	Generator     v1Generator  `json:"generator"`
	Source        v1Source     `json:"source"`
	DocumentCount int          `json:"document_count"`
	Documents     []v1Document `json:"documents"`
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

type v1Generator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
type v1Source struct {
	Commit *string `json:"commit,omitempty"`
}
type v1Document struct {
	Path        string     `json:"path"`
	Slug        string     `json:"slug"`
	Href        string     `json:"href"`
	Title       *string    `json:"title,omitempty"`
	TitleText   *string    `json:"title_text,omitempty"`
	Date        *time.Time `json:"date,omitempty"`
	Modified    *time.Time `json:"modified,omitempty"`
	Published   bool       `json:"published"`
	Draft       bool       `json:"draft"`
	Private     bool       `json:"private"`
	Template    string     `json:"template,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	Description *string    `json:"description,omitempty"`
	Feeds       []string   `json:"feeds,omitempty"`
	Aliases     []string   `json:"aliases,omitempty"`
}

func encodeV1(index Index) ([]byte, error) {
	if index.Schema == "" {
		index.Schema = Schema
	}
	if index.SchemaVersion == 0 {
		index.SchemaVersion = CurrentVersion
	}
	if index.DocumentCount != 0 && index.DocumentCount != len(index.Documents) {
		return nil, fmt.Errorf("document_count %d does not match documents length %d", index.DocumentCount, len(index.Documents))
	}
	index.DocumentCount = len(index.Documents)
	if index.Generator.Name == "" {
		return nil, fmt.Errorf("generator.name is required")
	}
	seenPaths := make(map[string]struct{}, len(index.Documents))
	sort.SliceStable(index.Documents, func(i, j int) bool { return index.Documents[i].Path < index.Documents[j].Path })
	docs := make([]v1Document, len(index.Documents))
	for i, d := range index.Documents {
		if d.Path == "" {
			return nil, fmt.Errorf("documents[%d].path is required", i)
		}
		if d.Slug == "" && d.Href == "" {
			return nil, fmt.Errorf("documents[%d] must have slug or href", i)
		}
		if _, exists := seenPaths[d.Path]; exists {
			return nil, fmt.Errorf("documents[%d].path is duplicated: %q", i, d.Path)
		}
		seenPaths[d.Path] = struct{}{}
		d.Tags = sortedStrings(d.Tags)
		d.Feeds = sortedStrings(d.Feeds)
		d.Aliases = sortedStrings(d.Aliases)
		docs[i] = v1Document{d.Path, d.Slug, d.Href, d.Title, d.TitleText, d.Date, d.Modified, d.Published, d.Draft, d.Private, d.Template, d.Tags, d.Description, d.Feeds, d.Aliases}
	}
	var commit *string
	if index.Source.Commit != "" {
		commit = &index.Source.Commit
	}
	return json.Marshal(v1Index{SchemaURL, index.Schema, index.SchemaVersion, v1Generator{index.Generator.Name, index.Generator.Version}, v1Source{commit}, index.DocumentCount, docs})
}

func decodeV1(data []byte) (Index, error) {
	var wire v1Index
	if err := json.Unmarshal(data, &wire); err != nil {
		return Index{}, fmt.Errorf("decode v1 JSON: %w", err)
	}
	var raw struct {
		Generator     map[string]json.RawMessage   `json:"generator"`
		Source        json.RawMessage              `json:"source"`
		DocumentCount *int                         `json:"document_count"`
		Documents     []map[string]json.RawMessage `json:"documents"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Index{}, fmt.Errorf("validate v1 structure: %w", err)
	}
	if raw.Generator == nil || raw.Source == nil || raw.DocumentCount == nil || raw.Documents == nil {
		return Index{}, fmt.Errorf("v1 requires generator, source, and documents")
	}
	if string(raw.Source) == "null" || string(raw.Source) == "" {
		return Index{}, fmt.Errorf("source must be an object")
	}
	var sourceFields map[string]json.RawMessage
	if err := json.Unmarshal(raw.Source, &sourceFields); err != nil {
		return Index{}, fmt.Errorf("source must be an object")
	}
	if commit, ok := sourceFields["commit"]; ok {
		var value string
		if isJSONNull(commit) || json.Unmarshal(commit, &value) != nil {
			return Index{}, fmt.Errorf("source.commit must be a string")
		}
	}
	for _, field := range []string{"name", "version"} {
		if _, ok := raw.Generator[field]; !ok {
			return Index{}, fmt.Errorf("generator.%s is required", field)
		}
		var value string
		if isJSONNull(raw.Generator[field]) {
			return Index{}, fmt.Errorf("generator.%s must be a string", field)
		}
		if err := json.Unmarshal(raw.Generator[field], &value); err != nil {
			return Index{}, fmt.Errorf("generator.%s must be a string", field)
		}
	}
	for i, document := range raw.Documents {
		for _, field := range []string{"path", "slug", "href", "published", "draft", "private"} {
			if _, ok := document[field]; !ok {
				return Index{}, fmt.Errorf("documents[%d].%s is required", i, field)
			}
		}
		for _, field := range []string{"path", "slug", "href"} {
			var value string
			if isJSONNull(document[field]) {
				return Index{}, fmt.Errorf("documents[%d].%s must be a string", i, field)
			}
			if err := json.Unmarshal(document[field], &value); err != nil {
				return Index{}, fmt.Errorf("documents[%d].%s must be a string", i, field)
			}
		}
		for _, field := range []string{"published", "draft", "private"} {
			var value bool
			if isJSONNull(document[field]) {
				return Index{}, fmt.Errorf("documents[%d].%s must be a boolean", i, field)
			}
			if err := json.Unmarshal(document[field], &value); err != nil {
				return Index{}, fmt.Errorf("documents[%d].%s must be a boolean", i, field)
			}
		}
		for _, field := range []string{"tags", "feeds", "aliases"} {
			if value, ok := document[field]; ok {
				if isJSONNull(value) {
					return Index{}, fmt.Errorf("documents[%d].%s must be an array of strings", i, field)
				}
				var values []string
				if err := json.Unmarshal(value, &values); err != nil {
					return Index{}, fmt.Errorf("documents[%d].%s must be an array of strings", i, field)
				}
			}
		}
		for _, field := range []string{"date", "modified", "template"} {
			if value, ok := document[field]; ok {
				if isJSONNull(value) {
					return Index{}, fmt.Errorf("documents[%d].%s must be a string", i, field)
				}
				if field == "template" {
					var text string
					if err := json.Unmarshal(value, &text); err != nil {
						return Index{}, fmt.Errorf("documents[%d].template must be a string", i)
					}
				} else {
					var date time.Time
					if err := json.Unmarshal(value, &date); err != nil {
						return Index{}, fmt.Errorf("documents[%d].%s must be an RFC 3339 date-time", i, field)
					}
				}
			}
		}
	}
	if wire.SchemaURL != SchemaURL {
		return Index{}, fmt.Errorf("$schema must be %q", SchemaURL)
	}
	if wire.Schema != Schema {
		return Index{}, fmt.Errorf("%w %q", ErrUnsupportedSchema, wire.Schema)
	}
	if wire.SchemaVersion != 1 {
		return Index{}, fmt.Errorf("%w %d", ErrUnsupportedVersion, wire.SchemaVersion)
	}
	if wire.Generator.Name == "" {
		return Index{}, fmt.Errorf("generator.name is required")
	}
	if wire.DocumentCount != len(wire.Documents) {
		return Index{}, fmt.Errorf("document_count %d does not match documents length %d", wire.DocumentCount, len(wire.Documents))
	}
	result := Index{Schema: wire.Schema, SchemaVersion: 1, Generator: Generator{wire.Generator.Name, wire.Generator.Version}, DocumentCount: wire.DocumentCount, Documents: make([]Document, len(wire.Documents))}
	if wire.Source.Commit != nil {
		result.Source.Commit = *wire.Source.Commit
	}
	seenPaths := make(map[string]struct{}, len(wire.Documents))
	for i, d := range wire.Documents {
		if d.Path == "" {
			return Index{}, fmt.Errorf("documents[%d].path is required", i)
		}
		if d.Slug == "" && d.Href == "" {
			return Index{}, fmt.Errorf("documents[%d] must have slug or href", i)
		}
		if _, exists := seenPaths[d.Path]; exists {
			return Index{}, fmt.Errorf("documents[%d].path is duplicated: %q", i, d.Path)
		}
		seenPaths[d.Path] = struct{}{}
		result.Documents[i] = Document{d.Path, d.Slug, d.Href, d.Title, d.TitleText, d.Date, d.Modified, d.Published, d.Draft, d.Private, d.Template, append([]string(nil), d.Tags...), d.Description, append([]string(nil), d.Feeds...), append([]string(nil), d.Aliases...)}
	}
	return result, nil
}

func isJSONNull(value []byte) bool { return bytes.Equal(bytes.TrimSpace(value), []byte("null")) }
