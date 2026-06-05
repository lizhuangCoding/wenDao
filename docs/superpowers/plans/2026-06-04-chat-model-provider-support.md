# Chat Model Provider Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep embedding on Doubao, but make the chat model selectable so the app can run on Doubao, DeepSeek, or an OpenAI-compatible endpoint without changing the rest of the AI stack.

**Architecture:** Add a provider field to AI config, route chat-model creation through a small factory, and keep the existing `LLMClient` interface so ThinkTank/RAG call sites do not change. Leave the embedding factory untouched. Keep legacy `DOUBAO_*` env vars as fallbacks so existing deployments continue to boot.

**Tech Stack:** Go, Viper, CloudWeGo Eino, CloudWeGo Eino ext providers, Go tests

---

### Task 1: Extend AI config for chat-provider selection

**Files:**
- Modify: `backend/config/config.go`
- Modify: `backend/config/config.yaml`
- Modify: `backend/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Add a config test that sets `AI_PROVIDER`, `AI_ENDPOINT`, `AI_CHAT_MODEL`, and `AI_API_KEY` and asserts `cfg.AI.Provider`, `cfg.AI.Endpoint`, `cfg.AI.LLMModel`, and `cfg.AI.APIKey` are loaded from env.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./config -run TestLoadConfig_BindsAIProviderFromEnv -v`
Expected: FAIL because `Provider` is not yet part of `AIConfig` and env binding does not exist.

- [ ] **Step 3: Write minimal implementation**

Add `Provider string \`mapstructure:"provider"\`` to `AIConfig`. Bind `ai.provider` to `AI_PROVIDER` and `LLM_PROVIDER`, and bind generic AI env vars first with Doubao names as fallback aliases. Update `config.yaml` comments to describe `provider` as `doubao`, `deepseek`, or `openai-compatible`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./config -run TestLoadConfig_BindsAIProviderFromEnv -v`
Expected: PASS.

### Task 2: Replace the Doubao-only chat factory with a provider-aware factory

**Files:**
- Modify: `backend/internal/pkg/eino/llm.go`
- Modify: `backend/cmd/server/bootstrap_infra.go`
- Add: `backend/internal/pkg/eino/llm_test.go`

- [ ] **Step 1: Write the failing test**

Add a unit test that constructs a minimal `config.AIConfig{Provider: "deepseek", APIKey: "x", LLMModel: "y"}` and asserts the factory returns a non-nil client for supported providers and errors for an unsupported provider string.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/pkg/eino -run TestNewLLMClient_ProviderRouting -v`
Expected: FAIL because `NewLLMClient` does not exist yet.

- [ ] **Step 3: Write minimal implementation**

Create a new `NewLLMClient(cfg *config.AIConfig) (LLMClient, error)` that switches on normalized provider names:

```go
switch provider {
case "", "doubao", "ark":
    // ark.NewChatModel(...)
case "deepseek":
    // deepseek.NewChatModel(...)
case "openai", "openai-compatible":
    // openai.NewChatModel(...)
default:
    return nil, fmt.Errorf("unsupported ai provider %q", provider)
}
```

Keep the existing wrapper methods (`Chat`, `ChatStream`, `GetModel`) unchanged so `ThinkTank` and RAG still receive a `model.ChatModel` that can satisfy `ToolCallingChatModel`.

Update `initAIComponents` to call `eino.NewLLMClient(&cfg.AI)` and log the selected provider instead of hardcoding Doubao.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/pkg/eino -run TestNewLLMClient_ProviderRouting -v`
Expected: PASS.

### Task 3: Update docs and verify the backend still builds

**Files:**
- Modify: `README.md`
- Modify: `backend/config/config.yaml`

- [ ] **Step 1: Write the failing documentation check**

No code test is required here; update the README section that lists AI environment variables so it explains that chat models are selectable while embedding remains on Doubao.

- [ ] **Step 2: Run verification**

Run: `cd backend && go test ./...`
Expected: PASS.

Run: `cd backend && gofmt -w config/config.go config/config_test.go internal/pkg/eino/llm.go internal/pkg/eino/llm_test.go cmd/server/bootstrap_infra.go && go test ./...`
Expected: PASS.

