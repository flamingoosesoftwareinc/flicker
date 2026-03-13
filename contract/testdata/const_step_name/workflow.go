package conststepname

import (
	"context"

	"github.com/flamingoosesoftwareinc/flicker"
)

const stepValidate = "validate-input"

type Request struct {
	Value string `json:"value"`
}

type Response struct {
	Valid bool `json:"valid"`
}

type constWorkflow struct {
	wc *flicker.WorkflowContext
}

var _ = flicker.Define[Request, Response]("const-workflow", "v1", func(wc *flicker.WorkflowContext) flicker.Workflow[Request, Response] {
	return &constWorkflow{wc: wc}
})

func (w *constWorkflow) Execute(ctx context.Context, req Request) (Response, error) {
	_, err := flicker.Run[bool](ctx, w.wc, stepValidate, func(ctx context.Context) (*bool, error) {
		b := true
		return &b, nil
	})
	if err != nil {
		return Response{}, err
	}

	return Response{Valid: true}, nil
}
