package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerSkipsFastSuccessRequestsByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	router := gin.New()
	router.Use(Logger(logger, "warn"))
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	if logs.Len() != 0 {
		t.Fatalf("expected no access log for fast successful request at warn level, got %d entries", logs.Len())
	}
}

func TestLoggerCanEmitAccessLogsForFastSuccessRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	router := gin.New()
	router.Use(Logger(logger, "info"))
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	if logs.Len() != 1 {
		t.Fatalf("expected one access log entry for successful request at info level, got %d", logs.Len())
	}
	entry := logs.All()[0]
	if entry.Level != zapcore.InfoLevel {
		t.Fatalf("expected info-level access log, got %s", entry.Level)
	}
	if entry.Message != "Request completed" {
		t.Fatalf("expected success access log message, got %q", entry.Message)
	}
	if got := entry.ContextMap()["status"]; got != int64(http.StatusNoContent) {
		t.Fatalf("expected status field to be present, got %#v", got)
	}
}
