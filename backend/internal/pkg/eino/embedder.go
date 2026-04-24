package eino

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino/components/embedding"

	"wenDao/config"
)

// Embedder 文本向量化接口
type Embedder interface {
	// Embed 单个文本向量化
	Embed(text string) ([]float32, error)
	// EmbedBatch 批量文本向量化
	EmbedBatch(texts []string) ([][]float32, error)
}

// arkEmbedder Ark Embedding 实现（使用 eino 框架）
type arkEmbedder struct {
	client embedding.Embedder
}

// NewDoubaoEmbedder 创建 Doubao Embedder 实例（使用 Ark）
func NewDoubaoEmbedder(cfg *config.AIConfig) (Embedder, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	embeddingConfig := &ark.EmbeddingConfig{
		APIKey: cfg.APIKey,
		Model:  cfg.EmbeddingModel,
	}

	client, err := ark.NewEmbedder(context.Background(), embeddingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ark embedder: %w", err)
	}

	return &arkEmbedder{client: client}, nil
}

// Embed 单个文本向量化
func (e *arkEmbedder) Embed(text string) ([]float32, error) {
	ctx := context.Background()
	embeddings, err := e.client.EmbedStrings(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("failed to embed text: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	// 打印真实维度，彻底排查不匹配问题
	fmt.Printf("[Embedder] MODEL DIMENSION: %d (Model: %s)\n", len(embeddings[0]), "Doubao")

	// 转换 float64 to float32
	result := make([]float32, len(embeddings[0]))
	for i, v := range embeddings[0] {
		result[i] = float32(v)
	}

	return result, nil
}

// EmbedBatch 批量文本向量化
func (e *arkEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	ctx := context.Background()
	embeddings, err := e.client.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to embed batch: %w", err)
	}

	if len(embeddings) > 0 {
		fmt.Printf("[Embedder] BATCH COMPLETED. Count: %d, Dimension: %d\n", len(embeddings), len(embeddings[0]))
	}

	// 转换 [][]float64 to [][]float32
	results := make([][]float32, len(embeddings))
	for i, embedding := range embeddings {
		results[i] = make([]float32, len(embedding))
		for j, v := range embedding {
			results[i][j] = float32(v)
		}
	}

	return results, nil
}
