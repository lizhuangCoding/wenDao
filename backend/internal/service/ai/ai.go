package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"wenDao/internal/pkg/eino"
	chatcore "wenDao/internal/service/chatcore"
)

var ErrAIDisabled = errors.New("ai service is unavailable")

// AIService AI 服务接口
type AIService interface {
	// Chat AI 对话
	Chat(question string, conversationID *int64, userID *int64) (string, error)
	// ChatStream AI 流式对话
	ChatStream(ctx context.Context, question string, conversationID *int64, userID *int64) (<-chan chatcore.StreamEvent, <-chan error)
	// GenerateSummary 生成文章摘要
	GenerateSummary(content string) (string, error)
}

// aiService AI 服务实现
type aiService struct {
	thinkTank chatcore.ThinkTankService
	llmClient eino.LLMClient
	logger    *zap.Logger
}

type disabledAIService struct {
	reason string
}

// NewAIService 创建 AI 服务实例
func NewAIService(llmClient eino.LLMClient, thinkTank chatcore.ThinkTankService, logger *zap.Logger) AIService {
	return &aiService{
		thinkTank: thinkTank,
		llmClient: llmClient,
		logger:    logger,
	}
}

func NewDisabledAIService(reason string) AIService {
	return &disabledAIService{reason: strings.TrimSpace(reason)}
}

func buildConversationTitle(question string) string {
	title := strings.TrimSpace(question)
	runes := []rune(title)
	if len(runes) > 30 {
		return string(runes[:30]) + "..."
	}
	return title
}

// Chat AI 对话
func (s *aiService) Chat(question string, conversationID *int64, userID *int64) (string, error) {
	resp, err := s.thinkTank.Chat(context.Background(), question, conversationID, userID)
	if err != nil {
		return "", err
	}
	return resp.Message, nil
}

// ChatStream AI 流式对话
func (s *aiService) ChatStream(ctx context.Context, question string, conversationID *int64, userID *int64) (<-chan chatcore.StreamEvent, <-chan error) {
	return s.thinkTank.ChatStream(ctx, question, conversationID, userID)
}

// GenerateSummary 生成文章摘要
func (s *aiService) GenerateSummary(content string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("article content is empty")
	}

	runes := []rune(content)
	if len(runes) > 2000 {
		content = string(runes[:2000])
	}

	prompt := fmt.Sprintf(`你是一个专业的文章摘要生成助手。请阅读以下文章内容，并生成一个简洁、专业、吸引人的摘要。
要求：
1. 摘要字数在 100-200 字之间。
2. 准确概括文章的核心观点。
3. 语言流畅，适合作为博客文章的简述。
4. 直接返回摘要内容，不要包含任何前缀（如“摘要：”或“这篇文章讲述了...”）。

文章内容：
%s`, content)

	messages := []eino.ChatMessage{
		{Role: "system", Content: "你是一个专业的博主助手，擅长撰写高质量的文章摘要。"},
		{Role: "user", Content: prompt},
	}

	summary, err := s.llmClient.Chat(messages)
	if err != nil {
		s.logger.Error("Failed to generate summary", zap.Error(err))
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	return strings.TrimSpace(summary), nil
}

func (s *disabledAIService) Chat(question string, conversationID *int64, userID *int64) (string, error) {
	return "", s.err()
}

func (s *disabledAIService) ChatStream(ctx context.Context, question string, conversationID *int64, userID *int64) (<-chan chatcore.StreamEvent, <-chan error) {
	eventCh := make(chan chatcore.StreamEvent)
	errCh := make(chan error, 1)
	errCh <- s.err()
	close(eventCh)
	close(errCh)
	return eventCh, errCh
}

func (s *disabledAIService) GenerateSummary(content string) (string, error) {
	return "", s.err()
}

func (s *disabledAIService) err() error {
	if s.reason == "" {
		return ErrAIDisabled
	}
	return fmt.Errorf("%w: %s", ErrAIDisabled, s.reason)
}
