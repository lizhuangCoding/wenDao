# 对话记忆模块技术说明

## 目标

对 AI 助手的多轮对话上下文做分层压缩，避免把全部历史消息直接塞进 Agent prompt。系统优先使用 LLM 生成结构化长期记忆，并按当前问题动态选择近期原文和相关旧片段；LLM 失败时退回规则摘要。

## 数据模型

新增 MySQL 表 `conversation_memories`，对应 `model.ConversationMemory`。

核心字段：

- `conversation_id`: 记忆所属会话。
- `user_id`: 记忆所属用户。
- `scope`: 记忆类型，当前支持 `conversation_summary`、`user_preference`、`project_fact`、`decision`、`open_thread`。
- `content`: 压缩后的记忆内容。
- `source_message_id_start` / `source_message_id_end`: 摘要覆盖的消息范围。
- `importance`: 预留权重字段，后续可用于排序和筛选。

表结构通过 `database.AutoMigrate` 自动迁移。

## 记忆分层策略

构造 Agent 输入时使用 `buildConversationMemoryForQuestion(question, history, memories)`：

1. `长期记忆`: 从 `conversation_memories` 读取的历史摘要。
2. `更早对话摘要`: 当前会话中超过近期预算窗口的旧消息，压缩成一句话。
3. `相关历史片段`: 如果当前问题包含能匹配旧消息的关键词，即使该消息不在最近窗口内，也会召回最多 `3` 条。
4. `最近对话`: 根据当前问题动态分配预算后保留原始角色和内容，每条最多截断到 `120` 字。

近期窗口不是固定条数，而是按问题类型估算预算：

- 指代型或很短的问题，如“继续”“这个呢”“原文在哪里”，使用更大的近期预算。
- 很长且自包含的问题，使用较小近期预算。
- 普通问题使用中等预算。

最终输入格式：

```text
历史上下文：
长期记忆：
- 用户此前关注 Manus 风格多 Agent 过程展示、工具调用日志和 MySQL 持久化。

更早对话摘要：
- 更早对话主要讨论：...

最近对话：
user: ...
assistant: ...

当前问题：
...
```

## 写入时机

在一次 AI 回答完成后，服务会调用 `updateConversationMemoryWithWarning`：

- 对话长度未超过近期窗口时不写入摘要。
- 优先调用 `ConversationMemorySummarizer`，由 LLM 输出结构化 JSON。
- 使用 `ConversationMemoryRepository.Upsert` 按 `conversation_id + scope` 更新同一类记忆。
- LLM 摘要失败时，降级为规则型 `conversation_summary`，主回答流程不失败。
- 写入失败只记录 AI 日志，不阻断主回答流程。

LLM 输出格式：

```json
{
  "conversation_summary": "一句话概括会话主线",
  "user_preferences": ["用户稳定偏好"],
  "project_facts": ["项目事实"],
  "decisions": ["已经确认的技术/产品决策"],
  "open_threads": ["仍未完成或后续要处理的问题"]
}
```

## Agent 接入点

手动编排和 ADK 编排共用同一份记忆：

- 手动路径：`queryForAgents := buildAgentQuery(question, memory)`。
- ADK 路径：将同样的 `queryForAgents` 作为 `schema.UserMessage` 输入 runner。

这样多 Agent 过程中的 Librarian、Journalist 和 Supervisor 都能看到一致的上下文。

## 当前限制与后续扩展

当前 LLM 摘要是同步热路径执行，并带规则降级。后续可将摘要更新移到后台任务，并改为真正增量摘要：

```text
旧摘要 + 新增早期消息 -> 新摘要
```

后续也可以将 `conversation_memories` 向量化，按语义相似度召回旧记忆，而不是只使用关键词匹配。
