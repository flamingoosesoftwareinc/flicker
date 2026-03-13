package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
	"github.com/flamingoosesoftwareinc/flicker/sqlite"
	"github.com/stretchr/testify/require"
)

// --- API types ---

type OrderRequest struct {
	UserID   string `json:"user_id"`
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type Order struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
	Status   string `json:"status"`
}

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Inventory struct {
	SKU       string `json:"sku"`
	Available int    `json:"available"`
}

type PaymentConfirmation struct {
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int    `json:"amount"`
}

type Confirmation struct {
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

// --- Fake API server ---

type apiCallCounts struct {
	createOrder  atomic.Int32
	getUser      atomic.Int32
	getInventory atomic.Int32
	confirmOrder atomic.Int32
}

func newFakeAPI(t *testing.T, counts *apiCallCounts) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/orders", func(w http.ResponseWriter, r *http.Request) {
		counts.createOrder.Add(1)

		var req OrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := Order{
			ID:       "order-001",
			UserID:   req.UserID,
			SKU:      req.SKU,
			Quantity: req.Quantity,
			Status:   "created",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("GET /api/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		counts.getUser.Add(1)

		resp := User{
			ID:    r.PathValue("id"),
			Name:  "Alice Smith",
			Email: "alice@example.com",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("GET /api/inventory/{sku}", func(w http.ResponseWriter, r *http.Request) {
		counts.getInventory.Add(1)

		resp := Inventory{
			SKU:       r.PathValue("sku"),
			Available: 42,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("POST /api/orders/{id}/confirm", func(w http.ResponseWriter, r *http.Request) {
		counts.confirmOrder.Add(1)

		body, _ := io.ReadAll(r.Body)
		var payment PaymentConfirmation
		_ = json.Unmarshal(body, &payment)

		resp := Confirmation{
			OrderID:       r.PathValue("id"),
			TransactionID: payment.TransactionID,
			Status:        "confirmed",
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(mux)
}

// --- Workflow implementation ---

type orderFulfillmentWorkflow struct {
	wc         *flicker.WorkflowContext
	httpClient *http.Client
	apiURL     string
}

func (w *orderFulfillmentWorkflow) Execute(
	ctx context.Context,
	req OrderRequest,
) (Confirmation, error) {
	var zero Confirmation

	// Step 1: Create the order.
	order, err := flicker.Run(ctx, w.wc, "create-order", func(ctx context.Context) (*Order, error) {
		return httpPost[Order](ctx, w.httpClient, w.apiURL+"/api/orders", req)
	})
	if err != nil {
		return zero, err
	}

	w.wc.Log("order created", "order_id", order.ID)

	// Step 2: Parallel enrichment — fetch user details + check inventory.
	var user *User
	var inventory *Inventory

	err = flicker.Parallel(
		ctx,
		w.wc,
		flicker.NewBranch(
			"fetch-user",
			func(ctx context.Context, wc *flicker.WorkflowContext) error {
				u, runErr := flicker.Run(ctx, wc, "call", func(ctx context.Context) (*User, error) {
					return httpGet[User](ctx, w.httpClient, w.apiURL+"/api/users/"+req.UserID)
				})
				if runErr != nil {
					return runErr
				}
				user = u
				return nil
			},
		),
		flicker.NewBranch(
			"check-inventory",
			func(ctx context.Context, wc *flicker.WorkflowContext) error {
				inv, runErr := flicker.Run(
					ctx,
					wc,
					"call",
					func(ctx context.Context) (*Inventory, error) {
						return httpGet[Inventory](
							ctx,
							w.httpClient,
							w.apiURL+"/api/inventory/"+req.SKU,
						)
					},
				)
				if runErr != nil {
					return runErr
				}
				inventory = inv
				return nil
			},
		),
	)
	if err != nil {
		return zero, err
	}

	w.wc.Log("enrichment complete",
		"user", user.Name,
		"available", inventory.Available)

	// Step 3: Validate inventory.
	_, err = flicker.Run(ctx, w.wc, "validate-inventory", func(_ context.Context) (*bool, error) {
		if inventory.Available < req.Quantity {
			return nil, fmt.Errorf("insufficient inventory: need %d, have %d",
				req.Quantity, inventory.Available)
		}
		return flicker.Val(true), nil
	})
	if err != nil {
		return zero, err
	}

	// Step 4: Sleep until processing window (simulates batch processing time).
	now, err := w.wc.Time.Now(ctx)
	if err != nil {
		return zero, err
	}

	err = w.wc.SleepUntil(ctx, now.Add(1*time.Hour))
	if err != nil {
		return zero, err
	}

	// Step 5: Wait for payment confirmation webhook.
	payment, err := flicker.WaitForEvent[PaymentConfirmation](
		ctx, w.wc, "await-payment",
		"payment:"+order.ID,
		24*time.Hour,
	)
	if err != nil {
		return zero, err
	}

	w.wc.Log("payment received", "transaction_id", payment.TransactionID)

	// Step 6: Confirm the order.
	confirmation, err := flicker.Run(
		ctx,
		w.wc,
		"confirm-order",
		func(ctx context.Context) (*Confirmation, error) {
			return httpPost[Confirmation](
				ctx,
				w.httpClient,
				w.apiURL+"/api/orders/"+order.ID+"/confirm",
				*payment,
			)
		},
	)
	if err != nil {
		return zero, err
	}

	w.wc.Log("order confirmed", "status", confirmation.Status)

	return *confirmation, nil
}

// --- HTTP helpers ---

func httpGet[T any](ctx context.Context, client *http.Client, url string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GET %s: status %d: %s", url, resp.StatusCode, body)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func httpPost[T any](
	ctx context.Context,
	client *http.Client,
	url string,
	payload any,
) (*T, error) {
	pr, pw := io.Pipe()

	go func() {
		pw.CloseWithError(json.NewEncoder(pw).Encode(payload))
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("POST %s: status %d: %s", url, resp.StatusCode, body)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// --- Test ---

func TestIntegration_OrderFulfillment(t *testing.T) {
	ctx := context.Background()

	// Fake API server with call counting.
	var counts apiCallCounts
	server := newFakeAPI(t, &counts)
	defer server.Close()

	// Deterministic clock and IDs.
	clock := &testClock{now: time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)}

	// SQLite store — shares the test clock for consistent ListSchedulable queries.
	store, err := sqlite.NewStore(ctx, "file::memory:?cache=shared",
		sqlite.WithNowFunc(clock.Now))
	require.NoError(t, err)
	defer func() { _ = store.Close() }()
	idCounter := 0

	eng := flicker.NewEngine(store,
		flicker.WithWorkers(4),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
	)

	// Register workflow.
	factory := flicker.Define(
		"order-fulfillment",
		"v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[OrderRequest, Confirmation] {
			return &orderFulfillmentWorkflow{
				wc:         wc,
				httpClient: server.Client(),
				apiURL:     server.URL,
			}
		},
	).Register(eng)

	// Submit the order.
	instance, err := factory.Submit(ctx, OrderRequest{
		UserID:   "user-42",
		SKU:      "widget-x",
		Quantity: 5,
	})
	require.NoError(t, err)
	require.Equal(t, "wf-001", instance.ID())

	// --- Cycle 1: HTTP calls + parallel enrichment + validate → suspends at SleepUntil ---
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err := instance.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusSuspended, status,
		"should suspend at SleepUntil")

	// API calls: create-order, get-user, get-inventory (3 calls).
	require.Equal(t, int32(1), counts.createOrder.Load(), "create-order should be called once")
	require.Equal(t, int32(1), counts.getUser.Load(), "get-user should be called once")
	require.Equal(t, int32(1), counts.getInventory.Load(), "get-inventory should be called once")
	require.Equal(t, int32(0), counts.confirmOrder.Load(), "confirm-order should not be called yet")

	// --- Advance clock past SleepUntil, promote ---
	clock.Advance(2 * time.Hour)
	promoted, err := eng.Promote(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, promoted)

	// --- Cycle 2: replays cached steps → suspends at WaitForEvent ---
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err = instance.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusSuspended, status,
		"should suspend at WaitForEvent")

	// No new API calls — all previous steps replayed from cache.
	require.Equal(t, int32(1), counts.createOrder.Load(), "create-order should NOT be called again")
	require.Equal(t, int32(1), counts.getUser.Load(), "get-user should NOT be called again")
	require.Equal(
		t,
		int32(1),
		counts.getInventory.Load(),
		"get-inventory should NOT be called again",
	)
	require.Equal(t, int32(0), counts.confirmOrder.Load(), "confirm-order should not be called yet")

	// --- Deliver payment event ---
	err = eng.SendEvent(ctx, "payment:order-001", PaymentConfirmation{
		OrderID:       "order-001",
		TransactionID: "txn-999",
		Amount:        4999,
	})
	require.NoError(t, err)

	// --- Cycle 3: replays cached steps → receives payment → confirms → completed ---
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err = instance.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusCompleted, status,
		"workflow should be completed")

	// Confirm-order called exactly once. No other new calls.
	require.Equal(t, int32(1), counts.createOrder.Load(), "create-order total calls")
	require.Equal(t, int32(1), counts.getUser.Load(), "get-user total calls")
	require.Equal(t, int32(1), counts.getInventory.Load(), "get-inventory total calls")
	require.Equal(t, int32(1), counts.confirmOrder.Load(), "confirm-order total calls")

	// --- Golden snapshot of final workflow state + all cached step results ---
	g := newGoldie(t)
	snap := buildSnapshot(t, ctx, store, "wf-001")
	assertGolden(t, g, snap)
}
