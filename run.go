package flicker

import (
	"context"
	"encoding/json"
	"fmt"
)

// Run executes a named durable step. On first execution, fn runs and
// the result is cached. On replay (retry), the cached result is deserialized
// into dest and fn is skipped. T must be JSON-serializable.
func Run[T any](
	ctx context.Context,
	wc *WorkflowContext,
	stepName string,
	dest *T,
	fn func(context.Context) (T, error),
) error {
	// Read-through: check cache.
	cached, err := wc.store.GetStepResult(ctx, wc.id, stepName)
	if err == nil && cached != nil {
		return json.Unmarshal(cached.Result, dest)
	}

	// Cache miss — execute.
	result, fnErr := fn(ctx)
	if fnErr != nil {
		return fnErr
	}

	// Write-through: cache successful result.
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal step %q result: %w", stepName, err)
	}

	if err := wc.store.SaveStepResult(ctx, &StepResult{
		WorkflowID: wc.id,
		StepName:   stepName,
		Result:     data,
	}); err != nil {
		return fmt.Errorf("save step %q result: %w", stepName, err)
	}

	*dest = result

	return nil
}
