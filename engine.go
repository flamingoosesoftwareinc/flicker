package flicker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

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

// WithTimePromoter sets the time-based promoter that handles SleepUntil
// deadlines and subscription timeouts. Defaults to a PollingTimePromoter
// with a 1-second interval. Cannot be nil.
func WithTimePromoter(p Promoter) EngineOption {
	return func(e *Engine) {
		e.timePromoter = p
	}
}

// WithPromoter adds an additional promoter that runs alongside the time
// promoter. Use for event-driven sources like message queues, webhooks,
// or database notification channels.
func WithPromoter(p Promoter) EngineOption {
	return func(e *Engine) {
		e.promoters = append(e.promoters, p)
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

// WithRunner sets a custom Runner for workflow execution.
// Defaults to a LocalRunner created from the engine's configuration.
func WithRunner(r Runner) EngineOption {
	return func(e *Engine) {
		e.runner = r
	}
}

// WithScheduler sets a custom Scheduler for dispatching work.
// Defaults to a PollingScheduler.
func WithScheduler(s Scheduler) EngineOption {
	return func(e *Engine) {
		e.scheduler = s
	}
}

// WithDrainTimeout sets the maximum time to wait for in-flight workflows
// to complete during graceful shutdown. Defaults to 30 seconds.
func WithDrainTimeout(d time.Duration) EngineOption {
	return func(e *Engine) {
		e.drainTimeout = d
	}
}

// WithPool sets a custom WorkerPool. Defaults to a PondPool.
// Use this to swap in ants, tunny, or a stdlib semaphore.
func WithPool(p WorkerPool) EngineOption {
	return func(e *Engine) {
		e.pool = p
	}
}

// Engine is the scheduler + runner combined. It polls the store for pending
// workflows, dispatches them to a worker pool, and executes them.
type Engine struct {
	store        WorkflowStore
	registry     map[string]registryEntry
	workers      int
	drainTimeout time.Duration
	logger       *slog.Logger
	idFunc       func() string
	nowFunc      func() time.Time
	runner       Runner
	scheduler    Scheduler
	pool         WorkerPool
	timePromoter Promoter
	promoters    []Promoter
	wg           sync.WaitGroup
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
		drainTimeout: 30 * time.Second,
		logger:       slog.Default(),
		idFunc:       func() string { return uuid.New().String() },
		nowFunc:      func() time.Time { return time.Now().UTC() },
	}

	for _, opt := range opts {
		opt(e)
	}

	// Default time promoter uses the engine's (possibly overridden) clock and logger.
	if e.timePromoter == nil {
		e.timePromoter = &PollingTimePromoter{
			Interval: time.Second,
			NowFunc:  e.nowFunc,
			Logger:   e.logger,
		}
	}

	return e
}

func (e *Engine) generateID() string {
	return e.idFunc()
}

func (e *Engine) register(def definition, policy RetryPolicy) {
	e.registry[def.defName()] = registryEntry{
		def:         def,
		retryPolicy: policy,
	}
}

// getRunner returns the configured runner, or builds a LocalRunner.
func (e *Engine) getRunner() Runner {
	if e.runner != nil {
		return e.runner
	}

	return &LocalRunner{
		registry: e.registry,
		store:    e.store,
		logger:   e.logger,
		nowFunc:  e.nowFunc,
		idFunc:   e.idFunc,
	}
}

// Start begins the scheduler + worker pool. It blocks until ctx is cancelled.
// On cancellation, it waits for in-flight workflows to finish (up to drain timeout).
func (e *Engine) Start(ctx context.Context) error {
	pool := e.pool
	if pool == nil {
		pool = NewPondPool(e.workers)
	}
	runner := e.getRunner()

	// Nudge channel — promoters signal here when workflows become schedulable,
	// causing the scheduler to poll immediately.
	nudge := make(chan struct{}, 1)
	ready := func() {
		select {
		case nudge <- struct{}{}:
		default: // already signaled, don't block
		}
	}

	// Start all promoters in separate goroutines.
	go func() {
		if err := e.timePromoter.Start(ctx, e.store, ready); err != nil {
			e.logger.Info("time promoter exited with error", "error", err)
		}
	}()
	for _, p := range e.promoters {
		go func() {
			if err := p.Start(ctx, e.store, ready); err != nil {
				e.logger.Info("promoter exited with error", "error", err)
			}
		}()
	}

	// Build or use the configured scheduler.
	sched := e.scheduler
	if sched == nil {
		sched = &PollingScheduler{
			Store: e.store,
			Limit: e.workers,
			Nudge: nudge,
		}
	}

	// drainCtx outlives the engine ctx — it gives in-flight workflows time
	// to finish (e.g., complete HTTP calls) after the engine stops accepting
	// new work. Force-cancelled when the drain timeout expires.
	drainCtx, drainCancel := context.WithCancel(context.Background())
	defer drainCancel()

	dispatch := func(records []*WorkflowRecord) {
		for _, record := range records {
			e.wg.Add(1)
			pool.Submit(func() {
				defer e.wg.Done()
				if err := runner.Run(drainCtx, record); err != nil {
					e.logger.Info("workflow execution failed",
						"workflow_id", record.ID, "error", err)
				}
			})
		}
	}

	_ = sched.Start(ctx, dispatch)

	// Start drain timer — force-cancels in-flight workflows if they don't
	// finish within the drain period.
	drainTimer := time.AfterFunc(e.drainTimeout, func() {
		e.logger.Info("drain timeout reached, force-cancelling in-flight workflows",
			"timeout", e.drainTimeout)
		drainCancel()
	})

	// Wait for all in-flight workflows to complete (either gracefully or
	// because the drain timer force-cancelled their context).
	e.wg.Wait()
	drainTimer.Stop()

	// Clean up the pool.
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
	runner := e.getRunner()

	records, err := e.store.ListSchedulable(ctx, e.workers)
	if err != nil {
		return fmt.Errorf("list schedulable: %w", err)
	}

	for _, record := range records {
		if err := runner.Run(ctx, record); err != nil {
			e.logger.Info("workflow execution failed", "workflow_id", record.ID, "error", err)
		}
	}

	return nil
}
