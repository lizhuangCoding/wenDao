package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPruneExpiredLogFilesDeletesOnlyOwnedExpiredLogs(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.Local)

	for _, name := range []string{
		"2026-04-01.log",
		"2026-04-02-ai-chat.log",
		"2026-04-03.log.gz",
		"2026-05-10.log",
		"2026-05-11-ai-chat.log",
		"random.log",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("test"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	if err := pruneExpiredLogFiles(dir, 14, now); err != nil {
		t.Fatalf("expected pruning to succeed, got %v", err)
	}

	for _, name := range []string{"2026-04-01.log", "2026-04-02-ai-chat.log", "2026-04-03.log.gz"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("expected expired generated log %s to be removed, stat err=%v", name, err)
		}
	}

	for _, name := range []string{"2026-05-10.log", "2026-05-11-ai-chat.log", "random.log"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s to be kept, got %v", name, err)
		}
	}
}

func TestPruneExpiredLogFilesIgnoresDisabledRetention(t *testing.T) {
	dir := t.TempDir()
	name := "2026-04-01.log"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("test"), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}

	if err := pruneExpiredLogFiles(dir, 0, time.Date(2026, 5, 18, 12, 0, 0, 0, time.Local)); err != nil {
		t.Fatalf("expected disabled pruning to succeed, got %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Fatalf("expected %s to be kept, got %v", name, err)
	}
}

func TestAILogDirDefaultsToLogWhenAppLogsGoStdout(t *testing.T) {
	if got := aiLogDir("stdout"); got != "log" {
		t.Fatalf("expected stdout AI logs to use log directory, got %q", got)
	}
}
