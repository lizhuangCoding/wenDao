package main

import (
	"wenDao/internal/handler"
)

func registerAdminRoutes(
	access routeAccessGroups,
	userHandler *handler.UserHandler,
	categoryHandler *handler.CategoryHandler,
	tagHandler *handler.TagHandler,
	collectionHandler *handler.CollectionHandler,
	articleHandler *handler.ArticleHandler,
	commentHandler *handler.CommentHandler,
	uploadHandler *handler.UploadHandler,
	siteHandler *handler.SiteHandler,
	statHandler *handler.StatHandler,
	aiObservabilityHandler *handler.AIObservabilityHandler,
	knowledgeDocumentHandler *handler.KnowledgeDocumentHandler,
	notificationHandler *handler.NotificationHandler,
) {
	admin := access.admin.Group("/admin")

	users := admin.Group("/users")
	users.GET("", userHandler.ListUsers)
	users.PUT("/:id/role", userHandler.UpdateUserRole)
	users.PUT("/:id/status", userHandler.UpdateUserStatus)

	articles := admin.Group("/articles")
	articles.GET("", articleHandler.AdminList)
	articles.POST("/batch-delete", articleHandler.BatchDelete)
	articles.GET("/:id", articleHandler.GetByID)
	articles.POST("", articleHandler.Create)
	articles.PUT("/:id", articleHandler.Update)
	articles.PUT("/:id/autosave", articleHandler.AutoSave)
	articles.DELETE("/:id", articleHandler.Delete)
	articles.PATCH("/:id/publish", articleHandler.Publish)
	articles.PATCH("/:id/draft", articleHandler.Draft)
	articles.PATCH("/:id/top", articleHandler.ToggleTop)
	articles.POST("/refresh-scores", articleHandler.UpdatePopularityScores)

	categories := admin.Group("/categories")
	categories.GET("", categoryHandler.AdminList)
	categories.POST("/batch-delete", categoryHandler.BatchDelete)
	categories.POST("", categoryHandler.Create)
	categories.PUT("/:id", categoryHandler.Update)
	categories.DELETE("/:id", categoryHandler.Delete)

	tags := admin.Group("/tags")
	tags.GET("", tagHandler.AdminList)
	tags.POST("/batch-delete", tagHandler.BatchDelete)
	tags.POST("", tagHandler.Create)
	tags.PUT("/:id", tagHandler.Update)
	tags.DELETE("/:id", tagHandler.Delete)

	collections := admin.Group("/collections")
	collections.GET("", collectionHandler.AdminList)
	collections.POST("/batch-delete", collectionHandler.BatchDelete)
	collections.POST("", collectionHandler.Create)
	collections.PUT("/:id", collectionHandler.Update)
	collections.DELETE("/:id", collectionHandler.Delete)

	comments := admin.Group("/comments")
	comments.GET("", commentHandler.AdminList)
	comments.POST("/batch-delete", commentHandler.BatchDelete)
	comments.DELETE("/:id", commentHandler.Delete)
	comments.POST("/:id/restore", commentHandler.Restore)

	knowledgeDocs := admin.Group("/knowledge-documents")
	knowledgeDocs.GET("", knowledgeDocumentHandler.List)
	knowledgeDocs.POST("/batch-delete", knowledgeDocumentHandler.BatchDelete)
	knowledgeDocs.GET("/:id", knowledgeDocumentHandler.Get)
	knowledgeDocs.POST("/:id/approve", knowledgeDocumentHandler.Approve)
	knowledgeDocs.POST("/:id/reject", knowledgeDocumentHandler.Reject)
	knowledgeDocs.DELETE("/:id", knowledgeDocumentHandler.Delete)

	admin.POST("/upload/image", uploadHandler.UploadImage)
	admin.GET("/stats/dashboard", statHandler.GetDashboardStats)
	admin.GET("/ai-observability/runs", aiObservabilityHandler.ListRuns)
	admin.POST("/ai-observability/runs/batch-delete", aiObservabilityHandler.BatchDeleteRuns)
	admin.PUT("/settings/sort-mode", articleHandler.SetSortMode)
	admin.PUT("/settings/slogan", siteHandler.SetSlogan)
	admin.PUT("/settings/contact-links", siteHandler.SetContactLinks)
	admin.POST("/notifications/broadcast", notificationHandler.Broadcast)
}
