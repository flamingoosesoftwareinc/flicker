package providers

import (
	"context"

	"github.com/flamingoosesoftwareinc/flicker"
)

type Request struct {
	Name string `json:"name"`
}

type Response struct {
	ID string `json:"id"`
}

type providerWorkflow struct {
	wc      *flicker.WorkflowContext
	entropy *flicker.Provider[int]
}

var _ = flicker.Define[Request, Response]("provider-workflow", "v1", func(wc *flicker.WorkflowContext) flicker.Workflow[Request, Response] {
	return &providerWorkflow{
		wc:      wc,
		entropy: flicker.NewProvider[int](wc, "entropy", func() (int, error) { return 4, nil }),
	}
})

func (w *providerWorkflow) Execute(ctx context.Context, req Request) (Response, error) {
	rng := flicker.NewProvider[string](w.wc, "request-id", func() (string, error) { return "gen-123", nil })

	id, err := w.wc.ID.New(ctx)
	if err != nil {
		return Response{}, err
	}

	_, err = rng.Get(ctx)
	if err != nil {
		return Response{}, err
	}

	return Response{ID: id}, nil
}
