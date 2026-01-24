package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
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
	cursor      int
	view        View
	mode        Mode
	filter      string
	filterInput textinput.Model
	cmdInput    textinput.Model
	width       int
	height      int
	err         error
	// Sort state
	sortBy       string             // "date", "title", "words", "path"
	sortOrder    services.SortOrder // SortAsc or SortDesc
	showSortMenu bool               // overlay visible
	sortMenuIdx  int                // selected option in menu
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

// sortOption defines a sort field option
type sortOption struct {
	label string
	value string
}

// sortOptions are the available sort fields
var sortOptions = []sortOption{
	{"Date", "date"},
	{"Title", "title"},
	{"Word Count", "words"},
	{"Path", "path"},
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
		sortBy:      "date",
		sortOrder:   services.SortDesc,
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

	case errMsg:
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle sort menu if visible
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

	case key.Matches(msg, keyMap.Sort):
		if m.view == ViewPosts {
			m.showSortMenu = true
			// Set sortMenuIdx to current sortBy
			for i, opt := range sortOptions {
				if opt.value == m.sortBy {
					m.sortMenuIdx = i
					break
				}
			}
		}
	}

	return m, nil
}

func (m Model) handleSortMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel without applying
		m.showSortMenu = false
		return m, nil

	case "enter":
		// Apply sort and close menu
		m.sortBy = sortOptions[m.sortMenuIdx].value
		m.showSortMenu = false
		m.cursor = 0
		return m, m.loadPosts()

	case "j", "down":
		// Move selection down
		if m.sortMenuIdx < len(sortOptions)-1 {
			m.sortMenuIdx++
		}
		return m, nil

	case "k", "up":
		// Move selection up
		if m.sortMenuIdx > 0 {
			m.sortMenuIdx--
		}
		return m, nil

	case "a":
		// Set ascending order
		m.sortOrder = services.SortAsc
		return m, nil

	case "d":
		// Set descending order
		m.sortOrder = services.SortDesc
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
	case "q", "quit":
		return m, tea.Quit
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

	layout := m.renderLayout(content)

	// Overlay sort menu if visible
	if m.showSortMenu {
		layout = m.overlayContent(layout, m.renderSortMenu())
	}

	return layout
}

func (m Model) renderLayout(content string) string {
	// Header
	header := headerStyle.Render("markata-go")
	header += " " + subtleStyle.Render(fmt.Sprintf("[%s]", m.view))

	// Sort indicator for posts view
	if m.view == ViewPosts {
		arrow := "↓"
		if m.sortOrder == services.SortAsc {
			arrow = "↑"
		}
		header += " " + subtleStyle.Render(fmt.Sprintf("[%s%s]", arrow, m.sortBy))
	}

	// Status bar
	var statusBar string
	switch m.mode {
	case ModeFilter:
		statusBar = "Filter: " + m.filterInput.View()
	case ModeCommand:
		statusBar = ":" + m.cmdInput.View()
	default:
		statusBar = subtleStyle.Render("j/k:move  /:filter  s:sort  ::cmd  ?:help  q:quit")
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

func (m Model) renderHelp() string {
	return `Help

Navigation:
  j / ↓      Move down
  k / ↑      Move up
  Enter      Select / view details
  Esc        Cancel / go back

Modes:
  /          Filter mode
  s          Sort menu (posts view)
  :          Command mode

Sort Menu:
  j/k        Navigate options
  a          Sort ascending
  d          Sort descending
  Enter      Apply sort
  Esc        Cancel

Commands:
  :posts     Show posts
  :tags      Show tags
  :quit      Exit

Press Esc to return.`
}

func (m Model) renderSortMenu() string {
	var sb strings.Builder

	// Menu title
	sb.WriteString("┌─ Sort By ─────────┐\n")

	// Sort options
	for i, opt := range sortOptions {
		prefix := "  "
		if i == m.sortMenuIdx {
			prefix = "> "
		}
		// Pad label to 16 chars for alignment
		label := fmt.Sprintf("%-16s", opt.label)
		sb.WriteString(fmt.Sprintf("│ %s%s│\n", prefix, label))
	}

	// Divider
	sb.WriteString("├───────────────────┤\n")

	// Order indicator
	ascMark := " "
	descMark := " "
	if m.sortOrder == services.SortAsc {
		ascMark = "●"
	} else {
		descMark = "●"
	}
	sb.WriteString(fmt.Sprintf("│ [a]sc %s [d]esc %s │\n", ascMark, descMark))

	// Footer
	sb.WriteString("│ [Enter] apply     │\n")
	sb.WriteString("└───────────────────┘")

	return sb.String()
}

func (m Model) overlayContent(base, overlay string) string {
	// Split both into lines
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	// Position the overlay (center it vertically, offset from left)
	startRow := 3 // Start a few rows down
	startCol := 2 // Left padding

	// Make a copy of base lines
	result := make([]string, len(baseLines))
	copy(result, baseLines)

	// Overlay each line
	for i, overlayLine := range overlayLines {
		row := startRow + i
		if row >= len(result) {
			break
		}

		// Ensure the base line is long enough
		baseLine := result[row]
		// Convert to runes for proper Unicode handling
		baseRunes := []rune(baseLine)
		overlayRunes := []rune(overlayLine)

		// Pad base line if needed
		for len(baseRunes) < startCol+len(overlayRunes) {
			baseRunes = append(baseRunes, ' ')
		}

		// Insert overlay
		for j, r := range overlayRunes {
			if startCol+j < len(baseRunes) {
				baseRunes[startCol+j] = r
			}
		}

		result[row] = string(baseRunes)
	}

	return strings.Join(result, "\n")
}

// Styles
var (
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	subtleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
)
