package test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/flamingoosesoftwareinc/flicker/engine"
	"github.com/flamingoosesoftwareinc/flicker/engine/sqlite"
	"github.com/flamingoosesoftwareinc/flicker/internal/generate"
	greeting "github.com/flamingoosesoftwareinc/flicker/workflows/greeting/v1"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

func TestGreetingWorkflow_HappyPath(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)

	defer func() { _ = store.Close() }()

	fake := generate.NewFake("wf", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	eng := engine.NewEngine(store,
		engine.WithWorkers(1),
		engine.WithIDFunc(fake.NewID),
		engine.WithNowFunc(fake.Now),
	)
	greetings := greeting.Definition.Register(eng)

	wf, err := greetings.Submit(ctx, greeting.Request{UserID: "user-42"})
	require.NoError(t, err)

	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err := wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, engine.StatusCompleted, status)

	snapshot := buildSnapshot(t, ctx, store, wf.ID())

	g := newGoldie(t)
	assertGolden(t, g, snapshot)
}

func TestGreetingWorkflow_StepCaching(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)

	defer func() { _ = store.Close() }()

	fake := generate.NewFake("wf", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	eng := engine.NewEngine(store,
		engine.WithWorkers(1),
		engine.WithIDFunc(fake.NewID),
		engine.WithNowFunc(fake.Now),
	)
	greetings := greeting.Definition.Register(eng)

	wf, err := greetings.Submit(ctx, greeting.Request{UserID: "user-42"})
	require.NoError(t, err)

	// First run — executes both steps, caches results.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	// Force back to pending to simulate retry.
	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)

	err = store.UpdateStatus(ctx, wf.ID(), engine.StatusPending, record.OCCVersion)
	require.NoError(t, err)

	// Second run — steps return cached results.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err := wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, engine.StatusCompleted, status)

	snapshot := buildSnapshot(t, ctx, store, wf.ID())

	g := newGoldie(t)
	assertGolden(t, g, snapshot)
}

// --- Helpers ---

type workflowSnapshot struct {
	Workflow workflowState `json:"workflow"`
	Steps    []stepState   `json:"steps"`
}

type workflowState struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Status   engine.Status `json:"status"`
	Error    string        `json:"error,omitempty"`
	Attempts int           `json:"attempts"`
}

type stepState struct {
	StepName string          `json:"step_name"`
	Result   json.RawMessage `json:"result,omitempty"`
	Error    string          `json:"error,omitempty"`
}

func buildSnapshot(
	t *testing.T,
	ctx context.Context,
	store *sqlite.Store,
	workflowID string,
) []byte {
	t.Helper()

	record, err := store.Get(ctx, workflowID)
	require.NoError(t, err)

	steps, err := store.ListStepResults(ctx, workflowID)
	require.NoError(t, err)

	snap := workflowSnapshot{
		Workflow: workflowState{
			ID:       record.ID,
			Type:     record.Type,
			Status:   record.Status,
			Error:    record.Error,
			Attempts: record.Attempts,
		},
	}

	for _, s := range steps {
		snap.Steps = append(snap.Steps, stepState{
			StepName: s.StepName,
			Result:   s.Result,
			Error:    s.Error,
		})
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	require.NoError(t, err)

	return data
}

func newGoldie(t *testing.T) *goldie.Goldie {
	t.Helper()

	return goldie.New(t,
		goldie.WithFixtureDir("testdata"),
		goldie.WithNameSuffix(".golden"),
	)
}

func assertGolden(t *testing.T, g *goldie.Goldie, data []byte) {
	t.Helper()

	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, g.Update(t, t.Name(), data))
	}

	g.Assert(t, t.Name(), data)
}
