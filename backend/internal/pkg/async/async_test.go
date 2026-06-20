package async

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestGoLogsReturnedErrors(t *testing.T) {
	core, logs := observer.New(zapcore.WarnLevel)
	logger := zap.New(core)
	done := make(chan struct{})

	Go(context.Background(), logger, "failing task", func(ctx context.Context) error {
		defer close(done)
		return errors.New("boom")
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected async task to finish")
	}

	entries := waitForLog(t, logs, "Async task failed")
	if len(entries) != 1 {
		t.Fatalf("expected one error log entry, got %d", len(entries))
	}
	if entries[0].ContextMap()["task"] != "failing task" {
		t.Fatalf("expected task name in log context, got %#v", entries[0].ContextMap())
	}
}

func TestGoRecoversAndLogsPanics(t *testing.T) {
	core, logs := observer.New(zapcore.ErrorLevel)
	logger := zap.New(core)
	done := make(chan struct{})

	Go(context.Background(), logger, "panic task", func(ctx context.Context) error {
		defer close(done)
		panic("boom")
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected async task to finish")
	}

	entries := waitForLog(t, logs, "Async task panicked")
	if len(entries) != 1 {
		t.Fatalf("expected one panic log entry, got %d", len(entries))
	}
	if entries[0].ContextMap()["task"] != "panic task" {
		t.Fatalf("expected task name in log context, got %#v", entries[0].ContextMap())
	}
}

func waitForLog(t *testing.T, logs *observer.ObservedLogs, message string) []observer.LoggedEntry {
	t.Helper()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		entries := logs.FilterMessage(message).All()
		if len(entries) > 0 {
			return entries
		}
		select {
		case <-deadline:
			return nil
		case <-ticker.C:
		}
	}
}
