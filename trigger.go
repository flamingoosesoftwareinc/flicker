package flicker

import (
	"context"
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
		logger.Error("time promotion failed", "error", err)
		return 0, err
	}

	if n > 0 {
		logger.Info("time promotion: promoted suspended workflows", "count", n)
	}

	return n, nil
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
		logger.Error("subscription timeout failed", "error", err)
		return 0, err
	}

	if n > 0 {
		logger.Info("subscription timeout: timed out subscriptions", "count", n)
	}

	return n, nil
}
