# 问道 WenDao

一个面向技术内容创作与 AI 辅助研究的全栈博客平台。

`WenDao` 将传统博客系统、管理后台、AI 问答、多 Agent 研究流程与知识沉淀能力结合在一起，当前仓库包含 Go 后端与 React 前端两部分。

## Preview

- 技术博客前台：文章浏览、分类筛选、Markdown 渲染、评论互动
- 管理后台：文章管理、分类管理、评论管理、知识文档审核
- AI 能力：AI 对话、流式输出、RAG 检索、研究型多 Agent 协作
- 知识沉淀：研究结果可转化为知识文档，进一步进入内容与检索链路

## Highlights

- Go + Gin + Gorm 的后端结构，按 `handler / service / repository` 分层
- React + TypeScript + Vite 前端，使用 Zustand 与 TanStack Query 管理状态和请求
- 支持 JWT 认证、GitHub OAuth 登录、用户头像上传与用户名更新
- 支持 Markdown 文章编辑、摘要生成、自动保存、封面上传
- 集成向量检索与 Redis Vector，用于 AI 问答与知识召回
- 内置 ThinkTank 多 Agent 研究流，支持阶段事件、步骤日志与会话记忆

## Tech Stack

**Backend**

- Go
- Gin
- Gorm
- MySQL
- Redis / Redis Stack
- Zap
- Viper

**Frontend**

- React 18
- TypeScript
- Vite
- Tailwind CSS
- Zustand
- TanStack Query
- Axios
- Framer Motion

**AI / Knowledge**

- Eino
- Embedding / LLM 接入
- RAG 检索链
- 多 Agent 研究与知识文档管理

## Architecture

```text
frontend (React + Vite)
  -> REST API / SSE
backend (Gin)
  -> service layer
  -> repository layer
  -> MySQL
  -> Redis / Redis Vector
  -> AI / RAG / ThinkTank workflow
```

## Project Structure

```text
wenDao/
├── backend/                    # Go 后端
│   ├── cmd/server/             # 服务入口
│   ├── config/                 # 配置文件与环境变量
│   ├── internal/
│   │   ├── handler/            # HTTP 处理层
│   │   ├── middleware/         # 中间件
│   │   ├── model/              # 数据模型
│   │   ├── repository/         # 数据访问层
│   │   ├── service/            # 业务逻辑与 AI 编排
│   │   └── pkg/                # 公共基础能力
│   ├── uploads/                # 本地上传目录
│   └── migrations/             # 数据迁移相关
├── frontend/                   # React 前端
│   ├── src/api/                # API 请求封装
│   ├── src/components/         # 通用与业务组件
│   ├── src/pages/              # 页面级入口
│   ├── src/views/              # 视图实现
│   ├── src/store/              # Zustand 状态
│   ├── src/hooks/              # 自定义 Hooks
│   └── src/styles/             # 全局样式
├── docs/                       # 设计文档、方案和计划
├── examples/                   # 实验性示例
└── scripts/                    # 辅助脚本
```

## Quick Start

### 1. Start backend

```bash
cd backend
go run ./cmd/server
```

默认端口：`8089`

### 2. Start frontend

```bash
cd frontend
npm ci
npm run dev
```

默认地址：`http://localhost:3000`

前端开发服务器会将 `/api` 与 `/uploads` 代理到后端。

## Common Commands

### Backend

```bash
cd backend
go test ./...
go build ./cmd/server
go fmt ./...
```

### Frontend

```bash
cd frontend
npm run dev
npm run build
npm run lint
npm run preview
```

## Configuration

后端主要读取 `backend/config/config.yaml`，并支持使用 `backend/config/.env` 或环境变量覆盖。

常见环境变量包括：

- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `JWT_SECRET`
- `DOUBAO_API_KEY`
- `GITHUB_CLIENT_ID`
- `GITHUB_CLIENT_SECRET`
- `GITHUB_CALLBACK_URL`
- `REDIS_HOST`
- `REDIS_VECTOR_HOST`

前端可通过 `frontend/.env` 指定：

```env
VITE_API_BASE_URL=/api
```

## Current Capabilities

- 博客文章发布与展示
- 分类与评论管理
- 后台内容编辑
- 用户注册、登录与 GitHub OAuth
- AI 聊天与流式输出
- 基于向量检索的知识召回
- 多 Agent 研究过程展示
- 知识文档审核与沉淀

## Notes

- 仓库中不应提交真实的 `.env`、日志、上传文件和本地构建产物
- 当前前端子项目已有单独的说明文档，见 [frontend/README.md](./frontend/README.md)
- `docs/` 目录中保留了较多设计稿与计划文档，可作为后续迭代参考

## License

当前仓库尚未声明独立 License；如需开源发布，建议补充明确的许可证文件。
