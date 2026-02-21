package plugins

import (
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestParseDateString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		// Standard formats
		{
			name:  "RFC3339",
			input: "2024-01-15T10:30:00Z",
			want:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:  "ISO datetime with T",
			input: "2024-01-15T10:30:00",
			want:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:  "datetime with space",
			input: "2024-01-15 10:30:00",
			want:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:  "date only",
			input: "2024-01-15",
			want:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},

		// Single-digit hours (issue #34)
		{
			name:  "single-digit hour",
			input: "2025-02-08 1:00:00",
			want:  time.Date(2025, 2, 8, 1, 0, 0, 0, time.UTC),
		},
		{
			name:  "single-digit hour with T",
			input: "2025-02-08T1:00:00",
			want:  time.Date(2025, 2, 8, 1, 0, 0, 0, time.UTC),
		},
		{
			name:  "single-digit hour 9am",
			input: "2024-06-20 9:30:00",
			want:  time.Date(2024, 6, 20, 9, 30, 0, 0, time.UTC),
		},

		// Malformed time components (issue #34)
		{
			name:  "malformed time with extra zero",
			input: "2025-07-14 8:011:00",
			want:  time.Date(2025, 7, 14, 8, 11, 0, 0, time.UTC),
		},
		{
			name:  "malformed time multiple extra zeros",
			input: "2025-07-14 08:001:030",
			want:  time.Date(2025, 7, 14, 8, 1, 30, 0, time.UTC),
		},

		// Date without time
		{
			name:  "date with slashes",
			input: "2024/01/15",
			want:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "US date format",
			input: "01/15/2024",
			want:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},

		// Named month formats
		{
			name:  "full month name",
			input: "January 15, 2024",
			want:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "abbreviated month name",
			input: "Jan 15, 2024",
			want:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "day first with full month",
			input: "15 January 2024",
			want:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},

		// Datetime with slashes
		{
			name:  "datetime with slashes",
			input: "2024/01/15 10:30:00",
			want:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},

		// Without seconds
		{
			name:  "datetime without seconds",
			input: "2024-01-15 10:30",
			want:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:  "datetime with T without seconds",
			input: "2024-01-15T10:30",
			want:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},

		// Whitespace handling
		{
			name:  "leading whitespace",
			input: "  2024-01-15",
			want:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "trailing whitespace",
			input: "2024-01-15  ",
			want:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},

		// Error cases
		{
			name:    "invalid date",
			input:   "not a date",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDateString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDateString(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("parseDateString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeDateString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "already normalized",
			input: "2024-01-15 10:30:00",
			want:  "2024-01-15 10:30:00",
		},
		{
			name:  "single-digit hour with space",
			input: "2024-01-15 1:30:00",
			want:  "2024-01-15 01:30:00",
		},
		{
			name:  "single-digit hour with T",
			input: "2024-01-15T1:30:00",
			want:  "2024-01-15T01:30:00",
		},
		{
			name:  "malformed minutes",
			input: "2024-01-15 8:011:00",
			want:  "2024-01-15 08:11:00",
		},
		{
			name:  "whitespace trimmed",
			input: "  2024-01-15 10:30:00  ",
			want:  "2024-01-15 10:30:00",
		},
		{
			name:  "no time component",
			input: "2024-01-15",
			want:  "2024-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeDateString(tt.input)
			if got != tt.want {
				t.Errorf("normalizeDateString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestApplyMetadata_AuthorAliases(t *testing.T) {
	p := &LoadPlugin{}

	tests := []struct {
		name       string
		metadata   map[string]interface{}
		wantAuthor string
	}{
		{
			name:       "author field takes priority",
			metadata:   map[string]interface{}{"author": "alice", "by": "bob", "writer": "charlie"},
			wantAuthor: "alice",
		},
		{
			name:       "by alias resolves to author",
			metadata:   map[string]interface{}{"by": "bob"},
			wantAuthor: "bob",
		},
		{
			name:       "writer alias resolves to author",
			metadata:   map[string]interface{}{"writer": "charlie"},
			wantAuthor: "charlie",
		},
		{
			name:       "by takes priority over writer",
			metadata:   map[string]interface{}{"by": "bob", "writer": "charlie"},
			wantAuthor: "bob",
		},
		{
			name:       "no author fields leaves author nil",
			metadata:   map[string]interface{}{"title": "some post"},
			wantAuthor: "",
		},
		{
			name:       "by and writer not stored in Extra",
			metadata:   map[string]interface{}{"by": "bob"},
			wantAuthor: "bob",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := &models.Post{}
			if err := p.applyMetadata(post, tt.metadata); err != nil {
				t.Fatalf("applyMetadata() error = %v", err)
			}

			if tt.wantAuthor == "" {
				if post.Author != nil {
					t.Errorf("expected Author to be nil, got %q", *post.Author)
				}
			} else {
				if post.Author == nil {
					t.Fatalf("expected Author to be %q, got nil", tt.wantAuthor)
				}
				if *post.Author != tt.wantAuthor {
					t.Errorf("Author = %q, want %q", *post.Author, tt.wantAuthor)
				}
			}

			// Verify aliases are not stored in Extra
			if tt.name == "by and writer not stored in Extra" {
				if val := post.Get("by"); val != nil {
					t.Errorf("'by' should not be in Extra, got %v", val)
				}
				if val := post.Get("writer"); val != nil {
					t.Errorf("'writer' should not be in Extra, got %v", val)
				}
			}
		})
	}
}

func TestApplyMetadata_DateAliases(t *testing.T) {
	p := &LoadPlugin{}

	tests := []struct {
		name        string
		metadata    map[string]interface{}
		wantDate    time.Time
		wantMod     time.Time
		wantErr     bool
		wantDateSet bool
		wantModSet  bool
	}{
		{
			name:        "publishdate wins over date and pubdate",
			metadata:    map[string]interface{}{"date": "2024-01-01", "publishdate": "2024-02-01", "pubdate": "2024-03-01"},
			wantDate:    time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			wantDateSet: true,
		},
		{
			name:        "date used when publishdate absent",
			metadata:    map[string]interface{}{"date": "2024-01-15"},
			wantDate:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			wantDateSet: true,
		},
		{
			name:        "pubdate used when others absent",
			metadata:    map[string]interface{}{"pubdate": "2024-03-10"},
			wantDate:    time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
			wantDateSet: true,
		},
		{
			name:       "lastmod wins over modified and updated",
			metadata:   map[string]interface{}{"modified": "2024-01-01", "lastmod": "2024-02-01", "updated": "2024-03-01"},
			wantMod:    time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			wantModSet: true,
		},
		{
			name:       "updated_at used when higher precedence absent",
			metadata:   map[string]interface{}{"updated_at": "2024-04-05"},
			wantMod:    time.Date(2024, 4, 5, 0, 0, 0, 0, time.UTC),
			wantModSet: true,
		},
		{
			name:     "invalid date returns error",
			metadata: map[string]interface{}{"date": "not-a-date"},
			wantErr:  true,
		},
		{
			name:     "invalid modified returns error",
			metadata: map[string]interface{}{"lastmod": "not-a-date"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post := &models.Post{}
			err := p.applyMetadata(post, tt.metadata)
			if (err != nil) != tt.wantErr {
				t.Fatalf("applyMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if tt.wantDateSet {
				if post.Date == nil {
					t.Fatalf("expected Date to be set")
				}
				if !post.Date.Equal(tt.wantDate) {
					t.Errorf("Date = %v, want %v", post.Date, tt.wantDate)
				}
			} else if post.Date != nil {
				t.Errorf("expected Date to be nil")
			}
			if tt.wantModSet {
				if post.Modified == nil {
					t.Fatalf("expected Modified to be set")
				}
				if !post.Modified.Equal(tt.wantMod) {
					t.Errorf("Modified = %v, want %v", post.Modified, tt.wantMod)
				}
			} else if post.Modified != nil {
				t.Errorf("expected Modified to be nil")
			}
		})
	}
}
