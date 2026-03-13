package flicker

import "github.com/alitto/pond/v2"

// WorkerPool abstracts the worker pool used by the engine to dispatch
// workflow executions. The default implementation wraps pond. Implement
// this interface to use ants, tunny, or a stdlib semaphore instead.
type WorkerPool interface {
	// Submit enqueues a function for execution by a worker.
	Submit(fn func())

	// StopAndWait signals the pool to stop accepting new work and blocks
	// until all in-flight tasks complete.
	StopAndWait()
}

// PondPool wraps pond/v2 to implement WorkerPool.
type PondPool struct {
	pool pond.Pool
}

// NewPondPool creates a WorkerPool backed by pond with the given concurrency.
func NewPondPool(workers int) *PondPool {
	return &PondPool{pool: pond.NewPool(workers)}
}

func (p *PondPool) Submit(fn func()) { p.pool.Submit(fn) }
func (p *PondPool) StopAndWait()     { p.pool.StopAndWait() }
