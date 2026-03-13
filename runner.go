package flicker

import (
	"context"
	"encoding/json"
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
	tel      *telemetry
}

// Run executes a single workflow: transitions to running, calls Execute,
// and resolves the outcome (complete, retry, suspend, fail, cancel).
func (r *LocalRunner) Run(ctx context.Context, record *WorkflowRecord) error {
	entry, ok := r.registry[record.Type]
	if !ok {
		return &DefinitionNotFoundError{Type: record.Type}
	}

	// Start workflow span and active metric.
	ctx, span := r.tel.startWorkflowSpan(ctx, record)
	start := time.Now()
	r.tel.adjustActive(ctx, 1, record.Type, record.Version)

	// Transition to running.
	if err := r.store.UpdateStatus(
		ctx,
		record.ID,
		StatusRunning,
		nil,
		record.OCCVersion,
	); err != nil {
		r.tel.adjustActive(ctx, -1, record.Type, record.Version)
		r.tel.endSpanWithError(span, err)
		return fmt.Errorf("set running: %w", err)
	}

	occAfterRunning := record.OCCVersion + 1

	// Prefetch step results into a cache map.
	steps, err := r.store.ListStepResults(ctx, record.Type, record.Version, record.ID)
	if err != nil {
		r.tel.adjustActive(ctx, -1, record.Type, record.Version)
		r.tel.endSpanWithError(span, err)
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
		tel:       r.tel,
	}
	wc.Time = NewTimeProvider(wc, r.nowFunc)
	wc.ID = NewIDProvider(wc, r.idFunc)

	// Check cancellation signal before executing.
	signal, sigErr := r.store.GetSignal(ctx, record.ID)
	if sigErr != nil {
		r.tel.adjustActive(ctx, -1, record.Type, record.Version)
		r.tel.endSpanWithError(span, sigErr)
		return fmt.Errorf("get signal: %w", sigErr)
	}

	var resultJSON json.RawMessage
	var execErr error
	if signal == SignalCancelRequested {
		execErr = ErrCancelled
	} else {
		execErr = panicToError(func() error {
			var err error
			resultJSON, err = entry.def.executeWorkflow(ctx, wc, record.Payload)
			return err
		})
	}

	// End span — suspends are not errors from a tracing perspective.
	r.tel.adjustActive(ctx, -1, record.Type, record.Version)
	r.tel.recordDuration(ctx, record.Type, record.Version, time.Since(start))
	if execErr != nil {
		if _, ok := IsSuspend(execErr); ok {
			span.End()
		} else {
			r.tel.endSpanWithError(span, execErr)
		}
	} else {
		span.End()
	}

	return r.resolveOutcome(ctx, record, entry, occAfterRunning, resultJSON, execErr)
}

func (r *LocalRunner) resolveOutcome(
	ctx context.Context,
	record *WorkflowRecord,
	entry registryEntry,
	occVersion int,
	resultJSON json.RawMessage,
	execErr error,
) error {
	if execErr != nil {
		// Cancellation is terminal — no retry.
		var cancelledErr *CancelledError
		if errors.As(execErr, &cancelledErr) {
			r.tel.recordCompleted(ctx, record.Type, record.Version, StatusCancelled)
			return r.store.UpdateStatus(ctx, record.ID, StatusCancelled, nil, occVersion)
		}

		// Permanent failure — no retry.
		var permErr *PermanentError
		if errors.As(execErr, &permErr) {
			r.tel.recordCompleted(ctx, record.Type, record.Version, StatusFailed)
			return r.store.SetError(ctx, record.ID, StatusFailed, permErr.Err.Error(), occVersion)
		}

		if se, ok := IsSuspend(execErr); ok {
			r.tel.adjustSuspended(ctx, 1)
			return r.store.Suspend(ctx, record.ID, se.ResumeAt, occVersion)
		}

		if record.Attempts+1 >= entry.retryPolicy.MaxAttempts {
			r.tel.recordCompleted(ctx, record.Type, record.Version, StatusFailed)
			return r.store.SetError(ctx, record.ID, StatusFailed, execErr.Error(), occVersion)
		}

		delay := entry.retryPolicy.BaseDelay << record.Attempts
		if delay > entry.retryPolicy.MaxDelay {
			delay = entry.retryPolicy.MaxDelay
		}

		return r.store.SetRetry(ctx, record.ID, r.nowFunc().Add(delay), occVersion)
	}

	// Success — save result.
	r.tel.recordCompleted(ctx, record.Type, record.Version, StatusCompleted)
	return r.store.UpdateStatus(ctx, record.ID, StatusCompleted, resultJSON, occVersion)
}
