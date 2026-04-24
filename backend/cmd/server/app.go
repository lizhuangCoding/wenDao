package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wenDao/config"
)

type runDependencies struct {
	loadServerEnv      func() error
	loadConfig         func() (*config.Config, error)
	initLogger         func(config.LogConfig) *zap.Logger
	initInfrastructure func(*config.Config, *zap.Logger) (*infrastructure, error)
	initRepositories   func(*gorm.DB) *repositories
	initAIComponents   func(*config.Config, *zap.Logger, *redis.Client) (*aiComponents, error)
	initServices       func(*config.Config, *zap.Logger, *repositories, *infrastructure, *aiComponents) (*appServices, func(), error)
	initHandlers       func(*config.Config, *repositories, *appServices, *redis.Client) *appHandlers
	buildRouter        func(*config.Config, *zap.Logger, *redis.Client, *appHandlers) http.Handler
	listenAndServe     func(*http.Server) error
}

var appRunDependencies = runDependencies{
	loadServerEnv:      loadServerEnv,
	loadConfig:         config.LoadConfig,
	initLogger:         initLogger,
	initInfrastructure: initInfrastructure,
	initRepositories:   initRepositories,
	initAIComponents:   initAIComponents,
	initServices:       initServices,
	initHandlers:       initHandlers,
	buildRouter: func(cfg *config.Config, logger *zap.Logger, rdb *redis.Client, handlers *appHandlers) http.Handler {
		return buildRouter(cfg, logger, rdb, handlers)
	},
	listenAndServe: func(srv *http.Server) error {
		return srv.ListenAndServe()
	},
}

func MustRun() {
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}

func Run() error {
	deps := appRunDependencies

	_ = deps.loadServerEnv()

	cfg, err := deps.loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := deps.initLogger(cfg.Log)
	defer func() {
		_ = logger.Sync()
	}()

	infra, err := deps.initInfrastructure(cfg, logger)
	if err != nil {
		return fmt.Errorf("init infrastructure: %w", err)
	}

	repos := deps.initRepositories(infra.db)

	aiCore, err := deps.initAIComponents(cfg, logger, infra.rdbVector)
	if err != nil {
		logger.Warn("AI components unavailable, continuing in degraded mode", zap.Error(err))
	}

	services, cleanup, err := deps.initServices(cfg, logger, repos, infra, aiCore)
	if err != nil {
		return fmt.Errorf("init services: %w", err)
	}
	defer cleanup()

	handlers := deps.initHandlers(cfg, repos, services, infra.rdb)
	router := deps.buildRouter(cfg, logger, infra.rdb, handlers)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	logger.Info("Server starting", zap.String("addr", ":"+cfg.Server.Port), zap.String("mode", cfg.Server.Mode))
	if err := deps.listenAndServe(srv); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}
