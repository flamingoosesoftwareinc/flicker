package flicker

import (
	"context"
	"sync"
)

// LeaseStore tracks which workflows are currently being executed.
// Implementations are ephemeral — a process restart starts clean.
type LeaseStore interface {
	// Acquire claims a workflow for execution. Returns true if acquired,
	// false if already held (idempotent, not an error).
	Acquire(ctx context.Context, workflowID string) (bool, error)

	// Release removes the lease. Releasing a non-existent lease succeeds
	// silently (idempotent).
	Release(ctx context.Context, workflowID string) error

	// IsHeld returns whether a workflow has an active lease.
	IsHeld(ctx context.Context, workflowID string) (bool, error)
}

type memoryLeaseStore struct {
	mu     sync.Mutex
	leases map[string]struct{}
}

// NewMemoryLeaseStore returns an in-process LeaseStore backed by a Go map.
func NewMemoryLeaseStore() LeaseStore {
	return &memoryLeaseStore{
		leases: make(map[string]struct{}),
	}
}

func (s *memoryLeaseStore) Acquire(_ context.Context, workflowID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, held := s.leases[workflowID]; held {
		return false, nil
	}

	s.leases[workflowID] = struct{}{}
	return true, nil
}

func (s *memoryLeaseStore) Release(_ context.Context, workflowID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.leases, workflowID)
	return nil
}

func (s *memoryLeaseStore) IsHeld(_ context.Context, workflowID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, held := s.leases[workflowID]
	return held, nil
}
