package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
	"github.com/flamingoosesoftwareinc/flicker/sqlite"
	"github.com/stretchr/testify/require"
)

// --- Cancellation workflow ---

type cancelWorkflow struct {
	wc      *flicker.WorkflowContext
	stepRan *bool
}

func (w *cancelWorkflow) Execute(ctx context.Context, _ struct{}) error {
	_, err := flicker.Run(ctx, w.wc, "step_one", func(ctx context.Context) (*string, error) {
		*w.stepRan = true
		return flicker.Val("done"), nil
	})

	return err
}

// --- Tests ---

func TestCancel_SignalBeforeExecution(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	idCounter := 0
	stepRan := false

	eng := flicker.NewEngine(store,
		flicker.WithWorkers(1),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
	)

	cancelDef := flicker.Define(
		"cancel_test",
		"v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[struct{}] {
			return &cancelWorkflow{wc: wc, stepRan: &stepRan}
		},
	)
	factory := cancelDef.Register(eng)

	wf, err := factory.Submit(ctx, struct{}{})
	require.NoError(t, err)

	// Set cancellation signal before running.
	err = store.SetSignal(ctx, wf.ID(), flicker.SignalCancelRequested)
	require.NoError(t, err)

	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)
	require.Equal(t, flicker.StatusCancelled, record.Status)
	require.False(t, stepRan, "step should not have executed after cancellation signal")

	snapshot := buildSnapshot(t, ctx, store, wf.ID())
	g := newGoldie(t)
	assertGolden(t, g, snapshot)
}

func TestCancel_SignalDuringExecution(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	idCounter := 0
	step2Ran := false

	eng := flicker.NewEngine(store,
		flicker.WithWorkers(1),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
	)

	// A workflow with two steps. We set the cancel signal after step 1
	// executes (during step 2's signal check).
	multiStepDef := flicker.Define(
		"cancel_multi",
		"v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[struct{}] {
			return &cancelMultiWorkflow{wc: wc, store: store, step2Ran: &step2Ran}
		},
	)
	factory := multiStepDef.Register(eng)

	wf, err := factory.Submit(ctx, struct{}{})
	require.NoError(t, err)

	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)

	// The workflow should be cancelled because step 2 checked the signal.
	// Note: since the cancel signal was set during execution (in step 1's fn),
	// the runner's pre-execution check didn't catch it, but Run[T] for step 2 did.
	require.Equal(t, flicker.StatusCancelled, record.Status)
	require.False(t, step2Ran, "step 2 should not execute after cancellation")
}

type cancelMultiWorkflow struct {
	wc       *flicker.WorkflowContext
	store    *sqlite.Store
	step2Ran *bool
}

func (w *cancelMultiWorkflow) Execute(ctx context.Context, _ struct{}) error {
	// Step 1 runs and sets the cancel signal as a side effect.
	_, err := flicker.Run(ctx, w.wc, "step_one", func(ctx context.Context) (*string, error) {
		// Set cancel signal mid-execution.
		_ = w.store.SetSignal(ctx, w.wc.WorkflowID(), flicker.SignalCancelRequested)
		return flicker.Val("step1_done"), nil
	})
	if err != nil {
		return err
	}

	// Step 2 should detect the signal and return ErrCancelled.
	_, err = flicker.Run(ctx, w.wc, "step_two", func(ctx context.Context) (*string, error) {
		*w.step2Ran = true
		return flicker.Val("step2_done"), nil
	})

	return err
}

func TestCancel_ErrCancelledSentinel(t *testing.T) {
	// Verify ErrCancelled is the specific error type.
	require.ErrorIs(t, flicker.ErrCancelled, flicker.ErrCancelled)
	require.Contains(t, flicker.ErrCancelled.Error(), "cancelled")
}

func TestCancel_NoRetryOnCancel(t *testing.T) {
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

	cancelDef := flicker.Define(
		"cancel_noretry",
		"v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[struct{}] {
			stepRan := false
			return &cancelWorkflow{wc: wc, stepRan: &stepRan}
		},
	)
	// Even with many retries allowed, cancellation should be terminal.
	factory := cancelDef.Register(eng, flicker.RetryPolicy{
		MaxAttempts: 10,
		BaseDelay:   time.Second,
		MaxDelay:    time.Second,
	})

	wf, err := factory.Submit(ctx, struct{}{})
	require.NoError(t, err)

	err = store.SetSignal(ctx, wf.ID(), flicker.SignalCancelRequested)
	require.NoError(t, err)

	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)
	require.Equal(t, flicker.StatusCancelled, record.Status,
		"cancellation should be terminal, no retry")
	require.Equal(t, 0, record.Attempts,
		"attempts should not increment on cancellation")
}
