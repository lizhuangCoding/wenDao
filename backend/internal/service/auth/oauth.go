package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"wenDao/config"
)

var ErrGitHubOAuthConfigMissing = errors.New("github oauth config is incomplete")

// GitHubUserInfo GitHub 用户信息
type GitHubUserInfo struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// OAuthService OAuth 服务接口
type OAuthService interface {
	GetGitHubAuthURL(state string) string
	ExchangeGitHubCode(code string) (*GitHubUserInfo, error)
}

// oauthService OAuth 服务实现
type oauthService struct {
	cfg *config.Config
}

// NewOAuthService 创建 OAuth 服务实例
func NewOAuthService(cfg *config.Config) OAuthService {
	return &oauthService{cfg: cfg}
}

// GetGitHubAuthURL 生成 GitHub 授权 URL
func (s *oauthService) GetGitHubAuthURL(state string) string {
	if err := ValidateGitHubOAuthConfig(s.cfg); err != nil {
		return ""
	}

	params := url.Values{}
	params.Add("client_id", s.cfg.OAuth.GitHub.ClientID)
	params.Add("redirect_uri", s.cfg.OAuth.GitHub.CallbackURL)
	params.Add("scope", "user:email")
	params.Add("state", state)

	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

// ExchangeGitHubCode 用 code 换取 access token 并获取用户信息
func (s *oauthService) ExchangeGitHubCode(code string) (*GitHubUserInfo, error) {
	if err := ValidateGitHubOAuthConfig(s.cfg); err != nil {
		return nil, err
	}

	// 1. 用 code 换取 access_token
	tokenURL := "https://github.com/login/oauth/access_token"
	params := url.Values{}
	params.Add("client_id", s.cfg.OAuth.GitHub.ClientID)
	params.Add("client_secret", s.cfg.OAuth.GitHub.ClientSecret)
	params.Add("code", code)

	resp, err := http.PostForm(tokenURL, params)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 解析 access_token
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	accessToken := values.Get("access_token")
	if accessToken == "" {
		return nil, fmt.Errorf("no access token in response: %s", string(body))
	}

	// 2. 用 access_token 获取用户信息
	userInfo, err := s.getGitHubUserInfo(accessToken)
	if err != nil {
		return nil, err
	}

	// 3. 如果用户邮箱为空，获取主邮箱
	if userInfo.Email == "" {
		email, err := s.getGitHubPrimaryEmail(accessToken)
		if err == nil {
			userInfo.Email = email
		}
	}

	return userInfo, nil
}

// getGitHubUserInfo 获取 GitHub 用户信息
func (s *oauthService) getGitHubUserInfo(accessToken string) (*GitHubUserInfo, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var userInfo GitHubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// getGitHubPrimaryEmail 获取 GitHub 主邮箱
func (s *oauthService) getGitHubPrimaryEmail(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no primary email found")
}

func ValidateGitHubOAuthConfig(cfg *config.Config) error {
	if cfg == nil || cfg.OAuth.GitHub.ClientID == "" || cfg.OAuth.GitHub.ClientSecret == "" || cfg.OAuth.GitHub.CallbackURL == "" {
		return ErrGitHubOAuthConfigMissing
	}
	return nil
}
