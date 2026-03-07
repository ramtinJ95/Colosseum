package status

import (
	"context"
	"sync"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
)

const (
	defaultSpikeWindow      = 1 * time.Second
	defaultHysteresisWindow = 500 * time.Millisecond
)

type WorkspaceProvider interface {
	List() ([]workspace.Workspace, error)
}

type workspaceState struct {
	confirmed    agent.Status
	confirmedAt  time.Time
	pending      agent.Status
	pendingFirst time.Time
}

type PollerOption func(*Poller)

func WithSpikeWindow(d time.Duration) PollerOption {
	return func(p *Poller) { p.spikeWindow = d }
}

func WithHysteresisWindow(d time.Duration) PollerOption {
	return func(p *Poller) { p.hysteresisWindow = d }
}

type Poller struct {
	detector         *Detector
	provider         WorkspaceProvider
	interval         time.Duration
	spikeWindow      time.Duration
	hysteresisWindow time.Duration
	updates          chan Update
	states           map[string]*workspaceState
	mu               sync.RWMutex
}

func NewPoller(detector *Detector, provider WorkspaceProvider, interval time.Duration, opts ...PollerOption) *Poller {
	if interval <= 0 {
		interval = 1500 * time.Millisecond
	}
	p := &Poller{
		detector:         detector,
		provider:         provider,
		interval:         interval,
		spikeWindow:      defaultSpikeWindow,
		hysteresisWindow: defaultHysteresisWindow,
		updates:          make(chan Update, 64),
		states:           make(map[string]*workspaceState),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Poller) Updates() <-chan Update {
	return p.updates
}

func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	workspaces, err := p.provider.List()
	if err != nil {
		now := time.Now()
		p.mu.Lock()
		for id, state := range p.states {
			if state.confirmed != agent.StatusStopped {
				prev := state.confirmed
				state.confirmed = agent.StatusStopped
				state.confirmedAt = now
				select {
				case p.updates <- Update{
					WorkspaceID: id,
					Previous:    prev,
					Current:     agent.StatusStopped,
				}:
				default:
				}
			}
		}
		p.mu.Unlock()
		return
	}

	activeIDs := make(map[string]struct{}, len(workspaces))
	for _, ws := range workspaces {
		activeIDs[ws.ID] = struct{}{}
	}
	p.mu.Lock()
	for id := range p.states {
		if _, ok := activeIDs[id]; !ok {
			delete(p.states, id)
		}
	}
	p.mu.Unlock()

	now := time.Now()
	for _, ws := range workspaces {
		agentPane, ok := ws.PaneTargets["agent"]
		if !ok {
			continue
		}

		detected, content, err := p.detector.Detect(ctx, agentPane, ws.AgentType)
		if err != nil {
			detected = agent.StatusStopped
		}

		p.mu.Lock()
		state, exists := p.states[ws.ID]
		if !exists {
			state = &workspaceState{}
			p.states[ws.ID] = state
		}

		if p.shouldTransition(state, detected, now) {
			previous := state.confirmed
			state.confirmed = detected
			state.confirmedAt = now
			state.pending = 0
			state.pendingFirst = time.Time{}
			p.mu.Unlock()

			select {
			case p.updates <- Update{
				WorkspaceID: ws.ID,
				Previous:    previous,
				Current:     detected,
				PaneContent: content,
			}:
			default:
			}
		} else {
			p.mu.Unlock()
		}
	}
}

// shouldTransition applies spike detection and hysteresis filtering to
// prevent status flicker from transient terminal content (spinner
// animations, dynamic counters). Urgent statuses (Waiting, Error,
// Stopped) and initial detection bypass filtering entirely.
func (p *Poller) shouldTransition(state *workspaceState, detected agent.Status, now time.Time) bool {
	if detected == state.confirmed {
		state.pending = 0
		state.pendingFirst = time.Time{}
		return false
	}

	// Immediate transitions: urgent states and first detection.
	if isUrgentStatus(detected) || state.confirmed == agent.StatusUnknown {
		return true
	}

	// Track the candidate state.
	if detected != state.pending {
		state.pending = detected
		state.pendingFirst = now
	}

	// Spike window: new state must be sustained.
	if p.spikeWindow > 0 && now.Sub(state.pendingFirst) < p.spikeWindow {
		return false
	}

	// Hysteresis: current state must have been held long enough.
	if p.hysteresisWindow > 0 && now.Sub(state.confirmedAt) < p.hysteresisWindow {
		return false
	}

	return true
}

func isUrgentStatus(s agent.Status) bool {
	return s == agent.StatusWaiting || s == agent.StatusError || s == agent.StatusStopped
}

func (p *Poller) CurrentStatus(workspaceID string) agent.Status {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if state, ok := p.states[workspaceID]; ok {
		return state.confirmed
	}
	return agent.StatusUnknown
}
