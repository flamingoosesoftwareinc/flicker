package test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
	"github.com/flamingoosesoftwareinc/flicker/sqlite"
	order "github.com/flamingoosesoftwareinc/flicker/workflows/order/v1"
	"github.com/stretchr/testify/require"
)

func TestOrderWorkflow_SuspendResume(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)

	defer func() { _ = store.Close() }()

	// Controllable clock.
	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	// Track HTTP call counts per endpoint.
	calls := &callCounter{}

	reservationExpiry := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC) // 1 hour from start

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

	// In-memory repository.
	repo := &memRepo{}

	// Manual trigger for test-controlled promotion.
	trigger := flicker.NewManualTimeTrigger()
	trigger.SetDeps(flicker.TriggerDeps{
		Store:  store,
		NowFn:  clock.Now,
		Logger: slog.Default(),
	})

	idCounter := 0
	eng := flicker.NewEngine(store,
		flicker.WithWorkers(1),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
	)

	def := order.NewDefinition(srv.Client(), srv.URL, repo)
	orders := def.Register(eng)

	wf, err := orders.Submit(ctx, order.Request{
		OrderID:    "ord-1",
		CustomerID: "cust-1",
		Amount:     99.99,
	})
	require.NoError(t, err)

	// --- First RunOnce: executes steps 1-2, hits SleepUntil, suspends ---
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err := wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusSuspended, status, "should be suspended after SleepUntil")

	// Steps 1-2 executed, steps 3-4 did not.
	require.Equal(t, 1, calls.Get("/users/cust-1"), "fetch_user should have been called once")
	require.Equal(
		t,
		1,
		calls.Get("/inventory/reserve"),
		"reserve_inventory should have been called once",
	)
	require.Equal(
		t,
		0,
		calls.Get("/payments/ord-1"),
		"check_payment should not have been called yet",
	)

	// --- Advance clock past the reservation expiry ---
	clock.Advance(2 * time.Hour)

	// Promote suspended → pending via manual trigger.
	promoted, err := trigger.Promote(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, promoted, "should promote exactly 1 workflow")

	status, err = wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusPending, status, "should be pending after promotion")

	// --- Second RunOnce: resumes, steps 1-2 cache hit, executes steps 3-4 ---
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err = wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusCompleted, status, "should be completed after resume")

	// Steps 1-2 were NOT re-executed (cache hit).
	require.Equal(t, 1, calls.Get("/users/cust-1"), "fetch_user should still be 1 (cached)")
	require.Equal(
		t,
		1,
		calls.Get("/inventory/reserve"),
		"reserve_inventory should still be 1 (cached)",
	)
	// Steps 3-4 executed.
	require.Equal(t, 1, calls.Get("/payments/ord-1"), "check_payment should have been called once")

	// Verify the repository got the result.
	require.NotNil(t, repo.Last())
	require.Equal(t, "confirmed", repo.Last().Status)
	require.Equal(t, "pay-ref-001", repo.Last().PaymentRef)

	// Golden snapshot.
	snapshot := buildSnapshot(t, ctx, store, wf.ID())

	g := newGoldie(t)
	assertGolden(t, g, snapshot)
}

// --- Test helpers ---

// testClock provides a controllable clock for tests.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = c.now.Add(d)
}

// callCounter tracks HTTP call counts by path.
type callCounter struct {
	mu    sync.Mutex
	calls map[string]int
}

func (c *callCounter) Inc(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.calls == nil {
		c.calls = make(map[string]int)
	}

	c.calls[path]++
}

func (c *callCounter) Get(path string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.calls[path]
}

// memRepo is a simple in-memory Repository for tests.
type memRepo struct {
	mu     sync.Mutex
	orders []*order.OrderResult
}

func (r *memRepo) SaveOrder(_ context.Context, result *order.OrderResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.orders = append(r.orders, result)

	return nil
}

func (r *memRepo) Last() *order.OrderResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.orders) == 0 {
		return nil
	}

	return r.orders[len(r.orders)-1]
}
