package async

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestTaskRunnerRetriesAndRecordsStats(t *testing.T) {
	runner := NewTaskRunner(context.Background(), zap.NewNop())

	var attempts int32
	done := make(chan struct{})
	if err := runner.Submit(context.Background(), "retry-task", func(ctx context.Context) error {
		current := atomic.AddInt32(&attempts, 1)
		if current < 3 {
			return errors.New("boom")
		}
		close(done)
		return nil
	}, WithRetries(2), WithRetryDelay(func(attempt int) time.Duration { return time.Millisecond })); err != nil {
		t.Fatalf("expected submit to succeed, got %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected retry task to complete")
	}

	stats := runner.Stats()
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if stats.Submitted != 1 || stats.Succeeded != 1 || stats.Retried != 2 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestTaskRunnerShutdownCancelsRunningTasksAndWaits(t *testing.T) {
	runner := NewTaskRunner(context.Background(), zap.NewNop())

	done := make(chan struct{})
	if err := runner.Submit(context.Background(), "blocking-task", func(ctx context.Context) error {
		defer close(done)
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatalf("expected submit to succeed, got %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runner.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("expected shutdown to succeed, got %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected running task to stop on shutdown")
	}

	stats := runner.Stats()
	if stats.Canceled != 1 {
		t.Fatalf("expected one canceled task, got %#v", stats)
	}
}

func TestTaskRunnerTimeoutRecordsTimedOutTask(t *testing.T) {
	runner := NewTaskRunner(context.Background(), zap.NewNop())

	done := make(chan struct{})
	if err := runner.Submit(context.Background(), "timeout-task", func(ctx context.Context) error {
		defer close(done)
		<-ctx.Done()
		return ctx.Err()
	}, WithTimeout(20*time.Millisecond)); err != nil {
		t.Fatalf("expected submit to succeed, got %v", err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected timeout task to finish")
	}

	stats := runner.Stats()
	if stats.TimedOut != 1 || stats.Failed != 1 {
		t.Fatalf("expected timeout/failure stats, got %#v", stats)
	}
}
