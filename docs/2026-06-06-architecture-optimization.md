# 架构问题分析与优化方案

> 基于 2026-06-06 对全项目（backend + frontend）的深度审查，涵盖 Bug、性能瓶颈、多 Agent 协作设计缺陷三个维度，以及对应的优化方案。

---

## 第一部分：多 Agent 协作模式问题与优化

### 1.1 现状概述

ThinkTank 当前的多 Agent 协作模式本质是**串行管道**，而非真正的"圆桌讨论"：

```
Clarifier → Librarian → Journalist → Synthesizer → AcceptanceReviewer → 用户
  (LLM)      (RAG)       (HTTP)        (LLM)          (LLM)
  1-3s       0.5-1s       2-5s         2-5s           2-5s

总延迟: 7.5s - 19s（所有步骤串行等待）
```

每个 Agent 独立完成任务、产出结果、传递给下一个，Agent 之间没有协商、没有并行、没有信息回流。

---

### 1.2 问题一：Librarian 和 Journalist 强行串行执行

**文件**：`backend/internal/service/chat/thinktank_manual_flow.go:35-60`

**现状**：Librarian.Search 完成后，根据 `CoverageStatus` 判断是否需要 Journalist。它们互不依赖（本地搜索和网络搜索可以同时进行），却强制串行。

**影响**：每次对话额外浪费 2.5-6s 延迟。

**优化方案**：并行执行 Librarian 和 Journalist

```go
// 伪代码示意
var (
    localResult *LibrarianResult
    webResult   *JournalistResult
    localErr    error
    webErr      error
)
var wg sync.WaitGroup
wg.Add(2)
go func() {
    defer wg.Done()
    localResult, localErr = s.librarian.Search(ctx, query)
}()
go func() {
    defer wg.Done()
    webResult, webErr = s.journalist.Research(ctx, query, LibrarianResult{})
}()
wg.Wait()

// Synthesizer 收到两边的结果后按需组合
// 即使 Librarian 失败，Journalist 结果仍可用
```

**收益**：端到端延迟从 7.5-19s 降至 4.5-11s。

---

### 1.3 问题二：Librarian → Journalist 触发条件过于粗暴

**文件**：`backend/internal/service/chat/thinktank_librarian.go:69-85`

**现状**：用文档数量 `>= 2` 判断"足够"，完全不做语义质量评估。2 篇内容贫乏的文章算"足够"，跳过网络搜索；1 篇高质量的文章算"部分"，触发不必要的网络搜索。

**优化方案**：

- 方案 A：将判断交给 LLM（Synthesizer），让它在收到并行返回的本地 + 网络结果后自己决定如何组合
- 方案 B：如果必须保留前置判断，增加最小字符长度阈值和关键信息覆盖度检查
- 推荐方案 A：并行化后 Synthesizer 天然有全部信息，无需前置粗糙判断

---

### 1.4 问题三：Journalist 完全忽略 Librarian 的结果

**文件**：`backend/internal/service/chat/thinktank_journalist.go:46-111`

**现状**：Journalist 虽然接收了 `local LibrarianResult`，但实际发送给搜索 API 的 query 就是原始的 `question`，Librarian 发现的信息完全没被利用。

**优化方案**：用 Librarian 结果丰富 Journalist 的搜索查询

```go
func buildEnrichedSearchQuery(question string, local *LibrarianResult) string {
    if local == nil || local.Summary == "" {
        return question
    }
    // 从本地结果中提取关键信息缺口，追加特异性搜索词
    // 或使用 LLM 生成 "已知X，是否还有Y?" 式的定向查询
    return fmt.Sprintf("%s (%s)", question, extractKnowledgeGaps(local))
}
```

---

### 1.5 问题四：Agent 失败时缺乏降级路径

**文件**：`backend/internal/service/chat/thinktank_manual_flow.go:37-78`

**现状**：

| 失败场景 | 当前行为 | 合理行为 |
|----------|----------|----------|
| Librarian 失败 | 立即终止，整个回答失败 | Journalist 兜底 |
| Journalist 失败 | 立即终止，整个回答失败 | 只用 Librarian 结果回答 |
| Synthesizer 失败 | 所有检索结果丢弃 | 直接返回原始搜索摘要 + 来源 |

**优化方案**：建立三级降级链

```
Librarian ──失败──→ Journalist 兜底 ──失败──→ LLM 凭知识直接回答("抱歉，无法搜索到相关信息...")
Journalist ──失败──→ Librarian 兜底 ──失败──→ LLM 凭知识直接回答
Synthesizer ──失败──→ 模板化拼接 Librarian + Journalist 原始结果
```

在代码中以装饰器或责任链模式实现：

```go
func (o *thinkTankOrchestrator) searchWithFallback(
    ctx context.Context, query string,
) (*LibrarianResult, *JournalistResult, error) {

    local, localErr := o.service.librarian.Search(ctx, query)
    web, webErr := o.service.journalist.Research(ctx, query, localToJournalistInput(local))

    // 两边都失败才返回错误
    if localErr != nil && webErr != nil {
        return nil, nil, fmt.Errorf("all search sources failed")
    }
    return local, web, nil
}
```

---

### 1.6 问题五：Clarifier 投入产出比低

**文件**：`backend/internal/service/chat/thinktank_orchestrator.go:55-76`

**现状**：每次对话第一步是调 Clarifier（一次独立 LLM 调用），95%+ 的情况返回 `ShouldAskUser = false`，产出的"意图画像"只是追加到 Agent 查询文本末尾的一段结构化描述，对后续理解和搜索几乎没有实质帮助。这是每次白付的一次 LLM API 调用。

**优化方案**：

- 方案 A：移除 Clarifier，将意图分析的逻辑融入 Synthesizer 的 prompt 中（零额外 LLM 调用）
- 方案 B：改为异步或可选 — 仅当用户问题明显模糊（少于 5 个字或全是代词）时才触发
- 方案 C：让 Clarifier 产出具体指导（如"搜索重点：XX 方面"、"本地知识库已知：XX 方面"），而不是生成一个 JSON 画像

---

### 1.7 问题六：修订 = 完整重跑整个流程

**文件**：`backend/internal/service/chat/thinktank_intent_review.go:13, 535-556`

**现状**：`maxReviewRevisions = 1`，且修订方式是把整个原始答案 + 缺失维度垒成文本块，**重新跑一遍完整的 ADK 运行器**（Planner → Executor → Replanner）。这不是修订，是重做。

**优化方案**：改为定向修复

```go
// 将审查器的反馈转化为定向的补充指令
// 不要跑完整的 Planner/Executor，而是直接让 Synthesizer 在已有答案基础上补充
func (o *thinkTankOrchestrator) reviseAnswerTargeted(
    ctx context.Context,
    originalAnswer string,
    review AcceptanceReview,
) (string, error) {
    // 只补充缺失维度，不需要重新搜索
    revisionPrompt := buildTargetedRevisionPrompt(originalAnswer, review.MissingDimensions)
    return o.service.synthesizer.Refine(ctx, revisionPrompt)
}
```

同时去掉 `maxReviewRevisions = 1` 的硬限制，改为最大 2-3 次，每次修订都缩小范围。

---

### 1.8 问题七：ADK Checkpoint 存储在进程内存

**文件**：`backend/internal/service/chat/thinktank_adk.go:110-135`

**现状**：`thinkTankCheckpointStore` 是 `sync.Mutex + map[string][]byte` 的进程内存储。服务重启/部署/水平扩展时，所有进行中的 ADK 运行全部丢失。

**优化方案**：将 Checkpoint 存储改为 Redis

```go
type redisCheckpointStore struct {
    rdb *redis.Client
}

func (s *redisCheckpointStore) Get(ctx context.Context, id string) ([]byte, error) {
    return s.rdb.Get(ctx, "adk:checkpoint:"+id).Bytes()
}

func (s *redisCheckpointStore) Set(ctx context.Context, id string, data []byte) error {
    return s.rdb.Set(ctx, "adk:checkpoint:"+id, data, 30*time.Minute).Err()
}
```

**收益**：服务重启后用户可无缝恢复进行中的对话。

---

### 1.9 问题八：Replanner 的否定规则堆积

**文件**：`backend/internal/service/chat/thinktank_adk.go:54-61`

**现状**：一个不断增长的"不要提及 X / 不要描述 Y"规则列表。每当 LLM 暴露内部信息，就加一条否定规则。否定约束是脆弱的 — LLM 经常无法完美遵守。

**优化方案**：使用结构化 JSON Schema 约束输出格式

让 Replanner 的输出分成两个字段，天然隔离过程内容和最终答案：

```json
{
  "plan_status": {
    "steps_executed": 3,
    "sources_consulted": ["local", "web"],
    ...
  },
  "response": "用户看到清晰答案..."
}
```

配合 JSON mode / structured output API，而不是用自由文本 + 否定约束。

---

### 1.10 问题九：执行路径由依赖注入决定，用户不可控

**文件**：`backend/internal/service/chat/thinktank_orchestrator.go:225-337`

**现状**：走 ADK 流程还是手动流程，取决于 `NewThinkTankService` 构造时是否成功注入了 `adkRunner`。用户无法选择，代码也不透明。

**优化方案**：将执行策略显式化为配置项或请求参数

```go
type ThinkTankExecutionMode string

const (
    ModeADK       ThinkTankExecutionMode = "adk"
    ModeManual    ThinkTankExecutionMode = "manual"
    ModeAuto      ThinkTankExecutionMode = "auto" // 默认，ADK 可用则用
)
```

---

## 第二部分：性能优化

### 2.1 数据库层面

#### 问题 2.1.1：文章列表 N+1 预加载

**文件**：`backend/internal/repository/article/article.go:138`

```go
// 现状
r.db.Preload("Category").Preload("Author").Offset(offset).Limit(limit).Find(&articles)
// 20 篇文章 = 1 + 20 + 20 = 41 条 SQL
```

**优化方案**：

```go
// 方案 A：Preload 指定字段 + batch
r.db.Preload("Category", func(db *gorm.DB) *gorm.DB {
    return db.Select("id", "name")
}).Preload("Author", func(db *gorm.DB) *gorm.DB {
    return db.Select("id", "username", "avatar")
}).Offset(offset).Limit(limit).Find(&articles)

// 方案 B：用 JOIN（更快）
r.db.Select("articles.*, categories.name as category_name, users.username as author_name").
    Joins("LEFT JOIN categories ON categories.id = articles.category_id").
    Joins("LEFT JOIN users ON users.id = articles.author_id").
    Offset(offset).Limit(limit).Find(&articles)
```

**收益**：SQL 从 41 条降至 1 条。

---

#### 问题 2.1.2：Save() 全量更新无变更的字段

**文件**：`backend/internal/repository/article/article.go:161, 229`

```go
// 现状：更新 Popularity 时把 Content（longtext）也写回 DB
r.db.Save(article)
```

**优化方案**：

```go
// 只更新变更的字段
r.db.Model(&model.Article{}).Where("id = ?", article.ID).UpdateColumns(map[string]interface{}{
    "popularity": article.Popularity,
})
```

**收益**：Popularity 更新操作的数据量从 50-200KB 降至几十字节。

---

#### 问题 2.1.3：GetAllPublished 拉全表无分页

**文件**：`backend/internal/repository/article/article.go:234`

**现状**：Popularity 定时任务 `GetAllPublished()` 拉所有文章（含 longtext content），如果有 10000 篇文章会占用数百 MB 内存。

**优化方案**：

```go
func (r *articleRepository) GetAllPublished(batchSize int, offset int) ([]model.Article, error) {
    var articles []model.Article
    err := r.db.Select("id", "view_count", "comment_count", "like_count", "published_at", "created_at", "popularity").
        Where("status = ?", model.ArticleStatusPublished).
        Order("id ASC").
        Offset(offset).Limit(batchSize).
        Find(&articles).Error
    return articles, err
}
```

**收益**：Popularity 更新从一次性处理变为分批处理，内存占用可控。

---

#### 问题 2.1.4：缺失复合索引

**文件**：`backend/internal/model/chat_message.go`、`conversation_memory.go`

**现状**：
- `chat_messages` 表仅单列索引 `idx_conversation(conversation_id)`，查询 `ORDER BY created_at` 触发 filesort
- `conversation_memory` 表同样缺少 `(conversation_id, scope, updated_at)` 复合索引

**优化方案**：添加复合索引

```sql
ALTER TABLE chat_messages ADD INDEX idx_conv_created (conversation_id, created_at);
ALTER TABLE conversation_memory ADD INDEX idx_mem_conv_scope (conversation_id, scope, updated_at);
```

**收益**：每次对话加载历史消息时消除 filesort。

---

### 2.2 Redis 层面

#### 问题 2.2.1：文章缓存惊群效应

**文件**：`backend/internal/service/article/article_read.go:28`

**现状**：热门文章 100 个并发请求同时 cache miss → 100 个 goroutine 都去查 DB 然后争抢写同一个 Redis key。

**优化方案**：引入 singleflight 模式

```go
import "golang.org/x/sync/singleflight"

var articleCacheGroup singleflight.Group

func (s *articleService) GetByIDWithCache(ctx context.Context, id int64) (*model.Article, error) {
    // singleflight 保证同一个 key 只有一个 goroutine 去查 DB
    result, err, _ := articleCacheGroup.Do(fmt.Sprintf("article:%d", id), func() (interface{}, error) {
        return s.repo.GetByID(id)
    })
    // ...
}
```

---

#### 问题 2.2.2：slug 查询不走缓存

**文件**：`backend/internal/service/article/article_read.go:33`

**现状**：`GetBySlug` 查 DB → 写 `article:detail:<id>` 缓存 → 下次用 slug 再查同一篇文章还是 miss。

**优化方案**：增加 slug 到 ID 的映射缓存

```go
func (s *articleService) GetBySlug(slug string) (*model.Article, error) {
    // 先查 slug 映射
    id, err := s.rdb.Get(ctx, "article:slug:"+slug).Int64()
    if err == nil {
        return s.GetByIDWithCache(ctx, id) // 复用 ID 缓存
    }
    article, err := s.repo.GetBySlug(slug)
    // 同时缓存 slug 映射和 article 详情
    s.rdb.Set(ctx, "article:slug:"+slug, article.ID, 30*time.Minute)
    return article, err
}
```

---

#### 问题 2.2.3：限流是固定窗口，边界有漏洞

**文件**：`backend/internal/middleware/ratelimit.go:109-123`

**现状**：`INCR + EXPIRE` 实现固定窗口。59 秒用满 10 次 + 01 秒再 10 次 = 2 秒通过 20 次。

**优化方案**：改用滑动窗口算法（Lua 脚本）

```lua
-- 滑动窗口限流 Lua 脚本
-- KEYS[1]: 限流 key, ARGV[1]: 窗口大小(ms), ARGV[2]: 限制次数
local now = redis.call('TIME')
local window = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local key = KEYS[1]
redis.call('ZREMRANGEBYSCORE', key, 0, now[1]*1000 + now[2]/1000 - window)
local count = redis.call('ZCARD', key)
if count < limit then
    redis.call('ZADD', key, now[1]*1000 + now[2]/1000, now[1]*1000 .. now[2])
    redis.call('PEXPIRE', key, window)
    return 1
end
return 0
```

---

### 2.3 前端层面

#### 问题 2.3.1：每次 stream chunk 深拷贝整个对话树

**文件**：`frontend/src/store/chatStore.ts:496-519`

**现状**：`onChunk` 事件（LLM 每吐一个 token 都触发）对整个 `conversations` 对象做深层 spread 拷贝。导致侧边栏、消息列表、导航栏等所有消费 `conversations` 的组件全部重渲染。

**优化方案**：拆分 store

```typescript
// store/streamingStore.ts — 只存当前流式输出的文本
// 每次 chunk 只更新这一个 string，只有当前消息组件重渲染

// store/chatStore.ts — 存对话列表、消息列表等
// 只在流式完成（onDone）时才更新
```

**收益**：流式输出时前端 CPU 占用降低 80%+，渲染性能质变。

---

#### 问题 2.3.2：SSE buffer 字符串拼接每次重新分配

**文件**：`frontend/src/api/chat.ts:40`

**现状**：`buffer += decoder.decode(...)` 每次 chunk 重新分配整个 buffer。

**优化方案**：用数组收集 chunk，仅在需要解析时才 join

```typescript
const decoder = new TextDecoder()
const chunks: string[] = []

// 收集
const text = decoder.decode(value, { stream: true })
if (text) chunks.push(text)

// 需要时再解析
const full = chunks.join('')
const events = full.split('\n\n')
```

---

#### 问题 2.3.3：缺失路由级 code splitting

**文件**：`frontend/vite.config.ts:34-75`

**现状**：`manualChunks` 只拆了 vendor 库，业务代码全在一个 bundle。访问登录页的用户也要下载 AI Chat 和 3D 星球的所有代码。

**优化方案**：增加路由级 split

```typescript
rollupOptions: {
  output: {
    manualChunks(id) {
      if (id.includes('node_modules')) {
        // 现有的 vendor split...
      }
      // 新增：按路由拆分
      if (id.includes('/src/pages/AIChat') || id.includes('/src/views/ai-chat')) return 'page-chat'
      if (id.includes('/src/pages/{admin}') || id.includes('/src/views/admin')) return 'page-admin'
      if (id.includes('three') || id.includes('@react-three')) return 'vendor-three'
    }
  }
}
```

---

#### 问题 2.3.4：流式结束后冗余 API 请求

**文件**：`frontend/src/store/chatStore.ts:524`

**现状**：`onDone` 回调又调 `chatApi.getConversation()` 拉一次完整数据，但流式过程已经把所有 step 和完整答案都传过来了。

**优化方案**：直接使用流式过程中积累的完整数据，不再额外发 API 请求。

---

### 2.4 LLM / AI 路径

#### 问题 2.4.1：每次对话从 DB 加载全量历史消息

**文件**：`backend/internal/service/chat/thinktank_orchestrator.go:110, 245`

**现状**：`loadHistory(conv.ID)` → 查 MySQL `chat_messages` 全表，200 轮对话时每次拉 ~50KB。

**优化方案**：对话历史用 Redis 缓存

```go
func (c *thinkTankConversations) loadHistoryWithCache(convID int64) ([]model.ChatMessage, error) {
    key := fmt.Sprintf("chat:history:%d", convID)
    data, err := c.rdb.Get(ctx, key).Bytes()
    if err == nil {
        var msgs []model.ChatMessage
        json.Unmarshal(data, &msgs)
        return msgs, nil
    }
    msgs, err := c.messageRepo.GetByConversationID(convID)
    // 缓存 10 分钟
    data, _ := json.Marshal(msgs)
    c.rdb.Set(ctx, key, data, 10*time.Minute)
    return msgs, err
}
```

---

#### 问题 2.4.2：用 rune 计数代替 token 计数

**文件**：`backend/internal/service/chat/thinktank_memory.go:236`

**现状**：`recentMemoryRuneBudget` 用 Unicode 字符数做内存预算。中文 ~1.8 字符/token，英文 ~4 字符/token，rune 计数严重不准。

**优化方案**：引入 tokenizer 做真实 token 计数

```go
import "github.com/pkou/go-tiktoken"

var enc, _ = tiktoken.GetEncoding("cl100k_base") // OpenAI 编码

func countTokens(text string) int {
    tokens, _ := enc.Encode(text, nil, nil)
    return len(tokens)
}
```

---

#### 问题 2.4.3：内存构建时大量的 `[]rune()` 转换

**文件**：`backend/internal/service/chat/thinktank_memory.go:223`

**现状**：`selectRecentMemoryStart` 在循环中对每条历史消息做 `[]rune()` 转换（200 条消息 = 200 次分配）。

**优化方案**：用 `utf8.RuneCountInString()` 原地计数，不需要分配

```go
// 现状
for _, msg := range history {
    charCount += len([]rune(msg.Content))
}

// 改为
for _, msg := range history {
    charCount += utf8.RuneCountInString(msg.Content)
}
```

---

#### 问题 2.4.4：验收审核阻塞流式输出

**文件**：`backend/internal/service/chat/thinktank_orchestrator.go:291`

**现状**：ADK 流式完成后同步调 `reviewAnswer()` → 2-5s → 可能还同步调修订 → 2-5s。用户一直看"审核中..."。

**优化方案**：答案先展示，审核异步跑

```
流式结束 → 立即 emit 完整答案给用户
         → 后台异步运行审核
            → 通过：无操作
            → 需要修订：追加一条"补充说明"消息
            → 需要问用户：追加一条交互问题
```

---

### 2.5 HTTP 层面

#### 问题 2.5.1：缺失 gzip 压缩中间件

**文件**：`backend/cmd/server/bootstrap_http.go:47-55`

**优化方案**：

```go
import "github.com/gin-contrib/gzip"

router.Use(gzip.Gzip(gzip.DefaultCompression))

// 对 SSE 流式响应置不同的 compression level
// 或用自定义的 gzip.ExcludedExtensions 排除 SSE endpoint
```

**收益**：文章列表 50-200KB 的 JSON 响应压缩到 10-30KB。

---

#### 问题 2.5.2：HTTP Server 无超时配置

**文件**：`backend/cmd/server/app.go:93-96`

**优化方案**：

```go
srv := &http.Server{
    Addr:              ":" + cfg.Server.Port,
    Handler:           router,
    ReadTimeout:       10 * time.Second,
    ReadHeaderTimeout: 5 * time.Second,
    WriteTimeout:      0,          // 0 因为 SSE 需要长连接
    IdleTimeout:       120 * time.Second,
    MaxHeaderBytes:    1 << 20,    // 1MB
}
```

---

#### 问题 2.5.3：无优雅退出

**文件**：`backend/cmd/server/app.go:93`

**优化方案**：

```go
go func() {
    if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
        logger.Fatal("server stopped unexpectedly", zap.Error(err))
    }
}()

quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := srv.Shutdown(ctx); err != nil {
    logger.Error("forced shutdown", zap.Error(err))
}
```

---

## 第三部分：Bug 修复

### 3.1 run_hub.go channel 关闭竞态

**文件**：`backend/internal/service/chat/run_hub.go:86-113`

**问题**：`publish` 在释放锁后向 subscriber channel 发送事件，但 subscriber 可能在另一个 goroutine 中被取消并关闭 channel，导致向已关闭 channel 发送数据的 panic。

**修复方案**：

```go
// 在 RunEntry 中增加 closed 标记
type RunEntry struct {
    mu          sync.Mutex
    closed      bool           // 新增
    snapshot    *RunSnapshot
    subscribers map[chan StreamEvent]struct{}
}

func (e *RunEntry) subscribe(ch chan StreamEvent) (cancel func()) {
    e.mu.Lock()
    defer e.mu.Unlock()
    if e.closed {
        return func() {}
    }
    e.subscribers[ch] = struct{}{}
    return func() {
        e.mu.Lock()
        defer e.mu.Unlock()
        delete(e.subscribers, ch)
    }
}

// publish 时判断 closed 标记
func (h *chatRunHub) publish(runID int64, event StreamEvent) {
    h.mu.RLock()
    entry := h.runs[runID]
    h.mu.RUnlock()
    if entry == nil {
        return
    }
    entry.mu.Lock()
    if entry.closed {
        entry.mu.Unlock()
        return
    }
    subscribers := make([]chan StreamEvent, 0, len(entry.subscribers))
    for ch := range entry.subscribers {
        subscribers = append(subscribers, ch)
    }
    entry.mu.Unlock()
    // ... 安全发送
}
```

---

### 3.2 Chat 方法无法取消

**文件**：`backend/internal/pkg/eino/llm.go:148`

**问题**：`Chat` 方法内部使用 `context.Background()`，调用方无法取消或超时。

**修复方案**：修改接口签名

```go
type LLMClient interface {
    Chat(ctx context.Context, messages []ChatMessage) (string, error)
    ChatStream(ctx context.Context, messages []ChatMessage) (<-chan string, <-chan error)
    GetModel() model.ChatModel
}
```

同步修改所有调用方（Synthesizer、MemorySummarizer、RAGChain 等）传入对应 context。

---

### 3.3 HTTP Response Body 未 drain

**文件**：`thinktank_tools.go:274, 414`、`thinktank_journalist.go:83`、`auth/oauth.go:72, 120, 143`

**问题**：`defer resp.Body.Close()` 没有先 drain body，剩余字节阻止 HTTP 连接复用。

**修复方案**：

```go
defer func() {
    io.Copy(io.Discard, resp.Body)
    resp.Body.Close()
}()
```

---

### 3.4 AI 回答泄漏内部调试信息

**详见本文档 1.9 节**

WebFetch 错误文本"证据局限性: 百度百科返回状态码 404"泄漏到用户回答中。

**修复**：
- `thinktank_adk_resume.go:isNonFinalToolLimitationAnswer` 补充 WebFetch 错误标记
- `thinktank_orchestrator.go:692-694` Executor 推理文本不应不加过滤地进入 adkWebNotes

---

### 3.5 Embedder 硬编码导致多 provider 兼容性问题

**文件**：`backend/cmd/server/bootstrap_infra.go:104`

**问题**：embedder 固定使用 Doubao，配置 `AI_PROVIDER=deepseek` 时 embedder 初始化失败导致 AI 功能全不可用。

**修复方案**：embedder 失败时优雅降级（向量搜索不可用，但 LLM 对话正常），同时日志明确记录降级信息。

---

## 第四部分：架构级优化路线图

### 短中期（1-3 次迭代）

| 优先级 | 项目 | 工作量 | 收益 |
|--------|------|--------|------|
| P0 | Librarian + Journalist 并行化 | 中 | 延迟降低 40% |
| P0 | Agent 失败降级链 | 中 | 可用性大幅提升 |
| P0 | run_hub channel race 修复 | 小 | 稳定性 |
| P0 | Chat context 传递 | 中 | 可取消性 |
| P1 | 文章缓存 singleflight | 小 | 防止 DB 惊群 |
| P1 | 前端 stream store 拆分 | 中 | CPU 降低 80% |
| P1 | Checkpoint 存储改 Redis | 中 | 重启不丢运行 |
| P1 | DB N+1 查询修复 | 小 | SQL 量降低 97% |
| P2 | Clarifier 移除或条件化 | 小 | 每次省 1 次 LLM 调用 |
| P2 | 修订改为定向修复 | 中 | 减少不必要的重复搜索 |
| P2 | token 计数替代 rune 计数 | 小 | context window 准确 |
| P2 | gzip 中间件 | 小 | 带宽减半 |
| P3 | HTTP timeouts + 优雅退出 | 小 | 运维稳定性 |
| P3 | 前端路由级 code splitting | 中 | 首屏体积减小 |

---

> **文档版本**：v1.0
> **创建日期**：2026-06-06
> **上一版本文档**：`docs/2026-06-05-code-review-requirements.md`（Bug + 扩展需求）
> **本次聚焦**：多 Agent 架构设计缺陷 + 性能优化方案
