package tui

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/status"
	"github.com/ramtinj/colosseum/internal/tui/dialog"
	"github.com/ramtinj/colosseum/internal/tui/preview"
	"github.com/ramtinj/colosseum/internal/tui/sidebar"
	"github.com/ramtinj/colosseum/internal/tui/theme"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type viewState int

const (
	viewNormal viewState = iota
	viewNewWorkspace
	viewDeleteConfirm
	viewHelp
)

type StatusUpdateMsg status.Update

type errMsg struct{ err error }

type workspaceCreatedMsg struct{ ws *workspace.Workspace }

type workspaceDeletedMsg struct{ id string }

type App struct {
	state     viewState
	sidebar   sidebar.Model
	preview   preview.Model
	newDialog dialog.NewWorkspaceModel
	delDialog dialog.DeleteModel
	helpDialog dialog.HelpModel
	keys      KeyMap
	theme     theme.Theme
	store     *workspace.Store
	manager   *workspace.Manager
	poller    *status.Poller
	detector  *status.Detector
	statusBar string
	width     int
	height    int
	ready     bool
}

func NewApp(store *workspace.Store, manager *workspace.Manager, poller *status.Poller, detector *status.Detector) App {
	return App{
		sidebar:  sidebar.New(),
		preview:  preview.New(),
		keys:     DefaultKeyMap(),
		theme:    theme.DefaultTheme(),
		store:    store,
		manager:  manager,
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
		return a, nil

	case errMsg:
		a.statusBar = fmt.Sprintf("Error: %v", msg.err)
		return a, nil

	case workspacesLoadedMsg:
		a.sidebar.SetWorkspaces(msg.workspaces)
		a.updatePreviewContent()
		return a, nil

	case StatusUpdateMsg:
		a.sidebar.UpdateWorkspaceStatus(msg.WorkspaceID, msg.Current)
		if ws := a.sidebar.SelectedWorkspace(); ws != nil && ws.ID == msg.WorkspaceID {
			a.preview.SetContent(
				fmt.Sprintf("%s (%s)", ws.Title, ws.AgentType),
				msg.PaneContent,
			)
		}
		cmds = append(cmds, a.listenForUpdates())
		return a, tea.Batch(cmds...)

	case workspaceCreatedMsg:
		a.state = viewNormal
		a.statusBar = fmt.Sprintf("Created workspace %q", msg.ws.Title)
		cmds = append(cmds, a.loadWorkspaces)
		return a, tea.Batch(cmds...)

	case workspaceDeletedMsg:
		a.state = viewNormal
		a.statusBar = "Workspace deleted"
		cmds = append(cmds, a.loadWorkspaces)
		return a, tea.Batch(cmds...)
	}

	switch a.state {
	case viewNewWorkspace:
		return a.updateNewDialog(msg)
	case viewDeleteConfirm:
		return a.updateDeleteDialog(msg)
	case viewHelp:
		return a.updateHelpDialog(msg)
	default:
		return a.updateNormal(msg)
	}
}

func (a App) updateNormal(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, a.keys.Quit):
			return a, tea.Quit

		case key.Matches(msg, a.keys.New):
			a.state = viewNewWorkspace
			a.newDialog = dialog.NewNewWorkspace()
			return a, a.newDialog.Init()

		case key.Matches(msg, a.keys.Delete):
			if ws := a.sidebar.SelectedWorkspace(); ws != nil {
				a.state = viewDeleteConfirm
				a.delDialog = dialog.NewDelete(ws.ID, ws.Title)
			}
			return a, nil

		case key.Matches(msg, a.keys.Enter):
			if ws := a.sidebar.SelectedWorkspace(); ws != nil {
				a.statusBar = fmt.Sprintf("Switched to %q — tmux prefix+L to return", ws.Title)
				return a, a.switchToWorkspace(ws.ID)
			}
			return a, nil

		case key.Matches(msg, a.keys.Help):
			a.state = viewHelp
			a.helpDialog = dialog.NewHelp()
			return a, nil

		case key.Matches(msg, a.keys.JumpNext):
			a.jumpToNextAttention()
			a.updatePreviewContent()
			return a, nil
		}

		var cmd tea.Cmd
		a.sidebar, cmd = a.sidebar.Update(msg)
		cmds = append(cmds, cmd)
		a.updatePreviewContent()
	}

	var cmd tea.Cmd
	a.preview, cmd = a.preview.Update(msg)
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

func (a App) updateNewDialog(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case dialog.NewWorkspaceCancelMsg:
		a.state = viewNormal
		a.statusBar = ""
		return a, nil
	case dialog.NewWorkspaceMsg:
		nw := msg.(dialog.NewWorkspaceMsg)
		a.state = viewNormal
		a.statusBar = fmt.Sprintf("Creating workspace %q...", nw.Name)
		return a, a.createWorkspace(nw)
	}

	var cmd tea.Cmd
	a.newDialog, cmd = a.newDialog.Update(msg)
	return a, cmd
}

func (a App) updateDeleteDialog(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case dialog.DeleteCancelMsg:
		a.state = viewNormal
		a.statusBar = ""
		return a, nil
	case dialog.DeleteConfirmMsg:
		dm := msg.(dialog.DeleteConfirmMsg)
		a.state = viewNormal
		a.statusBar = "Deleting workspace..."
		return a, a.deleteWorkspace(dm.WorkspaceID)
	}

	var cmd tea.Cmd
	a.delDialog, cmd = a.delDialog.Update(msg)
	return a, cmd
}

func (a App) updateHelpDialog(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case dialog.HelpCloseMsg:
		a.state = viewNormal
		return a, nil
	}

	var cmd tea.Cmd
	a.helpDialog, cmd = a.helpDialog.Update(msg)
	return a, cmd
}

func (a App) View() string {
	if !a.ready {
		return "Loading..."
	}

	sidebarView := a.sidebar.View()
	previewView := a.preview.View()
	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, previewView)

	helpBar := a.sidebar.ShortHelp()
	if a.statusBar != "" {
		helpBar = a.theme.StatusWaiting.Render(a.statusBar) + "  " + helpBar
	}

	view := lipgloss.JoinVertical(lipgloss.Left, main, helpBar)

	switch a.state {
	case viewNewWorkspace:
		overlay := a.newDialog.View()
		view = a.placeOverlay(view, overlay)
	case viewDeleteConfirm:
		overlay := a.delDialog.View()
		view = a.placeOverlay(view, overlay)
	case viewHelp:
		overlay := a.helpDialog.View()
		view = a.placeOverlay(view, overlay)
	}

	return view
}

func (a App) placeOverlay(bg, overlay string) string {
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("235")),
	)
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
	contentHeight := a.height - 2

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
		a.preview.SetContent(title, "Session not running.\n\nPress 'd' to remove or 'n' to create a new workspace.")
		return
	}
	a.preview.SetContent(title, paneContent)
}

func (a *App) jumpToNextAttention() {
	ws := a.sidebar.Workspaces
	if len(ws) == 0 {
		return
	}
	start := a.sidebar.Cursor
	for i := 1; i <= len(ws); i++ {
		idx := (start + i) % len(ws)
		s := ws[idx].Status
		if s == agent.StatusWaiting || s == agent.StatusError {
			a.sidebar.Cursor = idx
			return
		}
	}
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

func (a App) createWorkspace(nw dialog.NewWorkspaceMsg) tea.Cmd {
	return func() tea.Msg {
		absPath, err := filepath.Abs(nw.Path)
		if err != nil {
			return errMsg{err: fmt.Errorf("resolve path: %w", err)}
		}
		ws, err := a.manager.Create(context.Background(), nw.Name, nw.AgentType, absPath, nw.Branch, nw.Layout)
		if err != nil {
			return errMsg{err: fmt.Errorf("create workspace: %w", err)}
		}
		return workspaceCreatedMsg{ws: ws}
	}
}

func (a App) deleteWorkspace(id string) tea.Cmd {
	return func() tea.Msg {
		if err := a.manager.Delete(context.Background(), id); err != nil {
			return errMsg{err: fmt.Errorf("delete workspace: %w", err)}
		}
		return workspaceDeletedMsg{id: id}
	}
}

func (a App) switchToWorkspace(id string) tea.Cmd {
	return func() tea.Msg {
		if err := a.manager.SwitchTo(context.Background(), id); err != nil {
			return errMsg{err: fmt.Errorf("switch to workspace: %w", err)}
		}
		return nil
	}
}
