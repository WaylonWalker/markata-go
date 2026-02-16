package config

import (
	"fmt"
	"strconv"
	"strings"
)

// KeySegment represents a dot path segment with an optional array index.
type KeySegment struct {
	Key   string
	Index *int
}

// ParseKeyPath parses dotted paths with optional indexes, e.g. "feeds[0].title".
func ParseKeyPath(path string) ([]KeySegment, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("empty key path")
	}

	segments := make([]KeySegment, 0)
	var buf strings.Builder
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '.':
			if buf.Len() == 0 {
				return nil, fmt.Errorf("invalid key path: %q", path)
			}
			segments = append(segments, KeySegment{Key: buf.String()})
			buf.Reset()
		case '[':
			if buf.Len() == 0 {
				return nil, fmt.Errorf("invalid key path: %q", path)
			}
			end := strings.IndexByte(path[i:], ']')
			if end == -1 {
				return nil, fmt.Errorf("invalid key path: %q", path)
			}
			indexText := path[i+1 : i+end]
			idx, err := strconv.Atoi(indexText)
			if err != nil {
				return nil, fmt.Errorf("invalid index %q", indexText)
			}
			segments = append(segments, KeySegment{Key: buf.String(), Index: &idx})
			buf.Reset()
			i += end
		default:
			buf.WriteByte(path[i])
		}
	}
	if buf.Len() > 0 {
		segments = append(segments, KeySegment{Key: buf.String()})
	}

	return segments, nil
}

func keySegmentsEqual(a, b []KeySegment) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !strings.EqualFold(a[i].Key, b[i].Key) {
			return false
		}
		if (a[i].Index == nil) != (b[i].Index == nil) {
			return false
		}
		if a[i].Index != nil && b[i].Index != nil && *a[i].Index != *b[i].Index {
			return false
		}
	}
	return true
}
