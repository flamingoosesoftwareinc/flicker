package flicker

import (
	"context"
	"time"
)

// Workflow is the core interface. R is the request type, Resp is the response type.
// Execute runs the workflow logic. The return value determines the outcome:
//   - return resp, nil → completed successfully with result
//   - return zero, err → transient failure, retry per RetryPolicy
//   - return zero, Permanent(err) → permanent failure, don't retry
//   - return zero, &SuspendError{} → suspend until ResumeAt
//
// Use struct{} as Resp for fire-and-forget workflows with no meaningful result.
type Workflow[R, Resp any] interface {
	Execute(ctx context.Context, request R) (Resp, error)
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
