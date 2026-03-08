package worktrunk

import (
	"context"
	"fmt"
	"testing"
)

type mockCommander struct {
	responses map[string]string
	errs      map[string]error
	calls     [][]string
}

func (m *mockCommander) Run(_ context.Context, args ...string) (string, error) {
	m.calls = append(m.calls, append([]string(nil), args...))
	key := fmt.Sprint(args)
	if err, ok := m.errs[key]; ok {
		return "", err
	}
	return m.responses[key], nil
}

func TestClientListParsesJSON(t *testing.T) {
	wt := &mockCommander{
		responses: map[string]string{
			fmt.Sprint([]string{"-C", "/repo", "list", "--format=json"}): `[{"branch":"feature","path":"/repo/.worktrees/feature","kind":"worktree","is_main":false,"is_current":false}]`,
		},
	}
	client := NewClientWithCommanders(wt, &mockCommander{})

	infos, err := client.List(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("infos = %d, want 1", len(infos))
	}
	if infos[0].Branch != "feature" {
		t.Fatalf("branch = %q, want feature", infos[0].Branch)
	}
}

func TestClientCreateResolvesAuthoritativePathViaList(t *testing.T) {
	wt := &mockCommander{
		responses: map[string]string{
			fmt.Sprint([]string{"-C", "/repo", "switch", "--create", "--no-cd", "--base", "main", "feature"}): ``,
			fmt.Sprint([]string{"-C", "/repo", "list", "--format=json"}):                                      `[{"branch":"main","path":"/repo","kind":"worktree","is_main":true,"is_current":true},{"branch":"feature","path":"/repo/.worktrees/feature","kind":"worktree","is_main":false,"is_current":false}]`,
		},
	}
	git := &mockCommander{
		responses: map[string]string{
			fmt.Sprint([]string{"-C", "/repo", "merge-base", "feature", "main"}): "abc123",
		},
	}
	client := NewClientWithCommanders(wt, git)

	snapshot, err := client.Create(context.Background(), "/repo", "feature", "main")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if snapshot.CheckoutPath != "/repo/.worktrees/feature" {
		t.Fatalf("checkout path = %q, want /repo/.worktrees/feature", snapshot.CheckoutPath)
	}
	if snapshot.MergeBaseSHA != "abc123" {
		t.Fatalf("merge base = %q, want abc123", snapshot.MergeBaseSHA)
	}
}

func TestClientResolvePathAcceptsNestedDirectories(t *testing.T) {
	wt := &mockCommander{
		responses: map[string]string{
			fmt.Sprint([]string{"-C", "/repo/.worktrees/feature", "list", "--format=json"}): `[{"branch":"main","path":"/repo","kind":"worktree","is_main":true,"is_current":false},{"branch":"feature","path":"/repo/.worktrees/feature","kind":"worktree","is_main":false,"is_current":true}]`,
		},
	}
	git := &mockCommander{
		responses: map[string]string{
			fmt.Sprint([]string{"-C", "/repo/.worktrees/feature/pkg", "rev-parse", "--show-toplevel"}): "/repo/.worktrees/feature",
			fmt.Sprint([]string{"-C", "/repo/.worktrees/feature", "merge-base", "feature", "main"}):    "abc123",
		},
	}
	client := NewClientWithCommanders(wt, git)

	snapshot, err := client.ResolvePath(context.Background(), "/repo/.worktrees/feature/pkg")
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	if snapshot.CheckoutPath != "/repo/.worktrees/feature" {
		t.Fatalf("checkout path = %q, want /repo/.worktrees/feature", snapshot.CheckoutPath)
	}
	if snapshot.Branch != "feature" {
		t.Fatalf("branch = %q, want feature", snapshot.Branch)
	}
	if snapshot.DefaultBranch != "main" {
		t.Fatalf("default branch = %q, want main", snapshot.DefaultBranch)
	}
	if snapshot.MergeBaseSHA != "abc123" {
		t.Fatalf("merge base = %q, want abc123", snapshot.MergeBaseSHA)
	}
}
