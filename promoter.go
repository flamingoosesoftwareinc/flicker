package flicker

import (
	"context"
	"log/slog"
	"time"
)

// Promoter detects when suspended workflows should become schedulable,
// mutates the store to promote them, and signals the engine via ready().
// Multiple promoters run concurrently — each handles its own trigger source
// (time-based deadlines, message queues, webhooks, etc.).
//
// Start blocks until ctx is cancelled.
type Promoter interface {
	Start(ctx context.Context, store WorkflowStore, ready func()) error
}

// PollingTimePromoter is the default time-based promoter. It polls the store
// on a fixed interval to promote suspended workflows whose SleepUntil deadline
// has passed and to time out expired event subscriptions.
type PollingTimePromoter struct {
	// Interval between promotion cycles.
	Interval time.Duration

	// NowFunc returns the current time. Defaults to time.Now().UTC().
	NowFunc func() time.Time

	// Logger for promotion activity. Defaults to slog.Default().
	Logger *slog.Logger
}

func (p *PollingTimePromoter) Start(ctx context.Context, store WorkflowStore, ready func()) error {
	nowFn := p.NowFunc
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}

	logger := p.Logger
	if logger == nil {
		logger = slog.Default()
	}

	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			promoted, _ := promote(ctx, store, nowFn, logger)
			timedOut, _ := timeOutSubscriptions(ctx, store, nowFn, logger)

			if promoted > 0 || timedOut > 0 {
				ready()
			}
		}
	}
}
