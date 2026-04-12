package worker

import (
	"context"
	"log"
	"sync"
)

// Job represents a photo upload task.
type Job struct {
	PhotoID string
	AlbumID string
	S3Key   string
	Data    []byte

	// ProcessFn is called by the worker. Injected so the pool has no import cycle
	// with s3client/store — the handler wires it up at startup.
	ProcessFn func(ctx context.Context, job Job) error
}

// Pool is a fixed-size goroutine pool draining a buffered job channel.
// memSem limits concurrent memory-heavy uploads to prevent OOM.
type Pool struct {
	jobs   chan Job
	wg     sync.WaitGroup
	memSem chan struct{}
}

// New creates a pool with workerCount goroutines, a buffered channel of queueCap,
// and a memory semaphore limiting concurrent uploads to maxConcurrentUploads.
func New(workerCount, queueCap, maxConcurrentUploads int) *Pool {
	p := &Pool{
		jobs:   make(chan Job, queueCap),
		memSem: make(chan struct{}, maxConcurrentUploads),
	}
	p.wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go p.run()
	}
	return p
}

// Submit enqueues a job. Non-blocking if queue has capacity; drops with a log if full.
func (p *Pool) Submit(job Job) {
	select {
	case p.jobs <- job:
	default:
		log.Printf("worker pool full, dropping job for photo %s", job.PhotoID)
	}
}

// Shutdown waits for all in-flight jobs to finish, then returns.
func (p *Pool) Shutdown() {
	close(p.jobs)
	p.wg.Wait()
}

func (p *Pool) run() {
	defer p.wg.Done()
	for job := range p.jobs {
		p.memSem <- struct{}{} // acquire memory slot
		if err := job.ProcessFn(context.Background(), job); err != nil {
			log.Printf("worker error photo %s: %v", job.PhotoID, err)
		}
		job.Data = nil  // allow GC
		<-p.memSem      // release memory slot
	}
}
