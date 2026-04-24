package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"wenDao/config"
	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService   service.UserService
	uploadService service.UploadService
	oauthService  service.OAuthService
	cfg           *config.Config
}

// NewUserHandler 创建用户处理器实例
func NewUserHandler(userService service.UserService, uploadService service.UploadService, oauthService service.OAuthService, cfg *config.Config) *UserHandler {
	return &UserHandler{
		userService:   userService,
		uploadService: uploadService,
		oauthService:  oauthService,
		cfg:           cfg,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Username string `json:"username" binding:"required,min=2,max=50"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UpdateUsernameRequest 修改用户名请求
type UpdateUsernameRequest struct {
	Username string `json:"username" binding:"required,min=2,max=50"`
}

// Register 用户注册
func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	user, err := h.userService.Register(req.Email, req.Password, req.Username)
	if err != nil {
		if err.Error() == "email already exists" {
			response.Error(c, response.CodeInvalidParams, "Email already exists")
			return
		}
		response.InternalError(c, "Failed to register")
		return
	}

	// 清除敏感信息
	user.PasswordHash = nil

	response.Success(c, user)
}

// Login 用户登录
func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	token, user, err := h.userService.Login(req.Email, req.Password)
	if err != nil {
		if err.Error() == "invalid email or password" {
			response.Unauthorized(c, "Invalid email or password")
			return
		}
		if err.Error() == "account is banned" {
			response.Forbidden(c, "Account is banned")
			return
		}
		response.InternalError(c, "Failed to login")
		return
	}

	// 生成 Refresh Token
	refreshToken, err := h.userService.GenerateRefreshToken(user.ID, user.Role)
	if err != nil {
		response.InternalError(c, "Failed to generate refresh token")
		return
	}

	// 清除敏感信息
	user.PasswordHash = nil

	// 设置 Access Token Cookie
	isRelease := h.cfg.Server.Mode == "release"
	c.SetCookie(
		"token",
		token,
		h.cfg.JWT.AccessExpireHours*3600,
		"/",
		"",
		isRelease,
		true,
	)

	// 设置 Refresh Token Cookie（有效期7天）
	c.SetCookie(
		"refresh_token",
		refreshToken,
		h.cfg.JWT.RefreshExpireDays*24*3600,
		"/",
		"",
		isRelease,
		true,
	)

	response.Success(c, gin.H{
		"access_token":  token,
		"refresh_token": refreshToken,
		"expires_in":    h.cfg.JWT.AccessExpireHours * 3600,
		"user":          user,
	})
}

// Logout 用户登出
func (h *UserHandler) Logout(c *gin.Context) {
	token, exists := c.Get("token")
	if exists {
		_ = h.userService.Logout(token.(string))
	}

	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		refreshToken = c.GetHeader("X-Refresh-Token")
	}

	if refreshToken != "" {
		_ = h.userService.Logout(refreshToken)
	}

	c.SetCookie("token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	response.Success(c, gin.H{
		"message": "Logged out successfully",
	})
}

// GetCurrentUser 获取当前用户信息
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Missing user ID")
		return
	}

	user, err := h.userService.GetCurrentUser(userID.(int64))
	if err != nil {
		if err.Error() == "user not found" {
			response.NotFound(c, "User not found")
			return
		}
		response.InternalError(c, "Failed to get user")
		return
	}

	user.PasswordHash = nil
	response.Success(c, user)
}

// UpdateUsername 修改用户名
func (h *UserHandler) UpdateUsername(c *gin.Context) {
	var req UpdateUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Missing user ID")
		return
	}

	if err := h.userService.UpdateUsername(userID.(int64), req.Username); err != nil {
		if err.Error() == "username already exists" {
			response.Error(c, response.CodeInvalidParams, "Username already exists")
			return
		}
		response.InternalError(c, "Failed to update username")
		return
	}

	response.Success(c, nil)
}

// UploadAvatar 上传当前用户头像
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.InvalidParams(c, "Missing file parameter")
		return
	}
	defer file.Close()

	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Missing user ID")
		return
	}

	upload, err := h.uploadService.UploadImage(file, header, userID.(int64))
	if err != nil {
		switch {
		case err.Error() == "file type not allowed":
			response.InvalidParams(c, "File type not allowed. Only jpg, png, gif, webp are supported.")
		case strings.HasPrefix(err.Error(), "file size exceeds limit"):
			response.InvalidParams(c, err.Error())
		default:
			response.InternalError(c, "Failed to upload avatar")
		}
		return
	}

	if err := h.userService.UpdateAvatar(userID.(int64), upload.FilePath); err != nil {
		_ = h.uploadService.CleanupByFilePath(upload.FilePath)
		response.InternalError(c, "Failed to upload avatar")
		return
	}

	user, err := h.userService.GetCurrentUser(userID.(int64))
	if err != nil {
		if err.Error() == "user not found" {
			response.NotFound(c, "User not found")
			return
		}
		response.InternalError(c, "Failed to get user")
		return
	}

	user.PasswordHash = nil
	response.Success(c, user)
}

// GitHubLogin GitHub OAuth 跳转
func (h *UserHandler) GitHubLogin(c *gin.Context) {
	if err := service.ValidateGitHubOAuthConfig(h.cfg); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	state := generateRandomState()
	isRelease := h.cfg.Server.Mode == "release"
	c.SetCookie("oauth_state", state, 600, "/", "", isRelease, true)

	authURL := h.oauthService.GetGitHubAuthURL(state)
	c.Redirect(302, authURL)
}

// GitHubCallback GitHub OAuth 回调
func (h *UserHandler) GitHubCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		response.InvalidParams(c, "Missing code")
		return
	}

	savedState, err := c.Cookie("oauth_state")
	if err != nil || savedState != state {
		response.Forbidden(c, "Invalid state")
		return
	}

	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	token, user, err := h.userService.GitHubOAuthLogin(code)
	if err != nil {
		response.InternalError(c, "Failed to login with GitHub")
		return
	}

	user.PasswordHash = nil

	refreshToken, err := h.userService.GenerateRefreshToken(user.ID, user.Role)
	if err != nil {
		response.InternalError(c, "Failed to generate refresh token")
		return
	}

	isRelease := h.cfg.Server.Mode == "release"
	c.SetCookie("token", token, h.cfg.JWT.AccessExpireHours*3600, "/", "", isRelease, true)
	c.SetCookie("refresh_token", refreshToken, h.cfg.JWT.RefreshExpireDays*24*3600, "/", "", isRelease, true)

	redirectURL := h.cfg.Site.URL
	if redirectURL == "" {
		redirectURL = "http://localhost:3000"
	}
	if parsedURL, parseErr := url.Parse(redirectURL); parseErr != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		redirectURL = "http://localhost:3000"
	}

	c.Redirect(302, redirectURL)
}

// generateRandomState 生成随机 state
func generateRandomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
