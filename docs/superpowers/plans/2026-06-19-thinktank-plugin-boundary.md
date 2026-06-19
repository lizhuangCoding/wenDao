# ThinkTank Plugin Boundary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the current ThinkTank multi-agent service into the default in-process agent plugin behind a stable plugin interface.

**Architecture:** Add plugin contracts and an in-memory registry in `backend/internal/service/chatcore`, adapt the existing ThinkTank service with a `ThinkTankPlugin`, and make `AIService` depend on the plugin interface instead of the concrete ThinkTank interface. Existing API routes, SSE event payloads, run persistence, and ThinkTank internals stay unchanged.

**Tech Stack:** Go, existing `chatcore.StreamEvent`, existing `chatcore.ThinkTankService`, standard `testing` package.

---

### File Structure

- Create `backend/internal/service/chatcore/plugin.go`: plugin manifest, run input, resume input, plugin interface, and adapter from `ThinkTankService`.
- Create `backend/internal/service/chatcore/registry.go`: in-memory plugin registry and default plugin selection.
- Create `backend/internal/service/chatcore/plugin_test.go`: contract tests for the ThinkTank adapter and registry.
- Modify `backend/internal/service/ai/ai.go`: replace direct `ThinkTankService` dependency with default `AgentPlugin`.
- Modify `backend/internal/service/ai/ai_context_test.go`: update stub to the plugin interface and verify context propagation through the plugin boundary.
- Modify `backend/internal/service/facade.go`: expose plugin types and constructors.
- Modify `backend/cmd/server/bootstrap_services.go`: register the ThinkTank plugin and pass the registry/default plugin into AI service construction.

### Task 1: Define Plugin Contract And Registry

**Files:**
- Create: `backend/internal/service/chatcore/plugin.go`
- Create: `backend/internal/service/chatcore/registry.go`
- Test: `backend/internal/service/chatcore/plugin_test.go`

- [ ] **Step 1: Write failing adapter and registry tests**

```go
func TestThinkTankPluginDelegatesChatAndStream(t *testing.T) {
    inner := &recordingThinkTankService{chatMessage: "plugin ok"}
    plugin := NewThinkTankPlugin(inner)
    response, err := plugin.Run(context.Background(), AgentRunInput{Question: "hello"})
    if err != nil || response.Message != "plugin ok" {
        t.Fatalf("expected delegated chat response, got response=%#v err=%v", response, err)
    }
    events, errs := plugin.RunStream(context.Background(), AgentRunInput{Question: "stream"})
    if event := <-events; event.Type != StreamEventChunk || event.Message != "stream ok" {
        t.Fatalf("expected delegated stream event, got %#v", event)
    }
    if err := <-errs; err != nil {
        t.Fatalf("expected stream err channel to close cleanly, got %v", err)
    }
}

func TestPluginRegistryReturnsDefaultPlugin(t *testing.T) {
    registry := NewPluginRegistry()
    plugin := NewThinkTankPlugin(&recordingThinkTankService{})
    if err := registry.Register(plugin, WithDefaultPlugin()); err != nil {
        t.Fatalf("expected register success, got %v", err)
    }
    got, ok := registry.Default()
    if !ok || got.Manifest().Name != "thinktank" {
        t.Fatalf("expected thinktank default plugin, got %#v ok=%v", got, ok)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chatcore`

Expected: FAIL because `NewThinkTankPlugin`, `AgentRunInput`, `NewPluginRegistry`, and `WithDefaultPlugin` do not exist.

- [ ] **Step 3: Implement minimal plugin contract and registry**

Add:

```go
type AgentPlugin interface {
    Manifest() PluginManifest
    Run(ctx context.Context, input AgentRunInput) (*ThinkTankChatResponse, error)
    RunStream(ctx context.Context, input AgentRunInput) (<-chan StreamEvent, <-chan error)
    ResumeStream(ctx context.Context, input AgentResumeInput) (<-chan StreamEvent, <-chan error)
}
```

The `ThinkTankPlugin` forwards to the wrapped `ThinkTankService`. The registry stores plugins by manifest name and tracks one default plugin.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && env GOTOOLCHAIN=go1.25.3 go test ./internal/service/chatcore`

Expected: PASS.

### Task 2: Route AI Service Through AgentPlugin

**Files:**
- Modify: `backend/internal/service/ai/ai.go`
- Modify: `backend/internal/service/ai/ai_context_test.go`

- [ ] **Step 1: Write failing AI service plugin test**

Update `ai_context_test.go` so the stub implements `chatcore.AgentPlugin`, then assert `AIService.Chat`, `ChatStream`, and `ResumeChatStream` call the plugin with caller context and input IDs.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && env GOTOOLCHAIN=go1.25.3 go test ./internal/service/ai -run TestAIService`

Expected: FAIL while `NewAIService` still expects `ThinkTankService`.

- [ ] **Step 3: Change AI service dependency**

Change `aiService` from:

```go
thinkTank chatcore.ThinkTankService
```

to:

```go
agent chatcore.AgentPlugin
```

`Chat`, `ChatStream`, and `ResumeChatStream` should call `agent.Run`, `agent.RunStream`, and `agent.ResumeStream`.

- [ ] **Step 4: Run AI service tests**

Run: `cd backend && env GOTOOLCHAIN=go1.25.3 go test ./internal/service/ai`

Expected: PASS.

### Task 3: Register ThinkTank As Default Plugin

**Files:**
- Modify: `backend/internal/service/facade.go`
- Modify: `backend/cmd/server/bootstrap_services.go`
- Test: existing backend tests

- [ ] **Step 1: Expose constructors through the service facade**

Expose `AgentPlugin`, `PluginRegistry`, `NewThinkTankPlugin`, `NewPluginRegistry`, and `WithDefaultPlugin` from `backend/internal/service/facade.go`.

- [ ] **Step 2: Wire bootstrap through the registry**

After constructing `thinkTankService`, create a registry, register `NewThinkTankPlugin(thinkTankService)` as default, retrieve the default plugin, and pass that plugin into `NewAIService`.

- [ ] **Step 3: Run backend tests**

Run: `cd backend && env GOTOOLCHAIN=go1.25.3 go test ./...`

Expected: PASS.

### Self-Review

- Spec coverage: The plan adds a standard plugin interface, wraps ThinkTank as a plugin, and leaves remote HTTP/gRPC for a later adapter.
- Placeholder scan: No TBD/TODO placeholders remain.
- Type consistency: The plan consistently uses `AgentPlugin`, `AgentRunInput`, `AgentResumeInput`, `PluginRegistry`, and `ThinkTankPlugin`.
