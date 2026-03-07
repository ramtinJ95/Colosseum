package preview

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ramtinj/colosseum/internal/tui/theme"
)

type Model struct {
	viewport  viewport.Model
	title     string
	tabs      []string
	activeTab int
	Width     int
	Height    int
	ready     bool
	theme     theme.Theme
}

func New() Model {
	return Model{theme: theme.DefaultTheme()}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	viewportHeight := height - 2
	if len(m.tabs) > 1 {
		viewportHeight--
	}
	if !m.ready {
		m.viewport = viewport.New(width, viewportHeight)
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = viewportHeight
	}
}

func (m *Model) SetTabs(tabs []string, active int) {
	m.tabs = tabs
	m.activeTab = active
}

func (m *Model) SetContent(title, content string) {
	m.title = title
	if m.viewport.Width > 0 {
		content = wrapContent(content, m.viewport.Width)
	}
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

func (m Model) WithTheme(t theme.Theme) Model {
	m.theme = t
	return m
}

func wrapContent(content string, width int) string {
	if width <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapLine(line, width)...)
	}
	return strings.Join(wrapped, "\n")
}

func wrapLine(line string, width int) []string {
	if line == "" || lipgloss.Width(line) <= width {
		return []string{line}
	}

	words := strings.Fields(line)
	if len(words) == 0 {
		return breakLongToken(line, width)
	}

	var result []string
	currentParts := breakLongToken(words[0], width)
	result = append(result, currentParts[:len(currentParts)-1]...)
	current := currentParts[len(currentParts)-1]
	for _, word := range words[1:] {
		wordParts := breakLongToken(word, width)
		for _, part := range wordParts[:len(wordParts)-1] {
			if current != "" {
				result = append(result, current)
				current = ""
			}
			result = append(result, part)
		}

		word = wordParts[len(wordParts)-1]
		candidate := current + " " + word
		if current == "" {
			candidate = word
		}
		if lipgloss.Width(candidate) <= width {
			current = candidate
			continue
		}
		if current != "" {
			result = append(result, current)
		}
		current = word
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func breakLongToken(token string, width int) []string {
	if token == "" || lipgloss.Width(token) <= width {
		return []string{token}
	}

	var parts []string
	var current []rune
	for _, r := range []rune(token) {
		next := append(current, r)
		if lipgloss.Width(string(next)) > width && len(current) > 0 {
			parts = append(parts, string(current))
			current = []rune{r}
			continue
		}
		current = next
	}
	if len(current) > 0 {
		parts = append(parts, string(current))
	}
	return parts
}
