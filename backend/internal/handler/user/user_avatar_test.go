package user

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/config"
	"wenDao/internal/model"
	pkgjwt "wenDao/internal/pkg/jwt"
	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
)

type stubUserService struct {
	currentUser        *model.User
	getCurrentUserErr  error
	updateAvatarUserID int64
	updateAvatarURL    string
	updateAvatarErr    error
	updateUsernameID   int64
	updateUsername     string
	updateUsernameErr  error
	refreshToken       string
	refreshTokenErr    error
	refreshTokenUserID int64
	refreshTokenRole   string
	githubLoginCode    string
	githubLoginToken   string
	githubLoginUser    *model.User
	githubLoginErr     error
}

func (s *stubUserService) Register(email, password, username string) (*model.User, error) {
	return nil, nil
}

func (s *stubUserService) Login(email, password string) (string, *model.User, error) {
	return "", nil, nil
}

func (s *stubUserService) GitHubOAuthLogin(code string) (string, *model.User, error) {
	s.githubLoginCode = code
	if s.githubLoginErr != nil {
		return "", nil, s.githubLoginErr
	}
	return s.githubLoginToken, s.githubLoginUser, nil
}

func (s *stubUserService) Logout(token string) error {
	return nil
}

func (s *stubUserService) GetCurrentUser(userID int64) (*model.User, error) {
	if s.getCurrentUserErr != nil {
		return nil, s.getCurrentUserErr
	}
	return s.currentUser, nil
}

func (s *stubUserService) GenerateRefreshToken(userID int64, role string) (string, error) {
	s.refreshTokenUserID = userID
	s.refreshTokenRole = role
	if s.refreshTokenErr != nil {
		return "", s.refreshTokenErr
	}
	return s.refreshToken, nil
}

func (s *stubUserService) UpdateAvatar(userID int64, avatarURL string) error {
	s.updateAvatarUserID = userID
	s.updateAvatarURL = avatarURL
	return s.updateAvatarErr
}

func (s *stubUserService) UpdateUsername(userID int64, username string) error {
	s.updateUsernameID = userID
	s.updateUsername = username
	return s.updateUsernameErr
}

type stubUploadService struct {
	receivedUserID   int64
	receivedFilename string
	receivedBody     string
	upload           *model.Upload
	err              error
	cleanupFilePath  string
	cleanupErr       error
}

func (s *stubUploadService) UploadImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	body, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	s.receivedUserID = userID
	s.receivedFilename = header.Filename
	s.receivedBody = string(body)
	if s.err != nil {
		return nil, s.err
	}
	return s.upload, nil
}

func (s *stubUploadService) UploadArticleImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	return s.UploadImage(file, header, userID)
}

func (s *stubUploadService) UploadCoverImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	return s.UploadImage(file, header, userID)
}

func (s *stubUploadService) CleanupByFilePath(filePath string) error {
	s.cleanupFilePath = filePath
	return s.cleanupErr
}

type stubOAuthService struct{}

func (s *stubOAuthService) GetGitHubAuthURL(state string) string {
	return ""
}

func (s *stubOAuthService) ExchangeGitHubCode(code string) (*service.GitHubUserInfo, error) {
	return nil, nil
}

func TestAuthHandlerGetUserInfo_IncludesAvatarURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	avatarURL := "/uploads/avatar.png"
	passwordHash := "secret"
	userService := &stubUserService{
		currentUser: &model.User{
			ID:           42,
			Email:        "user@example.com",
			Username:     "tester",
			Role:         "user",
			AvatarURL:    &avatarURL,
			PasswordHash: &passwordHash,
			Status:       "active",
			CreatedAt:    time.Unix(1700000000, 0),
			UpdatedAt:    time.Unix(1700000300, 0),
		},
	}
	h := NewAuthHandler(userService, &config.Config{JWT: config.JWTConfig{AccessExpireHours: 2}}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	c.Set("user_id", int64(42))

	h.GetUserInfo(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object data, got %T", resp.Data)
	}

	if got := int(data["expires_in"].(float64)); got != 7200 {
		t.Fatalf("expected expires_in 7200, got %d", got)
	}

	userData, ok := data["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested user object, got %T", data["user"])
	}

	if got := userData["avatar_url"]; got != avatarURL {
		t.Fatalf("expected avatar_url %q, got %#v", avatarURL, got)
	}

	if _, exists := userData["password_hash"]; exists {
		t.Fatalf("expected password_hash to be omitted from response")
	}
}

func TestAuthHandlerRefresh_ReturnsInternalErrorWhenRefreshRotationFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpireHours: 1,
			RefreshExpireDays: 7,
		},
	}
	refreshToken, err := pkgjwt.GenerateRefreshToken(99, "user", cfg.JWT.Secret, cfg.JWT.RefreshExpireDays)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	userService := &stubUserService{refreshTokenErr: errors.New("refresh failed")}
	h := NewAuthHandler(userService, cfg, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(`{"refresh_token":"`+refreshToken+`"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Refresh(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusInternalServerError, w.Code, w.Body.String())
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code != response.CodeInternalError {
		t.Fatalf("expected response code %d, got %d", response.CodeInternalError, resp.Code)
	}
}

func TestUserHandlerUploadAvatar_UpdatesAvatarAndReturnsUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	avatarURL := "/uploads/avatars/new.png"
	userService := &stubUserService{
		currentUser: &model.User{
			ID:        7,
			Email:     "avatar@example.com",
			Username:  "avatar-user",
			Role:      "user",
			AvatarURL: &avatarURL,
			Status:    "active",
		},
	}
	uploadService := &stubUploadService{
		upload: &model.Upload{
			FilePath: avatarURL,
		},
	}
	h := NewUserHandler(userService, uploadService, &stubOAuthService{}, &config.Config{})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("avatar-image-content")); err != nil {
		t.Fatalf("failed to write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/users/me/avatar", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("user_id", int64(7))

	h.UploadAvatar(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if uploadService.receivedUserID != 7 {
		t.Fatalf("expected upload service to receive user id 7, got %d", uploadService.receivedUserID)
	}

	if uploadService.receivedFilename != "avatar.png" {
		t.Fatalf("expected upload filename avatar.png, got %q", uploadService.receivedFilename)
	}

	if uploadService.receivedBody != "avatar-image-content" {
		t.Fatalf("expected uploaded body to be forwarded, got %q", uploadService.receivedBody)
	}

	if userService.updateAvatarUserID != 7 {
		t.Fatalf("expected UpdateAvatar user id 7, got %d", userService.updateAvatarUserID)
	}

	if userService.updateAvatarURL != avatarURL {
		t.Fatalf("expected UpdateAvatar url %q, got %q", avatarURL, userService.updateAvatarURL)
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected user object response, got %T", resp.Data)
	}

	if got := data["avatar_url"]; got != avatarURL {
		t.Fatalf("expected avatar_url %q, got %#v", avatarURL, got)
	}
}

func TestUserHandlerUploadAvatar_CleansUpUploadWhenAvatarUpdateFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	uploadPath := "/uploads/avatars/broken.png"
	userService := &stubUserService{updateAvatarErr: errors.New("db write failed")}
	uploadService := &stubUploadService{
		upload: &model.Upload{FilePath: uploadPath},
	}
	h := NewUserHandler(userService, uploadService, &stubOAuthService{}, &config.Config{})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("avatar-image-content")); err != nil {
		t.Fatalf("failed to write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/users/me/avatar", body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set("user_id", int64(7))

	h.UploadAvatar(c)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	if uploadService.cleanupFilePath != uploadPath {
		t.Fatalf("expected cleanup for %q, got %q", uploadPath, uploadService.cleanupFilePath)
	}
}

func TestUserHandlerGitHubCallback_SetsCookiesAndRedirectsWithoutTokenQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userService := &stubUserService{
		githubLoginToken: "access-token-value",
		githubLoginUser: &model.User{
			ID:       12,
			Username: "octocat",
			Role:     "user",
			Status:   "active",
		},
		refreshToken: "refresh-token-value",
	}
	cfg := &config.Config{
		Server: config.ServerConfig{Mode: "release"},
		JWT: config.JWTConfig{
			AccessExpireHours: 1,
			RefreshExpireDays: 7,
		},
		Site: config.SiteConfig{URL: "https://frontend.example.com"},
	}
	h := NewUserHandler(userService, &stubUploadService{}, &stubOAuthService{}, cfg)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/github/callback?code=oauth-code&state=expected-state", nil)
	c.Request.AddCookie(&http.Cookie{Name: "oauth_state", Value: "expected-state"})

	h.GitHubCallback(c)

	if w.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, w.Code)
	}

	location := w.Result().Header.Get("Location")
	if location != cfg.Site.URL {
		t.Fatalf("expected redirect location %q, got %q", cfg.Site.URL, location)
	}
	if strings.Contains(location, "token=") {
		t.Fatalf("expected redirect location without token query, got %q", location)
	}

	tokenCookie := findCookieByName(t, w.Result(), "token")
	if tokenCookie.Value != "access-token-value" {
		t.Fatalf("expected token cookie value to be set")
	}
	if !tokenCookie.Secure {
		t.Fatalf("expected token cookie to be secure in release mode")
	}

	refreshCookie := findCookieByName(t, w.Result(), "refresh_token")
	if refreshCookie.Value != "refresh-token-value" {
		t.Fatalf("expected refresh token cookie value to be set")
	}
	if !refreshCookie.Secure {
		t.Fatalf("expected refresh token cookie to be secure in release mode")
	}

	if userService.refreshTokenUserID != 12 {
		t.Fatalf("expected refresh token user id 12, got %d", userService.refreshTokenUserID)
	}
	if userService.refreshTokenRole != "user" {
		t.Fatalf("expected refresh token role user, got %q", userService.refreshTokenRole)
	}
}

func TestUserHandlerGitHubLogin_UsesSecureStateCookieInReleaseMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewUserHandler(&stubUserService{}, &stubUploadService{}, &stubOAuthService{}, &config.Config{
		Server: config.ServerConfig{Mode: "release"},
		OAuth: config.OAuthConfig{GitHub: config.GitHubOAuthConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			CallbackURL:  "http://localhost:8089/api/auth/github/callback",
		}},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/github", nil)

	h.GitHubLogin(c)

	if w.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, w.Code)
	}

	stateCookie := findCookieByName(t, w.Result(), "oauth_state")
	if !stateCookie.Secure {
		t.Fatalf("expected oauth_state cookie to be secure in release mode")
	}
}

func TestUserHandlerGitHubLogin_ReturnsBadRequestWhenConfigMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewUserHandler(&stubUserService{}, &stubUploadService{}, &stubOAuthService{}, &config.Config{
		OAuth: config.OAuthConfig{GitHub: config.GitHubOAuthConfig{}},
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/github", nil)

	h.GitHubLogin(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusBadRequest, w.Code, w.Body.String())
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != response.CodeInvalidParams {
		t.Fatalf("expected invalid params code, got %d", resp.Code)
	}
	if !strings.Contains(strings.ToLower(resp.Message), "github oauth") {
		t.Fatalf("expected clear github oauth config message, got %q", resp.Message)
	}
}

func TestAuthHandlerRefresh_SucceedsWithoutRedisClient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpireHours: 1,
			RefreshExpireDays: 7,
		},
	}
	refreshToken, err := pkgjwt.GenerateRefreshToken(99, "user", cfg.JWT.Secret, cfg.JWT.RefreshExpireDays)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	userService := &stubUserService{refreshToken: "rotated-refresh-token"}
	h := NewAuthHandler(userService, cfg, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(`{"refresh_token":"`+refreshToken+`"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected no panic when redis client is nil, got %v", r)
		}
	}()

	h.Refresh(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusOK, w.Code, w.Body.String())
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
