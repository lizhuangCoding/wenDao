package eino

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"wenDao/config"
)

const chatStreamBufferSize = 1

// LLMClient LLM 客户端接口
type LLMClient interface {
	// Chat 对话生成
	Chat(messages []ChatMessage) (string, error)
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

// arkLLMClient Ark LLM 实现（使用 eino 框架）
type arkLLMClient struct {
	client      model.ChatModel
	temperature float32
	maxTokens   int
}

// NewDoubaoLLMClient 创建 Doubao LLM 客户端（使用 Ark）
func NewDoubaoLLMClient(cfg *config.AIConfig) (LLMClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	chatConfig := &ark.ChatModelConfig{
		APIKey: cfg.APIKey,
		Model:  cfg.LLMModel,
	}

	client, err := ark.NewChatModel(context.Background(), chatConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ark chat model: %w", err)
	}

	return &arkLLMClient{
		client:      client,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
	}, nil
}

// Chat 对话生成
func (c *arkLLMClient) Chat(messages []ChatMessage) (string, error) {
	ctx := context.Background()

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
func (c *arkLLMClient) ChatStream(ctx context.Context, messages []ChatMessage) (<-chan string, <-chan error) {
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
func (c *arkLLMClient) GetModel() model.ChatModel {
	return c.client
}
