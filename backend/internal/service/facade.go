package service

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/pkg/eino"
	articlerepo "wenDao/internal/repository/article"
	categoryrepo "wenDao/internal/repository/category"
	chatrepo "wenDao/internal/repository/chat"
	collectionrepo "wenDao/internal/repository/collection"
	commentrepo "wenDao/internal/repository/comment"
	knowledgerepo "wenDao/internal/repository/knowledge"
	notifrepo "wenDao/internal/repository/notification"
	settingrepo "wenDao/internal/repository/setting"
	statrepo "wenDao/internal/repository/stat"
	tagrepo "wenDao/internal/repository/tag"
	uploadrepo "wenDao/internal/repository/upload"
	userrepo "wenDao/internal/repository/user"
	aisvc "wenDao/internal/service/ai"
	articlesvc "wenDao/internal/service/article"
	authsvc "wenDao/internal/service/auth"
	categorysvc "wenDao/internal/service/category"
	chatsvc "wenDao/internal/service/chat"
	chatcore "wenDao/internal/service/chatcore"
	collectionsvc "wenDao/internal/service/collection"
	commentsvc "wenDao/internal/service/comment"
	knowledgesvc "wenDao/internal/service/knowledge"
	notifsvc "wenDao/internal/service/notification"
	settingsvc "wenDao/internal/service/setting"
	statsvc "wenDao/internal/service/stat"
	tagsvc "wenDao/internal/service/tag"
	uploadsvc "wenDao/internal/service/upload"
	usersvc "wenDao/internal/service/user"
)

type OAuthService = authsvc.OAuthService
type GitHubUserInfo = authsvc.GitHubUserInfo
type VerificationService = authsvc.VerificationService
type VerificationPurpose = authsvc.VerificationPurpose
type VerificationEmailSender = authsvc.VerificationEmailSender
type UserService = usersvc.UserService
type CategoryService = categorysvc.CategoryService
type TagService = tagsvc.TagService
type CollectionService = collectionsvc.CollectionService
type SettingService = settingsvc.SettingService
type VectorService = aisvc.VectorService
type ArticleChunk = aisvc.ArticleChunk
type ArticleSemanticProfileRepository = aisvc.ArticleSemanticProfileRepository
type KnowledgeDocumentService = knowledgesvc.KnowledgeDocumentService
type KnowledgeSourceInput = knowledgesvc.KnowledgeSourceInput
type CreateKnowledgeDocumentInput = knowledgesvc.CreateKnowledgeDocumentInput
type AIService = aisvc.AIService
type WritingAction = aisvc.WritingAction
type WritingRequest = aisvc.WritingRequest
type WritingResult = aisvc.WritingResult
type ArticleService = articlesvc.ArticleService
type CommentService = commentsvc.CommentService
type CommentServiceOption = commentsvc.CommentServiceOption
type CommentReplyNotificationSender = commentsvc.CommentReplyNotificationSender
type NotificationService = notifsvc.NotificationService
type UploadService = uploadsvc.UploadService
type UploadCleanupResult = uploadsvc.UploadCleanupResult
type StatService = statsvc.StatService
type DashboardStats = statsvc.DashboardStats
type AILogger = aisvc.AILogger
type AILogEntry = aisvc.AILogEntry
type LogRotationConfig = aisvc.LogRotationConfig
type ThinkTankService = chatcore.ThinkTankService
type StreamEvent = chatcore.StreamEvent
type ResearchConfig = chatsvc.ResearchConfig
type ConversationMemorySummarizer = chatsvc.ConversationMemorySummarizer
type Librarian = chatsvc.Librarian
type Journalist = chatsvc.Journalist
type ThinkTankSynthesizer = chatsvc.ThinkTankSynthesizer
type RunMetricsConfig = chatsvc.RunMetricsConfig

var ErrAIDisabled = aisvc.ErrAIDisabled
var ErrUnsupportedWritingAction = aisvc.ErrUnsupportedWritingAction
var ErrWritingContentEmpty = aisvc.ErrWritingContentEmpty
var ErrVerificationCodeInvalid = authsvc.ErrVerificationCodeInvalid
var ErrVerificationCodeTooFrequent = authsvc.ErrVerificationCodeTooFrequent
var ErrVerificationUnavailable = authsvc.ErrVerificationUnavailable
var ErrVerificationEmailNotConfigured = authsvc.ErrVerificationEmailNotConfigured

const (
	PurposeRegister      = authsvc.PurposeRegister
	PurposePasswordReset = authsvc.PurposePasswordReset
)

const (
	WritingActionPolish   = aisvc.WritingActionPolish
	WritingActionExpand   = aisvc.WritingActionExpand
	WritingActionShorten  = aisvc.WritingActionShorten
	WritingActionSEOTitle = aisvc.WritingActionSEOTitle
)

func NewOAuthService(cfg *config.Config) OAuthService { return authsvc.NewOAuthService(cfg) }
func ValidateGitHubOAuthConfig(cfg *config.Config) error {
	return authsvc.ValidateGitHubOAuthConfig(cfg)
}
func NewVerificationService(cfg *config.Config, rdb *redis.Client, sender VerificationEmailSender) VerificationService {
	return authsvc.NewVerificationService(cfg, rdb, sender)
}
func NewUserService(repo userrepo.UserRepository, oauth OAuthService, cfg *config.Config, rdb *redis.Client) UserService {
	return usersvc.NewUserService(repo, oauth, cfg, rdb)
}
func NewCategoryService(repo categoryrepo.CategoryRepository) CategoryService {
	return categorysvc.NewCategoryService(repo)
}
func NewTagService(repo tagrepo.TagRepository) TagService {
	return tagsvc.NewTagService(repo)
}
func NewCollectionService(repo collectionrepo.CollectionRepository, articleRepo articlerepo.ArticleRepository) CollectionService {
	return collectionsvc.NewCollectionService(repo, articleRepo)
}
func NewSettingService(repo settingrepo.SettingRepository) SettingService {
	return settingsvc.NewSettingService(repo)
}
func NewVectorService(store eino.RedisVectorStore, embedder eino.Embedder, logger *zap.Logger, profileRepos ...ArticleSemanticProfileRepository) VectorService {
	return aisvc.NewVectorService(store, embedder, logger, profileRepos...)
}
func NewKnowledgeDocumentService(docRepo knowledgerepo.KnowledgeDocumentRepository, srcRepo knowledgerepo.KnowledgeDocumentSourceRepository, vector VectorService, articleRepo articlerepo.ArticleRepository, categoryRepo categoryrepo.CategoryRepository, logger *zap.Logger) KnowledgeDocumentService {
	return knowledgesvc.NewKnowledgeDocumentService(docRepo, srcRepo, vector, articleRepo, categoryRepo, logger)
}
func NewDisabledAIService(reason string) AIService { return aisvc.NewDisabledAIService(reason) }
func NewAIService(llm eino.LLMClient, thinkTank ThinkTankService, logger *zap.Logger) AIService {
	return aisvc.NewAIService(llm, thinkTank, logger)
}
func NewArticleService(articleRepo articlerepo.ArticleRepository, categoryRepo categoryrepo.CategoryRepository, rdb *redis.Client, vector VectorService, logger *zap.Logger, extras ...any) ArticleService {
	return articlesvc.NewArticleService(articleRepo, categoryRepo, rdb, vector, logger, extras...)
}
func NewCommentService(commentRepo commentrepo.CommentRepository, articleRepo articlerepo.ArticleRepository, options ...CommentServiceOption) CommentService {
	return commentsvc.NewCommentService(commentRepo, articleRepo, options...)
}
func NewSMTPCommentReplyEmailSender(cfg config.EmailConfig, siteURL string) CommentReplyNotificationSender {
	return commentsvc.NewSMTPCommentReplyEmailSender(cfg, siteURL)
}
func WithReplyNotificationSender(sender CommentReplyNotificationSender) CommentServiceOption {
	return commentsvc.WithReplyNotificationSender(sender)
}
func WithCommentNotificationService(notifSvc NotificationService) CommentServiceOption {
	return commentsvc.WithNotificationService(notifSvc)
}
func WithCommentUserRepository(userRepo userrepo.UserRepository) CommentServiceOption {
	return commentsvc.WithUserRepository(userRepo)
}
func WithArticleCacheInvalidation(rdb *redis.Client) CommentServiceOption {
	return commentsvc.WithArticleCacheInvalidation(rdb)
}
func NewNotificationService(repo notifrepo.NotificationRepository) NotificationService {
	return notifsvc.NewNotificationService(repo)
}
func NewUploadService(repo uploadrepo.UploadRepository, cfg *config.Config) UploadService {
	return uploadsvc.NewUploadService(repo, cfg)
}
func NewStatService(repo *statrepo.StatRepository, rdb *redis.Client) *StatService {
	return statsvc.NewStatService(repo, rdb)
}
func NewAILoggerWithRotation(logDir string, rotation LogRotationConfig) (AILogger, error) {
	return aisvc.NewAILoggerWithRotation(logDir, rotation)
}
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
func NewRunMetricsConfig(cfg config.AIConfig) RunMetricsConfig {
	return chatsvc.NewRunMetricsConfig(cfg)
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
