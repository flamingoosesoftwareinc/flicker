package flicker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"
)

// StopOption is a functional option for Stop().
type StopOption func(*stopConfig)

type stopConfig struct {
	err error
}

// WithError marks the stop as a permanent failure with the given error.
func WithError(err error) StopOption {
	return func(c *stopConfig) {
		c.err = err
	}
}

// WorkflowContext is the framework handle embedded by workflow structs.
// Workflows see Stop(), Log(), Time, ID, and SleepUntil — nothing else.
type WorkflowContext struct {
	id        string
	wfType    string
	version   string
	store     WorkflowStore
	logger    *slog.Logger
	stopped   atomic.Bool
	stopCfg   stopConfig
	seenSteps map[string]struct{}
	stepCache map[string]*StepResult
	nowFn     func() time.Time
	sleep     *Provider[time.Time]

	// Time provides durable time operations. w.Time.Now(ctx) returns a
	// cached timestamp that survives replay.
	Time *TimeProvider

	// ID provides durable ID generation. w.ID.New(ctx) returns a cached
	// identifier that survives replay.
	ID *IDProvider
}

// WorkflowID returns the workflow instance ID.
func (wc *WorkflowContext) WorkflowID() string {
	return wc.id
}

// Stop signals that the workflow should stop. Call return after Stop().
// With no options: clean completion. With WithError: permanent failure.
func (wc *WorkflowContext) Stop(opts ...StopOption) {
	wc.stopped.Store(true)

	for _, opt := range opts {
		opt(&wc.stopCfg)
	}
}

// Stopped returns true if Stop was called.
func (wc *WorkflowContext) Stopped() bool {
	return wc.stopped.Load()
}

// StopError returns the error passed to Stop via WithError, or nil.
func (wc *WorkflowContext) StopError() error {
	return wc.stopCfg.err
}

// Log writes a structured log message using slog key-value style.
func (wc *WorkflowContext) Log(msg string, args ...any) {
	if wc.logger != nil {
		wc.logger.Info(msg, args...)
	}
}

// SleepUntil suspends the workflow until the given time. The resume time is
// durably cached so it survives replay. On re-execution after promotion, if
// the wall clock has passed the cached time, execution continues normally.
func (wc *WorkflowContext) SleepUntil(ctx context.Context, resumeAt time.Time) error {
	if wc.sleep == nil {
		wc.sleep = NewProvider(
			wc,
			"_sleep.until",
			func() (time.Time, error) { return resumeAt, nil },
		)
	}

	// Update the generator to capture the current resumeAt value.
	wc.sleep.gen = func() (time.Time, error) { return resumeAt, nil }

	cached, err := wc.sleep.Get(ctx)
	if err != nil {
		return err
	}

	if wc.nowFn().Before(cached) {
		return &SuspendError{ResumeAt: cached}
	}

	return nil
}

// WaitForEvent suspends the workflow until an external event with the given
// correlation key arrives via Engine.SendEvent, or until the timeout elapses.
//
// On first execution: saves a subscription and suspends.
// On replay after event delivery: returns the deserialized event payload.
// On replay after timeout: returns ErrEventTimeout.
//
// T is the expected event payload type (must be JSON-deserializable).
func WaitForEvent[T any](
	ctx context.Context,
	wc *WorkflowContext,
	stepName string,
	correlationKey string,
	timeout time.Duration,
) (*T, error) {
	// Check if a result already exists (event delivered or timeout marker).
	result, err := Run(ctx, wc, stepName, func(ctx context.Context) (*T, error) {
		// No cached result — this is the first execution. Save a subscription
		// and suspend the workflow. The step function itself never completes
		// successfully on first run; instead we save the subscription and
		// return an error that causes suspension.
		deadline := wc.nowFn().Add(timeout)

		if subErr := wc.store.SaveSubscription(ctx, &Subscription{
			WorkflowID:     wc.id,
			Type:           wc.wfType,
			Version:        wc.version,
			StepName:       stepName,
			CorrelationKey: correlationKey,
			Deadline:       deadline,
			CreatedAt:      wc.nowFn(),
		}); subErr != nil {
			return nil, fmt.Errorf("save subscription: %w", subErr)
		}

		return nil, &SuspendError{ResumeAt: deadline}
	})
	// If the step function returned a SuspendError, propagate it so the
	// engine suspends the workflow.
	if err != nil {
		if _, ok := IsSuspend(err); ok {
			return nil, err
		}

		// Check if the cached result is a timeout marker.
		if errors.Is(err, ErrEventTimeout) {
			return nil, ErrEventTimeout
		}

		return nil, err
	}

	return result, nil
}
