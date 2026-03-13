package flicker

import (
	"context"
	"errors"
	"time"
)

// ErrStepNotFound is returned by GetStepResult when no cached result exists
// for the given step. Callers must distinguish this from real storage errors.
var ErrStepNotFound = errors.New("step result not found")

// WorkflowRecord is the persisted state of a workflow instance.
type WorkflowRecord struct {
	ID         string
	Type       string
	Version    string
	Status     Status
	Signal     Signal
	Payload    []byte
	Error      string
	RetryAfter time.Time
	Attempts   int
	OCCVersion int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// StepResult is the cached result of a durable step.
type StepResult struct {
	Type       string
	Version    string
	WorkflowID string
	StepName   string
	Result     []byte
	Error      string
	CreatedAt  time.Time
}

// Subscription records a workflow waiting for an external event.
// The correlation key is the lookup key for SendEvent.
type Subscription struct {
	WorkflowID     string
	Type           string
	Version        string
	StepName       string
	CorrelationKey string
	Deadline       time.Time
	CreatedAt      time.Time
}

// WorkflowStore is the interface for workflow persistence.
type WorkflowStore interface {
	Create(ctx context.Context, record *WorkflowRecord) error
	Get(ctx context.Context, id string) (*WorkflowRecord, error)
	UpdateStatus(ctx context.Context, id string, status Status, occVersion int) error
	SetError(ctx context.Context, id string, status Status, errMsg string, occVersion int) error
	SetRetry(ctx context.Context, id string, retryAfter time.Time, occVersion int) error
	Suspend(ctx context.Context, id string, resumeAt time.Time, occVersion int) error
	PromoteSuspended(ctx context.Context, now time.Time) (int, error)
	ListSchedulable(ctx context.Context, limit int) ([]*WorkflowRecord, error)
	SaveStepResult(ctx context.Context, result *StepResult) error
	GetStepResult(
		ctx context.Context,
		wfType, version, workflowID, stepName string,
	) (*StepResult, error)
	ListStepResults(ctx context.Context, wfType, version, workflowID string) ([]*StepResult, error)

	// Event subscriptions.
	SaveSubscription(ctx context.Context, sub *Subscription) error
	// ResumeSubscription delivers an event payload to the workflow waiting on
	// the given correlation key. It saves the payload as the step result,
	// deletes the subscription, and promotes the workflow to pending.
	ResumeSubscription(ctx context.Context, correlationKey string, payload []byte) error
	// TimeOutSubscriptions finds subscriptions past their deadline, saves a
	// timeout marker as the step result, deletes the subscription, and
	// promotes the workflow to pending. Returns the number timed out.
	TimeOutSubscriptions(ctx context.Context, now time.Time) (int, error)
}
