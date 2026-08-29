package contentindex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"
)

type v1Index struct {
	SchemaURL     string       `json:"$schema"`
	Schema        string       `json:"schema"`
	SchemaVersion json.Number  `json:"schema_version"`
	Scope         string       `json:"scope"`
	Generator     v1Generator  `json:"generator"`
	Source        v1Source     `json:"source"`
	DocumentCount json.Number  `json:"document_count"`
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
	Dirty  *bool   `json:"dirty,omitempty"`
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
	Image       *string    `json:"image,omitempty"`
	Video       *string    `json:"video,omitempty"`
	Avatar      *string    `json:"avatar,omitempty"`
	Bio         *string    `json:"bio,omitempty"`
	Thumbnail   *string    `json:"thumbnail,omitempty"`
	Cover       *string    `json:"cover,omitempty"`
	OGImage     *string    `json:"og_image,omitempty"`
	Author      *string    `json:"author,omitempty"`
	Authors     []string   `json:"authors,omitempty"`
	Category    *string    `json:"category,omitempty"`
	Categories  []string   `json:"categories,omitempty"`
}

var privateV2ForbiddenFields = [...]string{"image", "video", "bio", "thumbnail", "cover", "og_image"}

func encodeV1(index Index) ([]byte, error) {
	return encodeVersion(index, 1, SchemaURL, false)
}

func encodeV2(index Index) ([]byte, error) {
	return encodeVersion(index, 2, V2SchemaURL, true)
}

//nolint:gocyclo // Versioned wire-format validation is intentionally isolated.
func encodeVersion(index Index, version int, schemaURL string, allowPrivate bool) ([]byte, error) {
	if index.Schema == "" {
		index.Schema = Schema
	}
	if index.SchemaVersion == 0 {
		index.SchemaVersion = version
	}
	if index.SchemaVersion != version {
		return nil, fmt.Errorf("schema version %d encoder received index version %d", version, index.SchemaVersion)
	}
	if index.Scope == "" {
		if allowPrivate {
			index.Scope = PublicMetadataScope
		} else {
			index.Scope = PublicScope
		}
	}
	if index.Scope != PublicScope && (index.Scope != PublicMetadataScope || !allowPrivate) {
		return nil, fmt.Errorf("unsupported scope %q", index.Scope)
	}
	if index.DocumentCount != 0 && index.DocumentCount != len(index.Documents) {
		return nil, fmt.Errorf("document_count %d does not match documents length %d", index.DocumentCount, len(index.Documents))
	}
	index.DocumentCount = len(index.Documents)
	for i := range index.Documents {
		if allowPrivate && index.Documents[i].Private && index.Scope == PublicScope {
			return nil, fmt.Errorf("scope %q cannot contain private documents", PublicScope)
		}
		if allowPrivate && index.Documents[i].Private {
			if field := forbiddenPrivateDocumentField(index.Documents[i]); field != "" {
				return nil, fmt.Errorf("documents[%d].%s is forbidden for private v2 documents", i, field)
			}
		}
	}
	if index.Generator.Name == "" {
		return nil, fmt.Errorf("generator.name is required")
	}
	seenPaths := make(map[string]struct{}, len(index.Documents))
	sort.SliceStable(index.Documents, func(i, j int) bool { return index.Documents[i].Path < index.Documents[j].Path })
	docs := make([]v1Document, len(index.Documents))
	for i := range index.Documents {
		d := &index.Documents[i]
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
		docs[i] = v1Document{Path: d.Path, Slug: d.Slug, Href: d.Href, Title: d.Title, TitleText: d.TitleText, Date: d.Date, Modified: d.Modified, Published: d.Published, Draft: d.Draft, Private: d.Private, Template: d.Template, Tags: d.Tags, Description: d.Description, Feeds: d.Feeds, Aliases: d.Aliases, Image: d.Image, Video: d.Video, Avatar: d.Avatar, Bio: d.Bio, Thumbnail: d.Thumbnail, Cover: d.Cover, OGImage: d.OGImage, Author: d.Author, Authors: append([]string(nil), d.Authors...), Category: d.Category, Categories: sortedStrings(d.Categories)}
	}
	var commit *string
	if index.Source.Commit != "" {
		commit = &index.Source.Commit
	}
	if index.Source.Dirty != nil && index.Source.Commit == "" {
		return nil, fmt.Errorf("source.dirty requires source.commit")
	}
	if index.Source.Commit != "" && index.Source.Dirty == nil {
		return nil, fmt.Errorf("source.commit requires source.dirty")
	}
	return json.Marshal(v1Index{schemaURL, index.Schema, json.Number(strconv.Itoa(index.SchemaVersion)), index.Scope, v1Generator{index.Generator.Name, index.Generator.Version}, v1Source{commit, index.Source.Dirty}, json.Number(strconv.Itoa(index.DocumentCount)), docs})
}

func decodeV1(data []byte) (Index, error) {
	return decodeVersion(data, 1, SchemaURL, false)
}

func decodeV2(data []byte) (Index, error) {
	return decodeVersion(data, 2, V2SchemaURL, true)
}

//nolint:gocyclo // Versioned wire-format validation is intentionally isolated.
func decodeVersion(data []byte, expectedVersion int, schemaURL string, allowPrivate bool) (Index, error) {
	var wire v1Index
	if err := json.Unmarshal(data, &wire); err != nil {
		return Index{}, fmt.Errorf("decode content index v%d JSON: %w", expectedVersion, err)
	}
	var raw struct {
		SchemaVersion json.RawMessage              `json:"schema_version"`
		Generator     map[string]json.RawMessage   `json:"generator"`
		Source        json.RawMessage              `json:"source"`
		DocumentCount json.RawMessage              `json:"document_count"`
		Scope         *string                      `json:"scope"`
		Documents     []map[string]json.RawMessage `json:"documents"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Index{}, fmt.Errorf("validate content index v%d structure: %w", expectedVersion, err)
	}
	if raw.Generator == nil || raw.Source == nil || raw.Scope == nil || raw.DocumentCount == nil || raw.Documents == nil {
		return Index{}, fmt.Errorf("content index v%d requires generator, source, scope, document_count, and documents", expectedVersion)
	}
	if string(raw.Source) == "null" || string(raw.Source) == "" {
		return Index{}, fmt.Errorf("source must be an object")
	}
	if !isJSONNumber(raw.SchemaVersion) || !isJSONNumber(raw.DocumentCount) {
		return Index{}, fmt.Errorf("schema_version and document_count must be JSON numbers")
	}
	var sourceFields map[string]json.RawMessage
	if err := json.Unmarshal(raw.Source, &sourceFields); err != nil {
		return Index{}, fmt.Errorf("source must be an object")
	}
	if commit, ok := sourceFields["commit"]; ok {
		if _, hasDirty := sourceFields["dirty"]; !hasDirty {
			return Index{}, fmt.Errorf("source.commit requires source.dirty")
		}
		var value string
		if isJSONNull(commit) || json.Unmarshal(commit, &value) != nil {
			return Index{}, fmt.Errorf("source.commit must be a string")
		}
		if value == "" {
			return Index{}, fmt.Errorf("source.commit must not be empty")
		}
	}
	if dirty, ok := sourceFields["dirty"]; ok {
		if _, ok := sourceFields["commit"]; !ok {
			return Index{}, fmt.Errorf("source.dirty requires source.commit")
		}
		var value bool
		if isJSONNull(dirty) || json.Unmarshal(dirty, &value) != nil {
			return Index{}, fmt.Errorf("source.dirty must be a boolean")
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
		if err := validatePrivateDocumentWire(i, document, allowPrivate); err != nil {
			return Index{}, err
		}
		for _, field := range []string{"tags", "feeds", "aliases", "authors", "categories"} {
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
		for _, field := range []string{"date", "modified", "template", "image", "video", "avatar", "bio", "thumbnail", "cover", "og_image", "author", "category"} {
			if value, ok := document[field]; ok {
				if isJSONNull(value) {
					return Index{}, fmt.Errorf("documents[%d].%s must be a string", i, field)
				}
				if field != "date" && field != "modified" {
					var text string
					if err := json.Unmarshal(value, &text); err != nil {
						return Index{}, fmt.Errorf("documents[%d].%s must be a string", i, field)
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
	if wire.SchemaURL != schemaURL {
		return Index{}, fmt.Errorf("$schema must be %q", schemaURL)
	}
	if wire.Schema != Schema {
		return Index{}, fmt.Errorf("%w %q", ErrUnsupportedSchema, wire.Schema)
	}
	version, err := jsonInteger(json.Number(string(raw.SchemaVersion)))
	if err != nil {
		return Index{}, fmt.Errorf("schema_version must be an integer: %w", err)
	}
	if version != expectedVersion {
		return Index{}, fmt.Errorf("%w %d", ErrUnsupportedVersion, version)
	}
	if wire.Scope == "" {
		return Index{}, fmt.Errorf("scope is required")
	}
	if allowPrivate && wire.Scope != PublicScope && wire.Scope != PublicMetadataScope {
		return Index{}, fmt.Errorf("unsupported scope %q for content index v2", wire.Scope)
	}
	if allowPrivate && wire.Scope != PublicMetadataScope {
		for i := range wire.Documents {
			document := &wire.Documents[i]
			if document.Private {
				return Index{}, fmt.Errorf("documents[%d].private requires scope %q", i, PublicMetadataScope)
			}
		}
	}
	if wire.Generator.Name == "" {
		return Index{}, fmt.Errorf("generator.name is required")
	}
	documentCount, err := jsonInteger(json.Number(string(raw.DocumentCount)))
	if err != nil {
		return Index{}, fmt.Errorf("document_count must be an integer: %w", err)
	}
	if documentCount != len(wire.Documents) {
		return Index{}, fmt.Errorf("document_count %d does not match documents length %d", documentCount, len(wire.Documents))
	}
	result := Index{Schema: wire.Schema, SchemaVersion: version, Scope: wire.Scope, Generator: Generator{wire.Generator.Name, wire.Generator.Version}, DocumentCount: documentCount, Documents: make([]Document, len(wire.Documents))}
	if wire.Source.Commit != nil {
		result.Source.Commit = *wire.Source.Commit
	}
	result.Source.Dirty = wire.Source.Dirty
	seenPaths := make(map[string]struct{}, len(wire.Documents))
	for i := range wire.Documents {
		d := &wire.Documents[i]
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
		result.Documents[i] = Document{Path: d.Path, Slug: d.Slug, Href: d.Href, Title: d.Title, TitleText: d.TitleText, Date: d.Date, Modified: d.Modified, Published: d.Published, Draft: d.Draft, Private: d.Private, Template: d.Template, Tags: append([]string(nil), d.Tags...), Description: d.Description, Feeds: append([]string(nil), d.Feeds...), Aliases: append([]string(nil), d.Aliases...), Image: d.Image, Video: d.Video, Avatar: d.Avatar, Bio: d.Bio, Thumbnail: d.Thumbnail, Cover: d.Cover, OGImage: d.OGImage, Author: d.Author, Authors: append([]string(nil), d.Authors...), Category: d.Category, Categories: append([]string(nil), d.Categories...)}
	}
	return result, nil
}

func forbiddenPrivateDocumentField(document Document) string {
	fields := map[string]*string{
		"image":     document.Image,
		"video":     document.Video,
		"bio":       document.Bio,
		"thumbnail": document.Thumbnail,
		"cover":     document.Cover,
		"og_image":  document.OGImage,
	}
	for _, field := range privateV2ForbiddenFields {
		if fields[field] != nil {
			return field
		}
	}
	return ""
}

func validatePrivateDocumentWire(index int, document map[string]json.RawMessage, allowPrivate bool) error {
	if !allowPrivate {
		return nil
	}
	private, ok := document["private"]
	if !ok {
		return nil
	}
	var isPrivate bool
	if err := json.Unmarshal(private, &isPrivate); err != nil || !isPrivate {
		return nil
	}
	for _, field := range privateV2ForbiddenFields {
		if _, ok := document[field]; ok {
			return fmt.Errorf("documents[%d].%s is forbidden for private v2 documents", index, field)
		}
	}
	return nil
}

func isJSONNull(value []byte) bool { return bytes.Equal(bytes.TrimSpace(value), []byte("null")) }
