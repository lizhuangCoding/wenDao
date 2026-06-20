package httpcookie

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"wenDao/config"
)

const CSRFTokenCookieName = "csrf_token"

func ShouldUseSecureCookies(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}

	siteURL := strings.TrimSpace(cfg.Site.URL)
	if siteURL == "" {
		return cfg.Server.Mode == "release"
	}

	parsed, err := url.Parse(siteURL)
	if err != nil {
		return cfg.Server.Mode == "release"
	}
	return strings.EqualFold(parsed.Scheme, "https")
}

// SetAuthCookies sets both access_token and refresh_token cookies on the response.
func SetAuthCookies(c *gin.Context, cfg *config.Config, token, refreshToken string) {
	secure := ShouldUseSecureCookies(cfg)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("token", token, cfg.JWT.AccessExpireHours*3600, "/", "", secure, true)
	c.SetCookie("refresh_token", refreshToken, cfg.JWT.RefreshExpireDays*24*3600, "/", "", secure, true)
	SetCSRFCookie(c, cfg)
}

// ClearAuthCookies clears both access_token and refresh_token cookies.
func ClearAuthCookies(c *gin.Context, cfg *config.Config) {
	secure := ShouldUseSecureCookies(cfg)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("token", "", -1, "/", "", secure, true)
	c.SetCookie("refresh_token", "", -1, "/", "", secure, true)
	c.SetCookie(CSRFTokenCookieName, "", -1, "/", "", secure, false)
}

func SetCSRFCookie(c *gin.Context, cfg *config.Config) string {
	token := generateCSRFToken()
	secure := ShouldUseSecureCookies(cfg)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(CSRFTokenCookieName, token, cfg.JWT.RefreshExpireDays*24*3600, "/", "", secure, false)
	return token
}

// SetOAuthStateCookie sets the oauth_state cookie for CSRF protection during OAuth flows.
func SetOAuthStateCookie(c *gin.Context, cfg *config.Config, state string) {
	secure := ShouldUseSecureCookies(cfg)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("oauth_state", state, 600, "/", "", secure, true)
}

// ClearOAuthStateCookie clears the oauth_state cookie.
func ClearOAuthStateCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)
}

func generateCSRFToken() string {
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(token[:])
}
