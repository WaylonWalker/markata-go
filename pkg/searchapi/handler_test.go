package searchapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestHandler_Search(t *testing.T) {
	title1 := "Go Programming Guide"
	title2 := "Private Diary Entry"
	desc2 := "A private thought"
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	posts := []*models.Post{
		{
			Path:      "posts/go-guide.md",
			Title:     &title1,
			Content:   "Learn Go programming with examples and best practices.",
			Slug:      "go-guide",
			Href:      "/go-guide",
			Tags:      []string{"go", "programming"},
			Published: true,
			Date:      &date,
		},
		{
			Path:        "posts/diary.md",
			Title:       &title2,
			Description: &desc2,
			Content:     "Super secret encrypted content that should never be searchable.",
			Slug:        "diary",
			Href:        "/diary",
			Tags:        []string{"personal"},
			Published:   true,
			Private:     true,
			Date:        &date,
		},
	}

	cacheDir := t.TempDir()
	h := NewHandler(posts, cacheDir, DefaultConfig())
	defer h.Close()

	t.Run("basic search", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=go", http.NoBody)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp SearchResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Total == 0 {
			t.Error("expected results for 'go'")
		}
		if resp.Query != "go" {
			t.Errorf("query = %q, want %q", resp.Query, "go")
		}
	})

	t.Run("missing query", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search", http.NoBody)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/search?q=test", http.NoBody)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})

	t.Run("private post content not searchable", func(t *testing.T) {
		// Search for words ONLY in the private post's content (not in title/description/tags).
		// "practices" appears only in the public post, so use a unique content-only term.
		// The private post content has "Super secret encrypted content" but content is stripped.
		// However, "secret" is a synonym of "private" (which is in the title), so we
		// must search for terms that only exist in the content field, not metadata.
		req := httptest.NewRequest("GET", "/api/search?q=searchable", http.NoBody)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		var resp SearchResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		for _, r := range resp.Results {
			if r.Path == "posts/diary.md" {
				t.Error("private post should not be found via content-only terms")
			}
		}
	})

	t.Run("private post title is searchable", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=diary", http.NoBody)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		var resp SearchResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		found := false
		for _, r := range resp.Results {
			if r.Path == "posts/diary.md" {
				found = true
				if !r.Private {
					t.Error("private flag should be set on result")
				}
			}
		}
		if !found {
			t.Error("private post should be findable by title")
		}
	})

	t.Run("fuzzy search", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=programing&fuzzy=true", http.NoBody)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		var resp SearchResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !resp.Fuzzy {
			t.Error("expected fuzzy=true in response")
		}
	})

	t.Run("limit enforcement", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/search?q=go&limit=999", http.NoBody)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		var resp SearchResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Limit != 100 {
			t.Errorf("limit = %d, want 100 (max limit)", resp.Limit)
		}
	})

	t.Run("CORS preflight", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/search", http.NoBody)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected 204, got %d", w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Error("expected CORS origin header")
		}
	})
}
