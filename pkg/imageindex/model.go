// Package imageindex defines the versioned Markata image inventory format and
// the build-time collector used by the generated image library.
package imageindex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
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
	Post  string
	Href  string
	Title string
	Cover bool
	Embed bool
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
	Alt        string     `json:"alt,omitempty"`
	MIMEType   string     `json:"mime_type,omitempty"`
	PosterSrc  string     `json:"poster_src,omitempty"`
	AddedAt    *time.Time `json:"added_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	Cover      bool       `json:"cover"`
	Embed      bool       `json:"embed,omitempty"`
	Uses       []wireUse  `json:"uses,omitempty"`
}

type wireUse struct {
	Post  string `json:"post"`
	Href  string `json:"href"`
	Title string `json:"title,omitempty"`
	Cover bool   `json:"cover"`
	Embed bool   `json:"embed,omitempty"`
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

// Parse decodes a supported image-index artifact and ignores unknown fields.
func Parse(data []byte) (Index, error) {
	var wire wireIndex
	if err := json.Unmarshal(data, &wire); err != nil {
		return Index{}, fmt.Errorf("decode image index JSON: %w", err)
	}

	var raw struct {
		SchemaURL     *string         `json:"$schema"`
		Schema        *string         `json:"schema"`
		SchemaVersion *int            `json:"schema_version"`
		Generator     json.RawMessage `json:"generator"`
		ImageCount    *int            `json:"image_count"`
		Images        json.RawMessage `json:"images"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Index{}, fmt.Errorf("validate image index structure: %w", err)
	}
	if raw.SchemaURL == nil || raw.Schema == nil || raw.SchemaVersion == nil || raw.Generator == nil || raw.ImageCount == nil || raw.Images == nil {
		return Index{}, fmt.Errorf("image index requires $schema, schema, schema_version, generator, image_count, and images")
	}
	if *raw.SchemaURL != SchemaURL {
		return Index{}, fmt.Errorf("unsupported image index $schema %q", *raw.SchemaURL)
	}
	if *raw.Schema != Schema {
		return Index{}, fmt.Errorf("unsupported image index schema %q", *raw.Schema)
	}
	if *raw.SchemaVersion > CurrentVersion {
		return Index{}, fmt.Errorf("image index schema version %d: %w", *raw.SchemaVersion, ErrUnsupportedVersion)
	}
	if *raw.SchemaVersion != CurrentVersion {
		return Index{}, fmt.Errorf("image index schema version %d: %w", *raw.SchemaVersion, ErrUnsupportedVersion)
	}

	var images []wireImage
	if err := json.Unmarshal(raw.Images, &images); err != nil {
		return Index{}, fmt.Errorf("images must be an array: %w", err)
	}
	if *raw.ImageCount != len(images) {
		return Index{}, fmt.Errorf("image_count %d does not match images length %d", *raw.ImageCount, len(images))
	}
	if trimmed := bytes.TrimSpace(raw.Images); len(trimmed) == 0 || trimmed[0] != '[' {
		return Index{}, fmt.Errorf("images must be an array")
	}

	index := Index{
		Schema:        wire.Schema,
		SchemaVersion: wire.SchemaVersion,
		Generator:     Generator{Name: wire.Generator.Name, Version: wire.Generator.Version},
		ImageCount:    wire.ImageCount,
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
		if image.LastUsedAt != nil {
			lastUsedAt := image.LastUsedAt.UTC()
			image.LastUsedAt = &lastUsedAt
		}

		image.Uses = append([]Use(nil), image.Uses...)
		sort.SliceStable(image.Uses, func(a, b int) bool {
			if image.Uses[a].Post != image.Uses[b].Post {
				return image.Uses[a].Post < image.Uses[b].Post
			}
			return image.Uses[a].Href < image.Uses[b].Href
		})
		for _, use := range image.Uses {
			if use.Post == "" && use.Href == "" {
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
