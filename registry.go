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

func (d *WorkflowDef[R]) defVersion() string {
	return d.version
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
func (f *Factory[R]) Submit(ctx context.Context, request R) (*Instance, error) {
	id := f.engine.generateID()

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
