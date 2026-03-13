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

// --- Panicking workflows ---

type panicStepWorkflow struct {
	wc *flicker.WorkflowContext
}

func (w *panicStepWorkflow) Execute(ctx context.Context, _ struct{}) error {
	_, err := flicker.Run(ctx, w.wc, "panic_step", func(ctx context.Context) (*string, error) {
		panic("step function exploded")
	})

	return err
}

var panicStepDef = flicker.Define(
	"panic_step",
	"v1",
	func(wc *flicker.WorkflowContext) flicker.Workflow[struct{}] {
		return &panicStepWorkflow{wc: wc}
	},
)

type panicExecuteWorkflow struct {
	wc *flicker.WorkflowContext
}

func (w *panicExecuteWorkflow) Execute(_ context.Context, _ struct{}) error {
	panic("execute exploded")
}

var panicExecuteDef = flicker.Define(
	"panic_execute",
	"v1",
	func(wc *flicker.WorkflowContext) flicker.Workflow[struct{}] {
		return &panicExecuteWorkflow{wc: wc}
	},
)

// --- Tests ---

func TestPanic_StepFunction(t *testing.T) {
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

	// Register with max attempts 1 so it fails immediately.
	factory := panicStepDef.Register(eng, flicker.RetryPolicy{
		MaxAttempts: 1,
		BaseDelay:   time.Second,
		MaxDelay:    time.Second,
	})

	wf, err := factory.Submit(ctx, struct{}{})
	require.NoError(t, err)

	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)
	require.Equal(t, flicker.StatusFailed, record.Status)
	require.Contains(t, record.Error, "panic recovered")
	require.Contains(t, record.Error, "step function exploded")
	// Should contain stack trace.
	require.Contains(t, record.Error, "goroutine")
}

func TestPanic_ExecuteFunction(t *testing.T) {
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

	factory := panicExecuteDef.Register(eng, flicker.RetryPolicy{
		MaxAttempts: 1,
		BaseDelay:   time.Second,
		MaxDelay:    time.Second,
	})

	wf, err := factory.Submit(ctx, struct{}{})
	require.NoError(t, err)

	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)
	require.Equal(t, flicker.StatusFailed, record.Status)
	require.Contains(t, record.Error, "panic recovered")
	require.Contains(t, record.Error, "execute exploded")
	require.Contains(t, record.Error, "goroutine")
}

func TestPanic_StepIsRetryable(t *testing.T) {
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

	// With 3 max attempts, a panic on first attempt should be retryable.
	factory := panicStepDef.Register(eng, flicker.RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    time.Second,
	})

	wf, err := factory.Submit(ctx, struct{}{})
	require.NoError(t, err)

	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)
	require.Equal(t, flicker.StatusPending, record.Status, "panic should be retryable")
	require.Equal(t, 1, record.Attempts)
}
