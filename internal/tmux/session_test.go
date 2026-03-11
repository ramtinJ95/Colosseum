package tmux

import (
	"context"
	"testing"
)

func TestCreateSession(t *testing.T) {
	mock := NewMockCommander(MockResponse{Output: "%0\n", Err: nil})
	client := NewClient(mock)

	paneID, err := client.CreateSession(context.Background(), "colo-myproject", "/home/user/projects/myproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paneID != "%0" {
		t.Errorf("expected pane ID %%0, got %q", paneID)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.Calls))
	}

	args := mock.Calls[0].Args
	expected := []string{"new-session", "-d", "-s", "colo-myproject", "-c", "/home/user/projects/myproject", "-P", "-F", "#{pane_id}"}
	assertArgs(t, args, expected)
}

func TestKillSession(t *testing.T) {
	mock := NewMockCommander(MockResponse{Output: "", Err: nil})
	client := NewClient(mock)

	err := client.KillSession(context.Background(), "colo-myproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.Calls))
	}

	expected := []string{"kill-session", "-t", "colo-myproject"}
	assertArgs(t, mock.Calls[0].Args, expected)
}

func TestCreateDetachedSessionWithCommand(t *testing.T) {
	mock := NewMockCommander(MockResponse{Output: "", Err: nil})
	client := NewClient(mock)

	err := client.CreateDetachedSessionWithCommand(
		context.Background(),
		"colo-dashboard",
		"/home/user/projects/myproject",
		[]string{"env", "COLOSSEUM_DASHBOARD_INTERNAL=1", "/tmp/colosseum"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"new-session",
		"-d",
		"-s", "colo-dashboard",
		"-c", "/home/user/projects/myproject",
		"env",
		"COLOSSEUM_DASHBOARD_INTERNAL=1",
		"/tmp/colosseum",
	}
	assertArgs(t, mock.Calls[0].Args, expected)
}

func TestSessionExists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		mock := NewMockCommander(MockResponse{Output: "", Err: nil})
		client := NewClient(mock)

		if !client.SessionExists(context.Background(), "colo-myproject") {
			t.Fatal("expected session to exist")
		}

		expected := []string{"has-session", "-t", "colo-myproject"}
		assertArgs(t, mock.Calls[0].Args, expected)
	})

	t.Run("not found", func(t *testing.T) {
		mock := NewMockCommander(MockResponse{
			Output: "",
			Err:    &TmuxError{Args: []string{"has-session"}, Stderr: "session not found"},
		})
		client := NewClient(mock)

		if client.SessionExists(context.Background(), "ghost") {
			t.Fatal("expected session to not exist")
		}
	})
}

func TestListSessions(t *testing.T) {
	mock := NewMockCommander(MockResponse{
		Output: "colo-project1\ncolo-project2\nother-session\ncolo-project3",
		Err:    nil,
	})
	client := NewClient(mock)

	sessions, err := client.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"colo-project1", "colo-project2", "colo-project3"}
	if len(sessions) != len(expected) {
		t.Fatalf("expected %d sessions, got %d: %v", len(expected), len(sessions), sessions)
	}
	for i, name := range expected {
		if sessions[i] != name {
			t.Errorf("session[%d]: expected %q, got %q", i, name, sessions[i])
		}
	}
}

func TestSwitchClientInstallsDashboardBinding(t *testing.T) {
	mock := NewMockCommander(
		MockResponse{Output: "dashboard\n", Err: nil},
		MockResponse{Output: "", Err: nil},
		MockResponse{Output: "", Err: nil},
	)
	client := NewClient(mock)

	if err := client.SwitchClient(context.Background(), "colo-myproject"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(mock.Calls))
	}

	assertArgs(t, mock.Calls[0].Args, []string{
		"display-message",
		"-p",
		"#{session_name}",
	})
	assertArgs(t, mock.Calls[1].Args, []string{
		"bind-key",
		"-N", "Colosseum dashboard",
		"-T", "prefix",
		"e",
		"switch-client",
		"-t", "dashboard",
	})
	assertArgs(t, mock.Calls[2].Args, []string{
		"switch-client",
		"-t", "colo-myproject",
	})
}

func TestAttachSession(t *testing.T) {
	mock := NewMockCommander(MockResponse{Output: "", Err: nil})
	client := NewClient(mock)

	if err := client.AttachSession(context.Background(), "colo-myproject"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertArgs(t, mock.Calls[0].Args, []string{
		"attach-session",
		"-t", "colo-myproject",
	})
	if !mock.Calls[0].Interactive {
		t.Fatal("attach-session should run interactively")
	}
}

func TestCurrentSession(t *testing.T) {
	mock := NewMockCommander(MockResponse{Output: "colo-dashboard\n", Err: nil})
	client := NewClient(mock)

	session, err := client.CurrentSession(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "colo-dashboard" {
		t.Fatalf("session = %q, want %q", session, "colo-dashboard")
	}

	assertArgs(t, mock.Calls[0].Args, []string{
		"display-message",
		"-p",
		"#{session_name}",
	})
}

func TestLastSession(t *testing.T) {
	mock := NewMockCommander(MockResponse{Output: "colo-feature\n", Err: nil})
	client := NewClient(mock)

	session, err := client.LastSession(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session != "colo-feature" {
		t.Fatalf("session = %q, want %q", session, "colo-feature")
	}

	assertArgs(t, mock.Calls[0].Args, []string{
		"display-message",
		"-p",
		"#{client_last_session}",
	})
}

func assertArgs(t *testing.T, got, expected []string) {
	t.Helper()
	if len(got) != len(expected) {
		t.Fatalf("arg count: expected %d, got %d\nexpected: %v\ngot:      %v", len(expected), len(got), expected, got)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("arg[%d]: expected %q, got %q", i, expected[i], got[i])
		}
	}
}
