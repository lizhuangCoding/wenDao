package user

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/pkg/httpcookie"
	"wenDao/internal/pkg/pagination"
	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
	"wenDao/internal/svcerrors"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService         service.UserService
	uploadService       service.UploadService
	oauthService        service.OAuthService
	verificationService service.VerificationService
	cfg                 *config.Config
	logger              *zap.Logger
}

// NewUserHandler 创建用户处理器实例
func NewUserHandler(userService service.UserService, uploadService service.UploadService, oauthService service.OAuthService, verificationService service.VerificationService, cfg *config.Config) *UserHandler {
	logger := zap.L()
	if logger == nil {
		logger = zap.NewNop()
	}
	return &UserHandler{
		userService:         userService,
		uploadService:       uploadService,
		oauthService:        oauthService,
		verificationService: verificationService,
		cfg:                 cfg,
		logger:              logger,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email            string `json:"email" binding:"required,email"`
	Password         string `json:"password" binding:"required,min=6"`
	Username         string `json:"username" binding:"required,min=2,max=50"`
	VerificationCode string `json:"verification_code" binding:"required,min=4,max=10"`
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

type UpdatePreferencesRequest struct {
	CommentReplyEmailEnabled *bool `json:"comment_reply_email_enabled" binding:"required"`
}

// Register 用户注册
func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	exists, err := h.userService.EmailExists(req.Email)
	if err != nil {
		response.InternalError(c, "Failed to check email")
		return
	}
	if exists {
		response.Error(c, response.CodeInvalidParams, "Email already exists")
		return
	}

	if h.verificationService == nil {
		response.ServiceUnavailable(c, "Verification service is unavailable")
		return
	}
	if err := h.verificationService.VerifyCode(c.Request.Context(), req.Email, service.PurposeRegister, req.VerificationCode); err != nil {
		h.handleVerificationVerifyError(c, err)
		return
	}

	user, err := h.userService.Register(req.Email, req.Password, req.Username)
	if err != nil {
		if errors.Is(err, svcerrors.ErrEmailAlreadyExists) {
			response.Error(c, response.CodeInvalidParams, "Email already exists")
			return
		}
		response.InternalError(c, "Failed to register")
		return
	}

	token, loginUser, err := h.userService.Login(req.Email, req.Password)
	if err != nil {
		response.InternalError(c, "Failed to login after registration")
		return
	}
	if loginUser != nil {
		user = loginUser
	}

	refreshToken, err := h.userService.GenerateRefreshToken(user.ID, user.Role)
	if err != nil {
		response.InternalError(c, "Failed to generate refresh token")
		return
	}

	user.PasswordHash = nil

	httpcookie.SetAuthCookies(c, h.cfg, token, refreshToken)

	response.Success(c, gin.H{
		"access_token":  token,
		"refresh_token": refreshToken,
		"expires_in":    h.cfg.JWT.AccessExpireHours * 3600,
		"user":          user,
	})
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
		if errors.Is(err, svcerrors.ErrInvalidEmailOrPassword) {
			response.Unauthorized(c, "Invalid email or password")
			return
		}
		if errors.Is(err, svcerrors.ErrAccountBanned) {
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

	user.PasswordHash = nil

	httpcookie.SetAuthCookies(c, h.cfg, token, refreshToken)

	response.Success(c, gin.H{
		"access_token":  token,
		"refresh_token": refreshToken,
		"expires_in":    h.cfg.JWT.AccessExpireHours * 3600,
		"user":          user,
	})
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
		if errors.Is(err, svcerrors.ErrUsernameAlreadyExists) {
			response.Error(c, response.CodeInvalidParams, "Username already exists")
			return
		}
		response.InternalError(c, "Failed to update username")
		return
	}

	response.Success(c, nil)
}

func (h *UserHandler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Missing user ID")
		return
	}

	if err := h.userService.UpdateCommentReplyEmailEnabled(userID.(int64), *req.CommentReplyEmailEnabled); err != nil {
		if errors.Is(err, svcerrors.ErrUserNotFound) {
			response.NotFound(c, "User not found")
			return
		}
		response.InternalError(c, "Failed to update preferences")
		return
	}

	user, err := h.userService.GetCurrentUser(userID.(int64))
	if err != nil {
		response.InternalError(c, "Failed to get user")
		return
	}
	user.PasswordHash = nil
	response.Success(c, user)
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
		if errors.Is(err, svcerrors.ErrFileTypeNotAllowed) {
			response.InvalidParams(c, "File type not allowed. Only jpg, png, gif, webp are supported.")
		} else if errors.Is(err, svcerrors.ErrFileSizeExceedsLimit) {
			response.InvalidParams(c, err.Error())
		} else {
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
		if errors.Is(err, svcerrors.ErrUserNotFound) {
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
	httpcookie.SetOAuthStateCookie(c, h.cfg, state)

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

	httpcookie.ClearOAuthStateCookie(c)

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

	httpcookie.SetAuthCookies(c, h.cfg, token, refreshToken)

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

// ListUsers 获取用户列表（管理员）
func (h *UserHandler) ListUsers(c *gin.Context) {
	p := pagination.FromQuery(c)
	role := c.Query("role")
	status := c.Query("status")
	search := c.Query("search")

	users, total, err := h.userService.ListUsers(p.Page, p.PageSize, role, status, search)
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to list users", err)
		return
	}

	response.Success(c, gin.H{
		"data":       users,
		"total":      total,
		"page":       p.Page,
		"pageSize":   p.PageSize,
		"totalPages": pagination.TotalPages(total, p.PageSize),
	})
}

// UpdateUserRole 更新用户角色（管理员）
func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid user ID")
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=user admin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	// 不能修改自己的角色
	currentUserID, _ := c.Get("user_id")
	if currentUserID.(int64) == userID {
		response.Error(c, response.CodeInvalidParams, "Cannot modify your own role")
		return
	}

	if err := h.userService.UpdateUserRole(userID, req.Role); err != nil {
		response.InternalErrorWithErr(c, "Failed to update user role", err)
		return
	}

	response.Success(c, nil)
}

// UpdateUserStatus 更新用户状态（管理员：封禁/解封）
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid user ID")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=active banned"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	// 不能修改自己的状态
	currentUserID, _ := c.Get("user_id")
	if currentUserID.(int64) == userID {
		response.Error(c, response.CodeInvalidParams, "Cannot modify your own status")
		return
	}

	if err := h.userService.UpdateUserStatus(userID, req.Status); err != nil {
		response.InternalErrorWithErr(c, "Failed to update user status", err)
		return
	}

	response.Success(c, nil)
}
