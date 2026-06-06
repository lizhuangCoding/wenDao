# 功能扩展规划文档

> 基于 2026-06-06 对项目现有功能的全量梳理，识别功能空白并规划可扩展方向。
> 与本系列其他文档的关系：`2026-06-05-code-review-requirements.md`（Bug + 基础优化）、`2026-06-06-architecture-optimization.md`（多 Agent 架构 + 性能优化）。

---

## 一、项目当前能力全景

### 已有功能清单

| 模块 | 已有功能 |
|------|----------|
| **文章** | CRUD、草稿/发布、Markdown 编辑 + 预览、自动保存、置顶、封面图、Slug、分类关联、热度排序、AI 向量化入库 |
| **分类** | CRUD |
| **评论** | 发表/删除/列表、回复邮件通知（HTML 模板） |
| **用户** | 注册/登录、邮箱验证、密码重置、GitHub OAuth、头像上传、JWT + Refresh Token、Redis 黑名单 |
| **AI Chat** | ThinkTank 多 Agent 流程、SSE 流式输出、对话管理、中断恢复、对话记忆、RAG 本地检索、Web 搜索 |
| **知识文档** | 列表/查看/审批/驳回/删除、来源追踪、AI 自动创建研究草稿、审批后自动生成文章 |
| **统计** | 全站 PV / UV / 评论数、日维度趋势图、Redis 实时计数 |
| **上传** | 图片上传 + 压缩、定时清理过期文件 |
| **后台** | 文章/分类/评论/知识文档管理、数据面板 |
| **Docker** | Docker Compose 一键部署、Caddy 反向代理 + HTTPS |

---

## 二、功能扩展优先级矩阵

判断标准：
- **P0**：用户感知强，补缺核心体验，工作量可控
- **P1**：提升内容运营效率，增加用户粘性
- **P2**：锦上添花，扩展平台能力边界
- **P3**：长期愿景，需要较大架构调整

---

## 三、P0 —— 核心体验补缺

### 3.1 文章版本历史

**现状**：`PUT /api/admin/articles/:id/autosave` 直接覆盖文章表，没有历史记录。作者无法查看或回退到之前的版本。

**需求描述**：
- 每次自动保存或手动保存时，将当前版本存入 `article_versions` 表
- 管理员可在文章编辑页查看版本列表，对比差异，回退到任意历史版本
- 版本列表只保留标题 + 时间 + 字数摘要，不展示完整内容（避免列表撑爆）

**数据模型**：
```sql
CREATE TABLE article_versions (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    article_id  BIGINT NOT NULL,
    title       VARCHAR(255),
    content     LONGTEXT,
    created_at  DATETIME DEFAULT NOW(),
    INDEX idx_article_id (article_id)
);
```

**涉及文件**：
- `backend/internal/model/article_version.go`（新增）
- `backend/internal/repository/article/`（新增版本保存/查询/回退方法）
- `backend/internal/handler/article/article.go`（新增 `GET /api/admin/articles/:id/versions`、`POST /api/admin/articles/:id/versions/:versionId/restore`）
- `frontend/src/views/admin/articles/ArticleEditor.tsx`（新增版本历史面板）

---

### 3.2 定时发布文章

**现状**：文章模型有 `PublishedAt` 字段，但发布接口是即时生效的。作者无法设置未来时间自动发布。

**需求描述**：
- 文章编辑页新增"定时发布"选项，支持设定未来的发布时间
- 后端定时任务（每分钟检查一次）自动发布到期文章
- 已在定时列表中的文章显示"待发布"状态标识

**涉及文件**：
- `backend/internal/model/article.go`（新增 `ScheduledPublishAt` 字段）
- `backend/internal/handler/article/article.go`（新增 `POST /api/admin/articles/:id/schedule` 接口）
- `backend/cmd/server/background_tasks.go`（新增定时检查并发布的 goroutine）

---

### 3.3 对话分享与导出

**现状**：AI 对话完全没有分享和导出功能。用户无法把一段有价值的 AI 回答分享给别人或保存下来。

**需求描述**：

**对话分享**：
- 用户可生成对话分享链接（可选是否包含完整对话历史，或仅分享单轮问答）
- 分享链接为只读页面，不需要登录
- 分享者可在对话列表里管理/撤销已生成的分享链接
- 分享页面和当前对话展示页共用大部分 UI

**对话导出**：
- 导出格式：Markdown（直接可用）、PDF（打印友好）
- 导出内容：完整对话记录，含 AI 思考过程和最终回答

**数据模型**：
```sql
CREATE TABLE conversation_shares (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    conversation_id BIGINT NOT NULL,
    share_code      VARCHAR(64) UNIQUE NOT NULL,
    message_ids     JSON,       -- 可选：只分享指定消息
    created_at      DATETIME DEFAULT NOW(),
    INDEX idx_share_code (share_code)
);
```

**涉及文件**：
- `backend/internal/model/conversation_share.go`（新增）
- `backend/internal/handler/chat/chat.go`（新增分享 CRUD）
- `frontend/src/pages/SharedConversation.tsx`（新增只读分享页）
- `frontend/src/components/chat/`（新增分享/导出按钮）

---

### 3.4 前端密码重置页面

**现状**：`PasswordResetForm` 组件已存在于 `frontend/src/components/auth/PasswordResetForm.tsx`，但路由 `/reset-password` 没有在 `router.tsx` 中注册。用户只能通过 API 调接口，没有可访问的页面。

**需求描述**：
- 注册 `/reset-password` 路由，指向 PasswordReset 页面
- 支持两步：输入邮箱 → 接收验证码 → 输入新密码
- 登录页增加"忘记密码"链接

**涉及文件**：
- `frontend/src/router.tsx`（注册路由）
- `frontend/src/pages/ResetPassword.tsx`（新增或复用 PasswordResetForm）

---

## 四、P1 —— 运营效率与用户粘性

### 4.1 用户管理后台

**现状**：Admin 后台可以管理文章、分类、评论、知识文档，但不能管理用户。没有用户列表、被封禁用户无法从后台操作。

**需求描述**：
- 后台新增"用户管理"菜单项
- 用户列表：支持按用户名/邮箱搜索、按角色/状态筛选、分页
- 操作：封禁/解封、修改角色（用户 ↔ 管理员）、查看用户发表的评论和对话

**数据模型**：
- 用户表已有 `Status`（active/banned）和 `Role`（user/admin）字段，无需额外字段

**涉及文件**：
- `backend/internal/handler/admin/` 或新建 `backend/internal/handler/user/admin_user.go`
- `backend/internal/service/user/`（新增管理员用户服务方法）
- `frontend/src/views/admin/users/UserList.tsx`（新增）
- `frontend/src/components/admin/AdminLayout.tsx`（导航新增条目）

---

### 4.2 全文搜索

**现状**：文章列表和知识文档列表仅支持 `LIKE` 关键字匹配。搜索能力很弱——不支持分词、不检索文章正文、不做相关度排序。

**需求描述**：

**方案 A（轻量）**：MySQL 全文索引
- 对 `articles.title` 和 `articles.content` 建 FULLTEXT 索引
- 搜索接口 `GET /api/search?q=xxx&type=articles|knowledge|all`
- 搜索结果页展示标题、摘要高亮、相关度排序

**方案 B（推荐）**：Meilisearch
- 部署 Meilisearch 实例（已有 Docker Compose，可追加）
- 文章发布/更新时同步索引到 Meilisearch
- 支持中文分词、模糊搜索、搜索建议、分面筛选
- 搜索结果时延 < 10ms（vs MySQL FULLTEXT 的 100-500ms）

**涉及文件**：
- 搜索服务：`backend/internal/service/search/`（新增）
- 搜索接口：`backend/internal/handler/search/`（新增，公共接口）
- 索引同步：文章和知识文档的 service 层调用搜索服务写入索引
- 前端搜索页：`frontend/src/pages/Search.tsx`（新增）

---

### 4.3 站内通知系统

**现状**：仅评论回复有邮件通知。用户被回复后没有站内通知提醒，也无法在站内查看历史通知。

**需求描述**：
- 通知类型：评论回复、评论被点赞、文章被评论（作者）、AI 回答有新交互、系统公告
- 前端导航栏增加通知铃铛（未读数角标）
- 通知列表页：已读/未读筛选、全部标为已读、点击跳转到对应内容
- 通知产生后可选是否同时发邮件（用户可在设置中关闭）

**数据模型**：
```sql
CREATE TABLE notifications (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id     BIGINT NOT NULL,
    type        VARCHAR(32) NOT NULL,   -- comment_reply, comment_like, article_comment, system
    title       VARCHAR(255),
    content     TEXT,
    link        VARCHAR(512),
    is_read     TINYINT DEFAULT 0,
    created_at  DATETIME DEFAULT NOW(),
    INDEX idx_user_read (user_id, is_read, created_at)
);
```

**涉及文件**：
- `backend/internal/model/notification.go`（新增）
- `backend/internal/service/notification/`（新增）
- `backend/internal/handler/notification/`（新增）
- `frontend/src/components/common/NotificationBell.tsx`（新增）

---

### 4.4 知识文档批量导入

**现状**：知识文档只能由 ThinkTank AI 在研究过程中自动创建。管理员无法手动添加或批量导入外部知识。

**需求描述**：
- 后台知识文档列表页新增"手动创建"按钮，支持填写标题、摘要、内容、来源 URL
- 支持 Markdown 文件批量上传导入（拖拽一批 .md 文件 → 自动创建对应文档）
- 导入后自动触发向量化，使其可被 RAG 检索

**涉及文件**：
- `backend/internal/handler/knowledge/knowledge_document.go`（新增创建/批量导入接口）
- `backend/internal/service/knowledge/`（新增手动创建服务方法）
- `frontend/src/views/admin/knowledge-documents/`（新增创建/导入 UI）

---

### 4.5 文章标签系统

**现状**：文章只能归属到一个分类。没有标签机制，无法做更细粒度的主题标注和交叉检索。

**需求描述**：
- 新增标签模型，支持多对多关联
- 文章编辑页新增标签输入（自动补全已有标签、自由创建新标签）
- 文章详情页显示标签，点击标签可筛选该标签下的所有文章
- 首页可显示标签云或热门标签

**数据模型**：
```sql
CREATE TABLE tags (
    id   BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(64) UNIQUE NOT NULL
);
CREATE TABLE article_tags (
    article_id BIGINT NOT NULL,
    tag_id     BIGINT NOT NULL,
    PRIMARY KEY (article_id, tag_id)
);
```

---

### 4.6 AI 回答反馈

**现状**：AI 回答没有任何反馈机制（点赞/点踩/评论）。没有 RLHF 数据可以用于优化回答质量。

**需求描述**：
- 每条 AI 回答消息下方显示"有帮助"和"没帮助"按钮
- 点踩时弹出可选的反馈理由（不准确、不完整、不相关、格式问题等）
- 反馈数据存入数据库，后台统计面板可查看 AI 回答质量趋势
- 可选：多次点踩的回答可标记为需要人工审查

**数据模型**：
```sql
CREATE TABLE chat_feedbacks (
    id         BIGINT PRIMARY KEY AUTO_INCREMENT,
    message_id BIGINT NOT NULL,
    user_id    BIGINT NOT NULL,
    rating     TINYINT NOT NULL,   -- 1=有帮助, -1=没帮助
    reason     VARCHAR(64),
    created_at DATETIME DEFAULT NOW(),
    UNIQUE INDEX idx_msg_user (message_id, user_id)
);
```

---

## 五、P2 —— 平台能力扩展

### 5.1 公共知识库页面

**需求描述**：
- 新增公共路由 `/knowledge`，展示已审批的知识文档列表
- 每个文档卡片显示标题、摘要、来源、标签
- 支持分类筛选和搜索
- 文档详情页渲染为可读的文章格式

**涉及文件**：
- `frontend/src/pages/KnowledgeBase.tsx`（新增）
- 新增 `GET /api/knowledge-documents/public` 接口
- `frontend/src/components/home/Header.tsx`（导航增加知识库入口）

---

### 5.2 RSS / Atom Feed

**需求描述**：
- 生成全站 RSS feed（`/rss.xml`）
- 文章发布/更新时自动更新 feed
- 可选按分类生成分类专属 feed

---

### 5.3 SEO 完善

**当前问题**：文章页面无 meta 标签优化、无结构化数据。

**需求描述**：
- 每个文章页面注入 `meta description`（使用文章摘要）
- 注入 `og:title`、`og:description`、`og:image`（使用文章封面图）
- 注入 JSON-LD 结构化数据（`Article` schema）
- 自动生成 `sitemap.xml`
- 配置 `robots.txt`

**涉及文件**：
- `frontend/index.html`（注入 meta / og 标签）
- `backend/cmd/server/bootstrap_http.go`（新增 `/sitemap.xml`、`/robots.txt` 路由）

---

### 5.4 评论功能增强

**需求描述**：
- 评论支持 Markdown 语法（代码块、引用、链接）
- 评论点赞（已有评论模型，新增 like_count 字段 + 点赞接口）
- 评论排序（最新/最热/最早）

---

### 5.5 AI 模型选择

**需求描述**：
- 对话页面增加模型选择器（下拉切换当前使用的 LLM）
- 后端根据用户选择的模型动态路由到对应的 provider
- 管理员可在设置中配置可用的模型列表及其参数

---

### 5.6 文章系列/合集

**需求描述**：
- 新增合集模型（title, description, cover_image）
- 文章可添加到合集（多对多，带排序序号）
- 合集页面展示：目录导航 + 上一篇/下一篇
- 首页可展示"推荐合集"

**数据模型**：
```sql
CREATE TABLE collections (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    title       VARCHAR(255) NOT NULL,
    description TEXT,
    cover_image VARCHAR(512),
    sort_order  INT DEFAULT 0,
    created_at  DATETIME DEFAULT NOW()
);
CREATE TABLE collection_articles (
    collection_id BIGINT NOT NULL,
    article_id    BIGINT NOT NULL,
    sort_order    INT DEFAULT 0,
    PRIMARY KEY (collection_id, article_id)
);
```

---

## 六、P3 —— 长期愿景

### 6.1 多语言完整支持

**需求描述**：
- 前端全量 i18n（当前仅首页部分翻译）
- 后端错误消息国际化（Accept-Language header）
- 文章内容不做翻译，但 UI 和交互文案全双语
- Admin 后台全翻译

---

### 6.2 Plugin 系统

**需求描述**：
- ThinkTank Agent 实现为可插拔 Plugin
- Plugin 通过定义标准接口（`type AgentPlugin interface { Name() string; Run(ctx, input) (output, error) }`）接入
- Plugin 可独立部署，通过 gRPC 或 HTTP 与主流程通信
- 管理员可在后台启用/禁用 Plugin

---

### 6.3 AI 多模态支持

**需求描述**：
- AI 对话支持图片输入（上传图片 → LLM 理解图片内容并回答）
- 知识库支持 PDF 文档 OCR 入库
- AI 回答支持生成图表（柱状图、饼图、流程图）

---

### 6.4 权限系统重构

**需求描述**：
- 从 `user` / `admin` 二元角色升级为 RBAC 权限模型
- 角色：超级管理员、编辑、作者、版主、普通用户
- 每个角色可配置权限（发布文章、管理评论、上传文件、审批知识文档等）
- 管理员后台可创建/编辑角色，分配权限

---

### 6.5 实时协作文档编辑

**需求描述**：
- 多人可同时编辑同一篇文章
- 基于 WebSocket 的 OT（Operational Transformation）或 CRDT 同步
- 显示在线协作者头像、光标位置
- 编辑历史可回溯每个人的变更

---

## 七、实施路线图

```
Phase 1（基础补缺）
├── 文章定时发布        (3.2)
├── 文章版本历史        (3.1)
├── 密码重置页面        (3.4)
└── AI 回答反馈         (4.6)

Phase 2（体验升级）
├── 对话分享与导出      (3.3)
├── 用户管理后台        (4.1)
├── 文章标签系统        (4.5)
├── 站内通知系统        (4.3)
└── SEO 完善            (5.3)

Phase 3（搜索与发现）
├── 全文搜索            (4.2)
├── 公共知识库页面      (5.1)
├── RSS Feed            (5.2)
├── 评论增强            (5.4)
└── 文章合集            (5.6)

Phase 4（平台化）
├── 知识文档批量导入    (4.4)
├── AI 模型选择          (5.5)
├── 多语言支持          (6.1)
└── 权限系统重构        (6.4)

Phase 5（前瞻）
├── Plugin 系统          (6.2)
├── AI 多模态           (6.3)
└── 实时协作编辑        (6.5)
```

---

> **文档版本**：v1.0
> **创建日期**：2026-06-06
> **相关文档**：`docs/2026-06-05-code-review-requirements.md`、`docs/2026-06-06-architecture-optimization.md`
