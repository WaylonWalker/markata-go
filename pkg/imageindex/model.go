// Package imageindex defines the versioned Markata image inventory format and
// the build-time collector used by the generated image library.
package imageindex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	// Schema is the stable logical identity of the image index.
	Schema = "markata.image-index"
	// SchemaURL identifies the released wire-format generation.
	SchemaURL = "markata://schemas/image-index/v1"
	// CurrentVersion is the newest image-index generation emitted by this package.
	CurrentVersion = 1
	// GeneratorName is the producer name written to generated artifacts.
	GeneratorName = "markata-go"
)

// Index is the normalized internal representation of an image index.
type Index struct {
	Schema        string
	SchemaVersion int
	Generator     Generator
	ImageCount    int
	Images        []Image
}

// Generator identifies the program that produced an index.
type Generator struct {
	Name    string
	Version string
}

// Image describes one deduplicated canonical image or video source.
type Image struct {
	Src        string
	Width      int
	Height     int
	Alt        string
	MIMEType   string
	PosterSrc  string
	AddedAt    *time.Time
	LastUsedAt *time.Time
	Cover      bool
	Embed      bool
	Uses       []Use
}

// Use describes one public post relationship for a media source.
type Use struct {
	Href    string
	Title   string
	Caption string
	Cover   bool
	Embed   bool
}

type wireIndex struct {
	SchemaURL     string        `json:"$schema"`
	Schema        string        `json:"schema"`
	SchemaVersion int           `json:"schema_version"`
	Generator     wireGenerator `json:"generator"`
	ImageCount    int           `json:"image_count"`
	Images        []wireImage   `json:"images"`
}

type wireGenerator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type wireImage struct {
	Src        string     `json:"src"`
	Width      int        `json:"width"`
	Height     int        `json:"height"`
	Alt        string     `json:"alt"`
	MIMEType   string     `json:"mime_type"`
	PosterSrc  string     `json:"poster_src,omitempty"`
	AddedAt    *time.Time `json:"added_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	Cover      bool       `json:"cover"`
	Embed      bool       `json:"embed,omitempty"`
	Uses       []wireUse  `json:"uses"`
}

type wireUse struct {
	Href    string `json:"href"`
	Title   string `json:"title,omitempty"`
	Caption string `json:"caption,omitempty"`
	Cover   bool   `json:"cover"`
	Embed   bool   `json:"embed,omitempty"`
}

// Marshal encodes an image index as deterministic, compact JSON.
func Marshal(index Index) ([]byte, error) {
	normalized, err := normalize(index)
	if err != nil {
		return nil, err
	}

	wireImages := make([]wireImage, len(normalized.Images))
	for i := range normalized.Images {
		image := &normalized.Images[i]
		wireUses := make([]wireUse, len(image.Uses))
		for j, use := range image.Uses {
			wireUses[j] = wireUse(use)
		}
		wireImages[i] = wireImage{
			Src:        image.Src,
			Width:      image.Width,
			Height:     image.Height,
			Alt:        image.Alt,
			MIMEType:   image.MIMEType,
			PosterSrc:  image.PosterSrc,
			AddedAt:    image.AddedAt,
			LastUsedAt: image.LastUsedAt,
			Cover:      image.Cover,
			Embed:      image.Embed,
			Uses:       wireUses,
		}
	}

	return json.Marshal(wireIndex{
		SchemaURL:     SchemaURL,
		Schema:        normalized.Schema,
		SchemaVersion: normalized.SchemaVersion,
		Generator:     wireGenerator{Name: normalized.Generator.Name, Version: normalized.Generator.Version},
		ImageCount:    normalized.ImageCount,
		Images:        wireImages,
	})
}

// Parse strictly decodes a supported v1 image-index artifact and ignores
// unknown fields. Required fields are checked for both presence and the JSON
// types declared by the v1 schema.
func Parse(data []byte) (Index, error) {
	fields, err := imageIndexJSONObject(data, "image index")
	if err != nil {
		return Index{}, fmt.Errorf("decode image index JSON: %w", err)
	}
	envelope, err := parseV1Envelope(fields)
	if err != nil {
		return Index{}, err
	}
	images, err := parseV1Images(envelope.rawImages, envelope.imageCount)
	if err != nil {
		return Index{}, err
	}

	index := Index{
		Schema:        envelope.schema,
		SchemaVersion: envelope.schemaVersion,
		Generator:     envelope.generator,
		ImageCount:    envelope.imageCount,
		Images:        make([]Image, len(images)),
	}
	for i := range images {
		image := &images[i]
		index.Images[i] = Image{
			Src:        image.Src,
			Width:      image.Width,
			Height:     image.Height,
			Alt:        image.Alt,
			MIMEType:   image.MIMEType,
			PosterSrc:  image.PosterSrc,
			AddedAt:    image.AddedAt,
			LastUsedAt: image.LastUsedAt,
			Cover:      image.Cover,
			Embed:      image.Embed,
			Uses:       make([]Use, len(image.Uses)),
		}
		for j, use := range image.Uses {
			index.Images[i].Uses[j] = Use(use)
		}
	}

	return normalize(index)
}

type v1Envelope struct {
	schema        string
	schemaVersion int
	generator     Generator
	imageCount    int
	rawImages     json.RawMessage
}

func parseV1Envelope(fields map[string]json.RawMessage) (v1Envelope, error) {
	var envelope v1Envelope
	var schemaURL string
	required := []struct {
		name        string
		destination interface{}
	}{
		{name: "$schema", destination: &schemaURL},
		{name: "schema", destination: &envelope.schema},
		{name: "schema_version", destination: &envelope.schemaVersion},
		{name: "image_count", destination: &envelope.imageCount},
	}
	for _, field := range required {
		if err := decodeRequiredJSONField(fields, field.name, "image index", field.destination); err != nil {
			return v1Envelope{}, err
		}
	}
	if schemaURL != SchemaURL {
		return v1Envelope{}, fmt.Errorf("unsupported image index $schema %q", schemaURL)
	}
	if envelope.schema != Schema {
		return v1Envelope{}, fmt.Errorf("unsupported image index schema %q", envelope.schema)
	}
	if envelope.schemaVersion != CurrentVersion {
		return v1Envelope{}, fmt.Errorf("image index schema version %d: %w", envelope.schemaVersion, ErrUnsupportedVersion)
	}
	if envelope.imageCount < 0 {
		return v1Envelope{}, fmt.Errorf("image_count %d cannot be negative", envelope.imageCount)
	}
	generator, err := parseV1Generator(fields)
	if err != nil {
		return v1Envelope{}, err
	}
	envelope.generator = generator
	envelope.rawImages, err = requiredJSONField(fields, "images", "image index")
	if err != nil {
		return v1Envelope{}, err
	}
	if trimmed := bytes.TrimSpace(envelope.rawImages); len(trimmed) == 0 || trimmed[0] != '[' {
		return v1Envelope{}, fmt.Errorf("images must be an array")
	}
	return envelope, nil
}

func parseV1Generator(fields map[string]json.RawMessage) (Generator, error) {
	generatorFields, err := decodeJSONObjectField(fields, "generator", "image index")
	if err != nil {
		return Generator{}, err
	}
	var generator Generator
	for _, field := range []struct {
		name        string
		destination interface{}
	}{
		{name: "name", destination: &generator.Name},
		{name: "version", destination: &generator.Version},
	} {
		if err := decodeRequiredJSONField(generatorFields, field.name, "image index.generator", field.destination); err != nil {
			return Generator{}, err
		}
	}
	if generator.Name == "" {
		return Generator{}, fmt.Errorf("image index generator.name is required")
	}
	return generator, nil
}

func parseV1Images(rawImages json.RawMessage, imageCount int) ([]wireImage, error) {
	var imageValues []json.RawMessage
	if err := json.Unmarshal(rawImages, &imageValues); err != nil {
		return nil, fmt.Errorf("images must be an array: %w", err)
	}
	if imageCount != len(imageValues) {
		return nil, fmt.Errorf("image_count %d does not match images length %d", imageCount, len(imageValues))
	}
	images := make([]wireImage, len(imageValues))
	for i, rawImage := range imageValues {
		if err := validateV1ImageJSON(rawImage, i); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rawImage, &images[i]); err != nil {
			return nil, fmt.Errorf("images[%d] has an invalid value: %w", i, err)
		}
	}
	return images, nil
}

func imageIndexJSONObject(data []byte, context string) (map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, fmt.Errorf("%s must be an object", context)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, fmt.Errorf("%s must be an object", context)
	}
	return fields, nil
}

func requiredJSONField(fields map[string]json.RawMessage, name, context string) (json.RawMessage, error) {
	value, ok := fields[name]
	if !ok || isJSONNull(value) {
		return nil, fmt.Errorf("%s.%s is required", context, name)
	}
	return value, nil
}

func decodeRequiredJSONField(fields map[string]json.RawMessage, name, context string, destination interface{}) error {
	value, err := requiredJSONField(fields, name, context)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(value, destination); err != nil {
		return fmt.Errorf("%s.%s has an invalid value: %w", context, name, err)
	}
	return nil
}

func decodeOptionalJSONField(fields map[string]json.RawMessage, name, context string, destination interface{}) error {
	value, ok := fields[name]
	if !ok {
		return nil
	}
	if isJSONNull(value) {
		return fmt.Errorf("%s.%s must not be null", context, name)
	}
	if err := json.Unmarshal(value, destination); err != nil {
		return fmt.Errorf("%s.%s has an invalid value: %w", context, name, err)
	}
	return nil
}

func decodeJSONObjectField(fields map[string]json.RawMessage, name, context string) (map[string]json.RawMessage, error) {
	value, err := requiredJSONField(fields, name, context)
	if err != nil {
		return nil, err
	}
	return imageIndexJSONObject(value, context+"."+name)
}

func isJSONNull(value json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}

func validateV1ImageJSON(data json.RawMessage, imageIndex int) error {
	context := fmt.Sprintf("images[%d]", imageIndex)
	fields, err := imageIndexJSONObject(data, context)
	if err != nil {
		return err
	}

	var src string
	var width int
	var height int
	var alt string
	var mimeType string
	var cover bool
	for _, field := range []struct {
		name        string
		destination interface{}
	}{
		{name: "src", destination: &src},
		{name: "width", destination: &width},
		{name: "height", destination: &height},
		{name: "alt", destination: &alt},
		{name: "mime_type", destination: &mimeType},
		{name: "cover", destination: &cover},
	} {
		if err := decodeRequiredJSONField(fields, field.name, context, field.destination); err != nil {
			return err
		}
	}
	if src == "" {
		return fmt.Errorf("%s.src is required", context)
	}
	if width < 0 {
		return fmt.Errorf("%s.width cannot be negative", context)
	}
	if height < 0 {
		return fmt.Errorf("%s.height cannot be negative", context)
	}

	var posterSrc string
	var addedAt time.Time
	var lastUsedAt time.Time
	var embed bool
	for _, field := range []struct {
		name        string
		destination interface{}
	}{
		{name: "poster_src", destination: &posterSrc},
		{name: "added_at", destination: &addedAt},
		{name: "last_used_at", destination: &lastUsedAt},
		{name: "embed", destination: &embed},
	} {
		if err := decodeOptionalJSONField(fields, field.name, context, field.destination); err != nil {
			return err
		}
	}

	rawUses, err := requiredJSONField(fields, "uses", context)
	if err != nil {
		return err
	}
	trimmedUses := bytes.TrimSpace(rawUses)
	if len(trimmedUses) == 0 || trimmedUses[0] != '[' {
		return fmt.Errorf("%s.uses must be an array", context)
	}
	var uses []json.RawMessage
	if err := json.Unmarshal(rawUses, &uses); err != nil {
		return fmt.Errorf("%s.uses must be an array: %w", context, err)
	}
	for useIndex, rawUse := range uses {
		if err := validateV1UseJSON(rawUse, imageIndex, useIndex); err != nil {
			return err
		}
	}
	return nil
}

func validateV1UseJSON(data json.RawMessage, imageIndex, useIndex int) error {
	context := fmt.Sprintf("images[%d].uses[%d]", imageIndex, useIndex)
	fields, err := imageIndexJSONObject(data, context)
	if err != nil {
		return err
	}
	var href string
	if err := decodeRequiredJSONField(fields, "href", context, &href); err != nil {
		return err
	}
	if href == "" {
		return fmt.Errorf("%s.href is required", context)
	}
	var cover bool
	if err := decodeRequiredJSONField(fields, "cover", context, &cover); err != nil {
		return err
	}
	var title string
	if err := decodeOptionalJSONField(fields, "title", context, &title); err != nil {
		return err
	}
	var caption string
	if err := decodeOptionalJSONField(fields, "caption", context, &caption); err != nil {
		return err
	}
	var embed bool
	return decodeOptionalJSONField(fields, "embed", context, &embed)
}

func normalize(index Index) (Index, error) {
	if index.Schema == "" {
		index.Schema = Schema
	}
	if index.Schema != Schema {
		return Index{}, fmt.Errorf("unsupported image index schema %q", index.Schema)
	}
	if index.SchemaVersion == 0 {
		index.SchemaVersion = CurrentVersion
	}
	if index.SchemaVersion != CurrentVersion {
		return Index{}, fmt.Errorf("image index schema version %d: %w", index.SchemaVersion, ErrUnsupportedVersion)
	}
	if index.Generator.Name == "" {
		return Index{}, fmt.Errorf("image index generator.name is required")
	}
	if index.ImageCount != 0 && index.ImageCount != len(index.Images) {
		return Index{}, fmt.Errorf("image_count %d does not match images length %d", index.ImageCount, len(index.Images))
	}

	index.Images = append([]Image(nil), index.Images...)
	sort.SliceStable(index.Images, func(i, j int) bool {
		return index.Images[i].Src < index.Images[j].Src
	})
	seen := make(map[string]struct{}, len(index.Images))
	for i := range index.Images {
		image := &index.Images[i]
		if image.Src == "" {
			return Index{}, fmt.Errorf("images[%d].src is required", i)
		}
		if image.Width < 0 || image.Height < 0 {
			return Index{}, fmt.Errorf("images[%d] dimensions cannot be negative", i)
		}
		if _, exists := seen[image.Src]; exists {
			return Index{}, fmt.Errorf("images[%d].src is duplicated: %q", i, image.Src)
		}
		seen[image.Src] = struct{}{}
		normalizeImageDates(image)

		if image.Uses == nil {
			image.Uses = make([]Use, 0)
		} else {
			uses := make([]Use, len(image.Uses))
			copy(uses, image.Uses)
			image.Uses = uses
		}
		sort.SliceStable(image.Uses, func(a, b int) bool {
			return image.Uses[a].Href < image.Uses[b].Href
		})
		for useIndex := range image.Uses {
			use := &image.Uses[useIndex]
			use.Caption = strings.TrimSpace(use.Caption)
			if use.Href == "" {
				return Index{}, fmt.Errorf("images[%d].uses contains an empty relationship", i)
			}
			if use.Cover {
				image.Cover = true
			}
			if use.Embed {
				image.Embed = true
			}
		}
	}
	index.ImageCount = len(index.Images)
	return index, nil
}

func normalizeImageDates(image *Image) {
	if image.AddedAt != nil {
		addedAt := image.AddedAt.UTC()
		image.AddedAt = &addedAt
	}
	if image.LastUsedAt != nil {
		lastUsedAt := image.LastUsedAt.UTC()
		image.LastUsedAt = &lastUsedAt
	}
}
