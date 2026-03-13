package dynamicstepname

import (
	"context"
	"fmt"

	"github.com/flamingoosesoftwareinc/flicker"
)

type Request struct {
	Name string `json:"name"`
}

type Response struct {
	OK bool `json:"ok"`
}

type dynamicWorkflow struct {
	wc *flicker.WorkflowContext
}

var _ = flicker.Define[Request, Response]("dynamic-workflow", "v1", func(wc *flicker.WorkflowContext) flicker.Workflow[Request, Response] {
	return &dynamicWorkflow{wc: wc}
})

func (w *dynamicWorkflow) Execute(ctx context.Context, req Request) (Response, error) {
	stepName := fmt.Sprintf("step-%s", req.Name)

	_, err := flicker.Run[string](ctx, w.wc, stepName, func(ctx context.Context) (*string, error) {
		s := "ok"
		return &s, nil
	})
	if err != nil {
		return Response{}, err
	}

	return Response{OK: true}, nil
}
