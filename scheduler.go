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
	store    WorkflowStore
	limit    int
	interval time.Duration
	nudge    <-chan struct{}
}

// Start runs the polling loop until ctx is cancelled.
func (s *PollingScheduler) Start(ctx context.Context, dispatch func([]*WorkflowRecord)) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.poll(ctx, dispatch)
		case <-s.nudge:
			s.poll(ctx, dispatch)
		}
	}
}

func (s *PollingScheduler) poll(ctx context.Context, dispatch func([]*WorkflowRecord)) {
	records, err := s.store.ListSchedulable(ctx, s.limit)
	if err != nil {
		return
	}

	if len(records) > 0 {
		dispatch(records)
	}
}
