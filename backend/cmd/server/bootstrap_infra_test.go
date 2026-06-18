package main

import (
	"strings"
	"testing"

	"wenDao/config"
)

func TestProviderUsesArkEmbeddingOnlyForArkCompatibleProviders(t *testing.T) {
	tests := []struct {
		provider string
		want     bool
	}{
		{provider: "", want: true},
		{provider: "doubao", want: true},
		{provider: "ark", want: true},
		{provider: " Doubao ", want: true},
		{provider: "deepseek", want: false},
		{provider: "openai", want: false},
		{provider: "openai-compatible", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if got := providerUsesArkEmbedding(tt.provider); got != tt.want {
				t.Fatalf("providerUsesArkEmbedding(%q) = %v, want %v", tt.provider, got, tt.want)
			}
		})
	}
}

func TestMigrateDatabase_DisabledModeSkipsDatabaseAccess(t *testing.T) {
	cfg := &config.Config{Migration: config.MigrationConfig{Mode: "disabled"}}

	if err := migrateDatabase(nil, cfg); err != nil {
		t.Fatalf("expected disabled migration mode to skip database access, got %v", err)
	}
}

func TestMigrateDatabase_RejectsUnsupportedMode(t *testing.T) {
	cfg := &config.Config{Migration: config.MigrationConfig{Mode: "surprise"}}

	err := migrateDatabase(nil, cfg)
	if err == nil {
		t.Fatal("expected unsupported migration mode to fail")
	}
	if !strings.Contains(err.Error(), "unsupported migration mode") {
		t.Fatalf("expected unsupported migration mode error, got %v", err)
	}
}
