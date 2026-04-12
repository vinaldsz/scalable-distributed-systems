package worker_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/vinaldsouza/album-store/internal/worker"
)

func TestPool(t *testing.T) {
	const jobCount = 100
	var processed atomic.Int32

	pool := worker.New(5, 200, 100)

	for i := 0; i < jobCount; i++ {
		pool.Submit(worker.Job{
			PhotoID: "photo",
			ProcessFn: func(ctx context.Context, j worker.Job) error {
				processed.Add(1)
				return nil
			},
		})
	}

	pool.Shutdown()

	if got := processed.Load(); got != jobCount {
		t.Errorf("expected %d jobs processed, got %d", jobCount, got)
	}
}

func TestPoolShutdownDrainsQueue(t *testing.T) {
	var processed atomic.Int32

	pool := worker.New(2, 50, 50)

	for i := 0; i < 50; i++ {
		pool.Submit(worker.Job{
			PhotoID: "photo",
			ProcessFn: func(ctx context.Context, j worker.Job) error {
				processed.Add(1)
				return nil
			},
		})
	}

	pool.Shutdown() // must drain all 50 before returning

	if got := processed.Load(); got != 50 {
		t.Errorf("expected 50 jobs processed after shutdown, got %d", got)
	}
}
