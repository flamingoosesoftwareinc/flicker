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
	fn func(context.Context) (*T, error),
) (*T, error) {
	// Duplicate step name detection.
	if _, seen := wc.seenSteps[stepName]; seen {
		return nil, fmt.Errorf(
			"duplicate step name %q: each step must have a unique name",
			stepName,
		)
	}

	wc.seenSteps[stepName] = struct{}{}

	// Read-through: check cache.
	cached, err := wc.store.GetStepResult(ctx, wc.wfType, wc.version, wc.id, stepName)
	if err == nil && cached != nil {
		var dest T
		jerr := json.Unmarshal(cached.Result, &dest)
		if jerr != nil {
			return nil, jerr
		}
		return &dest, nil
	}

	// Cache miss — execute.
	result, fnErr := fn(ctx)
	if fnErr != nil {
		return nil, fnErr
	}

	// Write-through: cache successful result.
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal step %q result: %w", stepName, err)
	}

	if err := wc.store.SaveStepResult(ctx, &StepResult{
		Type:       wc.wfType,
		Version:    wc.version,
		WorkflowID: wc.id,
		StepName:   stepName,
		Result:     data,
		CreatedAt:  wc.nowFn(),
	}); err != nil {
		return nil, fmt.Errorf("save step %q result: %w", stepName, err)
	}

	return result, nil
}
