# Sub-project 2: 核心后端 API 设计规范

**日期**: 2026-04-04
**项目**: wenDao 博客平台

## 概述

实现博客平台的核心后端 API，包括用户系统、文章管理、分类管理、评论功能、图片上传。基于 Sub-project 1 完成的基础架构。

## 架构

**三层架构**: Handler → Service → Repository

**依赖关系**:
- Handler 依赖 Service
- Service 依赖 Repository
- Repository 依赖 GORM Model

## 实施计划

### Phase 1: 用户系统 (2-3天)

**Repository (`internal/repository/user.go`)**:
```go
type UserRepository interface {
    Create(user *model.User) error
    GetByID(id int64) (*model.User, error)
    GetByEmail(email string) (*model.User, error)
    GetByOAuth(provider string, oauthID string) (*model.User, error)
    Update(user *model.User) error
}
```

**Service (`internal/service/user.go`)**:
```go
type UserService interface {
    Register(email, password, username string) (*model.User, error)
    Login(email, password string) (token string, user *model.User, error)
    Logout(token string) error
    GetCurrentUser(userID int64) (*model.User, error)
}
```

**OAuth Service (`internal/service/oauth.go`)**:
```go
type OAuthService interface {
    GetGitHubAuthURL(state string) string
    ExchangeGitHubCode(code string) (*GitHubUserInfo, error)
}
```

**API 端点**:
- `POST /api/auth/register` - 邮箱注册
- `POST /api/auth/login` - 邮箱登录
- `POST /api/auth/logout` - 登出 (需认证)
- `GET /api/auth/me` - 获取当前用户 (需认证)
- `GET /api/auth/github` - GitHub OAuth 跳转
- `GET /api/auth/github/callback` - GitHub OAuth 回调

### Phase 2: 分类管理 (1天)

**Repository (`internal/repository/category.go`)**:
```go
type CategoryRepository interface {
    Create(category *model.Category) error
    GetByID(id int64) (*model.Category, error)
    GetBySlug(slug string) (*model.Category, error)
    List() ([]*model.Category, error)
    Update(category *model.Category) error
    Delete(id int64) error
    IncrementArticleCount(id int64) error
    DecrementArticleCount(id int64) error
}
```

**API 端点**:
- `GET /api/categories` - 分类列表 (公开)
- `GET /api/categories/:slug` - 分类详情 (公开)
- `POST /api/admin/categories` - 创建分类 (管理员)
- `PUT /api/admin/categories/:id` - 更新分类 (管理员)
- `DELETE /api/admin/categories/:id` - 删除分类 (管理员)

### Phase 3: 文章管理 (2-3天)

**Repository (`internal/repository/article.go`)**:
```go
type ArticleRepository interface {
    Create(article *model.Article) error
    GetByID(id int64) (*model.Article, error)
    GetBySlug(slug string) (*model.Article, error)
    List(filter ArticleFilter) ([]*model.Article, int64, error)
    Update(article *model.Article) error
    Delete(id int64) error
    IncrementViewCount(id int64) error
    IncrementCommentCount(id int64) error
    DecrementCommentCount(id int64) error
}
```

**Service 层逻辑**:
- 创建文章后生成 Slug (使用 hash.GenerateSlug)
- 发布文章时设置 published_at
- 缓存文章详情到 Redis

**API 端点**:
- `GET /api/articles` - 文章列表 (公开，只返回已发布)
- `GET /api/articles/:slug` - 文章详情 (公开)
- `POST /api/admin/articles` - 创建文章 (管理员)
- `PUT /api/admin/articles/:id` - 更新文章 (管理员)
- `DELETE /api/admin/articles/:id` - 删除文章 (管理员)
- `PATCH /api/admin/articles/:id/publish` - 发布文章 (管理员)
- `PATCH /api/admin/articles/:id/draft` - 转为草稿 (管理员)
- `GET /api/admin/articles` - 所有文章（含草稿，管理员）

### Phase 4: 评论功能 (2天)

**Repository (`internal/repository/comment.go`)**:
```go
type CommentRepository interface {
    Create(comment *model.Comment) error
    GetByID(id int64) (*model.Comment, error)
    GetByArticleID(articleID int64) ([]*model.Comment, error)
    Delete(id int64) error
}
```

**Service 层逻辑**:
- 两级评论限制：如果 parent_id 不为空，验证父评论必须是一级评论
- 创建评论时更新文章的 comment_count

**API 端点**:
- `GET /api/articles/:id/comments` - 文章评论列表 (公开)
- `POST /api/comments` - 发表评论 (需认证)
- `DELETE /api/comments/:id` - 删除评论 (本人或管理员)

### Phase 5: 图片上传 (1天)

**Service (`internal/service/upload.go`)**:
```go
type UploadService interface {
    UploadImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error)
}
```

**逻辑**:
- 验证文件类型（只允许 jpg/png/gif/webp）
- 验证文件大小（最大 10MB）
- 生成存储路径：`/uploads/{year}/{month}/{random}.{ext}`
- 保存文件到本地
- 记录到数据库

**API 端点**:
- `POST /api/admin/upload/image` - 上传图片 (管理员)

## 关键技术细节

### 1. JWT Token 管理
- 登录成功后生成 token，有效期 7 天
- Token 存储方式：HTTP-only Cookie 或 Authorization Header
- 登出时将 token 加入 Redis 黑名单

### 2. 密码加密
- 使用 bcrypt，cost=12
- 注册时加密，登录时验证

### 3. Slug 生成
- 文章创建后：`slug = hash.GenerateSlug(article.ID)`
- 生成后更新数据库

### 4. Redis 缓存
- 文章详情：`article:detail:{article_id}`, TTL 1小时
- 缓存更新：文章修改/删除时清除缓存

### 5. GitHub OAuth 流程
1. 用户点击登录 → 跳转到 GitHub 授权页
2. GitHub 回调 → 用 code 换取 access_token
3. 用 access_token 获取用户信息
4. 查询数据库：存在则登录，不存在则创建用户

### 6. 错误处理
- 使用统一的 response 包返回错误
- 敏感信息不暴露给前端（如数据库错误）

## 验证方案

每个 Phase 完成后测试对应的 API 端点：

**Phase 1 测试**:
```bash
# 注册
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"123456","username":"testuser"}'

# 登录
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"123456"}'

# 获取当前用户
curl http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer <token>"
```

## 时间估算

- Phase 1: 2-3 天
- Phase 2: 1 天
- Phase 3: 2-3 天
- Phase 4: 2 天
- Phase 5: 1 天
- **总计**: 8-10 天
