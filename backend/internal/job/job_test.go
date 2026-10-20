package job

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackoffGrowsFourfold(t *testing.T) {
	assert.Equal(t, 30*time.Second, Backoff(1))
	assert.Equal(t, 2*time.Minute, Backoff(2))
	assert.Equal(t, 8*time.Minute, Backoff(3))
}

func TestPoolLimitsConcurrency(t *testing.T) {
	p := NewPool(2)
	var running, peak atomic.Int32
	done := make(chan struct{})
	for range 8 {
		go func() {
			_ = p.Do(context.Background(), func(context.Context) error {
				n := running.Add(1)
				for {
					old := peak.Load()
					if n <= old || peak.CompareAndSwap(old, n) {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
				running.Add(-1)
				return nil
			})
			done <- struct{}{}
		}()
	}
	for range 8 {
		<-done
	}
	assert.Equal(t, int32(2), peak.Load())
}

func TestPoolHonoursCancellation(t *testing.T) {
	p := NewPool(1)
	block := make(chan struct{})
	go func() {
		_ = p.Do(context.Background(), func(context.Context) error { <-block; return nil })
	}()
	time.Sleep(5 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := p.Do(ctx, func(context.Context) error { return nil })
	require.ErrorIs(t, err, context.Canceled)
	close(block)
}
