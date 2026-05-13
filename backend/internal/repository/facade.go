package repository

import (
	"gorm.io/gorm"

	articlerepo "wenDao/internal/repository/article"
	categoryrepo "wenDao/internal/repository/category"
	chatrepo "wenDao/internal/repository/chat"
	commentrepo "wenDao/internal/repository/comment"
	knowledgerepo "wenDao/internal/repository/knowledge"
	settingrepo "wenDao/internal/repository/setting"
	statrepo "wenDao/internal/repository/stat"
	uploadrepo "wenDao/internal/repository/upload"
	userrepo "wenDao/internal/repository/user"
)

type UserRepository = userrepo.UserRepository
type ArticleRepository = articlerepo.ArticleRepository
type ArticleFilter = articlerepo.ArticleFilter
type CategoryRepository = categoryrepo.CategoryRepository
type CommentRepository = commentrepo.CommentRepository
type CommentFilter = commentrepo.CommentFilter
type ChatMessageRepository = chatrepo.ChatMessageRepository
type ConversationRepository = chatrepo.ConversationRepository
type ConversationRunRepository = chatrepo.ConversationRunRepository
type ConversationRunStepRepository = chatrepo.ConversationRunStepRepository
type ConversationMemoryRepository = chatrepo.ConversationMemoryRepository
type KnowledgeDocumentRepository = knowledgerepo.KnowledgeDocumentRepository
type KnowledgeDocumentSourceRepository = knowledgerepo.KnowledgeDocumentSourceRepository
type KnowledgeDocumentFilter = knowledgerepo.KnowledgeDocumentFilter
type UploadRepository = uploadrepo.UploadRepository
type SettingRepository = settingrepo.SettingRepository
type StatRepository = statrepo.StatRepository

func NewUserRepository(db *gorm.DB) UserRepository       { return userrepo.NewUserRepository(db) }
func NewArticleRepository(db *gorm.DB) ArticleRepository { return articlerepo.NewArticleRepository(db) }
func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return categoryrepo.NewCategoryRepository(db)
}
func NewCommentRepository(db *gorm.DB) CommentRepository { return commentrepo.NewCommentRepository(db) }
func NewChatMessageRepository(db *gorm.DB) ChatMessageRepository {
	return chatrepo.NewChatMessageRepository(db)
}
func NewConversationRepository(db *gorm.DB) ConversationRepository {
	return chatrepo.NewConversationRepository(db)
}
func NewConversationRunRepository(db *gorm.DB) ConversationRunRepository {
	return chatrepo.NewConversationRunRepository(db)
}
func NewConversationRunStepRepository(db *gorm.DB) ConversationRunStepRepository {
	return chatrepo.NewConversationRunStepRepository(db)
}
func NewConversationMemoryRepository(db *gorm.DB) ConversationMemoryRepository {
	return chatrepo.NewConversationMemoryRepository(db)
}
func NewKnowledgeDocumentRepository(db *gorm.DB) KnowledgeDocumentRepository {
	return knowledgerepo.NewKnowledgeDocumentRepository(db)
}
func NewKnowledgeDocumentSourceRepository(db *gorm.DB) KnowledgeDocumentSourceRepository {
	return knowledgerepo.NewKnowledgeDocumentSourceRepository(db)
}
func NewUploadRepository(db *gorm.DB) UploadRepository   { return uploadrepo.NewUploadRepository(db) }
func NewSettingRepository(db *gorm.DB) SettingRepository { return settingrepo.NewSettingRepository(db) }
func NewStatRepository(db *gorm.DB) *StatRepository      { return statrepo.NewStatRepository(db) }
