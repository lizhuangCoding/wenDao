package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/config"
	"wenDao/internal/model"
	pkgjwt "wenDao/internal/pkg/jwt"
)

type stubUserService struct {
	refreshToken       string
	refreshTokenErr    error
	refreshTokenUserID int64
	refreshTokenRole   string
}

func (s *stubUserService) Register(email, password, username string) (*model.User, error) {
	return nil, nil
}

func (s *stubUserService) Login(email, password string) (string, *model.User, error) {
	return "", nil, nil
}

func (s *stubUserService) EmailExists(email string) (bool, error) {
	return false, nil
}

func (s *stubUserService) ResetPassword(email, password string) error {
	return nil
}

func (s *stubUserService) GitHubOAuthLogin(code string) (string, *model.User, error) {
	return "", nil, nil
}

func (s *stubUserService) Logout(token string) error {
	return nil
}

func (s *stubUserService) GetCurrentUser(userID int64) (*model.User, error) {
	return nil, nil
}

func (s *stubUserService) GenerateRefreshToken(userID int64, role string) (string, error) {
	s.refreshTokenUserID = userID
	s.refreshTokenRole = role
	return s.refreshToken, s.refreshTokenErr
}

func (s *stubUserService) UpdateAvatar(userID int64, avatarURL string) error {
	return nil
}

func (s *stubUserService) UpdateUsername(userID int64, username string) error {
	return nil
}

func (s *stubUserService) UpdateCommentReplyEmailEnabled(userID int64, enabled bool) error {
	return nil
}

func (s *stubUserService) ListUsers(page, pageSize int, role, status, search string) ([]*model.User, int64, error) {
	return nil, 0, nil
}

func (s *stubUserService) UpdateUserRole(userID int64, role string) error {
	return nil
}

func (s *stubUserService) UpdateUserStatus(userID int64, status string) error {
	return nil
}

func (s *stubUserService) GetAllActiveUserIDs() ([]int64, error) {
	return nil, nil
}

func TestAuthHandlerLogoutClearsAccessAndRefreshCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Server: config.ServerConfig{Mode: "release"},
		Site:   config.SiteConfig{URL: "https://wendao.example.com"},
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpireHours: 1,
			RefreshExpireDays: 7,
		},
	}
	accessToken, err := pkgjwt.GenerateAccessToken(9, "user", cfg.JWT.Secret, cfg.JWT.AccessExpireHours)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	refreshToken, err := pkgjwt.GenerateRefreshToken(9, "user", cfg.JWT.Secret, cfg.JWT.RefreshExpireDays)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	h := NewAuthHandler(nil, cfg, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	c.Request.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	c.Set("token", accessToken)

	h.Logout(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}

	tokenCookie := findCookieByName(t, w.Result(), "token")
	if tokenCookie.MaxAge >= 0 || tokenCookie.Value != "" {
		t.Fatalf("expected token cookie to be expired, got value=%q maxAge=%d", tokenCookie.Value, tokenCookie.MaxAge)
	}
	if !tokenCookie.Secure {
		t.Fatalf("expected token deletion cookie to preserve secure policy")
	}

	refreshCookie := findCookieByName(t, w.Result(), "refresh_token")
	if refreshCookie.MaxAge >= 0 || refreshCookie.Value != "" {
		t.Fatalf("expected refresh cookie to be expired, got value=%q maxAge=%d", refreshCookie.Value, refreshCookie.MaxAge)
	}
	if !refreshCookie.Secure {
		t.Fatalf("expected refresh deletion cookie to preserve secure policy")
	}
}

func TestAuthHandlerRefreshRotatesAccessAndRefreshCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Server: config.ServerConfig{Mode: "release"},
		Site:   config.SiteConfig{URL: "https://wendao.example.com"},
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpireHours: 1,
			RefreshExpireDays: 7,
		},
	}
	oldRefreshToken, err := pkgjwt.GenerateRefreshToken(18, "user", cfg.JWT.Secret, cfg.JWT.RefreshExpireDays)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	userService := &stubUserService{refreshToken: "rotated-refresh-token"}
	h := NewAuthHandler(userService, cfg, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	c.Request.AddCookie(&http.Cookie{Name: "refresh_token", Value: oldRefreshToken})

	h.Refresh(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d with body %s", w.Code, w.Body.String())
	}
	if userService.refreshTokenUserID != 18 {
		t.Fatalf("expected refresh token user id 18, got %d", userService.refreshTokenUserID)
	}
	if userService.refreshTokenRole != "user" {
		t.Fatalf("expected refresh token role user, got %q", userService.refreshTokenRole)
	}

	var payload struct {
		Code int `json:"code"`
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Code != 0 {
		t.Fatalf("expected success code 0, got %d", payload.Code)
	}
	if payload.Data.AccessToken == "" {
		t.Fatal("expected response access token to be returned")
	}

	tokenCookie := findCookieByName(t, w.Result(), "token")
	if tokenCookie.Value != payload.Data.AccessToken {
		t.Fatalf("expected token cookie to match response access token")
	}
	if !tokenCookie.HttpOnly {
		t.Fatalf("expected access token cookie to be httpOnly")
	}

	refreshCookie := findCookieByName(t, w.Result(), "refresh_token")
	if refreshCookie.Value != "rotated-refresh-token" {
		t.Fatalf("expected rotated refresh token cookie, got %q", refreshCookie.Value)
	}
	if !refreshCookie.HttpOnly {
		t.Fatalf("expected refresh token cookie to be httpOnly")
	}

	csrfCookie := findCookieByName(t, w.Result(), "csrf_token")
	if csrfCookie.Value == "" {
		t.Fatal("expected csrf token cookie to be set")
	}
	if csrfCookie.HttpOnly {
		t.Fatal("expected csrf token cookie to be readable by frontend")
	}
}

func findCookieByName(t *testing.T, res *http.Response, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range res.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %q not found", name)
	return nil
}
