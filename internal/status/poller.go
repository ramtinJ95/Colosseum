package status

import (
	"context"
	"sync"
	"time"

	"github.com/ramtinj/colosseum/internal/agent"
	"github.com/ramtinj/colosseum/internal/workspace"
)

type WorkspaceProvider interface {
	List() ([]workspace.Workspace, error)
}

type Poller struct {
	detector *Detector
	provider WorkspaceProvider
	interval time.Duration
	updates  chan Update
	statuses map[string]agent.Status
	mu       sync.RWMutex
}

func NewPoller(detector *Detector, provider WorkspaceProvider, interval time.Duration) *Poller {
	if interval <= 0 {
		interval = 1500 * time.Millisecond
	}
	return &Poller{
		detector: detector,
		provider: provider,
		interval: interval,
		updates:  make(chan Update, 64),
		statuses: make(map[string]agent.Status),
	}
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
		p.mu.Lock()
		for id, prev := range p.statuses {
			if prev != agent.StatusStopped {
				p.statuses[id] = agent.StatusStopped
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
	for id := range p.statuses {
		if _, ok := activeIDs[id]; !ok {
			delete(p.statuses, id)
		}
	}
	p.mu.Unlock()

	for _, ws := range workspaces {
		agentPane, ok := ws.PaneTargets["agent"]
		if !ok {
			continue
		}

		current, content, err := p.detector.Detect(ctx, agentPane, ws.AgentType)
		if err != nil {
			current = agent.StatusStopped
		}

		p.mu.RLock()
		previous := p.statuses[ws.ID]
		p.mu.RUnlock()

		if current != previous {
			p.mu.Lock()
			p.statuses[ws.ID] = current
			p.mu.Unlock()

			select {
			case p.updates <- Update{
				WorkspaceID: ws.ID,
				Previous:    previous,
				Current:     current,
				PaneContent: content,
			}:
			default:
			}
		}
	}
}

func (p *Poller) CurrentStatus(workspaceID string) agent.Status {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.statuses[workspaceID]
}
