package simple

import (
	"context"

	"github.com/flamingoosesoftwareinc/flicker"
)

type OrderRequest struct {
	OrderID string `json:"order_id"`
	Amount  int    `json:"amount"`
}

type OrderResponse struct {
	Confirmed bool   `json:"confirmed"`
	Message   string `json:"message"`
}

type orderWorkflow struct {
	wc *flicker.WorkflowContext
}

var _ = flicker.Define[OrderRequest, OrderResponse]("order-process", "v1", func(wc *flicker.WorkflowContext) flicker.Workflow[OrderRequest, OrderResponse] {
	return &orderWorkflow{wc: wc}
})

func (w *orderWorkflow) Execute(ctx context.Context, req OrderRequest) (OrderResponse, error) {
	now, err := w.wc.Time.Now(ctx)
	if err != nil {
		return OrderResponse{}, err
	}

	result, err := flicker.Run[string](ctx, w.wc, "validate-order", func(ctx context.Context) (*string, error) {
		s := "validated"
		return &s, nil
	})
	if err != nil {
		return OrderResponse{}, err
	}

	_, err = flicker.Run[bool](ctx, w.wc, "charge-payment", func(ctx context.Context) (*bool, error) {
		b := true
		return &b, nil
	})
	if err != nil {
		return OrderResponse{}, err
	}

	_ = now
	_ = result

	return OrderResponse{Confirmed: true, Message: "done"}, nil
}
