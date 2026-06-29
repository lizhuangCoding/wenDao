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
		response.InternalError(c, "检查邮箱是否已注册失败，请稍后重试")
		return
	}
	if exists {
		response.Error(c, response.CodeInvalidParams, "该邮箱已注册，请直接登录或更换邮箱")
		return
	}

	if h.verificationService == nil {
		response.ServiceUnavailable(c, "验证码服务暂不可用，请稍后重试")
		return
	}
	if err := h.verificationService.VerifyCode(c.Request.Context(), req.Email, service.PurposeRegister, req.VerificationCode); err != nil {
		h.handleVerificationVerifyError(c, err)
		return
	}

	user, err := h.userService.Register(req.Email, req.Password, req.Username)
	if err != nil {
		if errors.Is(err, svcerrors.ErrEmailAlreadyExists) {
			response.Error(c, response.CodeInvalidParams, "该邮箱已注册，请直接登录或更换邮箱")
			return
		}
		response.InternalError(c, "注册失败，请稍后重试")
		return
	}

	token, loginUser, err := h.userService.Login(req.Email, req.Password)
	if err != nil {
		response.InternalError(c, "注册成功但自动登录失败，请前往登录页手动登录")
		return
	}
	if loginUser != nil {
		user = loginUser
	}

	refreshToken, err := h.userService.GenerateRefreshToken(user.ID, user.Role)
	if err != nil {
		response.InternalError(c, "生成登录凭证失败，请重新登录")
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
			response.Unauthorized(c, "邮箱或密码不正确，请检查后重试")
			return
		}
		if errors.Is(err, svcerrors.ErrAccountBanned) {
			response.Forbidden(c, "账号已被封禁，无法登录")
			return
		}
		response.InternalError(c, "登录失败，请稍后重试")
		return
	}

	// 生成 Refresh Token
	refreshToken, err := h.userService.GenerateRefreshToken(user.ID, user.Role)
	if err != nil {
		response.InternalError(c, "生成登录凭证失败，请重新登录")
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
		response.Unauthorized(c, "登录状态已失效，请重新登录后操作")
		return
	}

	if err := h.userService.UpdateUsername(userID.(int64), req.Username); err != nil {
		if errors.Is(err, svcerrors.ErrUsernameAlreadyExists) {
			response.Error(c, response.CodeInvalidParams, "用户名已被占用，请换一个用户名")
			return
		}
		response.InternalError(c, "修改用户名失败，请稍后重试")
		return
	}

	response.SuccessEmpty(c)
}

func (h *UserHandler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "登录状态已失效，请重新登录后操作")
		return
	}

	if err := h.userService.UpdateCommentReplyEmailEnabled(userID.(int64), *req.CommentReplyEmailEnabled); err != nil {
		if errors.Is(err, svcerrors.ErrUserNotFound) {
			response.NotFound(c, "当前用户不存在或已被禁用，请重新登录")
			return
		}
		response.InternalError(c, "保存通知偏好失败，请稍后重试")
		return
	}

	user, err := h.userService.GetCurrentUser(userID.(int64))
	if err != nil {
		response.InternalError(c, "获取最新用户信息失败，请刷新页面重试")
		return
	}
	user.PasswordHash = nil
	response.Success(c, user)
}

// UploadAvatar 上传当前用户头像
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.InvalidParams(c, "请选择要上传的头像文件")
		return
	}
	defer file.Close()

	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "登录状态已失效，请重新登录后操作")
		return
	}

	upload, err := h.uploadService.UploadImage(file, header, userID.(int64))
	if err != nil {
		if errors.Is(err, svcerrors.ErrFileTypeNotAllowed) {
			response.InvalidParams(c, "头像格式不支持，请上传 jpg、png、gif 或 webp 图片")
		} else if errors.Is(err, svcerrors.ErrFileSizeExceedsLimit) {
			response.InvalidParams(c, err.Error())
		} else {
			response.InternalError(c, "头像上传失败，请稍后重试")
		}
		return
	}

	if err := h.userService.UpdateAvatar(userID.(int64), upload.FilePath); err != nil {
		_ = h.uploadService.CleanupByFilePath(upload.FilePath)
		response.InternalError(c, "头像已上传但保存到个人资料失败，请稍后重试")
		return
	}

	user, err := h.userService.GetCurrentUser(userID.(int64))
	if err != nil {
		if errors.Is(err, svcerrors.ErrUserNotFound) {
			response.NotFound(c, "当前用户不存在或已被禁用，请重新登录")
			return
		}
		response.InternalError(c, "获取最新用户信息失败，请刷新页面重试")
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
		response.InvalidParams(c, "GitHub 登录回调缺少授权码，请重新发起登录")
		return
	}

	savedState, err := c.Cookie("oauth_state")
	if err != nil || savedState != state {
		response.Forbidden(c, "GitHub 登录状态校验失败，请重新发起登录")
		return
	}

	httpcookie.ClearOAuthStateCookie(c)

	token, user, err := h.userService.GitHubOAuthLogin(code)
	if err != nil {
		response.InternalError(c, "GitHub 登录失败，请稍后重试")
		return
	}

	user.PasswordHash = nil

	refreshToken, err := h.userService.GenerateRefreshToken(user.ID, user.Role)
	if err != nil {
		response.InternalError(c, "生成登录凭证失败，请重新登录")
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
		response.InternalErrorWithErr(c, "用户列表加载失败，请稍后重试", err)
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
		response.InvalidParams(c, "用户 ID 无效，请刷新页面后重试")
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
		response.Error(c, response.CodeInvalidParams, "不能修改自己的角色")
		return
	}

	if err := h.userService.UpdateUserRole(userID, req.Role); err != nil {
		response.InternalErrorWithErr(c, "更新用户角色失败，请稍后重试", err)
		return
	}

	response.SuccessEmpty(c)
}

// UpdateUserStatus 更新用户状态（管理员：封禁/解封）
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.InvalidParams(c, "用户 ID 无效，请刷新页面后重试")
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
		response.Error(c, response.CodeInvalidParams, "不能修改自己的账号状态")
		return
	}

	if err := h.userService.UpdateUserStatus(userID, req.Status); err != nil {
		response.InternalErrorWithErr(c, "更新用户状态失败，请稍后重试", err)
		return
	}

	response.SuccessEmpty(c)
}
