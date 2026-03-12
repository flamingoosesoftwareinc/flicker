package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/flamingoosesoftwareinc/flicker/internal/generate"
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
	store        WorkflowStore
	registry     map[string]registryEntry
	workers      int
	pollInterval time.Duration
	logger       *slog.Logger
	idFunc       func() string
	nowFunc      func() time.Time
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
		idFunc:       generate.ID,
		nowFunc:      generate.Now,
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
	work := make(chan string, e.workers)

	var wg sync.WaitGroup

	for range e.workers {
		wg.Add(1)

		go func() {
			defer wg.Done()
			e.worker(ctx, work)
		}()
	}

	e.scheduler(ctx, work)
	close(work)
	wg.Wait()

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

func (e *Engine) scheduler(ctx context.Context, work chan<- string) {
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

			for _, r := range records {
				select {
				case work <- r.ID:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func (e *Engine) worker(ctx context.Context, work <-chan string) {
	for {
		select {
		case <-ctx.Done():
			return
		case id, ok := <-work:
			if !ok {
				return
			}

			record, err := e.store.Get(ctx, id)
			if err != nil {
				e.logger.Info("load workflow failed", "workflow_id", id, "error", err)

				continue
			}

			if err := e.executeWorkflow(ctx, record); err != nil {
				e.logger.Info("workflow execution failed", "workflow_id", id, "error", err)
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
		id:     record.ID,
		store:  e.store,
		logger: e.logger,
	}
	wc.Time = &TimeProvider{wc: wc, nowFn: e.nowFunc}
	wc.ID = &IDProvider{wc: wc, newFn: e.idFunc}

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
