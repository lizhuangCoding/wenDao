package main

import "wenDao/internal/handler"

func registerUserSelfRoutes(
	access routeAccessGroups,
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
	notificationHandler *handler.NotificationHandler,
) {
	access.authenticated.POST("/auth/logout", authHandler.Logout)
	access.authenticated.GET("/auth/me", authHandler.GetUserInfo)
	access.authenticated.POST("/users/me/avatar", userHandler.UploadAvatar)
	access.authenticated.PUT("/users/me/username", userHandler.UpdateUsername)
	access.authenticated.PUT("/users/me/preferences", userHandler.UpdatePreferences)

	notifications := access.authenticated.Group("/notifications")
	notifications.GET("", notificationHandler.List)
	notifications.GET("/unread-count", notificationHandler.GetUnreadCount)
	notifications.PUT("/:id/read", notificationHandler.MarkRead)
	notifications.PUT("/read-all", notificationHandler.MarkAllRead)
}
