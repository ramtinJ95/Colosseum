package tmux

import (
	"context"
	"strings"
	"testing"
)

func TestSplitWindow(t *testing.T) {
	t.Run("horizontal", func(t *testing.T) {
		mock := NewMockCommander(MockResponse{Output: "%5", Err: nil})
		client := NewClient(mock)

		paneID, err := client.SplitWindow(context.Background(), "colo-myproject", true, "/tmp")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if paneID != "%5" {
			t.Errorf("expected pane ID %%5, got %q", paneID)
		}

		args := mock.Calls[0].Args
		expectedFormat := BuildFormat(FormatPaneID)
		expected := []string{"split-window", "-h", "-t", "colo-myproject", "-c", "/tmp", "-P", "-F", expectedFormat}
		assertArgs(t, args, expected)
	})

	t.Run("vertical", func(t *testing.T) {
		mock := NewMockCommander(MockResponse{Output: "%8", Err: nil})
		client := NewClient(mock)

		paneID, err := client.SplitWindow(context.Background(), "colo-myproject", false, "/tmp")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if paneID != "%8" {
			t.Errorf("expected pane ID %%8, got %q", paneID)
		}

		args := mock.Calls[0].Args
		if args[1] != "-v" {
			t.Errorf("expected -v for vertical split, got %q", args[1])
		}
	})
}

func TestCapturePane(t *testing.T) {
	capturedOutput := "$ echo hello\nhello\n$ _"
	mock := NewMockCommander(MockResponse{Output: capturedOutput, Err: nil})
	client := NewClient(mock)

	output, err := client.CapturePane(context.Background(), "%3", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != capturedOutput {
		t.Errorf("expected output %q, got %q", capturedOutput, output)
	}

	expected := []string{"capture-pane", "-p", "-t", "%3", "-S", "-50"}
	assertArgs(t, mock.Calls[0].Args, expected)
}

func TestCapturePaneTitle(t *testing.T) {
	mock := NewMockCommander(MockResponse{Output: "⠹ claude", Err: nil})
	client := NewClient(mock)

	title, err := client.CapturePaneTitle(context.Background(), "%3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "⠹ claude" {
		t.Errorf("expected title %q, got %q", "⠹ claude", title)
	}

	expected := []string{"display-message", "-t", "%3", "-p", "#{pane_title}"}
	assertArgs(t, mock.Calls[0].Args, expected)
}

func TestSendKeys(t *testing.T) {
	mock := NewMockCommander(
		MockResponse{Output: "", Err: nil},
		MockResponse{Output: "", Err: nil},
	)
	client := NewClient(mock)

	err := client.SendKeys(context.Background(), "%3", "ls -la", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 2 {
		t.Fatalf("expected 2 tmux calls, got %d", len(mock.Calls))
	}
	assertArgs(t, mock.Calls[0].Args, []string{"send-keys", "-t", "%3", "-l", "ls -la"})
	assertArgs(t, mock.Calls[1].Args, []string{"send-keys", "-t", "%3", "Enter"})
}

func TestSendKeysUsesPasteBufferForMultilineInput(t *testing.T) {
	mock := NewMockCommander(
		MockResponse{Output: "", Err: nil},
		MockResponse{Output: "", Err: nil},
		MockResponse{Output: "", Err: nil},
	)
	client := NewClient(mock)

	err := client.SendKeys(context.Background(), "%3", "line1\nline2", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 3 {
		t.Fatalf("expected 3 tmux calls, got %d", len(mock.Calls))
	}

	setBufferArgs := mock.Calls[0].Args
	if len(setBufferArgs) != 5 {
		t.Fatalf("unexpected set-buffer args: %v", setBufferArgs)
	}
	if setBufferArgs[0] != "set-buffer" || setBufferArgs[1] != "-b" {
		t.Fatalf("unexpected set-buffer prefix: %v", setBufferArgs[:2])
	}
	if !strings.HasPrefix(setBufferArgs[2], "colosseum-") {
		t.Fatalf("buffer name = %q, want colosseum-*", setBufferArgs[2])
	}
	assertArgs(t, setBufferArgs[3:], []string{"--", "line1\nline2"})

	assertArgs(t, mock.Calls[1].Args, []string{"paste-buffer", "-d", "-p", "-r", "-b", setBufferArgs[2], "-t", "%3"})
	assertArgs(t, mock.Calls[2].Args, []string{"send-keys", "-t", "%3", "Enter"})
}

func TestSendLiteralKeys(t *testing.T) {
	mock := NewMockCommander(MockResponse{Output: "", Err: nil})
	client := NewClient(mock)

	err := client.SendLiteralKeys(context.Background(), "%3", "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"send-keys", "-t", "%3", "-l", "hello world"}
	assertArgs(t, mock.Calls[0].Args, expected)

	// Verify no "Enter" is appended
	for _, arg := range mock.Calls[0].Args {
		if arg == "Enter" {
			t.Error("SendLiteralKeys should not append Enter")
		}
	}
}

func TestListPanes(t *testing.T) {
	mock := NewMockCommander(MockResponse{
		Output: "%1\t120\t40\n%2\t60\t40",
		Err:    nil,
	})
	client := NewClient(mock)

	panes, err := client.ListPanes(context.Background(), "colo-myproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(panes) != 2 {
		t.Fatalf("expected 2 panes, got %d", len(panes))
	}

	assertArgs(t, mock.Calls[0].Args, []string{"list-panes", "-t", "colo-myproject", "-F", BuildFormat(FormatPaneID, FormatPaneWidth, FormatPaneHeight)})

	if panes[0].ID != "%1" || panes[0].Width != 120 || panes[0].Height != 40 {
		t.Errorf("pane[0] mismatch: %+v", panes[0])
	}
	if panes[1].ID != "%2" || panes[1].Width != 60 || panes[1].Height != 40 {
		t.Errorf("pane[1] mismatch: %+v", panes[1])
	}
}

func TestResizePane(t *testing.T) {
	mock := NewMockCommander(MockResponse{Output: "", Err: nil})
	client := NewClient(mock)

	err := client.ResizePane(context.Background(), "%3", 200, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"resize-pane", "-t", "%3", "-x", "200", "-y", "50"}
	assertArgs(t, mock.Calls[0].Args, expected)
}
