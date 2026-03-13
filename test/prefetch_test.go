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

// countingStore wraps a real store and counts GetStepResult calls.
type countingStore struct {
	flicker.WorkflowStore
	getStepResultCalls atomic.Int64
}

func (s *countingStore) GetStepResult(
	ctx context.Context,
	wfType, version, workflowID, stepName string,
) (*flicker.StepResult, error) {
	s.getStepResultCalls.Add(1)
	return s.WorkflowStore.GetStepResult(ctx, wfType, version, workflowID, stepName)
}

func TestPrefetch_ReducesStoreCalls(t *testing.T) {
	ctx := context.Background()

	realStore, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)
	defer func() { _ = realStore.Close() }()

	store := &countingStore{WorkflowStore: realStore}

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

	// A workflow with 3 steps.
	multiDef := flicker.Define(
		"prefetch_multi",
		"v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[struct{}] {
			return &threeStepWorkflow{wc: wc}
		},
	)
	factory := multiDef.Register(eng)

	wf, err := factory.Submit(ctx, struct{}{})
	require.NoError(t, err)

	// First run: executes all steps (cache misses).
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err := wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusCompleted, status)

	// On first run, the prefetch loads 0 results (nothing cached yet).
	// Each step does a cache miss check via stepCache (nil) then falls through
	// to GetStepResult.
	firstRunCalls := store.getStepResultCalls.Load()

	// Force back to pending to simulate replay.
	record, err := realStore.Get(ctx, wf.ID())
	require.NoError(t, err)
	err = realStore.UpdateStatus(ctx, wf.ID(), flicker.StatusPending, record.OCCVersion)
	require.NoError(t, err)

	// Reset counter.
	store.getStepResultCalls.Store(0)

	// Second run: prefetch loads all 3 step results into the cache.
	// Run[T] should hit the in-memory cache, NOT call GetStepResult.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err = wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusCompleted, status)

	secondRunCalls := store.getStepResultCalls.Load()
	require.Equal(t, int64(0), secondRunCalls,
		"prefetch should eliminate all GetStepResult calls on replay; got %d", secondRunCalls)

	_ = firstRunCalls // logged but not strictly asserted
}

func TestPrefetch_CacheMissFallback(t *testing.T) {
	ctx := context.Background()

	realStore, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)
	defer func() { _ = realStore.Close() }()

	store := &countingStore{WorkflowStore: realStore}

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

	// A workflow where the first run saves step_one, suspends, and then
	// on resume a new step_two runs (not in prefetch cache).
	suspendResumeDef := flicker.Define(
		"prefetch_suspend",
		"v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[struct{}] {
			return &prefetchSuspendWorkflow{wc: wc, clock: clock}
		},
	)
	factory := suspendResumeDef.Register(eng)

	wf, err := factory.Submit(ctx, struct{}{})
	require.NoError(t, err)

	// First run: step_one runs, then SleepUntil suspends.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err := wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusSuspended, status)

	// Advance clock past the sleep time.
	clock.Advance(2 * time.Hour)

	promoted, err := eng.Promote(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, promoted)

	store.getStepResultCalls.Store(0)

	// Second run: prefetch loads step_one and _sleep.until:1.
	// step_one hits prefetch cache. step_two is new — cache miss falls through to store.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err = wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusCompleted, status)

	// step_two should have called GetStepResult once (cache miss, not in prefetch).
	// The prefetched steps (step_one, _sleep.until:1) should NOT trigger GetStepResult.
	calls := store.getStepResultCalls.Load()
	require.Equal(t, int64(1), calls,
		"only the new step should fall through to GetStepResult; got %d", calls)
}

// --- Test workflows ---

type threeStepWorkflow struct {
	wc *flicker.WorkflowContext
}

func (w *threeStepWorkflow) Execute(ctx context.Context, _ struct{}) error {
	_, err := flicker.Run(ctx, w.wc, "step_one", func(ctx context.Context) (*string, error) {
		return flicker.Val("one"), nil
	})
	if err != nil {
		return err
	}

	_, err = flicker.Run(ctx, w.wc, "step_two", func(ctx context.Context) (*string, error) {
		return flicker.Val("two"), nil
	})
	if err != nil {
		return err
	}

	_, err = flicker.Run(ctx, w.wc, "step_three", func(ctx context.Context) (*string, error) {
		return flicker.Val("three"), nil
	})

	return err
}

type prefetchSuspendWorkflow struct {
	wc    *flicker.WorkflowContext
	clock *testClock
}

func (w *prefetchSuspendWorkflow) Execute(ctx context.Context, _ struct{}) error {
	_, err := flicker.Run(ctx, w.wc, "step_one", func(ctx context.Context) (*string, error) {
		return flicker.Val("one"), nil
	})
	if err != nil {
		return err
	}

	sleepTime := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	if err := w.wc.SleepUntil(ctx, sleepTime); err != nil {
		return err
	}

	// This step only runs after resume — not in prefetch cache on second run.
	_, err = flicker.Run(ctx, w.wc, "step_two", func(ctx context.Context) (*string, error) {
		return flicker.Val("two"), nil
	})

	return err
}
