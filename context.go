package flicker

import (
	"log/slog"
	"sync/atomic"
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
// Workflows see Stop(), Log(), Time, and ID — nothing else.
type WorkflowContext struct {
	id      string
	store   WorkflowStore
	logger  *slog.Logger
	stopped atomic.Bool
	stopCfg stopConfig

	// Time provides durable time operations. w.Time.Now(ctx) returns a
	// cached timestamp that survives replay.
	Time *TimeProvider

	// ID provides durable ID generation. w.ID.New(ctx) returns a cached
	// identifier that survives replay.
	ID *IDProvider
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
