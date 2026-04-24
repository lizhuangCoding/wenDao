# 后端启动骨架与 ThinkTank 模块化重构设计

## 实施状态

- 第 1 批：已完成
- 第 2 批：未开始
- 第 3 批：未开始

## 1. 背景

当前后端已经进入“功能可用，但结构压力逐渐增大”的阶段。最突出的两个信号是：

1. `backend/internal/service/thinktank.go` 已达到约 1680 行，单文件同时承担了会话装配、ADK 编排、流式事件发送、记忆压缩、运行记录收尾、知识草稿沉淀等多种职责。
2. `backend/cmd/server/main.go` 已达到约 590 行，虽然已经做过一轮初始化拆分，但仍然混合了配置加载、日志初始化、基础设施初始化、AI 组件装配、HTTP 路由装配和服务启动。

这两个文件已经不是“看起来有点长”的问题，而是已经开始影响：

- 可维护性：修改一个能力时需要在巨型文件中寻找多个散落的逻辑点。
- 可扩展性：新增 Agent、记忆策略、启动步骤时，很容易继续往同一个文件堆。
- 可靠性：局部修改难以控制影响面，回归风险上升。
- 可测试性：核心流程过于集中，难以形成稳定的小粒度测试边界。

本设计聚焦一轮**不改外部行为、分批次落地**的后端结构重构，优先解决：

- `main.go` 退化为真正的启动入口。
- `thinktank.go` 从“神文件”演进为职责清晰的模块化结构。

---

## 2. 目标与非目标

### 2.1 目标

本次重构设计的目标是：

1. 将 `main.go` 收缩为真正的进程入口，只保留“启动应用”的职责。
2. 把启动编排迁移到明确的 bootstrap / app 层。
3. 将 `thinktank.go` 逐步拆分为多个小职责模块，降低单文件复杂度。
4. 保持现有 API、数据库模型、核心返回结构和业务行为稳定。
5. 在每一批重构后都能通过测试验证和最小范围回归验证。
6. 为后续新增 Agent、记忆策略、调研流程提供更清晰的扩展位。

### 2.2 非目标

以下内容不在本轮重构范围内：

1. 不更换 Gin / Service / Repository 的整体分层范式。
2. 不引入重量级 DI 框架或复杂 IoC 容器。
3. 不重写 ThinkTank 的业务逻辑，不改变现有多 Agent 产品能力。
4. 不同时处理所有大文件，只优先处理 `main.go` 和 `thinktank.go`。
5. 不在本轮中调整数据库 schema、路由接口或前端协议。

---

## 3. 当前结构问题

### 3.1 `main.go` 的问题

当前 `backend/cmd/server/main.go` 仍包含以下耦合：

1. 进程入口与应用启动编排耦合。
2. 基础设施初始化与业务服务装配耦合。
3. AI 初始化降级逻辑与 HTTP 启动逻辑耦合。
4. 路由注册、handler 装配与服务构建混在同一个文件中。

这会导致：

- `main.go` 很难快速读懂。
- 未来增加新的启动步骤时，继续往入口文件塞逻辑。
- 难以把“启动失败”和“模块降级”做成清晰的策略层。

### 3.2 `thinktank.go` 的问题

当前 `backend/internal/service/thinktank.go` 同时承担：

1. `ThinkTankService` 主流程入口。
2. 会话归属校验与会话恢复。
3. 聊天消息读写与元数据更新。
4. Conversation run / step 的持久化。
5. 会话记忆压缩、加载和落库。
6. ADK 中断恢复与 checkpoint 协调。
7. Stream event 的装配与输出。
8. Journalist 调研结果转知识草稿。

这会导致：

- 一个修改点往往横跨多个逻辑区块。
- 文件内部函数越来越依赖隐式上下文。
- 测试只能围着巨型 service 做堆叠式验证，不容易形成小边界。

---

## 4. 重构原则

本轮重构必须遵守以下原则：

1. **先稳后美**：先把边界立住，再追求更优雅的抽象。
2. **不改行为，只改结构**：除非发现明确缺陷，否则不顺手改业务规则。
3. **分批推进**：每一批都应可独立提交、可独立验证、可独立回滚。
4. **接口稳定**：对外 handler、service 接口、SSE 事件结构保持不变。
5. **小步验证**：每一批都必须通过测试后再进入下一批。
6. **显式依赖**：新模块之间的关系要通过构造和参数表达，而不是靠共享大对象和隐式状态。

---

## 5. 总体方案

本重构按三批推进。

### 5.1 第 1 批：文件级职责切分

目标：

- 不改变主要类型结构和依赖注入方式。
- 先把 `main.go` 和 `thinktank.go` 从“单文件堆逻辑”改成“同层多个文件协作”。

这一步本质上是**结构止血**：优先解决大文件可读性和定位成本问题。

### 5.2 第 2 批：ThinkTank 内部模块化

目标：

- 在保持 `ThinkTankService` 对外接口不变的前提下，拆出内部协调器和职责组件。
- 让 `thinkTankService` 从“全能实现”演进为“模块装配器 + 流程编排器”。

这一步本质上是**职责收束**：让多 Agent 编排链路具备长期扩展性。

### 5.3 第 3 批：收尾与结构收口

目标：

- 调整测试结构，补足关键集成测试。
- 收口启动层边界。
- 评估并清理剩余过大的核心生产代码文件。

这一步本质上是**工程固化**：确保结构调整不是一次性的“美化”，而是稳定的新骨架。

---

## 6. 第 1 批设计：文件级职责切分

### 6.1 `main.go` 收缩策略

重构后目标形态：

```go
func main() {
    if err := Run(); err != nil {
        log.Fatal(err)
    }
}
```

或：

```go
func main() {
    MustRun()
}
```

具体拆分建议如下。

### 6.2 新启动层文件建议

#### `backend/cmd/server/main.go`

职责：

- 进程入口。
- 调用应用启动函数。

不应再承担：

- 配置读取细节。
- 基础设施初始化细节。
- AI 装配细节。
- 路由注册细节。

#### `backend/cmd/server/app.go`

职责：

- 定义 `Run()` / `MustRun()`。
- 编排配置加载、日志初始化、bootstrap、HTTP Server 启动。

这是新的“应用启动总控层”，但不直接承担具体模块初始化细节。

#### `backend/cmd/server/bootstrap_infra.go`

职责：

- `loadServerEnv`
- 配置加载
- 日志初始化
- MySQL / Redis / Redis Vector 初始化
- 基础设施结构体组装

#### `backend/cmd/server/bootstrap_ai.go`

职责：

- AI 组件初始化
- 向量索引初始化
- 启动期向量同步
- AI 降级策略封装

要求：

- AI 初始化失败时，决定“降级继续”还是“必须失败”的策略必须集中在这一层，不应散落在 `main.go`。

#### `backend/cmd/server/bootstrap_http.go`

职责：

- repositories 构建
- services 构建
- handlers 构建
- router 构建
- routes 注册

目标：

- 让 HTTP 层装配逻辑与基础设施初始化解耦。

### 6.3 ThinkTank 文件拆分建议

第 1 批中，`thinkTankService` 结构和大部分方法签名保持不变，只移动函数位置。

建议拆分为：

#### `backend/internal/service/thinktank_service.go`

职责：

- `ThinkTankService` 接口
- `thinkTankService` 结构体
- `NewThinkTankService`
- `Chat`
- `ChatStream`

这里保留主流程入口，但不保留所有辅助细节。

#### `backend/internal/service/thinktank_conversation.go`

职责：

- 会话归属校验
- 会话消息读取 / 保存
- conversation metadata 更新
- pending run 查找

#### `backend/internal/service/thinktank_memory.go`

职责：

- 记忆压缩
- 历史上下文拼装
- 会话记忆读取与持久化

#### `backend/internal/service/thinktank_stream.go`

职责：

- 流式事件装配
- 阶段事件输出
- step 事件输出
- done / error 事件收尾辅助

#### `backend/internal/service/thinktank_adk_resume.go`

职责：

- ADK 中断恢复
- checkpoint id 生成
- pending context 解析
- 中断恢复时的状态承接

#### `backend/internal/service/thinktank_run_record.go`

职责：

- conversation run 的创建、完成、失败更新
- step 的创建与更新
- 执行记录的最终收尾

### 6.4 第 1 批验收标准

1. `main.go` 不超过约 40 行，只保留入口职责。
2. `thinktank.go` 被拆解，不再作为巨型总文件存在。
3. `ThinkTankService` 对外接口不变。
4. AI Chat 现有 handler 不需要改调用方式。
5. 后端测试通过，且 AI 对话主链路不回归。

---

## 7. 第 2 批设计：ThinkTank 内部模块化

第 2 批不再满足于“同一个 service 分多个文件”，而是把内部职责真正抽成可组合组件。

### 7.1 目标结构

建议把现有 `thinkTankService` 拆成以下内部协作者：

#### `thinkTankOrchestrator`

职责：

- 驱动 `Chat` / `ChatStream` 主流程。
- 控制“本地检索 -> 联网调研 -> 汇总 -> 持久化收尾”的顺序。
- 对外只暴露流程级方法。

#### `conversationManager`

职责：

- 处理会话归属
- 拉取历史消息
- 保存用户 / 助手消息
- 更新会话标题、时间戳等元数据

#### `memoryManager`

职责：

- 压缩历史消息
- 构造 agent query
- 读取和持久化 conversation memories

#### `runRecorder`

职责：

- 管理 `conversation_runs`
- 管理 `conversation_run_steps`
- 标准化“开始 / 中断 / 完成 / 失败”的记录行为

#### `streamEmitter`

职责：

- 标准化 SSE 事件输出模型
- 保证阶段事件、step 事件、chunk 事件的边界一致

#### `researchDraftSink`

职责：

- 把 Journalist 的结构化结果转换为知识草稿
- 将知识草稿入库逻辑从主流程里抽离

### 7.2 关系原则

第 2 批中建议遵守：

1. `thinkTankService` 只保留少量字段，更多依赖注入到小组件。
2. 主流程通过组合这些组件完成，而不是直接操作 repo。
3. 一个组件只解决一个清晰问题。

目标效果：

- 以后新增新的记忆策略时，只改 `memoryManager`。
- 以后新增新的运行记录规则时，只改 `runRecorder`。
- 以后新增新的事件阶段时，主要改 `streamEmitter`。

### 7.3 第 2 批验收标准

1. `thinkTankService` 明显瘦身，更多像装配器而不是“全能类”。
2. repo 访问路径集中，不再在主流程到处散落。
3. 新增单元测试可围绕组件写，而不是只能写超大集成测试。
4. Chat / ChatStream 的返回行为保持不变。

---

## 8. 第 3 批设计：收尾与工程固化

### 8.1 测试结构整理

建议把当前大而集中的测试进一步拆分：

- `thinktank_service_test.go`
- `thinktank_memory_test.go`
- `thinktank_stream_test.go`
- `thinktank_run_record_test.go`
- `thinktank_adk_resume_test.go`

目标：

- 让失败定位更快。
- 减少“一个测试文件理解成本过高”的问题。

### 8.2 启动层边界收口

第 3 批后应形成稳定启动链路：

```text
main.go
  -> Run()
    -> load config/env
    -> init logger
    -> bootstrap infrastructure
    -> bootstrap AI
    -> bootstrap HTTP
    -> start server
```

要求：

- 启动路径清晰。
- 降级路径清晰。
- 不再出现“主函数里继续堆初始化细节”的趋势。

### 8.3 评估剩余大文件

完成前两批后，再评估以下文件是否需要继续收口：

- `backend/internal/service/article.go`
- `backend/internal/service/upload.go`
- `backend/internal/handler/article.go`

这一步只做评估和必要拆分，不强行把所有偏长文件都拆碎。

### 8.4 第 3 批验收标准

1. 启动结构稳定，入口职责固定。
2. ThinkTank 测试可按职责定位问题。
3. 剩余大文件至少完成一轮边界评估，并对最有价值的点做整理。
4. 项目结构不再鼓励“继续往神文件里堆逻辑”。

---

## 9. 风险与控制措施

### 9.1 风险

1. 重构中移动函数位置时引入隐藏依赖问题。
2. ThinkTank 流式链路重构时破坏 SSE 行为。
3. ADK 恢复和 pending run 逻辑在拆分后出现边界遗漏。
4. 启动层拆分后，AI 降级路径和服务装配关系变复杂。

### 9.2 控制措施

1. 每一批只做一种层级的改变，不混入新功能。
2. 先补测试，再移动代码。
3. 第 1 批优先做文件级切分，不急着上抽象。
4. 保留关键集成测试，验证 Chat / ChatStream 主链路。
5. 每一批独立提交，保证可回滚。

---

## 10. 实施顺序

推荐顺序如下：

1. 第 1 批：启动骨架和 ThinkTank 文件级切分。
2. 第 2 批：ThinkTank 内部模块化。
3. 第 3 批：测试整理、启动层收口、剩余大文件评估。

每一批完成后都执行：

1. 后端测试
2. 关键 AI 对话链路验证
3. 启动链路验证

---

## 11. 设计结论

本设计不追求一次性“重写后端”，而是通过三批渐进式重构，把当前最危险的两个结构热点 `main.go` 和 `thinktank.go` 收口到可维护、可扩展、可验证的状态。

最终目标不是把代码“拆得很碎”，而是形成稳定的工程边界：

- `main.go` 只负责启动。
- 启动编排由专门 bootstrap 层负责。
- ThinkTank 由多个小职责模块协作完成。
- 未来扩展新的 Agent、记忆策略或执行阶段时，不需要继续修改单个神文件。

这套方案的核心价值在于：**在不牺牲现有稳定性的前提下，为项目后续迭代建立长期可持续的后端骨架。**
