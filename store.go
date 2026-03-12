package flicker

import (
	"context"
	"time"
)

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
}
