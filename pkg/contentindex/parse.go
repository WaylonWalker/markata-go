package contentindex

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Parse decodes any released Content Index generation into the current model.
// Unknown object fields are ignored. Future generations are rejected rather
// than being silently interpreted as an older format.
func Parse(data []byte) (Index, error) {
	var envelope struct {
		Schema        *string `json:"schema"`
		SchemaVersion *int    `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Index{}, fmt.Errorf("malformed content index JSON: %w", err)
	}
	if envelope.Schema == nil || *envelope.Schema == "" {
		return Index{}, ErrMissingSchema
	}
	if *envelope.Schema != Schema {
		return Index{}, fmt.Errorf("%w %q", ErrUnsupportedSchema, *envelope.Schema)
	}
	if envelope.SchemaVersion == nil {
		return Index{}, ErrMissingVersion
	}
	switch *envelope.SchemaVersion {
	case 1:
		return decodeV1(data)
	default:
		return Index{}, fmt.Errorf("%w %d (latest supported: %d)", ErrUnsupportedVersion, *envelope.SchemaVersion, CurrentVersion)
	}
}

// Marshal emits the newest supported compact JSON representation.
func Marshal(index Index) ([]byte, error) {
	if index.Schema != "" && index.Schema != Schema {
		return nil, fmt.Errorf("%w %q", ErrUnsupportedSchema, index.Schema)
	}
	if index.SchemaVersion != 0 && index.SchemaVersion != CurrentVersion {
		return nil, fmt.Errorf("%w %d", ErrUnsupportedVersion, index.SchemaVersion)
	}
	return encodeV1(index)
}

// ValidJSON reports whether data is a valid supported Content Index artifact.
func ValidJSON(data []byte) bool {
	return json.Valid(bytes.TrimSpace(data)) && func() bool { _, err := Parse(data); return err == nil }()
}
