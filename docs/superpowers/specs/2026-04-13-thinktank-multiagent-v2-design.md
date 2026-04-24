# ThinkTank 多 Agent AI 助手 V2 设计文档

## 1. 背景

问道博客当前 AI 助手已经具备以下基础能力：

- 基于 Ark / Doubao 的聊天模型接入
- 基于 Redis Vector 的文章向量检索
- 基于 `RAGChain` 的站内知识问答
- 基于 MySQL 的会话与消息持久化
- 基于 SSE 的流式聊天输出

当前实现的核心链路是“单次提问 -> 向量检索 -> 直接生成回答”，主要入口与关键实现位于：

- `backend/internal/service/ai.go`
- `backend/internal/pkg/eino/chain.go`
- `backend/internal/pkg/eino/retriever.go`
- `backend/internal/pkg/eino/vectorstore.go`
- `backend/internal/handler/ai.go`
- `frontend/src/store/chatStore.ts`
- `frontend/src/pages/AIChat.tsx`

该实现已经满足第一版 RAG 问答，但无法支撑更复杂的调研型问题，主要限制包括：

1. 只有单链路 RAG，没有多 Agent 任务拆解与协作。
2. 没有“先本地检索、再外部调研、再汇总审计”的完整闭环。
3. 没有在聊天过程中向用户追问并继续执行的状态能力。
4. 没有“调研结果待审核后再入库”的知识治理机制。
5. 前端只能处理单助手文本流，不能表达阶段性执行状态。

因此，本设计定义问道博客 AI 助手的第二版：**基于 Eino ADK + Ark 的多 Agent 协作 AI 助手**。该版本将直接替代现有单 RAG 产品形态，旧 RAG 不再作为最终产品入口，而是下沉为多 Agent 体系中的本地检索能力。

---

## 2. 目标与非目标

### 2.1 目标

本次 V2 设计目标如下：

1. 用多 Agent 协作架构替代当前单 RAG 链路。
2. 保留现有 AIChat 产品入口，但重构其执行内核。
3. 将当前 RAG 能力收编为 `Librarian` 子 Agent 的本地知识检索能力。
4. 增加 `Journalist` 外部联网调研能力，用于在本地知识不足时补充事实来源。
5. 支持在当前聊天会话中对用户进行追问，并在用户回答后继续执行原任务。
6. 前端只展示阶段，不展示 Agent 内部细节。
7. 详细执行摘要写入独立的 `*-ai-chat.log` 日志文件，与当前通用后端日志隔离。
8. 新增独立的“知识文档”领域模型；`Journalist` 产出的结构化调研结果先进入待审核状态，管理员审核通过后再进入向量库。
9. 审核通过后的知识文档进入全站共享知识库，作为后续 `Librarian` 的可检索数据源。

### 2.2 非目标

以下内容不属于本次第一阶段范围：

1. 不建设独立的“研究任务中心”产品。
2. 不在前端展示完整 Agent 内部推理链或逐条工具调用细节。
3. 不把联网调研结果直接自动入库。
4. 不把知识文档直接等同于博客文章草稿。
5. 不在首版引入复杂的多租户隔离知识库。
6. 不要求一次性实现通用工作流平台；首版重点是服务现有 AIChat 场景。

---

## 3. 用户确认后的关键产品决策

本设计基于以下已确认决策：

1. **产品入口**：保留当前 AIChat 入口，但重构执行内核。
2. **产品形态**：不保留旧版单 RAG，对外只保留多 Agent V2。
3. **前端展示方式**：只展示阶段，不展示具体 Agent 细节。
4. **日志策略**：详细协作过程写入独立 AI 聊天日志，文件名使用日期加 `ai-chat` 后缀，与现有日志隔离。
5. **调研范围**：首版同时支持本地知识检索和外部联网调研。
6. **知识沉淀策略**：`Journalist` 产出调研结果后，不直接入库；先生成待审核知识文档。
7. **知识存储方式**：新增独立知识表，不复用文章草稿，不落为单纯 Markdown 文件。
8. **追问交互方式**：在当前聊天里直接追问，用户回答后继续原任务。
9. **知识共享范围**：审核通过后的知识文档进入全站共享知识库。

---

## 4. 参考实现模式

本设计参考 Eino 官方多 Agent 能力与示例的设计方向，重点参考以下模式：

1. **Eino ADK 多 Agent 框架定位**
   - ADK 支持 Agent、Multi-Agent、工具调用、中断恢复等能力。
2. **Supervisor 模式**
   - 适合作为多个专家 Agent 的调度层。
3. **Plan-Execute / Plan-Execute-Replan 模式**
   - 适合作为复杂调研任务的规划与重规划骨架。
4. **Ark 模型接入**
   - 继续使用 Ark 作为问答与 Agent 推理模型供应方。

在本项目中，首版不追求照搬示例目录结构，而是吸收其架构思想，并落到当前既有的 Gin + Service + Repository + Eino 封装体系中。

参考资料：

- `https://github.com/cloudwego/eino-examples/tree/main/adk/multiagent`
- `https://github.com/cloudwego/eino-examples/tree/main/adk/multiagent/supervisor`
- `https://github.com/cloudwego/eino-examples/tree/main/adk/multiagent/plan-execute-replan`
- `https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_preview/`
- `https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/supervisor/`
- `https://cloudwego.cn/docs/eino/core_modules/eino_adk/agent_implementation/plan_execute/`
- `https://www.cloudwego.io/docs/eino/ecosystem_integration/chat_model/agentic_model_ark/`

---

## 5. 总体架构

### 5.1 总体思路

保留现有 AIChat 的用户入口与会话模型，但将原本由 `AIService -> RAGChain` 驱动的单链路执行方式，替换为一个新的 **ThinkTank 多 Agent 编排层**。

整体架构如下：

```text
AIChat Page
   ↓
/api/ai/chat / /api/ai/chat/stream
   ↓
AIHandler
   ↓
ThinkTankService
   ├── Planner
   ├── Supervisor
   ├── LibrarianAgent
   │    └── 复用现有 RAG / Retriever / VectorStore
   ├── JournalistAgent
   │    └── 外部搜索与调研
   ├── Synthesizer / Replanner
   ├── AskUserStateManager
   ├── StageEventEmitter
   ├── AILogger
   └── KnowledgeDraftService
   ↓
Conversation / ChatMessage / KnowledgeDocument / VectorIndex
```

### 5.2 核心原则

1. **对外保持入口稳定**：仍通过当前 AIChat 与既有 API 访问。
2. **对内彻底更换内核**：旧 RAG 不再是总入口，而是 `Librarian` 子能力。
3. **职责明确拆分**：规划、调度、检索、联网调研、汇总、追问、日志、知识沉淀都应独立建模。
4. **尽量复用现有基础设施**：会话、消息、向量检索、Ark 模型、SSE 流式输出继续沿用。
5. **阶段可见、细节隐藏**：前端只看阶段，详细执行摘要写日志。
6. **知识治理优先**：联网调研结果必须先经过审核，才能进入共享知识库。

---

## 6. Agent 角色定义

### 6.1 Planner

职责：

1. 理解用户问题。
2. 判断问题是否缺少关键约束。
3. 产出高层任务计划。
4. 决定优先走本地知识、联网调研，还是先追问。

输入：

- 用户当前消息
- 当前会话上下文（必要时）
- 当前是否存在待补充问题状态

输出：

- `plan_summary`
- `requires_clarification`
- `clarification_question`
- `execution_strategy`（`local_only` / `local_then_web` / `web_first`）

说明：

- 首版重点是输出结构化计划摘要，不暴露长推理链。
- 若问题范围过大或存在关键歧义，Planner 可以触发追问。

### 6.2 Supervisor

职责：

1. 接收 Planner 的高层计划。
2. 按顺序调度 `Librarian`、`Journalist`、`Synthesizer`。
3. 汇总阶段结果，控制阶段切换。
4. 把执行摘要写入 AI 专用日志。

输入：

- Planner 输出
- 当前会话状态
- 已完成的 Agent 产物

输出：

- 当前阶段事件
- 子 Agent 调度命令
- 最终汇总上下文

说明：

- 首版 Supervisor 负责串行执行与必要的补轮，不要求实现通用并行 Agent 编排。

### 6.3 LibrarianAgent

职责：

1. 检索站内和共享知识库中的相关内容。
2. 基于检索结果判断本地知识是否足够支撑回答。
3. 输出本地证据摘要与缺口分析。

输入：

- 用户问题
- 可选范围限制（未来可扩展）

输出：

- `local_findings`
- `local_sources`
- `coverage_status`（`sufficient` / `partial` / `insufficient`）
- `followup_hint`

实现策略：

- 复用现有 `RedisRetriever`、`RedisVectorStore`、`Embedder`、`RAG` 提示构造思路。
- 但不再让 `RAGChain` 直接决定最终回答，而是让它服务于本地知识整理。
- 首版可通过“检索 + 局部总结”实现，不强依赖完整重写成通用 Agent Tool。

### 6.4 JournalistAgent

职责：

1. 在本地知识不足时进行联网调研。
2. 对外部来源进行筛选、整理、归纳。
3. 产出适合回答用户的问题摘要。
4. 在需要沉淀时生成候选知识文档草稿。

输入：

- 用户问题
- Librarian 输出的知识缺口
- Planner/Supervisor 指令

输出：

- `web_findings`
- `web_sources`
- `research_summary`
- `knowledge_draft_candidate`

约束：

- 首版只记录来源摘要，不在前端暴露完整搜索过程。
- 不直接自动入库。
- 应限制搜索轮次、来源数量、超时时间，以避免无限扩张。

### 6.5 Synthesizer / Replanner

职责：

1. 汇总 `Librarian` 与 `Journalist` 的结果。
2. 判断结果是否已经覆盖用户问题。
3. 生成最终回答。
4. 若覆盖不足，可请求 Supervisor 再进行一轮补充执行。

输入：

- 本地知识结果
- 联网调研结果
- 用户原始问题

输出：

- `final_answer`
- `final_sources`
- `coverage_review`
- `needs_replan`

说明：

- 首版重规划只允许小范围补轮，避免流程无限循环。

---

## 7. 执行流程设计

### 7.1 标准执行流

```text
用户提问
  ↓
Planner 解析问题
  ↓
是否需要追问？
  ├── 是 -> 写入等待追问状态 -> 在当前聊天返回问题 -> 等待用户回复
  └── 否 -> Supervisor 开始执行
              ↓
            Librarian 检索本地知识
              ↓
            本地知识是否充足？
              ├── 是 -> 交给 Synthesizer 汇总
              └── 否 -> Journalist 联网调研
                           ↓
                         Synthesizer 汇总与审计
                           ↓
                         是否需要补轮？
                           ├── 是 -> 重新分发局部任务
                           └── 否 -> 输出最终回答
                                      ↓
                              生成候选知识文档（可选）
                                      ↓
                              管理员审核通过后入向量库
```

### 7.2 追问与继续执行

当 Planner 判断用户问题缺少关键约束时：

1. 不直接进入正式执行。
2. 生成一个澄清问题。
3. 将当前会话标记为“等待用户补充”。
4. 前端在当前聊天窗口展示一条助手追问。
5. 用户回复后，系统识别该会话存在待补充状态。
6. 将用户回复与原问题组合，继续执行未完成流程。

关键点：

- 用户无需新建会话。
- 不从头丢弃旧流程。
- 追问状态必须可以从数据库恢复。

### 7.3 阶段事件流

前端只关心阶段，不关心内部实现细节。建议定义稳定的阶段枚举：

- `analyzing`
- `clarifying`
- `local_search`
- `web_research`
- `synthesizing`
- `completed`
- `failed`

并为每个阶段定义统一中文描述，例如：

- `analyzing`：正在理解你的问题
- `clarifying`：需要你补充一点信息
- `local_search`：正在检索站内知识
- `web_research`：正在进行外部调研
- `synthesizing`：正在整理最终结论
- `completed`：回答已生成
- `failed`：本次执行失败

---

## 8. 后端模块设计

### 8.1 目录与职责建议

建议新增或重构以下模块：

#### `backend/internal/service/thinktank.go`

作为新的多 Agent 统一编排入口，负责：

- 接收聊天请求
- 校验会话归属
- 驱动 Planner / Supervisor / Synthesizer
- 保存消息与会话状态
- 输出阶段事件
- 写日志
- 触发知识草稿生成

#### `backend/internal/service/thinktank_planner.go`

负责：

- 问题解析
- 追问判断
- 高层计划摘要生成

#### `backend/internal/service/thinktank_supervisor.go`

负责：

- 子 Agent 调度
- 阶段切换
- 执行结果收集

#### `backend/internal/service/thinktank_librarian.go`

负责：

- 封装本地知识检索能力
- 复用向量检索、相似度判断、RAG 提示整理能力

#### `backend/internal/service/thinktank_journalist.go`

负责：

- 封装外部调研能力
- 统一外部来源摘要结构
- 生成知识草稿候选

#### `backend/internal/service/thinktank_synthesizer.go`

负责：

- 汇总本地与外部来源
- 生成最终回答
- 判断是否需要补轮

#### `backend/internal/service/knowledge_document.go`

负责：

- 候选知识文档的创建、审核、拒绝、入库
- 审核通过后触发向量化

#### `backend/internal/service/ai_log.go`

负责：

- AI 专用日志落盘
- 结构化记录阶段、Agent 摘要、错误信息、来源信息

### 8.2 现有模块的调整策略

#### `backend/internal/service/ai.go`

调整方向：

- 保留 `GenerateSummary` 能力。
- 原 `Chat` / `ChatStream` 逐步迁移到 ThinkTankService。
- 最终 `AIService` 可以退化为更薄的一层，或者被 ThinkTankService 替换。

#### `backend/internal/pkg/eino/chain.go`

调整方向：

- 不再代表整个 AI 助手入口。
- 保留为本地知识检索与站内回答提示构造能力，供 `Librarian` 使用。
- 后续可以拆分为：
  - 检索与评分
  - 本地知识摘要生成
  - 站内问答提示模板

#### `backend/internal/handler/ai.go`

调整方向：

- 保持 `/api/ai/chat` 与 `/api/ai/chat/stream` 路由稳定。
- 扩展 SSE 事件类型，支持阶段事件与追问事件。
- `ChatResponse` 可逐步扩展为包含：
  - `message`
  - `stage`
  - `requires_user_input`
  - `question_type`
  - `sources`

---

## 9. 数据模型设计

### 9.1 现有模型继续保留

#### `conversations`

继续作为聊天会话主表。

#### `chat_messages`

继续作为用户与助手消息记录表。

### 9.2 新增模型一：会话执行状态

建议新增：`conversation_runs` 或 `conversation_agent_states`

用于存储某次会话当前执行状态，建议字段如下：

| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint | 主键 |
| conversation_id | bigint | 对应会话 |
| user_id | bigint | 发起用户 |
| status | varchar | `running` / `waiting_user` / `completed` / `failed` |
| current_stage | varchar | 当前阶段 |
| original_question | text | 原始问题 |
| normalized_question | text | 归一化问题 |
| pending_question | text | 待用户回答的追问 |
| pending_context | json | 等待继续执行的上下文 |
| last_plan | json | 最近一次高层计划摘要 |
| last_error | text | 最近一次错误信息 |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |
|
说明：

- 首版不一定要完整保存每一步 Agent 内部状态，但必须能支持“追问后继续”。
- `pending_context` 是实现“会话内追问恢复”的关键字段。

### 9.3 新增模型二：知识文档

建议新增：`knowledge_documents`

字段建议：

| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint | 主键 |
| title | varchar(255) | 文档标题 |
| summary | text | 摘要 |
| content | longtext | 正文 |
| status | varchar(32) | `pending_review` / `approved` / `rejected` |
| source_type | varchar(32) | `research` / `manual` |
| created_by_user_id | bigint | 创建人 |
| reviewed_by_user_id | bigint | 审核人 |
| reviewed_at | datetime | 审核时间 |
| review_note | text | 审核备注 |
| vectorized_at | datetime | 入向量库时间 |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

说明：

- `content` 存放结构化调研结果正文。
- 首版 `source_type` 可以先支持 `research` 与 `manual`，为后续后台手工录入预留空间。
- 只有 `approved` 的知识文档才允许进入向量库。

### 9.4 新增模型三：知识来源

建议新增：`knowledge_document_sources`

字段建议：

| 字段 | 类型 | 说明 |
|---|---|---|
| id | bigint | 主键 |
| knowledge_document_id | bigint | 所属知识文档 |
| source_url | text | 来源链接 |
| source_title | varchar(500) | 来源标题 |
| source_domain | varchar(255) | 来源域名 |
| source_snippet | text | 来源摘要 |
| sort_order | int | 排序 |
| created_at | datetime | 创建时间 |

说明：

- 用于保留联网调研的来源清单。
- 有助于后台审核与后续回答溯源。

### 9.5 新增模型四：知识向量元数据扩展

当前向量库主要索引文章分块。为支持全站共享知识库，建议统一向量元数据结构，新增或规范如下字段：

- `source_kind`：`article` / `knowledge_document`
- `source_id`：来源对象 ID
- `title`
- `content`
- `chunk_index`
- `status`

这样 `Librarian` 在检索时即可识别命中的内容来自文章还是知识文档。

---

## 10. API 设计

### 10.1 保留现有聊天入口

继续保留：

- `POST /api/ai/chat`
- `POST /api/ai/chat/stream`

原因：

- 前端入口稳定。
- 便于在不修改产品路径的情况下完成 V2 升级。

### 10.2 流式事件扩展

当前 SSE 事件主要是：

- `start`
- `chunk`
- `done`
- `error`

建议扩展为：

- `start`
- `stage`
- `question`
- `chunk`
- `done`
- `error`

说明：

#### `stage`

用于广播当前阶段，例如：

```json
{
  "stage": "local_search",
  "label": "正在检索站内知识"
}
```

#### `question`

用于表示当前执行被追问中断：

```json
{
  "stage": "clarifying",
  "message": "你更关注国内案例还是海外案例？",
  "requires_user_input": true
}
```

#### `chunk`

继续用于输出最终回答的累计文本快照。

### 10.3 后台管理接口新增

建议新增知识文档审核相关接口：

- `GET /api/admin/knowledge-documents`
- `GET /api/admin/knowledge-documents/:id`
- `POST /api/admin/knowledge-documents/:id/approve`
- `POST /api/admin/knowledge-documents/:id/reject`

说明：

- 管理员审核通过后，服务端触发向量化。
- 审核拒绝则保留文档与来源，不进入向量库。

---

## 11. 前端设计

### 11.1 总体原则

前端继续以当前 AIChat 页为入口，但升级聊天状态管理，使其能处理阶段事件与追问事件。

重点原则：

1. 仍然是一条聊天流。
2. 用户不直接看到 Agent 内部细节。
3. 用户只感知阶段切换和最终回答。
4. 如果系统需要追问，则把追问当作当前会话中的一条助手消息。

### 11.2 Store 改造方向

当前 `frontend/src/store/chatStore.ts` 主要维护：

- 对话列表
- 活跃对话
- 是否流式输出
- 最终助手消息快照

V2 建议增加：

- `currentStage`
- `currentStageLabel`
- `requiresUserInput`
- `pendingQuestion`
- `runStatus`

这样可以让 UI 在回答未完成前展示阶段状态。

### 11.3 UI 交互

建议在当前 AIChat 页面增加一个轻量阶段条或阶段提示区，用于显示：

- 正在理解你的问题
- 正在检索站内知识
- 正在进行外部调研
- 正在整理最终结论

如果进入追问状态：

- 当前输入框不禁用。
- 助手消息区显示追问文本。
- 用户继续输入后，视为“补充条件”而不是新的无关聊天。

### 11.4 管理后台新增页面

建议新增知识文档管理页：

- 列表页：待审核 / 已通过 / 已拒绝
- 详情页：查看正文、来源链接、调研摘要、审核操作

首版不要求复杂编辑器，优先支持：

- 浏览
- 审核通过
- 审核拒绝
- 查看来源

---

## 12. 日志设计

### 12.1 日志文件

新增独立 AI 聊天日志文件，与当前后端通用日志隔离，命名规则如下：

```text
backend/log/YYYY-MM-DD-ai-chat.log
```

示例：

```text
backend/log/2026-04-13-ai-chat.log
```

### 12.2 日志记录内容

日志中记录结构化执行摘要，至少包括：

- 会话 ID
- 用户 ID
- 当前 Run ID
- 阶段切换
- Planner 计划摘要
- 是否触发追问
- Librarian 命中文档数量、命中类型、覆盖判断
- Journalist 调研摘要、来源数量、来源域名
- Synthesizer 汇总结论摘要
- 是否生成候选知识文档
- 审核触发结果
- 错误信息与失败阶段

### 12.3 不记录的内容

为了避免日志过深和难以维护：

1. 不记录完整 chain-of-thought。
2. 不记录原始超长 Prompt 全文。
3. 不记录所有外部抓取文本全文。
4. 不在日志里重复保存整篇知识文档正文。

推荐保留“**执行摘要 + 关键决策 + 统计信息 + 错误信息**”即可。

---

## 13. 向量检索与知识库融合设计

### 13.1 现状

当前向量库只包含文章分块，来源主要来自已发布文章。

### 13.2 目标

V2 需要让 `Librarian` 同时可检索：

1. 已发布博客文章分块。
2. 审核通过的知识文档分块。

### 13.3 实现策略

统一向量索引结构，不拆两个完全独立的检索入口，而是在一个统一的索引或统一检索抽象中通过 `source_kind` 区分内容类型。

优点：

- 检索逻辑一致。
- 召回结果更容易统一排序。
- Librarian 不必同时维护两套完全不同的检索管线。

### 13.4 审核通过后的入库流程

1. 管理员审核知识文档。
2. 系统将文档正文切块。
3. 调用当前向量化流程生成 embedding。
4. 将块写入 Redis Vector，并带上 `source_kind=knowledge_document` 等元数据。
5. 记录 `vectorized_at`。

---

## 14. 错误处理设计

### 14.1 业务级错误与系统级错误区分

#### 业务级条件

以下情况不应直接判为系统失败：

- 本地知识命中不足
- 联网来源较少但仍可回答
- 用户问题需要补充条件
- 候选知识文档未生成

这些应转为：

- 阶段切换
- 追问
- 降级回答
- 跳过知识沉淀

#### 系统级失败

以下应视为真实错误：

- Ark 模型调用失败
- Redis 向量检索失败
- 外部调研工具异常并且无法恢复
- 会话状态保存失败
- SSE 输出中断且无法继续

### 14.2 降级策略

1. **Librarian 检索不足**：自动切到 Journalist。
2. **Journalist 联网失败**：若本地已有部分信息，则生成“部分回答 + 明确说明缺口”。
3. **知识草稿生成失败**：不影响本次回答，记录日志即可。
4. **审核后向量化失败**：保留为 `approved` 但标记未完成向量化，并允许重试。

---

## 15. 安全与治理

### 15.1 知识治理

- 联网调研结果不得自动直接进入共享知识库。
- 所有研究文档默认 `pending_review`。
- 只有管理员审核通过后才允许向量化与共享检索。

### 15.2 输出边界

- 前端不展示内部推理细节。
- 日志只记录摘要，不记录深层推理全文。
- 回答中应尽量区分“本地知识”与“联网补充”来源边界。

### 15.3 来源保留

- 联网调研产物必须保留来源清单，以便审核与后续追溯。
- 审核后台应允许查看来源链接与来源摘要。

---

## 16. 测试设计

### 16.1 后端测试重点

1. Planner 对“是否追问”的判断流程。
2. 会话处于 `waiting_user` 时的恢复执行流程。
3. Librarian 在文章与知识文档混合检索时的返回结果。
4. 本地知识不足时自动切换 Journalist。
5. Synthesizer 在本地 + 外部结果混合下的最终答案生成。
6. 知识草稿生成与待审核保存。
7. 审核通过后触发向量化。
8. AI 专用日志文件输出与隔离。
9. SSE 阶段事件输出正确性。

### 16.2 前端测试重点

1. 阶段事件渲染正确。
2. `question` 事件出现时能提示用户补充输入。
3. 聊天完成后仍能像现在一样在会话详情中恢复完整消息。
4. 流式输出与阶段事件并存时不会打乱 UI 状态。

---

## 17. 分阶段实施建议

### 阶段 1：替换执行内核

目标：

- 引入 ThinkTankService
- 保留旧 API 入口
- 跑通 Planner -> Librarian -> Journalist -> Synthesizer 基本链路
- 前端能看到阶段事件

### 阶段 2：支持追问恢复

目标：

- 增加会话执行状态表
- 支持 `waiting_user` 状态
- 用户在当前聊天回复后恢复执行

### 阶段 3：知识草稿与审核流

目标：

- 新增 `knowledge_documents`
- 新增后台审核 API 与页面
- 审核通过后触发向量化

### 阶段 4：共享知识库融合

目标：

- Librarian 能同时检索文章与审核通过的知识文档
- 优化来源区分与检索排序

---

## 18. 推荐文件改动范围

### 后端

建议新增：

- `backend/internal/model/knowledge_document.go`
- `backend/internal/model/knowledge_document_source.go`
- `backend/internal/model/conversation_run.go`
- `backend/internal/repository/knowledge_document.go`
- `backend/internal/repository/conversation_run.go`
- `backend/internal/service/thinktank.go`
- `backend/internal/service/thinktank_planner.go`
- `backend/internal/service/thinktank_supervisor.go`
- `backend/internal/service/thinktank_librarian.go`
- `backend/internal/service/thinktank_journalist.go`
- `backend/internal/service/thinktank_synthesizer.go`
- `backend/internal/service/knowledge_document.go`
- `backend/internal/service/ai_log.go`
- `backend/internal/handler/knowledge_document.go`

建议修改：

- `backend/internal/service/ai.go`
- `backend/internal/handler/ai.go`
- `backend/internal/service/vector.go`
- `backend/internal/pkg/eino/chain.go`
- `backend/internal/pkg/eino/retriever.go`
- `backend/cmd/server/main.go`
- `backend/config/config.go`
- `backend/config/config.yaml`

### 前端

建议修改：

- `frontend/src/store/chatStore.ts`
- `frontend/src/pages/AIChat.tsx`
- `frontend/src/api/chat.ts`
- `frontend/src/types/index.ts`
- `frontend/src/router.tsx`

建议新增：

- `frontend/src/views/admin/knowledge-documents/KnowledgeDocumentList.tsx`
- `frontend/src/views/admin/knowledge-documents/KnowledgeDocumentDetail.tsx`
- `frontend/src/api/knowledgeDocument.ts`

---

## 19. 风险与应对

### 风险 1：多 Agent 复杂度显著上升

应对：

- 采用 Supervisor 串行调度为主。
- 首版不做并行 Agent 扩展。
- 严格控制角色与职责边界。

### 风险 2：追问恢复状态易失真

应对：

- 将待追问上下文持久化到数据库。
- 不依赖前端临时状态保存。

### 风险 3：联网调研结果质量不稳定

应对：

- 首版保留来源列表。
- 只有审核通过后才允许入库。
- 回答阶段明确区分站内知识与外部补充。

### 风险 4：日志膨胀

应对：

- 只记录结构化摘要，不记录完整 Prompt 和全文抓取结果。
- 与主日志隔离到独立 `ai-chat` 文件。

### 风险 5：向量库混合来源后检索质量波动

应对：

- 在元数据中明确 `source_kind`。
- 后续为文章与知识文档设计不同的排序加权策略。

---

## 20. 验收标准

当以下条件成立时，可认为 V2 第一阶段设计实现成功：

1. 用户仍可通过当前 AIChat 页面进行提问。
2. 原单 RAG 产品路径已被多 Agent 编排流程替代。
3. 用户能看到阶段性提示，而不是内部 Agent 细节。
4. 系统在问题不清晰时能在当前聊天中追问，并在用户回答后继续执行。
5. 系统能先做本地知识检索，再在必要时进行联网调研。
6. 最终回答由统一汇总层生成，而不是单次 RAG 直接输出。
7. 详细执行摘要写入独立 `backend/log/YYYY-MM-DD-ai-chat.log`。
8. 系统能生成待审核知识文档。
9. 管理员审核通过后，知识文档能进入向量库并成为全站共享知识。

---

## 21. 结论

本设计选择的不是“在旧 RAG 上继续叠功能”，而是一次明确的架构升级：

- 保留用户入口
- 替换执行内核
- 将旧 RAG 收编为 `Librarian` 能力
- 引入 `Journalist` 外部调研
- 通过 `Planner + Supervisor + Synthesizer` 建立完整多 Agent 协作闭环
- 通过追问恢复、阶段事件、独立日志和审核入库机制，使问道博客的 AI 助手从“单次站内问答”升级为“可调研、可沉淀、可治理的多 Agent 助手”

这条路径能够在最大程度复用现有代码基础的前提下，为后续扩展更强的研究能力、知识积累能力和后台治理能力打下稳定结构。
