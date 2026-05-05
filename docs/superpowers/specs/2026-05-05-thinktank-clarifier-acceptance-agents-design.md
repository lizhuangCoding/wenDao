# ThinkTank 澄清维度与验收标准 Agent 设计

**目标**

在现有 ThinkTank 多 Agent 协作流程中新增两个显式 Agent：

- `ClarifierAgent`：理解用户真实意图，识别用户真正想了解的维度，决定是否需要追问。
- `AcceptanceAgent`：生成回答验收标准，并判断最终回答是否真正覆盖用户想了解的内容。

本次设计优先使用字节跳动开源的 Eino ADK 能力，不手写独立 Agent 框架。现有 `planner -> executor -> replanner` 链路继续保留，新增 Agent 作为 Eino ADK 层的前置意图网关和后置验收网关。

**现状**

当前后端已经在 `backend/internal/service/chat/thinktank_adk.go` 使用 Eino ADK 的 `planexecute` 预置流程：

```text
planner -> executor -> replanner
```

其中：

- `planner` 负责生成计划。
- `executor` 负责执行当前计划步骤，工具包括 `LocalSearch`、`WebSearch`、`WebFetch`、`DocWriter`、`ask_for_clarification`。
- `replanner` 负责判断是否完成，或继续生成剩余步骤。

现有问题：

- 澄清能力主要依赖 `ask_for_clarification` 工具，只有执行中发现缺信息时才触发，缺少前置“意图画像”和“维度识别”。
- 验收能力隐含在 `replanner` prompt 中，没有独立的验收标准，也没有明确的“回答是否满足用户真正意图”的结构化判断。
- 如果回答偏离用户真实需求，当前闭环更多依赖 replanner 继续规划，缺少受控的质量返工策略。

**方案选择**

采用“Eino ChatModelAgent + 现有 PlanExecute + 确定性验收网关”的方案。

```text
用户问题
  -> ClarifierAgent
      -> 关键缺失：中断并追问用户
      -> 普通宽泛：自动推断回答维度
  -> Eino PlanExecute
      -> planner
      -> executor
      -> replanner
  -> AcceptanceAgent
      -> pass：返回最终答案
      -> revise：带审核意见最多再执行 1 轮
      -> ask_user：中断并追问用户
  -> 返回最终答案或带限制说明的最佳答案
```

这个方案的边界是：Agent 做认知判断，服务层控制流程状态、循环次数、中断恢复和持久化。这样既使用 Eino ADK 的 Agent 能力，又避免让模型自由决定无限循环。

不采用的方案：

- 不把现有流程整体切换成 Eino Supervisor。当前 `planexecute` 链路已经稳定，直接改成 Supervisor 会改变执行语义，并重新引入 `transfer_to_agent` 调度不确定性。
- 不继续只强化 `ask_for_clarification` 工具。工具级追问无法承担完整的用户意图识别、维度归纳和验收标准生成。
- 不手写新的 Agent runtime。流程编排、Agent 运行、checkpoint、interrupt/resume 继续复用 Eino ADK。

**ClarifierAgent 设计**

职责：

1. 识别用户真实意图。
2. 判断用户想了解的维度。
3. 将原始问题归一化为更适合下游 Agent 执行的问题。
4. 判断是否需要追问用户。
5. 生成追问问题时只问一个关键问题，避免表单式盘问。

输入：

- 用户当前问题。
- 当前会话压缩记忆。
- 最近若干轮原始对话。
- 如果是从 `waiting_user` 恢复，则包括上一次追问和用户补充。

输出结构：

```json
{
  "normalized_question": "归一化后的问题",
  "intent": "用户真正想解决的问题",
  "answer_goal": "explain|research|compare|decision|tutorial|debug|write",
  "target_dimensions": ["用户关心的维度"],
  "constraints": {
    "time_range": "",
    "audience": "",
    "depth": "",
    "style": "",
    "source_policy": ""
  },
  "ambiguity_level": "low|medium|high",
  "should_ask_user": false,
  "clarification_question": "",
  "reason": "简短说明，不暴露长推理链"
}
```

追问原则：

- 如果缺失信息会改变“该回答什么”，就追问。
- 如果缺失信息只影响“回答得多细”，就自动推断默认维度继续回答。

关键缺失示例：

```text
用户：帮我调研一下这个项目
```

缺失对象，“这个项目”无法确定。需要追问：

```text
你想调研哪个项目？请提供项目名称、链接，或简单描述。
```

```text
用户：帮我看看这个报错怎么修
```

缺少报错内容、触发操作、相关代码和运行环境。需要追问：

```text
请把完整报错信息、触发操作和相关代码片段发我，我才能定位问题。
```

普通宽泛示例：

```text
用户：帮我分析一下 AI Agent 的发展趋势
```

对象明确，只是范围较宽。无需追问，自动推断维度：

```json
{
  "intent": "了解 AI Agent 的发展趋势",
  "target_dimensions": ["技术演进", "产品形态", "商业落地", "风险限制", "未来判断"],
  "should_ask_user": false
}
```

```text
用户：帮我调研一下李小龙
```

对象明确，默认按人物调研维度执行：

```text
生平背景、核心成就、武术思想、影视影响、争议与评价、参考来源
```

**AcceptanceAgent 设计**

职责：

1. 根据 ClarifierAgent 的意图画像生成回答验收标准。
2. 检查最终答案是否覆盖用户目标维度。
3. 检查答案是否答偏、遗漏关键维度、证据不足或输出形态不符合用户目的。
4. 给出结构化 verdict，供服务层决定返回、返工或追问。

输入：

- 原始用户问题。
- ClarifierAgent 输出的意图画像。
- PlanExecute 的最终答案。
- 可选的本地文章来源和外部来源摘要。
- 当前已返工次数。

输出结构：

```json
{
  "verdict": "pass|revise|ask_user",
  "score": 86,
  "matched_dimensions": ["已覆盖的维度"],
  "missing_dimensions": ["缺失的维度"],
  "unsupported_claims": ["证据不足的判断"],
  "format_issues": ["输出形态问题"],
  "revision_instruction": "下一轮需要补充或修正的内容",
  "user_question": "",
  "reason": "简短说明"
}
```

验收标准示例：

用户问题：

```text
帮我分析一下 AI Agent 的发展趋势
```

ClarifierAgent 推断维度：

```text
技术演进、产品形态、商业落地、风险限制、未来判断
```

AcceptanceAgent 验收时应判断：

- 是否说明了 Agent 从工具调用、工作流到多 Agent 协作的技术演进。
- 是否覆盖了产品形态，例如 Copilot、自动化助手、研究型 Agent、企业工作流 Agent。
- 是否讨论了商业落地条件，而不是只讲概念。
- 是否指出可靠性、成本、权限、安全和评估难点。
- 是否给出明确趋势判断，而不是只做资料罗列。

**闭环策略**

采用受控闭环，避免无限循环：

```text
首次答案 -> AcceptanceAgent
  pass：返回最终答案
  revise：带 revision_instruction 最多再执行 1 轮 PlanExecute
  ask_user：进入 waiting_user，等待用户补充

二次答案 -> AcceptanceAgent
  pass：返回最终答案
  revise/ask_user：
    - 如果仍缺关键用户信息，追问用户
    - 否则返回当前最佳答案，并在答案末尾说明限制
```

默认最大返工次数为 1。原因：

- 多 Agent + 联网调研本身成本和耗时较高。
- 返工过多容易让模型自我修辞，而不是实质改善。
- 对用户来说，明确说明限制通常比长时间后台循环更可控。

可以在配置中保留扩展点：

```yaml
ai:
  thinktank_max_review_revisions: 1
  thinktank_clarifier_enabled: true
  thinktank_acceptance_enabled: true
```

第一版也可以不新增配置，直接默认开启，后续再根据线上效果决定是否可配置。

**Eino 接入设计**

新增两个 Eino `ChatModelAgent`：

- `clarifier`
- `acceptance`

它们都使用现有 Ark tool-calling model 对应的 Eino model。Clarifier 和 Acceptance 首版不需要工具调用，只需要稳定 JSON 输出；如果后续需要更强约束，可将其封装为 `tool calling` 风格，让模型必须调用固定 schema 的工具返回结构。

建议新增内部构造函数：

```go
newClarifierAgent(ctx context.Context, model componentmodel.ToolCallingChatModel) (adk.Agent, error)
newAcceptanceAgent(ctx context.Context, model componentmodel.ToolCallingChatModel) (adk.Agent, error)
```

`thinkTankADKRunner` 扩展为：

```go
type thinkTankADKRunner struct {
    runner          *adk.Runner
    agent           adk.ResumableAgent
    checkpointStore *thinkTankCheckpointStore

    clarifier       adk.Agent
    acceptance      adk.Agent
}
```

执行方式：

- ClarifierAgent 可通过独立 runner 调用，或封装为 `RunClarifier` 方法直接执行。
- PlanExecute 继续使用现有 `runner.Run` 和 `runner.Resume`。
- AcceptanceAgent 在拿到最终答案后执行。

是否使用 Eino `SequentialAgent`：

- 第一版不建议把完整链路包装成一个大型 `SequentialAgent`，因为 PlanExecute 已经有自己的 checkpoint/resume 和 interrupt 语义。
- 更稳妥的做法是在服务层确定性调用 ClarifierAgent、PlanExecute、AcceptanceAgent，底层 Agent 仍使用 Eino ADK。
- 后续如果 Eino ADK workflow 组合需求明确，再把它升级为 `SequentialAgent{clarifier, plan_execute_replan, acceptance}`。

**数据流**

### 非流式 Chat

```text
Chat
  -> 加载会话历史和记忆
  -> ClarifierAgent
      -> should_ask_user=true：返回 RequiresUserInput
      -> should_ask_user=false：生成 normalized_question
  -> PlanExecute 使用 enhanced query
  -> AcceptanceAgent
      -> pass：保存并返回
      -> revise：追加审核意见再跑一轮 PlanExecute
      -> ask_user：保存 pending question
  -> 持久化最终回答和记忆
```

### 流式 ChatStream

```text
ChatStream
  -> stage: clarifying_intent
  -> step: Clarifier
  -> 如果需追问：
       status=waiting_user
       event: question
       保存 ADK checkpoint 或 pending_context
  -> stage: planning/executing
  -> 现有 ADK streamADKFlow
  -> stage: reviewing
  -> step: Acceptance
  -> 如果需返工：
       stage: revising
       带 revision_instruction 再跑一轮
  -> event: chunk/done
```

前端阶段建议：

| stage | 文案 |
|---|---|
| `clarifying_intent` | 正在理解你的真实意图 |
| `planning` | 正在规划回答路径 |
| `executing` | 正在检索和调研 |
| `reviewing` | 正在验收回答质量 |
| `revising` | 正在根据验收意见补充 |
| `clarifying` | 需要补充一点信息 |
| `completed` | 回答已生成 |

**Prompt 设计要点**

ClarifierAgent system prompt 要强调：

- 不要为了追问而追问。
- 优先自动推断常见维度。
- 只有关键缺失才设置 `should_ask_user=true`。
- 输出必须是 JSON。
- 不暴露长推理链。

AcceptanceAgent system prompt 要强调：

- 验收目标是“用户真正想知道的内容”，不是语法润色。
- 如果答案基本满足，只给 `pass`，不要过度吹毛求疵。
- 只有遗漏关键维度或答偏时才给 `revise`。
- 只有缺少用户信息导致无法验收或无法继续时才给 `ask_user`。
- 输出必须是 JSON。

**持久化设计**

第一版尽量复用现有模型：

- `conversation_runs.normalized_question`：保存 ClarifierAgent 的 `normalized_question`。
- `conversation_runs.last_plan`：保存 ClarifierAgent 意图画像、验收标准和最终 verdict 的精简 JSON。
- `conversation_runs.pending_question`：保存 ClarifierAgent 或 AcceptanceAgent 触发的追问。
- `conversation_runs.pending_context`：保存 Eino checkpoint、追问来源、已返工次数。
- `conversation_run_steps`：新增步骤记录：
  - `AgentName=clarifier`
  - `AgentName=acceptance`

不新增数据库表。原因是当前已有 `conversation_runs` 和 `conversation_run_steps` 能承载运行状态和过程日志，新增表会增加迁移成本，但第一版收益有限。

**错误处理**

- ClarifierAgent 失败：降级为旧流程，使用原始问题继续执行，并记录 warning step。
- ClarifierAgent 输出 JSON 解析失败：尝试提取 JSON；仍失败则降级。
- AcceptanceAgent 失败：返回 PlanExecute 最终答案，不阻断用户获得结果。
- AcceptanceAgent 输出 JSON 解析失败：视为 `pass`，并记录 warning step。
- 二次返工仍未通过：返回最佳答案，并追加“本回答仍有以下限制”。
- Eino checkpoint 恢复失败：沿用现有恢复失败处理，提示用户重新发送问题。

**测试计划**

后端单元测试：

- ClarifierAgent prompt 包含“关键缺失才追问”的策略。
- 普通宽泛问题不会触发追问，例如“帮我分析一下 AI Agent 的发展趋势”。
- 关键缺失问题会触发追问，例如“帮我看看这个报错怎么修”。
- ClarifierAgent 输出能被解析为 `ClarifierDecision`。
- AcceptanceAgent `pass` 时不会返工。
- AcceptanceAgent `revise` 时最多触发 1 轮返工。
- AcceptanceAgent 连续不通过时不会无限循环。
- AcceptanceAgent `ask_user` 时进入 `waiting_user`。
- 流式事件包含 `clarifying_intent`、`reviewing`、必要时包含 `revising`。
- 现有 `ask_for_clarification` interrupt/resume 行为不被破坏。

集成测试：

- ADK runner 可同时保留 PlanExecute 和新增 Agent。
- `Chat` 优先走 Clarifier + ADK + Acceptance。
- `ChatStream` 在 Clarifier 追问后能持久化 pending run。
- 用户补充后能继续执行，而不是开启无关新 run。

手工验证：

- “帮我调研一下李小龙”：不追问，按人物调研维度回答。
- “帮我分析一下 AI Agent 的发展趋势”：不追问，自动覆盖趋势分析维度。
- “帮我看看这个报错怎么修”：追问完整报错和上下文。
- “Redis 和 MySQL 有什么区别”：不追问，按对比维度回答。
- “我现在只能选一个，Redis 和 MySQL 用哪个”：追问业务场景。

**风险与缓解**

风险 1：ClarifierAgent 追问过多。

- 缓解：prompt 明确“关键缺失才追问”；单元测试覆盖宽泛问题不追问。

风险 2：AcceptanceAgent 过度挑剔，导致频繁返工。

- 缓解：默认最多返工 1 次；prompt 明确“基本满足就 pass”。

风险 3：整体延迟增加。

- 缓解：Clarifier 和 Acceptance 都不使用工具，输出短 JSON；Acceptance 失败不阻断返回。

风险 4：JSON 输出不稳定。

- 缓解：提供解析兜底；必要时后续改为固定 schema 工具调用。

风险 5：流程状态变复杂。

- 缓解：不新增表，复用 `conversation_runs` 和 `conversation_run_steps`；服务层集中控制状态转换。

**实施顺序**

1. 定义 `ClarifierDecision`、`AcceptanceReview` 内部结构体和 JSON 解析函数。
2. 新增 ClarifierAgent 和 AcceptanceAgent 构造函数。
3. 在 `thinkTankADKRunner` 中装配两个 Agent。
4. 在非流式 `Chat` 中接入 Clarifier 和 Acceptance。
5. 在流式 `ChatStream` 中接入 Clarifier、Acceptance、返工和追问事件。
6. 增加步骤记录和运行快照字段写入。
7. 补齐单元测试和关键集成测试。

**验收标准**

- 普通宽泛问题不会被不必要地追问。
- 关键缺失问题会提出一个明确、可回答的追问。
- PlanExecute 仍然优先执行 Redis 知识库检索。
- 最终回答经过 AcceptanceAgent 的结构化验收。
- 审核不通过时最多返工 1 次，不会无限循环。
- 返工仍无法满足时返回最佳答案，并明确说明限制。
- 现有多 Agent 流式事件、断线恢复和 `ask_for_clarification` 中断恢复不发生回归。

**参考**

- Eino ADK: https://www.cloudwego.io/docs/eino/core_modules/eino_adk/
- Eino Plan Execute: https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/plan_execute/
- 当前 ADK 实现：`backend/internal/service/chat/thinktank_adk.go`
- 当前流式编排：`backend/internal/service/chat/thinktank_orchestrator.go`
