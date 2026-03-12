package flicker

import (
	"context"
	"log/slog"
	"time"
)

// Trigger evaluates conditions and promotes workflows. Triggers run
// alongside the scheduler — they don't execute workflows, they make
// suspended workflows schedulable.
type Trigger interface {
	Start(ctx context.Context, deps TriggerDeps) error
}

// TriggerDeps is the set of dependencies provided to triggers by the engine.
type TriggerDeps struct {
	Store  WorkflowStore
	NowFn  func() time.Time
	Logger *slog.Logger
}

// TimeTrigger promotes suspended workflows whose resume time has arrived.
// It runs a ticker loop at the configured interval.
type TimeTrigger struct {
	interval time.Duration
}

// NewTimeTrigger creates a TimeTrigger with the given poll interval.
// If interval is zero, it defaults to the engine's poll interval.
func NewTimeTrigger(interval time.Duration) *TimeTrigger {
	return &TimeTrigger{interval: interval}
}

// Start runs the trigger loop until ctx is cancelled.
func (t *TimeTrigger) Start(ctx context.Context, deps TriggerDeps) error {
	interval := t.interval
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
			n, err := deps.Store.PromoteSuspended(ctx, deps.NowFn())
			if err != nil {
				deps.Logger.Info("time trigger: promote failed", "error", err)

				continue
			}

			if n > 0 {
				deps.Logger.Info("time trigger: promoted suspended workflows", "count", n)
			}
		}
	}
}
