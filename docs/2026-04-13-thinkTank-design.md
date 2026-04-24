# ThinkTank Matrix (圆桌·矩阵) 设计文档

## 1. 项目概述
ThinkTank Matrix 是一个基于 Eino 框架构建的高级多智能体协作系统。其核心目标是处理复杂的、跨领域的研究任务，通过模拟人类专家组的协作模式，将碎片化的互联网信息转化为结构化的本地知识资产。

系统不仅能够执行简单的问答，更能通过**自主规划、专家协作、结果审计以及人机交互**，完成从“提出问题”到“产出深度调研报告并入库”的全链路闭环。

## 2. 核心协作范式：嵌套式混合架构
ThinkTank 采用了一种**“分层且递归”**的混合多智能体架构，主要由两个核心模式嵌套而成：

### 2.1 外层：战略层 (Plan-Execute-Replan)
系统最外层采用 **PE-R (Plan-Execute-Replan)** 模式。这种模式赋予了系统“宏观思考”的能力：
- **Planner (战略官)**：负责将用户输入的复杂模糊任务拆解为一系列可执行的子任务。
- **Executor (执行器)**：作为任务下发中心，将子任务转交给下游的专家团队。
- **Replanner (审计官)**：在执行层返回结果后，对比初始目标进行质量评估。如果结果不完整或执行失败，审计官会重新制定计划，触发新一轮的执行循环。

### 2.2 内层：执行层 (Supervisor)
`Executor` 内部封装了一个 **Supervisor (指挥官)** 节点。该节点负责微观层面的资源调配：
- **StrikeTeamSupervisor**：接收来自战略层的子任务，并根据职能将其分发给最合适的专家。它不直接处理具体工作，而是监督专家的进度并整合最终报告。
- **专家 Agents**：
    - **Librarian (图书管理员)**：专注 **RAG (检索增强生成)**。通过本地 Redis 向量库检索已有的私人笔记和文档。如果本地知识不足，它会建议启动外部调研。
    - **Journalist (外勤记者)**：专注 **ReAct (Reasoning and Acting) 调研**。它能够自主进行多轮搜索、跨源验证，并将最终成果以 Markdown 格式持久化到本地，同时触发向量索引更新。

## 3. 核心 Agent 职能定义

| Agent 角色 | 核心模式 | 核心能力 | 工具集 (Capabilities) |
| :--- | :--- | :--- | :--- |
| **Planner** | Plan-Execute | 任务拆解、路径规划 | LLM 推理 |
| **Replanner** | Replan | 质量评估、计划修正 | LLM 推理 |
| **Supervisor** | Supervisor | 专家调度、结果整合 | LLM 路由 |
| **Librarian** | RAG | 本地知识检索 | `LocalSearch`, `AskUser` |
| **Journalist** | ReAct | 深度互联网调研、知识内化 | `WebSearch`, `DocWriter`, `AskUser`, `LocalSearch` |

## 4. 协作流转逻辑
1. **输入与拆解**：用户输入“调研 2024 年大模型在工业界的落地情况”。`Planner` 将其拆解为：a) 检索本地已有案例；b) 搜索最新新闻和论文。
2. **指派与执行**：
    - `Supervisor` 先将任务 a 分派给 `Librarian`。
    - `Librarian` 使用 `LocalSearch` 发现库中只有 2023 年的数据，通过 `AskUser` 告知用户并建议补充调研。
    - `Supervisor` 收到反馈后，指派 `Journalist` 执行任务 b。
3. **深度调研与内化**：`Journalist` 进行多轮搜索，调用 `DocWriter` 生成 Markdown 文档。
4. **结果汇总与审计**：`Supervisor` 整合 `Librarian` 和 `Journalist` 的报告，提交给 `Replanner`。
5. **任务完成**：`Replanner` 确认报告涵盖了所有要求，向用户输出最终答案。

## 5. 关键技术特性

### 5.1 权限最小化与精准分发 (ToolKit)
系统通过 `ToolKit` 结构体对工具进行分类分发。这种设计确保了 Agent 的职能专注化（例如：`Librarian` 不需要 `WebSearch` 权限），降低了 LLM 调用工具时的误操作风险。

### 5.2 人机协作与中断恢复 (Human-in-the-Loop)
利用 Eino 的 **Checkpoint** 机制和 `AskUser` 工具：
- 当 Agent 遇到模棱两可的情况时，会触发中断，等待用户输入。
- `adk.Runner` 能够保存当前任务状态（Checkpoint），用户回答后，系统可以从中断处完美恢复，避免重复执行已完成的步骤。

### 5.3 知识闭环 (Knowledge Loop)
`Journalist` 生成的文档不仅仅是展示给用户的，它会被保存到本地目录并自动同步到 Redis 向量库。这意味着下一次执行任务时，`Librarian` 就能检索到这些“新知识”，实现了系统知识库的自我进化。

---
*文档版本：v1.0*
*编写日期：2026-04-13*
