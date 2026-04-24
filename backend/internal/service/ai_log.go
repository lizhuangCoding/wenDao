package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
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

type aiLogger struct {
	mu      sync.Mutex
	file    *os.File
	encoder *json.Encoder
}

// NewAILogger 创建独立 ai-chat 日志器
func NewAILogger(logDir string) (AILogger, error) {
	if logDir == "" {
		logDir = "log"
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}
	filePath := filepath.Join(logDir, time.Now().Format("2006-01-02")+"-ai-chat.log")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &aiLogger{file: file, encoder: json.NewEncoder(file)}, nil
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
	if l.file == nil {
		return nil
	}
	return l.file.Close()
}
