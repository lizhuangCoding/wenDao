package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"wenDao/config"
	"wenDao/internal/handler"
	"wenDao/internal/middleware"
)

func registerAuthRoutes(
	api *gin.RouterGroup,
	cfg *config.Config,
	rdb *redis.Client,
	rateLimits routeRateLimitFactory,
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
) {
	auth := api.Group("/auth")
	auth.Use(middleware.RateLimit(rdb, rateLimits.ip("auth-global", cfg.RateLimit.Global, time.Second, "认证接口访问过于频繁")))

	auth.POST("/register/code", middleware.RateLimit(rdb, rateLimits.ip("auth-register-code", cfg.RateLimit.VerificationCode, time.Minute, "注册验证码发送过于频繁")), userHandler.RequestRegisterCode)
	auth.POST("/register", middleware.RateLimit(rdb, rateLimits.ip("auth-register", cfg.RateLimit.Register, time.Minute, "注册请求过于频繁")), userHandler.Register)
	auth.POST("/login", middleware.RateLimit(rdb, rateLimits.ip("auth-login", cfg.RateLimit.Login, time.Minute, "登录尝试过于频繁")), userHandler.Login)
	auth.POST("/password-reset/code", middleware.RateLimit(rdb, rateLimits.ip("auth-password-reset-code", cfg.RateLimit.PasswordReset, time.Minute, "密码重置验证码发送过于频繁")), userHandler.RequestPasswordResetCode)
	auth.POST("/password-reset/confirm", middleware.RateLimit(rdb, rateLimits.ip("auth-password-reset-confirm", cfg.RateLimit.PasswordReset, time.Minute, "密码重置提交过于频繁")), userHandler.ConfirmPasswordReset)
	auth.GET("/github", userHandler.GitHubLogin)
	auth.POST("/refresh", middleware.RateLimit(rdb, rateLimits.ip("auth-refresh", cfg.RateLimit.Refresh, time.Minute, "登录状态刷新过于频繁")), authHandler.Refresh)
	auth.GET("/github/callback", userHandler.GitHubCallback)
}
