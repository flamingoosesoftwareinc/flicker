package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/flamingoosesoftwareinc/flicker"
)

// Workflow is a multi-step order processing workflow that demonstrates
// suspend/resume via SleepUntil. It fetches a user, reserves inventory,
// sleeps until the reservation deadline, checks payment, and saves the result.
type Workflow struct {
	*flicker.WorkflowContext
	client  *http.Client
	baseURL string
	repo    Repository
}

// NewDefinition creates a workflow definition with the given dependencies.
func NewDefinition(
	client *http.Client,
	baseURL string,
	repo Repository,
) *flicker.WorkflowDef[Request] {
	return flicker.Define[Request]("order", "v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[Request] {
			return &Workflow{
				WorkflowContext: wc,
				client:          client,
				baseURL:         baseURL,
				repo:            repo,
			}
		},
	)
}

func (w *Workflow) Execute(ctx context.Context, req Request) error {
	// Step 1: fetch user.
	user, err := flicker.Run(ctx, w.WorkflowContext, "fetch_user",
		func(ctx context.Context) (*User, error) {
			w.Log("fetching user", "customer_id", req.CustomerID)

			resp, err := w.client.Get(w.baseURL + "/users/" + req.CustomerID)
			if err != nil {
				return nil, fmt.Errorf("fetch user: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			var u User
			if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
				return nil, fmt.Errorf("decode user: %w", err)
			}

			return &u, nil
		},
	)
	if err != nil {
		return err
	}

	// Step 2: reserve inventory.
	reservation, err := flicker.Run(ctx, w.WorkflowContext, "reserve_inventory",
		func(ctx context.Context) (*Reservation, error) {
			w.Log("reserving inventory", "order_id", req.OrderID, "user", user.Name)

			resp, err := w.client.Post(
				w.baseURL+"/inventory/reserve",
				"application/json",
				nil,
			)
			if err != nil {
				return nil, fmt.Errorf("reserve inventory: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			var r Reservation
			if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
				return nil, fmt.Errorf("decode reservation: %w", err)
			}

			return &r, nil
		},
	)
	if err != nil {
		return err
	}

	// Between steps: sleep until reservation deadline approaches.
	w.Log("sleeping until reservation deadline", "expires_at", reservation.ExpiresAt)

	if err := w.SleepUntil(ctx, reservation.ExpiresAt); err != nil {
		return err
	}

	// Step 3: check payment status.
	payment, err := flicker.Run(ctx, w.WorkflowContext, "check_payment",
		func(ctx context.Context) (*PaymentResult, error) {
			w.Log("checking payment", "order_id", req.OrderID)

			resp, err := w.client.Get(w.baseURL + "/payments/" + req.OrderID)
			if err != nil {
				return nil, fmt.Errorf("check payment: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			var p PaymentResult
			if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
				return nil, fmt.Errorf("decode payment: %w", err)
			}

			return &p, nil
		},
	)
	if err != nil {
		return err
	}

	// Between steps: decide outcome based on payment status.
	status := "confirmed"
	if payment.Status != "approved" {
		status = "cancelled"
	}

	// Step 4: save final result.
	_, err = flicker.Run(ctx, w.WorkflowContext, "save_result",
		func(ctx context.Context) (*OrderResult, error) {
			result := &OrderResult{
				OrderID:       req.OrderID,
				Status:        status,
				PaymentRef:    payment.Reference,
				ReservationID: reservation.ID,
			}

			w.Log("saving order result", "status", status)

			if err := w.repo.SaveOrder(ctx, result); err != nil {
				return nil, fmt.Errorf("save order: %w", err)
			}

			return result, nil
		},
	)

	return err
}
