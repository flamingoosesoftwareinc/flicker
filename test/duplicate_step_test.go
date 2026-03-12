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

type duplicateStepRequest struct {
	Value string `json:"value"`
}

type duplicateStepWorkflow struct {
	*flicker.WorkflowContext
}

func (w *duplicateStepWorkflow) Execute(ctx context.Context, req duplicateStepRequest) error {
	_, err := flicker.Run(ctx, w.WorkflowContext, "same_name",
		func(_ context.Context) (*string, error) {
			return flicker.Val("first"), nil
		},
	)
	if err != nil {
		return err
	}

	_, err = flicker.Run(ctx, w.WorkflowContext, "same_name",
		func(_ context.Context) (*string, error) {
			return flicker.Val("second"), nil
		},
	)

	return err
}

func TestDuplicateStepName_Errors(t *testing.T) {
	ctx := context.Background()

	store, err := sqlite.NewStore(ctx, ":memory:")
	require.NoError(t, err)

	defer func() { _ = store.Close() }()

	def := flicker.Define[duplicateStepRequest]("duplicate_test", "v1",
		func(wc *flicker.WorkflowContext) flicker.Workflow[duplicateStepRequest] {
			return &duplicateStepWorkflow{WorkflowContext: wc}
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

	factory := def.Register(eng, flicker.RetryPolicy{MaxAttempts: 1})

	wf, err := factory.Submit(ctx, duplicateStepRequest{Value: "test"})
	require.NoError(t, err)

	err = eng.RunOnce(ctx)
	require.NoError(t, err)

	// Workflow should have failed due to duplicate step name.
	status, err := wf.Status(ctx)
	require.NoError(t, err)
	require.Equal(t, flicker.StatusFailed, status)

	record, err := store.Get(ctx, wf.ID())
	require.NoError(t, err)
	require.Contains(t, record.Error, "duplicate step name")
	require.Contains(t, record.Error, "same_name")
}
