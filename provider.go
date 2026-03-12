package flicker

import (
	"context"
	"fmt"
)

// Provider is a durable source of non-deterministic values. Each call to
// Get produces a value via the generator function, caches it as a named
// step, and returns it. On replay, the cached value is returned without
// calling the generator. The step name is auto-incremented: "prefix:1",
// "prefix:2", etc.
//
// Use this to make any non-deterministic value (time, IDs, random numbers,
// external lookups) deterministic across retries.
type Provider[T any] struct {
	wc      *WorkflowContext
	prefix  string
	counter int
	gen     func() (T, error)
}

// NewProvider creates a Provider bound to a WorkflowContext. The prefix
// namespaces this provider's steps (e.g., "_time.now", "_id.new", "uuidv7").
// The gen function produces the non-deterministic value on first execution.
func NewProvider[T any](wc *WorkflowContext, prefix string, gen func() (T, error)) *Provider[T] {
	return &Provider[T]{
		wc:     wc,
		prefix: prefix,
		gen:    gen,
	}
}

// Get returns the next value from this provider. On first execution, gen()
// is called and the result cached. On replay, the cached value is returned.
func (p *Provider[T]) Get(ctx context.Context) (T, error) {
	p.counter++
	stepName := fmt.Sprintf("%s:%d", p.prefix, p.counter)

	result, err := Run(ctx, p.wc, stepName, func(_ context.Context) (*T, error) {
		v, err := p.gen()
		if err != nil {
			return nil, err
		}

		return &v, nil
	})
	if err != nil {
		var zero T
		return zero, err
	}

	return *result, nil
}
