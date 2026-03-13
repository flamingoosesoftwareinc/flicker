package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
	"github.com/flamingoosesoftwareinc/flicker/sqlite"
	order "github.com/flamingoosesoftwareinc/flicker/workflows/order/v1"
	"github.com/stretchr/testify/require"
)

func TestEngine_StartWithPoolAndTrigger(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, "file::memory:?cache=shared")
	require.NoError(t, err)

	defer func() { _ = store.Close() }()

	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	calls := &callCounter{}

	reservationExpiry := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Inc(r.URL.Path)

		switch {
		case r.URL.Path == "/users/cust-1" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(order.User{
				ID: "cust-1", Name: "Alice", Email: "alice@example.com",
			})
		case r.URL.Path == "/inventory/reserve" && r.Method == "POST":
			_ = json.NewEncoder(w).Encode(order.Reservation{
				ID:        "res-001",
				ExpiresAt: reservationExpiry,
			})
		case r.URL.Path == "/payments/ord-1" && r.Method == "GET":
			_ = json.NewEncoder(w).Encode(order.PaymentResult{
				Status: "approved", Reference: "pay-ref-001",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	repo := &memRepo{}

	idCounter := 0
	eng := flicker.NewEngine(store,
		flicker.WithWorkers(2),
		flicker.WithPollInterval(10*time.Millisecond),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
		flicker.WithTimePromoter(&flicker.PollingTimePromoter{
			Interval: 10 * time.Millisecond,
			NowFunc:  clock.Now,
		}),
	)

	def := order.NewDefinition(srv.Client(), srv.URL, repo)
	orders := def.Register(eng)

	// Start the engine in a goroutine — it blocks until ctx is cancelled.
	engCtx, engCancel := context.WithCancel(ctx)
	defer engCancel()

	engDone := make(chan error, 1)

	go func() {
		engDone <- eng.Start(engCtx)
	}()

	// Submit a workflow — the scheduler should pick it up.
	wf, err := orders.Submit(ctx, order.Request{
		OrderID:    "ord-1",
		CustomerID: "cust-1",
		Amount:     99.99,
	})
	require.NoError(t, err)

	// Wait for the workflow to suspend (steps 1-2 execute, SleepUntil fires).
	awaitStatus(t, ctx, wf, flicker.StatusSuspended, 2*time.Second)

	// Verify steps 1-2 ran, step 3 did not.
	require.Equal(t, 1, calls.Get("/users/cust-1"))
	require.Equal(t, 1, calls.Get("/inventory/reserve"))
	require.Equal(t, 0, calls.Get("/payments/ord-1"))

	// Advance clock past reservation expiry — TimeTrigger will promote.
	clock.Advance(2 * time.Hour)

	// Wait for the workflow to complete (trigger promotes, scheduler re-executes).
	awaitStatus(t, ctx, wf, flicker.StatusCompleted, 2*time.Second)

	// Steps 1-2 cached, steps 3-4 executed.
	require.Equal(t, 1, calls.Get("/users/cust-1"), "fetch_user should still be 1 (cached)")
	require.Equal(
		t,
		1,
		calls.Get("/inventory/reserve"),
		"reserve_inventory should still be 1 (cached)",
	)
	require.Equal(t, 1, calls.Get("/payments/ord-1"), "check_payment should have been called once")

	require.NotNil(t, repo.Last())
	require.Equal(t, "confirmed", repo.Last().Status)

	// Shut down the engine.
	engCancel()

	select {
	case err := <-engDone:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("engine did not shut down in time")
	}

	// Golden snapshot.
	snapshot := buildSnapshot(t, ctx, store, wf.ID())

	g := newGoldie(t)
	assertGolden(t, g, snapshot)
}

// awaitStatus polls for a workflow status, failing if the timeout is reached.
func awaitStatus(
	t *testing.T,
	ctx context.Context,
	wf *flicker.Instance,
	want flicker.Status,
	timeout time.Duration,
) {
	t.Helper()

	deadline := time.After(timeout)

	for {
		select {
		case <-deadline:
			status, _ := wf.Status(ctx)
			t.Fatalf("timed out waiting for status %q, last saw %q", want, status)
		default:
			status, err := wf.Status(ctx)
			require.NoError(t, err)

			if status == want {
				return
			}

			time.Sleep(5 * time.Millisecond)
		}
	}
}
