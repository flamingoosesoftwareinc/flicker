package unresolvablefactory

import (
	"context"

	"github.com/flamingoosesoftwareinc/flicker"
)

type Request struct {
	ID string `json:"id"`
}

type Response struct {
	OK bool `json:"ok"`
}

type unresolvedWorkflow struct {
	wc *flicker.WorkflowContext
}

func makeFactory() func(*flicker.WorkflowContext) flicker.Workflow[Request, Response] {
	return func(wc *flicker.WorkflowContext) flicker.Workflow[Request, Response] {
		return &unresolvedWorkflow{wc: wc}
	}
}

var factory = makeFactory()

var _ = flicker.Define[Request, Response]("unresolved-workflow", "v1", factory)

func (w *unresolvedWorkflow) Execute(ctx context.Context, req Request) (Response, error) {
	return Response{OK: true}, nil
}
