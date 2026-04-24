# AI/RAG System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement AI-powered Q&A system using RAG (Retrieval-Augmented Generation) with Doubao LLM and Redis Vector

**Architecture:** Three-layer eino wrapper (Embedder, LLMClient, RedisVectorStore) → Service layer (VectorService, AIService) → Handler layer (AIHandler). Synchronous vectorization on article publish/update, asynchronous execution to avoid blocking.

**Tech Stack:** eino framework, Doubao Embedding/LLM APIs, Redis Stack (Redis Vector with HNSW index), Go 1.22+

---

## File Structure

**New files to create:**
```
backend/internal/
├── pkg/eino/
│   ├── embedder.go        # Doubao Embedding API wrapper (Embedder interface + doubaoEmbedder)
│   ├── llm.go             # Doubao LLM API wrapper (LLMClient interface + doubaoLLMClient)
│   └── vectorstore.go     # Redis Vector operations (RedisVectorStore interface + redisVectorStore)
├── service/
│   ├── vector.go          # VectorService (article chunking, vectorization, search)
│   └── ai.go              # AIService (RAG workflow: embed question → search → build prompt → LLM)
└── handler/
    └── ai.go              # AIHandler (POST /api/ai/chat endpoint)
```

**Files to modify:**
```
backend/
├── config/config.go           # Update AIConfig struct with LLMModel, Temperature, MaxTokens
├── config/config.yaml         # Add ai section with all parameters
├── config/.env.example        # Add DOUBAO_API_KEY
├── internal/service/article.go # Inject VectorService, add goroutine calls in Publish/Update/Delete
├── cmd/server/main.go         # Initialize eino components, services, handler; add /api/ai route
└── go.mod                     # Add eino and RediSearch dependencies
```

---

## Task 1: Install Dependencies

**Files:**
- Modify: `backend/go.mod`

- [ ] **Step 1.1: Add eino dependencies**

Run in `backend/` directory:
```bash
cd backend
go get github.com/cloudwego/eino
go get github.com/cloudwego/eino-ext/components/model/doubao
go get github.com/RediSearch/redisearch-go/v2
```

Expected: Dependencies added to go.mod and go.sum

- [ ] **Step 1.2: Run go mod tidy**

```bash
go mod tidy
```

Expected: go.mod and go.sum updated successfully

- [ ] **Step 1.3: Commit dependency changes**

```bash
git add go.mod go.sum
git commit -m "deps: add eino and RediSearch dependencies for AI/RAG system

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Configuration Setup

**Files:**
- Modify: `backend/config/config.go:80-91`
- Modify: `backend/config/config.yaml`
- Modify: `backend/config/.env.example`

- [ ] **Step 2.1: Update AIConfig struct**

AIConfig already exists at lines 80-91. Update it to match spec requirements:

```go
// AIConfig AI 配置
type AIConfig struct {
	APIKey         string  `mapstructure:"api_key"`
	EmbeddingModel string  `mapstructure:"embedding_model"`
	LLMModel       string  `mapstructure:"llm_model"`
	Temperature    float32 `mapstructure:"temperature"`
	MaxTokens      int     `mapstructure:"max_tokens"`
	TopK           int     `mapstructure:"top_k"`
}
```

Replace lines 80-91 in `config/config.go` with the above.

- [ ] **Step 2.2: Add AI configuration to config.yaml**

Append to `backend/config/config.yaml`:

```yaml
ai:
  api_key: ${DOUBAO_API_KEY}
  embedding_model: doubao-embedding-v1
  llm_model: doubao-pro-32k
  temperature: 0.7
  max_tokens: 500
  top_k: 3
```

- [ ] **Step 2.3: Add DOUBAO_API_KEY to .env.example**

Append to `backend/config/.env.example`:

```bash
# AI/RAG 配置
DOUBAO_API_KEY=your_doubao_api_key_here
```

- [ ] **Step 2.4: Commit configuration changes**

```bash
cd backend
git add config/config.go config/config.yaml config/.env.example
git commit -m "config: update AI configuration for Doubao integration

- Update AIConfig with LLMModel, Temperature, MaxTokens fields
- Add ai section to config.yaml with Doubao settings
- Add DOUBAO_API_KEY to .env.example

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Doubao Embedder Wrapper

**Files:**
- Create: `backend/internal/pkg/eino/embedder.go`

- [ ] **Step 3.1: Create embedder.go with interface and implementation**

```go
package eino

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/doubao"
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

// doubaoEmbedder Doubao Embedding 实现
type doubaoEmbedder struct {
	client embedding.Embedder
}

// NewDoubaoEmbedder 创建 Doubao Embedder 实例
func NewDoubaoEmbedder(cfg *config.AIConfig) (Embedder, error) {
	embedderConfig := doubao.NewEmbeddingConfig(cfg.APIKey, cfg.EmbeddingModel)
	client, err := doubao.NewEmbedder(context.Background(), embedderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Doubao embedder: %w", err)
	}

	return &doubaoEmbedder{client: client}, nil
}

// Embed 单个文本向量化
func (e *doubaoEmbedder) Embed(text string) ([]float32, error) {
	ctx := context.Background()
	resp, err := e.client.EmbedStrings(ctx, []string{text})
	if err != nil {
		return nil, fmt.Errorf("failed to embed text: %w", err)
	}

	if len(resp) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	return resp[0], nil
}

// EmbedBatch 批量文本向量化
func (e *doubaoEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	ctx := context.Background()
	embeddings, err := e.client.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("failed to embed batch: %w", err)
	}

	return embeddings, nil
}
```

- [ ] **Step 3.2: Verify syntax**

```bash
cd backend
go build -o /dev/null ./internal/pkg/eino/
```

Expected: No compilation errors

- [ ] **Step 3.3: Commit Doubao Embedder wrapper**

```bash
git add internal/pkg/eino/embedder.go
git commit -m "feat: implement Doubao Embedder wrapper

- Create Embedder interface with Embed and EmbedBatch methods
- Implement doubaoEmbedder using eino-ext/components/model/doubao
- Support single and batch text vectorization

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Doubao LLM Client Wrapper

**Files:**
- Create: `backend/internal/pkg/eino/llm.go`

- [ ] **Step 4.1: Create llm.go with interface and implementation**

```go
package eino

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/model/doubao"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"wenDao/config"
)

// LLMClient LLM 客户端接口
type LLMClient interface {
	// Chat 对话生成
	Chat(messages []ChatMessage) (string, error)
}

// ChatMessage 对话消息
type ChatMessage struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// doubaoLLMClient Doubao LLM 实现
type doubaoLLMClient struct {
	client      model.ChatModel
	temperature float32
	maxTokens   int
}

// NewDoubaoLLMClient 创建 Doubao LLM 客户端
func NewDoubaoLLMClient(cfg *config.AIConfig) (LLMClient, error) {
	chatConfig := doubao.NewChatModelConfig(cfg.APIKey, cfg.LLMModel)
	client, err := doubao.NewChatModel(context.Background(), chatConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Doubao chat model: %w", err)
	}

	return &doubaoLLMClient{
		client:      client,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
	}, nil
}

// Chat 对话生成
func (c *doubaoLLMClient) Chat(messages []ChatMessage) (string, error) {
	ctx := context.Background()

	// 转换消息格式
	chatMessages := make([]*schema.Message, 0, len(messages))
	for _, msg := range messages {
		chatMessages = append(chatMessages, &schema.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 构建生成请求
	req := &schema.ChatModelRequest{
		Messages: chatMessages,
		Config: &schema.GenerateConfig{
			Temperature: float64(c.temperature),
			MaxTokens:   c.maxTokens,
		},
	}

	// 调用 LLM
	resp, err := c.client.Generate(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	if resp.Content == "" {
		return "", fmt.Errorf("empty LLM response")
	}

	return resp.Content, nil
}
```

- [ ] **Step 4.2: Verify syntax**

```bash
cd backend
go build -o /dev/null ./internal/pkg/eino/
```

Expected: No compilation errors

- [ ] **Step 4.3: Commit Doubao LLM Client wrapper**

```bash
git add internal/pkg/eino/llm.go
git commit -m "feat: implement Doubao LLM Client wrapper

- Create LLMClient interface with Chat method
- Implement doubaoLLMClient using eino-ext/components/model/doubao
- Support temperature and max_tokens configuration
- Convert ChatMessage to schema.Message format

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Redis Vector Store Wrapper

**Files:**
- Create: `backend/internal/pkg/eino/vectorstore.go`

- [ ] **Step 5.1: Create vectorstore.go with interface and implementation**

```go
package eino

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/RediSearch/redisearch-go/v2/redisearch"
	"github.com/redis/go-redis/v9"
)

// RedisVectorStore Redis 向量存储接口
type RedisVectorStore interface {
	// InitIndex 初始化向量索引
	InitIndex(indexName string, dim int) error
	// Upsert 插入/更新向量
	Upsert(key string, vector []float32, metadata map[string]interface{}) error
	// UpsertBatch 批量插入/更新向量
	UpsertBatch(items []VectorItem) error
	// Delete 删除向量（支持通配符）
	Delete(pattern string) error
	// Search 向量搜索
	Search(vector []float32, topK int) ([]SearchResult, error)
}

// VectorItem 向量数据项
type VectorItem struct {
	Key      string
	Vector   []float32
	Metadata map[string]interface{}
}

// SearchResult 搜索结果
type SearchResult struct {
	Key      string
	Score    float32
	Metadata map[string]interface{}
}

// redisVectorStore Redis 向量存储实现
type redisVectorStore struct {
	client    *redis.Client
	indexName string
}

// NewRedisVectorStore 创建 Redis 向量存储实例
func NewRedisVectorStore(client *redis.Client, indexName string) RedisVectorStore {
	return &redisVectorStore{
		client:    client,
		indexName: indexName,
	}
}

// InitIndex 初始化向量索引
func (s *redisVectorStore) InitIndex(indexName string, dim int) error {
	ctx := context.Background()

	// 检查索引是否已存在
	result, err := s.client.Do(ctx, "FT.INFO", indexName).Result()
	if err == nil && result != nil {
		// 索引已存在，跳过创建
		return nil
	}

	// 创建向量索引
	// FT.CREATE article_embeddings ON HASH PREFIX 1 article: SCHEMA
	//   embedding VECTOR HNSW 6 TYPE FLOAT32 DIM 1024 DISTANCE_METRIC COSINE
	//   article_id NUMERIC
	//   chunk_index NUMERIC
	//   title TEXT
	//   content TEXT
	cmd := []interface{}{
		"FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", "1", "article:",
		"SCHEMA",
		"embedding", "VECTOR", "HNSW", "6",
		"TYPE", "FLOAT32",
		"DIM", dim,
		"DISTANCE_METRIC", "COSINE",
		"article_id", "NUMERIC",
		"chunk_index", "NUMERIC",
		"title", "TEXT",
		"content", "TEXT",
	}

	_, err = s.client.Do(ctx, cmd...).Result()
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

// Upsert 插入/更新向量
func (s *redisVectorStore) Upsert(key string, vector []float32, metadata map[string]interface{}) error {
	ctx := context.Background()

	// 将 float32 向量转换为字节数组
	vectorBytes := make([]byte, len(vector)*4)
	for i, v := range vector {
		bits := float32ToBits(v)
		vectorBytes[i*4] = byte(bits)
		vectorBytes[i*4+1] = byte(bits >> 8)
		vectorBytes[i*4+2] = byte(bits >> 16)
		vectorBytes[i*4+3] = byte(bits >> 24)
	}

	// 构建 HSET 命令参数
	fields := []interface{}{key}
	fields = append(fields, "embedding", vectorBytes)
	for k, v := range metadata {
		fields = append(fields, k, v)
	}

	// 存储到 Redis Hash
	_, err := s.client.HSet(ctx, key, fields[1:]...).Result()
	if err != nil {
		return fmt.Errorf("failed to upsert vector: %w", err)
	}

	return nil
}

// UpsertBatch 批量插入/更新向量
func (s *redisVectorStore) UpsertBatch(items []VectorItem) error {
	ctx := context.Background()
	pipe := s.client.Pipeline()

	for _, item := range items {
		// 将 float32 向量转换为字节数组
		vectorBytes := make([]byte, len(item.Vector)*4)
		for i, v := range item.Vector {
			bits := float32ToBits(v)
			vectorBytes[i*4] = byte(bits)
			vectorBytes[i*4+1] = byte(bits >> 8)
			vectorBytes[i*4+2] = byte(bits >> 16)
			vectorBytes[i*4+3] = byte(bits >> 24)
		}

		// 构建字段
		fields := []interface{}{"embedding", vectorBytes}
		for k, v := range item.Metadata {
			fields = append(fields, k, v)
		}

		pipe.HSet(ctx, item.Key, fields...)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to batch upsert: %w", err)
	}

	return nil
}

// Delete 删除向量（支持通配符）
func (s *redisVectorStore) Delete(pattern string) error {
	ctx := context.Background()

	// 使用 SCAN 查找匹配的 key
	var cursor uint64
	var keys []string

	for {
		var scanKeys []string
		var err error

		scanKeys, cursor, err = s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		keys = append(keys, scanKeys...)

		if cursor == 0 {
			break
		}
	}

	// 批量删除
	if len(keys) > 0 {
		_, err := s.client.Del(ctx, keys...).Result()
		if err != nil {
			return fmt.Errorf("failed to delete keys: %w", err)
		}
	}

	return nil
}

// Search 向量搜索
func (s *redisVectorStore) Search(vector []float32, topK int) ([]SearchResult, error) {
	ctx := context.Background()

	// 将 float32 向量转换为字节数组
	vectorBytes := make([]byte, len(vector)*4)
	for i, v := range vector {
		bits := float32ToBits(v)
		vectorBytes[i*4] = byte(bits)
		vectorBytes[i*4+1] = byte(bits >> 8)
		vectorBytes[i*4+2] = byte(bits >> 16)
		vectorBytes[i*4+3] = byte(bits >> 24)
	}

	// 构建 KNN 查询
	// FT.SEARCH article_embeddings "*=>[KNN 3 @embedding $query_vec]"
	//   PARAMS 2 query_vec <vector_bytes>
	//   RETURN 4 article_id chunk_index title content
	//   DIALECT 2
	query := fmt.Sprintf("*=>[KNN %d @embedding $query_vec]", topK)
	cmd := []interface{}{
		"FT.SEARCH", s.indexName,
		query,
		"PARAMS", "2", "query_vec", vectorBytes,
		"RETURN", "4", "article_id", "chunk_index", "title", "content",
		"DIALECT", "2",
	}

	result, err := s.client.Do(ctx, cmd...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	// 解析搜索结果
	results := make([]SearchResult, 0)
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) < 1 {
		return results, nil
	}

	// 第一个元素是总数
	totalCount, _ := resultSlice[0].(int64)
	if totalCount == 0 {
		return results, nil
	}

	// 从第二个元素开始是搜索结果
	for i := 1; i < len(resultSlice); i += 2 {
		if i+1 >= len(resultSlice) {
			break
		}

		key, _ := resultSlice[i].(string)
		fields, _ := resultSlice[i+1].([]interface{})

		// 解析字段
		metadata := make(map[string]interface{})
		score := float32(0.0)

		for j := 0; j < len(fields); j += 2 {
			if j+1 >= len(fields) {
				break
			}

			fieldName, _ := fields[j].(string)
			fieldValue := fields[j+1]

			if fieldName == "__embedding_score" {
				// 相似度分数
				scoreStr, _ := fieldValue.(string)
				fmt.Sscanf(scoreStr, "%f", &score)
			} else {
				metadata[fieldName] = fieldValue
			}
		}

		results = append(results, SearchResult{
			Key:      key,
			Score:    score,
			Metadata: metadata,
		})
	}

	return results, nil
}

// float32ToBits 将 float32 转换为 uint32 bits
func float32ToBits(f float32) uint32 {
	return *(*uint32)(unsafe.Pointer(&f))
}
```

Wait, I need to add the unsafe import. Let me fix that:

```go
package eino

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/redis/go-redis/v9"
)

// RedisVectorStore Redis 向量存储接口
type RedisVectorStore interface {
	// InitIndex 初始化向量索引
	InitIndex(indexName string, dim int) error
	// Upsert 插入/更新向量
	Upsert(key string, vector []float32, metadata map[string]interface{}) error
	// UpsertBatch 批量插入/更新向量
	UpsertBatch(items []VectorItem) error
	// Delete 删除向量（支持通配符）
	Delete(pattern string) error
	// Search 向量搜索
	Search(vector []float32, topK int) ([]SearchResult, error)
}

// VectorItem 向量数据项
type VectorItem struct {
	Key      string
	Vector   []float32
	Metadata map[string]interface{}
}

// SearchResult 搜索结果
type SearchResult struct {
	Key      string
	Score    float32
	Metadata map[string]interface{}
}

// redisVectorStore Redis 向量存储实现
type redisVectorStore struct {
	client    *redis.Client
	indexName string
}

// NewRedisVectorStore 创建 Redis 向量存储实例
func NewRedisVectorStore(client *redis.Client, indexName string) RedisVectorStore {
	return &redisVectorStore{
		client:    client,
		indexName: indexName,
	}
}

// InitIndex 初始化向量索引
func (s *redisVectorStore) InitIndex(indexName string, dim int) error {
	ctx := context.Background()

	// 检查索引是否已存在
	result, err := s.client.Do(ctx, "FT.INFO", indexName).Result()
	if err == nil && result != nil {
		// 索引已存在，跳过创建
		return nil
	}

	// 创建向量索引
	cmd := []interface{}{
		"FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", "1", "article:",
		"SCHEMA",
		"embedding", "VECTOR", "HNSW", "6",
		"TYPE", "FLOAT32",
		"DIM", dim,
		"DISTANCE_METRIC", "COSINE",
		"article_id", "NUMERIC",
		"chunk_index", "NUMERIC",
		"title", "TEXT",
		"content", "TEXT",
	}

	_, err = s.client.Do(ctx, cmd...).Result()
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

// Upsert 插入/更新向量
func (s *redisVectorStore) Upsert(key string, vector []float32, metadata map[string]interface{}) error {
	ctx := context.Background()

	// 将 float32 向量转换为字节数组
	vectorBytes := float32SliceToBytes(vector)

	// 构建 HSET 命令参数
	fields := []interface{}{"embedding", vectorBytes}
	for k, v := range metadata {
		fields = append(fields, k, v)
	}

	// 存储到 Redis Hash
	_, err := s.client.HSet(ctx, key, fields...).Result()
	if err != nil {
		return fmt.Errorf("failed to upsert vector: %w", err)
	}

	return nil
}

// UpsertBatch 批量插入/更新向量
func (s *redisVectorStore) UpsertBatch(items []VectorItem) error {
	ctx := context.Background()
	pipe := s.client.Pipeline()

	for _, item := range items {
		// 将 float32 向量转换为字节数组
		vectorBytes := float32SliceToBytes(item.Vector)

		// 构建字段
		fields := []interface{}{"embedding", vectorBytes}
		for k, v := range item.Metadata {
			fields = append(fields, k, v)
		}

		pipe.HSet(ctx, item.Key, fields...)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to batch upsert: %w", err)
	}

	return nil
}

// Delete 删除向量（支持通配符）
func (s *redisVectorStore) Delete(pattern string) error {
	ctx := context.Background()

	// 使用 SCAN 查找匹配的 key
	var cursor uint64
	var keys []string

	for {
		var scanKeys []string
		var err error

		scanKeys, cursor, err = s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys: %w", err)
		}

		keys = append(keys, scanKeys...)

		if cursor == 0 {
			break
		}
	}

	// 批量删除
	if len(keys) > 0 {
		_, err := s.client.Del(ctx, keys...).Result()
		if err != nil {
			return fmt.Errorf("failed to delete keys: %w", err)
		}
	}

	return nil
}

// Search 向量搜索
func (s *redisVectorStore) Search(vector []float32, topK int) ([]SearchResult, error) {
	ctx := context.Background()

	// 将 float32 向量转换为字节数组
	vectorBytes := float32SliceToBytes(vector)

	// 构建 KNN 查询
	query := fmt.Sprintf("*=>[KNN %d @embedding $query_vec]", topK)
	cmd := []interface{}{
		"FT.SEARCH", s.indexName,
		query,
		"PARAMS", "2", "query_vec", vectorBytes,
		"RETURN", "4", "article_id", "chunk_index", "title", "content",
		"DIALECT", "2",
	}

	result, err := s.client.Do(ctx, cmd...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	// 解析搜索结果
	results := make([]SearchResult, 0)
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) < 1 {
		return results, nil
	}

	// 第一个元素是总数
	totalCount, _ := resultSlice[0].(int64)
	if totalCount == 0 {
		return results, nil
	}

	// 从第二个元素开始是搜索结果
	for i := 1; i < len(resultSlice); i += 2 {
		if i+1 >= len(resultSlice) {
			break
		}

		key, _ := resultSlice[i].(string)
		fields, _ := resultSlice[i+1].([]interface{})

		// 解析字段
		metadata := make(map[string]interface{})
		score := float32(0.0)

		for j := 0; j < len(fields); j += 2 {
			if j+1 >= len(fields) {
				break
			}

			fieldName, _ := fields[j].(string)
			fieldValue := fields[j+1]

			if fieldName == "__embedding_score" {
				// 相似度分数
				scoreStr, _ := fieldValue.(string)
				fmt.Sscanf(scoreStr, "%f", &score)
			} else {
				metadata[fieldName] = fieldValue
			}
		}

		results = append(results, SearchResult{
			Key:      key,
			Score:    score,
			Metadata: metadata,
		})
	}

	return results, nil
}

// float32SliceToBytes 将 float32 切片转换为字节数组
func float32SliceToBytes(vector []float32) []byte {
	vectorBytes := make([]byte, len(vector)*4)
	for i, v := range vector {
		bits := float32ToBits(v)
		vectorBytes[i*4] = byte(bits)
		vectorBytes[i*4+1] = byte(bits >> 8)
		vectorBytes[i*4+2] = byte(bits >> 16)
		vectorBytes[i*4+3] = byte(bits >> 24)
	}
	return vectorBytes
}

// float32ToBits 将 float32 转换为 uint32 bits
func float32ToBits(f float32) uint32 {
	return *(*uint32)(unsafe.Pointer(&f))
}
```

This implementation is too long for a single step. Let me simplify the plan by breaking this into verification steps only. Actually, the plan is growing too large. Let me write a more concise plan following the DRY principle:

