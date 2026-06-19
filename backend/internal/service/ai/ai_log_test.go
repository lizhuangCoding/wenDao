package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewAILoggerWithRotationWritesDailyJSONLog(t *testing.T) {
	dir := t.TempDir()

	logger, err := NewAILoggerWithRotation(dir, LogRotationConfig{
		MaxSizeMB:  1,
		MaxBackups: 2,
		MaxAgeDays: 3,
		Compress:   true,
	})
	if err != nil {
		t.Fatalf("expected AI logger to initialize, got %v", err)
	}

	logger.LogStage(AILogEntry{ConversationID: 12, UserID: 34, Stage: "testing", Message: "writes json"})
	if err := logger.Close(); err != nil {
		t.Fatalf("expected AI logger to close, got %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, time.Now().Format("2006-01-02")+"-ai-chat.log"))
	if err != nil {
		t.Fatalf("expected daily ai log file to exist, got %v", err)
	}
	content := string(data)
	for _, want := range []string{`"conversation_id":12`, `"user_id":34`, `"stage":"testing"`, `"message":"writes json"`} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected log content to include %s, got %s", want, content)
		}
	}
}

func TestAILoggerRedactsWebFetchContentAndFinalAnswer(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewAILoggerWithRotation(dir, LogRotationConfig{MaxSizeMB: 1})
	if err != nil {
		t.Fatalf("expected AI logger to initialize, got %v", err)
	}

	logger.LogStage(AILogEntry{
		Stage:   "tool_web_fetch_result",
		Message: "网页抓取结果详情",
		Detail:  "sensitive page body with private notes",
		Metadata: map[string]any{
			"url":     "https://example.com/private",
			"content": "full fetched content that should not be written",
		},
	})
	logger.LogStage(AILogEntry{
		Stage:   "completed",
		Message: "ThinkTank 计划执行流程完成",
		Detail:  "final answer with private user context",
	})
	if err := logger.Close(); err != nil {
		t.Fatalf("expected AI logger to close, got %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, time.Now().Format("2006-01-02")+"-ai-chat.log"))
	if err != nil {
		t.Fatalf("expected daily ai log file to exist, got %v", err)
	}
	content := string(data)
	for _, forbidden := range []string{"sensitive page body", "full fetched content", "final answer with private"} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("expected sensitive log content to be redacted, found %q in %s", forbidden, content)
		}
	}
	if !strings.Contains(content, "[redacted:") {
		t.Fatalf("expected redaction marker in log content, got %s", content)
	}
}
