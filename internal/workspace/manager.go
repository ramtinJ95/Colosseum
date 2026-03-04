package workspace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ramtinj/colosseum/internal/agent"
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

func NewManager(store *Store, sessions SessionCreator, prefix string) *Manager {
	return &Manager{
		store:         store,
		sessions:      sessions,
		sessionPrefix: prefix,
	}
}

func (m *Manager) Create(ctx context.Context, title string, agentType agent.AgentType, projectPath string, branch string, layout LayoutType) (*Workspace, error) {
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

	def, _ := agent.Get(agentType)
	if def != nil {
		launchCmd := def.Binary
		for _, flag := range def.LaunchFlags {
			launchCmd += " " + flag
		}
		_ = m.sessions.SendKeys(ctx, agentPaneID, launchCmd)
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

	_ = m.sessions.KillSession(ctx, ws.Title)

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
