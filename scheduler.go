package flicker

import (
	"context"
	"time"
)

// Scheduler polls (or listens) for schedulable work and dispatches it.
type Scheduler interface {
	Start(ctx context.Context, dispatch func([]*WorkflowRecord)) error
}

// PollingScheduler polls the store at a fixed interval.
type PollingScheduler struct {
	store    WorkflowStore
	limit    int
	interval time.Duration
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
			records, err := s.store.ListSchedulable(ctx, s.limit)
			if err != nil {
				continue
			}

			if len(records) > 0 {
				dispatch(records)
			}
		}
	}
}
