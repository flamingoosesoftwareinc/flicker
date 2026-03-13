package flicker

import (
	"context"
	"time"
)

// Scheduler polls (or listens) for schedulable work and dispatches it.
type Scheduler interface {
	Start(ctx context.Context, dispatch func([]*WorkflowRecord)) error
}

// PollingScheduler polls the store at a fixed interval. It also accepts
// a nudge channel — when a promoter signals that work is ready, the
// scheduler polls immediately instead of waiting for the next tick.
type PollingScheduler struct {
	// Store to query for schedulable workflows.
	Store WorkflowStore

	// Limit is the maximum number of workflows to dequeue per poll cycle.
	Limit int

	// Interval between poll cycles. Defaults to 1 second if zero.
	Interval time.Duration

	// Nudge triggers an immediate poll when a promoter signals readiness.
	// May be nil (no nudge support).
	Nudge <-chan struct{}
}

// Start runs the polling loop until ctx is cancelled.
func (s *PollingScheduler) Start(ctx context.Context, dispatch func([]*WorkflowRecord)) error {
	interval := s.Interval
	if interval == 0 {
		interval = time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.poll(ctx, dispatch)
		case <-s.Nudge:
			s.poll(ctx, dispatch)
		}
	}
}

func (s *PollingScheduler) poll(ctx context.Context, dispatch func([]*WorkflowRecord)) {
	records, err := s.Store.ListSchedulable(ctx, s.Limit)
	if err != nil {
		return
	}

	if len(records) > 0 {
		dispatch(records)
	}
}
