package preview

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	viewport    viewport.Model
	title       string
	Width       int
	Height      int
	ready       bool
}

func New() Model {
	return Model{}
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
	if !m.ready {
		m.viewport = viewport.New(width, height-2)
		m.ready = true
	} else {
		m.viewport.Width = width
		m.viewport.Height = height - 2
	}
}

func (m *Model) SetContent(title, content string) {
	m.title = title
	m.viewport.SetContent(content)
}
