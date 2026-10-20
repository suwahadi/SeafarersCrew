package job

import (
	"context"
	"runtime"
)

// Pool bounds concurrent CPU-heavy work such as image and PDF processing.
type Pool struct {
	slots chan struct{}
}

// NewPool returns a Pool with size slots; size <= 0 uses the CPU count.
func NewPool(size int) *Pool {
	if size <= 0 {
		size = runtime.NumCPU()
	}
	return &Pool{slots: make(chan struct{}, size)}
}

// Do runs fn once a slot is free, or returns ctx.Err() if ctx ends first.
func (p *Pool) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	select {
	case p.slots <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-p.slots }()
	return fn(ctx)
}
