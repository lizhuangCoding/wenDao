package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	pkgjwt "wenDao/internal/pkg/jwt"
	"wenDao/internal/pkg/response"
)

// AuthRequired 验证登录
func AuthRequired(jwtSecret string, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 Header 或 Cookie 中提取 token
		token := extractToken(c)
		if token == "" {
			response.Unauthorized(c, "Missing authorization token")
			c.Abort()
			return
		}

		// 解析 token
		claims, err := pkgjwt.ParseToken(token, jwtSecret)
		if err != nil {
			response.Unauthorized(c, "Invalid token")
			c.Abort()
			return
		}

		// 检查 token 是否在黑名单中（已登出）
		blacklisted, err := pkgjwt.IsTokenBlacklisted(rdb, token)
		if err != nil {
			// Redis 错误不影响业务，只记录日志
			// 可以在这里记录日志
		} else if blacklisted {
			response.Unauthorized(c, "Token has been revoked")
			c.Abort()
			return
		}

		// 将用户信息注入 context
		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Set("token", token)

		c.Next()
	}
}

// AdminRequired 验证管理员权限
func AdminRequired(jwtSecret string, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 先验证登录
		token := extractToken(c)
		if token == "" {
			response.Unauthorized(c, "Missing authorization token")
			c.Abort()
			return
		}

		claims, err := pkgjwt.ParseToken(token, jwtSecret)
		if err != nil {
			response.Unauthorized(c, "Invalid token")
			c.Abort()
			return
		}

		// 检查黑名单
		blacklisted, err := pkgjwt.IsTokenBlacklisted(rdb, token)
		if err == nil && blacklisted {
			response.Unauthorized(c, "Token has been revoked")
			c.Abort()
			return
		}

		// 验证管理员权限
		if claims.Role != "admin" {
			response.Forbidden(c, "Admin permission required")
			c.Abort()
			return
		}

		// 注入用户信息
		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Set("token", token)

		c.Next()
	}
}

// extractToken 从请求中提取 token
func extractToken(c *gin.Context) string {
	// 1. 从 Authorization Header 提取（优先）
	// 格式：Authorization: Bearer <token>
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// 2. 从 Cookie 中提取
	token, err := c.Cookie("token")
	if err == nil && token != "" {
		return token
	}

	return ""
}
