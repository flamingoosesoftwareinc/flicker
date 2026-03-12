package flicker

import (
	"context"
	"time"
)

// TimeProvider is a durable service for time operations.
// Each call to Now caches the result as a step, making it deterministic on replay.
type TimeProvider struct {
	p *Provider[time.Time]
}

// NewTimeProvider creates a TimeProvider backed by the given clock function.
func NewTimeProvider(wc *WorkflowContext, nowFn func() time.Time) *TimeProvider {
	return &TimeProvider{p: NewProvider(wc, "_time.now", nowFn)}
}

// Now returns the current time. On first execution, the real time is captured
// and cached. On replay, the cached value is returned.
func (tp *TimeProvider) Now(ctx context.Context) (time.Time, error) {
	return tp.p.Get(ctx)
}

// IDProvider is a durable service for ID generation.
// Each call to New caches the result as a step, making it deterministic on replay.
type IDProvider struct {
	p *Provider[string]
}

// NewIDProvider creates an IDProvider backed by the given ID generator function.
func NewIDProvider(wc *WorkflowContext, newFn func() string) *IDProvider {
	return &IDProvider{p: NewProvider(wc, "_id.new", newFn)}
}

// New returns a new unique identifier. On first execution, a real ID is generated
// and cached. On replay, the cached value is returned.
func (ip *IDProvider) New(ctx context.Context) (string, error) {
	return ip.p.Get(ctx)
}
