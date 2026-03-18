package flicker

import (
	"context"
	"encoding/json"
	"fmt"
)

// definition is the type-erased interface for the engine registry.
type definition interface {
	defName() string
	defVersion() string
	executeWorkflow(
		ctx context.Context,
		wc *WorkflowContext,
		payload []byte,
	) (json.RawMessage, error)
}

// WorkflowDef ties a workflow type to its identity and constructor.
type WorkflowDef[R, Resp any] struct {
	name    string
	version string
	factory func(*WorkflowContext) Workflow[R, Resp]
}

// Define creates a workflow definition with an explicit name and version.
func Define[R, Resp any](
	name, version string,
	factory func(*WorkflowContext) Workflow[R, Resp],
) *WorkflowDef[R, Resp] {
	return &WorkflowDef[R, Resp]{
		name:    name,
		version: version,
		factory: factory,
	}
}

// Register adds this workflow definition to the engine and returns a
// Factory that can submit new instances. This is the only way to submit
// workflows — the engine and identity are bound at registration.
func (d *WorkflowDef[R, Resp]) Register(e *Engine, policy ...RetryPolicy) *Factory[R, Resp] {
	p := DefaultRetryPolicy()
	if len(policy) > 0 {
		p = policy[0]
	}

	e.register(d, p)

	return &Factory[R, Resp]{
		def:    d,
		engine: e,
	}
}

func (d *WorkflowDef[R, Resp]) defName() string {
	return d.name + ":" + d.version
}

func (d *WorkflowDef[R, Resp]) defVersion() string {
	return d.version
}

func (d *WorkflowDef[R, Resp]) executeWorkflow(
	ctx context.Context,
	wc *WorkflowContext,
	payload []byte,
) (json.RawMessage, error) {
	var req R
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("unmarshal request: %w", err)
	}

	wf := d.factory(wc)

	resp, err := wf.Execute(ctx, req)
	if err != nil {
		return nil, err
	}

	result, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal result: %w", marshalErr)
	}

	return result, nil
}

// Factory is a registered workflow type bound to an engine.
// Use Submit to create new workflow instances.
type Factory[R, Resp any] struct {
	def    *WorkflowDef[R, Resp]
	engine *Engine
}

// SubmitOption configures a single Submit call.
type SubmitOption func(*submitConfig)

type submitConfig struct {
	id string
}

// WithID sets a deterministic workflow ID. If the ID already exists in the
// store, Submit will return an error — use this for idempotent submission
// where duplicate IDs are expected and the error can be safely ignored.
func WithID(id string) SubmitOption {
	return func(cfg *submitConfig) {
		cfg.id = id
	}
}

// Submit creates a new workflow instance and returns a typed handle to it.
func (f *Factory[R, Resp]) Submit(
	ctx context.Context,
	request R,
	opts ...SubmitOption,
) (*TypedInstance[Resp], error) {
	var cfg submitConfig

	for _, o := range opts {
		o(&cfg)
	}

	id := cfg.id
	if id == "" {
		id = f.engine.generateID()
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	if err := f.engine.store.Create(ctx, &WorkflowRecord{
		ID:      id,
		Type:    f.def.defName(),
		Version: f.def.version,
		Status:  StatusPending,
		Payload: payload,
	}); err != nil {
		return nil, fmt.Errorf("create workflow: %w", err)
	}

	f.engine.tel.recordSubmitted(ctx, f.def.defName(), f.def.version)

	return &TypedInstance[Resp]{
		Instance: &Instance{
			id:    id,
			store: f.engine.store,
		},
	}, nil
}

// Instance is a non-generic handle to a submitted workflow.
// Use it for status checks, heterogeneous collections, and operational queries.
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

// RawResult returns the raw JSON result of the workflow, or nil if not completed.
func (i *Instance) RawResult(ctx context.Context) (json.RawMessage, error) {
	record, err := i.store.Get(ctx, i.id)
	if err != nil {
		return nil, fmt.Errorf("get workflow: %w", err)
	}

	return record.Result, nil
}

// WorkflowResult is a typed result container with workflow metadata.
type WorkflowResult[Resp any] struct {
	ID       string
	Type     string
	Version  string
	Status   Status
	Response Resp
	Error    string
}

// TypedInstance is a generic handle to a submitted workflow that can
// deserialize the result into the response type.
type TypedInstance[Resp any] struct {
	*Instance
}

// Result returns the full workflow result with metadata and typed response.
// The Response field is only meaningful when Status is StatusCompleted.
func (i *TypedInstance[Resp]) Result(ctx context.Context) (*WorkflowResult[Resp], error) {
	record, err := i.store.Get(ctx, i.id)
	if err != nil {
		return nil, fmt.Errorf("get workflow: %w", err)
	}

	result := &WorkflowResult[Resp]{
		ID:      record.ID,
		Type:    record.Type,
		Version: record.Version,
		Status:  record.Status,
		Error:   record.Error,
	}

	if record.Status == StatusCompleted && len(record.Result) > 0 {
		if err := json.Unmarshal(record.Result, &result.Response); err != nil {
			return nil, fmt.Errorf("unmarshal result: %w", err)
		}
	}

	return result, nil
}
