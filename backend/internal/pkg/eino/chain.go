package eino

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

const ragStreamBufferSize = 1

// RAGChain 封装了 RAG 流程
type RAGChain struct {
	retriever   retriever.Retriever
	llm         model.ChatModel
	ragMinScore float32
	logger      *zap.Logger
}

// NewRAGChain 创建 RAG Chain
func NewRAGChain(r retriever.Retriever, llm model.ChatModel, ragMinScore float32, logger *zap.Logger) *RAGChain {
	return &RAGChain{
		retriever:   r,
		llm:         llm,
		ragMinScore: ragMinScore,
		logger:      logger,
	}
}

func (c *RAGChain) shouldUseFallback(docs []*schema.Document, threshold float32) bool {
	if len(docs) == 0 {
		c.logger.Info("[RAG] No documents retrieved, triggering fallback")
		return true
	}

	rawScore, ok := docs[0].MetaData["score"]
	if !ok {
		c.logger.Warn("[RAG] Document metadata missing 'score', triggering fallback")
		return true
	}

	var distance float32
	switch v := rawScore.(type) {
	case float32:
		distance = v
	case float64:
		distance = float32(v)
	case string:
		// 有时 Redis 返回字符串
		fmt.Sscanf(v, "%f", &distance)
	default:
		c.logger.Warn("[RAG] Unexpected score type", zap.Any("type", fmt.Sprintf("%T", rawScore)))
		return true
	}

	// 相似度分数 (similarity) = 1 - distance
	similarity := 1 - distance

	c.logger.Info("[RAG] Retrieval result",
		zap.Int("doc_count", len(docs)),
		zap.Float32("top_distance", distance),
		zap.Float32("top_similarity", similarity),
		zap.Float32("threshold", threshold))

	if similarity < threshold {
		c.logger.Info("[RAG] Similarity below threshold, triggering fallback")
		return true
	}

	return false
}

func (c *RAGChain) prepareMessages(ctx context.Context, question string) ([]*schema.Message, error) {
	c.logger.Info("[RAG] Starting process for question", zap.String("question", question))

	c.logger.Info("[RAG] Executing retrieval", zap.String("query", question))
	docs, err := c.retriever.Retrieve(ctx, question)
	if err != nil {
		c.logger.Error("[RAG] Retrieval error", zap.Error(err))
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}
	c.logger.Info("[RAG] Retrieval completed", zap.Int("doc_count", len(docs)))

	if c.shouldUseFallback(docs, c.ragMinScore) {
		msgs := c.buildFallbackMessages(question)
		c.logger.Info("[RAG] Using Fallback Prompt", zap.String("system_prompt", msgs[0].Content))
		return msgs, nil
	}

	msgs := c.buildRAGMessages(question, docs)
	c.logger.Info("[RAG] Using RAG Prompt", zap.String("system_prompt", msgs[0].Content))
	return msgs, nil
}

// SummarizeLocalFindings 执行本地知识检索与总结
func (c *RAGChain) SummarizeLocalFindings(ctx context.Context, question string) (string, []*schema.Document, error) {
	docs, err := c.retriever.Retrieve(ctx, question)
	if err != nil {
		return "", nil, fmt.Errorf("retrieval failed: %w", err)
	}
	if len(docs) == 0 {
		return "", docs, nil
	}
	messages := c.buildRAGMessages(question, docs)
	resp, err := c.llm.Generate(ctx, messages)
	if err != nil {
		return "", nil, fmt.Errorf("generation failed: %w", err)
	}
	return resp.Content, docs, nil
}

// Execute 执行 RAG 对话
func (c *RAGChain) Execute(ctx context.Context, question string) (string, error) {
	messages, err := c.prepareMessages(ctx, question)
	if err != nil {
		return "", err
	}

	resp, err := c.llm.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("generation failed: %w", err)
	}

	return resp.Content, nil
}

// ExecuteStream 流式执行 RAG 对话（返回累计文本快照）
func (c *RAGChain) ExecuteStream(ctx context.Context, question string) (<-chan string, <-chan error) {
	textCh := make(chan string, ragStreamBufferSize)
	errCh := make(chan error, 1)
	if ctx == nil {
		errCh <- fmt.Errorf("context is required")
		close(textCh)
		close(errCh)
		return textCh, errCh
	}

	messages, err := c.prepareMessages(ctx, question)
	if err != nil {
		errCh <- err
		close(textCh)
		close(errCh)
		return textCh, errCh
	}

	streamCtx, cancel := context.WithCancel(ctx)
	stream, err := c.llm.Stream(streamCtx, messages)
	if err != nil {
		cancel()
		errCh <- fmt.Errorf("generation failed: %w", err)
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
				reportError(fmt.Errorf("generation failed: %w", err))
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

func (c *RAGChain) buildRAGMessages(question string, docs []*schema.Document) []*schema.Message {
	systemPrompt := "你是问道博客的 AI 助手。你的首要任务是基于提供的文章片段回答用户问题。\n\n" +
		"回答规则：\n" +
		"1. 优先使用提供的文章片段作答，可以重组、概括、解释，但不能编造站内不存在的内容。\n" +
		"2. 如果文章片段只能部分回答问题，明确说明哪些结论来自本站内容，哪些内容本站没有覆盖。\n" +
		"3. 不要说“博客中提到”除非上下文里真的有对应信息。\n" +
		"4. 使用中文回答。\n" +
		"5. 输出尽量采用：结论 + 要点列表 + 简短补充说明。\n\n" +
		"以下是可用的文章片段：\n"

	for i, doc := range docs {
		c.logger.Info(fmt.Sprintf("[RAG] Doc %d content preview: %s", i+1, truncateString(doc.Content, 100)))
		systemPrompt += fmt.Sprintf("\n--- 文章片段 %d ---\n%s\n", i+1, doc.Content)
	}

	c.logger.Info("[RAG] Final RAG System Prompt", zap.String("prompt", systemPrompt))

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(question),
	}
}

func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

func (c *RAGChain) buildFallbackMessages(question string) []*schema.Message {
	systemPrompt := "你是问道博客的 AI 助手。当前知识库检索没有找到足够相关的文章内容。\n\n" +
		"现在请你基于通用知识回答用户问题，并严格遵守以下规则：\n" +
		"1. 回答开头必须明确写出：以下回答基于通用知识，不是来自本站文章内容。\n" +
		"2. 不要假装引用、总结或转述本站文章。\n" +
		"3. 如果问题有歧义，采用最合理的理解作答，并说明你的假设。\n" +
		"4. 如果问题缺少关键条件，明确指出缺少什么信息。\n" +
		"5. 回答要具体、实用，避免空泛套话。\n" +
		"6. 使用中文回答。"

	c.logger.Info("[RAG] Final Fallback System Prompt", zap.String("prompt", systemPrompt))

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(question),
	}
}
