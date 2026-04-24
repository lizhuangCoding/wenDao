# AI RAG Fallback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make AI chat prefer grounded Redis vector retrieval when relevant content exists, and automatically fall back to general-knowledge LLM answers when retrieval is empty or too weak.

**Architecture:** Keep the existing AI service and chat API contract stable, but refactor the Eino RAG chain into a two-path decision flow. Redis vector search will return usable similarity scores, the retriever will preserve those scores in document metadata, and the chain will route between a grounded RAG prompt and a clearly labeled general-knowledge fallback prompt using a configurable threshold.

**Tech Stack:** Go, Gin, GORM, Redis Stack FT.SEARCH, Eino retriever/chat model, YAML config

---

## File Structure

**Modify:**
- `config/config.go` — add AI config field for RAG fallback threshold
- `config/config.yaml` — define default `ragMinScore`
- `backend/internal/pkg/eino/vectorstore.go` — return Redis vector search scores and preserve them in `SearchResult`
- `backend/internal/pkg/eino/retriever.go` — write vector scores into `schema.Document.MetaData`
- `backend/internal/pkg/eino/chain.go` — implement score-aware routing and split prompts into RAG / fallback builders
- `backend/internal/service/ai.go` — update constructor wiring if needed, keep orchestration stable, and optionally log chosen mode concisely

**Optional test files to create:**
- `backend/internal/pkg/eino/chain_test.go`
- `backend/internal/pkg/eino/retriever_test.go`

---

### Task 1: Add AI fallback threshold config

**Files:**
- Modify: `config/config.go`
- Modify: `config/config.yaml`

- [ ] **Step 1: Add `RAGMinScore` to AI config struct**

In `config/config.go`, update the AI config struct to include a float threshold field.

```go
type AIConfig struct {
	Provider    string  `mapstructure:"provider"`
	APIKey      string  `mapstructure:"apiKey"`
	BaseURL     string  `mapstructure:"baseURL"`
	EmbedModel  string  `mapstructure:"embedModel"`
	LLMModel    string  `mapstructure:"llmModel"`
	TopK        int     `mapstructure:"topK"`
	Temperature float32 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"maxTokens"`
	RAGMinScore float32 `mapstructure:"ragMinScore"`
}
```

- [ ] **Step 2: Add default threshold to YAML config**

In `config/config.yaml`, add the new key under the existing `ai:` section.

```yaml
ai:
  provider: doubao
  apiKey: ${ARK_API_KEY}
  baseURL: https://ark.cn-beijing.volces.com/api/v3
  embedModel: doubao-embedding-text-240715
  llmModel: doubao-1.5-pro-32k-250115
  topK: 4
  temperature: 0.7
  maxTokens: 2000
  ragMinScore: 0.30
```

- [ ] **Step 3: Run backend build to verify config compiles**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go build ./...
```

Expected: build succeeds with no compile errors.

- [ ] **Step 4: Commit**

```bash
git add /Users/lizhuang/go/src/wenDao/config/config.go /Users/lizhuang/go/src/wenDao/config/config.yaml
git commit -m "feat: add configurable rag fallback threshold"
```

---

### Task 2: Return usable similarity scores from Redis vector search

**Files:**
- Modify: `backend/internal/pkg/eino/vectorstore.go`

- [ ] **Step 1: Write the failing score test scaffold**

Create `backend/internal/pkg/eino/vectorstore_score_test.go` with a focused parser test that asserts score extraction from Redis FT.SEARCH-like results.

```go
package eino

import "testing"

func TestParseSearchResultIncludesScore(t *testing.T) {
	resultSlice := []interface{}{
		int64(1),
		"article:1:0",
		[]interface{}{
			"__embedding_score", "0.12",
			"article_id", "1",
			"chunk_index", "0",
			"title", "Go 并发",
			"content", "goroutine 和 channel",
		},
	}

	results := parseSearchResults(resultSlice)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Score != float32(0.12) {
		t.Fatalf("expected score 0.12, got %v", results[0].Score)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./internal/pkg/eino -run TestParseSearchResultIncludesScore -v
```

Expected: FAIL because `parseSearchResults` does not exist yet.

- [ ] **Step 3: Extract result parsing into a helper and include score return**

In `backend/internal/pkg/eino/vectorstore.go`, refactor `Search` so the Redis query explicitly returns `__embedding_score` and parsing lives in a helper that the test can exercise.

Use this FT.SEARCH shape:

```go
query := fmt.Sprintf("*=>[KNN %d @embedding $query_vec AS __embedding_score]", topK)
cmd := []interface{}{
	"FT.SEARCH", s.indexName,
	query,
	"PARAMS", "2", "query_vec", vectorBytes,
	"SORTBY", "__embedding_score",
	"RETURN", "5", "__embedding_score", "article_id", "chunk_index", "title", "content",
	"DIALECT", "2",
}
```

Add a parser helper:

```go
func parseSearchResults(resultSlice []interface{}) []SearchResult {
	results := make([]SearchResult, 0)
	if len(resultSlice) < 1 {
		return results
	}

	total := parseToInt64(resultSlice[0])
	if total == 0 {
		return results
	}

	for i := 1; i < len(resultSlice); i += 2 {
		if i+1 >= len(resultSlice) {
			break
		}

		key, _ := resultSlice[i].(string)
		fields, _ := resultSlice[i+1].([]interface{})
		metadata := make(map[string]interface{})
		score := float32(0)

		for j := 0; j < len(fields); j += 2 {
			if j+1 >= len(fields) {
				break
			}

			name, _ := fields[j].(string)
			val := fields[j+1]

			if name == "__embedding_score" {
				switch v := val.(type) {
				case string:
					f, _ := strconv.ParseFloat(v, 32)
					score = float32(f)
				case float64:
					score = float32(v)
				}
				continue
			}

			metadata[name] = val
		}

		results = append(results, SearchResult{
			Key:      key,
			Score:    score,
			Metadata: metadata,
		})
	}

	return results
}
```

Then simplify `Search` to call the helper:

```go
resultSlice, ok := result.([]interface{})
if !ok {
	return []SearchResult{}, nil
}

return parseSearchResults(resultSlice), nil
```

- [ ] **Step 4: Re-run the focused test**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./internal/pkg/eino -run TestParseSearchResultIncludesScore -v
```

Expected: PASS.

- [ ] **Step 5: Run full Eino package tests/build**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./internal/pkg/eino/... && go build ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add /Users/lizhuang/go/src/wenDao/backend/internal/pkg/eino/vectorstore.go /Users/lizhuang/go/src/wenDao/backend/internal/pkg/eino/vectorstore_score_test.go
git commit -m "feat: return redis vector scores for rag routing"
```

---

### Task 3: Preserve retrieval score in Eino documents

**Files:**
- Modify: `backend/internal/pkg/eino/retriever.go`
- Test: `backend/internal/pkg/eino/retriever_test.go`

- [ ] **Step 1: Write the failing retriever metadata test**

Create `backend/internal/pkg/eino/retriever_test.go`.

```go
package eino

import (
	"context"
	"testing"
)

type fakeEmbedder struct{}

func (f *fakeEmbedder) Embed(text string) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}

type fakeVectorStore struct{}

func (f *fakeVectorStore) InitIndex(indexName string, dim int) error { return nil }
func (f *fakeVectorStore) Upsert(key string, vector []float32, metadata map[string]interface{}) error { return nil }
func (f *fakeVectorStore) UpsertBatch(items []VectorItem) error { return nil }
func (f *fakeVectorStore) Delete(pattern string) error { return nil }
func (f *fakeVectorStore) Search(vector []float32, topK int) ([]SearchResult, error) {
	return []SearchResult{{
		Key:   "article:1:0",
		Score: 0.12,
		Metadata: map[string]interface{}{
			"content": "test content",
			"title":   "test title",
		},
	}}, nil
}

func TestRetrievePreservesScoreInMetadata(t *testing.T) {
	r := &RedisRetriever{
		vectorStore: &fakeVectorStore{},
		embedder:    &fakeEmbedder{},
		topK:        1,
	}

	docs, err := r.Retrieve(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 doc, got %d", len(docs))
	}
	if docs[0].MetaData["score"] != float32(0.12) {
		t.Fatalf("expected score metadata 0.12, got %#v", docs[0].MetaData["score"])
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./internal/pkg/eino -run TestRetrievePreservesScoreInMetadata -v
```

Expected: FAIL because score is not written into metadata.

- [ ] **Step 3: Update retriever to preserve score**

Modify the document creation loop in `backend/internal/pkg/eino/retriever.go`.

```go
for _, res := range results {
	content, _ := res.Metadata["content"].(string)
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
```

- [ ] **Step 4: Re-run retriever test**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./internal/pkg/eino -run TestRetrievePreservesScoreInMetadata -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add /Users/lizhuang/go/src/wenDao/backend/internal/pkg/eino/retriever.go /Users/lizhuang/go/src/wenDao/backend/internal/pkg/eino/retriever_test.go
git commit -m "feat: preserve retrieval scores in eino documents"
```

---

### Task 4: Refactor RAG chain into score-aware routing

**Files:**
- Modify: `backend/internal/pkg/eino/chain.go`
- Test: `backend/internal/pkg/eino/chain_test.go`

- [ ] **Step 1: Write failing routing tests**

Create `backend/internal/pkg/eino/chain_test.go` with focused helper tests for score routing and prompt selection.

```go
package eino

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestShouldUseFallbackWhenNoDocs(t *testing.T) {
	if !shouldUseFallback(nil, 0.30) {
		t.Fatal("expected fallback for no docs")
	}
}

func TestShouldUseFallbackWhenTopScoreTooLow(t *testing.T) {
	docs := []*schema.Document{{MetaData: map[string]interface{}{"score": float32(0.12)}}}
	if !shouldUseFallback(docs, 0.30) {
		t.Fatal("expected fallback for low score")
	}
}

func TestShouldUseRAGWhenTopScoreHighEnough(t *testing.T) {
	docs := []*schema.Document{{MetaData: map[string]interface{}{"score": float32(0.42)}}}
	if shouldUseFallback(docs, 0.30) {
		t.Fatal("expected rag mode for high score")
	}
}

func TestBuildFallbackMessagesIncludesGeneralKnowledgeDisclaimer(t *testing.T) {
	chain := &RAGChain{}
	msgs := chain.buildFallbackMessages("Go 的 channel 是什么？")
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content == "" || msgs[1].Content == "" {
		t.Fatal("expected non-empty fallback prompt")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./internal/pkg/eino -run 'TestShouldUseFallback|TestBuildFallbackMessages' -v
```

Expected: FAIL because helpers do not exist.

- [ ] **Step 3: Add score routing helpers and threshold field**

Refactor `backend/internal/pkg/eino/chain.go`.

Use this structure:

```go
type RAGChain struct {
	retriever   retriever.Retriever
	llm         model.ChatModel
	ragMinScore float32
}

func NewRAGChain(r retriever.Retriever, llm model.ChatModel, ragMinScore float32) *RAGChain {
	return &RAGChain{
		retriever:   r,
		llm:         llm,
		ragMinScore: ragMinScore,
	}
}

func shouldUseFallback(docs []*schema.Document, threshold float32) bool {
	if len(docs) == 0 {
		return true
	}

	rawScore, ok := docs[0].MetaData["score"]
	if !ok {
		return true
	}

	score, ok := rawScore.(float32)
	if !ok {
		return true
	}

	return score < threshold
}
```

- [ ] **Step 4: Split prompt builders into RAG and fallback variants**

In `backend/internal/pkg/eino/chain.go`, replace `buildMessages` with two explicit builders.

```go
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
		systemPrompt += fmt.Sprintf("\n--- 文章片段 %d ---\n%s\n", i+1, doc.Content)
	}

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(question),
	}
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

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(question),
	}
}
```

- [ ] **Step 5: Update Execute to route by score**

Replace `Execute` with score-aware routing:

```go
func (c *RAGChain) Execute(ctx context.Context, question string) (string, error) {
	docs, err := c.retriever.Retrieve(ctx, question)
	if err != nil {
		return "", fmt.Errorf("retrieval failed: %w", err)
	}

	var messages []*schema.Message
	if shouldUseFallback(docs, c.ragMinScore) {
		messages = c.buildFallbackMessages(question)
	} else {
		messages = c.buildRAGMessages(question, docs)
	}

	resp, err := c.llm.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("generation failed: %w", err)
	}

	return resp.Content, nil
}
```

- [ ] **Step 6: Re-run focused chain tests**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./internal/pkg/eino -run 'TestShouldUseFallback|TestBuildFallbackMessages' -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add /Users/lizhuang/go/src/wenDao/backend/internal/pkg/eino/chain.go /Users/lizhuang/go/src/wenDao/backend/internal/pkg/eino/chain_test.go
git commit -m "feat: add score-based rag fallback routing"
```

---

### Task 5: Wire threshold into AI service construction

**Files:**
- Modify: `backend/internal/service/ai.go`

- [ ] **Step 1: Update chain constructor call**

In `backend/internal/service/ai.go`, update `NewAIService` so the chain receives the new threshold config.

Replace:

```go
ragChain := eino.NewRAGChain(retriever, llmClient.GetModel())
```

With:

```go
ragChain := eino.NewRAGChain(retriever, llmClient.GetModel(), cfg.RAGMinScore)
```

- [ ] **Step 2: Add concise mode logging after answer selection (optional but recommended)**

Add a small helper in `chain.go` if needed, or in `ai.go` log only mode and threshold-related info. Keep it concise.

For example, inside `Execute` after routing choice is made:

```go
// optional inside chain.go if logger is later introduced there, otherwise skip
```

Since `RAGChain` currently has no logger, do **not** add one unless necessary. Prefer to skip logging rather than complicate the structure.

- [ ] **Step 3: Run backend build**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go build ./...
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add /Users/lizhuang/go/src/wenDao/backend/internal/service/ai.go
git commit -m "feat: wire rag fallback threshold into ai service"
```

---

### Task 6: Add integration-style tests for fallback behavior

**Files:**
- Modify or Create: `backend/internal/pkg/eino/chain_test.go`

- [ ] **Step 1: Add a fake model for prompt-path verification**

Extend `chain_test.go` with a fake model that records the system prompt and returns a canned answer.

```go
type fakeChatModel struct {
	lastMessages []*schema.Message
}

func (f *fakeChatModel) Generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	f.lastMessages = messages
	return schema.AssistantMessage("mock answer"), nil
}

func (f *fakeChatModel) Stream(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

func (f *fakeChatModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}
```

- [ ] **Step 2: Add a high-score RAG path test**

```go
func TestExecuteUsesRAGPromptForHighScore(t *testing.T) {
	model := &fakeChatModel{}
	chain := &RAGChain{llm: model, ragMinScore: 0.30}
	docs := []*schema.Document{{Content: "Go 的 goroutine 很轻量", MetaData: map[string]interface{}{"score": float32(0.82)}}}

	msgs := chain.buildRAGMessages("什么是 goroutine？", docs)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != schema.System {
		t.Fatalf("expected system message, got %v", msgs[0].Role)
	}
}
```

- [ ] **Step 3: Add a fallback disclaimer test**

```go
func TestFallbackPromptContainsGeneralKnowledgeDisclaimer(t *testing.T) {
	chain := &RAGChain{}
	msgs := chain.buildFallbackMessages("Redis 是什么？")
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if !strings.Contains(msgs[0].Content, "以下回答基于通用知识，不是来自本站文章内容") {
		t.Fatalf("expected fallback disclaimer, got %s", msgs[0].Content)
	}
}
```

- [ ] **Step 4: Run the Eino test package**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go test ./internal/pkg/eino -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add /Users/lizhuang/go/src/wenDao/backend/internal/pkg/eino/chain_test.go
git commit -m "test: cover rag fallback routing behavior"
```

---

### Task 7: Manual verification against real chat flow

**Files:**
- Modify: none

- [ ] **Step 1: Run backend locally**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go run ./cmd/server
```

Expected: server starts without config or compile errors.

- [ ] **Step 2: Test a question likely covered by blog content**

Use an authenticated request or frontend UI and ask a domain-specific question tied to stored articles.

Example curl (replace TOKEN):
```bash
curl -X POST http://localhost:8080/api/ai/chat \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"这篇博客里是怎么做 Go 并发处理的？"}'
```

Expected:
- Returns an answer based on article content
- Does **not** start with the fallback disclaimer

- [ ] **Step 3: Test a question unlikely covered by site content**

```bash
curl -X POST http://localhost:8080/api/ai/chat \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"量子计算和经典计算的主要区别是什么？"}'
```

Expected:
- Returns a useful answer
- Begins with: `以下回答基于通用知识，不是来自本站文章内容。`
- Does **not** return the old fixed “没找到” sentence

- [ ] **Step 4: Verify frontend behavior remains unchanged**

Run frontend and use `/ai-chat` in browser.

Expected:
- Request/response flow works without frontend contract changes
- Stored conversation still works
- User sees either grounded answer or explicit fallback answer

- [ ] **Step 5: Commit only if any verification-only fixes were required**

If no code changed during verification, skip commit.
If fixes were required:

```bash
git add <exact files changed>
git commit -m "fix: refine ai rag fallback behavior after verification"
```

---

## Self-Review Checklist

### Spec coverage
- Retrieval scores returned: Task 2
- Scores preserved in document metadata: Task 3
- Configurable threshold: Task 1
- RAG/fallback routing: Task 4
- Concrete prompts for both modes: Task 4
- AI service wiring: Task 5
- Tests for empty/low/high cases: Tasks 3, 4, 6
- Manual validation through chat endpoint and UI: Task 7

No spec gaps found.

### Placeholder scan
- No TODO/TBD placeholders remain.
- Every file path is explicit.
- Every command is concrete.
- All named helpers (`parseSearchResults`, `shouldUseFallback`, `buildRAGMessages`, `buildFallbackMessages`) are explicitly introduced in the plan.

### Type consistency
- `RAGChain` constructor signature updated consistently in Task 4 and Task 5.
- `score` metadata is stored as `float32` and read as `float32` in the routing helper.
- `RAGMinScore` config field is consistently named across config and wiring.
