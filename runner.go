package flicker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Runner executes a single workflow record to completion (or failure).
type Runner interface {
	Run(ctx context.Context, record *WorkflowRecord) error
}

// LocalRunner executes workflows in the current process.
type LocalRunner struct {
	registry map[string]registryEntry
	store    WorkflowStore
	logger   *slog.Logger
	nowFunc  func() time.Time
	idFunc   func() string
}

// Run executes a single workflow: transitions to running, calls Execute,
// and resolves the outcome (complete, retry, suspend, fail, cancel).
func (r *LocalRunner) Run(ctx context.Context, record *WorkflowRecord) error {
	entry, ok := r.registry[record.Type]
	if !ok {
		return &DefinitionNotFoundError{Type: record.Type}
	}

	// Transition to running.
	if err := r.store.UpdateStatus(ctx, record.ID, StatusRunning, record.OCCVersion); err != nil {
		return fmt.Errorf("set running: %w", err)
	}

	occAfterRunning := record.OCCVersion + 1

	// Prefetch step results into a cache map.
	steps, err := r.store.ListStepResults(ctx, record.Type, record.Version, record.ID)
	if err != nil {
		return fmt.Errorf("prefetch step results: %w", err)
	}

	stepCache := make(map[string]*StepResult, len(steps))
	for _, s := range steps {
		stepCache[s.StepName] = s
	}

	// Create the workflow context — fully initialized before the factory sees it.
	wc := &WorkflowContext{
		id:        record.ID,
		wfType:    record.Type,
		version:   record.Version,
		store:     r.store,
		logger:    r.logger,
		nowFn:     r.nowFunc,
		idFn:      r.idFunc,
		seenSteps: make(map[string]struct{}),
		stepCache: stepCache,
		mu:        &sync.Mutex{},
	}
	wc.Time = NewTimeProvider(wc, r.nowFunc)
	wc.ID = NewIDProvider(wc, r.idFunc)

	// Check cancellation signal before executing.
	signal, sigErr := r.store.GetSignal(ctx, record.ID)
	if sigErr != nil {
		return fmt.Errorf("get signal: %w", sigErr)
	}

	var execErr error
	if signal == SignalCancelRequested {
		execErr = ErrCancelled
	} else {
		execErr = panicToError(func() error {
			return entry.def.executeWorkflow(ctx, wc, record.Payload)
		})
	}

	return r.resolveOutcome(ctx, record, entry, occAfterRunning, wc, execErr)
}

func (r *LocalRunner) resolveOutcome(
	ctx context.Context,
	record *WorkflowRecord,
	entry registryEntry,
	occVersion int,
	wc *WorkflowContext,
	execErr error,
) error {
	// Check if Stop was called.
	if wc.Stopped() {
		if stopErr := wc.StopError(); stopErr != nil {
			return r.store.SetError(ctx, record.ID, StatusFailed, stopErr.Error(), occVersion)
		}

		return r.store.UpdateStatus(ctx, record.ID, StatusCompleted, occVersion)
	}

	if execErr != nil {
		// Cancellation is terminal — no retry.
		var cancelledErr *CancelledError
		if errors.As(execErr, &cancelledErr) {
			return r.store.UpdateStatus(ctx, record.ID, StatusCancelled, occVersion)
		}

		if se, ok := IsSuspend(execErr); ok {
			return r.store.Suspend(ctx, record.ID, se.ResumeAt, occVersion)
		}

		if record.Attempts+1 >= entry.retryPolicy.MaxAttempts {
			return r.store.SetError(ctx, record.ID, StatusFailed, execErr.Error(), occVersion)
		}

		delay := entry.retryPolicy.BaseDelay << record.Attempts
		if delay > entry.retryPolicy.MaxDelay {
			delay = entry.retryPolicy.MaxDelay
		}

		return r.store.SetRetry(ctx, record.ID, r.nowFunc().Add(delay), occVersion)
	}

	return r.store.UpdateStatus(ctx, record.ID, StatusCompleted, occVersion)
}
