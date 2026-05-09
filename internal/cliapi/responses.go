package cliapi

import (
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type Workspace struct {
	ID                string                      `json:"id"`
	Title             string                      `json:"title"`
	AgentType         agent.AgentType             `json:"agent_type"`
	ProjectPath       string                      `json:"project_path"`
	Branch            string                      `json:"branch,omitempty"`
	BaseBranch        string                      `json:"base_branch,omitempty"`
	RepositoryID      string                      `json:"repository_id,omitempty"`
	CheckoutID        string                      `json:"checkout_id,omitempty"`
	ExperimentID      string                      `json:"experiment_id,omitempty"`
	CheckoutBackend   workspace.Backend           `json:"checkout_backend,omitempty"`
	CheckoutOwnership workspace.CheckoutOwnership `json:"checkout_ownership,omitempty"`
	Layout            workspace.LayoutType        `json:"layout"`
	Status            string                      `json:"status"`
	SessionName       string                      `json:"session_name"`
	PaneTargets       map[string]string           `json:"pane_targets,omitempty"`
	CreatedAt         time.Time                   `json:"created_at"`
}

type WorkspaceListResponse struct {
	Workspaces []Workspace `json:"workspaces"`
}

type WorkspaceResponse struct {
	Workspace Workspace `json:"workspace"`
}

type StatusResponse struct {
	Workspace Workspace `json:"workspace"`
	Status    string    `json:"status"`
}

type Pane struct {
	Role   string `json:"role,omitempty"`
	Target string `json:"target"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type PaneListResponse struct {
	Workspace Workspace `json:"workspace"`
	Panes     []Pane    `json:"panes"`
}

type PaneReadResponse struct {
	Workspace Workspace `json:"workspace"`
	Pane      string    `json:"pane"`
	Target    string    `json:"target"`
	Lines     int       `json:"lines"`
	Content   string    `json:"content"`
}

type WaitStatusResponse struct {
	Workspace Workspace `json:"workspace"`
	Status    string    `json:"status"`
	Desired   string    `json:"desired"`
	ElapsedMS int64     `json:"elapsed_ms"`
}

type WaitOutputResponse struct {
	Workspace Workspace `json:"workspace"`
	Pane      string    `json:"pane"`
	Target    string    `json:"target"`
	Match     string    `json:"match"`
	ElapsedMS int64     `json:"elapsed_ms"`
}

func NewWorkspace(ws workspace.Workspace) Workspace {
	paneTargets := make(map[string]string, len(ws.PaneTargets))
	for role, target := range ws.PaneTargets {
		paneTargets[role] = target
	}
	return Workspace{
		ID:                ws.ID,
		Title:             ws.Title,
		AgentType:         ws.AgentType,
		ProjectPath:       ws.ProjectPath,
		Branch:            ws.Branch,
		BaseBranch:        ws.BaseBranch,
		RepositoryID:      ws.RepositoryID,
		CheckoutID:        ws.CheckoutID,
		ExperimentID:      ws.ExperimentID,
		CheckoutBackend:   ws.CheckoutBackend,
		CheckoutOwnership: ws.CheckoutOwnership,
		Layout:            ws.Layout,
		Status:            ws.Status.String(),
		SessionName:       ws.SessionName,
		PaneTargets:       paneTargets,
		CreatedAt:         ws.CreatedAt,
	}
}

func NewWorkspaces(workspaces []workspace.Workspace) []Workspace {
	result := make([]Workspace, 0, len(workspaces))
	for _, ws := range workspaces {
		result = append(result, NewWorkspace(ws))
	}
	return result
}
