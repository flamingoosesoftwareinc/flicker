package flicker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryLeaseStore_AcquireRelease(t *testing.T) {
	ctx := context.Background()
	ls := NewMemoryLeaseStore()

	acquired, err := ls.Acquire(ctx, "wf-1")
	require.NoError(t, err)
	require.True(t, acquired)

	err = ls.Release(ctx, "wf-1")
	require.NoError(t, err)

	// Can re-acquire after release.
	acquired, err = ls.Acquire(ctx, "wf-1")
	require.NoError(t, err)
	require.True(t, acquired)
}

func TestMemoryLeaseStore_DoubleAcquire(t *testing.T) {
	ctx := context.Background()
	ls := NewMemoryLeaseStore()

	acquired, err := ls.Acquire(ctx, "wf-1")
	require.NoError(t, err)
	require.True(t, acquired)

	// Second acquire for same ID returns false.
	acquired, err = ls.Acquire(ctx, "wf-1")
	require.NoError(t, err)
	require.False(t, acquired)
}

func TestMemoryLeaseStore_ReleaseNonExistent(t *testing.T) {
	ctx := context.Background()
	ls := NewMemoryLeaseStore()

	// Releasing a lease that was never acquired succeeds silently.
	err := ls.Release(ctx, "wf-does-not-exist")
	require.NoError(t, err)
}

func TestMemoryLeaseStore_IsHeld(t *testing.T) {
	ctx := context.Background()
	ls := NewMemoryLeaseStore()

	held, err := ls.IsHeld(ctx, "wf-1")
	require.NoError(t, err)
	require.False(t, held)

	_, _ = ls.Acquire(ctx, "wf-1")

	held, err = ls.IsHeld(ctx, "wf-1")
	require.NoError(t, err)
	require.True(t, held)

	_ = ls.Release(ctx, "wf-1")

	held, err = ls.IsHeld(ctx, "wf-1")
	require.NoError(t, err)
	require.False(t, held)
}

func TestMemoryLeaseStore_ConcurrentAcquire(t *testing.T) {
	ctx := context.Background()
	ls := NewMemoryLeaseStore()

	const goroutines = 100
	var wins atomic.Int32
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			acquired, err := ls.Acquire(ctx, "wf-1")
			require.NoError(t, err)
			if acquired {
				wins.Add(1)
			}
		}()
	}

	wg.Wait()
	require.Equal(t, int32(1), wins.Load(), "exactly one goroutine should win the lease")
}
