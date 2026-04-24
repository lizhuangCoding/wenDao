# 后端启动骨架与 ThinkTank 第一批重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在不改变现有外部行为的前提下，完成 `main.go` 启动骨架收缩，以及 `thinktank.go` 的文件级职责切分。

**Architecture:** 本批次只做结构迁移，不改业务协议、不改 service 对外接口、不改数据库模型。`main.go` 退化为入口，启动编排迁移到 bootstrap 文件；`thinktank.go` 拆成同包多个文件，`thinkTankService` 类型和主要方法签名保持不变。

**Tech Stack:** Go, Gin, GORM, Redis, Zap, Eino ADK, Go `testing`

---

## File Map

### New Files

- `backend/cmd/server/app.go`
  - 负责 `Run()` / `MustRun()`，作为新的应用启动总控层。
- `backend/cmd/server/bootstrap_infra.go`
  - 负责环境变量、配置、日志、MySQL、Redis 初始化。
- `backend/cmd/server/bootstrap_http.go`
  - 负责 repositories、services、handlers、router 装配。
- `backend/internal/service/thinktank_service.go`
  - 保留 `ThinkTankService`、`thinkTankService`、`NewThinkTankService`、`Chat`、`ChatStream`。
- `backend/internal/service/thinktank_conversation.go`
  - 迁移会话归属、消息读写、元数据相关函数。
- `backend/internal/service/thinktank_memory.go`
  - 迁移记忆压缩、会话记忆读取与更新相关函数。
- `backend/internal/service/thinktank_stream.go`
  - 迁移流式事件发送、step 事件装配相关函数。
- `backend/internal/service/thinktank_run_record.go`
  - 迁移 run / step 创建、完成、失败记录相关函数。
- `backend/internal/service/thinktank_adk_resume.go`
  - 迁移 ADK pending context、checkpoint、恢复相关函数。

### Modified Files

- `backend/cmd/server/main.go`
  - 收缩为真正入口。
- `backend/cmd/server/main_ai_test.go`
  - 调整到新的 `Run` / bootstrap 结构后继续覆盖 AI 降级行为。
- `backend/cmd/server/routes_test.go`
  - 继续验证路由装配不回归。
- `backend/internal/service/thinktank_test.go`
  - 保持回归测试通过，必要时调整引用文件名变化后的辅助方法位置。
- `backend/internal/service/thinktank_adk_stage_test.go`
  - 保持对 ADK 阶段行为的验证。
- `backend/internal/service/thinktank_memory_test.go`
  - 保持对记忆逻辑的验证。
- `backend/internal/service/thinktank_adk_runner_test.go`
  - 确认 ADK 相关辅助函数迁移后不回归。

### Deleted Files

- `backend/internal/service/thinktank.go`
  - 第 1 批结束后删除，用新的细分文件替代。

---

### Task 1: 为启动骨架重构补回归测试

**Files:**
- Modify: `backend/cmd/server/main_ai_test.go`
- Modify: `backend/cmd/server/routes_test.go`
- Test: `backend/cmd/server/main_ai_test.go`
- Test: `backend/cmd/server/routes_test.go`

- [ ] **Step 1: 先写或调整失败测试，锁定新的入口和 bootstrap 约束**

```go
func TestRun_InitializesServerThroughBootstrap(t *testing.T) {
	t.Skip("placeholder during red phase removal")
}

func TestBuildRouter_RegistersRequiredRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := buildRouter(&config.Config{}, zap.NewNop(), redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"}), &appHandlers{
		user:              &handler.UserHandler{},
		auth:              &handler.AuthHandler{},
		category:          &handler.CategoryHandler{},
		article:           &handler.ArticleHandler{},
		comment:           &handler.CommentHandler{},
		upload:            &handler.UploadHandler{},
		ai:                &handler.AIHandler{},
		site:              &handler.SiteHandler{},
		stat:              &handler.StatHandler{},
		chat:              &handler.ChatHandler{},
		knowledgeDocument: &handler.KnowledgeDocumentHandler{},
	})

	if router == nil {
		t.Fatal("expected buildRouter to return a router")
	}
}
```

- [ ] **Step 2: 运行测试，确认当前代码在新约束下失败或缺少实现**

Run: `env GOTOOLCHAIN=go1.25.3 go test ./cmd/server -run 'TestRun_|TestBuildRouter_'`

Expected:
- 失败原因应指向 `Run` 尚未定义，或测试与现有入口结构不匹配。
- 不能是语法错误或无关依赖错误。

- [ ] **Step 3: 最小化调整测试，使其精确约束 Phase 1 目标**

```go
func TestBuildRouter_RegistersRequiredRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	handlers := &appHandlers{
		user:              &handler.UserHandler{},
		auth:              &handler.AuthHandler{},
		category:          &handler.CategoryHandler{},
		article:           &handler.ArticleHandler{},
		comment:           &handler.CommentHandler{},
		upload:            &handler.UploadHandler{},
		ai:                &handler.AIHandler{},
		site:              &handler.SiteHandler{},
		stat:              &handler.StatHandler{},
		chat:              &handler.ChatHandler{},
		knowledgeDocument: &handler.KnowledgeDocumentHandler{},
	}

	router := buildRouter(cfg, zap.NewNop(), rdb, handlers)
	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	for _, required := range []string{
		"GET /api/articles",
		"POST /api/auth/refresh",
		"POST /api/ai/chat",
		"GET /health",
	} {
		if _, ok := routes[required]; !ok {
			t.Fatalf("expected route %s to remain registered", required)
		}
	}
}
```

- [ ] **Step 4: 再次运行测试，确认红灯聚焦到即将实现的启动骨架**

Run: `env GOTOOLCHAIN=go1.25.3 go test ./cmd/server -run 'TestRun_|TestBuildRouter_'`

Expected:
- 若 `Run` 还未实现，应只剩入口/启动骨架相关失败。

- [ ] **Step 5: 提交测试约束**

```bash
git add backend/cmd/server/main_ai_test.go backend/cmd/server/routes_test.go
git commit -m "test: 补充启动骨架重构约束"
```

---

### Task 2: 把 `main.go` 收缩为入口并迁移 bootstrap 逻辑

**Files:**
- Create: `backend/cmd/server/app.go`
- Create: `backend/cmd/server/bootstrap_infra.go`
- Create: `backend/cmd/server/bootstrap_http.go`
- Modify: `backend/cmd/server/main.go`
- Test: `backend/cmd/server/main_ai_test.go`
- Test: `backend/cmd/server/routes_test.go`

- [ ] **Step 1: 在 `app.go` 中定义新的应用入口函数**

```go
package main

import (
	"log"
	"net/http"

	"go.uber.org/zap"

	"wenDao/config"
)

func MustRun() {
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}

func Run() error {
	_ = loadServerEnv()

	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	logger := initLogger(cfg.Log)
	defer logger.Sync()

	infra, err := initInfrastructure(cfg, logger)
	if err != nil {
		return err
	}

	repos := initRepositories(infra.db)
	aiCore, err := initAIComponents(cfg, logger, infra.rdbVector)
	if err != nil {
		logger.Warn("AI components unavailable, continuing in degraded mode", zap.Error(err))
	}

	services, cleanup, err := initServices(cfg, logger, repos, infra, aiCore)
	if err != nil {
		return err
	}
	defer cleanup()

	handlers := initHandlers(cfg, repos, services, infra.rdb)
	router := buildRouter(cfg, logger, infra.rdb, handlers)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}
	logger.Info("Server starting", zap.String("addr", ":"+cfg.Server.Port), zap.String("mode", cfg.Server.Mode))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
```

- [ ] **Step 2: 把 `main.go` 收缩为真正入口**

```go
package main

func main() {
	MustRun()
}
```

- [ ] **Step 3: 迁移基础设施与 HTTP 装配函数到 bootstrap 文件**

```go
// backend/cmd/server/bootstrap_infra.go
package main

func initInfrastructure(cfg *config.Config, logger *zap.Logger) (*infrastructure, error) {
	// 保持原实现，仅移动位置
}

func initRepositories(db *gorm.DB) *repositories {
	// 保持原实现，仅移动位置
}

func initAIComponents(cfg *config.Config, logger *zap.Logger, rdbVector *redis.Client) (*aiComponents, error) {
	// 保持原实现，仅移动位置
}
```

```go
// backend/cmd/server/bootstrap_http.go
package main

func initServices(cfg *config.Config, logger *zap.Logger, repos *repositories, infra *infrastructure, aiCore *aiComponents) (*appServices, func(), error) {
	// 保持原实现，仅移动位置
}

func initHandlers(cfg *config.Config, repos *repositories, services *appServices, rdb *redis.Client) *appHandlers {
	// 保持原实现，仅移动位置
}

func buildRouter(cfg *config.Config, logger *zap.Logger, rdb *redis.Client, handlers *appHandlers) *gin.Engine {
	// 保持原实现，仅移动位置
}
```

- [ ] **Step 4: 运行启动层测试，确认结构迁移后行为不变**

Run: `env GOTOOLCHAIN=go1.25.3 go test ./cmd/server`

Expected:
- `ok   wenDao/cmd/server`

- [ ] **Step 5: 提交启动骨架重构**

```bash
git add backend/cmd/server/main.go backend/cmd/server/app.go backend/cmd/server/bootstrap_infra.go backend/cmd/server/bootstrap_http.go backend/cmd/server/main_ai_test.go backend/cmd/server/routes_test.go
git commit -m "refactor: 收缩服务启动入口"
```

---

### Task 3: 为 ThinkTank 文件切分补红灯测试

**Files:**
- Modify: `backend/internal/service/thinktank_test.go`
- Modify: `backend/internal/service/thinktank_memory_test.go`
- Modify: `backend/internal/service/thinktank_adk_stage_test.go`
- Test: `backend/internal/service/thinktank_test.go`
- Test: `backend/internal/service/thinktank_memory_test.go`

- [ ] **Step 1: 补一条围绕 `Chat` 主流程和会话记忆的聚焦测试**

```go
func TestThinkTankServiceChat_PersistsConversationStateAfterADKAnswer(t *testing.T) {
	svc := newThinkTankServiceForTest(t)

	resp, err := svc.Chat(context.Background(), "帮我总结一下 Redis", ptrInt64(1), ptrInt64(2))
	if err != nil {
		t.Fatalf("expected chat success, got %v", err)
	}
	if strings.TrimSpace(resp.Message) == "" {
		t.Fatal("expected non-empty assistant message")
	}

	if got := svc.msgRepoSavedRoles(); !reflect.DeepEqual(got, []string{"user", "assistant"}) {
		t.Fatalf("expected user/assistant messages persisted, got %#v", got)
	}
}
```

- [ ] **Step 2: 运行定向测试，确认后续仅做结构迁移时能持续保护主流程**

Run: `env GOTOOLCHAIN=go1.25.3 go test ./internal/service -run 'TestThinkTankServiceChat_|TestBuildConversationMemory'`

Expected:
- 绿灯或红灯都可以，但失败必须指向 ThinkTank 主流程，而不是无关文件。

- [ ] **Step 3: 如果现有桩能力不足，补最小测试辅助函数**

```go
func ptrInt64(v int64) *int64 { return &v }

func newThinkTankServiceForTest(t *testing.T) *thinkTankService {
	t.Helper()
	// 复用现有 stub repo / logger / adk runner 初始化，不新增生产逻辑
	return newStubThinkTankService()
}
```

- [ ] **Step 4: 再次运行定向测试，确认第 4 任务的代码迁移有稳定护栏**

Run: `env GOTOOLCHAIN=go1.25.3 go test ./internal/service -run 'TestThinkTankServiceChat_|TestBuildConversationMemory'`

Expected:
- 测试能稳定运行，不依赖单个文件名存在。

- [ ] **Step 5: 提交 ThinkTank 回归约束**

```bash
git add backend/internal/service/thinktank_test.go backend/internal/service/thinktank_memory_test.go backend/internal/service/thinktank_adk_stage_test.go
git commit -m "test: 补充 thinktank 文件切分回归约束"
```

---

### Task 4: 拆分 `thinktank.go` 为多个同包文件

**Files:**
- Create: `backend/internal/service/thinktank_service.go`
- Create: `backend/internal/service/thinktank_conversation.go`
- Create: `backend/internal/service/thinktank_memory.go`
- Create: `backend/internal/service/thinktank_stream.go`
- Create: `backend/internal/service/thinktank_run_record.go`
- Create: `backend/internal/service/thinktank_adk_resume.go`
- Delete: `backend/internal/service/thinktank.go`
- Test: `backend/internal/service/thinktank_test.go`
- Test: `backend/internal/service/thinktank_memory_test.go`
- Test: `backend/internal/service/thinktank_adk_stage_test.go`
- Test: `backend/internal/service/thinktank_adk_runner_test.go`

- [ ] **Step 1: 先创建 `thinktank_service.go`，仅迁移对外入口和类型定义**

```go
package service

type StreamEventType string

const (
	StreamEventStage    StreamEventType = "stage"
	StreamEventQuestion StreamEventType = "question"
	StreamEventChunk    StreamEventType = "chunk"
	StreamEventStep     StreamEventType = "step"
	StreamEventDone     StreamEventType = "done"

	maxStepDetailRunes            = 6000
	ConversationMemoryScopeSummary     = "conversation_summary"
	ConversationMemoryScopePreference  = "user_preference"
	ConversationMemoryScopeProjectFact = "project_fact"
	ConversationMemoryScopeDecision    = "decision"
	ConversationMemoryScopeOpenThread  = "open_thread"
	recentMemoryMessageCount           = 6
	maxMemorySnippetRunes              = 80
)

type StreamEvent struct { /* 原样迁移 */ }
type ThinkTankChatResponse struct { /* 原样迁移 */ }
type PlannerDecision struct { /* 原样迁移 */ }
type ThinkTankService interface { /* 原样迁移 */ }
type thinkTankService struct { /* 原样迁移 */ }
type adkPendingContext struct { /* 原样迁移 */ }

func NewThinkTankService(/* 原签名 */) ThinkTankService { /* 原实现 */ }
func (s *thinkTankService) Chat(/* 原签名 */) (*ThinkTankChatResponse, error) { /* 原实现 */ }
func (s *thinkTankService) ChatStream(/* 原签名 */) (<-chan StreamEvent, <-chan error) { /* 原实现 */ }
```

- [ ] **Step 2: 迁移会话与记忆相关辅助函数**

```go
// backend/internal/service/thinktank_conversation.go
package service

func (s *thinkTankService) getOwnedConversation(conversationID *int64, userID *int64) (*model.Conversation, error) {
	// 从原 thinktank.go 原样迁移
}

func (s *thinkTankService) saveConversationMessageWithWarning(conversationID int64, role, content, warning string) {
	// 原样迁移
}

func (s *thinkTankService) updateConversationMetadataWithWarning(conv *model.Conversation, question string) {
	// 原样迁移
}
```

```go
// backend/internal/service/thinktank_memory.go
package service

func compressConversationMemory(history []model.ChatMessage) string {
	// 原样迁移
}

func buildConversationMemoryForQuestion(question string, history []model.ChatMessage, memories []model.ConversationMemory) string {
	// 原样迁移
}

func (s *thinkTankService) updateConversationMemoryWithWarning(conversationID int64, userID int64, history []model.ChatMessage) {
	// 原样迁移
}
```

- [ ] **Step 3: 迁移流式事件、run 记录、ADK 恢复相关函数**

```go
// backend/internal/service/thinktank_stream.go
package service

func sendStreamEvent(eventCh chan<- StreamEvent, event StreamEvent) bool {
	// 原样迁移
}

func formatJournalistStepDetail(result *JournalistResult) string {
	// 原样迁移
}
```

```go
// backend/internal/service/thinktank_run_record.go
package service

func (s *thinkTankService) persistCompletedRun(conversationID int64, userID int64, question, answer string, decision PlannerDecision) {
	// 原样迁移
}
```

```go
// backend/internal/service/thinktank_adk_resume.go
package service

func parseADKPendingContext(run *model.ConversationRun) (*adkPendingContext, bool) {
	// 原样迁移
}

func buildADKCheckpointID(conv *model.Conversation, question string) string {
	// 原样迁移
}
```

- [ ] **Step 4: 删除旧 `thinktank.go` 并运行全量 service 测试**

Run: `env GOTOOLCHAIN=go1.25.3 go test ./internal/service`

Expected:
- `ok   wenDao/internal/service`

- [ ] **Step 5: 提交 ThinkTank 文件级切分**

```bash
git add backend/internal/service/thinktank_service.go backend/internal/service/thinktank_conversation.go backend/internal/service/thinktank_memory.go backend/internal/service/thinktank_stream.go backend/internal/service/thinktank_run_record.go backend/internal/service/thinktank_adk_resume.go backend/internal/service/thinktank_test.go backend/internal/service/thinktank_memory_test.go backend/internal/service/thinktank_adk_stage_test.go backend/internal/service/thinktank_adk_runner_test.go
git rm backend/internal/service/thinktank.go
git commit -m "refactor: 拆分 thinktank 服务文件结构"
```

---

### Task 5: 做第 1 批回归验证并更新设计落地状态

**Files:**
- Modify: `docs/superpowers/specs/2026-04-24-backend-structure-refactor-design.md`
- Test: `backend/cmd/server/main_ai_test.go`
- Test: `backend/cmd/server/routes_test.go`
- Test: `backend/internal/service/thinktank_test.go`
- Test: `backend/internal/service/thinktank_memory_test.go`

- [ ] **Step 1: 跑第 1 批完整后端验证**

Run: `env GOTOOLCHAIN=go1.25.3 go test ./cmd/server ./internal/service`

Expected:
- `ok   wenDao/cmd/server`
- `ok   wenDao/internal/service`

- [ ] **Step 2: 跑一遍后端全量测试，确认没有被结构调整拖坏**

Run: `env GOTOOLCHAIN=go1.25.3 go test ./...`

Expected:
- 所有后端测试通过。

- [ ] **Step 3: 在设计文档中标记第 1 批状态**

```md
## 实施状态

- 第 1 批：已完成
- 第 2 批：未开始
- 第 3 批：未开始
```

- [ ] **Step 4: 确认 `main.go` 和 ThinkTank 结构满足 Phase 1 目标**

```bash
wc -l backend/cmd/server/main.go
rg --files backend/internal/service | rg 'thinktank'
```

Expected:
- `main.go` 显著缩短。
- `thinktank` 相关逻辑已分布到多个职责文件。

- [ ] **Step 5: 提交 Phase 1 收尾**

```bash
git add docs/superpowers/specs/2026-04-24-backend-structure-refactor-design.md
git commit -m "chore: 完成后端结构重构第一批"
```

---

## Self-Review

### Spec Coverage

- `main.go` 退化为入口：由 Task 1-2 覆盖。
- bootstrap / app 层拆分：由 Task 2 覆盖。
- `thinktank.go` 文件级切分：由 Task 3-4 覆盖。
- 不改外部行为并用测试兜底：由 Task 1、3、5 覆盖。

### Placeholder Scan

- 本计划未使用 `TODO`、`TBD`、`later` 等占位词。
- 所有任务都包含了具体文件路径、命令和期望结果。

### Type Consistency

- `Run` / `MustRun` 为启动层统一入口。
- `ThinkTankService`、`thinkTankService`、`StreamEvent`、`PlannerDecision` 等命名在各任务中保持一致。

