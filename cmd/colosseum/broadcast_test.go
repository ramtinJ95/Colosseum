package main

import (
	"testing"

	"github.com/ramtinj/colosseum/internal/workspace"
)

func TestParseBroadcastWorkspaceNames(t *testing.T) {
	names := parseBroadcastWorkspaceNames(" alpha, beta ,, gamma ")
	want := []string{"alpha", "beta", "gamma"}

	if len(names) != len(want) {
		t.Fatalf("len(names) = %d, want %d (%v)", len(names), len(want), names)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestResolveBroadcastWorkspaceIDs(t *testing.T) {
	workspaces := []workspace.Workspace{
		{ID: "ws-1", Title: "alpha"},
		{ID: "ws-2", Title: "beta"},
	}

	ids, err := resolveBroadcastWorkspaceIDs(workspaces, []string{"beta", "alpha"})
	if err != nil {
		t.Fatalf("resolveBroadcastWorkspaceIDs: %v", err)
	}

	want := []string{"ws-2", "ws-1"}
	if len(ids) != len(want) {
		t.Fatalf("len(ids) = %d, want %d (%v)", len(ids), len(want), ids)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("ids[%d] = %q, want %q", i, ids[i], want[i])
		}
	}
}

func TestResolveBroadcastWorkspaceIDsReturnsMissingWorkspace(t *testing.T) {
	workspaces := []workspace.Workspace{
		{ID: "ws-1", Title: "alpha"},
	}

	_, err := resolveBroadcastWorkspaceIDs(workspaces, []string{"missing"})
	if err == nil {
		t.Fatal("expected missing workspace error")
	}
}

func TestBroadcastStatusLine(t *testing.T) {
	tests := []struct {
		name   string
		result workspace.BroadcastResult
		want   string
	}{
		{
			name:   "delivered only",
			result: workspace.BroadcastResult{Delivered: []string{"a", "b"}},
			want:   "Broadcast sent to 2 workspaces",
		},
		{
			name: "partial failure",
			result: workspace.BroadcastResult{
				Delivered: []string{"a"},
				Failed:    []workspace.BroadcastFailure{{WorkspaceID: "ws-2"}},
			},
			want: "Broadcast sent to 1 workspace (1 failed)",
		},
		{
			name: "failed only",
			result: workspace.BroadcastResult{
				Failed: []workspace.BroadcastFailure{{WorkspaceID: "ws-2"}, {WorkspaceID: "ws-3"}},
			},
			want: "Broadcast failed for 2 workspaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := broadcastStatusLine(tt.result); got != tt.want {
				t.Fatalf("broadcastStatusLine() = %q, want %q", got, tt.want)
			}
		})
	}
}
