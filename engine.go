package flicker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/alitto/pond/v2"
	"github.com/google/uuid"
)

// EngineOption configures the engine.
type EngineOption func(*Engine)

// WithWorkers sets the number of worker goroutines.
func WithWorkers(n int) EngineOption {
	return func(e *Engine) {
		e.workers = n
	}
}

// WithPollInterval sets how often the scheduler polls for schedulable work.
func WithPollInterval(d time.Duration) EngineOption {
	return func(e *Engine) {
		e.pollInterval = d
	}
}

// WithPromoteInterval sets how often the engine promotes suspended workflows
// whose resume time has passed and times out expired event subscriptions.
// If not set, time-based promotion must be triggered manually via Promote().
func WithPromoteInterval(d time.Duration) EngineOption {
	return func(e *Engine) {
		e.promoteInterval = d
	}
}

// WithLogger sets the engine's logger.
func WithLogger(l *slog.Logger) EngineOption {
	return func(e *Engine) {
		e.logger = l
	}
}

// WithIDFunc sets a custom ID generator for workflow instances.
// Defaults to UUID v4. Useful for deterministic IDs in tests.
func WithIDFunc(fn func() string) EngineOption {
	return func(e *Engine) {
		e.idFunc = fn
	}
}

// WithNowFunc sets a custom time provider for the engine's internal clock
// (retry scheduling) and for the durable TimeProvider on WorkflowContext.
// Defaults to time.Now().UTC(). Useful for deterministic timestamps in tests.
func WithNowFunc(fn func() time.Time) EngineOption {
	return func(e *Engine) {
		e.nowFunc = fn
	}
}

// Engine is the scheduler + runner combined. It polls the store for pending
// workflows, dispatches them to a worker pool, and executes them.
type Engine struct {
	store           WorkflowStore
	registry        map[string]registryEntry
	workers         int
	pollInterval    time.Duration
	promoteInterval time.Duration
	logger          *slog.Logger
	idFunc          func() string
	nowFunc         func() time.Time
}

type registryEntry struct {
	def         definition
	retryPolicy RetryPolicy
}

// NewEngine creates a new engine with the given store and options.
func NewEngine(store WorkflowStore, opts ...EngineOption) *Engine {
	e := &Engine{
		store:        store,
		registry:     make(map[string]registryEntry),
		workers:      1,
		pollInterval: time.Second,
		logger:       slog.Default(),
		idFunc:       func() string { return uuid.New().String() },
		nowFunc:      func() time.Time { return time.Now().UTC() },
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

func (e *Engine) generateID() string {
	return e.idFunc()
}

func (e *Engine) now() time.Time {
	return e.nowFunc()
}

func (e *Engine) register(def definition, policy RetryPolicy) {
	e.registry[def.defName()] = registryEntry{
		def:         def,
		retryPolicy: policy,
	}
}

// Start begins the scheduler + worker pool. It blocks until ctx is cancelled.
func (e *Engine) Start(ctx context.Context) error {
	pool := pond.NewPool(e.workers)

	// Launch built-in promotion loops as separate goroutines — they must
	// not consume worker pool slots.
	if e.promoteInterval > 0 {
		go promotionLoop(ctx, e.promoteInterval, e.store, e.nowFunc, e.logger)
		go subscriptionTimeoutLoop(ctx, e.promoteInterval, e.store, e.nowFunc, e.logger)
	}

	e.scheduler(ctx, pool)
	pool.StopAndWait()

	return nil
}

// Promote runs one time-based promotion cycle. Use in tests with RunOnce
// for explicit control over when suspended workflows get promoted.
func (e *Engine) Promote(ctx context.Context) (int, error) {
	return promote(ctx, e.store, e.nowFunc, e.logger)
}

// TimeOutSubscriptions runs one subscription timeout cycle. Use in tests
// with RunOnce for explicit control over when event subscriptions time out.
func (e *Engine) TimeOutSubscriptions(ctx context.Context) (int, error) {
	return timeOutSubscriptions(ctx, e.store, e.nowFunc, e.logger)
}

// SendEvent delivers an event payload to a workflow waiting on the given
// correlation key. The payload is saved as the step result, the subscription
// is deleted, and the workflow is promoted to pending.
func (e *Engine) SendEvent(ctx context.Context, correlationKey string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}

	if err := e.store.ResumeSubscription(ctx, correlationKey, data); err != nil {
		return fmt.Errorf("resume subscription: %w", err)
	}

	return nil
}

// RunOnce executes a single poll cycle — useful for testing.
func (e *Engine) RunOnce(ctx context.Context) error {
	records, err := e.store.ListSchedulable(ctx, e.workers)
	if err != nil {
		return fmt.Errorf("list schedulable: %w", err)
	}

	for _, record := range records {
		if err := e.executeWorkflow(ctx, record); err != nil {
			e.logger.Info("workflow execution failed", "workflow_id", record.ID, "error", err)
		}
	}

	return nil
}

func (e *Engine) scheduler(ctx context.Context, pool pond.Pool) {
	ticker := time.NewTicker(e.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			records, err := e.store.ListSchedulable(ctx, e.workers)
			if err != nil {
				e.logger.Info("poll failed", "error", err)

				continue
			}

			for _, record := range records {
				pool.Submit(func() {
					if err := e.executeWorkflow(ctx, record); err != nil {
						e.logger.Info("workflow execution failed",
							"workflow_id", record.ID, "error", err)
					}
				})
			}
		}
	}
}

func (e *Engine) executeWorkflow(ctx context.Context, record *WorkflowRecord) error {
	entry, ok := e.registry[record.Type]
	if !ok {
		return fmt.Errorf("no definition registered for %q", record.Type)
	}

	// Transition to running.
	if err := e.store.UpdateStatus(ctx, record.ID, StatusRunning, record.OCCVersion); err != nil {
		return fmt.Errorf("set running: %w", err)
	}

	occAfterRunning := record.OCCVersion + 1

	// Create the workflow context — fully initialized before the factory sees it.
	wc := &WorkflowContext{
		id:        record.ID,
		wfType:    record.Type,
		version:   record.Version,
		store:     e.store,
		logger:    e.logger,
		nowFn:     e.nowFunc,
		seenSteps: make(map[string]struct{}),
	}
	wc.Time = NewTimeProvider(wc, e.nowFunc)
	wc.ID = NewIDProvider(wc, e.idFunc)

	execErr := entry.def.executeWorkflow(ctx, wc, record.Payload)

	return e.resolveOutcome(ctx, record, entry, occAfterRunning, wc, execErr)
}

func (e *Engine) resolveOutcome(
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
			return e.store.SetError(ctx, record.ID, StatusFailed, stopErr.Error(), occVersion)
		}

		return e.store.UpdateStatus(ctx, record.ID, StatusCompleted, occVersion)
	}

	if execErr != nil {
		if se, ok := IsSuspend(execErr); ok {
			return e.store.Suspend(ctx, record.ID, se.ResumeAt, occVersion)
		}

		if record.Attempts+1 >= entry.retryPolicy.MaxAttempts {
			return e.store.SetError(ctx, record.ID, StatusFailed, execErr.Error(), occVersion)
		}

		delay := entry.retryPolicy.BaseDelay << record.Attempts
		if delay > entry.retryPolicy.MaxDelay {
			delay = entry.retryPolicy.MaxDelay
		}

		return e.store.SetRetry(ctx, record.ID, e.now().Add(delay), occVersion)
	}

	return e.store.UpdateStatus(ctx, record.ID, StatusCompleted, occVersion)
}
