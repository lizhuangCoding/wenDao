package site

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"wenDao/config"
)

func TestRobotsTxtUsesAbsoluteSitemapFromForwardedRequestWhenSiteURLMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewSiteHandler(&config.Config{}, nil, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "http://internal.local/robots.txt", nil)
	c.Request.Header.Set("X-Forwarded-Proto", "https")
	c.Request.Header.Set("X-Forwarded-Host", "blog.example.com")

	h.RobotsTxt(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Sitemap: https://blog.example.com/sitemap.xml") {
		t.Fatalf("expected absolute sitemap URL from forwarded host, got:\n%s", w.Body.String())
	}
}
