package flicker

import (
	"context"
	"time"
)

// Workflow is the core interface. R is the request/input type.
// Execute runs the workflow logic. The return value determines the outcome:
//   - return nil → completed successfully
//   - return error → transient failure, retry per RetryPolicy
//   - Stop(WithError(err)) then return nil → permanent failure, don't retry
type Workflow[R any] interface {
	Execute(ctx context.Context, request R) error
}

// Status represents where the workflow currently is.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
	StatusSuspended Status = "suspended"
)

// Signal represents what you want the workflow to do (separate from status).
type Signal string

const (
	SignalNone            Signal = ""
	SignalCancelRequested Signal = "cancel_requested"
)

// RetryPolicy controls retry behavior for workflows.
type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// Val wraps a value in a pointer. Use in step functions that return
// primitive types: return flicker.Val("hello"), nil
func Val[T any](v T) *T {
	return &v
}

// DefaultRetryPolicy returns sensible safe defaults.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
	}
}
