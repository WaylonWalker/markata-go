package buildstats

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	maxSpanNameBytes       = 128
	maxSpanAttributes      = 16
	maxSpanAttributeKey    = 64
	maxSpanAttributeValue  = 256
	defaultUnnamedSpanName = "unnamed"
)

// Attribute is a bounded, sanitized key/value annotation attached to a build span.
type Attribute struct {
	Key   string
	Value string
}

// StringAttribute creates a string-valued span attribute.
func StringAttribute(key, value string) Attribute {
	return Attribute{Key: key, Value: value}
}

// SpanTiming is one completed operation in a build trace. StartOffset is
// relative to the beginning of the build profile so traces remain portable.
type SpanTiming struct {
	ID         string            `json:"id"`
	ParentID   string            `json:"parent_id,omitempty"`
	Name       string            `json:"name"`
	Stage      string            `json:"stage,omitempty"`
	Plugin     string            `json:"plugin,omitempty"`
	StartOffset time.Duration     `json:"start_offset"`
	Duration   time.Duration     `json:"duration"`
	Status     string            `json:"status"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Span represents one in-flight build operation.
type Span struct {
	profile    *Profile
	id         string
	parentID   string
	name       string
	stage      string
	plugin     string
	startedAt  time.Time
	attributes map[string]string
	once       sync.Once
}

type spanContextKey struct{}

type spanContextValue struct {
	profile *Profile
	id      string
}

// StartSpan begins a nested operation for the active build profile. When no
// build profile is active it returns a no-op span, keeping instrumentation cheap
// and safe to leave in normal code paths.
func StartSpan(ctx context.Context, name string, attributes ...Attribute) (context.Context, *Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	profile := activeProfile.Load()
	if profile == nil {
		return ctx, &Span{}
	}

	name = sanitizeSpanName(name)
	startedAt := time.Now()
	parentID := ""
	if parent, ok := ctx.Value(spanContextKey{}).(spanContextValue); ok && parent.profile == profile {
		parentID = parent.id
	}

	profile.mu.Lock()
	stage := profile.current
	plugin := profile.plugin
	profile.mu.Unlock()

	id := fmt.Sprintf("span-%06d", profile.spanSeq.Add(1))
	span := &Span{
		profile:    profile,
		id:         id,
		parentID:   parentID,
		name:       name,
		stage:      stage,
		plugin:     plugin,
		startedAt:  startedAt,
		attributes: sanitizeSpanAttributes(attributes),
	}
	return context.WithValue(ctx, spanContextKey{}, spanContextValue{profile: profile, id: id}), span
}

// End records a successful span completion.
func (s *Span) End() {
	s.finish("ok")
}

// EndError records an errored span completion without retaining the error text,
// which could contain credentials, query strings, or sensitive filesystem data.
func (s *Span) EndError(err error) {
	if err == nil {
		s.finish("ok")
		return
	}
	s.finish("error")
}

func (s *Span) finish(status string) {
	if s == nil || s.profile == nil {
		return
	}
	s.once.Do(func() {
		finishedAt := time.Now()
		startOffset := s.startedAt.Sub(s.profile.start)
		if startOffset < 0 {
			startOffset = 0
		}
		timing := SpanTiming{
			ID:          s.id,
			ParentID:    s.parentID,
			Name:        s.name,
			Stage:       s.stage,
			Plugin:      s.plugin,
			StartOffset: startOffset,
			Duration:    finishedAt.Sub(s.startedAt),
			Status:      status,
			Attributes:  cloneSpanAttributes(s.attributes),
		}
		s.profile.mu.Lock()
		s.profile.spans = append(s.profile.spans, timing)
		s.profile.mu.Unlock()
	})
}

func sanitizeSpanName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultUnnamedSpanName
	}
	return truncateBytes(name, maxSpanNameBytes)
}

func sanitizeSpanAttributes(attributes []Attribute) map[string]string {
	if len(attributes) == 0 {
		return nil
	}
	result := make(map[string]string, min(len(attributes), maxSpanAttributes))
	for _, attribute := range attributes {
		if len(result) >= maxSpanAttributes {
			break
		}
		key := strings.TrimSpace(attribute.Key)
		if key == "" || sensitiveSpanAttribute(key) {
			continue
		}
		key = truncateBytes(key, maxSpanAttributeKey)
		result[key] = truncateBytes(strings.TrimSpace(attribute.Value), maxSpanAttributeValue)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func sensitiveSpanAttribute(key string) bool {
	key = strings.ToLower(key)
	for _, blocked := range []string{"authorization", "cookie", "password", "passwd", "secret", "token", "credential"} {
		if strings.Contains(key, blocked) {
			return true
		}
	}
	return false
}

func cloneSpanAttributes(attributes map[string]string) map[string]string {
	if len(attributes) == 0 {
		return nil
	}
	clone := make(map[string]string, len(attributes))
	for key, value := range attributes {
		clone[key] = value
	}
	return clone
}

func truncateBytes(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
