package flicker

import (
	"context"
	"encoding/json"
	"errors"
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

	// Check cancellation signal before each step.
	if wc.store != nil {
		signal, sigErr := wc.store.GetSignal(ctx, wc.id)
		if sigErr != nil {
			return nil, fmt.Errorf("check signal for step %q: %w", stepName, sigErr)
		}
		if signal == SignalCancelRequested {
			return nil, ErrCancelled
		}
	}

	// Read-through: check prefetched cache first, then fall through to store.
	var cached *StepResult
	if wc.stepCache != nil {
		cached = wc.stepCache[stepName]
	}
	if cached == nil {
		var err error
		cached, err = wc.store.GetStepResult(ctx, wc.wfType, wc.version, wc.id, stepName)
		if err != nil && !errors.Is(err, ErrStepNotFound) {
			return nil, fmt.Errorf("get step %q result: %w", stepName, err)
		}
	}
	if cached != nil {
		// Check for cached errors (e.g., event timeout markers).
		if cached.Error != "" {
			return nil, cachedStepError(cached.Error)
		}

		var dest T
		if jerr := json.Unmarshal(cached.Result, &dest); jerr != nil {
			return nil, fmt.Errorf("unmarshal step %q cached result: %w", stepName, jerr)
		}
		return &dest, nil
	}

	// Cache miss — execute with panic recovery.
	var result *T
	fnErr := panicToError(func() error {
		var innerErr error
		result, innerErr = fn(ctx)
		return innerErr
	})
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

// stepErrorTimeout is the sentinel string stored in StepResult.Error
// when an event subscription times out.
const stepErrorTimeout = "event_timeout"

// cachedStepError maps a stored error string back to a semantic error.
func cachedStepError(errStr string) error {
	switch errStr {
	case stepErrorTimeout:
		return ErrEventTimeout
	default:
		return fmt.Errorf("cached step error: %s", errStr)
	}
}
