package test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
	"github.com/flamingoosesoftwareinc/flicker/sqlite"
	"github.com/stretchr/testify/require"
)

func TestShutdown_InFlightWorkflowCompletes(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, "file::memory:?cache=shared")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	idCounter := 0
	var stepExecuted atomic.Bool
	stepStarted := make(chan struct{})

	eng := flicker.NewEngine(store,
		flicker.WithWorkers(2),
		flicker.WithPollInterval(10*time.Millisecond),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
		flicker.WithDrainTimeout(5*time.Second),
	)

	// A workflow that signals when it starts and takes a bit to complete.
	slowDef := flicker.Define(
		"slow",
		"v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[struct{}] {
			return &slowWorkflow{wc: wc, executed: &stepExecuted, started: stepStarted}
		},
	)
	factory := slowDef.Register(eng)

	_, err = factory.Submit(ctx, struct{}{})
	require.NoError(t, err)

	engCtx, engCancel := context.WithCancel(ctx)
	defer engCancel()

	engDone := make(chan error, 1)
	go func() {
		engDone <- eng.Start(engCtx)
	}()

	// Wait for the step function to start executing.
	select {
	case <-stepStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("workflow step did not start in time")
	}

	// Cancel the engine context while the workflow is in-flight.
	engCancel()

	// The engine should wait for in-flight workflows to finish.
	select {
	case err := <-engDone:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("engine did not shut down in time")
	}

	// The step should have completed — the drain context stays alive after
	// engine context cancellation, giving in-flight work time to finish.
	require.True(t, stepExecuted.Load(), "in-flight workflow step should have completed")
}

func TestShutdown_DrainTimeoutForceKillsInFlight(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, "file::memory:?cache=shared")
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	clock := &testClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	idCounter := 0
	stepStarted := make(chan struct{})
	var ctxWasCancelled atomic.Bool

	eng := flicker.NewEngine(store,
		flicker.WithWorkers(2),
		flicker.WithPollInterval(10*time.Millisecond),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(clock.Now),
		// Very short drain timeout — force-cancels the long-running step.
		flicker.WithDrainTimeout(200*time.Millisecond),
	)

	factory := flicker.Define(
		"hanging",
		"v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[struct{}] {
			return &hangingWorkflow{
				wc:              wc,
				started:         stepStarted,
				ctxWasCancelled: &ctxWasCancelled,
			}
		},
	).Register(eng)

	_, err = factory.Submit(ctx, struct{}{})
	require.NoError(t, err)

	engCtx, engCancel := context.WithCancel(ctx)
	defer engCancel()

	engDone := make(chan error, 1)
	go func() {
		engDone <- eng.Start(engCtx)
	}()

	// Wait for the step to start.
	select {
	case <-stepStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("workflow step did not start in time")
	}

	// Cancel the engine — starts the drain period.
	engCancel()

	// Engine should finish within drain timeout + some buffer.
	select {
	case err := <-engDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("engine did not shut down in time")
	}

	// The drain timeout should have force-cancelled the step's context.
	require.True(t, ctxWasCancelled.Load(),
		"drain timeout should have force-cancelled in-flight workflow context")
}

type slowWorkflow struct {
	wc       *flicker.WorkflowContext
	executed *atomic.Bool
	started  chan struct{}
}

func (w *slowWorkflow) Execute(ctx context.Context, _ struct{}) error {
	_, err := flicker.Run(ctx, w.wc, "slow_step", func(_ context.Context) (*string, error) {
		close(w.started)
		time.Sleep(200 * time.Millisecond)
		w.executed.Store(true)
		return flicker.Val("completed"), nil
	})

	return err
}

// hangingWorkflow simulates a step that blocks on the context (like an HTTP
// call waiting for a response). It only returns when the context is cancelled.
type hangingWorkflow struct {
	wc              *flicker.WorkflowContext
	started         chan struct{}
	ctxWasCancelled *atomic.Bool
}

func (w *hangingWorkflow) Execute(ctx context.Context, _ struct{}) error {
	_, err := flicker.Run(ctx, w.wc, "hanging_step", func(ctx context.Context) (*string, error) {
		close(w.started)
		// Block until force-cancelled.
		<-ctx.Done()
		w.ctxWasCancelled.Store(true)
		return nil, ctx.Err()
	})

	return err
}
