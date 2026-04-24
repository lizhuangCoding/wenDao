package main

import (
	"errors"
	"testing"

	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/service"
)

func TestInitServices_DisablesAIWhenComponentsUnavailable(t *testing.T) {
	cfg := &config.Config{}

	services, cleanup, err := initServices(cfg, zap.NewNop(), &repositories{}, &infrastructure{}, nil)
	if err != nil {
		t.Fatalf("expected initServices to degrade gracefully, got %v", err)
	}
	if cleanup == nil {
		t.Fatal("expected cleanup function")
	}
	defer cleanup()

	if services == nil || services.ai == nil {
		t.Fatal("expected AI service to be initialized in disabled mode")
	}

	if _, err := services.ai.Chat("你好", nil, nil); !errors.Is(err, service.ErrAIDisabled) {
		t.Fatalf("expected ErrAIDisabled from degraded AI service, got %v", err)
	}
}
