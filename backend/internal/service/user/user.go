package user

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"wenDao/config"
	"wenDao/internal/model"
	"wenDao/internal/pkg/hash"
	pkgjwt "wenDao/internal/pkg/jwt"
	"wenDao/internal/repository"
)

// UserService 用户服务接口
type UserService interface {
	Register(email, password, username string) (*model.User, error)
	Login(email, password string) (string, *model.User, error)
	GitHubOAuthLogin(code string) (string, *model.User, error)
	Logout(token string) error
	GetCurrentUser(userID int64) (*model.User, error)
	GenerateRefreshToken(userID int64, role string) (string, error)
	UpdateAvatar(userID int64, avatarURL string) error
	UpdateUsername(userID int64, username string) error
}

// userService 用户服务实现
type userService struct {
	userRepo     repository.UserRepository
	oauthService OAuthService
	cfg          *config.Config
	rdb          *redis.Client
}

// NewUserService 创建用户服务实例
func NewUserService(
	userRepo repository.UserRepository,
	oauthService OAuthService,
	cfg *config.Config,
	rdb *redis.Client,
) UserService {
	return &userService{
		userRepo:     userRepo,
		oauthService: oauthService,
		cfg:          cfg,
		rdb:          rdb,
	}
}

// Register 用户注册
func (s *userService) Register(email, password, username string) (*model.User, error) {
	// 检查邮箱是否已存在
	existingUser, err := s.userRepo.GetByEmail(email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if existingUser != nil {
		return nil, errors.New("email already exists")
	}

	// 加密密码
	passwordHash, err := hash.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 设置默认头像（使用 DiceBear 生成动态头像）
	defaultAvatar := fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", username)

	// 创建用户
	user := &model.User{
		Email:        email,
		Username:     username,
		PasswordHash: &passwordHash,
		AvatarURL:    &defaultAvatar,
		AvatarSource: model.AvatarSourceDefault,
		Status:       "active",
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login 用户登录
func (s *userService) Login(email, password string) (string, *model.User, error) {
	// 查询用户
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errors.New("invalid email or password")
		}
		return "", nil, fmt.Errorf("failed to get user: %w", err)
	}

	// 检查账号状态
	if !user.IsActive() {
		return "", nil, errors.New("account is banned")
	}

	// 验证密码
	if user.PasswordHash == nil || !hash.CheckPassword(password, *user.PasswordHash) {
		return "", nil, errors.New("invalid email or password")
	}

	// 生成 JWT token
	token, err := pkgjwt.GenerateAccessToken(user.ID, user.Role, s.cfg.JWT.Secret, s.cfg.JWT.AccessExpireHours)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return token, user, nil
}

// GitHubOAuthLogin GitHub OAuth 登录
func (s *userService) GitHubOAuthLogin(code string) (string, *model.User, error) {
	// 用 code 换取用户信息
	githubUser, err := s.oauthService.ExchangeGitHubCode(code)
	if err != nil {
		return "", nil, fmt.Errorf("failed to exchange github code: %w", err)
	}

	// 查询用户是否已存在
	oauthID := strconv.FormatInt(githubUser.ID, 10)
	user, err := s.userRepo.GetByOAuth("github", oauthID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil, fmt.Errorf("failed to get user by oauth: %w", err)
	}

	// 如果用户不存在，创建新用户
	if user == nil {
		if githubUser.Email == "" {
			return "", nil, errors.New("github email is required")
		}

		existingByEmail, err := s.userRepo.GetByEmail(githubUser.Email)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, fmt.Errorf("failed to check github email: %w", err)
		}
		if existingByEmail != nil {
			return "", nil, errors.New("email already exists")
		}

		username, err := s.resolveGitHubUsername(githubUser.Login, githubUser.ID)
		if err != nil {
			return "", nil, err
		}

		provider := "github"
		user = &model.User{
			Username:      username,
			Email:         githubUser.Email,
			Role:          "user",
			Status:        "active",
			OAuthProvider: &provider,
			OAuthID:       &oauthID,
			AvatarURL:     &githubUser.AvatarURL,
			AvatarSource:  model.AvatarSourceGitHub,
		}

		if err := s.userRepo.Create(user); err != nil {
			return "", nil, fmt.Errorf("failed to create user: %w", err)
		}
	} else if shouldSyncGitHubAvatar(user.AvatarSource) {
		user.AvatarURL = &githubUser.AvatarURL
		user.AvatarSource = model.AvatarSourceGitHub
		if err := s.userRepo.Update(user); err != nil {
			return "", nil, fmt.Errorf("failed to update user github avatar: %w", err)
		}
	}

	// 检查账号状态
	if !user.IsActive() {
		return "", nil, errors.New("account is banned")
	}

	// 生成 JWT token
	token, err := pkgjwt.GenerateAccessToken(user.ID, user.Role, s.cfg.JWT.Secret, s.cfg.JWT.AccessExpireHours)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return token, user, nil
}

func (s *userService) resolveGitHubUsername(login string, githubID int64) (string, error) {
	if login == "" {
		login = "github-user"
	}

	_, err := s.userRepo.GetByUsername(login)
	if err == nil {
		suffix := strconv.FormatInt(githubID, 10)
		if len(suffix) > 6 {
			suffix = suffix[len(suffix)-6:]
		}
		fallback := fmt.Sprintf("%s-%s", trimUsernameBase(login, 50-len(suffix)-1), suffix)
		_, err = s.userRepo.GetByUsername(fallback)
		if err == nil {
			return "", errors.New("github username already exists")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("failed to check github username: %w", err)
		}
		return fallback, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("failed to check github username: %w", err)
	}
	return login, nil
}

func trimUsernameBase(base string, maxLen int) string {
	if maxLen <= 0 {
		return "github"
	}
	if len(base) <= maxLen {
		return base
	}
	return base[:maxLen]
}

func shouldSyncGitHubAvatar(avatarSource string) bool {
	return avatarSource == "" || avatarSource == model.AvatarSourceDefault || avatarSource == model.AvatarSourceGitHub
}

// Logout 用户登出（将 token 加入黑名单）
func (s *userService) Logout(token string) error {
	// 解析 token 获取过期时间
	claims, err := pkgjwt.ParseToken(token, s.cfg.JWT.Secret)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// 计算剩余有效期
	expireTime := time.Until(claims.ExpiresAt.Time)
	if expireTime <= 0 {
		// token 已过期，无需加入黑名单
		return nil
	}

	// 将 token 加入黑名单
	if err := pkgjwt.AddToBlacklist(s.rdb, token, expireTime); err != nil {
		return fmt.Errorf("failed to add token to blacklist: %w", err)
	}

	return nil
}

// GetCurrentUser 获取当前用户信息
func (s *userService) GetCurrentUser(userID int64) (*model.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GenerateRefreshToken 生成 Refresh Token
func (s *userService) GenerateRefreshToken(userID int64, role string) (string, error) {
	return pkgjwt.GenerateRefreshToken(userID, role, s.cfg.JWT.Secret, s.cfg.JWT.RefreshExpireDays)
}

// UpdateAvatar 更新用户头像
func (s *userService) UpdateAvatar(userID int64, avatarURL string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	user.AvatarURL = &avatarURL
	user.AvatarSource = model.AvatarSourceCustom
	return s.userRepo.Update(user)
}

// UpdateUsername 更新用户名
func (s *userService) UpdateUsername(userID int64, username string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	// 检查用户名是否已存在
	existing, err := s.userRepo.GetByUsername(username)
	if err == nil && existing.ID != userID {
		return errors.New("username already exists")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	user.Username = username
	return s.userRepo.Update(user)
}
