# AI Chat 断线重连与心跳设计

**目标**

让 AI 流式对话在页面关闭、刷新、网络短暂中断后仍能恢复到同一次运行中的回答，并通过 SSE 心跳降低长时间调研阶段被代理层或浏览器误判断线的概率。

**现状问题**

- 前端聊天状态仅保存在 `zustand` 内存中，页面离开后本地占位消息和运行态会立即丢失。
- 后端 `ChatStream` 是单次请求生命周期内的直接事件转发，没有“恢复同一次运行”的接口和协议。
- 会话详情接口虽然能返回历史消息和步骤，但不会返回“当前活跃 run 摘要”，前端无法判断当前会话是否仍在执行。
- 长耗时阶段可能数十秒没有 chunk 输出，现有 SSE 没有心跳，连接容易被中间代理提前关闭。

**方案选择**

本次采用“运行快照 + 可恢复 SSE 订阅”方案，而不是完整事件溯源方案。

- 每次流式回答都绑定稳定 `run_id`
- 后端持久化当前运行快照：状态、阶段、待补充问题、最新答案快照、最近步骤
- 前端重新进入页面后，先通过会话详情拿到活跃 run 摘要，再使用恢复接口重新订阅该 `run_id`
- SSE 增加 `heartbeat` 事件，前端通过心跳超时判断是否需要自动重连

这样可以保证：

- 用户重新进入页面时能回到同一次回答，而不是重新发起一次新问题
- 断线期间的回答正文至少能恢复到最新快照
- 连接恢复后继续接收后续实时更新

不做的事情：

- 不实现按 `last_event_id` 的逐事件精确补放
- 不做跨浏览器标签页共享连接
- 不做 Service Worker 后台保活

## 后端设计

### 1. ConversationRun 扩展为运行快照

当前 `conversation_runs` 已有 `status`、`current_stage`、`pending_question`、`pending_context` 等字段。本次约定：

- `pending_context`
  - `running` 时存当前回答快照
  - `waiting_user` 时存 ADK checkpoint 等恢复上下文
  - `completed` 时存最终回答
- 新增明确的快照字段：
  - `last_answer`：当前累计答案快照
  - `heartbeat_at`：最近一次服务端心跳时间

如果不希望新增字段过多，也可以把 `last_answer` 继续复用到 `pending_context`，但这会和 ADK checkpoint 语义冲突。本次建议新增字段，避免恢复逻辑混杂。

### 2. Conversation detail 返回活跃 run 摘要

`GET /api/chat/conversations/:id` 增加：

- `active_run`
  - `id`
  - `status`
  - `current_stage`
  - `pending_question`
  - `last_answer`
  - `heartbeat_at`
  - `can_resume`
- `active_steps`
  - 仅返回当前活跃 run 对应的步骤，避免和历史 run 步骤混在一起

前端进入聊天页时据此决定：

- 只是展示历史消息
- 自动恢复一个仍在运行的回答
- 把 `waiting_user` 的补充问题重新展示出来

### 3. 新增恢复流接口

新增接口：

- `POST /api/ai/chat/stream/resume`

请求体：

- `conversation_id`
- `run_id`

行为：

- 校验 run 归属当前用户和当前会话
- 如果 run 已 `completed/failed`，直接推送一次快照和 `done/error`
- 如果 run 仍在 `running`，先推送 `resume` 起始事件和当前快照，再接入该 run 的实时事件广播
- 如果 run 为 `waiting_user`，推送快照和 `question` 事件

### 4. 引入运行级事件总线

为了让同一个运行可以被新连接重新订阅，需要在服务端增加一个短生命周期的运行中心，例如 `chatRunHub`：

- key 为 `run_id`
- 保存最近一次运行的：
  - 当前阶段
  - 最新答案快照
  - 最新步骤列表
  - 最后心跳时间
  - 若干 SSE 订阅者
- 新连接恢复时先拿快照，再订阅后续增量事件
- run 结束后延迟一小段时间清理，给页面切回留出窗口

这个 hub 只承担“运行期广播和恢复”，不是长期存储。

### 5. SSE 心跳

SSE 新增 `heartbeat` 事件，默认每 10 秒发送一次，payload 包含：

- `run_id`
- `ts`
- `stage`

服务端在以下情况下发送：

- 正常长任务期间定时发送
- 恢复连接建立后立即发送一次，帮助前端快速确认连接活着

### 6. 服务端快照更新点

在流式流程中更新 run 快照：

- 收到 `stage` 时更新 `current_stage`
- 收到 `chunk` 时更新 `last_answer`
- 收到 `question` 时更新 `pending_question`
- 完成时更新 `status=completed` 和最终答案
- 失败时更新 `status=failed` 和 `last_error`

## 前端设计

### 1. Chat Store 增加活跃运行态

为每个 conversation 增加：

- `activeRun`
  - `id`
  - `status`
  - `stage`
  - `lastAnswer`
  - `pendingQuestion`
  - `heartbeatAt`
  - `canResume`

全局增加：

- `reconnectAttempts`
- `lastHeartbeatAt`
- `isRecovering`

### 2. 页面恢复流程

当用户进入聊天页面或刷新后：

1. 拉取 conversation detail
2. 如果存在 `active_run.can_resume`
3. 先把 `last_answer` 和 `active_steps` 映射到当前 assistant 消息
4. 再自动调用 `resume stream`
5. 连接恢复后继续实时覆盖 assistant 占位消息

如果 run 已是 `waiting_user`：

- 不重开一个新 run
- 直接把补充问题和步骤状态恢复到 UI

### 3. 心跳超时与自动重连

前端维护心跳超时窗口，例如 25 秒：

- 收到任何 `chunk/stage/step/heartbeat` 都刷新 `lastHeartbeatAt`
- 超时后标记连接失活
- 对 `running` run 发起有限次自动重连
- 重连成功后显示“已恢复回答”
- 超过次数后显示“连接已断开，可手动继续恢复”

### 4. 新增 SSE 事件类型

前端新增处理：

- `resume`
- `snapshot`
- `heartbeat`

其中：

- `resume` 表示当前是恢复连接，不是一次新提问
- `snapshot` 用来同步 `last_answer + steps + status`
- `heartbeat` 仅用于保活和重连判断，不直接改正文

## 错误处理

- 恢复接口发现 run 不属于当前用户：返回 403
- run 不存在或已被清理：返回 404，前端退回普通历史态
- run 已完成：后端返回最终快照和 `done`
- 心跳超时但 detail 显示 run 已完成：前端直接刷新为已完成态，不继续重连
- 页面恢复时如果本地没有占位 assistant 消息：允许根据快照自动补一条 assistant 消息

## 测试策略

后端：

- handler 测试覆盖恢复接口的 `running/waiting_user/completed/not_found` 分支
- service 测试覆盖 run hub 订阅、快照广播、结束清理
- SSE 测试覆盖 `heartbeat`、`snapshot`、`resume` 事件序列

前端：

- store 测试或集成行为测试覆盖：
  - detail 返回 active run 后自动恢复
  - heartbeat 超时触发重连
  - waiting_user 恢复后不重复创建 assistant 占位消息
- 至少跑 `npm run lint` 和 `npm run build`

## 实施边界

第一版只保证：

- 同标签页刷新/关闭后重新进入能恢复当前运行
- 网络短断后前端能自动重连
- 长耗时阶段连接不因无输出而轻易断开

第一版不保证：

- 恢复后逐字重放断线期间所有 chunk
- 浏览器彻底关闭后后台继续保活同一 HTTP 连接
- 多端同时观看同一会话时的复杂协同状态
