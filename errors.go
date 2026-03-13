package flicker

import "fmt"

// CancelledError is returned when a workflow is cancelled via SignalCancelRequested.
// It is a terminal error — the workflow will not be retried.
type CancelledError struct{}

func (e *CancelledError) Error() string { return "workflow cancelled" }

// ErrCancelled is the singleton CancelledError for convenience.
var ErrCancelled error = &CancelledError{}

// EventTimeoutError is returned by WaitForEvent when the event did not arrive
// before the deadline. Workflows should handle this as a permanent decision
// point — the event is not coming.
type EventTimeoutError struct{}

func (e *EventTimeoutError) Error() string { return "event wait timed out" }

// ErrEventTimeout is the singleton EventTimeoutError for convenience.
var ErrEventTimeout error = &EventTimeoutError{}

// StepNotFoundError is returned by GetStepResult when no cached result exists
// for the given step. Callers must distinguish this from real storage errors.
type StepNotFoundError struct{}

func (e *StepNotFoundError) Error() string { return "step result not found" }

// ErrStepNotFound is the singleton StepNotFoundError for convenience.
var ErrStepNotFound error = &StepNotFoundError{}

// DefinitionNotFoundError is returned when a workflow type has no registered
// definition in the engine's registry.
type DefinitionNotFoundError struct {
	Type string
}

func (e *DefinitionNotFoundError) Error() string {
	return fmt.Sprintf("no definition registered for %q", e.Type)
}

// WorkflowExecutionError wraps an error from a single workflow execution
// within a dispatch or RunOnce cycle. It carries the workflow ID for
// identification.
type WorkflowExecutionError struct {
	WorkflowID string
	Err        error
}

func (e *WorkflowExecutionError) Error() string {
	return fmt.Sprintf("workflow %s: %s", e.WorkflowID, e.Err)
}

func (e *WorkflowExecutionError) Unwrap() error { return e.Err }
