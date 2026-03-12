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

// User is a struct returned from the "fetch" step — exercises struct caching.
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Greeting is the final output — exercises struct with computed fields.
type Greeting struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

// Workflow is a multi-step durable workflow that fetches a user,
// computes a greeting, and persists the result.
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
	// Step 1: fetch user (durable read, returns struct).
	user, err := flicker.Run(ctx, w.WorkflowContext, "fetch_user",
		func(_ context.Context) (*User, error) {
			w.Log("fetching user", "user_id", req.UserID)

			return &User{ID: req.UserID, Name: "Alice"}, nil
		},
	)
	if err != nil {
		return err
	}

	// Step 2: get a durable timestamp.
	now, err := w.Time.Now(ctx)
	if err != nil {
		return err
	}

	// Between steps: pure computation on cached data.
	msg := fmt.Sprintf("Hello, %s! (%s)", user.Name, now.Format("2006-01-02T15:04:05Z"))

	// Step 3: send greeting (durable write, returns struct).
	_, err = flicker.Run(ctx, w.WorkflowContext, "send_greeting",
		func(_ context.Context) (*Greeting, error) {
			w.Log("sending greeting", "greeting", msg)

			return &Greeting{Message: msg, UserID: user.ID}, nil
		},
	)

	return err
}
