package flicker

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/trace"
)

// Branch is a named parallel execution path within a workflow. The name is
// used as the scope prefix for all step names within the branch, ensuring
// deterministic replay regardless of goroutine scheduling order.
//
// Construct via NewBranch — the zero value is not usable.
type Branch struct {
	name string
	run  func(ctx context.Context, wc *WorkflowContext) error
}

// NewBranch creates a named parallel branch. Panics if name is empty or run is nil
// — these are programmer errors that must be caught at init time.
func NewBranch(name string, run func(ctx context.Context, wc *WorkflowContext) error) Branch {
	if name == "" {
		panic("flicker: branch name must not be empty")
	}
	if run == nil {
		panic("flicker: branch run function must not be nil")
	}
	return Branch{name: name, run: run}
}

// Parallel executes branches concurrently, each in its own named scope.
// All branches run to completion before Parallel returns.
//
// Error semantics:
//   - If any branch returns a non-suspend error, the first such error is returned.
//   - If all errors are SuspendErrors, the one with the latest ResumeAt is returned
//     (so the workflow resumes after all branches can proceed).
//   - If all branches succeed, returns nil.
func Parallel(ctx context.Context, wc *WorkflowContext, branches ...Branch) error {
	errs := make([]error, len(branches))

	var wg sync.WaitGroup

	for i, b := range branches {
		wg.Add(1)

		go func(idx int, branch Branch) {
			defer wg.Done()

			branchCtx := ctx
			var branchSpan trace.Span
			if wc.tel != nil {
				branchCtx, branchSpan = wc.tel.startBranchSpan(ctx, branch.name)
			}

			scope := wc.Scope(branch.name)
			errs[idx] = panicToError(func() error {
				return branch.run(branchCtx, scope)
			})

			if branchSpan != nil {
				if branchErr := errs[idx]; branchErr != nil {
					if _, ok := IsSuspend(branchErr); !ok {
						wc.tel.endSpanWithError(branchSpan, branchErr)
						return
					}
				}
				branchSpan.End()
			}
		}(i, b)
	}

	wg.Wait()

	var latestSuspend *SuspendError
	var firstErr error

	for _, err := range errs {
		if err == nil {
			continue
		}

		if se, ok := IsSuspend(err); ok {
			if latestSuspend == nil || se.ResumeAt.After(latestSuspend.ResumeAt) {
				latestSuspend = se
			}
		} else if firstErr == nil {
			firstErr = err
		}
	}

	// Non-suspend errors take priority — trigger retry.
	if firstErr != nil {
		return firstErr
	}

	// If any branch suspended, propagate the latest resume time.
	if latestSuspend != nil {
		return latestSuspend
	}

	return nil
}
