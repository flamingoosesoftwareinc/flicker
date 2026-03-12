package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"
)

// Workflow is the core interface. R is the request/input type.
// Execute runs the workflow logic. The return value determines the outcome:
//   - return nil → completed successfully
//   - return error → transient failure, retry per RetryPolicy
//   - Stop(WithError(err)) then return nil → permanent failure, don't retry
type Workflow[R any] interface {
	Execute(ctx context.Context, request R) error
}

// Status represents where the workflow currently is.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
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

// DefaultRetryPolicy returns sensible safe defaults.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
	}
}

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

// WorkflowContext is the framework handle passed to workflow constructors.
// Workflows embed this to get access to Stop(), Log(), and durable services
// like Time and ID.
type WorkflowContext struct {
	id      string
	store   WorkflowStore
	logger  *slog.Logger
	stopped atomic.Bool
	stopCfg stopConfig

	// Time provides durable time operations. w.Time.Now(ctx) returns a
	// cached timestamp that survives replay.
	Time *TimeProvider

	// ID provides durable ID generation. w.ID.New(ctx) returns a cached
	// identifier that survives replay.
	ID *IDProvider
}

// Stop signals that the workflow should stop. Call return after Stop().
// With no options: clean completion. With WithError: permanent failure.
func (wc *WorkflowContext) Stop(opts ...StopOption) {
	wc.stopped.Store(true)

	for _, opt := range opts {
		opt(&wc.stopCfg)
	}
}

// Stopped returns true if Stop was called.
func (wc *WorkflowContext) Stopped() bool {
	return wc.stopped.Load()
}

// StopError returns the error passed to Stop via WithError, or nil.
func (wc *WorkflowContext) StopError() error {
	return wc.stopCfg.err
}

// Log writes a structured log message using slog key-value style.
func (wc *WorkflowContext) Log(msg string, args ...any) {
	if wc.logger != nil {
		wc.logger.Info(msg, args...)
	}
}

// TimeProvider is a durable service for time operations.
// Each call to Now caches the result as a step, making it deterministic on replay.
type TimeProvider struct {
	wc      *WorkflowContext
	counter int
	nowFn   func() time.Time
}

// Now returns the current time. On first execution, the real time is captured
// and cached. On replay, the cached value is returned.
func (tp *TimeProvider) Now(ctx context.Context) (time.Time, error) {
	tp.counter++
	stepName := fmt.Sprintf("_time.now:%d", tp.counter)

	var t time.Time

	err := Run(ctx, tp.wc, stepName, &t, func(ctx context.Context) (time.Time, error) {
		return tp.nowFn(), nil
	})

	return t, err
}

// IDProvider is a durable service for ID generation.
// Each call to New caches the result as a step, making it deterministic on replay.
type IDProvider struct {
	wc      *WorkflowContext
	counter int
	newFn   func() string
}

// New returns a new unique identifier. On first execution, a real ID is generated
// and cached. On replay, the cached value is returned.
func (ip *IDProvider) New(ctx context.Context) (string, error) {
	ip.counter++
	stepName := fmt.Sprintf("_id.new:%d", ip.counter)

	var id string

	err := Run(ctx, ip.wc, stepName, &id, func(ctx context.Context) (string, error) {
		return ip.newFn(), nil
	})

	return id, err
}

// Run executes a named durable step. On first execution, fn runs and
// the result is cached. On replay (retry), the cached result is deserialized
// into dest and fn is skipped. T must be JSON-serializable.
func Run[T any](
	ctx context.Context,
	wc *WorkflowContext,
	stepName string,
	dest *T,
	fn func(context.Context) (T, error),
) error {
	// Read-through: check cache.
	cached, err := wc.store.GetStepResult(ctx, wc.id, stepName)
	if err == nil && cached != nil {
		return json.Unmarshal(cached.Result, dest)
	}

	// Cache miss — execute.
	result, fnErr := fn(ctx)
	if fnErr != nil {
		return fnErr
	}

	// Write-through: cache successful result.
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal step %q result: %w", stepName, err)
	}

	if err := wc.store.SaveStepResult(ctx, &StepResult{
		WorkflowID: wc.id,
		StepName:   stepName,
		Result:     data,
	}); err != nil {
		return fmt.Errorf("save step %q result: %w", stepName, err)
	}

	*dest = result

	return nil
}

// definition is the type-erased interface for the engine registry.
type definition interface {
	defName() string
	executeWorkflow(ctx context.Context, wc *WorkflowContext, payload []byte) error
}

// WorkflowDef ties a workflow type to its identity and constructor.
type WorkflowDef[R any] struct {
	name    string
	version string
	factory func(*WorkflowContext) Workflow[R]
}

// Define creates a workflow definition with an explicit name and version.
func Define[R any](
	name, version string,
	factory func(*WorkflowContext) Workflow[R],
) *WorkflowDef[R] {
	return &WorkflowDef[R]{
		name:    name,
		version: version,
		factory: factory,
	}
}

// Register adds this workflow definition to the engine and returns a
// Factory that can submit new instances. This is the only way to submit
// workflows — the engine and identity are bound at registration.
func (d *WorkflowDef[R]) Register(e *Engine, policy ...RetryPolicy) *Factory[R] {
	p := DefaultRetryPolicy()
	if len(policy) > 0 {
		p = policy[0]
	}

	e.register(d, p)

	return &Factory[R]{
		def:    d,
		engine: e,
	}
}

func (d *WorkflowDef[R]) defName() string {
	return d.name + ":" + d.version
}

func (d *WorkflowDef[R]) executeWorkflow(
	ctx context.Context,
	wc *WorkflowContext,
	payload []byte,
) error {
	var req R
	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("unmarshal request: %w", err)
	}

	wf := d.factory(wc)

	return wf.Execute(ctx, req)
}

// Factory is a registered workflow type bound to an engine.
// Use Submit to create new workflow instances.
type Factory[R any] struct {
	def    *WorkflowDef[R]
	engine *Engine
}

// Submit creates a new workflow instance and returns a handle to it.
// The workflow ID is generated by the engine's ID provider.
func (f *Factory[R]) Submit(ctx context.Context, request R) (*Instance, error) {
	id := f.engine.generateID()

	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	if err := f.engine.store.Create(ctx, &WorkflowRecord{
		ID:      id,
		Type:    f.def.defName(),
		Status:  StatusPending,
		Payload: payload,
	}); err != nil {
		return nil, fmt.Errorf("create workflow: %w", err)
	}

	return &Instance{
		id:    id,
		store: f.engine.store,
	}, nil
}

// Instance is a handle to a submitted workflow. Use it to query status.
type Instance struct {
	id    string
	store WorkflowStore
}

// ID returns the workflow instance ID.
func (i *Instance) ID() string {
	return i.id
}

// Status returns the current status of the workflow.
func (i *Instance) Status(ctx context.Context) (Status, error) {
	record, err := i.store.Get(ctx, i.id)
	if err != nil {
		return "", fmt.Errorf("get workflow: %w", err)
	}

	return record.Status, nil
}
