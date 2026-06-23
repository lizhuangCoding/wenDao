# WenDao 问道

WenDao 是一个前后端分离的个人技术博客平台，内置文章发布、合集管理、评论互动、用户认证、后台管理，以及基于 RAG 的多智能体 AI 助手能力。项目可以作为个人博客直接部署，也可以作为带 AI 问答和知识沉淀能力的内容平台继续扩展。

## AI 助手效果展示

| 多 Agent 协作过程 | AI 问答与参考文章 |
| --- | --- |
| ![多 Agent 协作过程](./docs/screenshots/1.png) | ![AI 问答与参考文章](./docs/screenshots/2.png) |

| 模型与对话体验 | 研究过程与结果沉淀 |
| --- | --- |
| ![模型与对话体验](./docs/screenshots/3.png) | ![研究过程与结果沉淀](./docs/screenshots/4.png) |

## 核心能力

- 文章系统：Markdown 文章、草稿/发布、分类、合集、封面图、正文图片上传。
- 用户系统：邮箱注册登录、GitHub OAuth、个人资料、评论通知偏好。
- 评论互动：两级评论、点赞/点踩、用户自删、管理员删除与恢复。
- 站内通知：评论回复通知、管理员广播通知。
- AI 助手：多智能体 ThinkTank 编排、RAG 知识召回、流式对话、历史记录、分享与导出。
- AI 写作：文章润色、扩展、缩短、SEO 标题生成。
- 知识文档：外部 URL 内容抓取、审核入库、转为发布文章。
- 后台管理：文章、分类、合集、评论、用户、知识文档、统计和站点设置。
- 3D 文章轨道：基于语义嵌入和 UMAP 降维的文章知识图谱展示。

## 技术栈

### 后端

- Go 1.24
- Gin
- GORM
- Viper + godotenv
- Zap + lumberjack
- JWT Access Token + Refresh Token
- MySQL 8+
- Redis / Redis Stack / Redis Vector
- CloudWeGo Eino，支持 Doubao、DeepSeek、OpenAI 兼容 Provider

### 前端

- React 18 + TypeScript
- Vite 5
- React Router
- Zustand
- TanStack React Query
- Tailwind CSS
- TDesign React
- Framer Motion
- Three.js / React Three Fiber
- Axios

## 项目结构

```text
wenDao/
├── backend/
│   ├── cmd/server/          # 后端入口和依赖装配
│   ├── config/              # 配置结构、YAML、.env 示例
│   ├── internal/            # handler、middleware、model、repository、service、pkg
│   ├── migrations/          # 版本化 SQL 迁移
│   ├── uploads/             # 本地上传文件
│   └── log/                 # 本地运行日志
├── frontend/
│   ├── src/api/             # API client
│   ├── src/components/      # 通用组件和业务组件
│   ├── src/pages/           # 前台页面
│   ├── src/views/admin/     # 后台管理页面
│   ├── src/store/           # Zustand store
│   ├── src/hooks/           # 自定义 hooks
│   └── src/styles/          # 全局样式
└── docs/
    ├── screenshots/         # README 展示图
    ├── deployment.md        # 部署文档
    └── 问道博客平台设计文档.md
```

## 本地开发

### 1. 准备后端配置

```bash
cd backend
cp config/.env.example config/.env
```

至少需要按本机环境调整数据库、Redis 和 JWT 配置：

```env
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=change-me
DB_NAME=wendao
REDIS_HOST=localhost
REDIS_PORT=6379
JWT_SECRET=change-me-to-a-long-random-secret
```

AI、GitHub OAuth、邮箱验证码等能力可以先留空；需要启用时再补充对应环境变量。

### 2. 启动后端

```bash
cd backend
go run ./cmd/server
```

默认监听端口：`8089`。

### 3. 启动前端

```bash
cd frontend
npm ci
npm run dev
```

默认访问地址：`http://localhost:3000`。Vite 会将 `/api` 和 `/uploads` 代理到后端 `8089`。

## 常用命令

### 后端

```bash
cd backend
go test ./...
go build ./cmd/server
go fmt ./...
```

### 前端

```bash
cd frontend
npm run dev
npm run build
npm run lint
npm run preview
```

## 配置说明

后端主要读取 `backend/config/config.yaml`，并支持通过 `backend/config/.env` 或环境变量覆盖。常用变量包括：

- `DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASSWORD`、`DB_NAME`
- `MIGRATION_MODE`、`MIGRATION_PATH`
- `REDIS_HOST`、`REDIS_PORT`、`REDIS_PASSWORD`
- `REDIS_VECTOR_HOST`、`REDIS_VECTOR_PORT`、`REDIS_VECTOR_PASSWORD`
- `JWT_SECRET`
- `SITE_URL`
- `GITHUB_CLIENT_ID`、`GITHUB_CLIENT_SECRET`、`GITHUB_CALLBACK_URL`
- `EMAIL_SMTP_HOST`、`EMAIL_USERNAME`、`EMAIL_PASSWORD`、`EMAIL_FROM_ADDRESS`
- `AI_PROVIDER`、`AI_API_KEY`、`AI_ENDPOINT`、`AI_CHAT_MODEL`
- `DOUBAO_API_KEY`、`DOUBAO_ENDPOINT`、`DOUBAO_CHAT_MODEL`、`DOUBAO_EMBEDDING_MODEL`
- `RESEARCH_ENDPOINT`、`RESEARCH_API_KEY`

生产环境默认使用 `MIGRATION_MODE=versioned`，按 `backend/migrations/*.sql` 执行版本化迁移。本地开发如需使用 GORM 自动建表，可以显式设置 `MIGRATION_MODE=auto`。

## Docker 部署

项目支持通过 Docker Compose 部署完整栈：

- Caddy：HTTPS 终止与反向代理
- Nginx：托管 Vite 构建后的前端资源
- Backend：Go API 服务
- MySQL：关系数据
- Redis Stack：缓存和向量检索

准备生产环境变量：

```bash
cp .env.production.example .env.production
```

域名部署：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

仅 IP 访问部署：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.ip.yml up -d --build
```

更多部署、迁移、GitHub Actions 和备份说明见 [docs/deployment.md](./docs/deployment.md)。

## 创建管理员

生产部署后可在后端容器内运行管理员初始化命令。若 `.env.production` 已设置 `ADMIN_EMAIL`、`ADMIN_USERNAME`、`ADMIN_PASSWORD`：

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml -f docker-compose.ip.yml exec \
  backend /app/wendao-init-admin
```

也可以只在执行命令时传入管理员账号信息，避免把真实密码提交到仓库。

## 开发约定

- Go 代码使用标准 `gofmt`。
- Handler 保持薄层，业务逻辑放在 `internal/service`，持久化放在 `internal/repository`。
- 前端使用 TypeScript、React function components、Tailwind utilities 和 `@` 路径别名。
- 后端测试使用 Go `testing`，测试文件与被测代码同目录。
- 前端改动至少运行 `npm run build` 和 `npm run lint`。
- 不提交真实 `.env`、密钥、日志、上传文件和本地构建产物。

## 相关文档

- [平台设计文档](./docs/问道博客平台设计文档.md)
- [生产部署文档](./docs/deployment.md)
- [前端说明](./frontend/README.md)

## License

当前仓库尚未声明独立 License。如需公开分发或商用，请先补充明确的许可证文件。
