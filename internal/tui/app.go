package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ramtinj/colosseum/internal/status"
	"github.com/ramtinj/colosseum/internal/tui/preview"
	"github.com/ramtinj/colosseum/internal/tui/sidebar"
	"github.com/ramtinj/colosseum/internal/tui/theme"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type StatusUpdateMsg status.Update

type App struct {
	sidebar  sidebar.Model
	preview  preview.Model
	keys     KeyMap
	theme    theme.Theme
	store    *workspace.Store
	poller   *status.Poller
	detector *status.Detector
	width    int
	height   int
	ready    bool
}

func NewApp(store *workspace.Store, poller *status.Poller, detector *status.Detector) App {
	return App{
		sidebar:  sidebar.New(),
		preview:  preview.New(),
		keys:     DefaultKeyMap(),
		theme:    theme.DefaultTheme(),
		store:    store,
		poller:   poller,
		detector: detector,
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.loadWorkspaces,
		a.listenForUpdates(),
	)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		a.layoutPanels()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		}
		var cmd tea.Cmd
		a.sidebar, cmd = a.sidebar.Update(msg)
		cmds = append(cmds, cmd)

		a.updatePreviewContent()

	case StatusUpdateMsg:
		a.sidebar.UpdateWorkspaceStatus(msg.WorkspaceID, msg.Current)
		if ws := a.sidebar.SelectedWorkspace(); ws != nil && ws.ID == msg.WorkspaceID {
			a.preview.SetContent(
				fmt.Sprintf("%s (%s)", ws.Title, ws.AgentType),
				msg.PaneContent,
			)
		}
		cmds = append(cmds, a.listenForUpdates())

	case workspacesLoadedMsg:
		a.sidebar.SetWorkspaces(msg.workspaces)
		a.updatePreviewContent()
	}

	var cmd tea.Cmd
	a.preview, cmd = a.preview.Update(msg)
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

func (a App) View() string {
	if !a.ready {
		return "Loading..."
	}

	sidebarView := a.sidebar.View()
	previewView := a.preview.View()

	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, previewView)

	helpBar := a.sidebar.ShortHelp()

	return lipgloss.JoinVertical(lipgloss.Left, main, helpBar)
}

func (a *App) layoutPanels() {
	sidebarWidth := a.width / 3
	if sidebarWidth < 30 {
		sidebarWidth = 30
	}
	if sidebarWidth > 50 {
		sidebarWidth = 50
	}
	previewWidth := a.width - sidebarWidth
	contentHeight := a.height - 2 // room for help bar

	a.sidebar.Width = sidebarWidth - 2
	a.sidebar.Height = contentHeight - 2

	a.preview.SetSize(previewWidth-2, contentHeight-2)
}

func (a *App) updatePreviewContent() {
	ws := a.sidebar.SelectedWorkspace()
	if ws == nil {
		a.preview.SetContent("", "")
		return
	}

	title := fmt.Sprintf("%s (%s)", ws.Title, ws.AgentType)

	agentPane, ok := ws.PaneTargets["agent"]
	if !ok {
		a.preview.SetContent(title, "No agent pane configured")
		return
	}

	_, paneContent, err := a.detector.Detect(context.Background(), agentPane, ws.AgentType)
	if err != nil {
		a.preview.SetContent(title, fmt.Sprintf("Error capturing pane: %v", err))
		return
	}
	a.preview.SetContent(title, paneContent)
}

func (a App) listenForUpdates() tea.Cmd {
	if a.poller == nil {
		return nil
	}
	ch := a.poller.Updates()
	return func() tea.Msg {
		update, ok := <-ch
		if !ok {
			return nil
		}
		return StatusUpdateMsg(update)
	}
}

type workspacesLoadedMsg struct {
	workspaces []workspace.Workspace
}

func (a App) loadWorkspaces() tea.Msg {
	if a.store == nil {
		return workspacesLoadedMsg{}
	}
	ws, err := a.store.List()
	if err != nil {
		return workspacesLoadedMsg{}
	}
	return workspacesLoadedMsg{workspaces: ws}
}
