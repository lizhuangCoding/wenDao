package main

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/service"
)

var _ func() error = Run

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

	if _, err := services.ai.Chat(context.Background(), "你好", nil, nil); !errors.Is(err, service.ErrAIDisabled) {
		t.Fatalf("expected ErrAIDisabled from degraded AI service, got %v", err)
	}
}

func TestBuildCoreServices_WiresSharedDependencies(t *testing.T) {
	cfg := &config.Config{}

	core := buildCoreServices(cfg, zap.NewNop(), &repositories{}, &infrastructure{})
	if core == nil {
		t.Fatal("expected core services")
	}
	if core.oauth == nil {
		t.Fatal("expected oauth service")
	}
	if core.verification == nil {
		t.Fatal("expected verification service")
	}
	if core.user == nil {
		t.Fatal("expected user service")
	}
	if core.category == nil || core.tag == nil || core.collection == nil || core.setting == nil {
		t.Fatal("expected base content services")
	}
	if core.notification == nil || core.upload == nil || core.stat == nil {
		t.Fatal("expected runtime services")
	}
	if core.taskRunner == nil {
		t.Fatal("expected task runner")
	}
}

func TestNewDisabledAIStack_ProvidesSafeDefaults(t *testing.T) {
	stack := newDisabledAIStack(&repositories{}, zap.NewNop())
	if stack == nil {
		t.Fatal("expected ai stack")
	}
	if stack.ai == nil {
		t.Fatal("expected disabled ai service")
	}
	if stack.cleanup == nil {
		t.Fatal("expected cleanup")
	}
	if stack.vector != nil {
		t.Fatal("expected no vector service in disabled stack")
	}
}
