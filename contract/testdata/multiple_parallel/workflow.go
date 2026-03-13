package multipleparallel

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

type multiParallelWorkflow struct {
	wc *flicker.WorkflowContext
}

var _ = flicker.Define[Request, Response]("multi-parallel", "v1", func(wc *flicker.WorkflowContext) flicker.Workflow[Request, Response] {
	return &multiParallelWorkflow{wc: wc}
})

func (w *multiParallelWorkflow) Execute(ctx context.Context, req Request) (Response, error) {
	_, err := flicker.Run[string](ctx, w.wc, "setup", func(ctx context.Context) (*string, error) {
		s := "ready"
		return &s, nil
	})
	if err != nil {
		return Response{}, err
	}

	// First parallel: fetch data from two sources.
	err = flicker.Parallel(ctx, w.wc,
		flicker.NewBranch("fetch-users", func(ctx context.Context, wc *flicker.WorkflowContext) error {
			_, err := flicker.Run[string](ctx, wc, "call-user-api", func(ctx context.Context) (*string, error) {
				s := "users"
				return &s, nil
			})
			return err
		}),
		flicker.NewBranch("fetch-products", func(ctx context.Context, wc *flicker.WorkflowContext) error {
			_, err := flicker.Run[string](ctx, wc, "call-product-api", func(ctx context.Context) (*string, error) {
				s := "products"
				return &s, nil
			})
			return err
		}),
	)
	if err != nil {
		return Response{}, err
	}

	// Second parallel: notify downstream systems.
	err = flicker.Parallel(ctx, w.wc,
		flicker.NewBranch("notify-warehouse", func(ctx context.Context, wc *flicker.WorkflowContext) error {
			_, err := flicker.Run[bool](ctx, wc, "send-warehouse-event", func(ctx context.Context) (*bool, error) {
				b := true
				return &b, nil
			})
			return err
		}),
		flicker.NewBranch("notify-billing", func(ctx context.Context, wc *flicker.WorkflowContext) error {
			_, err := flicker.Run[bool](ctx, wc, "send-billing-event", func(ctx context.Context) (*bool, error) {
				b := true
				return &b, nil
			})
			return err
		}),
		flicker.NewBranch("notify-analytics", func(ctx context.Context, wc *flicker.WorkflowContext) error {
			_, err := flicker.Run[int](ctx, wc, "send-analytics-event", func(ctx context.Context) (*int, error) {
				v := 1
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
