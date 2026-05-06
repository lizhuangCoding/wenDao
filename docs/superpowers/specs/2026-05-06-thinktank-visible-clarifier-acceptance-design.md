# ThinkTank 可见化澄清与验收体验设计

**目标**

修正第一版 ClarifierAgent / AcceptanceAgent 的产品表现问题。现有版本已经把两个 Agent 接入后端流程，但用户实际看到的效果仍然像普通追问和普通回答，无法感知：

- 澄清 Agent 是否真正理解了用户的实际需求。
- 后续回答是否基于这份需求画像生成。
- 评分 Agent 是否真的验收了最终内容。

本次目标是把两个 Agent 的核心产物变成“可见但简洁”的用户体验：

- ClarifierAgent 产出结构化需求画像，用于后续生成，也在必要时以清晰追问展示给用户。
- AcceptanceAgent 产出验收摘要，说明最终答案是否符合用户问题、评分是多少、覆盖了哪些维度、仍有什么限制。

**非目标**

- 不把聊天界面改成完整工作流看板。
- 不展示长推理链或复杂内部 JSON。
- 不让用户每次都先填表。
- 不引入新的 Agent runtime，继续使用 Eino ADK / ChatModelAgent。

**现状问题**

用户输入：

```text
我要学习知识
```

当前 ClarifierAgent 可能只返回：

```text
请问您想要学习哪个具体的知识领域呢？
```

这句话虽然触发了追问，但没有体现“澄清维度”的价值。更合理的表现应该告诉用户：

- 系统理解到的需求是什么。
- 目前缺少哪些关键信息。
- 为什么这些信息会影响后续回答。
- 用户可以按什么格式补充。

同时，AcceptanceAgent 当前只在后台决定 `pass|revise|ask_user`，即使模型返回了 `score`、`matched_dimensions`、`missing_dimensions`，最终用户也看不到评分和验收摘要。

**设计原则**

1. 澄清不是简单追问，而是形成需求画像。
2. 追问只在关键缺失时发生，但追问内容必须解释缺失维度。
3. 普通宽泛问题不打断用户，但要把推断出的回答维度传给后续生成。
4. 验收不是后台静默判断，而是在最终答案中给出简洁结论。
5. 可见内容要短，避免把 Agent 内部结构直接倒给用户。

## 方案选择

采用“结构化 Agent 输出 + 后端格式化展示”的方案。

```text
用户问题
  -> ClarifierAgent 生成需求画像
      -> 关键缺失：返回结构化澄清消息
      -> 可继续：把需求画像注入后续 Agent 查询，并记录到过程步骤
  -> Eino PlanExecute 生成答案
  -> AcceptanceAgent 按需求画像验收答案
      -> pass：追加验收摘要
      -> revise：自动修订一次，再追加验收摘要
      -> ask_user：返回结构化验收追问
```

不采用“让模型直接写完整可见文本”的方案。原因是模型自由写可见文案会导致格式不稳定，也不利于测试。Agent 仍返回结构化字段，后端负责生成稳定的用户可见文本。

## ClarifierAgent 设计

### 输出结构增强

保留现有字段：

```json
{
  "normalized_question": "",
  "intent": "",
  "answer_goal": "",
  "target_dimensions": [],
  "constraints": {},
  "ambiguity_level": "",
  "should_ask_user": false,
  "clarification_question": "",
  "reason": ""
}
```

新增面向产品体验的字段：

```json
{
  "need_summary": "我理解到的用户实际需求",
  "missing_dimensions": ["缺失的关键维度"],
  "why_needed": "为什么这些信息会影响后续回答",
  "suggested_reply": "用户可以直接复制/参考的回复模板"
}
```

这些字段仍由 ClarifierAgent 生成，但后端会做兜底：

- `need_summary` 为空时，用 `intent` 兜底。
- `missing_dimensions` 为空但需要追问时，从 `clarification_question` 兜底。
- `suggested_reply` 为空时，根据缺失维度生成通用模板。

### 追问展示格式

当 `should_ask_user=true` 时，不再只展示一句问题，而是展示：

```text
我理解你是想：制定一个学习计划，但目前学习目标还不够明确。

为了后续回答更精确，需要确认：
1. 学习领域：例如 AI、编程、投资、写作
2. 当前基础：零基础 / 入门 / 进阶
3. 学习目标：理解概念 / 能做项目 / 应对考试 / 职业提升

为什么需要这些信息：
不同领域、基础和目标会决定学习路径、资料难度和练习方式。

你可以这样回复：
我想学 AI，目前零基础，目标是能做一个小项目。
```

### 不追问时的表现

当问题宽泛但可回答时，不打断用户。例如：

```text
帮我分析一下 AI Agent 的发展趋势
```

ClarifierAgent 会推断：

```text
技术演进、产品形态、商业落地、风险限制、未来判断
```

这些内容会：

- 注入后续 Agent 查询，影响最终答案。
- 记录到多 Agent 过程面板中的 `ClarifierAgent` 步骤。
- 不额外插入一条聊天消息，避免打断体验。

过程面板示例：

```text
ClarifierAgent：已识别需求
实际需求：分析 AI Agent 的发展趋势
回答维度：技术演进、产品形态、商业落地、风险限制、未来判断
处理方式：无需追问，按推断维度继续调研
```

## AcceptanceAgent 设计

### 输出结构增强

保留现有字段：

```json
{
  "verdict": "pass|revise|ask_user",
  "score": 86,
  "matched_dimensions": [],
  "missing_dimensions": [],
  "unsupported_claims": [],
  "format_issues": [],
  "revision_instruction": "",
  "user_question": "",
  "reason": ""
}
```

新增：

```json
{
  "summary": "面向用户的简短验收结论"
}
```

如果模型没有给出 `summary`，后端根据已有字段生成。

### 最终答案展示格式

最终答案末尾追加简洁验收摘要：

```text
验收摘要：通过，评分 86/100
已覆盖：学习路径、阶段目标、实践方式、资源建议
仍需注意：由于未提供每日可投入时间，计划节奏按通用情况估算。
```

如果发生自动修订：

```text
验收摘要：初稿需要修订，已自动补充关键缺失项，最终评分 82/100
修订重点：补充实践路径和阶段任务
仍需注意：时间安排按通用节奏估算。
```

如果 AcceptanceAgent 判断仍需要用户补充：

```text
验收时发现还缺少一个关键信息：
你每天大概能投入多长时间学习？

为什么需要：
学习计划的节奏、任务量和阶段目标都依赖你的时间投入。
```

### 评分展示原则

- 只展示 `0-100` 的整数分。
- 不展示复杂评分表。
- `pass` 一般展示“通过”。
- `revise` 后展示“已自动修订”。
- 如果 AcceptanceAgent 出错并走兜底，不展示虚假的 `100/100`。

## 后端数据流

### 非流式

```text
Chat
  -> clarifyAgentQuery
      -> should_ask_user：返回结构化澄清消息
      -> continue：query 注入需求画像
  -> ADK 或手动 fallback 生成答案
  -> reviewAnswer
      -> ask_user：返回结构化验收追问
      -> revise：自动修订一次
      -> pass：追加验收摘要
  -> 返回最终答案
```

### 流式

```text
ChatStream
  -> stage: clarifying_intent
  -> emit ClarifierAgent step
  -> 需要追问：question 事件返回结构化澄清消息
  -> 继续执行多 Agent
  -> stage: reviewing
  -> emit AcceptanceAgent step
  -> revise 时 stage: revising
  -> 最终 chunk 包含答案正文 + 验收摘要
```

## 前端展示

本次前端保持轻改：

- 顶部阶段条继续显示当前阶段。
- 结构化澄清消息作为 assistant 消息显示。
- 验收摘要直接作为最终答案末尾 Markdown 显示。
- Agent 过程面板继续显示 ClarifierAgent / AcceptanceAgent 步骤详情。

不新增复杂评分卡组件。这样改动小，且移动端不会被流程信息撑得过重。

## 关键示例

### 示例 1：关键缺失

用户：

```text
我要学习知识
```

预期：

```text
我理解你是想：制定一个学习计划，但目前学习领域和目标还不够明确。

为了后续回答更精确，需要确认：
1. 学习领域：例如 AI、编程、投资、写作
2. 当前基础：零基础 / 入门 / 进阶
3. 学习目标：理解概念 / 能做项目 / 应对考试 / 职业提升

为什么需要这些信息：
不同领域、基础和目标会决定学习路径、资料难度和练习方式。

你可以这样回复：
我想学 AI，目前零基础，目标是能做一个小项目。
```

### 示例 2：宽泛但可回答

用户：

```text
帮我分析一下 AI Agent 的发展趋势
```

预期：

- 不追问。
- 内部需求画像包含：技术演进、产品形态、商业落地、风险限制、未来判断。
- 最终答案末尾展示：

```text
验收摘要：通过，评分 88/100
已覆盖：技术演进、产品形态、商业落地、风险限制、未来判断
仍需注意：具体市场数据会随时间变化，建议结合最新行业报告更新判断。
```

### 示例 3：初稿不符合

AcceptanceAgent 返回 `revise`：

```text
缺失维度：实践路径、阶段任务
```

预期：

- 系统自动修订一次。
- 用户只看到修订后的答案。
- 末尾展示：

```text
验收摘要：初稿缺少实践路径，已自动修订，最终评分 84/100
修订重点：补充阶段任务和练习项目
```

## 错误处理

- ClarifierAgent 调用失败：继续使用原问题，不展示“需求理解卡片”。
- ClarifierAgent JSON 解析失败：使用默认决策，不追问。
- AcceptanceAgent 调用失败：返回最终答案，不展示虚假评分。
- AcceptanceAgent 要求修订但修订失败：返回已有答案，并展示回答限制，不伪装成完全通过。
- 用户补充 pending 问题后：合并原始问题、系统追问和用户补充，跳过重复澄清。

## 测试计划

后端单元测试：

- `formatClarifierQuestion` 对“我要学习知识”输出需求理解、缺失维度、建议回复。
- `buildClarifiedAgentQuery` 包含新增需求画像字段。
- `appendAcceptanceSummary` 对 `pass` 输出评分和覆盖维度。
- `appendAcceptanceSummary` 对 `revise` 输出“已自动修订”。
- AcceptanceAgent 失败时不追加虚假评分。
- 流式路径发出 ClarifierAgent / AcceptanceAgent step。

集成测试：

- 非流式澄清追问返回结构化消息。
- 流式澄清追问返回结构化 question 事件。
- 宽泛但可回答问题不追问，并最终追加验收摘要。
- 修订失败时最终答案包含回答限制而不是失败。

前端验证：

- 结构化澄清消息 Markdown 显示正常。
- 最终答案中的验收摘要显示正常。
- 过程面板能展示 ClarifierAgent / AcceptanceAgent 步骤。

## 验收标准

- 用户输入“我要学习知识”时，不再只得到一句“想学习哪个领域”，而是得到结构化需求澄清。
- 用户能从最终答案看到 AcceptanceAgent 的评分和验收摘要。
- 宽泛但明确的问题不被过度追问。
- 评分摘要不会在 AcceptanceAgent 失败时伪造。
- 非流式和流式体验一致。
- 现有 ThinkTank 多 Agent 执行能力不退化。
