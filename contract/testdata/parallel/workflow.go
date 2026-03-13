package parallel

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

type parallelWorkflow struct {
	wc *flicker.WorkflowContext
}

var _ = flicker.Define[Request, Response]("parallel-workflow", "v1", func(wc *flicker.WorkflowContext) flicker.Workflow[Request, Response] {
	return &parallelWorkflow{wc: wc}
})

func (w *parallelWorkflow) Execute(ctx context.Context, req Request) (Response, error) {
	err := flicker.Parallel(ctx, w.wc,
		flicker.NewBranch("fetch-user", func(ctx context.Context, wc *flicker.WorkflowContext) error {
			_, err := flicker.Run[string](ctx, wc, "call-api", func(ctx context.Context) (*string, error) {
				s := "user-data"
				return &s, nil
			})
			return err
		}),
		flicker.NewBranch("check-inventory", func(ctx context.Context, wc *flicker.WorkflowContext) error {
			_, err := flicker.Run[int](ctx, wc, "query-db", func(ctx context.Context) (*int, error) {
				v := 42
				return &v, nil
			})
			return err
		}),
	)
	if err != nil {
		return Response{}, err
	}

	return Response{OK: true}, nil
}
