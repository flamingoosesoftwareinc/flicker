package flicker

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// promote runs one time-based promotion cycle, moving suspended workflows
// whose resume time has passed back to pending.
func promote(
	ctx context.Context,
	store WorkflowStore,
	nowFn func() time.Time,
	logger *slog.Logger,
) (int, error) {
	n, err := store.PromoteSuspended(ctx, nowFn())
	if err != nil {
		logger.Info("time promotion: failed", "error", err)
		return 0, err
	}

	if n > 0 {
		logger.Info("time promotion: promoted suspended workflows", "count", n)
	}

	return n, nil
}

// promotionLoop runs a ticker that promotes suspended workflows at the
// configured interval. Built into the engine — not a pluggable interface.
func promotionLoop(
	ctx context.Context,
	interval time.Duration,
	store WorkflowStore,
	nowFn func() time.Time,
	logger *slog.Logger,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := promote(ctx, store, nowFn, logger); err != nil {
				logger.Info("promotion loop: cycle failed", "error", err)
			}
		}
	}
}

// subscriptionTimeoutLoop runs a ticker that times out expired event
// subscriptions at the configured interval.
func subscriptionTimeoutLoop(
	ctx context.Context,
	interval time.Duration,
	store WorkflowStore,
	nowFn func() time.Time,
	logger *slog.Logger,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := timeOutSubscriptions(ctx, store, nowFn, logger); err != nil {
				logger.Info("subscription timeout loop: cycle failed", "error", err)
			}
		}
	}
}

// timeOutSubscriptions runs one cycle of timing out expired event
// subscriptions. Expired subscriptions get a timeout marker saved as
// their step result and the workflow is promoted back to pending.
func timeOutSubscriptions(
	ctx context.Context,
	store WorkflowStore,
	nowFn func() time.Time,
	logger *slog.Logger,
) (int, error) {
	n, err := store.TimeOutSubscriptions(ctx, nowFn())
	if err != nil {
		logger.Info("subscription timeout: failed", "error", err)
		return 0, err
	}

	if n > 0 {
		logger.Info("subscription timeout: timed out subscriptions", "count", n)
	}

	return n, nil
}

// ErrEventTimeout is returned by WaitForEvent when the event did not arrive
// before the deadline. Workflows should handle this as a permanent decision
// point — the event is not coming.
var ErrEventTimeout = fmt.Errorf("event wait timed out")
