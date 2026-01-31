package slugmatch

import (
	"testing"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{"identical", "hello", "hello", 0},
		{"empty_a", "", "hello", 5},
		{"empty_b", "hello", "", 5},
		{"both_empty", "", "", 0},
		{"one_char_diff", "hello", "hallo", 1},
		{"kitten_sitting", "kitten", "sitting", 3},
		{"flaw_lawn", "flaw", "lawn", 2},
		{"case_sensitive", "Hello", "hello", 1},
		{"insertion", "abc", "abcd", 1},
		{"deletion", "abcd", "abc", 1},
		{"complete_diff", "abc", "xyz", 3},
		{"transposition", "ab", "ba", 2}, // Note: standard Levenshtein, not Damerau
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LevenshteinDistance(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestLevenshteinDistance_Symmetric(t *testing.T) {
	// Levenshtein distance should be symmetric: d(a,b) == d(b,a)
	pairs := []struct{ a, b string }{
		{"hello", "hallo"},
		{"kitten", "sitting"},
		{"abc", "xyz"},
		{"test", ""},
	}

	for _, pair := range pairs {
		d1 := LevenshteinDistance(pair.a, pair.b)
		d2 := LevenshteinDistance(pair.b, pair.a)
		if d1 != d2 {
			t.Errorf("LevenshteinDistance not symmetric: d(%q,%q)=%d, d(%q,%q)=%d",
				pair.a, pair.b, d1, pair.b, pair.a, d2)
		}
	}
}

func TestNormalizedDistance(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want float64
	}{
		{"identical", "hello", "hello", 0.0},
		{"both_empty", "", "", 0.0},
		{"one_diff", "hello", "hallo", 0.2},  // 1/5
		{"complete_diff", "abc", "xyz", 1.0}, // 3/3
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizedDistance(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("NormalizedDistance(%q, %q) = %f, want %f", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func TestFindSimilarSlugs(t *testing.T) {
	posts := []*models.Post{
		{Slug: "hello-world", Title: strPtr("Hello World")},
		{Slug: "hello-there", Title: strPtr("Hello There")},
		{Slug: "goodbye-world", Title: strPtr("Goodbye World")},
		{Slug: "something-different", Title: strPtr("Something Different")},
		{Slug: "helo-world", Title: strPtr("Helo World")}, // typo
	}

	tests := []struct {
		name       string
		target     string
		maxResults int
		wantSlugs  []string
	}{
		{
			name:       "exact_match",
			target:     "hello-world",
			maxResults: 3,
			wantSlugs:  []string{"hello-world", "helo-world", "hello-there"},
		},
		{
			name:       "typo_match",
			target:     "helo-world",
			maxResults: 3,
			wantSlugs:  []string{"helo-world", "hello-world"}, // hello-there is too different
		},
		{
			name:       "with_path_prefix",
			target:     "/posts/hello-world",
			maxResults: 2,
			wantSlugs:  []string{"hello-world", "helo-world"},
		},
		{
			name:       "max_results_limit",
			target:     "hello-world",
			maxResults: 1,
			wantSlugs:  []string{"hello-world"},
		},
		{
			name:       "no_good_matches",
			target:     "completely-unrelated-long-slug-that-matches-nothing",
			maxResults: 3,
			wantSlugs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindSimilarSlugs(tt.target, posts, tt.maxResults)
			gotSlugs := make([]string, len(got))
			for i, p := range got {
				gotSlugs[i] = p.Slug
			}

			if len(gotSlugs) != len(tt.wantSlugs) {
				t.Errorf("FindSimilarSlugs() returned %d results, want %d\ngot: %v\nwant: %v",
					len(gotSlugs), len(tt.wantSlugs), gotSlugs, tt.wantSlugs)
				return
			}

			for i, want := range tt.wantSlugs {
				if gotSlugs[i] != want {
					t.Errorf("FindSimilarSlugs()[%d] = %q, want %q", i, gotSlugs[i], want)
				}
			}
		})
	}
}

func TestFindSimilarSlugs_EdgeCases(t *testing.T) {
	posts := []*models.Post{
		{Slug: "test-post", Title: strPtr("Test Post")},
	}

	t.Run("nil_posts", func(t *testing.T) {
		got := FindSimilarSlugs("test", nil, 5)
		if got != nil {
			t.Errorf("expected nil for nil posts, got %v", got)
		}
	})

	t.Run("empty_posts", func(t *testing.T) {
		got := FindSimilarSlugs("test", []*models.Post{}, 5)
		if got != nil {
			t.Errorf("expected nil for empty posts, got %v", got)
		}
	})

	t.Run("zero_max_results", func(t *testing.T) {
		got := FindSimilarSlugs("test", posts, 0)
		if got != nil {
			t.Errorf("expected nil for zero maxResults, got %v", got)
		}
	})

	t.Run("negative_max_results", func(t *testing.T) {
		got := FindSimilarSlugs("test", posts, -1)
		if got != nil {
			t.Errorf("expected nil for negative maxResults, got %v", got)
		}
	})

	t.Run("nil_post_in_slice", func(t *testing.T) {
		postsWithNil := []*models.Post{nil, posts[0], nil}
		got := FindSimilarSlugs("test-post", postsWithNil, 5)
		if len(got) != 1 || got[0].Slug != "test-post" {
			t.Errorf("expected to skip nil posts, got %v", got)
		}
	})

	t.Run("empty_slug", func(t *testing.T) {
		postsWithEmpty := []*models.Post{{Slug: "", Title: strPtr("Empty")}, posts[0]}
		got := FindSimilarSlugs("test-post", postsWithEmpty, 5)
		if len(got) != 1 || got[0].Slug != "test-post" {
			t.Errorf("expected to skip empty slugs, got %v", got)
		}
	})
}

func TestFindSimilarByTitle(t *testing.T) {
	posts := []*models.Post{
		{Slug: "post-1", Title: strPtr("Getting Started with Go")},
		{Slug: "post-2", Title: strPtr("Getting Started with Python")},
		{Slug: "post-3", Title: strPtr("Advanced Go Techniques")},
		{Slug: "post-4", Title: nil}, // No title
	}

	tests := []struct {
		name       string
		query      string
		maxResults int
		wantSlugs  []string
	}{
		{
			name:       "exact_title",
			query:      "Getting Started with Go",
			maxResults: 3,
			wantSlugs:  []string{"post-1", "post-2"},
		},
		{
			name:       "partial_match",
			query:      "Getting Started with Go",
			maxResults: 2,
			wantSlugs:  []string{"post-1", "post-2"}, // Both have similar titles
		},
		{
			name:       "empty_query",
			query:      "",
			maxResults: 3,
			wantSlugs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindSimilarByTitle(tt.query, posts, tt.maxResults)
			gotSlugs := make([]string, len(got))
			for i, p := range got {
				gotSlugs[i] = p.Slug
			}

			if len(gotSlugs) != len(tt.wantSlugs) {
				t.Errorf("FindSimilarByTitle() returned %d results, want %d\ngot: %v\nwant: %v",
					len(gotSlugs), len(tt.wantSlugs), gotSlugs, tt.wantSlugs)
				return
			}

			for i, want := range tt.wantSlugs {
				if gotSlugs[i] != want {
					t.Errorf("FindSimilarByTitle()[%d] = %q, want %q", i, gotSlugs[i], want)
				}
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/posts/hello-world", "hello-world"},
		{"posts/hello-world", "hello-world"},
		{"/blog/my-post/", "my-post"},
		{"/articles/test", "test"},
		{"/pages/about", "about"},
		{"simple-slug", "simple-slug"},
		{"/UPPER-CASE/", "upper-case"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizePath(tt.input)
			if got != tt.want {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
