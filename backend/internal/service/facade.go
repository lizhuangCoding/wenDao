package service

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/pkg/eino"
	articlerepo "wenDao/internal/repository/article"
	categoryrepo "wenDao/internal/repository/category"
	chatrepo "wenDao/internal/repository/chat"
	commentrepo "wenDao/internal/repository/comment"
	knowledgerepo "wenDao/internal/repository/knowledge"
	settingrepo "wenDao/internal/repository/setting"
	statrepo "wenDao/internal/repository/stat"
	uploadrepo "wenDao/internal/repository/upload"
	userrepo "wenDao/internal/repository/user"
	aisvc "wenDao/internal/service/ai"
	articlesvc "wenDao/internal/service/article"
	authsvc "wenDao/internal/service/auth"
	categorysvc "wenDao/internal/service/category"
	chatsvc "wenDao/internal/service/chat"
	chatcore "wenDao/internal/service/chatcore"
	commentsvc "wenDao/internal/service/comment"
	knowledgesvc "wenDao/internal/service/knowledge"
	settingsvc "wenDao/internal/service/setting"
	statsvc "wenDao/internal/service/stat"
	uploadsvc "wenDao/internal/service/upload"
	usersvc "wenDao/internal/service/user"
)

type OAuthService = authsvc.OAuthService
type GitHubUserInfo = authsvc.GitHubUserInfo
type UserService = usersvc.UserService
type CategoryService = categorysvc.CategoryService
type SettingService = settingsvc.SettingService
type VectorService = aisvc.VectorService
type ArticleChunk = aisvc.ArticleChunk
type KnowledgeDocumentService = knowledgesvc.KnowledgeDocumentService
type KnowledgeSourceInput = knowledgesvc.KnowledgeSourceInput
type CreateKnowledgeDocumentInput = knowledgesvc.CreateKnowledgeDocumentInput
type AIService = aisvc.AIService
type ArticleService = articlesvc.ArticleService
type CommentService = commentsvc.CommentService
type UploadService = uploadsvc.UploadService
type StatService = statsvc.StatService
type DashboardStats = statsvc.DashboardStats
type AILogger = aisvc.AILogger
type AILogEntry = aisvc.AILogEntry
type ThinkTankService = chatcore.ThinkTankService
type StreamEvent = chatcore.StreamEvent
type ResearchConfig = chatsvc.ResearchConfig
type ConversationMemorySummarizer = chatsvc.ConversationMemorySummarizer
type Librarian = chatsvc.Librarian
type Journalist = chatsvc.Journalist
type ThinkTankSynthesizer = chatsvc.ThinkTankSynthesizer

var ErrAIDisabled = aisvc.ErrAIDisabled

func NewOAuthService(cfg *config.Config) OAuthService { return authsvc.NewOAuthService(cfg) }
func ValidateGitHubOAuthConfig(cfg *config.Config) error {
	return authsvc.ValidateGitHubOAuthConfig(cfg)
}
func NewUserService(repo userrepo.UserRepository, oauth OAuthService, cfg *config.Config, rdb *redis.Client) UserService {
	return usersvc.NewUserService(repo, oauth, cfg, rdb)
}
func NewCategoryService(repo categoryrepo.CategoryRepository) CategoryService {
	return categorysvc.NewCategoryService(repo)
}
func NewSettingService(repo settingrepo.SettingRepository) SettingService {
	return settingsvc.NewSettingService(repo)
}
func NewVectorService(store eino.RedisVectorStore, embedder eino.Embedder, logger *zap.Logger) VectorService {
	return aisvc.NewVectorService(store, embedder, logger)
}
func NewKnowledgeDocumentService(docRepo knowledgerepo.KnowledgeDocumentRepository, srcRepo knowledgerepo.KnowledgeDocumentSourceRepository, vector VectorService, articleRepo articlerepo.ArticleRepository, categoryRepo categoryrepo.CategoryRepository, logger *zap.Logger) KnowledgeDocumentService {
	return knowledgesvc.NewKnowledgeDocumentService(docRepo, srcRepo, vector, articleRepo, categoryRepo, logger)
}
func NewDisabledAIService(reason string) AIService { return aisvc.NewDisabledAIService(reason) }
func NewAIService(llm eino.LLMClient, thinkTank ThinkTankService, logger *zap.Logger) AIService {
	return aisvc.NewAIService(llm, thinkTank, logger)
}
func NewArticleService(articleRepo articlerepo.ArticleRepository, categoryRepo categoryrepo.CategoryRepository, rdb *redis.Client, vector VectorService, logger *zap.Logger) ArticleService {
	return articlesvc.NewArticleService(articleRepo, categoryRepo, rdb, vector, logger)
}
func NewCommentService(commentRepo commentrepo.CommentRepository, articleRepo articlerepo.ArticleRepository) CommentService {
	return commentsvc.NewCommentService(commentRepo, articleRepo)
}
func NewUploadService(repo uploadrepo.UploadRepository, cfg *config.Config) UploadService {
	return uploadsvc.NewUploadService(repo, cfg)
}
func NewStatService(repo *statrepo.StatRepository, rdb *redis.Client) *StatService {
	return statsvc.NewStatService(repo, rdb)
}
func NewAILogger(logDir string) (AILogger, error)        { return aisvc.NewAILogger(logDir) }
func NewLibrarianService(chain *eino.RAGChain) Librarian { return chatsvc.NewLibrarianService(chain) }
func NewJournalist(cfg *config.AIConfig) Journalist      { return chatsvc.NewJournalist(cfg) }
func NewThinkTankSynthesizer(llm eino.LLMClient) ThinkTankSynthesizer {
	return chatsvc.NewThinkTankSynthesizer(llm)
}
func NewConversationMemorySummarizer(llm eino.LLMClient) ConversationMemorySummarizer {
	return chatsvc.NewConversationMemorySummarizer(llm)
}
func NewThinkTankADKRunner(ctx context.Context, llm eino.LLMClient, librarian Librarian, knowledgeDocSvc KnowledgeDocumentService, researchCfg ResearchConfig) (any, error) {
	return chatsvc.NewThinkTankADKRunner(ctx, llm, librarian, knowledgeDocSvc, researchCfg)
}
func NewThinkTankService(
	librarian Librarian,
	journalist Journalist,
	synthesizer ThinkTankSynthesizer,
	runRepo chatrepo.ConversationRunRepository,
	runStepRepo chatrepo.ConversationRunStepRepository,
	memoryRepo chatrepo.ConversationMemoryRepository,
	convRepo chatrepo.ConversationRepository,
	msgRepo chatrepo.ChatMessageRepository,
	knowledgeSvc KnowledgeDocumentService,
	logger AILogger,
	options ...any,
) ThinkTankService {
	return chatsvc.NewThinkTankService(librarian, journalist, synthesizer, runRepo, runStepRepo, memoryRepo, convRepo, msgRepo, knowledgeSvc, logger, options...)
}

const (
	StreamEventStage     = chatcore.StreamEventStage
	StreamEventQuestion  = chatcore.StreamEventQuestion
	StreamEventChunk     = chatcore.StreamEventChunk
	StreamEventStep      = chatcore.StreamEventStep
	StreamEventResume    = chatcore.StreamEventResume
	StreamEventSnapshot  = chatcore.StreamEventSnapshot
	StreamEventHeartbeat = chatcore.StreamEventHeartbeat
	StreamEventDone      = chatcore.StreamEventDone
)

func IsAIDisabled(err error) bool { return errors.Is(err, ErrAIDisabled) }
