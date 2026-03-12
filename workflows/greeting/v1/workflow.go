package v1

import (
	"context"
	"fmt"

	"github.com/flamingoosesoftwareinc/flicker"
)

// Request is the input to the greeting workflow.
type Request struct {
	UserID string `json:"user_id"`
}

// Workflow is a minimal two-step durable workflow that fetches a name
// and sends a greeting.
type Workflow struct {
	*flicker.WorkflowContext
}

// Definition is the workflow's identity and constructor.
var Definition = flicker.Define[Request]("greeting", "v1",
	func(wc *flicker.WorkflowContext) flicker.Workflow[Request] {
		return &Workflow{WorkflowContext: wc}
	},
)

func (w *Workflow) Execute(ctx context.Context, req Request) error {
	// Step 1: fetch name (durable read).
	var name string

	if err := flicker.Run(
		ctx,
		w.WorkflowContext,
		"fetch_name",
		&name,
		func(_ context.Context) (string, error) {
			w.Log("fetching name", "user_id", req.UserID)

			return "Alice", nil
		},
	); err != nil {
		return err
	}

	// Step 2: get a durable timestamp.
	now, err := w.Time.Now(ctx)
	if err != nil {
		return err
	}

	// Step 3: send greeting (durable write).
	var greeting string

	if err := flicker.Run(
		ctx,
		w.WorkflowContext,
		"send_greeting",
		&greeting,
		func(_ context.Context) (string, error) {
			msg := fmt.Sprintf("Hello, %s! (%s)", name, now.Format("2006-01-02T15:04:05Z"))
			w.Log("sending greeting", "greeting", msg)

			return msg, nil
		},
	); err != nil {
		return err
	}

	return nil
}
