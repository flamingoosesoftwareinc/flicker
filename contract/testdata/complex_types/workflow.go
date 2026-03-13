package complextypes

import (
	"context"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
)

type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
}

type OrderItem struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type ComplexRequest struct {
	UserID    string            `json:"user_id"`
	Items     []OrderItem       `json:"items"`
	Metadata  map[string]string `json:"metadata"`
	Address   *Address          `json:"address"`
	CreatedAt time.Time         `json:"created_at"`
}

type ComplexResponse struct {
	Results []string `json:"results"`
	Total   int      `json:"total"`
}

type ItemResult struct {
	SKU    string `json:"sku"`
	Status string `json:"status"`
}

type complexWorkflow struct {
	wc *flicker.WorkflowContext
}

var _ = flicker.Define[ComplexRequest, ComplexResponse]("complex-workflow", "v1", func(wc *flicker.WorkflowContext) flicker.Workflow[ComplexRequest, ComplexResponse] {
	return &complexWorkflow{wc: wc}
})

func (w *complexWorkflow) Execute(ctx context.Context, req ComplexRequest) (ComplexResponse, error) {
	_, err := flicker.Run[ItemResult](ctx, w.wc, "process-items", func(ctx context.Context) (*ItemResult, error) {
		return &ItemResult{SKU: "test", Status: "ok"}, nil
	})
	if err != nil {
		return ComplexResponse{}, err
	}

	_, err = flicker.Run[map[string]int](ctx, w.wc, "aggregate", func(ctx context.Context) (*map[string]int, error) {
		m := map[string]int{"total": 1}
		return &m, nil
	})
	if err != nil {
		return ComplexResponse{}, err
	}

	return ComplexResponse{Results: []string{"done"}, Total: 1}, nil
}
