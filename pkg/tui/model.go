package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
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
	feeds       []*lifecycle.Feed
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

type feedsLoadedMsg struct {
	feeds []*lifecycle.Feed
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

	m := Model{
		app:         app,
		view:        ViewPosts,
		mode:        ModeNormal,
		filterInput: filterInput,
		cmdInput:    cmdInput,
	}

	return m
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
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case postsLoadedMsg:
		m.posts = msg.posts
		return m, nil

	case tagsLoadedMsg:
		m.tags = msg.tags
		return m, nil

	case feedsLoadedMsg:
		m.feeds = msg.feeds
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
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

	case key.Matches(msg, keyMap.Up):
		if m.cursor > 0 {
			m.cursor--
		}

	case key.Matches(msg, keyMap.Down):
		maxIdx := len(m.posts) - 1
		if m.view == ViewTags {
			maxIdx = len(m.tags) - 1
		} else if m.view == ViewFeeds {
			maxIdx = len(m.feeds) - 1
		}
		if m.cursor < maxIdx {
			m.cursor++
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
		return m, m.loadPosts()

	case key.Matches(msg, keyMap.Tags):
		m.view = ViewTags
		m.cursor = 0
		return m, m.loadTags()

	case key.Matches(msg, keyMap.Feeds):
		m.view = ViewFeeds
		m.cursor = 0
		return m, m.loadFeeds()
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
		return m, m.loadPosts()
	case "tags", "t":
		m.view = ViewTags
		m.cursor = 0
		return m, m.loadTags()
	case "feeds", "f":
		m.view = ViewFeeds
		m.cursor = 0
		return m, m.loadFeeds()
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

func (m Model) loadFeeds() tea.Cmd {
	return func() tea.Msg {
		feeds, err := m.app.Feeds.List(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return feedsLoadedMsg{feeds}
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
		content = m.renderFeeds()
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
		statusBar = subtleStyle.Render("j/k:move  p:posts  t:tags  f:feeds  /:filter  ::cmd  ?:help  q:quit")
	}

	return fmt.Sprintf("%s\n\n%s\n\n%s", header, content, statusBar)
}

func (m Model) renderPosts() string {
	if len(m.posts) == 0 {
		return "No posts found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Posts (%d)\n\n", len(m.posts)))

	// Calculate visible range
	visibleLines := m.height - 8
	if visibleLines < 5 {
		visibleLines = 5
	}

	start := 0
	if m.cursor >= visibleLines {
		start = m.cursor - visibleLines + 1
	}
	end := start + visibleLines
	if end > len(m.posts) {
		end = len(m.posts)
	}

	for i := start; i < end; i++ {
		p := m.posts[i]
		title := "(untitled)"
		if p.Title != nil {
			title = *p.Title
		}

		line := fmt.Sprintf("  %s", title)
		if i == m.cursor {
			line = selectedStyle.Render("> " + title)
		}
		sb.WriteString(line + "\n")
	}

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

func (m Model) renderFeeds() string {
	if len(m.feeds) == 0 {
		return "No feeds found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Feeds (%d)\n\n", len(m.feeds)))

	// Column widths
	nameWidth := 20
	postsWidth := 6
	filterWidth := 30

	// Header row
	header := fmt.Sprintf("%-*s  %*s  %-*s  %s",
		nameWidth, "NAME",
		postsWidth, "POSTS",
		filterWidth, "FILTER",
		"OUTPUT")
	sb.WriteString(subtleStyle.Render(header) + "\n")

	// Calculate visible range
	visibleLines := m.height - 10
	if visibleLines < 5 {
		visibleLines = 5
	}

	start := 0
	if m.cursor >= visibleLines {
		start = m.cursor - visibleLines + 1
	}
	end := start + visibleLines
	if end > len(m.feeds) {
		end = len(m.feeds)
	}

	for i := start; i < end; i++ {
		f := m.feeds[i]

		name := truncateString(f.Name, nameWidth)
		postCount := len(f.Posts)

		filter := "(none)"
		// Note: lifecycle.Feed doesn't have a Filter field, so we show "(none)"
		// If we had filter info, we'd display it here

		output := f.Path
		if output == "" {
			output = "-"
		}

		line := fmt.Sprintf("  %-*s  %*d  %-*s  %s",
			nameWidth, name,
			postsWidth, postCount,
			filterWidth, truncateString(filter, filterWidth-2),
			output)

		if i == m.cursor {
			line = selectedStyle.Render(fmt.Sprintf("> %-*s  %*d  %-*s  %s",
				nameWidth-2, name,
				postsWidth, postCount,
				filterWidth, truncateString(filter, filterWidth-2),
				output))
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

// truncateString truncates a string to a maximum width, adding "..." if truncated
func truncateString(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return s[:maxWidth]
	}
	return s[:maxWidth-3] + "..."
}

func (m Model) renderHelp() string {
	return `Help

Navigation:
  j / ↓      Move down
  k / ↑      Move up
  Enter      Select / view details
  Esc        Cancel / go back

Views:
  p          Posts view
  t          Tags view
  f          Feeds view

Modes:
  /          Filter mode
  :          Command mode

Commands:
  :posts     Show posts
  :tags      Show tags
  :feeds     Show feeds
  :quit      Exit

Press Esc to return.`
}

// Styles
var (
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	subtleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
)
