package test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flamingoosesoftwareinc/flicker"
	"github.com/flamingoosesoftwareinc/flicker/sqlite"
	greeting "github.com/flamingoosesoftwareinc/flicker/workflows/greeting/v1"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

func TestGreetingWorkflow_HappyPath(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)

	defer func() { _ = store.Close() }()

	idCounter := 0
	eng := flicker.NewEngine(store,
		flicker.WithWorkers(1),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(fixedTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))),
	)
	greetings := greeting.Definition.Register(eng)

	wf, err := greetings.Submit(ctx, greeting.Request{UserID: "user-42"})
	require.NoError(t, err)

	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err := wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusCompleted, status)

	// Verify version was persisted.
	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)
	require.Equal(t, "v1", record.Version)

	snapshot := buildSnapshot(t, ctx, store, wf.ID())

	g := newGoldie(t)
	assertGolden(t, g, snapshot)
}

func TestGreetingWorkflow_StepCaching(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)

	defer func() { _ = store.Close() }()

	// Incrementing time — if steps re-execute on retry, the timestamp changes.
	var timeCall atomic.Int64

	idCounter := 0
	eng := flicker.NewEngine(store,
		flicker.WithWorkers(1),
		flicker.WithIDFunc(func() string {
			idCounter++
			return fmt.Sprintf("wf-%03d", idCounter)
		}),
		flicker.WithNowFunc(func() time.Time {
			n := timeCall.Add(1)
			return time.Date(2026, 1, 1, int(n), 0, 0, 0, time.UTC)
		}),
	)
	greetings := greeting.Definition.Register(eng)

	wf, err := greetings.Submit(ctx, greeting.Request{UserID: "user-42"})
	require.NoError(t, err)

	// First run — executes all steps, caches results.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	// Capture step results after first run.
	stepsAfterFirst, err := store.ListStepResults(ctx, "greeting:v1", "v1", wf.ID())
	require.NoError(t, err)

	// Force back to pending to simulate retry.
	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)

	err = store.UpdateStatus(ctx, wf.ID(), flicker.StatusPending, nil, record.OCCVersion)
	require.NoError(t, err)

	// Second run — steps should return cached results despite time advancing.
	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	status, err := wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusCompleted, status)

	// Step results must be identical — proves caching, not re-execution.
	stepsAfterSecond, err := store.ListStepResults(ctx, "greeting:v1", "v1", wf.ID())
	require.NoError(t, err)
	require.Equal(t, len(stepsAfterFirst), len(stepsAfterSecond))

	for i := range stepsAfterFirst {
		require.Equal(t, stepsAfterFirst[i].StepName, stepsAfterSecond[i].StepName)
		require.Equal(t, string(stepsAfterFirst[i].Result), string(stepsAfterSecond[i].Result),
			"step %q result changed on replay — caching broken", stepsAfterFirst[i].StepName)
	}

	snapshot := buildSnapshot(t, ctx, store, wf.ID())

	g := newGoldie(t)
	assertGolden(t, g, snapshot)
}

// --- Helpers ---

func fixedTime(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

type workflowSnapshot struct {
	Workflow workflowState `json:"workflow"`
	Steps    []stepState   `json:"steps"`
}

type workflowState struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Version  string         `json:"version"`
	Status   flicker.Status `json:"status"`
	Error    string         `json:"error,omitempty"`
	Attempts int            `json:"attempts"`
}

type stepState struct {
	StepName  string          `json:"step_name"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
	CreatedAt string          `json:"created_at"`
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

	steps, err := store.ListStepResults(ctx, record.Type, record.Version, workflowID)
	require.NoError(t, err)

	snap := workflowSnapshot{
		Workflow: workflowState{
			ID:       record.ID,
			Type:     record.Type,
			Version:  record.Version,
			Status:   record.Status,
			Error:    record.Error,
			Attempts: record.Attempts,
		},
	}

	for _, s := range steps {
		snap.Steps = append(snap.Steps, stepState{
			StepName:  s.StepName,
			Result:    s.Result,
			Error:     s.Error,
			CreatedAt: s.CreatedAt.UTC().Format(time.RFC3339Nano),
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
