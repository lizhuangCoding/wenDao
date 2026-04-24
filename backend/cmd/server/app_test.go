package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wenDao/config"
)

func stubRunDependencies(t *testing.T, overrides runDependencies) func() {
	t.Helper()

	original := appRunDependencies
	stubbed := original

	if overrides.loadServerEnv != nil {
		stubbed.loadServerEnv = overrides.loadServerEnv
	}
	if overrides.loadConfig != nil {
		stubbed.loadConfig = overrides.loadConfig
	}
	if overrides.initLogger != nil {
		stubbed.initLogger = overrides.initLogger
	}
	if overrides.initInfrastructure != nil {
		stubbed.initInfrastructure = overrides.initInfrastructure
	}
	if overrides.initRepositories != nil {
		stubbed.initRepositories = overrides.initRepositories
	}
	if overrides.initAIComponents != nil {
		stubbed.initAIComponents = overrides.initAIComponents
	}
	if overrides.initServices != nil {
		stubbed.initServices = overrides.initServices
	}
	if overrides.initHandlers != nil {
		stubbed.initHandlers = overrides.initHandlers
	}
	if overrides.buildRouter != nil {
		stubbed.buildRouter = overrides.buildRouter
	}
	if overrides.listenAndServe != nil {
		stubbed.listenAndServe = overrides.listenAndServe
	}

	appRunDependencies = stubbed
	return func() {
		appRunDependencies = original
	}
}

func TestRun_WrapsInitInfrastructureError(t *testing.T) {
	sentinel := errors.New("infra boom")

	restore := stubRunDependencies(t, runDependencies{
		loadServerEnv: func() error { return nil },
		loadConfig: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
		initLogger: func(config.LogConfig) *zap.Logger {
			return zap.NewNop()
		},
		initInfrastructure: func(*config.Config, *zap.Logger) (*infrastructure, error) {
			return nil, sentinel
		},
	})
	defer restore()

	err := Run()
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected Run error to wrap sentinel, got %v", err)
	}
	if !strings.Contains(err.Error(), "init infrastructure") {
		t.Fatalf("expected infrastructure context in error, got %v", err)
	}
}

func TestRun_WrapsListenAndServeError(t *testing.T) {
	sentinel := errors.New("listen boom")

	restore := stubRunDependencies(t, runDependencies{
		loadServerEnv: func() error { return nil },
		loadConfig: func() (*config.Config, error) {
			cfg := &config.Config{}
			cfg.Server.Port = "8089"
			return cfg, nil
		},
		initLogger: func(config.LogConfig) *zap.Logger {
			return zap.NewNop()
		},
		initInfrastructure: func(*config.Config, *zap.Logger) (*infrastructure, error) {
			return &infrastructure{}, nil
		},
		initRepositories: func(*gorm.DB) *repositories {
			return &repositories{}
		},
		initAIComponents: func(*config.Config, *zap.Logger, *redis.Client) (*aiComponents, error) {
			return nil, nil
		},
		initServices: func(*config.Config, *zap.Logger, *repositories, *infrastructure, *aiComponents) (*appServices, func(), error) {
			return &appServices{}, func() {}, nil
		},
		initHandlers: func(*config.Config, *repositories, *appServices, *redis.Client) *appHandlers {
			return &appHandlers{}
		},
		buildRouter: func(*config.Config, *zap.Logger, *redis.Client, *appHandlers) http.Handler {
			return http.NewServeMux()
		},
		listenAndServe: func(*http.Server) error {
			return sentinel
		},
	})
	defer restore()

	err := Run()
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected Run error to wrap sentinel, got %v", err)
	}
	if !strings.Contains(err.Error(), "listen and serve") {
		t.Fatalf("expected listen context in error, got %v", err)
	}
}
