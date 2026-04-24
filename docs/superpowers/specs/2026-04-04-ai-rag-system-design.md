# AI/RAG 系统设计规范 (Sub-project 3)

**项目**: 问道博客平台
**子项目**: Sub-project 3 - AI/RAG 系统
**日期**: 2026-04-04
**状态**: 已批准

## 概述

为问道博客平台添加基于 RAG (Retrieval-Augmented Generation) 的 AI 问答功能，用户可以基于已发布的文章内容与 AI 对话。采用同步向量化方案，使用 eino 框架集成豆包 (Doubao) 的 Embedding 和 LLM API，Redis Stack 作为向量数据库。

## 核心目标

1. **文章向量化**: 文章发布/更新时自动生成 embedding 并存储到 Redis Vector
2. **AI 对话**: 用户提问时，检索相关文章片段作为上下文，调用 LLM 生成回答
3. **集成现有系统**: 无缝集成到现有的三层架构中，不影响文章管理核心功能

## 技术选型

| 组件 | 技术选择 | 理由 |
|------|----------|------|
| RAG 框架 | eino | 字节官方 Go RAG 框架，与豆包集成良好 |
| LLM | 豆包 (Doubao) | 性能优秀，支持长上下文 |
| Embedding | 豆包 Embedding API | 与 LLM 同源，语义一致性好 |
| 向量数据库 | Redis Stack (Redis Vector) | 已部署，支持 HNSW 索引，低延迟 |
| 向量化时机 | 同步（文章发布/更新时） | 简化逻辑，确保数据一致性 |
| 分块策略 | 按段落分块（换行符分割） | 简单有效，保留语义完整性 |

## 系统架构

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                         前端 (Frontend)                       │
└───────────────────────────┬─────────────────────────────────┘
                            │ HTTP
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Gin Router (main.go)                      │
│  ┌──────────────────┐              ┌──────────────────┐    │
│  │  ArticleHandler  │              │    AIHandler     │    │
│  └────────┬─────────┘              └────────┬─────────┘    │
└───────────┼──────────────────────────────────┼──────────────┘
            │                                  │
            ▼                                  ▼
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                           │
│  ┌──────────────────┐              ┌──────────────────┐    │
│  │  ArticleService  │◄─────────────┤  VectorService   │    │
│  └────────┬─────────┘              └────────┬─────────┘    │
│           │                                  │              │
│           │                        ┌─────────▼─────────┐   │
│           │                        │    AIService      │   │
│           │                        └────────┬──────────┘   │
└───────────┼─────────────────────────────────┼──────────────┘
            │                                  │
            ▼                                  ▼
┌─────────────────────┐          ┌──────────────────────────┐
│  ArticleRepository  │          │    eino Wrapper Layer    │
│                     │          │  ┌──────────────────┐   │
│    (MySQL GORM)     │          │  │    Embedder      │   │
└─────────────────────┘          │  │  (Doubao API)    │   │
                                 │  └──────────────────┘   │
                                 │  ┌──────────────────┐   │
                                 │  │    LLMClient     │   │
                                 │  │  (Doubao API)    │   │
                                 │  └──────────────────┘   │
                                 │  ┌──────────────────┐   │
                                 │  │ RedisVectorStore │   │
                                 │  │  (Redis Vector)  │   │
                                 │  └──────────────────┘   │
                                 └──────────────────────────┘
```

### 数据流

#### 1. 文章发布/更新流程

```
用户发布文章
    ↓
ArticleHandler.Publish()
    ↓
ArticleService.Publish()
    ├─→ 更新文章状态到数据库
    └─→ VectorService.VectorizeArticle()
            ├─→ 分块文章内容
            ├─→ Embedder.Embed() (调用豆包 API)
            └─→ RedisVectorStore.Upsert() (存储到 Redis Vector)
```

#### 2. AI 对话流程

```
用户提问
    ↓
AIHandler.Chat()
    ↓
AIService.Chat(question)
    ├─→ Embedder.Embed(question) (问题向量化)
    ├─→ RedisVectorStore.Search() (检索 Top-K 相关片段)
    ├─→ 构建 Prompt (系统提示 + 相关片段 + 用户问题)
    └─→ LLMClient.Chat() (调用豆包 LLM 生成回答)
```

## 核心组件设计

### 1. VectorService

**职责**: 管理文章向量化生命周期

**接口**:
```go
type VectorService interface {
    // 向量化文章（发布/更新时调用）
    VectorizeArticle(articleID int64, title, content string) error

    // 删除文章向量（文章删除/下线时调用）
    DeleteArticleVector(articleID int64) error

    // 搜索相关文章片段（AI 对话时调用）
    SearchArticles(query string, topK int) ([]ArticleChunk, error)
}

type ArticleChunk struct {
    ArticleID int64   // 文章 ID
    ChunkID   string  // 片段 ID (article_id:chunk_index)
    Content   string  // 片段内容
    Score     float32 // 相似度分数
}
```

**实现细节**:
- **分块策略**: 按段落分块（`strings.Split(content, "\n\n")`），过滤空段落
- **向量存储**: 每个片段作为独立向量存储，Key 格式: `article:{articleID}:chunk:{index}`
- **元数据**: 存储 article_id, chunk_index, title, content
- **错误处理**: 向量化失败不阻塞文章发布，记录错误日志

### 2. AIService

**职责**: 处理用户问答请求，执行 RAG 流程

**接口**:
```go
type AIService interface {
    // AI 对话（基于 RAG）
    Chat(question string) (string, error)
}
```

**RAG 流程**:
1. **问题向量化**: 调用 Embedder 生成问题的 embedding
2. **检索相关片段**: 在 RedisVectorStore 中检索 Top-3 相关片段
3. **构建 Prompt**:
   ```
   System: 你是问道博客的 AI 助手，基于提供的文章片段回答用户问题。

   文章片段：
   [片段1内容]
   [片段2内容]
   [片段3内容]

   用户问题：{question}
   ```
4. **调用 LLM**: 使用 LLMClient 生成回答
5. **返回结果**: 返回 LLM 生成的文本

**参数配置**:
- `topK`: 检索片段数量（默认 3）
- `temperature`: LLM 生成温度（默认 0.7）
- `max_tokens`: 最大生成长度（默认 500）

### 3. eino Wrapper Layer

#### 3.1 Embedder

**职责**: 封装豆包 Embedding API

```go
type Embedder interface {
    Embed(text string) ([]float32, error)
    EmbedBatch(texts []string) ([][]float32, error)
}

type doubaoEmbedder struct {
    client *embeddings.Client
}

func NewDoubaoEmbedder(cfg *config.AIConfig) Embedder {
    client := embeddings.NewClient(
        embeddings.WithAPIKey(cfg.APIKey),
        embeddings.WithModel(cfg.EmbeddingModel), // e.g., "doubao-embedding-v1"
    )
    return &doubaoEmbedder{client: client}
}
```

**配置**:
- API Key: 从环境变量 `DOUBAO_API_KEY` 读取
- Model: `doubao-embedding-v1` (输出维度 1024)

#### 3.2 LLMClient

**职责**: 封装豆包 LLM API

```go
type LLMClient interface {
    Chat(messages []ChatMessage) (string, error)
}

type ChatMessage struct {
    Role    string // "system" | "user" | "assistant"
    Content string
}

type doubaoLLMClient struct {
    client *chat.Client
}

func NewDoubaoLLMClient(cfg *config.AIConfig) LLMClient {
    client := chat.NewClient(
        chat.WithAPIKey(cfg.APIKey),
        chat.WithModel(cfg.LLMModel), // e.g., "doubao-pro-32k"
    )
    return &doubaoLLMClient{client: client}
}
```

**配置**:
- API Key: 从环境变量 `DOUBAO_API_KEY` 读取
- Model: `doubao-pro-32k` (支持长上下文)
- Temperature: 0.7 (可配置)

#### 3.3 RedisVectorStore

**职责**: 封装 Redis Vector 操作

```go
type RedisVectorStore interface {
    // 初始化向量索引（应用启动时调用）
    InitIndex(indexName string, dim int) error

    // 插入/更新向量
    Upsert(key string, vector []float32, metadata map[string]interface{}) error

    // 批量插入
    UpsertBatch(items []VectorItem) error

    // 删除向量（支持通配符）
    Delete(pattern string) error

    // 向量搜索
    Search(vector []float32, topK int) ([]SearchResult, error)
}

type VectorItem struct {
    Key      string
    Vector   []float32
    Metadata map[string]interface{}
}

type SearchResult struct {
    Key      string
    Score    float32
    Metadata map[string]interface{}
}

type redisVectorStore struct {
    client    *redis.Client
    indexName string
}
```

**Redis Vector 索引配置**:
```redis
FT.CREATE article_embeddings
  ON HASH PREFIX 1 article:
  SCHEMA
    embedding VECTOR HNSW 6 TYPE FLOAT32 DIM 1024 DISTANCE_METRIC COSINE
    article_id NUMERIC
    chunk_index NUMERIC
    title TEXT
    content TEXT
```

**实现细节**:
- 使用 HNSW 算法，距离度量为 COSINE
- 索引名称: `article_embeddings`
- Key 前缀: `article:`
- 初始化在 `main.go` 应用启动时执行

## 与现有系统集成

### 4.1 修改 ArticleService

在文章发布/更新/删除操作后调用 VectorService:

```go
func (s *articleService) Publish(id int64) error {
    // 1. 更新文章状态
    article, err := s.articleRepo.GetByID(id)
    if err != nil {
        return err
    }
    article.Status = "published"
    if err := s.articleRepo.Update(article); err != nil {
        return err
    }

    // 2. 删除缓存
    cacheKey := fmt.Sprintf("article:slug:%s", article.Slug)
    s.rdb.Del(context.Background(), cacheKey)

    // 3. 向量化文章（异步，不阻塞主流程）
    go func() {
        if err := s.vectorService.VectorizeArticle(article.ID, article.Title, article.Content); err != nil {
            // 记录错误日志，但不影响文章发布
            log.Printf("Failed to vectorize article %d: %v", article.ID, err)
        }
    }()

    return nil
}

func (s *articleService) Update(id int64, req *dto.UpdateArticleRequest) error {
    // ... 更新文章逻辑 ...

    // 如果是已发布文章，重新向量化
    if article.Status == "published" {
        go func() {
            if err := s.vectorService.VectorizeArticle(article.ID, article.Title, article.Content); err != nil {
                log.Printf("Failed to re-vectorize article %d: %v", article.ID, err)
            }
        }()
    }

    return nil
}

func (s *articleService) Delete(id int64) error {
    // ... 删除文章逻辑 ...

    // 删除向量
    go func() {
        if err := s.vectorService.DeleteArticleVector(id); err != nil {
            log.Printf("Failed to delete article vector %d: %v", id, err)
        }
    }()

    return nil
}
```

**错误处理策略**:
- 向量化失败不阻塞文章操作（使用 goroutine 异步执行）
- 记录详细错误日志供排查
- 后续可添加重试机制或补偿任务

### 4.2 添加 AIHandler

创建新的 Handler 处理 AI 对话请求:

```go
// internal/handler/ai.go
type AIHandler struct {
    aiService service.AIService
}

func NewAIHandler(aiService service.AIService) *AIHandler {
    return &AIHandler{aiService: aiService}
}

// Chat 处理 AI 对话请求
// POST /api/ai/chat
// Request: { "question": "什么是 Go 语言的并发模型？" }
// Response: { "code": 0, "message": "success", "data": { "answer": "..." } }
func (h *AIHandler) Chat(c *gin.Context) {
    var req struct {
        Question string `json:"question" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        response.InvalidParams(c, "Missing question parameter")
        return
    }

    answer, err := h.aiService.Chat(req.Question)
    if err != nil {
        response.InternalError(c, "Failed to generate answer")
        return
    }

    response.Success(c, gin.H{
        "answer": answer,
    })
}
```

**路由配置** (在 `main.go` 中):
```go
// AI 对话路由（公开，无需认证）
ai := api.Group("/ai")
{
    ai.POST("/chat", aiHandler.Chat)
}
```

### 4.3 main.go 初始化流程

在 `main.go` 中添加 AI 相关组件初始化:

```go
func main() {
    // ... 现有初始化逻辑 ...

    // 6.5 初始化 eino Wrapper Layer
    embedder := eino.NewDoubaoEmbedder(&cfg.AI)
    llmClient := eino.NewDoubaoLLMClient(&cfg.AI)
    vectorStore := eino.NewRedisVectorStore(rdbVector, "article_embeddings")

    // 初始化 Redis Vector 索引（应用启动时执行一次）
    if err := vectorStore.InitIndex("article_embeddings", 1024); err != nil {
        logger.Fatal("Failed to init Redis Vector index", zap.Error(err))
    }
    logger.Info("Redis Vector index initialized successfully")

    // 7. 初始化 Service（添加 AI 相关）
    vectorService := service.NewVectorService(vectorStore, embedder, logger)
    aiService := service.NewAIService(vectorStore, embedder, llmClient, logger)

    // 修改 ArticleService 初始化（注入 VectorService）
    articleService := service.NewArticleService(articleRepo, categoryRepo, rdb, vectorService)

    // 8. 初始化 Handler（添加 AIHandler）
    aiHandler := handler.NewAIHandler(aiService)

    // 9. 初始化路由（添加 AI 路由）
    router := setupRouter(cfg, logger, db, rdb, rdbVector,
        userHandler, categoryHandler, articleHandler, commentHandler, uploadHandler, aiHandler)

    // ... 启动服务器 ...
}
```

## 配置扩展

在 `config/config.go` 中添加 AI 配置:

```go
type AIConfig struct {
    APIKey         string `mapstructure:"api_key"`
    EmbeddingModel string `mapstructure:"embedding_model"`
    LLMModel       string `mapstructure:"llm_model"`
    Temperature    float32 `mapstructure:"temperature"`
    MaxTokens      int    `mapstructure:"max_tokens"`
    TopK           int    `mapstructure:"top_k"`
}
```

在 `config/config.yaml` 中:
```yaml
ai:
  api_key: ${DOUBAO_API_KEY}
  embedding_model: doubao-embedding-v1
  llm_model: doubao-pro-32k
  temperature: 0.7
  max_tokens: 500
  top_k: 3
```

在 `config/.env.example` 中:
```bash
# AI/RAG 配置
DOUBAO_API_KEY=your_doubao_api_key_here
```

## 文件结构

新建文件:
```
backend/internal/
├── service/
│   ├── vector.go          # VectorService 实现
│   └── ai.go              # AIService 实现
├── handler/
│   └── ai.go              # AIHandler 实现
└── pkg/
    └── eino/
        ├── embedder.go    # Doubao Embedder 封装
        ├── llm.go         # Doubao LLMClient 封装
        └── vectorstore.go # RedisVectorStore 封装
```

修改文件:
```
backend/
├── config/config.go       # 添加 AIConfig
├── config/config.yaml     # 添加 ai 配置
├── config/.env.example    # 添加 DOUBAO_API_KEY
├── internal/service/article.go  # 注入 VectorService
└── cmd/server/main.go     # 添加 AI 组件初始化和路由
```

## 依赖包

需要在 `go.mod` 中添加:
```
github.com/cloudwego/eino
github.com/cloudwego/eino-ext/providers/doubao
github.com/RediSearch/redisearch-go/v2  (用于 Redis Vector 操作)
```

## 测试验证

### 1. 单元测试

测试核心组件:
- `embedder_test.go`: 测试 Embedder 调用豆包 API 生成向量
- `llm_test.go`: 测试 LLMClient 调用豆包 API 生成文本
- `vectorstore_test.go`: 测试 RedisVectorStore CRUD 操作
- `vector_service_test.go`: 测试文章分块、向量化流程
- `ai_service_test.go`: 测试 RAG 完整流程（Mock LLM 响应）

### 2. 集成测试

端到端验证:

**步骤 1: 发布文章并验证向量化**
```bash
# 1. 创建文章
curl -X POST http://localhost:8080/api/admin/articles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"title":"Go并发编程","content":"Go使用goroutine实现并发...\n\nChannel是Go的核心特性...","category_id":1}'

# 2. 发布文章
curl -X PATCH http://localhost:8080/api/admin/articles/1/publish \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# 3. 验证 Redis Vector 中存在向量
redis-cli
> FT.SEARCH article_embeddings "*" RETURN 3 article_id title content
```

**步骤 2: AI 对话**
```bash
curl -X POST http://localhost:8080/api/ai/chat \
  -d '{"question":"Go语言如何实现并发？"}' \
  -H "Content-Type: application/json"

# 预期响应:
{
  "code": 0,
  "message": "success",
  "data": {
    "answer": "根据文章内容，Go语言使用goroutine实现并发。Goroutine是Go的轻量级线程..."
  }
}
```

**步骤 3: 测试文章更新**
```bash
# 更新文章内容
curl -X PUT http://localhost:8080/api/admin/articles/1 \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d '{"title":"Go并发编程","content":"Go使用goroutine和channel实现高效并发..."}'

# 再次提问，验证检索到新内容
curl -X POST http://localhost:8080/api/ai/chat \
  -d '{"question":"Go并发有哪些特点？"}'
```

**步骤 4: 测试文章删除**
```bash
# 删除文章
curl -X DELETE http://localhost:8080/api/admin/articles/1 \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# 验证 Redis Vector 中向量已删除
redis-cli
> FT.SEARCH article_embeddings "@article_id:[1 1]"
# 应返回 0 results
```

### 3. 性能测试

监控关键指标:
- **向量化延迟**: 单篇文章向量化时间（目标 < 2s）
- **检索延迟**: Top-3 相似度搜索时间（目标 < 100ms）
- **LLM 响应时间**: 端到端 AI 对话时间（目标 < 5s）
- **并发性能**: 100 QPS 下的响应时间和错误率

## 错误处理和边界情况

### 错误处理策略

1. **向量化失败**:
   - 不阻塞文章发布/更新
   - 记录详细错误日志
   - 返回成功响应给用户
   - 后续可添加后台补偿任务

2. **AI 对话失败**:
   - 检索失败: 返回 500 错误，提示服务暂时不可用
   - LLM API 失败: 返回 500 错误，记录错误日志
   - 无相关文章: 返回友好提示"未找到相关文章，请换个问题试试"

3. **Redis Vector 不可用**:
   - 应用启动时索引初始化失败: Fatal 退出
   - 运行时连接失败: 降级处理，返回错误提示

### 边界情况

1. **空文章**: 跳过向量化，记录警告日志
2. **超长文章**: 限制最大分块数（如 100 块），超出部分截断
3. **特殊字符**: 清理文章内容，移除控制字符和无效 Unicode
4. **重复问题**: 无状态设计，每次请求独立处理

## 后续优化方向

1. **异步向量化队列**: 使用消息队列（Redis Stream）解耦文章发布和向量化
2. **增量更新**: 只向量化修改的段落，减少 API 调用
3. **多轮对话**: 支持上下文记忆，实现连续对话
4. **流式响应**: SSE 流式返回 LLM 生成内容，提升体验
5. **重排序**: 添加 Reranker 模型提升检索精度
6. **混合检索**: 结合关键词检索（BM25）和向量检索
7. **监控告警**: 添加 Prometheus 指标，监控向量化成功率、AI 响应时间等

## 总结

本设计采用同步向量化 + 简单 RAG 方案，核心特点:

1. **简单可靠**: 同步向量化确保数据一致性，逻辑清晰易维护
2. **模块化**: 三层封装（eino wrapper → Service → Handler），职责清晰
3. **无侵入**: 向量化失败不影响文章管理核心功能
4. **可扩展**: 预留异步化、多轮对话、流式响应等优化空间

技术栈完全符合原设计文档要求（eino + 豆包 + Redis Stack），与现有系统架构风格一致，可直接开始实施。
