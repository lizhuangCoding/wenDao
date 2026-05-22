package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/config"
	pkgjwt "wenDao/internal/pkg/jwt"
)

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
