package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadServerEnv_LoadsConfigDotEnvFromBackendRoot(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, ".env"), []byte("DOUBAO_API_KEY=test-key\n"), 0o644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	oldValue, hadValue := os.LookupEnv("DOUBAO_API_KEY")
	_ = os.Unsetenv("DOUBAO_API_KEY")
	defer func() {
		if hadValue {
			_ = os.Setenv("DOUBAO_API_KEY", oldValue)
			return
		}
		_ = os.Unsetenv("DOUBAO_API_KEY")
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	if err := loadServerEnv(); err != nil {
		t.Fatalf("expected config/.env to load successfully: %v", err)
	}

	if got := os.Getenv("DOUBAO_API_KEY"); got != "test-key" {
		t.Fatalf("expected DOUBAO_API_KEY to be loaded from config/.env, got %q", got)
	}
}
