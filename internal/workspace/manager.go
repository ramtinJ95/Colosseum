package workspace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/tmux"
)

type SessionCreator interface {
	CreateSession(ctx context.Context, name string, startDir string) (string, error)
	KillSession(ctx context.Context, name string) error
	SplitWindow(ctx context.Context, session string, horizontal bool, startDir string) (string, error)
	SwitchClient(ctx context.Context, name string) error
	SendKeys(ctx context.Context, target string, keys string) error
}

type Manager struct {
	store         *Store
	sessions      SessionCreator
	sessionPrefix string
}

type BroadcastFailure struct {
	WorkspaceID    string
	WorkspaceTitle string
	Err            error
}

type BroadcastResult struct {
	Requested int
	Delivered []string
	Failed    []BroadcastFailure
}

func NewManager(store *Store, sessions SessionCreator, prefix string) *Manager {
	return &Manager{
		store:         store,
		sessions:      sessions,
		sessionPrefix: prefix,
	}
}

func (m *Manager) Create(ctx context.Context, title string, agentType agent.AgentType, projectPath string, branch string, layout LayoutType) (*Workspace, error) {
	if !agent.IsSupported(agentType) {
		return nil, fmt.Errorf("unsupported agent type %q", agentType)
	}
	if !IsValidLayout(layout) {
		return nil, fmt.Errorf("invalid layout %q", layout)
	}

	existing, err := m.store.List()
	if err != nil {
		return nil, fmt.Errorf("checking existing workspaces: %w", err)
	}
	for _, ws := range existing {
		if ws.Title == title {
			return nil, fmt.Errorf("workspace %q already exists", title)
		}
	}

	id := uuid.New().String()
	sessionName := m.sessionPrefix + title

	agentPaneID, err := m.sessions.CreateSession(ctx, title, projectPath)
	if err != nil {
		return nil, fmt.Errorf("creating session for %q: %w", title, err)
	}
	rollback := true
	defer func() {
		if rollback {
			_ = m.sessions.KillSession(ctx, title)
		}
	}()

	paneTargets := map[string]string{
		"agent": agentPaneID,
	}

	if layout.PaneCount() >= 2 {
		paneID, err := m.sessions.SplitWindow(ctx, title, true, projectPath)
		if err != nil {
			return nil, fmt.Errorf("splitting window for shell pane: %w", err)
		}
		paneTargets["shell"] = paneID
	}

	if layout.PaneCount() >= 3 {
		paneID, err := m.sessions.SplitWindow(ctx, title, false, projectPath)
		if err != nil {
			return nil, fmt.Errorf("splitting window for logs pane: %w", err)
		}
		paneTargets["logs"] = paneID
	}

	def, ok := agent.Get(agentType)
	if !ok {
		return nil, fmt.Errorf("agent type %q is not registered", agentType)
	}
	launchCmd := def.Binary
	for _, flag := range def.LaunchFlags {
		launchCmd += " " + flag
	}
	if err := m.sessions.SendKeys(ctx, agentPaneID, launchCmd); err != nil {
		return nil, fmt.Errorf("launching agent %q: %w", agentType, err)
	}

	ws := Workspace{
		ID:          id,
		Title:       title,
		AgentType:   agentType,
		ProjectPath: projectPath,
		Branch:      branch,
		Layout:      layout,
		Status:      agent.StatusIdle,
		SessionName: sessionName,
		PaneTargets: paneTargets,
		CreatedAt:   time.Now(),
	}

	if err := m.store.Add(ws); err != nil {
		return nil, fmt.Errorf("saving workspace: %w", err)
	}

	rollback = false
	return &ws, nil
}

func (m *Manager) Delete(ctx context.Context, id string) error {
	ws, found, err := m.store.Get(id)
	if err != nil {
		return fmt.Errorf("getting workspace %q: %w", id, err)
	}
	if !found {
		return fmt.Errorf("workspace %q not found", id)
	}

	if err := m.sessions.KillSession(ctx, ws.Title); err != nil && !tmux.IsSessionNotFound(err) {
		return fmt.Errorf("killing session for %q: %w", ws.Title, err)
	}

	if err := m.store.Remove(id); err != nil {
		return fmt.Errorf("removing workspace %q: %w", id, err)
	}

	return nil
}

func (m *Manager) List() ([]Workspace, error) {
	return m.store.List()
}

func (m *Manager) SwitchTo(ctx context.Context, id string) error {
	ws, found, err := m.store.Get(id)
	if err != nil {
		return fmt.Errorf("getting workspace %q: %w", id, err)
	}
	if !found {
		return fmt.Errorf("workspace %q not found", id)
	}

	if err := m.sessions.SwitchClient(ctx, ws.Title); err != nil {
		return fmt.Errorf("switching to %q: %w", ws.Title, err)
	}

	return nil
}

func (m *Manager) Broadcast(ctx context.Context, prompt string, workspaceIDs []string) (BroadcastResult, error) {
	if strings.TrimSpace(prompt) == "" {
		return BroadcastResult{}, fmt.Errorf("broadcast prompt cannot be empty")
	}
	if len(workspaceIDs) == 0 {
		return BroadcastResult{}, fmt.Errorf("broadcast requires at least one workspace")
	}

	workspaces, err := m.store.List()
	if err != nil {
		return BroadcastResult{}, fmt.Errorf("listing workspaces: %w", err)
	}

	byID := make(map[string]Workspace, len(workspaces))
	for _, ws := range workspaces {
		byID[ws.ID] = ws
	}

	seen := make(map[string]struct{}, len(workspaceIDs))
	result := BroadcastResult{}

	for _, id := range workspaceIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result.Requested++

		ws, ok := byID[id]
		if !ok {
			result.Failed = append(result.Failed, BroadcastFailure{
				WorkspaceID: id,
				Err:         fmt.Errorf("workspace not found"),
			})
			continue
		}

		agentPane := ws.PaneTargets["agent"]
		if agentPane == "" {
			result.Failed = append(result.Failed, BroadcastFailure{
				WorkspaceID:    ws.ID,
				WorkspaceTitle: ws.Title,
				Err:            fmt.Errorf("workspace has no agent pane"),
			})
			continue
		}

		if err := m.sessions.SendKeys(ctx, agentPane, prompt); err != nil {
			result.Failed = append(result.Failed, BroadcastFailure{
				WorkspaceID:    ws.ID,
				WorkspaceTitle: ws.Title,
				Err:            fmt.Errorf("send prompt: %w", err),
			})
			continue
		}

		result.Delivered = append(result.Delivered, ws.Title)
	}

	return result, nil
}
