package workspace

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/worktrunk"
)

func TestManagerDeleteManagedCheckoutKeepsRuntimeWhenRemoveFails(t *testing.T) {
	h := newManagerHarness(t)
	req := defaultManagedWorkspaceRequest("managed-remove-fail", agent.Codex)
	req.Branch = "feature-remove"
	ws := h.mustCreateManaged(t, req)

	h.checkouts.removeErr = errors.New("dirty worktree")
	if err := h.mgr.Delete(h.ctx, ws.ID); err == nil {
		t.Fatal("expected delete to fail")
	}
	if len(h.sessions.killCalls) != 0 {
		t.Fatalf("kill calls = %v, want none", h.sessions.killCalls)
	}

	assertStateCounts(t, h.mustState(t), 1, 1, 1, 0)
}

func TestManagerDeleteManagedCheckoutRetriesWhenCheckoutAlreadyGone(t *testing.T) {
	h := newManagerHarness(t)
	h.sessions.killErr = errors.New("tmux unavailable")
	req := defaultManagedWorkspaceRequest("managed-kill-fail", agent.Codex)
	req.Branch = "feature-kill"
	ws := h.mustCreateManaged(t, req)

	if err := h.mgr.Delete(h.ctx, ws.ID); err == nil {
		t.Fatal("expected delete to fail")
	}
	if len(h.checkouts.removeCalls) != 1 {
		t.Fatalf("remove calls = %v, want one successful removal before kill failure", h.checkouts.removeCalls)
	}
	assertStateCounts(t, h.mustState(t), 1, 1, 1, 0)

	h.sessions.killErr = nil
	h.checkouts.removeErr = &worktrunk.CommandError{Binary: "wt", Args: []string{"remove"}, Stderr: "worktree not found"}
	if err := h.mgr.Delete(h.ctx, ws.ID); err != nil {
		t.Fatalf("retry Delete: %v", err)
	}
	assertStateCounts(t, h.mustState(t), 0, 1, 0, 0)
}

func TestManagerCreateWithWorktreePersistsManagedCheckout(t *testing.T) {
	events := []string{}
	h := newManagerHarness(t)
	h.sessions.events = &events
	h.checkouts.events = &events

	req := defaultManagedWorkspaceRequest("managed", agent.Claude)
	req.Branch = "feature-auth"
	ws := h.mustCreateManaged(t, req)

	if ws.Branch != "feature-auth" {
		t.Fatalf("workspace branch = %q, want feature-auth", ws.Branch)
	}
	if ws.CheckoutOwnership != OwnershipColosseumManaged {
		t.Fatalf("workspace ownership = %q, want %q", ws.CheckoutOwnership, OwnershipColosseumManaged)
	}

	state := h.mustState(t)
	assertStateCounts(t, state, 1, 1, 1, 0)
	if state.Checkouts[0].Ownership != OwnershipColosseumManaged {
		t.Fatalf("checkout ownership = %q, want managed", state.Checkouts[0].Ownership)
	}

	if err := h.mgr.Delete(h.ctx, ws.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(h.checkouts.removeCalls) != 1 {
		t.Fatalf("remove calls = %d, want 1", len(h.checkouts.removeCalls))
	}
	if got := h.checkouts.removeCalls[0].Branches; len(got) != 1 || got[0] != "feature-auth" {
		t.Fatalf("remove branches = %v, want [feature-auth]", got)
	}
	if len(events) < 2 || events[0] != "remove:feature-auth" || events[1] != "kill:"+ws.SessionName {
		t.Fatalf("events = %v, want remove before kill", events)
	}
}

func TestManagerCreateWithWorktreeGeneratesUniqueBranchesAcrossRecreate(t *testing.T) {
	h := newManagerHarness(t)
	req := defaultManagedWorkspaceRequest("Repeated Workspace", agent.Codex)

	first := h.mustCreateManaged(t, req)
	if err := h.mgr.Delete(h.ctx, first.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_ = h.mustCreateManaged(t, req)

	if len(h.checkouts.createCalls) != 2 {
		t.Fatalf("create calls = %d, want 2", len(h.checkouts.createCalls))
	}
	if h.checkouts.createCalls[0].Branch == h.checkouts.createCalls[1].Branch {
		t.Fatalf("generated branch reused across recreate: %q", h.checkouts.createCalls[0].Branch)
	}
	if !strings.HasPrefix(h.checkouts.createCalls[0].Branch, "feat-repeated-workspace-") {
		t.Fatalf("first generated branch = %q, want feat-repeated-workspace-*", h.checkouts.createCalls[0].Branch)
	}
	if !strings.HasPrefix(h.checkouts.createCalls[1].Branch, "feat-repeated-workspace-") {
		t.Fatalf("second generated branch = %q, want feat-repeated-workspace-*", h.checkouts.createCalls[1].Branch)
	}
}

func TestManagerCreateExperimentCreatesCandidatesAndBroadcasts(t *testing.T) {
	h := newManagerHarness(t)
	req := defaultExperimentRequest("Auth fix")
	req.Prompt = "ship the fix"

	result := h.mustCreateExperiment(t, req)

	if got := len(result.Workspaces); got != len(agent.Supported()) {
		t.Fatalf("workspaces = %d, want %d", got, len(agent.Supported()))
	}
	if result.Experiment == nil {
		t.Fatal("expected experiment metadata")
	}
	if result.Experiment.Status != ExperimentRunning {
		t.Fatalf("experiment status = %q, want %q", result.Experiment.Status, ExperimentRunning)
	}
	if len(result.Broadcast.Delivered) != len(agent.Supported()) {
		t.Fatalf("broadcast delivered = %d, want %d", len(result.Broadcast.Delivered), len(agent.Supported()))
	}
	assertStateCounts(t, h.mustState(t), len(agent.Supported()), 1, len(agent.Supported()), 1)
}

func TestManagerCreateExperimentGeneratesUniqueBranchesAcrossRecreate(t *testing.T) {
	h := newManagerHarness(t)
	req := defaultExperimentRequest("Auth fix")

	first := h.mustCreateExperiment(t, req)
	for _, ws := range first.Workspaces {
		if err := h.mgr.Delete(h.ctx, ws.ID); err != nil {
			t.Fatalf("Delete(%s): %v", ws.ID, err)
		}
	}
	_ = h.mustCreateExperiment(t, req)

	perRun := len(agent.Supported())
	if len(h.checkouts.createCalls) != perRun*2 {
		t.Fatalf("create calls = %d, want %d", len(h.checkouts.createCalls), perRun*2)
	}
	for i := 0; i < perRun; i++ {
		firstBranch := h.checkouts.createCalls[i].Branch
		secondBranch := h.checkouts.createCalls[perRun+i].Branch
		if firstBranch == secondBranch {
			t.Fatalf("candidate %d reused generated branch %q across recreate", i, firstBranch)
		}
		if !strings.HasPrefix(firstBranch, "exp-auth-fix-") {
			t.Fatalf("first branch = %q, want exp-auth-fix-*", firstBranch)
		}
		if !strings.HasPrefix(secondBranch, "exp-auth-fix-") {
			t.Fatalf("second branch = %q, want exp-auth-fix-*", secondBranch)
		}
	}
}

func TestManagerCreateStandaloneDoesNotRequireCheckoutLifecycle(t *testing.T) {
	store := newTestStore(t)
	sessions := &mockSessionCreator{}
	git := &mockGitInspector{
		repoRoots: map[string]string{
			"/repo/subdir": "/repo",
		},
		currentBranches: map[string]string{
			"/repo/subdir": "/repo-branch",
		},
		defaultBranches: map[string]string{
			"/repo": "main",
		},
		mergeBases: map[string]string{
			"/repo|/repo-branch|main": "abc123",
		},
	}

	mgr := NewManager(store, sessions, nil, "colo-")
	mgr.git = git

	ws, err := mgr.CreateStandalone(context.Background(), StandaloneWorkspaceRequest{
		Title:        "standalone",
		AgentType:    agent.Claude,
		CheckoutPath: "/repo/subdir",
		Layout:       LayoutAgent,
	})
	if err != nil {
		t.Fatalf("CreateStandalone: %v", err)
	}
	if ws.ProjectPath != "/repo/subdir" {
		t.Fatalf("project path = %q, want /repo/subdir", ws.ProjectPath)
	}

	state, err := store.LoadState()
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	assertStateCounts(t, state, 1, 1, 1, 0)
	if state.Checkouts[0].CheckoutPath != "/repo/subdir" {
		t.Fatalf("checkout path = %q, want /repo/subdir", state.Checkouts[0].CheckoutPath)
	}
	if state.Checkouts[0].RepoRoot != "/repo" {
		t.Fatalf("repo root = %q, want /repo", state.Checkouts[0].RepoRoot)
	}
	if state.Repositories[0].WorktrunkAvailable {
		t.Fatal("expected worktrunk availability to stay false without checkout lifecycle")
	}
}

func TestManagerCreateStandaloneSkipsCheckoutResolution(t *testing.T) {
	h := newManagerHarness(t)
	h.git.repoRoots["/repo"] = "/repo"
	h.git.currentBranches["/repo"] = "main"
	h.git.defaultBranches["/repo"] = "main"

	h.mustCreateStandalone(t, StandaloneWorkspaceRequest{
		Title:        "standalone",
		AgentType:    agent.Claude,
		CheckoutPath: "/repo",
		Layout:       LayoutAgent,
	})
	if len(h.checkouts.resolveCalls) != 0 {
		t.Fatalf("resolve calls = %v, want none", h.checkouts.resolveCalls)
	}
}
