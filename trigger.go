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
func NewTimeTrigger(interval time.Duration) *TimeTrigger {
	return &TimeTrigger{interval: interval}
}

// Start runs the trigger loop until ctx is cancelled.
func (t *TimeTrigger) Start(ctx context.Context, deps TriggerDeps) error {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			_, _ = promote(ctx, deps)
		}
	}
}

// ManualTimeTrigger is a TimeTrigger for tests. Instead of a ticker loop,
// call Promote() to promote suspended workflows on demand.
type ManualTimeTrigger struct {
	deps TriggerDeps
}

// NewManualTimeTrigger creates a trigger that promotes only when Promote() is called.
func NewManualTimeTrigger() *ManualTimeTrigger {
	return &ManualTimeTrigger{}
}

// Start captures the deps but does not loop — returns immediately.
// The engine calls this in Start(), but for tests with RunOnce you can
// skip it and just call Promote() after setting deps via SetDeps().
func (t *ManualTimeTrigger) Start(_ context.Context, deps TriggerDeps) error {
	t.deps = deps
	return nil
}

// SetDeps sets the trigger deps directly, for use with RunOnce-based tests
// where Start() is never called.
func (t *ManualTimeTrigger) SetDeps(deps TriggerDeps) {
	t.deps = deps
}

// Promote runs one promotion cycle, returning the number of workflows promoted.
func (t *ManualTimeTrigger) Promote(ctx context.Context) (int, error) {
	return promote(ctx, t.deps)
}

func promote(ctx context.Context, deps TriggerDeps) (int, error) {
	n, err := deps.Store.PromoteSuspended(ctx, deps.NowFn())
	if err != nil {
		deps.Logger.Info("time trigger: promote failed", "error", err)
		return 0, err
	}

	if n > 0 {
		deps.Logger.Info("time trigger: promoted suspended workflows", "count", n)
	}

	return n, nil
}
