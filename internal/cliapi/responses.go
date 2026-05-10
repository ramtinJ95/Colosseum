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

type BroadcastFailure struct {
	WorkspaceID    string `json:"workspace_id,omitempty"`
	WorkspaceTitle string `json:"workspace_title,omitempty"`
	Error          string `json:"error"`
}

type BroadcastResult struct {
	Requested int                `json:"requested"`
	Delivered []string           `json:"delivered,omitempty"`
	Failed    []BroadcastFailure `json:"failed,omitempty"`
}

type WorkspaceCreateResponse struct {
	Workspace  *Workspace            `json:"workspace,omitempty"`
	Workspaces []Workspace           `json:"workspaces,omitempty"`
	Experiment *workspace.Experiment `json:"experiment,omitempty"`
	Broadcast  BroadcastResult       `json:"broadcast,omitempty"`
}

type WorkspaceActionResponse struct {
	Workspace Workspace `json:"workspace"`
	Action    string    `json:"action"`
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

type PaneActionResponse struct {
	Workspace Workspace `json:"workspace"`
	Pane      string    `json:"pane"`
	Target    string    `json:"target"`
	Action    string    `json:"action"`
}

type AgentStatusReport struct {
	WorkspaceID string          `json:"workspace_id"`
	Pane        string          `json:"pane"`
	AgentType   agent.AgentType `json:"agent_type,omitempty"`
	Status      string          `json:"status"`
	Source      string          `json:"source,omitempty"`
	ReportedAt  time.Time       `json:"reported_at"`
}

type AgentReportResponse struct {
	Workspace Workspace         `json:"workspace"`
	Report    AgentStatusReport `json:"report"`
	Action    string            `json:"action"`
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

func NewBroadcastResult(result workspace.BroadcastResult) BroadcastResult {
	failures := make([]BroadcastFailure, 0, len(result.Failed))
	for _, failure := range result.Failed {
		message := ""
		if failure.Err != nil {
			message = failure.Err.Error()
		}
		failures = append(failures, BroadcastFailure{
			WorkspaceID:    failure.WorkspaceID,
			WorkspaceTitle: failure.WorkspaceTitle,
			Error:          message,
		})
	}
	return BroadcastResult{
		Requested: result.Requested,
		Delivered: append([]string(nil), result.Delivered...),
		Failed:    failures,
	}
}

func NewAgentStatusReport(report workspace.AgentStatusReport) AgentStatusReport {
	return AgentStatusReport{
		WorkspaceID: report.WorkspaceID,
		Pane:        report.Pane,
		AgentType:   report.AgentType,
		Status:      report.Status,
		Source:      report.Source,
		ReportedAt:  report.ReportedAt,
	}
}
