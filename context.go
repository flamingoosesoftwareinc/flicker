package flicker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
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
// Workflows see Stop(), Log(), Time, ID, SleepUntil, and Scope — nothing else.
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
	idFn      func() string
	prefix    string
	root      *WorkflowContext // nil for root context, set for scoped children
	mu        *sync.Mutex      // protects seenSteps during parallel execution

	sleepCounter int // auto-incremented counter for SleepUntil step names

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
	target := wc
	if wc.root != nil {
		target = wc.root
	}

	target.stopped.Store(true)

	for _, opt := range opts {
		opt(&target.stopCfg)
	}
}

// Stopped returns true if Stop was called.
func (wc *WorkflowContext) Stopped() bool {
	if wc.root != nil {
		return wc.root.stopped.Load()
	}

	return wc.stopped.Load()
}

// StopError returns the error passed to Stop via WithError, or nil.
func (wc *WorkflowContext) StopError() error {
	if wc.root != nil {
		return wc.root.stopCfg.err
	}

	return wc.stopCfg.err
}

// Log writes a structured log message using slog key-value style.
func (wc *WorkflowContext) Log(msg string, args ...any) {
	if wc.logger != nil {
		wc.logger.Info(msg, args...)
	}
}

// resolveStepName prepends the scope prefix to a step name.
func (wc *WorkflowContext) resolveStepName(name string) string {
	if wc.prefix == "" {
		return name
	}

	return wc.prefix + "/" + name
}

// trackStep records a step name and returns an error if it was already seen.
// Thread-safe for use during parallel execution.
func (wc *WorkflowContext) trackStep(name string) error {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	if _, seen := wc.seenSteps[name]; seen {
		return fmt.Errorf(
			"duplicate step name %q: each step must have a unique name",
			name,
		)
	}

	wc.seenSteps[name] = struct{}{}

	return nil
}

// SleepUntil suspends the workflow until the given time. The resume time is
// durably cached so it survives replay. On re-execution after promotion, if
// the wall clock has passed the cached time, execution continues normally.
func (wc *WorkflowContext) SleepUntil(ctx context.Context, resumeAt time.Time) error {
	wc.sleepCounter++
	stepName := fmt.Sprintf("_sleep.until:%d", wc.sleepCounter)

	cached, err := Run(ctx, wc, stepName, func(_ context.Context) (*time.Time, error) {
		return &resumeAt, nil
	})
	if err != nil {
		return err
	}

	if wc.nowFn().Before(*cached) {
		return &SuspendError{ResumeAt: *cached}
	}

	return nil
}

// Scope creates a child WorkflowContext where all step names are prefixed
// with the given name (e.g., "branch-a/step-name"). Use with Parallel for
// deterministic parallel branches. The child shares seenSteps and stepCache
// with the parent but has its own Time, ID, and sleep providers.
func (wc *WorkflowContext) Scope(name string) *WorkflowContext {
	prefix := name
	if wc.prefix != "" {
		prefix = wc.prefix + "/" + name
	}

	root := wc.root
	if root == nil {
		root = wc
	}

	child := &WorkflowContext{
		id:        wc.id,
		wfType:    wc.wfType,
		version:   wc.version,
		store:     wc.store,
		logger:    wc.logger,
		seenSteps: wc.seenSteps,
		stepCache: wc.stepCache,
		nowFn:     wc.nowFn,
		idFn:      wc.idFn,
		prefix:    prefix,
		root:      root,
		mu:        wc.mu,
	}

	child.Time = NewTimeProvider(child, wc.nowFn)
	child.ID = NewIDProvider(child, wc.idFn)

	return child
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
	// Resolve step name for the subscription — Run resolves separately.
	resolvedName := wc.resolveStepName(stepName)

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
			StepName:       resolvedName,
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
