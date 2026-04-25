# AI Chat 断线重连与心跳 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 AI 流式对话补上可恢复运行、自动重连和 SSE 心跳，保证页面刷新或短暂断线后还能接回同一次回答。

**Architecture:** 后端在 `conversation_run` 基础上增加运行快照和运行级广播中心，前端在会话详情加载后自动判断是否存在可恢复 run，并通过恢复流接口重新订阅。SSE 层新增心跳和快照事件，前后端都以 run 作为恢复单位，而不是重新发起一次新问题。

**Tech Stack:** Go, Gin, GORM, React, Zustand, TypeScript, SSE

---

### Task 1: 补齐后端恢复协议的数据结构

**Files:**
- Modify: `backend/internal/model/conversation_run.go`
- Modify: `backend/internal/repository/chat/conversation_run.go`
- Modify: `backend/internal/handler/chat/chat.go`
- Test: `backend/internal/handler/chat/chat_test.go`

- [ ] 新增 `ConversationRun` 快照字段和仓储查询能力
- [ ] 会话详情接口返回 `active_run`
- [ ] 增加测试覆盖 `active_run` 序列化结果

### Task 2: 引入运行级广播中心和快照更新

**Files:**
- Create: `backend/internal/service/chat/run_hub.go`
- Modify: `backend/internal/service/chat/thinktank_service.go`
- Modify: `backend/internal/service/chat/thinktank_orchestrator.go`
- Modify: `backend/internal/service/chat/thinktank_run_record.go`
- Modify: `backend/internal/service/chat/thinktank_stream.go`
- Test: `backend/internal/service/chat/thinktank_test.go`

- [ ] 新增 `chatRunHub` 管理运行中的订阅、快照和清理
- [ ] 在 `stage/chunk/step/question/done/error` 关键节点更新快照
- [ ] 为运行结束后的短暂恢复窗口保留 hub 状态
- [ ] 增加测试覆盖快照更新和 run 结束行为

### Task 3: 新增恢复流接口和心跳事件

**Files:**
- Modify: `backend/internal/handler/chat/ai.go`
- Modify: `backend/internal/service/ai/ai.go`
- Modify: `backend/internal/service/chat/thinktank_service.go`
- Modify: `backend/internal/service/chat/thinktank_stream.go`
- Test: `backend/internal/handler/chat/ai_stream_test.go`
- Test: `backend/internal/handler/chat/ai_test.go`

- [ ] 新增 `POST /api/ai/chat/stream/resume`
- [ ] SSE 增加 `resume`、`snapshot`、`heartbeat`
- [ ] 心跳定时发送并在恢复连接后立即发一次
- [ ] 测试覆盖 `running/waiting_user/completed/not_found`

### Task 4: 前端聊天状态支持恢复和重连

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/chat.ts`
- Modify: `frontend/src/store/chatStore.ts`

- [ ] 为会话详情和 SSE 新增 `active_run`、`snapshot`、`heartbeat` 类型
- [ ] 前端流式 API 支持普通流和恢复流
- [ ] store 增加 `activeRun`、`lastHeartbeatAt`、`isRecovering`
- [ ] 进入会话详情后如果存在活跃 run，自动恢复

### Task 5: 心跳超时与有限次自动重连

**Files:**
- Modify: `frontend/src/store/chatStore.ts`
- Modify: `frontend/src/api/chat.ts`

- [ ] 所有流式事件都刷新最后活跃时间
- [ ] 心跳超时后仅对 `running` run 发起有限次自动重连
- [ ] 重连成功后继续覆盖当前 assistant 消息
- [ ] 超过次数时落到可手动恢复状态，不重复创建新消息

### Task 6: 端到端验证与提交

**Files:**
- Verify: `backend/internal/handler/chat/*`
- Verify: `backend/internal/service/chat/*`
- Verify: `frontend/src/store/chatStore.ts`
- Verify: `frontend/src/api/chat.ts`

- [ ] 运行 `env GOTOOLCHAIN=go1.25.3 go test ./internal/handler/chat ./internal/service/chat`
- [ ] 运行 `env GOTOOLCHAIN=go1.25.3 go test ./...`
- [ ] 运行 `npm run lint`
- [ ] 运行 `npm run build`
- [ ] 提交并推送到 `origin/main`
