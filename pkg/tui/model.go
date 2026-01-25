package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/services"
)

// View represents different TUI views
type View string

const (
	ViewPosts      View = "posts"
	ViewTags       View = "tags"
	ViewFeeds      View = "feeds"
	ViewHelp       View = "help"
	ViewPostDetail View = "post_detail"
)

// Mode represents the input mode
type Mode string

const (
	ModeNormal  Mode = "normal"
	ModeFilter  Mode = "filter"
	ModeCommand Mode = "command"
)

// FilterContext tracks the active filter for drill-down navigation
type FilterContext struct {
	Type string // "tag" or "feed"
	Name string // The tag name or feed name
}

// Model is the main Bubble Tea model
type Model struct {
	app          *services.App
	posts        []*models.Post
	tags         []services.TagInfo
	feeds        []*lifecycle.Feed
	postsTable   table.Model
	tagsTable    table.Model
	cursor       int
	feedCursor   int
	view         View
	previousView View // Track previous view for returning from detail
	mode         Mode
	filter       string
	filterInput  textinput.Model
	cmdInput     textinput.Model
	width        int
	height       int
	err          error
	selectedPost *models.Post // The post being viewed in detail

	// Sort state
	sortBy       string             // "date", "title", "words", "path"
	sortOrder    services.SortOrder // SortAsc or SortDesc
	showSortMenu bool
	sortMenuIdx  int

	// Theme styling
	theme *Theme

	// Drill-down filter state
	activeFilter *FilterContext // Active tag/feed filter for drill-down navigation
}

// Messages
type postsLoadedMsg struct {
	posts []*models.Post
}

type tagsLoadedMsg struct {
	tags []services.TagInfo
}

type feedsLoadedMsg struct {
	feeds []*lifecycle.Feed
}

type errMsg struct {
	err error
}

type editorFinishedMsg struct {
	err error
}

// sortOption represents a sort field option
type sortOption struct {
	label string
	value string
}

// sortOptions available for sorting posts
var sortOptions = []sortOption{
	{"Date", "date"},
	{"Title", "title"},
	{"Word Count", "words"},
	{"Path", "path"},
}

// NewModel creates a new TUI model with default theme.
func NewModel(app *services.App) Model {
	return NewModelWithTheme(app, nil)
}

// NewModelWithTheme creates a new TUI model with the specified theme.
// If theme is nil, the default theme is used.
func NewModelWithTheme(app *services.App, theme *Theme) Model {
	if theme == nil {
		theme = DefaultTheme()
	}

	filterInput := textinput.New()
	filterInput.Placeholder = "e.g., published == True, 'python' in tags"
	filterInput.CharLimit = 100

	cmdInput := textinput.New()
	cmdInput.Placeholder = "Command..."
	cmdInput.CharLimit = 100

	// Initialize posts table with columns and theme
	postsTable := createPostsTableWithTheme(80, theme) // Default width, will be updated on resize

	// Initialize tags table with columns and theme
	tagsTable := createTagsTableWithTheme(80, theme) // Default width, will be updated on resize

	m := Model{
		app:         app,
		view:        ViewPosts,
		mode:        ModeNormal,
		filterInput: filterInput,
		cmdInput:    cmdInput,
		postsTable:  postsTable,
		tagsTable:   tagsTable,
		sortBy:      "date",
		sortOrder:   services.SortDesc,
		theme:       theme,
	}

	return m
}

// createTagsTableWithTheme creates and configures the tags table with theme colors.
func createTagsTableWithTheme(width int, theme *Theme) table.Model {
	// Column widths: TAG(30) + COUNT(10) + SLUG(remaining)
	// Account for padding/borders (approximately 8 chars)
	slugWidth := width - 30 - 10 - 8
	if slugWidth < 15 {
		slugWidth = 15
	}

	columns := []table.Column{
		{Title: "TAG", Width: 30},
		{Title: "COUNT", Width: 10},
		{Title: "SLUG", Width: slugWidth},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10), // Will be updated on resize
	)

	// Apply theme-aware styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.Colors.Border).
		BorderBottom(true).
		Bold(true).
		Foreground(theme.Colors.TableHeader)
	s.Selected = s.Selected.
		Foreground(theme.Colors.SelectedText).
		Background(theme.Colors.SelectedBg).
		Bold(true)
	s.Cell = s.Cell.
		Foreground(theme.Colors.TableCell)
	t.SetStyles(s)

	return t
}

// createPostsTableWithTheme creates and configures the posts table with theme colors.
func createPostsTableWithTheme(width int, theme *Theme) table.Model {
	// Column widths: TITLE(35) + DATE(12) + WORDS(8) + READ(8) + TAGS(18) + PATH(remaining)
	// Account for padding/borders (approximately 10 chars)
	pathWidth := width - 35 - 12 - 8 - 8 - 18 - 10
	if pathWidth < 10 {
		pathWidth = 10
	}

	columns := []table.Column{
		{Title: "TITLE", Width: 35},
		{Title: "DATE", Width: 12},
		{Title: "WORDS", Width: 8},
		{Title: "READ", Width: 8},
		{Title: "TAGS", Width: 18},
		{Title: "PATH", Width: pathWidth},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10), // Will be updated on resize
	)

	// Apply theme-aware styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.Colors.Border).
		BorderBottom(true).
		Bold(true).
		Foreground(theme.Colors.TableHeader)
	s.Selected = s.Selected.
		Foreground(theme.Colors.SelectedText).
		Background(theme.Colors.SelectedBg).
		Bold(true)
	s.Cell = s.Cell.
		Foreground(theme.Colors.TableCell)
	t.SetStyles(s)

	return t
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.loadPosts()
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update table dimensions with theme
		m.postsTable = createPostsTableWithTheme(msg.Width, m.theme)
		m.postsTable.SetHeight(msg.Height - 10) // Leave room for header/footer
		m.tagsTable = createTagsTableWithTheme(msg.Width, m.theme)
		m.tagsTable.SetHeight(msg.Height - 10)
		// Repopulate table if we have posts
		if len(m.posts) > 0 {
			m.postsTable.SetRows(m.postsToRows())
		}
		// Repopulate tags table if we have tags
		if len(m.tags) > 0 {
			m.tagsTable.SetRows(m.tagsToRows())
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case postsLoadedMsg:
		m.posts = msg.posts
		m.postsTable.SetRows(m.postsToRows())
		return m, nil

	case tagsLoadedMsg:
		m.tags = msg.tags
		m.tagsTable.SetRows(m.tagsToRows())
		return m, nil

	case feedsLoadedMsg:
		m.feeds = msg.feeds
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case editorFinishedMsg:
		// Reload posts in case content changed
		if msg.err != nil {
			m.err = msg.err
		}
		return m, m.loadPosts()
	}

	return m, nil
}

// postsToRows converts posts to table rows
func (m Model) postsToRows() []table.Row {
	rows := make([]table.Row, len(m.posts))
	for i, p := range m.posts {
		rows[i] = postToRow(p)
	}
	return rows
}

// tagsToRows converts tags to table rows
func (m Model) tagsToRows() []table.Row {
	rows := make([]table.Row, len(m.tags))
	for i, t := range m.tags {
		rows[i] = tagToRow(t)
	}
	return rows
}

// tagToRow converts a single tag to a table row
func tagToRow(t services.TagInfo) table.Row {
	// Tag name (truncate to 28 chars to leave room for selection indicator)
	name := t.Name
	if len(name) > 28 {
		name = name[:25] + "..."
	}

	// Count
	count := fmt.Sprintf("%d", t.Count)

	// Slug
	slug := t.Slug
	if slug == "" {
		slug = t.Name
	}

	return table.Row{name, count, slug}
}

// postToRow converts a single post to a table row
func postToRow(p *models.Post) table.Row {
	// Title (truncate to 33 chars to leave room for selection indicator)
	title := "(untitled)"
	if p.Title != nil && *p.Title != "" {
		title = *p.Title
	}
	if len(title) > 33 {
		title = title[:30] + "..."
	}

	// Date (YYYY-MM-DD format)
	date := ""
	if p.Date != nil {
		date = p.Date.Format("2006-01-02")
	}

	// Word count (from Extra field, populated by StatsPlugin)
	words := ""
	if wc, ok := p.Extra["word_count"].(int); ok {
		words = formatWordCount(wc)
	}

	// Reading time (from Extra field, populated by StatsPlugin)
	readTime := ""
	if rt, ok := p.Extra["reading_time"].(int); ok {
		readTime = formatReadingTime(rt)
	}

	// Tags (truncate to 16 chars)
	tags := strings.Join(p.Tags, ", ")
	if len(tags) > 16 {
		tags = tags[:13] + "..."
	}

	// Path (will be truncated by table column width)
	path := p.Path

	return table.Row{title, date, words, readTime, tags, path}
}

// formatWordCount formats a word count in a human-readable format (e.g., "1.5k")
func formatWordCount(count int) string {
	if count < 1000 {
		return fmt.Sprintf("%d", count)
	}
	if count < 10000 {
		// Show one decimal place for 1k-9.9k
		return fmt.Sprintf("%.1fk", float64(count)/1000)
	}
	// Round to nearest k for 10k+
	return fmt.Sprintf("%dk", count/1000)
}

// formatReadingTime formats reading time in a compact format (e.g., "2 min")
func formatReadingTime(minutes int) string {
	if minutes == 0 {
		return "<1 min"
	}
	if minutes == 1 {
		return "1 min"
	}
	if minutes < 60 {
		return fmt.Sprintf("%d min", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle sort menu keys when visible
	if m.showSortMenu {
		return m.handleSortMenuKey(msg)
	}

	// Handle mode-specific input
	switch m.mode {
	case ModeFilter:
		return m.handleFilterMode(msg)
	case ModeCommand:
		return m.handleCommandMode(msg)
	case ModeNormal:
		// Fall through to normal mode handling below
	}

	// Handle detail view keys separately
	if m.view == ViewPostDetail {
		return m.handleDetailViewKey(msg)
	}

	// Normal mode key handling
	return m.handleNormalModeKey(msg)
}

// handleMouse handles mouse events
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Only handle mouse in normal mode
	if m.mode != ModeNormal {
		return m, nil
	}

	// Don't handle mouse in sort menu, filter, or command mode
	if m.showSortMenu {
		return m, nil
	}

	// Handle wheel scrolling
	if msg.Action == tea.MouseActionPress {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			// Navigate up (same as 'k' key)
			return m.handleMouseNavigation(true)

		case tea.MouseButtonWheelDown:
			// Navigate down (same as 'j' key)
			return m.handleMouseNavigation(false)

		case tea.MouseButtonLeft:
			// Click to select (same as Enter key)
			if m.view == ViewPostDetail {
				// In detail view, clicking does nothing (could implement back navigation)
				return m, nil
			}
			return m.handleEnter()

		default:
			// Ignore other mouse buttons
			return m, nil
		}
	}

	return m, nil
}

// handleMouseNavigation handles mouse wheel scrolling
func (m Model) handleMouseNavigation(up bool) (tea.Model, tea.Cmd) {
	// Create a simulated key message for up/down
	var keyMsg tea.KeyMsg
	if up {
		keyMsg = tea.KeyMsg{Type: tea.KeyUp}
	} else {
		keyMsg = tea.KeyMsg{Type: tea.KeyDown}
	}

	// Let the table handle navigation when in posts view
	if m.view == ViewPosts {
		var cmd tea.Cmd
		m.postsTable, cmd = m.postsTable.Update(keyMsg)
		m.cursor = m.postsTable.Cursor()
		return m, cmd
	}
	// Let the table handle navigation when in tags view
	if m.view == ViewTags {
		var cmd tea.Cmd
		m.tagsTable, cmd = m.tagsTable.Update(keyMsg)
		m.cursor = m.tagsTable.Cursor()
		return m, cmd
	}
	// For feeds view, use feedCursor
	if m.view == ViewFeeds {
		if up {
			if m.feedCursor > 0 {
				m.feedCursor--
			}
		} else {
			maxIdx := len(m.feeds) - 1
			if m.feedCursor < maxIdx {
				m.feedCursor++
			}
		}
		return m, nil
	}
	return m, nil
}

// handleNormalModeKey handles key input in normal mode
func (m Model) handleNormalModeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keyMap.Quit):
		return m.handleQuitKey()

	case key.Matches(msg, keyMap.Up), key.Matches(msg, keyMap.Down):
		return m.handleNavigation(msg)

	case key.Matches(msg, keyMap.Filter):
		m.mode = ModeFilter
		m.filterInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, keyMap.Command):
		m.mode = ModeCommand
		m.cmdInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, keyMap.Help):
		m.view = ViewHelp

	case key.Matches(msg, keyMap.Posts):
		return m.handlePostsKey()

	case key.Matches(msg, keyMap.Tags):
		m.view = ViewTags
		m.cursor = 0
		m.tagsTable.SetCursor(0)
		return m, m.loadTags()

	case key.Matches(msg, keyMap.Feeds):
		m.view = ViewFeeds
		m.feedCursor = 0
		return m, m.loadFeeds()

	case key.Matches(msg, keyMap.Enter):
		return m.handleEnter()

	case key.Matches(msg, keyMap.Escape):
		return m.handleEscape()

	case key.Matches(msg, keyMap.Edit):
		return m.handleEditKey()

	case key.Matches(msg, keyMap.Sort):
		return m.handleSortKey()
	}

	return m, nil
}

// handleQuitKey handles the quit key, clearing active filter first if present
func (m Model) handleQuitKey() (tea.Model, tea.Cmd) {
	// If there's an active filter in posts view, clear it first instead of quitting
	if m.view == ViewPosts && m.activeFilter != nil {
		m.activeFilter = nil
		m.cursor = 0
		m.postsTable.SetCursor(0)
		return m, m.loadPosts()
	}
	return m, tea.Quit
}

// handlePostsKey handles navigation to posts view
func (m Model) handlePostsKey() (tea.Model, tea.Cmd) {
	m.view = ViewPosts
	m.cursor = 0
	m.postsTable.SetCursor(0)
	m.activeFilter = nil // Clear any active filter when explicitly navigating to posts
	return m, m.loadPosts()
}

// handleEditKey handles the edit key for editing posts
func (m Model) handleEditKey() (tea.Model, tea.Cmd) {
	if m.view == ViewPosts {
		return m, m.openInEditor()
	}
	return m, nil
}

func (m Model) handleSortKey() (tea.Model, tea.Cmd) {
	if m.view == ViewPosts {
		m.showSortMenu = true
		// Set sortMenuIdx to current sort field
		for i, opt := range sortOptions {
			if opt.value == m.sortBy {
				m.sortMenuIdx = i
				break
			}
		}
	}
	return m, nil
}

func (m Model) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Let the table handle navigation when in posts view
	if m.view == ViewPosts {
		var cmd tea.Cmd
		m.postsTable, cmd = m.postsTable.Update(msg)
		m.cursor = m.postsTable.Cursor()
		return m, cmd
	}
	// Let the table handle navigation when in tags view
	if m.view == ViewTags {
		var cmd tea.Cmd
		m.tagsTable, cmd = m.tagsTable.Update(msg)
		m.cursor = m.tagsTable.Cursor()
		return m, cmd
	}
	// For feeds view, use feedCursor
	if m.view == ViewFeeds {
		if key.Matches(msg, keyMap.Up) {
			if m.feedCursor > 0 {
				m.feedCursor--
			}
		} else {
			maxIdx := len(m.feeds) - 1
			if m.feedCursor < maxIdx {
				m.feedCursor++
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.mode = ModeNormal
		m.filterInput.Blur()
		return m, nil

	case tea.KeyEnter:
		m.filter = m.filterInput.Value()
		m.mode = ModeNormal
		m.filterInput.Blur()
		m.cursor = 0
		m.postsTable.SetCursor(0)
		return m, m.loadPosts()

	default:
		// Handle other keys through the text input
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	return m, cmd
}

func (m Model) handleCommandMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.mode = ModeNormal
		m.cmdInput.Blur()
		m.cmdInput.SetValue("")
		return m, nil

	case tea.KeyEnter:
		cmd := m.cmdInput.Value()
		m.cmdInput.SetValue("")
		m.mode = ModeNormal
		m.cmdInput.Blur()
		return m.executeCommand(cmd)

	default:
		// Handle other keys through the text input
	}

	var cmd tea.Cmd
	m.cmdInput, cmd = m.cmdInput.Update(msg)
	return m, cmd
}

func (m Model) executeCommand(cmd string) (tea.Model, tea.Cmd) {
	cmd = strings.TrimSpace(cmd)
	switch cmd {
	case "posts", "p":
		m.view = ViewPosts
		m.cursor = 0
		m.postsTable.SetCursor(0)
		m.activeFilter = nil // Clear any active filter when explicitly navigating to posts
		return m, m.loadPosts()
	case "tags", "t":
		m.view = ViewTags
		m.cursor = 0
		m.tagsTable.SetCursor(0)
		return m, m.loadTags()
	case "feeds", "f":
		m.view = ViewFeeds
		m.feedCursor = 0
		return m, m.loadFeeds()
	case "q", "quit":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewPosts:
		if len(m.posts) > 0 && m.cursor < len(m.posts) {
			m.selectedPost = m.posts[m.cursor]
			m.previousView = m.view
			m.view = ViewPostDetail
		}
	case ViewPostDetail:
		// Already in detail view, Enter does nothing
	case ViewHelp:
		// Return to previous view
		m.view = ViewPosts
	case ViewTags:
		// Drill down into selected tag - show posts with this tag
		if len(m.tags) > 0 && m.cursor < len(m.tags) {
			selectedTag := m.tags[m.cursor]
			m.activeFilter = &FilterContext{
				Type: "tag",
				Name: selectedTag.Name,
			}
			m.view = ViewPosts
			m.cursor = 0
			m.postsTable.SetCursor(0)
			return m, m.loadPostsForTag(selectedTag.Name)
		}
	case ViewFeeds:
		// Drill down into selected feed - show posts in this feed
		if len(m.feeds) > 0 && m.feedCursor < len(m.feeds) {
			selectedFeed := m.feeds[m.feedCursor]
			m.activeFilter = &FilterContext{
				Type: "feed",
				Name: selectedFeed.Name,
			}
			m.view = ViewPosts
			m.cursor = 0
			m.postsTable.SetCursor(0)
			return m, m.loadPostsForFeed(selectedFeed.Name)
		}
	}
	return m, nil
}

func (m Model) handleEscape() (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewPostDetail:
		m.view = m.previousView
		if m.view == "" {
			m.view = ViewPosts
		}
		m.selectedPost = nil
	case ViewHelp:
		m.view = ViewPosts
	case ViewPosts:
		// If there's an active filter, clear it and reload all posts
		if m.activeFilter != nil {
			m.activeFilter = nil
			m.cursor = 0
			m.postsTable.SetCursor(0)
			return m, m.loadPosts()
		}
	case ViewTags, ViewFeeds:
		// Escape does nothing in tag/feed list views
	}
	return m, nil
}

func (m Model) handleDetailViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keyMap.Quit):
		return m, tea.Quit

	case key.Matches(msg, keyMap.Escape):
		m.view = m.previousView
		if m.view == "" {
			m.view = ViewPosts
		}
		m.selectedPost = nil
		return m, nil

	case key.Matches(msg, keyMap.Edit):
		// Edit functionality - show coming soon message for now
		// This will be implemented in issue #221
		m.err = fmt.Errorf("edit feature coming soon (see issue #221)")
		return m, nil
	}

	return m, nil
}

func (m Model) handleSortMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.sortMenuIdx < len(sortOptions)-1 {
			m.sortMenuIdx++
		}
	case "k", "up":
		if m.sortMenuIdx > 0 {
			m.sortMenuIdx--
		}
	case "a":
		m.sortOrder = services.SortAsc
	case "d":
		m.sortOrder = services.SortDesc
	case "enter":
		m.sortBy = sortOptions[m.sortMenuIdx].value
		m.showSortMenu = false
		m.cursor = 0
		m.postsTable.SetCursor(0)
		return m, m.loadPosts()
	case "esc", "q":
		m.showSortMenu = false
	}
	return m, nil
}

// Commands
func (m Model) loadPosts() tea.Cmd {
	return func() tea.Msg {
		opts := services.ListOptions{
			SortBy:    m.sortBy,
			SortOrder: m.sortOrder,
			Filter:    m.filter,
		}
		posts, err := m.app.Posts.List(context.Background(), opts)
		if err != nil {
			return errMsg{err}
		}
		return postsLoadedMsg{posts}
	}
}

func (m Model) loadTags() tea.Cmd {
	return func() tea.Msg {
		tags, err := m.app.Tags.List(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return tagsLoadedMsg{tags}
	}
}

func (m Model) loadFeeds() tea.Cmd {
	return func() tea.Msg {
		feeds, err := m.app.Feeds.List(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return feedsLoadedMsg{feeds}
	}
}

// loadPostsForTag loads posts filtered by a specific tag
func (m Model) loadPostsForTag(tag string) tea.Cmd {
	return func() tea.Msg {
		opts := services.ListOptions{
			SortBy:    m.sortBy,
			SortOrder: m.sortOrder,
		}
		posts, err := m.app.Tags.GetPosts(context.Background(), tag, opts)
		if err != nil {
			return errMsg{err}
		}
		return postsLoadedMsg{posts}
	}
}

// loadPostsForFeed loads posts filtered by a specific feed
func (m Model) loadPostsForFeed(feedName string) tea.Cmd {
	return func() tea.Msg {
		opts := services.ListOptions{
			SortBy:    m.sortBy,
			SortOrder: m.sortOrder,
		}
		posts, err := m.app.Feeds.GetPosts(context.Background(), feedName, opts)
		if err != nil {
			return errMsg{err}
		}
		return postsLoadedMsg{posts}
	}
}

// getSelectedPost returns the currently selected post, or nil if none selected
func (m Model) getSelectedPost() *models.Post {
	if m.view != ViewPosts || len(m.posts) == 0 {
		return nil
	}
	if m.cursor < 0 || m.cursor >= len(m.posts) {
		return nil
	}
	return m.posts[m.cursor]
}

// getEditor returns the editor command to use based on environment variables
func getEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	// Check if vim exists
	if _, err := exec.LookPath("vim"); err == nil {
		return "vim"
	}
	return "nano"
}

// openInEditor opens the selected post in the user's editor
func (m Model) openInEditor() tea.Cmd {
	post := m.getSelectedPost()
	if post == nil {
		return nil
	}

	editor := getEditor()
	c := exec.Command(editor, post.Path)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err}
	})
}

// View renders the UI
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	var content string
	switch m.view {
	case ViewPosts:
		content = m.renderPosts()
	case ViewTags:
		content = m.renderTags()
	case ViewFeeds:
		content = m.renderFeeds()
	case ViewHelp:
		content = m.renderHelp()
	case ViewPostDetail:
		return m.renderPostDetail()
	}

	rendered := m.renderLayout(content)

	// Overlay sort menu if visible
	if m.showSortMenu {
		rendered = m.renderWithSortMenu(rendered)
	}

	return rendered
}

func (m Model) renderLayout(content string) string {
	// Header
	header := m.theme.HeaderStyle.Render("markata-go")
	header += " " + m.theme.SubtleStyle.Render(fmt.Sprintf("[%s]", m.view))

	// Show active filter in header if present
	if m.activeFilter != nil && m.view == ViewPosts {
		filterLabel := fmt.Sprintf(" → %s: %s", m.activeFilter.Type, m.activeFilter.Name)
		header += " " + activeFilterStyle.Render(filterLabel)
	}

	// Status bar
	var statusBar string
	switch m.mode {
	case ModeFilter:
		statusBar = "Filter: " + m.filterInput.View()
	case ModeCommand:
		statusBar = ":" + m.cmdInput.View()
	default:
		// Build sort indicator
		sortArrow := "↓"
		if m.sortOrder == services.SortAsc {
			sortArrow = "↑"
		}
		sortIndicator := fmt.Sprintf("[%s%s]", sortArrow, m.sortBy)
		// Show Esc hint when filter is active
		if m.activeFilter != nil && m.view == ViewPosts {
			statusBar = m.theme.SubtleStyle.Render(fmt.Sprintf("%s  j/k:move  s:sort  e:edit  Esc:clear filter  ?:help  q:quit", sortIndicator))
		} else {
			statusBar = m.theme.SubtleStyle.Render(fmt.Sprintf("%s  j/k:move  s:sort  e:edit  f:feeds  /:filter  ::cmd  ?:help  q:quit", sortIndicator))
		}
	}

	return fmt.Sprintf("%s\n\n%s\n\n%s", header, content, statusBar)
}

func (m Model) renderPosts() string {
	if len(m.posts) == 0 {
		return "No posts found."
	}

	var sb strings.Builder

	// Render the table with header showing count
	header := fmt.Sprintf("Posts (%d)", len(m.posts))
	sb.WriteString(m.theme.HeaderStyle.Render(header))
	sb.WriteString("\n\n")
	sb.WriteString(m.postsTable.View())

	return sb.String()
}

func (m Model) renderTags() string {
	if len(m.tags) == 0 {
		return "No tags found."
	}

	var sb strings.Builder

	// Render the table with header showing count
	header := fmt.Sprintf("Tags (%d)", len(m.tags))
	sb.WriteString(m.theme.HeaderStyle.Render(header))
	sb.WriteString("\n\n")
	sb.WriteString(m.tagsTable.View())

	return sb.String()
}

func (m Model) renderFeeds() string {
	if len(m.feeds) == 0 {
		return "No feeds found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Feeds (%d)\n\n", len(m.feeds)))

	// Calculate column widths based on terminal width
	// NAME(18) + POSTS(7) + WORDS(9) + TOT TIME(10) + AVG TIME(10) + OUTPUT(remaining)
	nameWidth := 18
	postsWidth := 7
	wordsWidth := 9
	totTimeWidth := 10
	avgTimeWidth := 10
	outputWidth := m.width - nameWidth - postsWidth - wordsWidth - totTimeWidth - avgTimeWidth - 14
	if outputWidth < 15 {
		outputWidth = 15
	}

	// Header
	header := fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
		nameWidth, "NAME",
		postsWidth, "POSTS",
		wordsWidth, "WORDS",
		totTimeWidth, "TOT TIME",
		avgTimeWidth, "AVG TIME",
		outputWidth, "OUTPUT")
	sb.WriteString(m.theme.SubtleStyle.Render(header))
	sb.WriteString("\n")

	// Calculate visible rows
	visibleLines := m.height - 10
	if visibleLines < 5 {
		visibleLines = 5
	}

	start := 0
	if m.feedCursor >= visibleLines {
		start = m.feedCursor - visibleLines + 1
	}
	end := start + visibleLines
	if end > len(m.feeds) {
		end = len(m.feeds)
	}

	for i := start; i < end; i++ {
		f := m.feeds[i]

		// Name (truncate if needed)
		name := f.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		}

		// Posts count
		postsCount := fmt.Sprintf("%d", len(f.Posts))

		// Calculate feed statistics from posts
		totalWords, totalReadingTime := calculateFeedStats(f.Posts)
		avgReadingTime := 0
		if len(f.Posts) > 0 {
			avgReadingTime = totalReadingTime / len(f.Posts)
		}

		// Format statistics
		wordsStr := formatWordCount(totalWords)
		totTimeStr := formatReadingTime(totalReadingTime)
		avgTimeStr := formatReadingTime(avgReadingTime)

		// Output path
		output := f.Path
		if len(output) > outputWidth {
			output = output[:outputWidth-3] + "..."
		}

		// Format the row
		prefix := "  "
		if i == m.feedCursor {
			prefix = "> "
		}

		line := fmt.Sprintf("%s%-*s  %-*s  %-*s  %-*s  %-*s  %-*s",
			prefix,
			nameWidth, name,
			postsWidth, postsCount,
			wordsWidth, wordsStr,
			totTimeWidth, totTimeStr,
			avgTimeWidth, avgTimeStr,
			outputWidth, output)

		if i == m.feedCursor {
			line = m.theme.SelectedStyle.Render(line)
		}

		sb.WriteString(line + "\n")
	}

	return sb.String()
}

// calculateFeedStats calculates total words and reading time for a feed's posts
func calculateFeedStats(posts []*models.Post) (totalWords, totalReadingTime int) {
	for _, post := range posts {
		if wc, ok := post.Extra["word_count"].(int); ok {
			totalWords += wc
		}
		if rt, ok := post.Extra["reading_time"].(int); ok {
			totalReadingTime += rt
		}
	}
	return totalWords, totalReadingTime
}

func (m Model) renderHelp() string {
	return `Help

Navigation:
  j / ↓      Move down
  k / ↑      Move up
  Enter      Select / view details
  Esc        Cancel / go back / clear filter

Views:
  p          Posts view
  t          Tags view
  f          Feeds view

Drill-Down Navigation:
  Enter      In tags view: show posts with selected tag
             In feeds view: show posts in selected feed
  Esc        Clear active filter, return to all posts

Actions:
  e          Edit selected post in $EDITOR
  s          Sort menu (Date, Title, Word Count, Path)

Modes:
  /          Filter mode (filter posts with expressions)
  :          Command mode

Filter Syntax:
  Press / to enter filter mode. Filter expressions support:

  Comparison:    published == True, date >= '2024-01-01'
  Membership:    'python' in tags, 'draft' not in tags
  Boolean:       published == True and featured == True
                 published == False or 'wip' in tags
  Strings:       title == 'My Post', slug != 'about'

  Fields: title, slug, date, published, tags, description

  Examples:
    published == True
    'python' in tags
    date >= '2024-01-01' and published == True
    'tutorial' in tags and published == True

Sort Menu:
  j/k/↑/↓    Navigate sort options
  a          Set ascending order
  d          Set descending order
  Enter      Apply sort
  Esc        Cancel

Commands:
  :posts     Show posts
  :tags      Show tags
  :feeds     Show feeds
  :quit      Exit

Press Esc to return.`
}

func (m Model) renderPostDetail() string {
	if m.selectedPost == nil {
		return "No post selected."
	}

	p := m.selectedPost

	// Calculate available width
	width := m.width
	if width < 40 {
		width = 80 // Default minimum
	}
	if width > 100 {
		width = 100 // Max width for readability
	}
	contentWidth := width - 4 // Account for border padding

	// Build the metadata section
	var metadata strings.Builder

	// Title
	title := "(untitled)"
	if p.Title != nil && *p.Title != "" {
		title = *p.Title
	}
	metadata.WriteString(fmt.Sprintf("  %s  %s\n", m.theme.DetailLabelStyle.Render("Title:"), title))

	// Path
	metadata.WriteString(fmt.Sprintf("  %s  %s\n", m.theme.DetailLabelStyle.Render("Path:"), p.Path))

	// Date
	dateStr := "(not set)"
	if p.Date != nil {
		dateStr = p.Date.Format("2006-01-02")
	}
	metadata.WriteString(fmt.Sprintf("  %s  %s\n", m.theme.DetailLabelStyle.Render("Date:"), dateStr))

	// Published
	publishedStr := "false"
	if p.Published {
		publishedStr = "true"
	}
	metadata.WriteString(fmt.Sprintf("  %s  %s\n", m.theme.DetailLabelStyle.Render("Published:"), publishedStr))

	// Tags
	tagsStr := "(none)"
	if len(p.Tags) > 0 {
		tagsStr = strings.Join(p.Tags, ", ")
	}
	metadata.WriteString(fmt.Sprintf("  %s  %s\n", m.theme.DetailLabelStyle.Render("Tags:"), tagsStr))

	// Word count (from stats plugin or fallback to counting)
	wordCount := 0
	if wc, ok := p.Extra["word_count"].(int); ok {
		wordCount = wc
	} else {
		wordCount = countWords(p.Content)
	}
	metadata.WriteString(fmt.Sprintf("  %s  %s\n", m.theme.DetailLabelStyle.Render("Words:"), formatNumber(wordCount)))

	// Reading time (from stats plugin)
	if rt, ok := p.Extra["reading_time"].(int); ok {
		metadata.WriteString(fmt.Sprintf("  %s  %s\n", m.theme.DetailLabelStyle.Render("Read Time:"), formatReadingTime(rt)))
	}

	// Character count (from stats plugin)
	if cc, ok := p.Extra["char_count"].(int); ok {
		metadata.WriteString(fmt.Sprintf("  %s  %s\n", m.theme.DetailLabelStyle.Render("Chars:"), formatNumber(cc)))
	}

	// Description
	if p.Description != nil && *p.Description != "" {
		desc := *p.Description
		if len(desc) > contentWidth-15 {
			desc = desc[:contentWidth-18] + "..."
		}
		metadata.WriteString(fmt.Sprintf("  %s  %s\n", m.theme.DetailLabelStyle.Render("Description:"), desc))
	}

	// Separator
	separator := strings.Repeat("─", contentWidth)

	// Content preview
	var preview strings.Builder
	preview.WriteString("\n  " + m.theme.DetailLabelStyle.Render("Content Preview:") + "\n")

	// Get content preview (first ~500 chars or 15 lines)
	previewContent := getContentPreview(p.Content, 500, 12, contentWidth-4)
	for _, line := range strings.Split(previewContent, "\n") {
		preview.WriteString("  " + line + "\n")
	}

	// Build the full content
	content := metadata.String() + "\n  " + separator + "\n" + preview.String()

	// Create the detail box
	detailBox := m.theme.DetailBoxStyle.
		Width(width).
		Render(content)

	// Status bar
	statusBar := m.theme.DetailStatusStyle.
		Width(width).
		Render("  [e]dit  [Esc] back  [q]uit")

	// Header
	header := m.theme.HeaderStyle.Render("markata-go")
	header += " " + m.theme.SubtleStyle.Render("[post_detail]")

	return header + "\n\n" + detailBox + "\n" + statusBar
}

// countWords counts the number of words in a string
func countWords(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Fields(s))
}

// formatNumber formats a number with comma separators
func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	// Format with commas
	s := fmt.Sprintf("%d", n)
	var result strings.Builder
	length := len(s)

	for i, c := range s {
		if i > 0 && (length-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}

	return result.String()
}

// getContentPreview returns a truncated preview of the content
func getContentPreview(content string, maxChars, maxLines, maxWidth int) string {
	if content == "" {
		return "(empty)"
	}

	// Split into lines
	lines := strings.Split(content, "\n")

	var result strings.Builder
	charCount := 0
	lineCount := 0

	for _, line := range lines {
		if lineCount >= maxLines || charCount >= maxChars {
			break
		}

		// Truncate long lines
		if utf8.RuneCountInString(line) > maxWidth {
			line = string([]rune(line)[:maxWidth-3]) + "..."
		}

		result.WriteString(line)
		result.WriteString("\n")

		charCount += len(line)
		lineCount++
	}

	output := strings.TrimRight(result.String(), "\n")

	// Add ellipsis if content was truncated
	if charCount >= maxChars || lineCount >= maxLines {
		output += "\n..."
	}

	return output
}

// renderWithSortMenu overlays the sort menu on top of the existing content
func (m Model) renderWithSortMenu(content string) string {
	menu := m.renderSortMenu()

	// Split content into lines
	contentLines := strings.Split(content, "\n")
	menuLines := strings.Split(menu, "\n")

	// Calculate position to center the menu
	menuWidth := 23 // Width of the menu box
	menuHeight := len(menuLines)

	startX := (m.width - menuWidth) / 2
	if startX < 0 {
		startX = 0
	}
	startY := (m.height - menuHeight) / 2
	if startY < 2 {
		startY = 2
	}

	// Overlay menu on content
	for i, menuLine := range menuLines {
		contentY := startY + i
		if contentY < len(contentLines) {
			line := contentLines[contentY]
			// Ensure line is long enough
			for len(line) < startX {
				line += " "
			}
			// Insert menu line
			runes := []rune(line)
			menuRunes := []rune(menuLine)
			if startX < len(runes) {
				newLine := string(runes[:startX]) + string(menuRunes)
				if startX+len(menuRunes) < len(runes) {
					newLine += string(runes[startX+len(menuRunes):])
				}
				contentLines[contentY] = newLine
			} else {
				contentLines[contentY] = line + menuLine
			}
		}
	}

	return strings.Join(contentLines, "\n")
}

// renderSortMenu renders the sort menu box
func (m Model) renderSortMenu() string {
	var sb strings.Builder

	// Build menu content
	sb.WriteString("┌─ Sort By ─────────┐\n")

	for i, opt := range sortOptions {
		prefix := "  "
		if i == m.sortMenuIdx {
			prefix = "> "
		}
		// Pad label to fixed width
		label := opt.label
		for len(label) < 16 {
			label += " "
		}
		sb.WriteString(fmt.Sprintf("│ %s%s │\n", prefix, label))
	}

	sb.WriteString("├───────────────────┤\n")

	// Show current order with highlight
	ascStyle := ""
	descStyle := ""
	if m.sortOrder == services.SortAsc {
		ascStyle = "*"
	} else {
		descStyle = "*"
	}
	sb.WriteString(fmt.Sprintf("│ [a]sc%s  [d]esc%s   │\n", ascStyle, descStyle))
	sb.WriteString("│ [Enter] apply     │\n")
	sb.WriteString("└───────────────────┘")

	return m.theme.SortMenuStyle.Render(sb.String())
}

// Styles
var (
	// Active filter style - shows current tag/feed filter
	activeFilterStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57"))
)

// CI trigger
