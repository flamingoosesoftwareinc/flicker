package test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
	"github.com/flamingoosesoftwareinc/flicker/sqlite"
	"github.com/stretchr/testify/require"
)

type hashRequest struct {
	Data string `json:"data"`
}

type hashWorkflow struct {
	*flicker.WorkflowContext
	hash *flicker.Provider[string]
}

func (w *hashWorkflow) Execute(ctx context.Context, req hashRequest) (struct{}, error) {
	var zero struct{}

	// Generate two durable hashes — deterministic on replay.
	first, err := w.hash.Get(ctx)
	if err != nil {
		return zero, err
	}

	second, err := w.hash.Get(ctx)
	if err != nil {
		return zero, err
	}

	_, err = flicker.Run(ctx, w.WorkflowContext, "use_hashes",
		func(_ context.Context) (*string, error) {
			return flicker.Val(first + ":" + second), nil
		},
	)

	return zero, err
}

func TestCustomProvider_SHA256(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)

	defer func() { _ = store.Close() }()

	// Counter to make each hash unique (simulates non-deterministic input).
	counter := 0

	def := flicker.Define[hashRequest, struct{}]("hash_test", "v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[hashRequest, struct{}] {
			return &hashWorkflow{
				WorkflowContext: wc,
				hash: flicker.NewProvider(wc, "sha256", func() (string, error) {
					counter++
					sum := sha256.Sum256([]byte(fmt.Sprintf("seed-%d", counter)))

					return hex.EncodeToString(sum[:]), nil
				}),
			}
		},
	)

	idCounter := 0
	eng := flicker.NewEngine(store,
		flicker.WithWorkers(1),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(fixedTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))),
	)

	factory := def.Register(eng)

	wf, err := factory.Submit(ctx, hashRequest{Data: "test"})
	require.NoError(t, err)

	// First run — generates and caches hashes.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err := wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusCompleted, status)

	stepsAfterFirst, err := store.ListStepResults(ctx, "hash_test:v1", "v1", wf.ID())
	require.NoError(t, err)

	// Force back to pending to simulate retry.
	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)

	err = store.UpdateStatus(ctx, wf.ID(), flicker.StatusPending, nil, record.OCCVersion)
	require.NoError(t, err)

	// Second run — counter keeps incrementing, but cached hashes are returned.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	stepsAfterSecond, err := store.ListStepResults(ctx, "hash_test:v1", "v1", wf.ID())
	require.NoError(t, err)

	for i := range stepsAfterFirst {
		require.Equal(t, stepsAfterFirst[i].StepName, stepsAfterSecond[i].StepName)
		require.Equal(
			t,
			string(stepsAfterFirst[i].Result),
			string(stepsAfterSecond[i].Result),
			"step %q result changed on replay — provider caching broken",
			stepsAfterFirst[i].StepName,
		)
	}

	snapshot := buildSnapshot(t, ctx, store, wf.ID())

	g := newGoldie(t)
	assertGolden(t, g, snapshot)
}
