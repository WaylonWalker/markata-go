package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{12, "12"},
		{123, "123"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatNumber(tt.input)
			if got != tt.want {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single word", "hello", 1},
		{"multiple words", "hello world", 2},
		{"with newlines", "hello\nworld\nfoo", 3},
		{"with extra spaces", "  hello   world  ", 2},
		{"markdown content", "# Title\n\nThis is a paragraph with several words.", 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countWords(tt.input)
			if got != tt.want {
				t.Errorf("countWords(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetContentPreview(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		maxChars int
		maxLines int
		maxWidth int
		wantLen  int // Check that output length is reasonable
	}{
		{"empty", "", 500, 12, 80, 7}, // "(empty)"
		{"short", "hello", 500, 12, 80, 5},
		{"multiline", "line1\nline2\nline3", 500, 12, 80, 17},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getContentPreview(tt.content, tt.maxChars, tt.maxLines, tt.maxWidth)
			if len(got) < tt.wantLen {
				t.Errorf("getContentPreview() length = %d, want at least %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestHandleEnter_PostsView(t *testing.T) {
	// Create a model with some posts
	m := Model{
		view:   ViewPosts,
		cursor: 0,
		posts: []*models.Post{
			{Path: "test.md", Content: "Test content"},
		},
	}

	// Simulate pressing Enter
	newModel, _ := m.handleEnter()
	newM, ok := newModel.(Model)
	if !ok {
		t.Fatal("expected Model type")
	}

	if newM.view != ViewPostDetail {
		t.Errorf("view = %q, want %q", newM.view, ViewPostDetail)
	}

	if newM.selectedPost == nil {
		t.Error("selectedPost is nil, expected a post")
	}

	if newM.previousView != ViewPosts {
		t.Errorf("previousView = %q, want %q", newM.previousView, ViewPosts)
	}
}

func TestHandleEscape_DetailView(t *testing.T) {
	title := "Test Post"
	m := Model{
		view:         ViewPostDetail,
		previousView: ViewPosts,
		selectedPost: &models.Post{Title: &title},
	}

	newModel, _ := m.handleEscape()
	newM, ok := newModel.(Model)
	if !ok {
		t.Fatal("expected Model type")
	}

	if newM.view != ViewPosts {
		t.Errorf("view = %q, want %q", newM.view, ViewPosts)
	}

	if newM.selectedPost != nil {
		t.Error("selectedPost should be nil after escape")
	}
}

func TestHandleDetailViewKey_Quit(t *testing.T) {
	m := Model{
		view: ViewPostDetail,
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.handleDetailViewKey(msg)

	// Check that the command is tea.Quit
	if cmd == nil {
		t.Error("expected quit command, got nil")
	}
}

func TestHandleDetailViewKey_Escape(t *testing.T) {
	title := "Test"
	m := Model{
		view:         ViewPostDetail,
		previousView: ViewPosts,
		selectedPost: &models.Post{Title: &title},
	}

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	newModel, _ := m.handleDetailViewKey(msg)
	newM, ok := newModel.(Model)
	if !ok {
		t.Fatal("expected Model type")
	}

	if newM.view != ViewPosts {
		t.Errorf("view = %q, want %q", newM.view, ViewPosts)
	}
}

func TestRenderPostDetail_NilPost(t *testing.T) {
	m := Model{
		view:         ViewPostDetail,
		selectedPost: nil,
		width:        80,
		height:       24,
	}

	result := m.renderPostDetail()
	if result != "No post selected." {
		t.Errorf("expected 'No post selected.', got %q", result)
	}
}

func TestRenderPostDetail_WithPost(t *testing.T) {
	title := "Test Title"
	desc := "Test description"
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	m := Model{
		view: ViewPostDetail,
		selectedPost: &models.Post{
			Title:       &title,
			Path:        "posts/test.md",
			Date:        &date,
			Published:   true,
			Tags:        []string{"go", "test"},
			Description: &desc,
			Content:     "# Test\n\nThis is test content.",
		},
		width:  80,
		height: 24,
		theme:  DefaultTheme(),
	}

	result := m.renderPostDetail()

	// Check that key elements are present
	if !contains(result, "Test Title") {
		t.Error("expected title in output")
	}
	if !contains(result, "posts/test.md") {
		t.Error("expected path in output")
	}
	if !contains(result, "2024-01-15") {
		t.Error("expected date in output")
	}
	if !contains(result, "true") {
		t.Error("expected published status in output")
	}
	if !contains(result, "go, test") {
		t.Error("expected tags in output")
	}
	if !contains(result, "[e]dit") {
		t.Error("expected edit keybinding in output")
	}
	if !contains(result, "[Esc] back") {
		t.Error("expected escape keybinding in output")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
