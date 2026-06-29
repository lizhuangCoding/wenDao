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
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
) {
	auth := api.Group("/auth")
	auth.Use(middleware.RateLimit(rdb, middleware.RateLimitConfig{
		Name:    "auth-global",
		Type:    middleware.IPLimit,
		Limit:   cfg.RateLimit.Global,
		Window:  time.Second,
		Message: rateLimitMessage("认证接口访问过于频繁", cfg.RateLimit.Global, time.Second),
	}))

	auth.POST("/register/code", middleware.RateLimit(rdb, middleware.RateLimitConfig{
		Name:    "auth-register-code",
		Type:    middleware.IPLimit,
		Limit:   cfg.RateLimit.VerificationCode,
		Window:  time.Minute,
		Message: rateLimitMessage("注册验证码发送过于频繁", cfg.RateLimit.VerificationCode, time.Minute),
	}), userHandler.RequestRegisterCode)
	auth.POST("/register", middleware.RateLimit(rdb, middleware.RateLimitConfig{
		Name:    "auth-register",
		Type:    middleware.IPLimit,
		Limit:   cfg.RateLimit.Register,
		Window:  time.Minute,
		Message: rateLimitMessage("注册请求过于频繁", cfg.RateLimit.Register, time.Minute),
	}), userHandler.Register)
	auth.POST("/login", middleware.RateLimit(rdb, middleware.RateLimitConfig{
		Name:    "auth-login",
		Type:    middleware.IPLimit,
		Limit:   cfg.RateLimit.Login,
		Window:  time.Minute,
		Message: rateLimitMessage("登录尝试过于频繁", cfg.RateLimit.Login, time.Minute),
	}), userHandler.Login)
	auth.POST("/password-reset/code", middleware.RateLimit(rdb, middleware.RateLimitConfig{
		Name:    "auth-password-reset-code",
		Type:    middleware.IPLimit,
		Limit:   cfg.RateLimit.PasswordReset,
		Window:  time.Minute,
		Message: rateLimitMessage("密码重置验证码发送过于频繁", cfg.RateLimit.PasswordReset, time.Minute),
	}), userHandler.RequestPasswordResetCode)
	auth.POST("/password-reset/confirm", middleware.RateLimit(rdb, middleware.RateLimitConfig{
		Name:    "auth-password-reset-confirm",
		Type:    middleware.IPLimit,
		Limit:   cfg.RateLimit.PasswordReset,
		Window:  time.Minute,
		Message: rateLimitMessage("密码重置提交过于频繁", cfg.RateLimit.PasswordReset, time.Minute),
	}), userHandler.ConfirmPasswordReset)
	auth.GET("/github", userHandler.GitHubLogin)
	auth.POST("/refresh", middleware.RateLimit(rdb, middleware.RateLimitConfig{
		Name:    "auth-refresh",
		Type:    middleware.IPLimit,
		Limit:   cfg.RateLimit.Refresh,
		Window:  time.Minute,
		Message: rateLimitMessage("登录状态刷新过于频繁", cfg.RateLimit.Refresh, time.Minute),
	}), authHandler.Refresh)
	auth.GET("/github/callback", userHandler.GitHubCallback)
}
