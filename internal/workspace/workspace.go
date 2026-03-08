package workspace

import (
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
)

type Backend string

const (
	BackendExternal  Backend = "external"
	BackendWorktrunk Backend = "worktrunk"
)

type CheckoutOwnership string

const (
	OwnershipAttached         CheckoutOwnership = "attached"
	OwnershipColosseumManaged CheckoutOwnership = "colosseum-managed"
)

type CheckoutCreatedFrom string

const (
	CreatedFromStandalone CheckoutCreatedFrom = "standalone"
	CreatedFromExperiment CheckoutCreatedFrom = "experiment"
)

type ExperimentStatus string

const (
	ExperimentDraft              ExperimentStatus = "draft"
	ExperimentRunning            ExperimentStatus = "running"
	ExperimentAwaitingEvaluation ExperimentStatus = "awaiting-evaluation"
	ExperimentCompleted          ExperimentStatus = "completed"
	ExperimentAbandoned          ExperimentStatus = "abandoned"
)

type EvaluationMethod string

const (
	EvaluationManual        EvaluationMethod = "manual"
	EvaluationVote          EvaluationMethod = "vote"
	EvaluationAgentAssisted EvaluationMethod = "agent-assisted"
)

type CreateMode string

const (
	CreateModeExistingCheckout CreateMode = "existing-checkout"
	CreateModeNewWorktree      CreateMode = "new-worktree"
	CreateModeExperimentRun    CreateMode = "experiment-run"
)

type ExperimentAgentStrategy string

const (
	ExperimentAgentSelected     ExperimentAgentStrategy = "selected-agent"
	ExperimentAgentAllSupported ExperimentAgentStrategy = "all-supported"
)

type Repository struct {
	ID                 string    `json:"id"`
	RootPath           string    `json:"root_path"`
	DefaultBranch      string    `json:"default_branch"`
	Backend            Backend   `json:"backend"`
	WorktrunkAvailable bool      `json:"worktrunk_available"`
	CreatedAt          time.Time `json:"created_at"`
}

type Checkout struct {
	ID                   string              `json:"id"`
	RepositoryID         string              `json:"repository_id"`
	RepoRoot             string              `json:"repo_root"`
	CheckoutPath         string              `json:"checkout_path"`
	Branch               string              `json:"branch"`
	BaseBranch           string              `json:"base_branch"`
	DefaultBranch        string              `json:"default_branch"`
	MergeBaseSHA         string              `json:"merge_base_sha,omitempty"`
	Backend              Backend             `json:"backend"`
	Ownership            CheckoutOwnership   `json:"ownership"`
	CreatedFrom          CheckoutCreatedFrom `json:"created_from"`
	DeleteBranchOnRemove bool                `json:"delete_branch_on_remove,omitempty"`
	ExperimentID         string              `json:"experiment_id,omitempty"`
	CreatedAt            time.Time           `json:"created_at"`
}

type Workspace struct {
	ID                string            `json:"id"`
	Title             string            `json:"title"`
	AgentType         agent.AgentType   `json:"agent_type"`
	ProjectPath       string            `json:"project_path"`
	Branch            string            `json:"branch"`
	BaseBranch        string            `json:"base_branch,omitempty"`
	RepositoryID      string            `json:"repository_id,omitempty"`
	CheckoutID        string            `json:"checkout_id,omitempty"`
	ExperimentID      string            `json:"experiment_id,omitempty"`
	CheckoutBackend   Backend           `json:"checkout_backend,omitempty"`
	CheckoutOwnership CheckoutOwnership `json:"checkout_ownership,omitempty"`
	Layout            LayoutType        `json:"layout"`
	Status            agent.Status      `json:"status"`
	SessionName       string            `json:"session_name"`
	PaneTargets       map[string]string `json:"pane_targets"`
	CreatedAt         time.Time         `json:"created_at"`
}

type Experiment struct {
	ID               string                  `json:"id"`
	RepositoryID     string                  `json:"repository_id"`
	RepoRoot         string                  `json:"repo_root"`
	Title            string                  `json:"title"`
	Prompt           string                  `json:"prompt,omitempty"`
	BaseBranch       string                  `json:"base_branch"`
	CheckoutIDs      []string                `json:"checkout_ids,omitempty"`
	WorkspaceIDs     []string                `json:"workspace_ids,omitempty"`
	Status           ExperimentStatus        `json:"status"`
	WinnerCheckoutID string                  `json:"winner_checkout_id,omitempty"`
	AgentStrategy    ExperimentAgentStrategy `json:"agent_strategy,omitempty"`
	CreatedAt        time.Time               `json:"created_at"`
}

type Evaluation struct {
	ID               string           `json:"id"`
	ExperimentID     string           `json:"experiment_id"`
	CheckoutIDs      []string         `json:"checkout_ids"`
	WinnerCheckoutID string           `json:"winner_checkout_id,omitempty"`
	Method           EvaluationMethod `json:"method"`
	Notes            string           `json:"notes,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
}

type State struct {
	Workspaces   []Workspace  `json:"workspaces,omitempty"`
	Repositories []Repository `json:"repositories,omitempty"`
	Checkouts    []Checkout   `json:"checkouts,omitempty"`
	Experiments  []Experiment `json:"experiments,omitempty"`
	Evaluations  []Evaluation `json:"evaluations,omitempty"`
}
