package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/services"
)

// View represents different TUI views
type View string

const (
	ViewPosts View = "posts"
	ViewTags  View = "tags"
	ViewFeeds View = "feeds"
	ViewHelp  View = "help"
)

// Mode represents the input mode
type Mode string

const (
	ModeNormal  Mode = "normal"
	ModeFilter  Mode = "filter"
	ModeCommand Mode = "command"
)

// Model is the main Bubble Tea model
type Model struct {
	app         *services.App
	posts       []*models.Post
	tags        []services.TagInfo
	postsTable  table.Model
	cursor      int
	view        View
	mode        Mode
	filter      string
	filterInput textinput.Model
	cmdInput    textinput.Model
	width       int
	height      int
	err         error
}

// Messages
type postsLoadedMsg struct {
	posts []*models.Post
}

type tagsLoadedMsg struct {
	tags []services.TagInfo
}

type errMsg struct {
	err error
}

// NewModel creates a new TUI model
func NewModel(app *services.App) Model {
	filterInput := textinput.New()
	filterInput.Placeholder = "Filter posts..."
	filterInput.CharLimit = 100

	cmdInput := textinput.New()
	cmdInput.Placeholder = "Command..."
	cmdInput.CharLimit = 100

	// Initialize posts table with columns
	postsTable := createPostsTable(80) // Default width, will be updated on resize

	m := Model{
		app:         app,
		view:        ViewPosts,
		mode:        ModeNormal,
		filterInput: filterInput,
		cmdInput:    cmdInput,
		postsTable:  postsTable,
	}

	return m
}

// createPostsTable creates and configures the posts table with the given width
func createPostsTable(width int) table.Model {
	// Column widths: TITLE(40) + DATE(12) + WORDS(8) + TAGS(20) + PATH(remaining)
	// Account for padding/borders (approximately 10 chars)
	pathWidth := width - 40 - 12 - 8 - 20 - 10
	if pathWidth < 10 {
		pathWidth = 10
	}

	columns := []table.Column{
		{Title: "TITLE", Width: 40},
		{Title: "DATE", Width: 12},
		{Title: "WORDS", Width: 8},
		{Title: "TAGS", Width: 20},
		{Title: "PATH", Width: pathWidth},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10), // Will be updated on resize
	)

	// Apply k9s-inspired styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("99"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)
	s.Cell = s.Cell.
		Foreground(lipgloss.Color("252"))
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
		// Update table dimensions
		m.postsTable = createPostsTable(msg.Width)
		m.postsTable.SetHeight(msg.Height - 10) // Leave room for header/footer
		// Repopulate table if we have posts
		if len(m.posts) > 0 {
			m.postsTable.SetRows(m.postsToRows())
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case postsLoadedMsg:
		m.posts = msg.posts
		m.postsTable.SetRows(m.postsToRows())
		return m, nil

	case tagsLoadedMsg:
		m.tags = msg.tags
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
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

// postToRow converts a single post to a table row
func postToRow(p *models.Post) table.Row {
	// Title (truncate to 38 chars to leave room for selection indicator)
	title := "(untitled)"
	if p.Title != nil && *p.Title != "" {
		title = *p.Title
	}
	if len(title) > 38 {
		title = title[:35] + "..."
	}

	// Date (YYYY-MM-DD format)
	date := ""
	if p.Date != nil {
		date = p.Date.Format("2006-01-02")
	}

	// Word count (from Extra field)
	words := ""
	if wc, ok := p.Extra["word_count"].(int); ok {
		words = fmt.Sprintf("%d", wc)
	}

	// Tags (truncate to 18 chars)
	tags := strings.Join(p.Tags, ", ")
	if len(tags) > 18 {
		tags = tags[:15] + "..."
	}

	// Path (will be truncated by table column width)
	path := p.Path

	return table.Row{title, date, words, tags, path}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle mode-specific input
	switch m.mode {
	case ModeFilter:
		return m.handleFilterMode(msg)
	case ModeCommand:
		return m.handleCommandMode(msg)
	case ModeNormal:
		// Fall through to normal mode handling below
	}

	// Normal mode key handling
	switch {
	case key.Matches(msg, keyMap.Quit):
		return m, tea.Quit

	case key.Matches(msg, keyMap.Up), key.Matches(msg, keyMap.Down):
		// Let the table handle navigation when in posts view
		if m.view == ViewPosts {
			var cmd tea.Cmd
			m.postsTable, cmd = m.postsTable.Update(msg)
			m.cursor = m.postsTable.Cursor()
			return m, cmd
		}
		// For other views, use manual cursor movement
		if key.Matches(msg, keyMap.Up) {
			if m.cursor > 0 {
				m.cursor--
			}
		} else {
			maxIdx := len(m.tags) - 1
			if m.cursor < maxIdx {
				m.cursor++
			}
		}

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
		m.view = ViewPosts
		m.cursor = 0
		m.postsTable.SetCursor(0)
		return m, m.loadPosts()

	case key.Matches(msg, keyMap.Tags):
		m.view = ViewTags
		m.cursor = 0
		return m, m.loadTags()
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
		return m, m.loadPosts()
	case "tags", "t":
		m.view = ViewTags
		m.cursor = 0
		return m, m.loadTags()
	case "q", "quit":
		return m, tea.Quit
	}
	return m, nil
}

// Commands
func (m Model) loadPosts() tea.Cmd {
	return func() tea.Msg {
		opts := services.ListOptions{
			SortBy:    "date",
			SortOrder: services.SortDesc,
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
		content = "Feeds view (coming soon)"
	case ViewHelp:
		content = m.renderHelp()
	}

	return m.renderLayout(content)
}

func (m Model) renderLayout(content string) string {
	// Header
	header := headerStyle.Render("markata-go")
	header += " " + subtleStyle.Render(fmt.Sprintf("[%s]", m.view))

	// Status bar
	var statusBar string
	switch m.mode {
	case ModeFilter:
		statusBar = "Filter: " + m.filterInput.View()
	case ModeCommand:
		statusBar = ":" + m.cmdInput.View()
	default:
		statusBar = subtleStyle.Render("j/k:move  /:filter  ::cmd  ?:help  q:quit")
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
	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("\n\n")
	sb.WriteString(m.postsTable.View())

	return sb.String()
}

func (m Model) renderTags() string {
	if len(m.tags) == 0 {
		return "No tags found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Tags (%d)\n\n", len(m.tags)))

	visibleLines := m.height - 8
	if visibleLines < 5 {
		visibleLines = 5
	}

	start := 0
	if m.cursor >= visibleLines {
		start = m.cursor - visibleLines + 1
	}
	end := start + visibleLines
	if end > len(m.tags) {
		end = len(m.tags)
	}

	for i := start; i < end; i++ {
		t := m.tags[i]
		line := fmt.Sprintf("  %s (%d)", t.Name, t.Count)
		if i == m.cursor {
			line = selectedStyle.Render(fmt.Sprintf("> %s (%d)", t.Name, t.Count))
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

func (m Model) renderHelp() string {
	return `Help

Navigation:
  j / ↓      Move down
  k / ↑      Move up
  Enter      Select / view details
  Esc        Cancel / go back

Modes:
  /          Filter mode
  :          Command mode

Commands:
  :posts     Show posts
  :tags      Show tags
  :quit      Exit

Press Esc to return.`
}

// Styles
var (
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	subtleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
)
