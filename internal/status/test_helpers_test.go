package status

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
)

const (
	testPollerInterval = 50 * time.Millisecond
	testDetectLines    = 50
	testWorkingContent = "⠹ Working (esc to interrupt)"
	testIdleContent    = ">\n"
	testWaitingContent = "Do you want to allow this?\n\n>"
)

type mockCapturer struct {
	mu      sync.Mutex
	content string
	title   string
	err     error
}

func (m *mockCapturer) SetContent(content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.content = content
}

func (m *mockCapturer) SetTitle(title string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.title = title
}

func (m *mockCapturer) CapturePane(_ context.Context, _ string, _ int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.content, m.err
}

func (m *mockCapturer) CapturePaneTitle(_ context.Context, _ string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.title, nil
}

type mockProvider struct {
	workspaces []workspace.Workspace
	reports    []workspace.AgentStatusReport
}

func (m *mockProvider) List() ([]workspace.Workspace, error) {
	return m.workspaces, nil
}

func (m *mockProvider) LoadState() (workspace.State, error) {
	return workspace.State{Workspaces: m.workspaces, AgentStatusReports: m.reports}, nil
}

type pollerHarness struct {
	t        *testing.T
	capturer *mockCapturer
	provider *mockProvider
	poller   *Poller
	ctx      context.Context
	cancel   context.CancelFunc
}

func newPollerHarness(t *testing.T, agentType agent.AgentType, initialContent string, opts ...PollerOption) *pollerHarness {
	t.Helper()
	capturer := &mockCapturer{content: initialContent}
	detector := NewDetector(capturer, testDetectLines)
	provider := &mockProvider{workspaces: []workspace.Workspace{statusTestWorkspace("ws-1", agentType)}}
	poller := NewPoller(detector, provider, testPollerInterval, append([]PollerOption{WithSpikeWindow(0), WithHysteresisWindow(0)}, opts...)...)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	h := &pollerHarness{
		t:        t,
		capturer: capturer,
		provider: provider,
		poller:   poller,
		ctx:      ctx,
		cancel:   cancel,
	}
	t.Cleanup(cancel)
	return h
}

func (h *pollerHarness) start() {
	h.t.Helper()
	go h.poller.Run(h.ctx)
}

func (h *pollerHarness) setContent(content string) {
	h.capturer.SetContent(content)
}

func (h *pollerHarness) setTitle(title string) {
	h.capturer.SetTitle(title)
}

func (h *pollerHarness) nextUpdate() Update {
	h.t.Helper()
	select {
	case update := <-h.poller.Updates():
		return update
	case <-time.After(750 * time.Millisecond):
		h.t.Fatal("timed out waiting for status update")
		return Update{}
	}
}

func (h *pollerHarness) expectNoUpdate(window time.Duration) {
	h.t.Helper()
	select {
	case update := <-h.poller.Updates():
		h.t.Fatalf("unexpected update: %+v", update)
	case <-time.After(window):
	}
}

func statusTestWorkspace(id string, agentType agent.AgentType) workspace.Workspace {
	return workspace.Workspace{
		ID:        id,
		AgentType: agentType,
		PaneTargets: map[string]string{
			"agent": "%0",
		},
	}
}
