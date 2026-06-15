package ai

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// AILogEntry AI 聊天结构化日志条目
type AILogEntry struct {
	Timestamp      string         `json:"timestamp"`
	ConversationID int64          `json:"conversation_id"`
	UserID         int64          `json:"user_id"`
	RunID          int64          `json:"run_id,omitempty"`
	Stage          string         `json:"stage"`
	Message        string         `json:"message"`
	Detail         string         `json:"detail,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// AILogger AI 聊天日志接口
type AILogger interface {
	LogStage(entry AILogEntry)
	LogError(entry AILogEntry)
	Close() error
}

// LogRotationConfig 控制 AI 聊天日志的大小轮转和历史保留。
type LogRotationConfig struct {
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

type aiLogger struct {
	mu      sync.Mutex
	writer  io.WriteCloser
	encoder *json.Encoder
}

// NewAILogger 创建独立 ai-chat 日志器
func NewAILogger(logDir string) (AILogger, error) {
	return NewAILoggerWithRotation(logDir, LogRotationConfig{
		MaxSizeMB:  100,
		MaxBackups: 7,
		MaxAgeDays: 10,
		Compress:   true,
	})
}

// NewAILoggerWithRotation 创建带轮转策略的独立 ai-chat 日志器。
func NewAILoggerWithRotation(logDir string, rotation LogRotationConfig) (AILogger, error) {
	if logDir == "" {
		logDir = "log"
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}
	if rotation.MaxSizeMB <= 0 {
		rotation.MaxSizeMB = 100
	}
	if rotation.MaxBackups <= 0 {
		rotation.MaxBackups = 7
	}
	if rotation.MaxAgeDays <= 0 {
		rotation.MaxAgeDays = 28
	}
	filePath := filepath.Join(logDir, time.Now().Format("2006-01-02")+"-ai-chat.log")
	writer := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    rotation.MaxSizeMB,
		MaxBackups: rotation.MaxBackups,
		MaxAge:     rotation.MaxAgeDays,
		Compress:   rotation.Compress,
	}
	return &aiLogger{writer: writer, encoder: json.NewEncoder(writer)}, nil
}

func (l *aiLogger) LogStage(entry AILogEntry) {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format("2006-01-02 15:04:05.000")
	}
	l.write(entry)
}

func (l *aiLogger) LogError(entry AILogEntry) {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().Format("2006-01-02 15:04:05.000")
	}
	l.write(entry)
}

func (l *aiLogger) write(entry AILogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = l.encoder.Encode(entry)
}

func (l *aiLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.writer == nil {
		return nil
	}
	return l.writer.Close()
}
