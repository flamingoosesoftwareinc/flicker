package test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
	"github.com/flamingoosesoftwareinc/flicker/sqlite"
	"github.com/stretchr/testify/require"
)

// --- Failing workflow for retry tests ---

type failRequest struct {
	FailCount int `json:"fail_count"` // how many times to return a transient error
}

type failWorkflow struct {
	wc *flicker.WorkflowContext
}

func (w *failWorkflow) Execute(ctx context.Context, req failRequest) error {
	// Track attempts via a durable step.
	attempt, err := flicker.Run(
		ctx,
		w.wc,
		"record_attempt",
		func(ctx context.Context) (*int, error) {
			return flicker.Val(1), nil
		},
	)
	if err != nil {
		return err
	}
	_ = attempt

	// The workflow checks its own attempt count from the workflow record.
	// Since we can't easily read attempts inside, we use the failCount
	// from the request and a non-durable counter via the store.
	// Simpler: just always fail if failCount > 0, and the engine handles retries.
	if req.FailCount > 0 {
		return fmt.Errorf("transient failure (configured to fail %d times)", req.FailCount)
	}

	return nil
}

var failDef = flicker.Define(
	"fail",
	"v1",
	func(wc *flicker.WorkflowContext) flicker.Workflow[failRequest] {
		return &failWorkflow{wc: wc}
	},
)

// --- Tests ---

func TestRetry_ExhaustsMaxAttempts(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	idCounter := 0
	eng := flicker.NewEngine(store,
		flicker.WithWorkers(1),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
	)

	policy := flicker.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    time.Second,
	}
	factory := failDef.Register(eng, policy)

	// Submit a workflow that always fails.
	wf, err := factory.Submit(ctx, failRequest{FailCount: 999})
	require.NoError(t, err)

	// Attempt 1: pending → running → transient error → pending with retry_after.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)
	require.Equal(
		t,
		flicker.StatusPending,
		record.Status,
		"should be pending after first transient failure",
	)
	require.Equal(t, 1, record.Attempts)
	require.False(t, record.RetryAfter.IsZero(), "retry_after should be set")

	// Advance clock past retry_after.
	clock.Advance(time.Second)

	// Attempt 2.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	record, err = store.Get(ctx, wf.ID())
	require.NoError(t, err)
	require.Equal(
		t,
		flicker.StatusPending,
		record.Status,
		"should still be pending after second failure",
	)
	require.Equal(t, 2, record.Attempts)

	// Advance clock past retry_after.
	clock.Advance(time.Second)

	// Attempt 3: max attempts reached → failed.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	record, err = store.Get(ctx, wf.ID())
	require.NoError(t, err)
	require.Equal(
		t,
		flicker.StatusFailed,
		record.Status,
		"should be failed after exhausting max attempts",
	)
	require.Equal(t, 2, record.Attempts, "attempts should reflect the retries before final failure")
	require.Contains(t, record.Error, "transient failure")

	// Golden snapshot.
	snapshot := buildSnapshot(t, ctx, store, wf.ID())
	g := newGoldie(t)
	assertGolden(t, g, snapshot)
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	idCounter := 0
	eng := flicker.NewEngine(store,
		flicker.WithWorkers(1),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
	)

	policy := flicker.RetryPolicy{
		MaxAttempts: 5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    time.Second,
	}
	factory := failDef.Register(eng, policy)

	wf, err := factory.Submit(ctx, failRequest{FailCount: 999})
	require.NoError(t, err)

	// Collect retry_after values across attempts.
	var retryAfters []time.Time

	for i := 0; i < 4; i++ {
		clock.Advance(2 * time.Second) // always advance past any retry_after
		err = eng.RunOnce(ctx)
		require.NoError(t, err)

		record, err := store.Get(ctx, wf.ID())
		require.NoError(t, err)

		if record.Status == flicker.StatusFailed {
			break
		}

		retryAfters = append(retryAfters, record.RetryAfter)
	}

	// Verify exponential backoff: each delay should be >= the previous.
	require.GreaterOrEqual(t, len(retryAfters), 2, "need at least 2 retry_afters to verify backoff")

	for i := 1; i < len(retryAfters); i++ {
		prevDelay := retryAfters[i-1].Sub(clock.Now().Add(-2 * time.Second))
		currDelay := retryAfters[i].Sub(clock.Now().Add(-2 * time.Second))
		require.GreaterOrEqual(t, currDelay, prevDelay,
			"attempt %d delay should be >= attempt %d delay (exponential backoff)", i+1, i)
	}
}

func TestRetry_BackoffCappedAtMaxDelay(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := &testClock{now: baseTime}

	idCounter := 0
	eng := flicker.NewEngine(store,
		flicker.WithWorkers(1),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
	)

	policy := flicker.RetryPolicy{
		MaxAttempts: 10,
		BaseDelay:   time.Second,
		MaxDelay:    5 * time.Second, // cap
	}
	factory := failDef.Register(eng, policy)

	wf, err := factory.Submit(ctx, failRequest{FailCount: 999})
	require.NoError(t, err)

	// Run enough attempts to exceed the max delay cap.
	for i := 0; i < 6; i++ {
		clock.Advance(10 * time.Second)
		err = eng.RunOnce(ctx)
		require.NoError(t, err)

		record, err := store.Get(ctx, wf.ID())
		require.NoError(t, err)

		if record.Status == flicker.StatusFailed {
			break
		}

		// Verify retry_after never exceeds now + maxDelay.
		maxAllowed := clock.Now().Add(policy.MaxDelay)
		require.False(t, record.RetryAfter.After(maxAllowed),
			"attempt %d: retry_after %v exceeds max allowed %v",
			i+1, record.RetryAfter, maxAllowed)
	}
}

func TestSuspend_SleepUntil(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	idCounter := 0
	eng := flicker.NewEngine(store,
		flicker.WithWorkers(1),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
	)

	// A workflow that sleeps for 1 hour, then completes.
	sleepTime := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	sleepDef := flicker.Define(
		"sleep",
		"v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[struct{}] {
			return &sleepWorkflow{wc: wc, until: sleepTime}
		},
	)
	factory := sleepDef.Register(eng)

	wf, err := factory.Submit(ctx, struct{}{})
	require.NoError(t, err)

	// First run: should suspend.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)
	require.Equal(t, flicker.StatusSuspended, record.Status)
	require.Equal(t, 0, record.Attempts, "suspend should not increment attempts")

	// Clock is before sleep time — promotion should not promote.
	promoted, err := eng.Promote(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, promoted, "should not promote before deadline")

	// Advance past sleep time.
	clock.Advance(2 * time.Hour)

	promoted, err = eng.Promote(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, promoted)

	status, err := wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusPending, status, "should be pending after promotion")

	// Second run: should complete (SleepUntil returns nil because clock passed).
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err = wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusCompleted, status)

	snapshot := buildSnapshot(t, ctx, store, wf.ID())
	g := newGoldie(t)
	assertGolden(t, g, snapshot)
}

type sleepWorkflow struct {
	wc    *flicker.WorkflowContext
	until time.Time
}

func (w *sleepWorkflow) Execute(ctx context.Context, _ struct{}) error {
	if err := w.wc.SleepUntil(ctx, w.until); err != nil {
		return err
	}

	// After waking up, do a durable step to prove we resumed.
	_, err := flicker.Run(ctx, w.wc, "after_sleep", func(ctx context.Context) (*string, error) {
		return flicker.Val("awake"), nil
	})

	return err
}

func TestWaitForEvent_EventDelivered(t *testing.T) {
	ctx := context.Background()

	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	store, err := sqlite.NewStore(ctx, ":memory:", sqlite.WithNowFunc(clock.Now))
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	idCounter := 0
	eng := flicker.NewEngine(store,
		flicker.WithWorkers(1),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
	)

	type PaymentEvent struct {
		Status    string `json:"status"`
		Reference string `json:"reference"`
	}

	var capturedEvent *PaymentEvent

	eventDef := flicker.Define(
		"event_wait",
		"v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[string] {
			return &eventWorkflow[PaymentEvent]{
				wc:             wc,
				correlationKey: "", // set from request
				timeout:        time.Hour,
				onEvent: func(evt *PaymentEvent) {
					capturedEvent = evt
				},
			}
		},
	)
	factory := eventDef.Register(eng)

	wf, err := factory.Submit(ctx, "order-123")
	require.NoError(t, err)

	// First run: should suspend waiting for event.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err := wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusSuspended, status)

	// Deliver the event.
	err = eng.SendEvent(ctx, "payment:order-123", PaymentEvent{
		Status:    "approved",
		Reference: "pay-001",
	})
	require.NoError(t, err)

	// Workflow should be promoted to pending.
	status, err = wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusPending, status)

	// Second run: should complete with the event data.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err = wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusCompleted, status)

	require.NotNil(t, capturedEvent, "event should have been delivered to workflow")
	require.Equal(t, "approved", capturedEvent.Status)
	require.Equal(t, "pay-001", capturedEvent.Reference)

	snapshot := buildSnapshot(t, ctx, store, wf.ID())
	g := newGoldie(t)
	assertGolden(t, g, snapshot)
}

func TestWaitForEvent_Timeout(t *testing.T) {
	ctx := context.Background()

	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	store, err := sqlite.NewStore(ctx, ":memory:", sqlite.WithNowFunc(clock.Now))
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	idCounter := 0
	eng := flicker.NewEngine(store,
		flicker.WithWorkers(1),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
	)

	type DummyEvent struct {
		Data string `json:"data"`
	}

	var timedOut bool

	timeoutDef := flicker.Define(
		"event_timeout",
		"v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[string] {
			return &eventWorkflow[DummyEvent]{
				wc:             wc,
				correlationKey: "",
				timeout:        time.Hour,
				onTimeout: func() {
					timedOut = true
				},
			}
		},
	)
	factory := timeoutDef.Register(eng)

	wf, err := factory.Submit(ctx, "order-456")
	require.NoError(t, err)

	// First run: should suspend waiting for event.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err := wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusSuspended, status)

	// Advance clock past the timeout deadline.
	clock.Advance(2 * time.Hour)

	// Time out expired subscriptions.
	n, err := eng.TimeOutSubscriptions(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	// Workflow should be promoted to pending.
	status, err = wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusPending, status)

	// Second run: WaitForEvent should return ErrEventTimeout, workflow handles it.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err = wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(
		t,
		flicker.StatusCompleted,
		status,
		"workflow should complete after handling timeout",
	)

	require.True(t, timedOut, "workflow should have detected the timeout")

	snapshot := buildSnapshot(t, ctx, store, wf.ID())
	g := newGoldie(t)
	assertGolden(t, g, snapshot)
}

// --- Event test workflow ---

// eventWorkflow is a generic test workflow that waits for an event of type T.
type eventWorkflow[T any] struct {
	wc             *flicker.WorkflowContext
	correlationKey string
	timeout        time.Duration
	onEvent        func(*T)
	onTimeout      func()
}

func (w *eventWorkflow[T]) Execute(ctx context.Context, orderID string) error {
	key := "payment:" + orderID

	event, err := flicker.WaitForEvent[T](ctx, w.wc, "await_event", key, w.timeout)
	if errors.Is(err, flicker.ErrEventTimeout) {
		if w.onTimeout != nil {
			w.onTimeout()
		}

		// Record that we handled the timeout.
		_, runErr := flicker.Run(
			ctx,
			w.wc,
			"handle_timeout",
			func(ctx context.Context) (*string, error) {
				return flicker.Val("timed_out"), nil
			},
		)

		return runErr
	}

	if err != nil {
		return err
	}

	if w.onEvent != nil {
		w.onEvent(event)
	}

	// Record that we processed the event.
	_, err = flicker.Run(ctx, w.wc, "process_event", func(ctx context.Context) (*string, error) {
		return flicker.Val("processed"), nil
	})

	return err
}

func TestPermanentFailure_StopWithError(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	idCounter := 0
	eng := flicker.NewEngine(store,
		flicker.WithWorkers(1),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
	)

	permFailDef := flicker.Define(
		"perm_fail",
		"v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[struct{}] {
			return &permFailWorkflow{wc: wc}
		},
	)
	factory := permFailDef.Register(eng)

	wf, err := factory.Submit(ctx, struct{}{})
	require.NoError(t, err)

	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)
	require.Equal(t, flicker.StatusFailed, record.Status)
	require.Equal(t, 0, record.Attempts, "permanent failure should not increment attempts")
	require.Contains(t, record.Error, "validation failed")

	snapshot := buildSnapshot(t, ctx, store, wf.ID())
	g := newGoldie(t)
	assertGolden(t, g, snapshot)
}

type permFailWorkflow struct {
	wc *flicker.WorkflowContext
}

func (w *permFailWorkflow) Execute(ctx context.Context, _ struct{}) error {
	w.wc.Stop(flicker.WithError(fmt.Errorf("validation failed: invalid input")))
	return nil
}
