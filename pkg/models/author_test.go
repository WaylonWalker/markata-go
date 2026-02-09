package models

import (
	"testing"
)

func TestAuthor_Validate(t *testing.T) {
	tests := []struct {
		name    string
		author  Author
		wantErr bool
	}{
		{
			name: "valid author with required fields",
			author: Author{
				ID:   "john-doe",
				Name: "John Doe",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			author: Author{
				Name: "John Doe",
			},
			wantErr: true,
		},
		{
			name: "missing Name",
			author: Author{
				ID: "john-doe",
			},
			wantErr: true,
		},
		{
			name: "valid CReDiT contributions",
			author: Author{
				ID:            "john-doe",
				Name:          "John Doe",
				Contributions: []string{"conceptualization", "writing-original-draft"},
			},
			wantErr: false,
		},
		{
			name: "invalid CReDiT contribution",
			author: Author{
				ID:            "john-doe",
				Name:          "John Doe",
				Contributions: []string{"invalid-role"},
			},
			wantErr: true,
		},
		{
			name: "valid simple role",
			author: Author{
				ID:   "john-doe",
				Name: "John Doe",
				Role: authorTestStrPtr("author"),
			},
			wantErr: false,
		},
		{
			name: "invalid simple role",
			author: Author{
				ID:   "john-doe",
				Name: "John Doe",
				Role: authorTestStrPtr("invalid-role"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.author.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Author.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthor_GetRoleDisplay(t *testing.T) {
	tests := []struct {
		name   string
		author Author
		want   string
	}{
		{
			name: "with custom contribution",
			author: Author{
				ID:           "john-doe",
				Name:         "John Doe",
				Contribution: authorTestStrPtr("Lead Developer"),
			},
			want: "Lead Developer",
		},
		{
			name: "with simple role",
			author: Author{
				ID:   "john-doe",
				Name: "John Doe",
				Role: authorTestStrPtr("editor"),
			},
			want: "editor",
		},
		{
			name: "with CReDiT contributions",
			author: Author{
				ID:            "john-doe",
				Name:          "John Doe",
				Contributions: []string{"conceptualization", "methodology"},
			},
			want: "conceptualization, methodology",
		},
		{
			name: "default to Author",
			author: Author{
				ID:   "john-doe",
				Name: "John Doe",
			},
			want: "Author",
		},
		{
			name: "contribution takes precedence over role",
			author: Author{
				ID:           "john-doe",
				Name:         "John Doe",
				Role:         strPtr("author"),
				Contribution: authorTestStrPtr("Technical Lead"),
			},
			want: "Technical Lead",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.author.GetRoleDisplay()
			if got != tt.want {
				t.Errorf("Author.GetRoleDisplay() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAuthor_IsPrimaryContributor(t *testing.T) {
	tests := []struct {
		name   string
		author Author
		want   bool
	}{
		{
			name: "with conceptualization",
			author: Author{
				ID:            "john-doe",
				Name:          "John Doe",
				Contributions: []string{"conceptualization"},
			},
			want: true,
		},
		{
			name: "with writing-original-draft",
			author: Author{
				ID:            "john-doe",
				Name:          "John Doe",
				Contributions: []string{"writing-original-draft"},
			},
			want: true,
		},
		{
			name: "with author role",
			author: Author{
				ID:   "john-doe",
				Name: "John Doe",
				Role: authorTestStrPtr("author"),
			},
			want: true,
		},
		{
			name: "with editor role (not primary)",
			author: Author{
				ID:   "john-doe",
				Name: "John Doe",
				Role: authorTestStrPtr("editor"),
			},
			want: false,
		},
		{
			name: "with validation contribution (not primary)",
			author: Author{
				ID:            "john-doe",
				Name:          "John Doe",
				Contributions: []string{"validation"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.author.IsPrimaryContributor()
			if got != tt.want {
				t.Errorf("Author.IsPrimaryContributor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthor_HasContribution(t *testing.T) {
	author := Author{
		ID:            "john-doe",
		Name:          "John Doe",
		Contributions: []string{"conceptualization", "methodology"},
	}

	if !author.HasContribution("conceptualization") {
		t.Error("expected HasContribution('conceptualization') to be true")
	}
	if !author.HasContribution("methodology") {
		t.Error("expected HasContribution('methodology') to be true")
	}
	if author.HasContribution("writing-original-draft") {
		t.Error("expected HasContribution('writing-original-draft') to be false")
	}
}

func TestValidateAuthors(t *testing.T) {
	tests := []struct {
		name    string
		authors map[string]Author
		wantErr bool
	}{
		{
			name:    "nil authors",
			authors: nil,
			wantErr: false,
		},
		{
			name:    "empty authors",
			authors: map[string]Author{},
			wantErr: false,
		},
		{
			name: "single default author",
			authors: map[string]Author{
				"john-doe": {
					ID:      "john-doe",
					Name:    "John Doe",
					Default: true,
				},
			},
			wantErr: false,
		},
		{
			name: "multiple default authors",
			authors: map[string]Author{
				"john-doe": {
					ID:      "john-doe",
					Name:    "John Doe",
					Default: true,
				},
				"jane-doe": {
					ID:      "jane-doe",
					Name:    "Jane Doe",
					Default: true,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid author in collection",
			authors: map[string]Author{
				"invalid": {
					ID: "invalid",
					// Missing Name
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAuthors(tt.authors)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAuthors() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDefaultAuthor(t *testing.T) {
	tests := []struct {
		name       string
		authors    map[string]Author
		wantAuthor bool
		wantID     string
	}{
		{
			name:       "nil authors",
			authors:    nil,
			wantAuthor: false,
			wantID:     "",
		},
		{
			name: "with default author",
			authors: map[string]Author{
				"john-doe": {
					ID:      "john-doe",
					Name:    "John Doe",
					Default: true,
				},
				"jane-doe": {
					ID:   "jane-doe",
					Name: "Jane Doe",
				},
			},
			wantAuthor: true,
			wantID:     "john-doe",
		},
		{
			name: "fallback to active author",
			authors: map[string]Author{
				"john-doe": {
					ID:     "john-doe",
					Name:   "John Doe",
					Active: true,
				},
			},
			wantAuthor: true,
			wantID:     "john-doe",
		},
		{
			name: "no default or active author",
			authors: map[string]Author{
				"john-doe": {
					ID:   "john-doe",
					Name: "John Doe",
				},
			},
			wantAuthor: false,
			wantID:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			author, id := GetDefaultAuthor(tt.authors)
			if (author != nil) != tt.wantAuthor {
				t.Errorf("GetDefaultAuthor() author = %v, wantAuthor %v", author, tt.wantAuthor)
			}
			if id != tt.wantID {
				t.Errorf("GetDefaultAuthor() id = %q, wantID %q", id, tt.wantID)
			}
		})
	}
}

// Helper function for creating string pointers in author tests
func authorTestStrPtr(s string) *string {
	return &s
}
