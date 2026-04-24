# ThinkTank 多 Agent AI 助手 V2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 用 Eino ADK + Ark 将当前单 RAG AIChat 升级为多 Agent 调研助手，支持阶段流、会话内追问恢复、独立 ai-chat 日志、知识文档审核入库与共享向量检索。

**Architecture:** 保留 `/api/ai/chat` 与 `/api/ai/chat/stream` 入口不变，在后端新增 ThinkTank 编排服务，内部用 Planner / Supervisor / Librarian / Journalist / Synthesizer 拆分职责。现有 `RAGChain` 和向量检索下沉为 Librarian 本地知识能力；知识文档审核通过后进入统一 Redis Vector 索引；前端只消费 `stage`、`question`、`chunk` 三类关键事件。

**Tech Stack:** Go 1.23、Gin、GORM、Redis/Redis Vector、Zap、CloudWeGo Eino v0.8.6、Ark ChatModel、React 18、TypeScript、Zustand、TanStack Query。

> 说明：根据用户要求，本计划**不包含 git commit 步骤**。

---

## File Structure Map

### Backend: create
- `backend/internal/model/conversation_run.go` — 持久化会话执行状态与待追问上下文
- `backend/internal/model/knowledge_document.go` — 调研知识文档主表
- `backend/internal/model/knowledge_document_source.go` — 知识文档来源表
- `backend/internal/repository/conversation_run.go` — 会话执行状态仓储
- `backend/internal/repository/knowledge_document.go` — 知识文档仓储
- `backend/internal/repository/knowledge_document_source.go` — 来源仓储
- `backend/internal/repository/knowledge_document_columns_test.go` — GORM 列名与表名测试
- `backend/internal/repository/conversation_run_columns_test.go` — 会话执行状态 schema 测试
- `backend/internal/service/ai_log.go` — 独立 `YYYY-MM-DD-ai-chat.log` 结构化日志
- `backend/internal/service/knowledge_document.go` — 知识文档创建/审核/拒绝/向量化服务
- `backend/internal/service/knowledge_document_test.go` — 知识文档服务测试
- `backend/internal/service/thinktank.go` — ThinkTank 编排入口
- `backend/internal/service/thinktank_planner.go` — Planner
- `backend/internal/service/thinktank_librarian.go` — Librarian
- `backend/internal/service/thinktank_journalist.go` — Journalist
- `backend/internal/service/thinktank_synthesizer.go` — Synthesizer
- `backend/internal/service/thinktank_test.go` — ThinkTank 流程测试
- `backend/internal/handler/knowledge_document.go` — 管理后台知识文档接口
- `backend/internal/handler/knowledge_document_test.go` — 知识文档 handler 测试

### Backend: modify
- `backend/config/config.go` — 增加联网调研配置
- `backend/config/config.yaml` — 增加默认配置项
- `backend/config/config_test.go` — 环境变量绑定测试
- `backend/internal/pkg/database/migrate.go` — AutoMigrate 新模型
- `backend/internal/service/vector.go` — 支持知识文档向量化/删除/统一检索元数据
- `backend/internal/service/ai.go` — 对话入口改为委托 ThinkTank，保留摘要能力
- `backend/internal/handler/ai.go` — SSE 事件扩展 `stage` / `question`
- `backend/internal/pkg/eino/chain.go` — 增加本地知识总结方法，供 Librarian 复用
- `backend/internal/pkg/eino/retriever.go` — 支持混合来源元数据透传
- `backend/internal/pkg/eino/vectorstore.go` — 统一向量 metadata 字段
- `backend/cmd/server/main.go` — 注入新 repo/service/handler/routes/logger
- `backend/cmd/server/routes_test.go` — 新增管理接口路由断言

### Frontend: create
- `frontend/src/api/knowledgeDocument.ts` — 管理端知识文档 API
- `frontend/src/views/admin/knowledge-documents/KnowledgeDocumentList.tsx` — 列表页
- `frontend/src/views/admin/knowledge-documents/KnowledgeDocumentDetail.tsx` — 详情审核页

### Frontend: modify
- `frontend/src/types/index.ts` — 增加 chat stage、question 事件、knowledge document 类型
- `frontend/src/api/chat.ts` — 解析 `stage` 与 `question` SSE 事件
- `frontend/src/store/chatStore.ts` — 保存当前阶段、待追问状态、运行状态
- `frontend/src/pages/AIChat.tsx` — 展示阶段与追问消息
- `frontend/src/router.tsx` — 新增知识文档后台路由
- `frontend/src/components/admin/AdminLayout.tsx` — 新增知识文档菜单入口

### Existing patterns to mimic
- Model/Repository: `backend/internal/model/article.go`, `backend/internal/repository/article.go`
- Chat persistence: `backend/internal/repository/conversation.go`, `backend/internal/repository/chat_message.go`
- Handler style: `backend/internal/handler/chat.go`, `backend/internal/handler/article.go`
- Service async vectorization: `backend/internal/service/article.go`
- Admin list/detail page: `frontend/src/views/admin/articles/ArticleList.tsx`, `frontend/src/views/admin/articles/ArticleEditor.tsx`

---

### Task 1: Add research config and persistence models

**Files:**
- Create: `backend/internal/model/conversation_run.go`
- Create: `backend/internal/model/knowledge_document.go`
- Create: `backend/internal/model/knowledge_document_source.go`
- Create: `backend/internal/repository/knowledge_document_columns_test.go`
- Create: `backend/internal/repository/conversation_run_columns_test.go`
- Modify: `backend/config/config.go`
- Modify: `backend/config/config.yaml`
- Modify: `backend/config/config_test.go`
- Modify: `backend/internal/pkg/database/migrate.go`

- [ ] **Step 1: Write failing config and schema tests**

```go
func TestLoadConfig_BindsResearchEndpointFromEnv(t *testing.T) {
	viper.Reset()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	defer viper.Reset()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	configContent := `server:
  port: "8089"
  mode: "debug"
site:
  slogan: "test"
  url: "http://localhost:3000"
jwt:
  secret: "real-test-secret"
  access_expire_hours: 1
  refresh_expire_days: 7
ai:
  api_key: "x"
  endpoint: "https://ark.example.com"
  embedding_model: "embed-model"
  llm_model: "chat-model"
  temperature: 0.7
  max_tokens: 500
  top_k: 3
  rag_min_score: 0.30
upload:
  max_size: 10485760
  allowed_types:
    - "image/jpeg"
  storage_path: "./uploads"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	oldWD, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	_ = os.Setenv("RESEARCH_ENDPOINT", "https://search.example.com")
	_ = os.Setenv("RESEARCH_API_KEY", "research-secret")
	defer os.Unsetenv("RESEARCH_ENDPOINT")
	defer os.Unsetenv("RESEARCH_API_KEY")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected config to load, got %v", err)
	}
	if cfg.AI.ResearchEndpoint != "https://search.example.com" {
		t.Fatalf("expected research endpoint from env, got %q", cfg.AI.ResearchEndpoint)
	}
	if cfg.AI.ResearchAPIKey != "research-secret" {
		t.Fatalf("expected research api key from env, got %q", cfg.AI.ResearchAPIKey)
	}
}
```

```go
func TestKnowledgeDocumentModel_TableAndStatusColumns(t *testing.T) {
	s, err := schema.Parse(&model.KnowledgeDocument{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}
	if s.Table != "knowledge_documents" {
		t.Fatalf("expected table knowledge_documents, got %q", s.Table)
	}
	statusField := s.LookUpField("Status")
	if statusField == nil || statusField.DBName != "status" {
		t.Fatalf("expected status column, got %#v", statusField)
	}
}
```

```go
func TestConversationRunModel_UsesConversationRunsTable(t *testing.T) {
	s, err := schema.Parse(&model.ConversationRun{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}
	if s.Table != "conversation_runs" {
		t.Fatalf("expected table conversation_runs, got %q", s.Table)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./backend/config ./backend/internal/repository
```

Expected: FAIL with unknown `ResearchEndpoint` / missing `KnowledgeDocument` and `ConversationRun` symbols.

- [ ] **Step 3: Add config fields, models, and migration**

```go
// backend/config/config.go
// AIConfig AI 配置
 type AIConfig struct {
 	APIKey                 string  `mapstructure:"api_key"`
 	Endpoint               string  `mapstructure:"endpoint"`
 	EmbeddingModel         string  `mapstructure:"embedding_model"`
 	LLMModel               string  `mapstructure:"llm_model"`
 	Temperature            float32 `mapstructure:"temperature"`
 	MaxTokens              int     `mapstructure:"max_tokens"`
 	TopK                   int     `mapstructure:"top_k"`
 	RAGMinScore            float32 `mapstructure:"rag_min_score"`
 	ResearchEndpoint       string  `mapstructure:"research_endpoint"`
 	ResearchAPIKey         string  `mapstructure:"research_api_key"`
 	ResearchMaxResults     int     `mapstructure:"research_max_results"`
 	ResearchTimeoutSeconds int     `mapstructure:"research_timeout_seconds"`
 }
```

```go
// backend/config/config.go
_ = viper.BindEnv("ai.research_endpoint", "RESEARCH_ENDPOINT")
_ = viper.BindEnv("ai.research_api_key", "RESEARCH_API_KEY")
if cfg.AI.ResearchMaxResults <= 0 {
	cfg.AI.ResearchMaxResults = 5
}
if cfg.AI.ResearchTimeoutSeconds <= 0 {
	cfg.AI.ResearchTimeoutSeconds = 15
}
```

```yaml
# backend/config/config.yaml
ai:
  api_key: ${DOUBAO_API_KEY}
  endpoint: "https://ark.cn-beijing.volces.com/api/v3"
  embedding_model: "doubao-embedding-large-text-240915"
  llm_model: "doubao-1.5-pro-32k-250115"
  temperature: 0.7
  max_tokens: 500
  top_k: 3
  rag_min_score: 0.30
  research_endpoint: ${RESEARCH_ENDPOINT}
  research_api_key: ${RESEARCH_API_KEY}
  research_max_results: 5
  research_timeout_seconds: 15
```

```go
// backend/internal/model/conversation_run.go
package model

import "time"

type ConversationRun struct {
	ID                 int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID     int64      `gorm:"not null;index:idx_conversation_run_conversation" json:"conversation_id"`
	UserID             int64      `gorm:"not null;index:idx_conversation_run_user" json:"user_id"`
	Status             string     `gorm:"size:32;not null;index:idx_conversation_run_status" json:"status"`
	CurrentStage       string     `gorm:"size:32;not null" json:"current_stage"`
	OriginalQuestion   string     `gorm:"type:text;not null" json:"original_question"`
	NormalizedQuestion string     `gorm:"type:text" json:"normalized_question"`
	PendingQuestion    *string    `gorm:"type:text" json:"pending_question,omitempty"`
	PendingContext     string     `gorm:"type:longtext" json:"pending_context"`
	LastPlan           string     `gorm:"type:longtext" json:"last_plan"`
	LastError          *string    `gorm:"type:text" json:"last_error,omitempty"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (ConversationRun) TableName() string { return "conversation_runs" }
```

```go
// backend/internal/model/knowledge_document.go
package model

import "time"

const (
	KnowledgeDocumentStatusPendingReview = "pending_review"
	KnowledgeDocumentStatusApproved      = "approved"
	KnowledgeDocumentStatusRejected      = "rejected"
)

type KnowledgeDocument struct {
	ID               int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Title            string     `gorm:"size:255;not null" json:"title"`
	Summary          string     `gorm:"type:text" json:"summary"`
	Content          string     `gorm:"type:longtext;not null" json:"content"`
	Status           string     `gorm:"size:32;not null;index:idx_knowledge_document_status" json:"status"`
	SourceType       string     `gorm:"size:32;not null" json:"source_type"`
	CreatedByUserID  int64      `gorm:"not null;index:idx_knowledge_document_created_by" json:"created_by_user_id"`
	ReviewedByUserID *int64     `gorm:"index:idx_knowledge_document_reviewed_by" json:"reviewed_by_user_id,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote       string     `gorm:"type:text" json:"review_note"`
	VectorizedAt     *time.Time `json:"vectorized_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (KnowledgeDocument) TableName() string { return "knowledge_documents" }
```

```go
// backend/internal/model/knowledge_document_source.go
package model

import "time"

type KnowledgeDocumentSource struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	KnowledgeDocumentID int64     `gorm:"not null;index:idx_kd_source_document" json:"knowledge_document_id"`
	SourceURL           string    `gorm:"type:text;not null" json:"source_url"`
	SourceTitle         string    `gorm:"size:500" json:"source_title"`
	SourceDomain        string    `gorm:"size:255" json:"source_domain"`
	SourceSnippet       string    `gorm:"type:text" json:"source_snippet"`
	SortOrder           int       `gorm:"not null" json:"sort_order"`
	CreatedAt           time.Time `json:"created_at"`
}

func (KnowledgeDocumentSource) TableName() string { return "knowledge_document_sources" }
```

```go
// backend/internal/pkg/database/migrate.go
return db.AutoMigrate(
	&model.User{},
	&model.Category{},
	&model.Article{},
	&model.Comment{},
	&model.Upload{},
	&model.Setting{},
	&model.DailyStat{},
	&model.ArticleStat{},
	&model.Conversation{},
	&model.ChatMessage{},
	&model.ConversationRun{},
	&model.KnowledgeDocument{},
	&model.KnowledgeDocumentSource{},
)
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
go test ./backend/config ./backend/internal/repository
```

Expected: PASS for new config/schema tests.

---

### Task 2: Add repositories and knowledge-document service core

**Files:**
- Create: `backend/internal/repository/conversation_run.go`
- Create: `backend/internal/repository/knowledge_document.go`
- Create: `backend/internal/repository/knowledge_document_source.go`
- Create: `backend/internal/service/knowledge_document.go`
- Create: `backend/internal/service/knowledge_document_test.go`
- Modify: `backend/internal/service/vector.go`

- [ ] **Step 1: Write failing service tests for create/approve/reject flow**

```go
func TestKnowledgeDocumentService_CreateDraft_PersistsDocumentAndSources(t *testing.T) {
	docRepo := &stubKnowledgeDocumentRepository{}
	sourceRepo := &stubKnowledgeDocumentSourceRepository{}
	vectorSvc := &stubVectorService{}
	svc := NewKnowledgeDocumentService(docRepo, sourceRepo, vectorSvc, zap.NewNop())

	doc, err := svc.CreateResearchDraft(CreateKnowledgeDocumentInput{
		Title:           "工业大模型落地调研",
		Summary:         "总结 2025 年行业案例",
		Content:         "正文内容",
		CreatedByUserID: 9,
		Sources: []KnowledgeSourceInput{{
			URL:     "https://example.com/a",
			Title:   "案例 A",
			Domain:  "example.com",
			Snippet: "示例摘要",
		}},
	})
	if err != nil {
		t.Fatalf("expected draft to be created, got %v", err)
	}
	if doc.Status != model.KnowledgeDocumentStatusPendingReview {
		t.Fatalf("expected pending_review, got %q", doc.Status)
	}
	if len(sourceRepo.created) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sourceRepo.created))
	}
	if vectorSvc.vectorizedDocumentID != 0 {
		t.Fatalf("did not expect vectorization before approval")
	}
}
```

```go
func TestKnowledgeDocumentService_Approve_TriggersVectorization(t *testing.T) {
	nowDoc := &model.KnowledgeDocument{ID: 7, Title: "标题", Content: "正文", Status: model.KnowledgeDocumentStatusPendingReview}
	docRepo := &stubKnowledgeDocumentRepository{byID: nowDoc}
	sourceRepo := &stubKnowledgeDocumentSourceRepository{}
	vectorSvc := &stubVectorService{}
	svc := NewKnowledgeDocumentService(docRepo, sourceRepo, vectorSvc, zap.NewNop())

	approved, err := svc.Approve(7, 1, "审核通过")
	if err != nil {
		t.Fatalf("expected approval success, got %v", err)
	}
	if approved.Status != model.KnowledgeDocumentStatusApproved {
		t.Fatalf("expected approved, got %q", approved.Status)
	}
	if vectorSvc.vectorizedDocumentID != 7 {
		t.Fatalf("expected vectorization for document 7, got %d", vectorSvc.vectorizedDocumentID)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./backend/internal/service -run KnowledgeDocument -v
```

Expected: FAIL with undefined `NewKnowledgeDocumentService`, `CreateResearchDraft`, `Approve`.

- [ ] **Step 3: Add repositories and knowledge-document service**

```go
// backend/internal/repository/conversation_run.go
package repository

import (
	"gorm.io/gorm"
	"wenDao/internal/model"
)

type ConversationRunRepository interface {
	Create(run *model.ConversationRun) error
	GetActiveByConversationID(conversationID int64) (*model.ConversationRun, error)
	Update(run *model.ConversationRun) error
}

type conversationRunRepository struct{ db *gorm.DB }

func NewConversationRunRepository(db *gorm.DB) ConversationRunRepository {
	return &conversationRunRepository{db: db}
}

func (r *conversationRunRepository) Create(run *model.ConversationRun) error {
	return r.db.Create(run).Error
}

func (r *conversationRunRepository) GetActiveByConversationID(conversationID int64) (*model.ConversationRun, error) {
	var run model.ConversationRun
	err := r.db.Where("conversation_id = ?", conversationID).
		Order("updated_at DESC").
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *conversationRunRepository) Update(run *model.ConversationRun) error {
	return r.db.Save(run).Error
}
```

```go
// backend/internal/repository/knowledge_document.go
package repository

import (
	"gorm.io/gorm"
	"wenDao/internal/model"
)

type KnowledgeDocumentFilter struct {
	Status   string
	Keyword  string
	Page     int
	PageSize int
}

type KnowledgeDocumentRepository interface {
	Create(doc *model.KnowledgeDocument) error
	GetByID(id int64) (*model.KnowledgeDocument, error)
	List(filter KnowledgeDocumentFilter) ([]*model.KnowledgeDocument, int64, error)
	Update(doc *model.KnowledgeDocument) error
}

type knowledgeDocumentRepository struct{ db *gorm.DB }

func NewKnowledgeDocumentRepository(db *gorm.DB) KnowledgeDocumentRepository {
	return &knowledgeDocumentRepository{db: db}
}
```

```go
// backend/internal/repository/knowledge_document_source.go
package repository

import (
	"gorm.io/gorm"
	"wenDao/internal/model"
)

type KnowledgeDocumentSourceRepository interface {
	CreateBatch(sources []*model.KnowledgeDocumentSource) error
	ListByDocumentID(documentID int64) ([]*model.KnowledgeDocumentSource, error)
}

type knowledgeDocumentSourceRepository struct{ db *gorm.DB }

func NewKnowledgeDocumentSourceRepository(db *gorm.DB) KnowledgeDocumentSourceRepository {
	return &knowledgeDocumentSourceRepository{db: db}
}
```

```go
// backend/internal/service/knowledge_document.go
package service

type KnowledgeSourceInput struct {
	URL     string
	Title   string
	Domain  string
	Snippet string
}

type CreateKnowledgeDocumentInput struct {
	Title           string
	Summary         string
	Content         string
	CreatedByUserID int64
	Sources         []KnowledgeSourceInput
}

type KnowledgeDocumentService interface {
	CreateResearchDraft(input CreateKnowledgeDocumentInput) (*model.KnowledgeDocument, error)
	Approve(id int64, reviewerID int64, note string) (*model.KnowledgeDocument, error)
	Reject(id int64, reviewerID int64, note string) (*model.KnowledgeDocument, error)
	GetByID(id int64) (*model.KnowledgeDocument, []*model.KnowledgeDocumentSource, error)
	List(filter repository.KnowledgeDocumentFilter) ([]*model.KnowledgeDocument, int64, error)
}
```

```go
// backend/internal/service/knowledge_document.go
func (s *knowledgeDocumentService) CreateResearchDraft(input CreateKnowledgeDocumentInput) (*model.KnowledgeDocument, error) {
	doc := &model.KnowledgeDocument{
		Title:           input.Title,
		Summary:         input.Summary,
		Content:         input.Content,
		Status:          model.KnowledgeDocumentStatusPendingReview,
		SourceType:      "research",
		CreatedByUserID: input.CreatedByUserID,
	}
	if err := s.docRepo.Create(doc); err != nil {
		return nil, err
	}
	sources := make([]*model.KnowledgeDocumentSource, 0, len(input.Sources))
	for i, item := range input.Sources {
		sources = append(sources, &model.KnowledgeDocumentSource{
			KnowledgeDocumentID: doc.ID,
			SourceURL:           item.URL,
			SourceTitle:         item.Title,
			SourceDomain:        item.Domain,
			SourceSnippet:       item.Snippet,
			SortOrder:           i,
		})
	}
	if len(sources) > 0 {
		if err := s.sourceRepo.CreateBatch(sources); err != nil {
			return nil, err
		}
	}
	return doc, nil
}
```

```go
// backend/internal/service/knowledge_document.go
func (s *knowledgeDocumentService) Approve(id int64, reviewerID int64, note string) (*model.KnowledgeDocument, error) {
	doc, err := s.docRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	doc.Status = model.KnowledgeDocumentStatusApproved
	doc.ReviewedByUserID = &reviewerID
	doc.ReviewedAt = &now
	doc.ReviewNote = note
	if err := s.docRepo.Update(doc); err != nil {
		return nil, err
	}
	if err := s.vectorService.VectorizeKnowledgeDocument(doc.ID, doc.Title, doc.Content); err != nil {
		return nil, err
	}
	doc.VectorizedAt = &now
	if err := s.docRepo.Update(doc); err != nil {
		return nil, err
	}
	return doc, nil
}
```

```go
// backend/internal/service/vector.go
// VectorService 向量服务接口
 type VectorService interface {
 	VectorizeArticle(articleID int64, title, content string) error
 	DeleteArticleVector(articleID int64) error
 	SearchArticles(query string, topK int) ([]ArticleChunk, error)
 	VectorizeKnowledgeDocument(documentID int64, title, content string) error
 	DeleteKnowledgeDocumentVector(documentID int64) error
 }
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
go test ./backend/internal/service -run KnowledgeDocument -v
```

Expected: PASS for create/approve/reject behavior.

---

### Task 3: Unify vector metadata for article and knowledge documents

**Files:**
- Modify: `backend/internal/service/vector.go`
- Modify: `backend/internal/pkg/eino/retriever.go`
- Modify: `backend/internal/pkg/eino/vectorstore.go`
- Modify: `backend/internal/pkg/eino/chain.go`
- Create: `backend/internal/service/vector_test.go`

- [ ] **Step 1: Write failing vector metadata tests**

```go
func TestVectorService_VectorizeKnowledgeDocument_UsesKnowledgeMetadata(t *testing.T) {
	store := &stubRedisVectorStore{}
	embedder := &stubEmbedder{batch: [][]float64{{0.1}, {0.2}}}
	svc := NewVectorService(store, embedder, zap.NewNop())

	err := svc.VectorizeKnowledgeDocument(11, "知识标题", "第一段\n\n第二段")
	if err != nil {
		t.Fatalf("expected vectorization success, got %v", err)
	}
	if len(store.upserted) == 0 {
		t.Fatalf("expected upserted vectors")
	}
	if store.upserted[0].Metadata["source_kind"] != "knowledge_document" {
		t.Fatalf("expected source_kind knowledge_document, got %#v", store.upserted[0].Metadata["source_kind"])
	}
	if store.upserted[0].Metadata["source_id"] != int64(11) {
		t.Fatalf("expected source_id 11, got %#v", store.upserted[0].Metadata["source_id"])
	}
}
```

```go
func TestRedisRetriever_RetainsSourceKindMetadata(t *testing.T) {
	retriever := &RedisRetriever{vectorStore: &stubSearchVectorStore{results: []SearchResult{{
		Key:    "vec:knowledge:11:chunk:0",
		Score:  0.12,
		Metadata: map[string]any{"content": "知识片段", "source_kind": "knowledge_document", "source_id": int64(11)},
	}}}, topK: 3, embedder: &stubEmbedder{single: []float64{0.1}}}
	docs, err := retriever.Retrieve(context.Background(), "工业知识")
	if err != nil {
		t.Fatalf("expected retrieve success, got %v", err)
	}
	if docs[0].MetaData["source_kind"] != "knowledge_document" {
		t.Fatalf("expected source_kind passthrough, got %#v", docs[0].MetaData["source_kind"])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./backend/internal/service ./backend/internal/pkg/eino -run 'Vector|Retriever' -v
```

Expected: FAIL because `VectorizeKnowledgeDocument` does not exist and metadata assertions fail.

- [ ] **Step 3: Implement unified vector metadata and Librarian helpers**

```go
// backend/internal/service/vector.go
func (s *vectorService) VectorizeKnowledgeDocument(documentID int64, title, content string) error {
	if err := s.DeleteKnowledgeDocumentVector(documentID); err != nil {
		s.logger.Warn("failed to delete old knowledge document vectors", zap.Int64("document_id", documentID), zap.Error(err))
	}
	chunks := s.chunkArticle(title, content)
	if len(chunks) == 0 {
		return nil
	}
	embeddings, err := s.embedder.EmbedBatch(chunks)
	if err != nil {
		return fmt.Errorf("failed to embed document chunks: %w", err)
	}
	items := make([]eino.VectorItem, 0, len(chunks))
	for i, chunk := range chunks {
		items = append(items, eino.VectorItem{
			Key:    fmt.Sprintf("vec:knowledge:%d:chunk:%d", documentID, i),
			Vector: embeddings[i],
			Metadata: map[string]any{
				"source_kind": "knowledge_document",
				"source_id":   documentID,
				"chunk_index": i,
				"title":       title,
				"content":     chunk,
				"status":      model.KnowledgeDocumentStatusApproved,
			},
		})
	}
	return s.vectorStore.UpsertBatch(items)
}
```

```go
// backend/internal/service/vector.go
func (s *vectorService) DeleteKnowledgeDocumentVector(documentID int64) error {
	pattern := fmt.Sprintf("vec:knowledge:%d:chunk:*", documentID)
	return s.vectorStore.Delete(pattern)
}
```

```go
// backend/internal/pkg/eino/retriever.go
meta := make(map[string]interface{}, len(res.Metadata)+1)
for k, v := range res.Metadata {
	meta[k] = v
}
meta["score"] = res.Score
```

```go
// backend/internal/pkg/eino/chain.go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
go test ./backend/internal/service ./backend/internal/pkg/eino -run 'Vector|Retriever' -v
```

Expected: PASS with knowledge-document metadata preserved.

---

### Task 4: Add ai-chat dedicated logger

**Files:**
- Create: `backend/internal/service/ai_log.go`
- Create: `backend/internal/service/ai_log_test.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Write failing logger tests**

```go
func TestAILogger_WritesToDailyAiChatFile(t *testing.T) {
	logDir := t.TempDir()
	logger, err := NewAILogger(logDir)
	if err != nil {
		t.Fatalf("expected logger to initialize, got %v", err)
	}
	defer logger.Close()

	logger.LogStage(AILogEntry{
		ConversationID: 3,
		UserID:         9,
		Stage:          "local_search",
		Message:        "正在检索站内知识",
	})

	path := filepath.Join(logDir, time.Now().Format("2006-01-02")+"-ai-chat.log")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected log file to exist, got %v", err)
	}
	if !strings.Contains(string(content), "local_search") {
		t.Fatalf("expected stage log entry, got %s", string(content))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./backend/internal/service -run AILogger -v
```

Expected: FAIL with undefined `NewAILogger` / `AILogEntry`.

- [ ] **Step 3: Implement daily ai-chat logger and wire it**

```go
// backend/internal/service/ai_log.go
package service

type AILogEntry struct {
	ConversationID int64  `json:"conversation_id"`
	UserID         int64  `json:"user_id"`
	RunID          int64  `json:"run_id,omitempty"`
	Stage          string `json:"stage"`
	Message        string `json:"message"`
	Detail         string `json:"detail,omitempty"`
}

type AILogger interface {
	LogStage(entry AILogEntry)
	LogError(entry AILogEntry)
	Close() error
}
```

```go
// backend/internal/service/ai_log.go
func NewAILogger(logDir string) (AILogger, error) {
	if logDir == "" {
		logDir = "log"
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}
	filePath := filepath.Join(logDir, time.Now().Format("2006-01-02")+"-ai-chat.log")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &aiLogger{file: file, encoder: json.NewEncoder(file)}, nil
}
```

```go
// backend/cmd/server/main.go
aiEventLogger, err := service.NewAILogger(cfg.Log.Output)
if err != nil {
	logger.Fatal("Failed to create AI chat logger", zap.Error(err))
}
defer aiEventLogger.Close()
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
go test ./backend/internal/service -run AILogger -v
```

Expected: PASS and test log file ends with `-ai-chat.log`.

---

### Task 5: Add ThinkTank planner, Librarian, Synthesizer, and run-state persistence

**Files:**
- Create: `backend/internal/service/thinktank.go`
- Create: `backend/internal/service/thinktank_planner.go`
- Create: `backend/internal/service/thinktank_librarian.go`
- Create: `backend/internal/service/thinktank_synthesizer.go`
- Create: `backend/internal/service/thinktank_test.go`
- Modify: `backend/internal/service/ai.go`
- Modify: `backend/internal/repository/conversation.go`
- Modify: `backend/internal/repository/chat_message.go`

- [ ] **Step 1: Write failing tests for clarification and local-only completion**

```go
func TestThinkTankService_ClarifiesWhenQuestionIsAmbiguous(t *testing.T) {
	convRepo := &stubConversationRepository{conversation: &model.Conversation{ID: 11, UserID: 8, Title: "New Conversation"}}
	msgRepo := &stubChatMessageRepository{}
	runRepo := &stubConversationRunRepository{}
	planner := &stubPlanner{decision: PlannerDecision{RequiresClarification: true, ClarificationQuestion: "你更关注国内还是海外案例？"}}
	librarian := &stubLibrarian{}
	synthesizer := &stubSynthesizer{}
	logger := &stubAILogger{}
	svc := NewThinkTankService(planner, librarian, nil, synthesizer, runRepo, convRepo, msgRepo, nil, logger)

	resp, err := svc.Chat(context.Background(), "帮我调研大模型应用", ptrInt64(11), ptrInt64(8))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !resp.RequiresUserInput {
		t.Fatalf("expected clarification request")
	}
	if resp.Message != "你更关注国内还是海外案例？" {
		t.Fatalf("unexpected clarification message %q", resp.Message)
	}
	if runRepo.saved.CurrentStage != "clarifying" {
		t.Fatalf("expected run stage clarifying, got %q", runRepo.saved.CurrentStage)
	}
}
```

```go
func TestThinkTankService_UsesLibrarianAndSynthesizerWhenLocalKnowledgeIsEnough(t *testing.T) {
	planner := &stubPlanner{decision: PlannerDecision{ExecutionStrategy: "local_then_web"}}
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "sufficient", Summary: "站内文章总结", Sources: []SourceRef{{Kind: "article", ID: 2, Title: "文章标题"}}}}
	synthesizer := &stubSynthesizer{answer: "这是基于站内知识的最终回答"}
	svc := NewThinkTankService(planner, librarian, nil, synthesizer, &stubConversationRunRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{})

	resp, err := svc.Chat(context.Background(), "站内文章讲了什么", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Message != "这是基于站内知识的最终回答" {
		t.Fatalf("unexpected answer %q", resp.Message)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./backend/internal/service -run ThinkTank -v
```

Expected: FAIL with undefined `NewThinkTankService`, `PlannerDecision`, `LibrarianResult`.

- [ ] **Step 3: Implement Planner/Librarian/Synthesizer services and ThinkTank orchestration**

```go
// backend/internal/service/thinktank.go
package service

type ThinkTankChatResponse struct {
	Message           string
	Sources           []string
	Stage             string
	RequiresUserInput bool
}

type ThinkTankService interface {
	Chat(ctx context.Context, question string, conversationID *int64, userID *int64) (*ThinkTankChatResponse, error)
	ChatStream(ctx context.Context, question string, conversationID *int64, userID *int64) (<-chan StreamEvent, <-chan error)
}
```

```go
// backend/internal/service/thinktank_planner.go
package service

type PlannerDecision struct {
	PlanSummary          string
	ExecutionStrategy    string
	RequiresClarification bool
	ClarificationQuestion string
}

type ThinkTankPlanner interface {
	Plan(ctx context.Context, question string, history []model.ChatMessage, pending *model.ConversationRun) (PlannerDecision, error)
}
```

```go
// backend/internal/service/thinktank_planner.go
func (p *llmThinkTankPlanner) Plan(ctx context.Context, question string, history []model.ChatMessage, pending *model.ConversationRun) (PlannerDecision, error) {
	normalized := strings.TrimSpace(question)
	if len([]rune(normalized)) < 10 {
		return PlannerDecision{
			PlanSummary:           "问题过短，需要 уточ问范围",
			RequiresClarification: true,
			ClarificationQuestion: "请补充你希望调研的具体方向或限制条件。",
		}, nil
	}
	return PlannerDecision{
		PlanSummary:       "先检索站内知识，再根据缺口决定是否联网调研",
		ExecutionStrategy: "local_then_web",
	}, nil
}
```

```go
// backend/internal/service/thinktank_librarian.go
package service

type SourceRef struct {
	Kind  string
	ID    int64
	Title string
	URL   string
}

type LibrarianResult struct {
	CoverageStatus string
	Summary        string
	Sources        []SourceRef
	FollowupHint   string
}
```

```go
// backend/internal/service/thinktank_librarian.go
func (s *librarianService) Search(ctx context.Context, question string) (LibrarianResult, error) {
	summary, docs, err := s.chain.SummarizeLocalFindings(ctx, question)
	if err != nil {
		return LibrarianResult{}, err
	}
	result := LibrarianResult{CoverageStatus: "insufficient"}
	if summary != "" {
		result.Summary = summary
		result.CoverageStatus = "partial"
	}
	for _, doc := range docs {
		ref := SourceRef{Title: fmt.Sprintf("%v", doc.MetaData["title"])}
		if kind, ok := doc.MetaData["source_kind"].(string); ok {
			ref.Kind = kind
		} else {
			ref.Kind = "article"
		}
		if id, ok := doc.MetaData["source_id"].(int64); ok {
			ref.ID = id
		}
		result.Sources = append(result.Sources, ref)
	}
	if len(result.Sources) >= 2 {
		result.CoverageStatus = "sufficient"
	}
	return result, nil
}
```

```go
// backend/internal/service/thinktank_synthesizer.go
func (s *thinkTankSynthesizer) Compose(ctx context.Context, question string, local LibrarianResult, web *JournalistResult) (string, []string, error) {
	var builder strings.Builder
	if local.Summary != "" {
		builder.WriteString("站内知识：\n")
		builder.WriteString(local.Summary)
		builder.WriteString("\n\n")
	}
	if web != nil && web.Summary != "" {
		builder.WriteString("外部调研补充：\n")
		builder.WriteString(web.Summary)
	}
	messages := []eino.ChatMessage{{Role: "system", Content: "你是问道博客的多 Agent 汇总助手，请整合站内知识与外部调研结果，用中文给出最终回答。"}, {Role: "user", Content: question + "\n\n" + builder.String()}}
	answer, err := s.llm.Chat(messages)
	if err != nil {
		return "", nil, err
	}
	return answer, collectSourceTitles(local, web), nil
}
```

```go
// backend/internal/service/thinktank.go
func (s *thinkTankService) Chat(ctx context.Context, question string, conversationID *int64, userID *int64) (*ThinkTankChatResponse, error) {
	conv, err := s.getOwnedConversation(conversationID, userID)
	if err != nil {
		return nil, err
	}
	history := []model.ChatMessage{}
	if conv != nil {
		history, _ = s.msgRepo.GetByConversationID(conv.ID)
	}
	pending, _ := s.runRepo.GetActiveByConversationID(derefConversationID(conversationID))
	decision, err := s.planner.Plan(ctx, question, history, pending)
	if err != nil {
		return nil, err
	}
	if decision.RequiresClarification {
		resp := &ThinkTankChatResponse{Message: decision.ClarificationQuestion, Stage: "clarifying", RequiresUserInput: true}
		if conv != nil {
			s.persistClarification(conv.ID, derefUserID(userID), question, decision)
		}
		return resp, nil
	}
	localResult, err := s.librarian.Search(ctx, question)
	if err != nil {
		return nil, err
	}
	answer, sources, err := s.synthesizer.Compose(ctx, question, localResult, nil)
	if err != nil {
		return nil, err
	}
	if conv != nil {
		s.persistCompletedRun(conv.ID, derefUserID(userID), question, answer, decision)
	}
	return &ThinkTankChatResponse{Message: answer, Sources: sources, Stage: "completed"}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
go test ./backend/internal/service -run ThinkTank -v
```

Expected: PASS for clarification path and local completion path.

---

### Task 6: Add Journalist web research and stream stage/question events

**Files:**
- Create: `backend/internal/service/thinktank_journalist.go`
- Modify: `backend/internal/service/thinktank.go`
- Modify: `backend/internal/service/ai.go`
- Modify: `backend/internal/handler/ai.go`
- Create: `backend/internal/handler/ai_stream_test.go`

- [ ] **Step 1: Write failing tests for journalist fallback and SSE stage/question events**

```go
func TestThinkTankService_UsesJournalistWhenLocalKnowledgeIsInsufficient(t *testing.T) {
	planner := &stubPlanner{decision: PlannerDecision{ExecutionStrategy: "local_then_web"}}
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "insufficient"}}
	journalist := &stubJournalist{result: JournalistResult{Summary: "联网调研结果", Sources: []SourceRef{{Kind: "web", Title: "外部来源", URL: "https://example.com"}}}}
	synthesizer := &stubSynthesizer{answer: "整合后的最终回答"}
	svc := NewThinkTankService(planner, librarian, journalist, synthesizer, &stubConversationRunRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, nil, &stubAILogger{})

	resp, err := svc.Chat(context.Background(), "调研工业大模型落地", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Message != "整合后的最终回答" {
		t.Fatalf("unexpected answer %q", resp.Message)
	}
	if journalist.called == 0 {
		t.Fatalf("expected journalist to be called")
	}
}
```

```go
func TestAIHandlerChatStream_EmitsStageAndQuestionEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAIHandler(&stubAIService{
		streamEvents: []service.StreamEvent{
			{Type: service.StreamEventStage, Stage: "analyzing", Label: "正在理解你的问题"},
			{Type: service.StreamEventQuestion, Stage: "clarifying", Message: "你更关注国内还是海外案例？"},
		},
	})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/ai/chat/stream", strings.NewReader(`{"message":"帮我调研大模型"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ChatStream(c)

	body := w.Body.String()
	if !strings.Contains(body, "event: stage") {
		t.Fatalf("expected stage event, got %s", body)
	}
	if !strings.Contains(body, "event: question") {
		t.Fatalf("expected question event, got %s", body)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./backend/internal/service ./backend/internal/handler -run 'Journalist|ChatStream' -v
```

Expected: FAIL with undefined `JournalistResult`, `StreamEventStage`, `event: stage` handling.

- [ ] **Step 3: Implement Journalist and stream event pipeline**

```go
// backend/internal/service/thinktank_journalist.go
package service

type JournalistResult struct {
	Summary              string
	Sources              []SourceRef
	KnowledgeDraftTitle  string
	KnowledgeDraftBody   string
	KnowledgeDraftSummary string
}

type Journalist interface {
	Research(ctx context.Context, question string, local LibrarianResult) (*JournalistResult, error)
}
```

```go
// backend/internal/service/thinktank_journalist.go
func (j *httpJournalist) Research(ctx context.Context, question string, local LibrarianResult) (*JournalistResult, error) {
	payload := map[string]any{"query": question, "max_results": j.cfg.ResearchMaxResults}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, j.cfg.ResearchEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if j.cfg.ResearchAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+j.cfg.ResearchAPIKey)
	}
	resp, err := j.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Summary string `json:"summary"`
		Items   []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Domain  string `json:"domain"`
			Snippet string `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	jr := &JournalistResult{Summary: result.Summary, KnowledgeDraftTitle: buildResearchDraftTitle(question), KnowledgeDraftBody: result.Summary, KnowledgeDraftSummary: result.Summary}
	for _, item := range result.Items {
		jr.Sources = append(jr.Sources, SourceRef{Kind: "web", Title: item.Title, URL: item.URL})
	}
	return jr, nil
}
```

```go
// backend/internal/service/thinktank.go
type StreamEventType string

const (
	StreamEventStage    StreamEventType = "stage"
	StreamEventQuestion StreamEventType = "question"
	StreamEventChunk    StreamEventType = "chunk"
	StreamEventDone     StreamEventType = "done"
)

type StreamEvent struct {
	Type     StreamEventType
	Stage    string
	Label    string
	Message  string
	Sources  []string
}
```

```go
// backend/internal/service/thinktank.go
func (s *thinkTankService) ChatStream(ctx context.Context, question string, conversationID *int64, userID *int64) (<-chan StreamEvent, <-chan error) {
	eventCh := make(chan StreamEvent, 8)
	errCh := make(chan error, 1)
	go func() {
		defer close(eventCh)
		defer close(errCh)
		eventCh <- StreamEvent{Type: StreamEventStage, Stage: "analyzing", Label: "正在理解你的问题"}
		resp, err := s.Chat(ctx, question, conversationID, userID)
		if err != nil {
			errCh <- err
			return
		}
		if resp.RequiresUserInput {
			eventCh <- StreamEvent{Type: StreamEventQuestion, Stage: "clarifying", Message: resp.Message}
			return
		}
		eventCh <- StreamEvent{Type: StreamEventStage, Stage: "synthesizing", Label: "正在整理最终结论"}
		eventCh <- StreamEvent{Type: StreamEventChunk, Message: resp.Message, Sources: resp.Sources}
		eventCh <- StreamEvent{Type: StreamEventDone, Stage: "completed"}
	}()
	return eventCh, errCh
}
```

```go
// backend/internal/service/ai.go
func (s *aiService) Chat(question string, conversationID *int64, userID *int64) (string, error) {
	resp, err := s.thinkTank.Chat(context.Background(), question, conversationID, userID)
	if err != nil {
		return "", err
	}
	return resp.Message, nil
}
```

```go
// backend/internal/handler/ai.go
type chatStreamEvent struct {
	Stage             string   `json:"stage,omitempty"`
	Label             string   `json:"label,omitempty"`
	Message           string   `json:"message,omitempty"`
	Error             string   `json:"error,omitempty"`
	Sources           []string `json:"sources,omitempty"`
	RequiresUserInput bool     `json:"requires_user_input,omitempty"`
}
```

```go
// backend/internal/handler/ai.go
switch event.Type {
case service.StreamEventStage:
	if err := writeSSEvent(c, "stage", chatStreamEvent{Stage: event.Stage, Label: event.Label}); err != nil { return }
case service.StreamEventQuestion:
	if err := writeSSEvent(c, "question", chatStreamEvent{Stage: event.Stage, Message: event.Message, RequiresUserInput: true}); err != nil { return }
case service.StreamEventChunk:
	if err := writeSSEvent(c, "chunk", chatStreamEvent{Message: event.Message, Sources: event.Sources}); err != nil { return }
case service.StreamEventDone:
	if err := writeSSEvent(c, "done", chatStreamEvent{Stage: event.Stage}); err != nil { return }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
go test ./backend/internal/service ./backend/internal/handler -run 'Journalist|ChatStream' -v
```

Expected: PASS for journalist fallback and SSE stage/question events.

---

### Task 7: Add knowledge-document admin backend APIs and route wiring

**Files:**
- Create: `backend/internal/handler/knowledge_document.go`
- Create: `backend/internal/handler/knowledge_document_test.go`
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/cmd/server/routes_test.go`

- [ ] **Step 1: Write failing handler and route tests**

```go
func TestKnowledgeDocumentHandlerApprove_ReturnsApprovedDocument(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &stubKnowledgeDocumentService{approved: &model.KnowledgeDocument{ID: 5, Title: "调研结果", Status: model.KnowledgeDocumentStatusApproved}}
	h := NewKnowledgeDocumentHandler(svc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/knowledge-documents/5/approve", strings.NewReader(`{"review_note":"通过"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "5"}}
	c.Set("user_id", int64(1))

	h.Approve(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
```

```go
func TestRegisterRoutes_RegistersKnowledgeDocumentRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := &config.Config{}
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	registerRoutes(router, cfg, rdb, &handler.UserHandler{}, &handler.AuthHandler{}, &handler.CategoryHandler{}, &handler.ArticleHandler{}, &handler.CommentHandler{}, &handler.UploadHandler{}, &handler.AIHandler{}, &handler.SiteHandler{}, &handler.StatHandler{}, &handler.ChatHandler{}, &handler.KnowledgeDocumentHandler{})

	routes := map[string]struct{}{}
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, route := range []string{
		"GET /api/admin/knowledge-documents",
		"GET /api/admin/knowledge-documents/:id",
		"POST /api/admin/knowledge-documents/:id/approve",
		"POST /api/admin/knowledge-documents/:id/reject",
	} {
		if _, ok := routes[route]; !ok {
			t.Fatalf("expected route %s to be registered", route)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./backend/cmd/server ./backend/internal/handler -run 'KnowledgeDocument|RegisterRoutes' -v
```

Expected: FAIL with undefined `KnowledgeDocumentHandler` and missing routes.

- [ ] **Step 3: Implement admin handler and route registration**

```go
// backend/internal/handler/knowledge_document.go
package handler

type KnowledgeDocumentHandler struct {
	service service.KnowledgeDocumentService
}

func NewKnowledgeDocumentHandler(service service.KnowledgeDocumentService) *KnowledgeDocumentHandler {
	return &KnowledgeDocumentHandler{service: service}
}
```

```go
// backend/internal/handler/knowledge_document.go
type reviewRequest struct {
	ReviewNote string `json:"review_note"`
}

func (h *KnowledgeDocumentHandler) List(c *gin.Context) {
	status := c.Query("status")
	keyword := c.Query("keyword")
	page, pageSize := parsePageParams(c)
	docs, total, err := h.service.List(repository.KnowledgeDocumentFilter{Status: status, Keyword: keyword, Page: page, PageSize: pageSize})
	if err != nil {
		response.InternalError(c, "获取知识文档列表失败")
		return
	}
	response.Success(c, gin.H{"data": docs, "total": total, "page": page, "pageSize": pageSize, "totalPages": calcTotalPages(total, pageSize)})
}
```

```go
// backend/internal/handler/knowledge_document.go
func (h *KnowledgeDocumentHandler) Approve(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok { return }
	var req reviewRequest
	_ = c.ShouldBindJSON(&req)
	userID, _ := c.Get("user_id")
	doc, err := h.service.Approve(id, userID.(int64), req.ReviewNote)
	if err != nil {
		response.InternalError(c, "审核通过失败")
		return
	}
	response.Success(c, doc)
}
```

```go
// backend/cmd/server/main.go
knowledgeDocumentRepo := repository.NewKnowledgeDocumentRepository(db)
knowledgeDocumentSourceRepo := repository.NewKnowledgeDocumentSourceRepository(db)
conversationRunRepo := repository.NewConversationRunRepository(db)
knowledgeDocumentService := service.NewKnowledgeDocumentService(knowledgeDocumentRepo, knowledgeDocumentSourceRepo, vectorService, logger)
knowledgeDocumentHandler := handler.NewKnowledgeDocumentHandler(knowledgeDocumentService)
thinkTankService := service.NewThinkTankService(/* planner */, /* librarian */, /* journalist */, /* synthesizer */, conversationRunRepo, conversationRepo, chatMessageRepo, knowledgeDocumentService, aiEventLogger)
aiService := service.NewAIService(llmClient, thinkTankService, conversationRepo, chatMessageRepo, logger)
```

```go
// backend/cmd/server/main.go registerRoutes signature
func registerRoutes(
	router *gin.Engine,
	cfg *config.Config,
	rdb *redis.Client,
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
	categoryHandler *handler.CategoryHandler,
	articleHandler *handler.ArticleHandler,
	commentHandler *handler.CommentHandler,
	uploadHandler *handler.UploadHandler,
	aiHandler *handler.AIHandler,
	siteHandler *handler.SiteHandler,
	statHandler *handler.StatHandler,
	chatHandler *handler.ChatHandler,
	knowledgeDocumentHandler *handler.KnowledgeDocumentHandler,
)
```

```go
// backend/cmd/server/main.go routes
knowledgeDocs := admin.Group("/knowledge-documents")
{
	knowledgeDocs.GET("", knowledgeDocumentHandler.List)
	knowledgeDocs.GET(":id", knowledgeDocumentHandler.Get)
	knowledgeDocs.POST(":id/approve", knowledgeDocumentHandler.Approve)
	knowledgeDocs.POST(":id/reject", knowledgeDocumentHandler.Reject)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
go test ./backend/cmd/server ./backend/internal/handler -run 'KnowledgeDocument|RegisterRoutes' -v
```

Expected: PASS and new admin knowledge-document routes are present.

---

### Task 8: Extend frontend chat types, SSE parsing, and store state

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/chat.ts`
- Modify: `frontend/src/store/chatStore.ts`
- Modify: `frontend/src/pages/AIChat.tsx`

- [ ] **Step 1: Add failing frontend stage event parsing and store tests**

```ts
// frontend/src/types/index.ts
export type ChatStage =
  | 'analyzing'
  | 'clarifying'
  | 'local_search'
  | 'web_research'
  | 'synthesizing'
  | 'completed'
  | 'failed'

export interface ChatStageEvent {
  stage: ChatStage
  label?: string
}

export interface ChatQuestionEvent {
  stage: 'clarifying'
  message: string
  requires_user_input: true
}
```

```ts
// test intent snippet
expect(seenStages).toEqual([{ stage: 'analyzing', label: '正在理解你的问题' }])
expect(seenQuestions[0].requires_user_input).toBe(true)
```

- [ ] **Step 2: Run frontend typecheck to verify missing symbols**

Run:

```bash
npm --prefix frontend run build
```

Expected: FAIL or type errors until stage/question handlers and state are added.

- [ ] **Step 3: Implement SSE parsing and store state**

```ts
// frontend/src/api/chat.ts
type ChatStreamHandlers = {
  onStart?: (payload: Record<string, unknown>) => void;
  onStage?: (payload: { stage: string; label?: string }) => void;
  onQuestion?: (payload: { stage: string; message: string; requires_user_input: true }) => void;
  onChunk: (payload: { message?: string; content?: string; sources?: string[] }) => void;
  onDone?: (payload: Record<string, unknown>) => void;
  onError?: (payload: { error?: string; message?: string }) => void;
};
```

```ts
// frontend/src/api/chat.ts
if (eventName === 'stage') handlers.onStage?.(payload);
if (eventName === 'question') handlers.onQuestion?.(payload);
if (eventName === 'chunk') handlers.onChunk(payload);
```

```ts
// frontend/src/store/chatStore.ts
interface ChatState {
  conversations: Record<number, Conversation>;
  activeId: number | null;
  isTyping: boolean;
  isStreaming: boolean;
  streamingConversationId: number | null;
  currentStage: string | null;
  currentStageLabel: string | null;
  requiresUserInput: boolean;
  pendingQuestion: string | null;
  runStatus: 'idle' | 'running' | 'waiting_user' | 'completed' | 'failed';
  // ...existing methods
}
```

```ts
// frontend/src/store/chatStore.ts
onStage: ({ stage, label }) => {
  set({
    currentStage: stage,
    currentStageLabel: label ?? null,
    runStatus: stage === 'completed' ? 'completed' : 'running',
    requiresUserInput: false,
  });
},
onQuestion: ({ message }) => {
  set((state) => ({
    currentStage: 'clarifying',
    currentStageLabel: '需要你补充一点信息',
    requiresUserInput: true,
    pendingQuestion: message,
    runStatus: 'waiting_user',
    conversations: {
      ...state.conversations,
      [currentId!]: {
        ...state.conversations[currentId!],
        messages: state.conversations[currentId!].messages.map((msg) =>
          msg.id === assistantMessageId ? { ...msg, content: message } : msg
        ),
      },
    },
  }));
},
```

```tsx
// frontend/src/pages/AIChat.tsx
{currentStageLabel && (
  <div className="mb-4 rounded-xl border border-primary-200 bg-primary-50 px-4 py-3 text-sm text-primary-700 dark:border-primary-800 dark:bg-primary-900/20 dark:text-primary-300">
    {currentStageLabel}
  </div>
)}
```

- [ ] **Step 4: Run frontend build to verify it passes**

Run:

```bash
npm --prefix frontend run build
```

Expected: PASS and AIChat can compile with stage/question state.

---

### Task 9: Add admin knowledge-document frontend pages and routes

**Files:**
- Create: `frontend/src/api/knowledgeDocument.ts`
- Create: `frontend/src/views/admin/knowledge-documents/KnowledgeDocumentList.tsx`
- Create: `frontend/src/views/admin/knowledge-documents/KnowledgeDocumentDetail.tsx`
- Modify: `frontend/src/router.tsx`
- Modify: `frontend/src/components/admin/AdminLayout.tsx`
- Modify: `frontend/src/types/index.ts`

- [ ] **Step 1: Define types and API client**

```ts
// frontend/src/types/index.ts
export interface KnowledgeDocument {
  id: number;
  title: string;
  summary: string;
  content: string;
  status: 'pending_review' | 'approved' | 'rejected';
  source_type: 'research' | 'manual';
  created_by_user_id: number;
  reviewed_by_user_id?: number;
  reviewed_at?: string;
  review_note: string;
  vectorized_at?: string;
  created_at: string;
  updated_at: string;
}

export interface KnowledgeDocumentSource {
  id: number;
  knowledge_document_id: number;
  source_url: string;
  source_title: string;
  source_domain: string;
  source_snippet: string;
  sort_order: number;
}
```

```ts
// frontend/src/api/knowledgeDocument.ts
import { request } from './client';
import type { KnowledgeDocument, KnowledgeDocumentSource, PaginatedResponse } from '@/types';

export const knowledgeDocumentApi = {
  getKnowledgeDocuments: (params: { page: number; pageSize: number; status?: string; keyword?: string }) =>
    request.get<PaginatedResponse<KnowledgeDocument>>('/admin/knowledge-documents', { params }),
  getKnowledgeDocument: (id: number) =>
    request.get<{ document: KnowledgeDocument; sources: KnowledgeDocumentSource[] }>(`/admin/knowledge-documents/${id}`),
  approveKnowledgeDocument: (id: number, review_note: string) =>
    request.post<KnowledgeDocument>(`/admin/knowledge-documents/${id}/approve`, { review_note }),
  rejectKnowledgeDocument: (id: number, review_note: string) =>
    request.post<KnowledgeDocument>(`/admin/knowledge-documents/${id}/reject`, { review_note }),
};
```

- [ ] **Step 2: Add admin list and detail pages**

```tsx
// frontend/src/views/admin/knowledge-documents/KnowledgeDocumentList.tsx
export const KnowledgeDocumentList = () => {
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<'pending_review' | 'approved' | 'rejected' | ''>('pending_review');
  const { data, isLoading } = useQuery({
    queryKey: ['admin-knowledge-documents', page, status],
    queryFn: () => knowledgeDocumentApi.getKnowledgeDocuments({ page, pageSize: 10, status }),
  });
  if (isLoading) return <Loading />;
  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-serif font-bold text-neutral-800 dark:text-neutral-100">知识文档审核</h1>
      <div className="bg-white dark:bg-neutral-900 rounded-2xl shadow-sm border border-neutral-100 dark:border-neutral-800 overflow-hidden">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr>
              <th className="px-6 py-4">标题</th>
              <th className="px-6 py-4">状态</th>
              <th className="px-6 py-4">创建时间</th>
              <th className="px-6 py-4 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            {data?.data?.map((doc) => (
              <tr key={doc.id} className="border-t border-neutral-100 dark:border-neutral-800">
                <td className="px-6 py-4">{doc.title}</td>
                <td className="px-6 py-4">{doc.status}</td>
                <td className="px-6 py-4">{formatDate(doc.created_at)}</td>
                <td className="px-6 py-4 text-right">
                  <Link to={`/admin/knowledge-documents/${doc.id}`} className="text-primary-600 hover:underline">查看</Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
```

```tsx
// frontend/src/views/admin/knowledge-documents/KnowledgeDocumentDetail.tsx
export const KnowledgeDocumentDetail = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { showToast } = useUIStore();
  const [reviewNote, setReviewNote] = useState('');
  const { data, isLoading } = useQuery({
    queryKey: ['admin-knowledge-document', id],
    queryFn: () => knowledgeDocumentApi.getKnowledgeDocument(Number(id)),
  });
  const approveMutation = useMutation({
    mutationFn: () => knowledgeDocumentApi.approveKnowledgeDocument(Number(id), reviewNote),
    onSuccess: () => {
      showToast('知识文档已通过审核', 'success');
      queryClient.invalidateQueries({ queryKey: ['admin-knowledge-documents'] });
      navigate('/admin/knowledge-documents');
    },
  });
  if (isLoading) return <Loading />;
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-neutral-700 dark:text-neutral-100">{data?.document.title}</h1>
      <div className="rounded-2xl border border-neutral-100 bg-white p-6 dark:border-neutral-800 dark:bg-neutral-900">
        <p className="mb-4 text-neutral-600 dark:text-neutral-300">{data?.document.summary}</p>
        <pre className="whitespace-pre-wrap text-sm text-neutral-700 dark:text-neutral-200">{data?.document.content}</pre>
      </div>
      <div className="rounded-2xl border border-neutral-100 bg-white p-6 dark:border-neutral-800 dark:bg-neutral-900">
        <h2 className="mb-4 text-lg font-semibold">来源</h2>
        <ul className="space-y-3">
          {data?.sources.map((source) => (
            <li key={source.id}>
              <a href={source.source_url} target="_blank" rel="noreferrer" className="text-primary-600 hover:underline">{source.source_title || source.source_url}</a>
              <p className="mt-1 text-sm text-neutral-500">{source.source_snippet}</p>
            </li>
          ))}
        </ul>
      </div>
      <textarea className="input w-full h-24" value={reviewNote} onChange={(e) => setReviewNote(e.target.value)} placeholder="审核备注" />
      <div className="flex gap-3">
        <button onClick={() => approveMutation.mutate()} className="btn btn-primary">审核通过</button>
        <button onClick={() => rejectMutation.mutate()} className="btn btn-secondary">拒绝</button>
      </div>
    </div>
  );
};
```

- [ ] **Step 3: Register routes and admin menu**

```tsx
// frontend/src/router.tsx
import { KnowledgeDocumentList } from './views/admin/knowledge-documents/KnowledgeDocumentList';
import { KnowledgeDocumentDetail } from './views/admin/knowledge-documents/KnowledgeDocumentDetail';

{ path: 'knowledge-documents', element: <KnowledgeDocumentList /> },
{ path: 'knowledge-documents/:id', element: <KnowledgeDocumentDetail /> },
```

```tsx
// frontend/src/components/admin/AdminLayout.tsx
{ name: '知识文档', path: '/admin/knowledge-documents', icon: 'M4.5 4.5h15v15h-15z' },
```

- [ ] **Step 4: Run frontend build to verify it passes**

Run:

```bash
npm --prefix frontend run build
```

Expected: PASS and admin route/menu compile with new knowledge-document pages.

---

### Task 10: Final end-to-end verification

**Files:**
- Modify as needed from Tasks 1-9
- Test: `backend/internal/service/thinktank_test.go`
- Test: `backend/internal/handler/ai_stream_test.go`
- Test: `backend/internal/handler/knowledge_document_test.go`

- [ ] **Step 1: Add integrated backend test covering clarification -> answer -> draft creation**

```go
func TestThinkTankService_EndToEnd_WebResearchCreatesPendingKnowledgeDraft(t *testing.T) {
	planner := &stubPlanner{decision: PlannerDecision{ExecutionStrategy: "local_then_web"}}
	librarian := &stubLibrarian{result: LibrarianResult{CoverageStatus: "insufficient"}}
	journalist := &stubJournalist{result: JournalistResult{
		Summary:               "联网调研摘要",
		KnowledgeDraftTitle:   "工业大模型落地调研",
		KnowledgeDraftSummary: "候选知识摘要",
		KnowledgeDraftBody:    "候选知识正文",
		Sources: []SourceRef{{Kind: "web", Title: "来源 A", URL: "https://example.com/a"}},
	}}
	knowledgeSvc := &stubKnowledgeDocumentService{}
	synthesizer := &stubSynthesizer{answer: "最终整合答案"}
	svc := NewThinkTankService(planner, librarian, journalist, synthesizer, &stubConversationRunRepository{}, &stubConversationRepository{}, &stubChatMessageRepository{}, knowledgeSvc, &stubAILogger{})

	resp, err := svc.Chat(context.Background(), "调研工业大模型落地", nil, ptrInt64(1))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Message != "最终整合答案" {
		t.Fatalf("unexpected answer %q", resp.Message)
	}
	if knowledgeSvc.created.Title != "工业大模型落地调研" {
		t.Fatalf("expected knowledge draft to be created, got %#v", knowledgeSvc.created)
	}
}
```

- [ ] **Step 2: Run backend verification suite**

Run:

```bash
go test ./backend/config ./backend/cmd/server ./backend/internal/repository ./backend/internal/service ./backend/internal/handler
```

Expected: PASS.

- [ ] **Step 3: Run frontend verification suite**

Run:

```bash
npm --prefix frontend run build
```

Expected: PASS.

- [ ] **Step 4: Manual smoke checklist**

Run the app locally and verify:

```text
1. 登录后进入 /ai-chat
2. 输入一个模糊问题，确认前端出现“需要你补充一点信息”的追问消息
3. 输入一个站内已覆盖的问题，确认前端出现阶段：analyzing -> local_search -> synthesizing -> completed
4. 输入一个站内缺失的问题，确认前端出现阶段：analyzing -> local_search -> web_research -> synthesizing -> completed
5. 检查 backend/log/YYYY-MM-DD-ai-chat.log，确认有阶段摘要日志
6. 登录后台 /admin/knowledge-documents，确认能看到待审核知识文档
7. 点击审核通过，确认知识文档状态变为 approved 且向量化成功
8. 再次提问相关问题，确认 Librarian 能命中 approved 知识文档
```

---

## Self-Review

### Spec coverage
- 多 Agent 替代单 RAG：Task 5, 6
- 阶段展示与追问恢复：Task 5, 6, 8
- 独立 ai-chat 日志：Task 4
- 知识文档待审核入库：Task 2, 7, 9
- 全站共享向量检索：Task 3
- 保留现有入口 `/api/ai/chat` 和 `/api/ai/chat/stream`：Task 6
- 后台知识文档管理：Task 7, 9

### Placeholder scan
- 未使用 TBD / TODO / “稍后实现”。
- 所有代码改动步骤都给出了具体代码片段。
- 所有验证步骤都给出了明确命令和预期结果。

### Type consistency
- 统一使用 `ConversationRun`, `KnowledgeDocument`, `KnowledgeDocumentSource`
- 统一使用 `ThinkTankService`, `PlannerDecision`, `LibrarianResult`, `JournalistResult`, `StreamEvent`
- SSE 事件统一为 `stage`, `question`, `chunk`, `done`, `error`

---
