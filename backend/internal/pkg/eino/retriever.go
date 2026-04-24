package eino

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

// RedisRetriever 实现了 Eino 的 retriever.Retriever 接口
type RedisRetriever struct {
	vectorStore RedisVectorStore
	topK        int
	embedder    Embedder
}

// NewRedisRetriever 创建一个新的 Redis 检索器
func NewRedisRetriever(vs RedisVectorStore, embedder Embedder, topK int) retriever.Retriever {
	if topK <= 0 {
		topK = 3
	}
	return &RedisRetriever{
		vectorStore: vs,
		embedder:    embedder,
		topK:        topK,
	}
}

// Retrieve 执行检索逻辑
func (r *RedisRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	vector, err := r.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query for retrieval: %w", err)
	}

	results, err := r.vectorStore.Search(vector, r.topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector store: %w", err)
	}

	docs := make([]*schema.Document, 0, len(results))
	for _, res := range results {
		var content string
		if val, ok := res.Metadata["content"]; ok {
			switch v := val.(type) {
			case string:
				content = v
			case []byte:
				content = string(v)
			default:
				content = fmt.Sprintf("%v", v)
			}
		}

		meta := make(map[string]interface{}, len(res.Metadata)+1)
		for k, v := range res.Metadata {
			meta[k] = v
		}
		meta["score"] = res.Score

		docs = append(docs, &schema.Document{
			ID:       res.Key,
			Content:  content,
			MetaData: meta,
		})
	}

	return docs, nil
}
