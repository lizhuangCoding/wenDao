package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"wenDao/config"
	"wenDao/internal/pkg/jwt"
	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	userService service.UserService
	cfg         *config.Config
	rdb         *redis.Client
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(userService service.UserService, cfg *config.Config, rdb *redis.Client) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		cfg:         cfg,
		rdb:         rdb,
	}
}

// RefreshTokenRequest 刷新 Token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh 刷新 Access Token (实现 Token 旋转)
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有 body，尝试从 Cookie 获取
		cookie, err := c.Cookie("refresh_token")
		if err != nil {
			response.Unauthorized(c, "Invalid request")
			return
		}
		req.RefreshToken = cookie
	}

	// 1. 验证 Refresh Token 有效性
	claims, err := jwt.ValidateRefreshToken(req.RefreshToken, h.cfg.JWT.Secret)
	if err != nil {
		response.Unauthorized(c, "Invalid or expired refresh token")
		return
	}

	// 2. 检查 Refresh Token 是否在黑名单
	if h.rdb != nil {
		isBlacklisted, _ := jwt.IsTokenBlacklisted(h.rdb, req.RefreshToken)
		if isBlacklisted {
			response.Unauthorized(c, "Refresh token has been revoked")
			return
		}
	}

	// 3. 生成新的 Access Token
	accessToken, err := jwt.GenerateAccessToken(claims.UserID, claims.Role, h.cfg.JWT.Secret, h.cfg.JWT.AccessExpireHours)
	if err != nil {
		response.InternalError(c, "Failed to generate access token")
		return
	}

	// 4. 实现 Token 旋转：生成新的 Refresh Token 并作废旧的
	newRefreshToken, err := h.userService.GenerateRefreshToken(claims.UserID, claims.Role)
	if err != nil {
		response.InternalError(c, "Failed to generate refresh token")
		return
	}

	// 将旧的 Refresh Token 加入黑名单
	remainingTime := time.Until(claims.ExpiresAt.Time)
	if h.rdb != nil && remainingTime > 0 {
		_ = jwt.AddToBlacklist(h.rdb, req.RefreshToken, remainingTime)
	}

	// 设置新的 Refresh Token Cookie
	isRelease := h.cfg.Server.Mode == "release"
	c.SetCookie(
		"refresh_token",
		newRefreshToken,
		h.cfg.JWT.RefreshExpireDays*24*3600,
		"/",
		"",
		isRelease, // secure
		true,      // httpOnly
	)

	response.Success(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    h.cfg.JWT.AccessExpireHours * 3600,
	})
}

// Logout 登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// 获取 Refresh Token（从 Cookie 或 Header）
	refreshToken := c.GetHeader("X-Refresh-Token")
	if refreshToken == "" {
		cookie, err := c.Cookie("refresh_token")
		if err != nil {
			refreshToken = ""
		} else {
			refreshToken = cookie
		}
	}

	// 将 Refresh Token 加入黑名单
	if refreshToken != "" {
		claims, err := jwt.ParseToken(refreshToken, h.cfg.JWT.Secret)
		if err == nil && claims != nil {
			remainingTime := time.Until(claims.ExpiresAt.Time)
			if remainingTime > 0 {
				_ = jwt.AddToBlacklist(h.rdb, refreshToken, remainingTime)
			}
		}
	}

	// 清除 Refresh Token Cookie
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	response.Success(c, gin.H{"message": "Logged out successfully"})
}

// GetUserInfo 返回用户信息（包含 token 过期时间）
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := h.userService.GetCurrentUser(userID.(int64))
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	user.PasswordHash = nil

	response.Success(c, gin.H{
		"user":       user,
		"expires_in": h.cfg.JWT.AccessExpireHours * 3600,
	})
}
