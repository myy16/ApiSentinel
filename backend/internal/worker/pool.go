package worker

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

var (
	ErrPoolClosed   = errors.New("worker pool is closed")
	ErrQueueFull    = errors.New("worker queue is full")
	ErrTaskTimeout  = errors.New("task execution timed out")
)

// Task represents a unit of work to be executed asynchronously.
type Task func(ctx context.Context)

// Pool manages a fixed number of worker goroutines consuming from a buffered job queue.
type Pool struct {
	workerCount int
	queueCap    int
	tasks       chan Task
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	closed      int32 // atomic boolean flag (0 = open, 1 = closed)
	activeJobs  int64 // atomic counter for currently executing jobs
}

// NewPool creates and starts a new worker pool with specified concurrency and buffer size.
func NewPool(workerCount, queueCap int) *Pool {
	if workerCount <= 0 {
		workerCount = 10
	}
	if queueCap <= 0 {
		queueCap = 5000
	}

	ctx, cancel := context.WithCancel(context.Background())

	p := &Pool{
		workerCount: workerCount,
		queueCap:    queueCap,
		tasks:       make(chan Task, queueCap),
		ctx:         ctx,
		cancel:      cancel,
	}

	p.start()
	return p
}

func (p *Pool) start() {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.workerLoop(i)
	}
	log.Info().Int("workers", p.workerCount).Int("queueCap", p.queueCap).Msg("Worker pool initialized")
}

func (p *Pool) workerLoop(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			// Drain remaining tasks before exiting
			p.drainRemaining()
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			p.executeTask(task)
		}
	}
}

func (p *Pool) executeTask(task Task) {
	atomic.AddInt64(&p.activeJobs, 1)
	defer atomic.AddInt64(&p.activeJobs, -1)

	// Protect against panic in worker jobs
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Msg("Recovered from panic in worker pool task")
		}
	}()

	task(p.ctx)
}

func (p *Pool) drainRemaining() {
	for {
		select {
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			p.executeTask(task)
		default:
			return
		}
	}
}

// Submit enqueues a task for asynchronous processing.
// Returns ErrPoolClosed if pool is shutting down, or ErrQueueFull if buffer is exhausted.
func (p *Pool) Submit(task Task) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		return ErrPoolClosed
	}

	select {
	case p.tasks <- task:
		return nil
	default:
		return ErrQueueFull
	}
}

// SubmitWithTimeout attempts to enqueue a task with a maximum wait time if the queue is temporarily full.
func (p *Pool) SubmitWithTimeout(task Task, timeout time.Duration) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		return ErrPoolClosed
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case p.tasks <- task:
		return nil
	case <-timer.C:
		return ErrQueueFull
	case <-p.ctx.Done():
		return ErrPoolClosed
	}
}

// Stop gracefully shuts down the pool, draining enqueued tasks up to the timeout duration.
func (p *Pool) Stop(timeout time.Duration) error {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return nil // Already stopped
	}

	close(p.tasks)
	p.cancel()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("Worker pool stopped gracefully")
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("worker pool shutdown timed out after %v", timeout)
	}
}

// ActiveJobs returns the number of tasks currently being processed.
func (p *Pool) ActiveJobs() int64 {
	return atomic.LoadInt64(&p.activeJobs)
}

// QueueLength returns the number of tasks currently waiting in the queue.
func (p *Pool) QueueLength() int {
	return len(p.tasks)
}
