package waitforevent

import (
	"context"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
)

type ApprovalRequest struct {
	ItemID string `json:"item_id"`
}

type ApprovalResponse struct {
	Approved bool `json:"approved"`
}

type ApprovalEvent struct {
	ApprovedBy string `json:"approved_by"`
	Decision   bool   `json:"decision"`
}

type approvalWorkflow struct {
	wc *flicker.WorkflowContext
}

var _ = flicker.Define[ApprovalRequest, ApprovalResponse]("approval-workflow", "v1", func(wc *flicker.WorkflowContext) flicker.Workflow[ApprovalRequest, ApprovalResponse] {
	return &approvalWorkflow{wc: wc}
})

func (w *approvalWorkflow) Execute(ctx context.Context, req ApprovalRequest) (ApprovalResponse, error) {
	event, err := flicker.WaitForEvent[ApprovalEvent](ctx, w.wc, "wait-approval", "approval:"+req.ItemID, 24*time.Hour)
	if err != nil {
		return ApprovalResponse{}, err
	}

	return ApprovalResponse{Approved: event.Decision}, nil
}
