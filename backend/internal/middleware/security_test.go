package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersAddsBrowserProtectionHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	headers := w.Result().Header
	if headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", headers.Get("X-Content-Type-Options"))
	}
	if headers.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("expected DENY frame header, got %q", headers.Get("X-Frame-Options"))
	}
	if headers.Get("Referrer-Policy") == "" {
		t.Fatalf("expected referrer policy header")
	}
	if headers.Get("Permissions-Policy") == "" {
		t.Fatalf("expected permissions policy header")
	}
}
