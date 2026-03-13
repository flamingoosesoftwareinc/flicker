package multipleworkflows

import (
	"context"

	"github.com/flamingoosesoftwareinc/flicker"
)

// First workflow

type OrderRequest struct {
	OrderID string `json:"order_id"`
}

type OrderResponse struct {
	Status string `json:"status"`
}

type orderWorkflow struct {
	wc *flicker.WorkflowContext
}

var _ = flicker.Define[OrderRequest, OrderResponse]("order-workflow", "v1", func(wc *flicker.WorkflowContext) flicker.Workflow[OrderRequest, OrderResponse] {
	return &orderWorkflow{wc: wc}
})

func (w *orderWorkflow) Execute(ctx context.Context, req OrderRequest) (OrderResponse, error) {
	_, err := flicker.Run[string](ctx, w.wc, "process-order", func(ctx context.Context) (*string, error) {
		s := "processed"
		return &s, nil
	})
	if err != nil {
		return OrderResponse{}, err
	}
	return OrderResponse{Status: "done"}, nil
}

// Second workflow

type ShipRequest struct {
	TrackingID string `json:"tracking_id"`
}

type ShipResponse struct {
	Shipped bool `json:"shipped"`
}

type shipWorkflow struct {
	wc *flicker.WorkflowContext
}

var _ = flicker.Define[ShipRequest, ShipResponse]("shipping-workflow", "v2", func(wc *flicker.WorkflowContext) flicker.Workflow[ShipRequest, ShipResponse] {
	return &shipWorkflow{wc: wc}
})

func (w *shipWorkflow) Execute(ctx context.Context, req ShipRequest) (ShipResponse, error) {
	_, err := flicker.Run[bool](ctx, w.wc, "ship-item", func(ctx context.Context) (*bool, error) {
		b := true
		return &b, nil
	})
	if err != nil {
		return ShipResponse{}, err
	}
	return ShipResponse{Shipped: true}, nil
}
