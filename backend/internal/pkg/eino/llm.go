package eino

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/ark"
	deepseek "github.com/cloudwego/eino-ext/components/model/deepseek"
	openai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"wenDao/config"
)

const chatStreamBufferSize = 1

// LLMClient LLM 客户端接口
type LLMClient interface {
	// Chat 对话生成
	Chat(ctx context.Context, messages []ChatMessage) (string, error)
	// ChatStream 流式对话生成（返回累计文本快照）
	ChatStream(ctx context.Context, messages []ChatMessage) (<-chan string, <-chan error)
	// GetModel 获取原始 Eino 模型
	GetModel() model.ChatModel
}

// ChatMessage 对话消息
type ChatMessage struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// chatModelClient LLM 实现（使用 eino 框架）
type chatModelClient struct {
	client      model.ChatModel
	temperature float32
	maxTokens   int
}

// NewLLMClient 创建可切换 provider 的 LLM 客户端
func NewLLMClient(cfg *config.AIConfig) (LLMClient, error) {
	return newLLMClient(cfg)
}

func newLLMClient(cfg *config.AIConfig) (LLMClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("ai config is required")
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	provider := normalizeAIProvider(cfg.Provider)

	client, err := buildChatModel(provider, cfg)
	if err != nil {
		return nil, err
	}

	return &chatModelClient{
		client:      client,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
	}, nil
}

func normalizeAIProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func buildChatModel(provider string, cfg *config.AIConfig) (model.ChatModel, error) {
	switch provider {
	case "", "doubao", "ark":
		chatConfig := &ark.ChatModelConfig{
			APIKey: cfg.APIKey,
			Model:  cfg.LLMModel,
		}
		if endpoint := strings.TrimSpace(cfg.Endpoint); endpoint != "" {
			chatConfig.BaseURL = endpoint
		}

		client, err := ark.NewChatModel(context.Background(), chatConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create doubao chat model: %w", err)
		}
		return client, nil
	case "deepseek":
		chatConfig := &deepseek.ChatModelConfig{
			APIKey:      cfg.APIKey,
			Model:       cfg.LLMModel,
			Temperature: cfg.Temperature,
			MaxTokens:   cfg.MaxTokens,
		}
		if endpoint := strings.TrimSpace(cfg.Endpoint); endpoint != "" {
			chatConfig.BaseURL = endpoint
		}

		client, err := deepseek.NewChatModel(context.Background(), chatConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create deepseek chat model: %w", err)
		}
		return client, nil
	case "openai", "openai-compatible":
		chatConfig := &openai.ChatModelConfig{
			APIKey:      cfg.APIKey,
			Model:       cfg.LLMModel,
			Temperature: float32Ptr(cfg.Temperature),
		}
		if cfg.MaxTokens > 0 {
			maxTokens := cfg.MaxTokens
			chatConfig.MaxTokens = &maxTokens
			chatConfig.MaxCompletionTokens = &maxTokens
		}
		if endpoint := strings.TrimSpace(cfg.Endpoint); endpoint != "" {
			chatConfig.BaseURL = endpoint
		}

		client, err := openai.NewChatModel(context.Background(), chatConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create openai-compatible chat model: %w", err)
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unsupported ai provider %q", provider)
	}
}

func float32Ptr(v float32) *float32 {
	if v == 0 {
		return nil
	}
	return &v
}

// Chat 对话生成
func (c *chatModelClient) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context is required")
	}
	schemaMessages := make([]*schema.Message, 0, len(messages))
	for _, msg := range messages {
		schemaMessages = append(schemaMessages, &schema.Message{
			Role:    schema.RoleType(msg.Role),
			Content: msg.Content,
		})
	}

	resp, err := c.client.Generate(ctx, schemaMessages, model.WithTemperature(c.temperature), model.WithMaxTokens(c.maxTokens))
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	if resp.Content == "" {
		return "", fmt.Errorf("empty LLM response")
	}

	return resp.Content, nil
}

// ChatStream 流式对话生成（返回累计文本快照）
func (c *chatModelClient) ChatStream(ctx context.Context, messages []ChatMessage) (<-chan string, <-chan error) {
	textCh := make(chan string, chatStreamBufferSize)
	errCh := make(chan error, 1)
	if ctx == nil {
		errCh <- fmt.Errorf("context is required")
		close(textCh)
		close(errCh)
		return textCh, errCh
	}

	schemaMessages := make([]*schema.Message, 0, len(messages))
	for _, msg := range messages {
		schemaMessages = append(schemaMessages, &schema.Message{
			Role:    schema.RoleType(msg.Role),
			Content: msg.Content,
		})
	}

	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := c.client.Stream(streamCtx, schemaMessages, model.WithTemperature(c.temperature), model.WithMaxTokens(c.maxTokens))
	if err != nil {
		cancel()
		errCh <- fmt.Errorf("failed to stream response: %w", err)
		close(textCh)
		close(errCh)
		return textCh, errCh
	}

	var closeOnce sync.Once
	closeStream := func() {
		closeOnce.Do(func() {
			stream.Close()
		})
	}

	reportError := func(err error) {
		if err == nil {
			return
		}
		select {
		case errCh <- err:
		default:
		}
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-streamCtx.Done():
			closeStream()
		case <-done:
		}
	}()

	go func() {
		defer close(done)
		defer close(textCh)
		defer close(errCh)
		defer cancel()
		defer closeStream()

		var builder strings.Builder

		for {
			msg, err := stream.Recv()
			if err != nil {
				if err == io.EOF || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(streamCtx.Err(), context.Canceled) || errors.Is(streamCtx.Err(), context.DeadlineExceeded) {
					return
				}
				reportError(fmt.Errorf("failed to receive stream chunk: %w", err))
				return
			}

			if msg == nil || msg.Content == "" {
				continue
			}

			builder.WriteString(msg.Content)
			snapshot := strings.Clone(builder.String())

			select {
			case <-streamCtx.Done():
				return
			case textCh <- snapshot:
			default:
				select {
				case <-textCh:
				default:
				}
				select {
				case <-streamCtx.Done():
					return
				case textCh <- snapshot:
				default:
				}
			}
		}
	}()

	return textCh, errCh
}

// GetModel 获取原始 Eino 模型
func (c *chatModelClient) GetModel() model.ChatModel {
	return c.client
}
