package workspace

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/tmux"
	"github.com/ramtinj/colosseum/internal/worktrunk"
)

type SessionCreator interface {
	CreateSession(ctx context.Context, name string, startDir string) (string, error)
	KillSession(ctx context.Context, name string) error
	SplitWindow(ctx context.Context, session string, horizontal bool, startDir string) (string, error)
	SwitchClient(ctx context.Context, name string) error
	SendKeys(ctx context.Context, target string, keys string, opts tmux.SendOptions) error
}

type CheckoutLifecycle interface {
	IsAvailable() bool
	ResolvePath(ctx context.Context, checkoutPath string) (worktrunk.Snapshot, error)
	Create(ctx context.Context, repoPath, branch, base string) (worktrunk.Snapshot, error)
	Remove(ctx context.Context, repoPath string, branches ...string) error
	Merge(ctx context.Context, checkoutPath, target string) error
}

type Manager struct {
	store         *Store
	sessions      SessionCreator
	checkouts     CheckoutLifecycle
	git           GitInspector
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

type StandaloneWorkspaceRequest struct {
	Title        string
	AgentType    agent.AgentType
	CheckoutPath string
	Layout       LayoutType
}

type ManagedWorkspaceRequest struct {
	Title      string
	AgentType  agent.AgentType
	RepoRoot   string
	Branch     string
	BaseBranch string
	Layout     LayoutType
}

type ExperimentRequest struct {
	Title          string
	Prompt         string
	RepoRoot       string
	BaseBranch     string
	CandidateCount int
	AgentStrategy  ExperimentAgentStrategy
	AgentType      agent.AgentType
	Layout         LayoutType
}

type ExperimentCreateResult struct {
	Experiment *Experiment
	Workspaces []*Workspace
	Broadcast  BroadcastResult
}

type createdCandidate struct {
	checkout  Checkout
	workspace *Workspace
}

func NewManager(store *Store, sessions SessionCreator, checkouts CheckoutLifecycle, prefix string) *Manager {
	return &Manager{
		store:         store,
		sessions:      sessions,
		checkouts:     checkouts,
		git:           NewGitInspector(),
		sessionPrefix: prefix,
	}
}

func (m *Manager) Create(ctx context.Context, title string, agentType agent.AgentType, projectPath string, _ string, layout LayoutType) (*Workspace, error) {
	return m.CreateStandalone(ctx, StandaloneWorkspaceRequest{
		Title:        title,
		AgentType:    agentType,
		CheckoutPath: projectPath,
		Layout:       layout,
	})
}

func (m *Manager) CreateStandalone(ctx context.Context, req StandaloneWorkspaceRequest) (*Workspace, error) {
	if err := m.validateCreate(req.Title, req.AgentType, req.Layout); err != nil {
		return nil, err
	}
	snapshot, err := m.resolveStandaloneCheckout(ctx, req.CheckoutPath)
	if err != nil {
		return nil, fmt.Errorf("resolve checkout: %w", err)
	}

	workspaceRecord, cleanupRuntime, err := m.createRuntime(ctx, req.Title, req.AgentType, snapshot.CheckoutPath, snapshot.Branch, snapshot.BaseBranch, BackendExternal, OwnershipAttached, "", req.Layout)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cleanupRuntime != nil {
			cleanupRuntime()
		}
	}()

	now := time.Now()
	repoRecord := Repository{
		ID:                 uuid.New().String(),
		RootPath:           snapshot.RepoRoot,
		DefaultBranch:      snapshot.DefaultBranch,
		Backend:            BackendExternal,
		WorktrunkAvailable: m.checkouts != nil && m.checkouts.IsAvailable(),
		CreatedAt:          now,
	}
	checkoutRecord := Checkout{
		ID:            uuid.New().String(),
		RepoRoot:      snapshot.RepoRoot,
		CheckoutPath:  snapshot.CheckoutPath,
		Branch:        snapshot.Branch,
		BaseBranch:    snapshot.BaseBranch,
		DefaultBranch: snapshot.DefaultBranch,
		MergeBaseSHA:  snapshot.MergeBaseSHA,
		Backend:       BackendExternal,
		Ownership:     OwnershipAttached,
		CreatedFrom:   CreatedFromStandalone,
		CreatedAt:     now,
	}

	if err := m.store.UpdateState(func(state *State) error {
		repo := upsertRepository(state, repoRecord)
		checkoutRecord.RepositoryID = repo.ID
		checkout := upsertCheckout(state, checkoutRecord)
		workspaceRecord.RepositoryID = repo.ID
		workspaceRecord.CheckoutID = checkout.ID
		return addWorkspace(state, *workspaceRecord)
	}); err != nil {
		return nil, fmt.Errorf("saving workspace: %w", err)
	}

	cleanupRuntime = nil
	return workspaceRecord, nil
}

func (m *Manager) resolveStandaloneCheckout(ctx context.Context, checkoutPath string) (worktrunk.Snapshot, error) {
	if m.git == nil {
		return worktrunk.Snapshot{}, fmt.Errorf("git inspector is not configured")
	}

	repoRoot, err := m.git.RepoRoot(ctx, checkoutPath)
	if err != nil {
		return worktrunk.Snapshot{}, err
	}
	branch, err := m.git.CurrentBranch(ctx, checkoutPath)
	if err != nil {
		return worktrunk.Snapshot{}, err
	}
	defaultBranch, err := m.git.DefaultBranch(ctx, repoRoot)
	if err != nil {
		return worktrunk.Snapshot{}, err
	}
	mergeBase, err := m.git.MergeBase(ctx, repoRoot, branch, defaultBranch)
	if err != nil {
		return worktrunk.Snapshot{}, err
	}

	return worktrunk.Snapshot{
		RepoRoot:      repoRoot,
		CheckoutPath:  checkoutPath,
		Branch:        branch,
		BaseBranch:    defaultBranch,
		DefaultBranch: defaultBranch,
		MergeBaseSHA:  mergeBase,
	}, nil
}

func (m *Manager) CreateWithWorktree(ctx context.Context, req ManagedWorkspaceRequest) (*Workspace, error) {
	if err := m.validateCreate(req.Title, req.AgentType, req.Layout); err != nil {
		return nil, err
	}
	if m.checkouts == nil || !m.checkouts.IsAvailable() {
		return nil, fmt.Errorf("worktrunk is not available in PATH")
	}

	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		branch = generatedStandaloneBranch(req.Title)
	}

	snapshot, err := m.checkouts.Create(ctx, req.RepoRoot, branch, req.BaseBranch)
	if err != nil {
		return nil, fmt.Errorf("create worktree: %w", err)
	}
	cleanupWorktree := func() {
		_ = m.checkouts.Remove(context.Background(), snapshot.RepoRoot, snapshot.Branch)
	}

	workspaceRecord, cleanupRuntime, err := m.createRuntime(ctx, req.Title, req.AgentType, snapshot.CheckoutPath, snapshot.Branch, snapshot.BaseBranch, BackendWorktrunk, OwnershipColosseumManaged, "", req.Layout)
	if err != nil {
		cleanupWorktree()
		return nil, err
	}
	defer func() {
		if cleanupRuntime != nil {
			cleanupRuntime()
		}
	}()

	now := time.Now()
	repoRecord := Repository{
		ID:                 uuid.New().String(),
		RootPath:           snapshot.RepoRoot,
		DefaultBranch:      snapshot.DefaultBranch,
		Backend:            BackendWorktrunk,
		WorktrunkAvailable: true,
		CreatedAt:          now,
	}
	checkoutRecord := Checkout{
		ID:            uuid.New().String(),
		RepoRoot:      snapshot.RepoRoot,
		CheckoutPath:  snapshot.CheckoutPath,
		Branch:        snapshot.Branch,
		BaseBranch:    snapshot.BaseBranch,
		DefaultBranch: snapshot.DefaultBranch,
		MergeBaseSHA:  snapshot.MergeBaseSHA,
		Backend:       BackendWorktrunk,
		Ownership:     OwnershipColosseumManaged,
		CreatedFrom:   CreatedFromStandalone,
		CreatedAt:     now,
	}

	if err := m.store.UpdateState(func(state *State) error {
		repo := upsertRepository(state, repoRecord)
		checkoutRecord.RepositoryID = repo.ID
		checkout := upsertCheckout(state, checkoutRecord)
		workspaceRecord.RepositoryID = repo.ID
		workspaceRecord.CheckoutID = checkout.ID
		return addWorkspace(state, *workspaceRecord)
	}); err != nil {
		cleanupWorktree()
		return nil, fmt.Errorf("saving workspace: %w", err)
	}

	cleanupRuntime = nil
	return workspaceRecord, nil
}

func (m *Manager) CreateExperiment(ctx context.Context, req ExperimentRequest) (*ExperimentCreateResult, error) {
	if err := m.validateCreate(req.Title, req.AgentType, req.Layout); err != nil {
		return nil, err
	}
	if m.checkouts == nil || !m.checkouts.IsAvailable() {
		return nil, fmt.Errorf("worktrunk is not available in PATH")
	}

	repoSnapshot, err := m.checkouts.ResolvePath(ctx, req.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve repository: %w", err)
	}

	baseBranch := strings.TrimSpace(req.BaseBranch)
	if baseBranch == "" {
		baseBranch = repoSnapshot.DefaultBranch
	}

	agents := experimentAgents(req.AgentStrategy, req.AgentType, req.CandidateCount)
	if len(agents) == 0 {
		return nil, fmt.Errorf("experiment requires at least one candidate")
	}

	now := time.Now()
	experiment := Experiment{
		ID:            uuid.New().String(),
		Title:         req.Title,
		RepoRoot:      repoSnapshot.RepoRoot,
		BaseBranch:    baseBranch,
		Prompt:        strings.TrimSpace(req.Prompt),
		Status:        ExperimentRunning,
		AgentStrategy: normalizeExperimentAgentStrategy(req.AgentStrategy),
		CreatedAt:     now,
	}
	repository := Repository{
		ID:                 uuid.New().String(),
		RootPath:           repoSnapshot.RepoRoot,
		DefaultBranch:      repoSnapshot.DefaultBranch,
		Backend:            BackendWorktrunk,
		WorktrunkAvailable: true,
		CreatedAt:          now,
	}

	created := make([]createdCandidate, 0, len(agents))
	for i, agentType := range agents {
		branch := generatedExperimentBranch(req.Title, agentType, i+1)
		snapshot, err := m.checkouts.Create(ctx, req.RepoRoot, branch, baseBranch)
		if err != nil {
			m.rollbackCandidates(created)
			return nil, fmt.Errorf("create experiment checkout %q: %w", branch, err)
		}

		workspaceTitle := generatedExperimentWorkspaceTitle(req.Title, agentType, i+1, len(agents))
		workspaceRecord, cleanupRuntime, err := m.createRuntime(ctx, workspaceTitle, agentType, snapshot.CheckoutPath, snapshot.Branch, snapshot.BaseBranch, BackendWorktrunk, OwnershipColosseumManaged, experiment.ID, req.Layout)
		if err != nil {
			_ = m.checkouts.Remove(context.Background(), snapshot.RepoRoot, snapshot.Branch)
			m.rollbackCandidates(created)
			return nil, err
		}

		checkoutRecord := Checkout{
			ID:            uuid.New().String(),
			RepoRoot:      snapshot.RepoRoot,
			CheckoutPath:  snapshot.CheckoutPath,
			Branch:        snapshot.Branch,
			BaseBranch:    snapshot.BaseBranch,
			DefaultBranch: snapshot.DefaultBranch,
			MergeBaseSHA:  snapshot.MergeBaseSHA,
			Backend:       BackendWorktrunk,
			Ownership:     OwnershipColosseumManaged,
			CreatedFrom:   CreatedFromExperiment,
			ExperimentID:  experiment.ID,
			CreatedAt:     now,
		}

		created = append(created, createdCandidate{
			checkout:  checkoutRecord,
			workspace: workspaceRecord,
		})
		_ = cleanupRuntime
	}

	if err := m.store.UpdateState(func(state *State) error {
		repo := upsertRepository(state, repository)
		experiment.RepositoryID = repo.ID

		checkoutIDs := make([]string, 0, len(created))
		workspaceIDs := make([]string, 0, len(created))
		for i := range created {
			created[i].checkout.RepositoryID = repo.ID
			checkout := upsertCheckout(state, created[i].checkout)
			created[i].workspace.RepositoryID = repo.ID
			created[i].workspace.CheckoutID = checkout.ID
			created[i].workspace.ExperimentID = experiment.ID
			if err := addWorkspace(state, *created[i].workspace); err != nil {
				return err
			}
			checkoutIDs = append(checkoutIDs, checkout.ID)
			workspaceIDs = append(workspaceIDs, created[i].workspace.ID)
		}
		experiment.CheckoutIDs = checkoutIDs
		experiment.WorkspaceIDs = workspaceIDs
		state.Experiments = append(state.Experiments, experiment)
		return nil
	}); err != nil {
		m.rollbackCandidates(created)
		return nil, fmt.Errorf("saving experiment: %w", err)
	}

	result := &ExperimentCreateResult{
		Experiment: &experiment,
		Workspaces: make([]*Workspace, 0, len(created)),
	}
	for i := range created {
		result.Workspaces = append(result.Workspaces, created[i].workspace)
	}

	if experiment.Prompt != "" {
		ids := make([]string, 0, len(result.Workspaces))
		for _, ws := range result.Workspaces {
			ids = append(ids, ws.ID)
		}
		broadcast, err := m.Broadcast(ctx, experiment.Prompt, ids)
		if err != nil {
			return result, fmt.Errorf("broadcast experiment prompt: %w", err)
		}
		result.Broadcast = broadcast
	}

	return result, nil
}

func (m *Manager) Delete(ctx context.Context, id string) error {
	state, err := m.store.LoadState()
	if err != nil {
		return fmt.Errorf("loading workspace state: %w", err)
	}

	ws, found := findWorkspace(state.Workspaces, id)
	if !found {
		return fmt.Errorf("workspace %q not found", id)
	}

	removeCheckout := false
	var checkoutRecord Checkout
	if ws.CheckoutID != "" {
		if record, ok := findCheckout(state.Checkouts, ws.CheckoutID); ok {
			checkoutRecord = record
			if ws.CheckoutOwnership == OwnershipColosseumManaged && countWorkspaceRefs(state.Workspaces, ws.CheckoutID) == 1 {
				removeCheckout = true
			}
		}
	}

	if removeCheckout {
		if err := m.checkouts.Remove(ctx, checkoutRecord.RepoRoot, checkoutRecord.Branch); err != nil {
			return fmt.Errorf("removing worktree %q: %w", checkoutRecord.Branch, err)
		}
	}

	if err := m.sessions.KillSession(ctx, m.workspaceSessionName(ws)); err != nil && !tmux.IsSessionNotFound(err) {
		return fmt.Errorf("killing session for %q: %w", ws.Title, err)
	}

	if err := m.store.UpdateState(func(next *State) error {
		next.Workspaces = filterWorkspaces(next.Workspaces, id)
		if removeCheckout {
			next.Checkouts = filterCheckouts(next.Checkouts, checkoutRecord.ID)
		}
		for i := range next.Experiments {
			next.Experiments[i].WorkspaceIDs = filterIDs(next.Experiments[i].WorkspaceIDs, id)
			if removeCheckout {
				next.Experiments[i].CheckoutIDs = filterIDs(next.Experiments[i].CheckoutIDs, checkoutRecord.ID)
				if next.Experiments[i].WinnerCheckoutID == checkoutRecord.ID {
					next.Experiments[i].WinnerCheckoutID = ""
				}
			}
			switch {
			case len(next.Experiments[i].CheckoutIDs) == 0:
				next.Experiments[i].Status = ExperimentAbandoned
			case len(next.Experiments[i].WorkspaceIDs) == 0 && next.Experiments[i].WinnerCheckoutID == "":
				next.Experiments[i].Status = ExperimentAwaitingEvaluation
			}
		}
		return nil
	}); err != nil {
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

	if err := m.sessions.SwitchClient(ctx, m.workspaceSessionName(ws)); err != nil {
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

		opts := tmux.SendOptions{}
		if def, ok := agent.Get(ws.AgentType); ok {
			opts.InputDelay = def.InputDelay
			opts.ForcePaste = def.PasteSingleLine && !strings.Contains(prompt, "\n")
			opts.DisableBracketedPaste = def.DisableBracketedPasteForMultiline && strings.Contains(prompt, "\n")
		}

		if err := m.sessions.SendKeys(ctx, agentPane, prompt, opts); err != nil {
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

func (m *Manager) validateCreate(title string, agentType agent.AgentType, layout LayoutType) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("workspace title cannot be empty")
	}
	if !agent.IsSupported(agentType) {
		return fmt.Errorf("unsupported agent type %q", agentType)
	}
	if !IsValidLayout(layout) {
		return fmt.Errorf("invalid layout %q", layout)
	}

	workspaces, err := m.store.List()
	if err != nil {
		return fmt.Errorf("checking existing workspaces: %w", err)
	}
	for _, ws := range workspaces {
		if ws.Title == title {
			return fmt.Errorf("workspace %q already exists", title)
		}
	}
	return nil
}

func (m *Manager) createRuntime(ctx context.Context, title string, agentType agent.AgentType, projectPath string, branch string, baseBranch string, backend Backend, ownership CheckoutOwnership, experimentID string, layout LayoutType) (*Workspace, func(), error) {
	id := uuid.New().String()
	sessionName := m.prefixedSessionName(title)

	agentPaneID, err := m.sessions.CreateSession(ctx, sessionName, projectPath)
	if err != nil {
		return nil, nil, fmt.Errorf("creating session for %q: %w", title, err)
	}
	cleanup := func() {
		_ = m.sessions.KillSession(context.Background(), sessionName)
	}

	paneTargets := map[string]string{"agent": agentPaneID}
	if layout.PaneCount() >= 2 {
		paneID, err := m.sessions.SplitWindow(ctx, sessionName, true, projectPath)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("splitting window for shell pane: %w", err)
		}
		paneTargets["shell"] = paneID
	}
	if layout.PaneCount() >= 3 {
		paneID, err := m.sessions.SplitWindow(ctx, sessionName, false, projectPath)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("splitting window for logs pane: %w", err)
		}
		paneTargets["logs"] = paneID
	}

	def, ok := agent.Get(agentType)
	if !ok {
		cleanup()
		return nil, nil, fmt.Errorf("agent type %q is not registered", agentType)
	}
	launchCmd := def.Binary
	for _, flag := range def.LaunchFlags {
		launchCmd += " " + flag
	}
	if err := m.sessions.SendKeys(ctx, agentPaneID, launchCmd, tmux.SendOptions{}); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("launching agent %q: %w", agentType, err)
	}

	return &Workspace{
		ID:                id,
		Title:             title,
		AgentType:         agentType,
		ProjectPath:       projectPath,
		Branch:            branch,
		BaseBranch:        baseBranch,
		CheckoutBackend:   backend,
		CheckoutOwnership: ownership,
		ExperimentID:      experimentID,
		Layout:            layout,
		Status:            agent.StatusIdle,
		SessionName:       sessionName,
		PaneTargets:       paneTargets,
		CreatedAt:         time.Now(),
	}, cleanup, nil
}

func (m *Manager) rollbackCandidates(created []createdCandidate) {
	for i := len(created) - 1; i >= 0; i-- {
		_ = m.sessions.KillSession(context.Background(), created[i].workspace.SessionName)
		_ = m.checkouts.Remove(context.Background(), created[i].checkout.RepoRoot, created[i].checkout.Branch)
	}
}

func (m *Manager) prefixedSessionName(title string) string {
	return m.sessionPrefix + title
}

func (m *Manager) workspaceSessionName(ws Workspace) string {
	if ws.SessionName != "" {
		return ws.SessionName
	}
	return m.prefixedSessionName(ws.Title)
}

func upsertRepository(state *State, candidate Repository) Repository {
	for i := range state.Repositories {
		if state.Repositories[i].RootPath == candidate.RootPath {
			state.Repositories[i].DefaultBranch = candidate.DefaultBranch
			state.Repositories[i].Backend = candidate.Backend
			state.Repositories[i].WorktrunkAvailable = candidate.WorktrunkAvailable
			return state.Repositories[i]
		}
	}
	state.Repositories = append(state.Repositories, candidate)
	return candidate
}

func upsertCheckout(state *State, candidate Checkout) Checkout {
	for i := range state.Checkouts {
		if state.Checkouts[i].RepoRoot == candidate.RepoRoot && state.Checkouts[i].CheckoutPath == candidate.CheckoutPath {
			state.Checkouts[i].Branch = candidate.Branch
			state.Checkouts[i].BaseBranch = candidate.BaseBranch
			state.Checkouts[i].DefaultBranch = candidate.DefaultBranch
			state.Checkouts[i].MergeBaseSHA = candidate.MergeBaseSHA
			state.Checkouts[i].Backend = candidate.Backend
			state.Checkouts[i].Ownership = candidate.Ownership
			state.Checkouts[i].CreatedFrom = candidate.CreatedFrom
			state.Checkouts[i].ExperimentID = candidate.ExperimentID
			return state.Checkouts[i]
		}
	}
	state.Checkouts = append(state.Checkouts, candidate)
	return candidate
}

func addWorkspace(state *State, candidate Workspace) error {
	for _, ws := range state.Workspaces {
		if ws.Title == candidate.Title {
			return fmt.Errorf("workspace %q already exists", candidate.Title)
		}
	}
	state.Workspaces = append(state.Workspaces, candidate)
	return nil
}

func findWorkspace(workspaces []Workspace, id string) (Workspace, bool) {
	for _, ws := range workspaces {
		if ws.ID == id {
			return ws, true
		}
	}
	return Workspace{}, false
}

func findCheckout(checkouts []Checkout, id string) (Checkout, bool) {
	for _, checkout := range checkouts {
		if checkout.ID == id {
			return checkout, true
		}
	}
	return Checkout{}, false
}

func countWorkspaceRefs(workspaces []Workspace, checkoutID string) int {
	count := 0
	for _, ws := range workspaces {
		if ws.CheckoutID == checkoutID {
			count++
		}
	}
	return count
}

func filterWorkspaces(workspaces []Workspace, id string) []Workspace {
	filtered := make([]Workspace, 0, len(workspaces))
	for _, ws := range workspaces {
		if ws.ID != id {
			filtered = append(filtered, ws)
		}
	}
	return filtered
}

func filterCheckouts(checkouts []Checkout, id string) []Checkout {
	filtered := make([]Checkout, 0, len(checkouts))
	for _, checkout := range checkouts {
		if checkout.ID != id {
			filtered = append(filtered, checkout)
		}
	}
	return filtered
}

func filterIDs(values []string, needle string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if value != needle {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func generatedStandaloneBranch(title string) string {
	return fmt.Sprintf("feat-%s-%s", slugify(title), time.Now().Format("20060102"))
}

func generatedExperimentBranch(title string, agentType agent.AgentType, index int) string {
	return fmt.Sprintf("exp-%s-%s-%s-a%d", slugify(title), time.Now().Format("20060102"), agentType, index)
}

func generatedExperimentWorkspaceTitle(title string, agentType agent.AgentType, index int, total int) string {
	label := string(agentType)
	if len(label) > 0 {
		label = strings.ToUpper(label[:1]) + label[1:]
	}
	if total <= 1 {
		return fmt.Sprintf("%s %s", title, label)
	}
	return fmt.Sprintf("%s %s %d", title, label, index)
}

func slugify(value string) string {
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = re.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "workspace"
	}
	return slug
}

func experimentAgents(strategy ExperimentAgentStrategy, selected agent.AgentType, count int) []agent.AgentType {
	switch normalizeExperimentAgentStrategy(strategy) {
	case ExperimentAgentAllSupported:
		return agent.Supported()
	default:
		if count <= 0 {
			count = 2
		}
		agents := make([]agent.AgentType, 0, count)
		for i := 0; i < count; i++ {
			agents = append(agents, selected)
		}
		return agents
	}
}

func normalizeExperimentAgentStrategy(strategy ExperimentAgentStrategy) ExperimentAgentStrategy {
	if strategy == ExperimentAgentAllSupported {
		return strategy
	}
	return ExperimentAgentSelected
}
