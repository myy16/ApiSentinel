package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool_Execution(t *testing.T) {
	pool := NewPool(4, 100)
	defer pool.Stop(2 * time.Second)

	var counter int64
	var wg sync.WaitGroup

	taskCount := 50
	wg.Add(taskCount)

	for i := 0; i < taskCount; i++ {
		err := pool.Submit(func(ctx context.Context) {
			defer wg.Done()
			atomic.AddInt64(&counter, 1)
		})
		if err != nil {
			t.Fatalf("unexpected error on submit: %v", err)
		}
	}

	wg.Wait()

	if atomic.LoadInt64(&counter) != int64(taskCount) {
		t.Errorf("expected counter %d, got %d", taskCount, counter)
	}
}

func TestWorkerPool_PanicRecovery(t *testing.T) {
	pool := NewPool(2, 10)
	defer pool.Stop(2 * time.Second)

	var wg sync.WaitGroup
	wg.Add(2)

	var completedHealthyTask int64

	// Task 1: panics
	_ = pool.Submit(func(ctx context.Context) {
		defer wg.Done()
		panic("simulated fatal panic in worker job")
	})

	// Task 2: healthy task
	_ = pool.Submit(func(ctx context.Context) {
		defer wg.Done()
		atomic.StoreInt64(&completedHealthyTask, 1)
	})

	wg.Wait()

	if atomic.LoadInt64(&completedHealthyTask) != 1 {
		t.Errorf("healthy task should complete despite prior task panic")
	}
}

func TestWorkerPool_GracefulDrain(t *testing.T) {
	pool := NewPool(2, 50)

	var processedCount int64

	for i := 0; i < 20; i++ {
		_ = pool.Submit(func(ctx context.Context) {
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt64(&processedCount, 1)
		})
	}

	err := pool.Stop(3 * time.Second)
	if err != nil {
		t.Fatalf("expected clean stop, got: %v", err)
	}

	if atomic.LoadInt64(&processedCount) != 20 {
		t.Errorf("expected all 20 queued tasks to be drained on stop, got %d", processedCount)
	}
}
