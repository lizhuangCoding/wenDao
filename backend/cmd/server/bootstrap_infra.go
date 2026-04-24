package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/gorm"

	"wenDao/config"
	"wenDao/internal/pkg/database"
	"wenDao/internal/pkg/eino"
	"wenDao/internal/repository"
)

type infrastructure struct {
	db        *gorm.DB
	rdb       *redis.Client
	rdbVector *redis.Client
}

type repositories struct {
	user                    repository.UserRepository
	article                 repository.ArticleRepository
	category                repository.CategoryRepository
	comment                 repository.CommentRepository
	chatMessage             repository.ChatMessageRepository
	conversation            repository.ConversationRepository
	conversationRun         repository.ConversationRunRepository
	conversationRunStep     repository.ConversationRunStepRepository
	conversationMemory      repository.ConversationMemoryRepository
	knowledgeDocument       repository.KnowledgeDocumentRepository
	knowledgeDocumentSource repository.KnowledgeDocumentSourceRepository
	upload                  repository.UploadRepository
	setting                 repository.SettingRepository
	stat                    *repository.StatRepository
}

type aiComponents struct {
	embedder    eino.Embedder
	llmClient   eino.LLMClient
	vectorStore eino.RedisVectorStore
}

func initInfrastructure(cfg *config.Config, logger *zap.Logger) (*infrastructure, error) {
	db, err := database.InitMySQL(&cfg.Database)
	if err != nil {
		return nil, err
	}
	logger.Info("MySQL connected successfully")

	if err := database.AutoMigrate(db); err != nil {
		return nil, err
	}
	logger.Info("Database migrated successfully")

	rdb := newRedisClient(cfg.Redis)
	logger.Info("Redis connected successfully")

	rdbVector := newRedisClient(cfg.RedisVector)
	logger.Info("Redis Vector connected successfully")

	return &infrastructure{db: db, rdb: rdb, rdbVector: rdbVector}, nil
}

func newRedisClient(cfg config.RedisConfig) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
}

func initRepositories(db *gorm.DB) *repositories {
	return &repositories{
		user:                    repository.NewUserRepository(db),
		article:                 repository.NewArticleRepository(db),
		category:                repository.NewCategoryRepository(db),
		comment:                 repository.NewCommentRepository(db),
		chatMessage:             repository.NewChatMessageRepository(db),
		conversation:            repository.NewConversationRepository(db),
		conversationRun:         repository.NewConversationRunRepository(db),
		conversationRunStep:     repository.NewConversationRunStepRepository(db),
		conversationMemory:      repository.NewConversationMemoryRepository(db),
		knowledgeDocument:       repository.NewKnowledgeDocumentRepository(db),
		knowledgeDocumentSource: repository.NewKnowledgeDocumentSourceRepository(db),
		upload:                  repository.NewUploadRepository(db),
		setting:                 repository.NewSettingRepository(db),
		stat:                    repository.NewStatRepository(db),
	}
}

func initAIComponents(cfg *config.Config, logger *zap.Logger, rdbVector *redis.Client) (*aiComponents, error) {
	embedder, err := eino.NewDoubaoEmbedder(&cfg.AI)
	if err != nil {
		return nil, fmt.Errorf("create doubao embedder: %w", err)
	}
	logger.Info("Doubao Embedder initialized successfully")

	llmClient, err := eino.NewDoubaoLLMClient(&cfg.AI)
	if err != nil {
		return nil, fmt.Errorf("create doubao llm client: %w", err)
	}
	logger.Info("Doubao LLM Client initialized successfully")

	const currentIndexName = "idx_wendao_v4"
	vectorStore := eino.NewRedisVectorStore(rdbVector, currentIndexName, logger)

	logger.Info("Detecting embedding model dimension...")
	testVec, err := embedder.Embed("dimension test")
	if err != nil {
		return nil, fmt.Errorf("detect embedding model dimension: %w", err)
	}
	actualDim := len(testVec)
	logger.Info("Model dimension detected", zap.Int("dimension", actualDim), zap.String("using_index", currentIndexName))

	if err := vectorStore.InitIndex(currentIndexName, actualDim); err != nil {
		return nil, fmt.Errorf("initialize vector index: %w", err)
	}
	logger.Info("Redis Vector index initialized successfully")

	return &aiComponents{
		embedder:    embedder,
		llmClient:   llmClient,
		vectorStore: vectorStore,
	}, nil
}

func loadServerEnv() error {
	if err := godotenv.Load("config/.env"); err == nil {
		return nil
	}
	return godotenv.Load()
}

func initLogger(cfg config.LogConfig) *zap.Logger {
	level := zap.InfoLevel
	_ = level.UnmarshalText([]byte(cfg.Level))

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	if cfg.Format != "json" {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var cores []zapcore.Core
	if cfg.Output == "stdout" || cfg.Output == "" {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level))
	} else {
		dir := filepath.Dir(cfg.Output)
		if dir == "." {
			dir = "log"
		}

		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Printf("Failed to create log directory: %v", err)
		}

		todayFilename := time.Now().Format("2006-01-02") + ".log"
		fullPath := filepath.Join(dir, todayFilename)

		fileWriter := &lumberjack.Logger{
			Filename:   fullPath,
			MaxSize:    100,
			MaxBackups: 7,
			MaxAge:     28,
			Compress:   true,
		}

		var encoder zapcore.Encoder
		if cfg.Format == "json" {
			encoder = zapcore.NewJSONEncoder(encoderConfig)
		} else {
			encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
			encoder = zapcore.NewConsoleEncoder(encoderConfig)
		}
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(fileWriter), level))
	}

	return zap.New(zapcore.NewTee(cores...), zap.AddCaller())
}
