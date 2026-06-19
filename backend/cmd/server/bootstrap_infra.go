package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	articleSemanticProfile  repository.ArticleSemanticProfileRepository
	category                repository.CategoryRepository
	tag                     repository.TagRepository
	collection              repository.CollectionRepository
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
	notification            repository.NotificationRepository
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

	if err := migrateDatabase(db, cfg); err != nil {
		return nil, err
	}
	logger.Info("Database migrated successfully")

	rdb := newRedisClient(cfg.Redis)
	logger.Info("Redis connected successfully")

	rdbVector := newRedisClient(cfg.RedisVector)
	logger.Info("Redis Vector connected successfully")

	return &infrastructure{db: db, rdb: rdb, rdbVector: rdbVector}, nil
}

func migrateDatabase(db *gorm.DB, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	switch cfg.Migration.Mode {
	case "versioned":
		return database.RunVersionedMigrations(db, database.VersionedMigrationOptions{Dir: cfg.Migration.Path})
	case "auto":
		return database.AutoMigrate(db)
	case "disabled":
		return nil
	default:
		return fmt.Errorf("unsupported migration mode %q", cfg.Migration.Mode)
	}
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
		articleSemanticProfile:  repository.NewArticleSemanticProfileRepository(db),
		category:                repository.NewCategoryRepository(db),
		tag:                     repository.NewTagRepository(db),
		collection:              repository.NewCollectionRepository(db),
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
		notification:            repository.NewNotificationRepository(db),
	}
}

func initAIComponents(cfg *config.Config, logger *zap.Logger, rdbVector *redis.Client) (*aiComponents, error) {
	llmClient, err := eino.NewLLMClient(&cfg.AI)
	if err != nil {
		return nil, fmt.Errorf("create ai llm client: %w", err)
	}
	logger.Info("AI LLM Client initialized successfully",
		zap.String("provider", cfg.AI.Provider),
		zap.String("model", cfg.AI.LLMModel))

	components := &aiComponents{llmClient: llmClient}
	if !providerUsesArkEmbedding(cfg.AI.Provider) {
		logger.Warn("Vector search disabled for non-Ark AI provider",
			zap.String("provider", cfg.AI.Provider),
			zap.String("reason", "no compatible embedding provider configured"))
		return components, nil
	}
	if rdbVector == nil {
		logger.Warn("Vector search disabled because Redis Vector client is unavailable")
		return components, nil
	}
	if strings.TrimSpace(cfg.AI.EmbeddingModel) == "" {
		logger.Warn("Vector search disabled because ai.embedding_model is empty")
		return components, nil
	}

	embedder, err := eino.NewDoubaoEmbedder(&cfg.AI)
	if err != nil {
		logger.Warn("Doubao Embedder unavailable, continuing without vector search", zap.Error(err))
		return components, nil
	}
	logger.Info("Doubao Embedder initialized successfully")

	const currentIndexName = "idx_wendao_v4"
	vectorStore := eino.NewRedisVectorStore(rdbVector, currentIndexName, logger)

	logger.Info("Detecting embedding model dimension...")
	testVec, err := embedder.Embed("dimension test")
	if err != nil {
		logger.Warn("Embedding dimension detection failed, continuing without vector search", zap.Error(err))
		return components, nil
	}
	actualDim := len(testVec)
	logger.Info("Model dimension detected", zap.Int("dimension", actualDim), zap.String("using_index", currentIndexName))

	if err := vectorStore.InitIndex(currentIndexName, actualDim); err != nil {
		logger.Warn("Redis Vector index unavailable, continuing without vector search", zap.Error(err))
		return components, nil
	}
	logger.Info("Redis Vector index initialized successfully")

	components.embedder = embedder
	components.vectorStore = vectorStore
	return components, nil
}

func providerUsesArkEmbedding(provider string) bool {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "doubao", "ark":
		return true
	default:
		return false
	}
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
		if err := pruneExpiredLogFiles(aiLogDir(cfg.Output), cfg.MaxAgeDays, time.Now()); err != nil {
			log.Printf("Failed to prune expired AI log files: %v", err)
		}
	} else {
		dir := logOutputDir(cfg.Output)

		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Printf("Failed to create log directory: %v", err)
		}
		if err := pruneExpiredLogFiles(dir, cfg.MaxAgeDays, time.Now()); err != nil {
			log.Printf("Failed to prune expired log files: %v", err)
		}

		todayFilename := time.Now().Format("2006-01-02") + ".log"
		fullPath := filepath.Join(dir, todayFilename)

		fileWriter := &lumberjack.Logger{
			Filename:   fullPath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
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

func logOutputDir(output string) string {
	output = strings.TrimSpace(output)
	if output == "" || output == "stdout" {
		return ""
	}
	if strings.HasSuffix(output, string(os.PathSeparator)) {
		return filepath.Clean(output)
	}
	if ext := filepath.Ext(output); ext != "" {
		dir := filepath.Dir(output)
		if dir == "." {
			return "log"
		}
		return dir
	}
	return filepath.Clean(output)
}

func aiLogDir(output string) string {
	if dir := logOutputDir(output); dir != "" {
		return dir
	}
	return "log"
}

var generatedLogFilePattern = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})(?:-ai-chat)?(?:-\d{4}-\d{2}-\d{2}T.+)?\.log(?:\.gz)?$`)

func pruneExpiredLogFiles(dir string, maxAgeDays int, now time.Time) error {
	if dir == "" || maxAgeDays <= 0 {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -maxAgeDays)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := generatedLogFilePattern.FindStringSubmatch(entry.Name())
		if len(matches) != 2 {
			continue
		}
		logDate, err := time.ParseInLocation("2006-01-02", matches[1], now.Location())
		if err != nil || !logDate.Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}
