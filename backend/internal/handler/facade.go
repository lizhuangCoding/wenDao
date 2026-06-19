package handler

import (
	"github.com/redis/go-redis/v9"

	"wenDao/config"
	articlehandler "wenDao/internal/handler/article"
	authhandler "wenDao/internal/handler/auth"
	categoryhandler "wenDao/internal/handler/category"
	chathandler "wenDao/internal/handler/chat"
	collectionhandler "wenDao/internal/handler/collection"
	commenthandler "wenDao/internal/handler/comment"
	knowledgehandler "wenDao/internal/handler/knowledge"
	notifhandler "wenDao/internal/handler/notification"
	sitehandler "wenDao/internal/handler/site"
	stathandler "wenDao/internal/handler/stat"
	taghandler "wenDao/internal/handler/tag"
	uploadhandler "wenDao/internal/handler/upload"
	userhandler "wenDao/internal/handler/user"
	"wenDao/internal/repository"
	"wenDao/internal/service"
)

type UserHandler = userhandler.UserHandler
type AuthHandler = authhandler.AuthHandler
type CategoryHandler = categoryhandler.CategoryHandler
type TagHandler = taghandler.TagHandler
type CollectionHandler = collectionhandler.CollectionHandler
type ArticleHandler = articlehandler.ArticleHandler
type CommentHandler = commenthandler.CommentHandler
type UploadHandler = uploadhandler.UploadHandler
type AIHandler = chathandler.AIHandler
type SiteHandler = sitehandler.SiteHandler
type StatHandler = stathandler.StatHandler
type ChatHandler = chathandler.ChatHandler
type AIObservabilityHandler = chathandler.AIObservabilityHandler
type KnowledgeDocumentHandler = knowledgehandler.KnowledgeDocumentHandler
type NotificationHandler = notifhandler.NotificationHandler

func NewUserHandler(userSvc service.UserService, uploadSvc service.UploadService, oauthSvc service.OAuthService, verificationSvc service.VerificationService, cfg *config.Config) *UserHandler {
	return userhandler.NewUserHandler(userSvc, uploadSvc, oauthSvc, verificationSvc, cfg)
}
func NewAuthHandler(userSvc service.UserService, cfg *config.Config, rdb *redis.Client) *AuthHandler {
	return authhandler.NewAuthHandler(userSvc, cfg, rdb)
}
func NewCategoryHandler(categorySvc service.CategoryService) *CategoryHandler {
	return categoryhandler.NewCategoryHandler(categorySvc)
}
func NewTagHandler(tagSvc service.TagService) *TagHandler {
	return taghandler.NewTagHandler(tagSvc)
}
func NewCollectionHandler(collectionSvc service.CollectionService) *CollectionHandler {
	return collectionhandler.NewCollectionHandler(collectionSvc)
}
func NewArticleHandler(articleSvc service.ArticleService, statSvc *service.StatService, settingSvc service.SettingService, collectionSvc ...service.CollectionService) *ArticleHandler {
	return articlehandler.NewArticleHandler(articleSvc, statSvc, settingSvc, collectionSvc...)
}
func NewCommentHandler(commentSvc service.CommentService, statSvc *service.StatService) *CommentHandler {
	return commenthandler.NewCommentHandler(commentSvc, statSvc)
}
func NewUploadHandler(uploadSvc service.UploadService) *UploadHandler {
	return uploadhandler.NewUploadHandler(uploadSvc)
}
func NewAIHandler(aiSvc service.AIService, cfg *config.Config, runRepos ...repository.ConversationRunRepository) *AIHandler {
	return chathandler.NewAIHandler(aiSvc, cfg, runRepos...)
}
func NewSiteHandler(cfg *config.Config, articleSvc service.ArticleService, settingSvc service.SettingService) *SiteHandler {
	return sitehandler.NewSiteHandler(cfg, articleSvc, settingSvc)
}
func NewStatHandler(statSvc *service.StatService) *StatHandler {
	return stathandler.NewStatHandler(statSvc)
}
func NewChatHandler(
	cfg *config.Config,
	convRepo repository.ConversationRepository,
	msgRepo repository.ChatMessageRepository,
	runRepo repository.ConversationRunRepository,
	runStepRepo repository.ConversationRunStepRepository,
	memoryRepo repository.ConversationMemoryRepository,
) *ChatHandler {
	return chathandler.NewChatHandler(cfg, convRepo, msgRepo, runRepo, runStepRepo, memoryRepo)
}
func NewAIObservabilityHandler(
	runRepo repository.ConversationRunRepository,
	runStepRepo repository.ConversationRunStepRepository,
) *AIObservabilityHandler {
	return chathandler.NewAIObservabilityHandler(runRepo, runStepRepo)
}
func NewKnowledgeDocumentHandler(knowledgeSvc service.KnowledgeDocumentService) *KnowledgeDocumentHandler {
	return knowledgehandler.NewKnowledgeDocumentHandler(knowledgeSvc)
}
func NewNotificationHandler(notifSvc service.NotificationService) *NotificationHandler {
	return notifhandler.NewNotificationHandler(notifSvc)
}
