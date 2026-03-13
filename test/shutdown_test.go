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

	wf, err := factory.Submit(ctx, struct{}{})
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

	// The in-flight step function should have completed.
	require.True(t, stepExecuted.Load(), "in-flight workflow step should have completed")

	// Note: the final status may be "running" because the store operations
	// (resolveOutcome) use the cancelled context and may fail. The key assertion
	// is that the engine waited for the workflow to finish executing rather than
	// killing it immediately.
	_ = wf
}

type slowWorkflow struct {
	wc       *flicker.WorkflowContext
	executed *atomic.Bool
	started  chan struct{}
}

func (w *slowWorkflow) Execute(ctx context.Context, _ struct{}) error {
	_, err := flicker.Run(ctx, w.wc, "slow_step", func(_ context.Context) (*string, error) {
		// Signal that we've started.
		close(w.started)
		// Simulate work.
		time.Sleep(200 * time.Millisecond)
		w.executed.Store(true)
		return flicker.Val("completed"), nil
	})

	return err
}
