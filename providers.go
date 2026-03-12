package flicker

import (
	"context"
	"fmt"
	"time"
)

// TimeProvider is a durable service for time operations.
// Each call to Now caches the result as a step, making it deterministic on replay.
type TimeProvider struct {
	wc      *WorkflowContext
	counter int
	nowFn   func() time.Time
}

// Now returns the current time. On first execution, the real time is captured
// and cached. On replay, the cached value is returned.
func (tp *TimeProvider) Now(ctx context.Context) (time.Time, error) {
	tp.counter++
	stepName := fmt.Sprintf("_time.now:%d", tp.counter)

	var t time.Time

	err := Run(ctx, tp.wc, stepName, &t, func(_ context.Context) (time.Time, error) {
		return tp.nowFn(), nil
	})

	return t, err
}

// IDProvider is a durable service for ID generation.
// Each call to New caches the result as a step, making it deterministic on replay.
type IDProvider struct {
	wc      *WorkflowContext
	counter int
	newFn   func() string
}

// New returns a new unique identifier. On first execution, a real ID is generated
// and cached. On replay, the cached value is returned.
func (ip *IDProvider) New(ctx context.Context) (string, error) {
	ip.counter++
	stepName := fmt.Sprintf("_id.new:%d", ip.counter)

	var id string

	err := Run(ctx, ip.wc, stepName, &id, func(_ context.Context) (string, error) {
		return ip.newFn(), nil
	})

	return id, err
}
