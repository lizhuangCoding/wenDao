# 双 Token 认证与文章目录实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现双 Token 认证机制（Access Token + Refresh Token）和文章页面左侧目录侧边栏

**Architecture:**
- 双 Token: Access Token(1小时) 存 LocalStorage，Refresh Token(7天) 存 HttpOnly Cookie
- 目录: 左侧 sticky 定位，从 Markdown 标题提取，支持锚点跳转和滚动高亮

**Tech Stack:**
- 后端: Go + Gin + JWT
- 前端: React + TypeScript

---

## 文件结构

### 双 Token 认证
- 修改: `backend/config/config.yaml` - 添加 access_expire_hours, refresh_expire_days
- 修改: `backend/config/config.go` - 添加新配置结构
- 修改: `backend/internal/pkg/jwt/jwt.go` - 新增 Access/Refresh Token 生成和解析
- 修改: `backend/internal/handler/auth.go` - 新增 refresh, logout 端点
- 修改: `backend/internal/service/user.go` - 登录返回双 Token
- 修改: `backend/internal/middleware/auth.go` - 调整 Token 验证逻辑
- 修改: `frontend/src/api/client.ts` - 401 自动刷新 Token
- 修改: `frontend/src/store/auth.ts` - 移除 Token 过期时间存储

### 文章目录
- 新建: `frontend/src/components/article/TableOfContents.tsx` - 目录组件
- 修改: `frontend/src/components/article/ArticleContent.tsx` - 添加标题 ID
- 修改: `frontend/src/pages/ArticleDetail.tsx` - 布局调整

---

## 实现步骤

### 阶段一：后端双 Token 认证

#### Task 1: 配置变更

- [ ] **Step 1: 修改 config.yaml 添加新配置**

修改 `backend/config/config.yaml`，JWT 配置部分：
```yaml
jwt:
  secret: ${JWT_SECRET}
  access_expire_hours: 1    # Access Token 过期时间：1小时
  refresh_expire_days: 7     # Refresh Token 过期时间：7天
```

- [ ] **Step 2: 修改 config.go 添加新配置结构**

修改 `backend/config/config.go`，JWTConfig 结构体：
```go
type JWTConfig struct {
    Secret           string `mapstructure:"secret"`
    AccessExpireHours int   `mapstructure:"access_expire_hours"`
    RefreshExpireDays  int   `mapstructure:"refresh_expire_days"`
}
```

- [ ] **Step 3: 提交**

```bash
git add backend/config/config.yaml backend/config/config.go
git commit -m "feat: add access and refresh token config"
```

#### Task 2: JWT 工具增强

- [ ] **Step 1: 修改 jwt.go 新增双 Token 函数**

修改 `backend/internal/pkg/jwt/jwt.go`：
```go
// GenerateAccessToken 生成 Access Token（短期）
func GenerateAccessToken(userID int64, role string, secret string, expireHours int) (string, error) {
    claims := Claims{
        UserID: userID,
        Role:   role,
        TokenType: "access",
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}

// GenerateRefreshToken 生成 Refresh Token（长期）
func GenerateRefreshToken(userID int64, role string, secret string, expireDays int) (string, error) {
    claims := Claims{
        UserID: userID,
        Role:   role,
        TokenType: "refresh",
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireDays) * 24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}

// ParseToken 解析并验证 token（兼容新旧）
func ParseToken(tokenString string, secret string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(secret), nil
    })
    if err != nil {
        return nil, err
    }
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }
    return nil, fmt.Errorf("invalid token")
}

// ValidateRefreshToken 验证 Refresh Token
func ValidateRefreshToken(tokenString string, secret string) (*Claims, error) {
    claims, err := ParseToken(tokenString, secret)
    if err != nil {
        return nil, err
    }
    // 检查 token 类型
    if claims.TokenType != "refresh" {
        return nil, fmt.Errorf("invalid token type")
    }
    return claims, nil
}
```

同时修改 Claims 结构体添加 TokenType 字段：
```go
type Claims struct {
    UserID     int64  `json:"user_id"`
    Role       string `json:"role"`
    TokenType  string `json:"token_type"`  // 新增：access 或 refresh
    jwt.RegisteredClaims
}
```

- [ ] **Step 2: 编译验证**

```bash
cd backend && go build -o wenDao-server ./cmd/server/main.go
```

- [ ] **Step 3: 提交**

```bash
git add backend/internal/pkg/jwt/jwt.go
git commit -m "feat: add access and refresh token generation"
```

#### Task 3: 新增认证端点

- [ ] **Step 1: 创建 auth handler**

修改 `backend/internal/handler/user.go` 或新建 `backend/internal/handler/auth.go`：
```go
package handler

import (
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
    "wenDao/internal/pkg/jwt"
    "wenDao/internal/pkg/response"
    "wenDao/internal/service"
)

// AuthHandler 认证处理器
type AuthHandler struct {
    userService  service.UserService
    jwtSecret    string
    accessExpire int
    refreshExpire int
}

func NewAuthHandler(userService service.UserService, jwtSecret string, accessExpire, refreshExpire int) *AuthHandler {
    return &AuthHandler{
        userService:   userService,
        jwtSecret:     jwtSecret,
        accessExpire:  accessExpire,
        refreshExpire: refreshExpire,
    }
}

// RefreshTokenRequest 刷新 Token 请求
type RefreshTokenRequest struct {
    RefreshToken string `json:"refresh_token"`
}

// Refresh 刷新 Access Token
func (h *AuthHandler) Refresh(c *gin.Context) {
    var req RefreshTokenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.InvalidParams(c, "Invalid request")
        return
    }

    // 验证 Refresh Token
    claims, err := jwt.ValidateRefreshToken(req.RefreshToken, h.jwtSecret)
    if err != nil {
        response.Unauthorized(c, "Invalid or expired refresh token")
        return
    }

    // 检查 Refresh Token 是否在黑名单
    isBlacklisted, _ := jwt.IsTokenBlacklisted(nil, req.RefreshToken)
    if isBlacklisted {
        response.Unauthorized(c, "Refresh token has been revoked")
        return
    }

    // 生成新的 Access Token
    accessToken, err := jwt.GenerateAccessToken(claims.UserID, claims.Role, h.jwtSecret, h.accessExpire)
    if err != nil {
        response.InternalError(c, "Failed to generate access token")
        return
    }

    response.Success(c, gin.H{
        "access_token": accessToken,
        "expires_in":   h.accessExpire * 3600,
    })
}

// Logout 登出
func (h *AuthHandler) Logout(c *gin.Context) {
    // 获取 Refresh Token（从 Cookie 或 Body）
    refreshToken := c.GetHeader("X-Refresh-Token")
    if refreshToken == "" {
        cookie, err := c.Cookie("refresh_token")
        if err != nil {
            refreshToken = ""
        } else {
            refreshToken = cookie
        }
    }

    // 将 Refresh Token 加入黑名单
    if refreshToken != "" {
        // 获取过期时间
        claims, err := jwt.ParseToken(refreshToken, h.jwtSecret)
        if err == nil && claims != nil {
            remainingTime := time.Until(claims.ExpiresAt.Time)
            if remainingTime > 0 {
                jwt.AddToBlacklist(nil, refreshToken, remainingTime)
            }
        }
    }

    response.Success(c, gin.H{"message": "Logged out successfully"})
}
```

- [ ] **Step 2: 修改 user service 登录逻辑**

修改 `backend/internal/service/user.go` 中的登录方法：
```go
// Login 返回 Access Token（Refresh Token 由 handler 通过 Cookie 设置）
func (s *userService) Login(username, password string) (string, error) {
    // ... 现有逻辑 ...

    // 生成 Access Token
    accessToken, err := jwt.GenerateAccessToken(user.ID, user.Role, s.jwtSecret, s.jwtAccessExpire)
    if err != nil {
        return "", err
    }

    return accessToken, nil
}

// GenerateRefreshToken 为用户生成 Refresh Token
func (s *userService) GenerateRefreshToken(userID int64, role string) (string, error) {
    return jwt.GenerateRefreshToken(userID, role, s.jwtSecret, s.jwtRefreshExpire)
}
```

需要在 userService 结构体中添加新字段：
```go
type userService struct {
    userRepo     repository.UserRepository
    rdb          *redis.Client
    jwtSecret    string
    jwtAccessExpire int   // 新增
    jwtRefreshExpire int  // 新增
}
```

修改 NewUserService 签名：
```go
func NewUserService(userRepo repository.UserRepository, rdb *redis.Client, jwtSecret string, accessExpire, refreshExpire int) UserService {
    return &userService{
        userRepo:         userRepo,
        rdb:              rdb,
        jwtSecret:        jwtSecret,
        jwtAccessExpire:  accessExpire,
        jwtRefreshExpire: refreshExpire,
    }
}
```

- [ ] **Step 3: 修改 handler 登录返回双 Token**

修改 `backend/internal/handler/user.go` 中登录成功后：
```go
// 登录成功后
refreshToken, _ := s.userService.GenerateRefreshToken(user.ID, user.Role)
// 设置 Refresh Token 到 HttpOnly Cookie
c.SetCookie("refresh_token", refreshToken, s.jwtRefreshExpire*24*3600, "/", "", false, true)

response.Success(c, gin.H{
    "access_token": accessToken,
    "expires_in":   s.jwtAccessExpire * 3600,
})
```

- [ ] **Step 4: 修改 main.go 注册新路由**

修改 `backend/cmd/server/main.go`：
```go
// 在 setupRouter 中添加
authHandler := handler.NewAuthHandler(userService, cfg.JWT.Secret, cfg.JWT.AccessExpireHours, cfg.JWT.RefreshExpireDays)
auth.POST("/refresh", authHandler.Refresh)
auth.POST("/logout", authHandler.Logout)
```

修改 NewUserService 调用：
```go
userService := service.NewUserService(userRepo, rdb, cfg.JWT.Secret, cfg.JWT.AccessExpireHours, cfg.JWT.RefreshExpireDays)
```

- [ ] **Step 5: 编译验证**

```bash
cd backend && go build -o wenDao-server ./cmd/server/main.go
```

- [ ] **Step 6: 提交**

```bash
git add backend/internal/handler/auth.go backend/internal/service/user.go backend/cmd/server/main.go
git commit -m "feat: add refresh token endpoint and http-only cookie"
```

---

### 阶段二：前端双 Token 处理

#### Task 4: 前端请求拦截器

- [ ] **Step 1: 修改 api/client.ts**

修改 `frontend/src/api/client.ts`，添加 401 自动刷新逻辑：
```typescript
import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios';
import { message } from 'tdesign-react';

const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: 10000,
});

// 请求拦截器
request.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('access_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// 响应拦截器
let isRefreshing = false;
let requests: (() => void)[] = [];

request.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean };

    // 401 且不是刷新 Token 请求
    if (error.response?.status === 401 && !originalRequest.url?.includes('/auth/refresh') && !originalRequest._retry) {
      if (isRefreshing) {
        return new Promise((resolve) => {
          requests.push(() => {
            resolve(request(originalRequest));
          });
        });
      }

      originalRequest._retry = true;
      isRefreshing = true;

      try {
        const response = await request.post('/auth/refresh', {}, {
          // Cookie 会自动发送
        });

        const { access_token, expires_in } = response.data.data;
        localStorage.setItem('access_token', access_token);
        localStorage.setItem('token_expires_at', String(Date.now() + expires_in * 1000));

        originalRequest.headers.Authorization = `Bearer ${access_token}`;
        requests.forEach((cb) => cb());
        requests = [];

        return request(originalRequest);
      } catch (refreshError) {
        localStorage.removeItem('access_token');
        localStorage.removeItem('token_expires_at');
        message.error('登录已过期，请重新登录');
        window.location.href = '/login';
        return Promise.reject(refreshError);
      } finally {
        isRefreshing = false;
      }
    }

    return Promise.reject(error);
  }
);

export { request };
```

- [ ] **Step 2: 编译验证**

```bash
cd frontend && npm run build
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/api/client.ts
git commit -f -m "feat: add auto token refresh in request interceptor"
```

---

### 阶段三：文章目录功能

#### Task 5: ArticleContent 组件修改

- [ ] **Step 1: 修改 ArticleContent 添加标题 ID**

修改 `frontend/src/components/article/ArticleContent.tsx`：
```tsx
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { useMemo } from 'react';

// 生成 slug
const slugify = (text: string): string => {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, '')
    .replace(/[\s_-]+/g, '-')
    .replace(/^-+|-+$/g, '');
};

// 提取标题
export const extractHeadings = (content: string): TocItem[] => {
  const headingRegex = /^(#{1,6})\s+(.+)$/gm;
  const headings: TocItem[] = [];
  let match;

  while ((match = headingRegex.exec(content)) !== null) {
    const level = match[1].length;
    const text = match[2].trim();
    headings.push({
      id: slugify(text),
      text,
      level,
    });
  }

  return headings;
};

// 自定义渲染器，为标题添加 ID
const renderHeading = (level: number, text: string): string => {
  const id = slugify(text);
  return `<h${level} id="${id}">${text}</h${level}>`;
};

// 在组件中使用
export const ArticleContent: React.FC<{ content: string }> = ({ content }) => {
  // 使用 useMemo 缓存 headings
  const headings = useMemo(() => extractHeadings(content), [content]);

  return (
    <div className="prose prose-lg max-w-none">
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={{
        h1: ({ children }) => {
          const text = String(children);
          const id = slugify(text);
          return <h1 id={id} className="text-3xl font-bold mt-8 mb-4">{children}</h1>;
        },
        h2: ({ children }) => {
          const text = String(children);
          const id = slugify(text);
          return <h2 id={id} className="text-2xl font-bold mt-6 mb-3">{children}</h2>;
        },
        h3: ({ children }) => {
          const text = String(children);
          const id = slugify(text);
          return <h3 id={id} className="text-xl font-bold mt-4 mb-2">{children}</h3>;
        },
        h4: ({ children }) => {
          const text = String(children);
          const id = slugify(text);
          return <h4 id={id} className="text-lg font-bold mt-3 mb-2">{children}</h4>;
        },
        h5: ({ children }) => {
          const text = String(children);
          const id = slugify(text);
          return <h5 id={id} className="text-base font-bold mt-2 mb-1">{children}</h5>;
        },
        h6: ({ children }) => {
          const text = String(children);
          const id = slugify(text);
          return <h6 id={id} className="text-sm font-bold mt-2 mb-1">{children}</h6>;
        },
      }}>
        {content}
      </ReactMarkdown>
    </div>
  );
};
```

- [ ] **Step 2: 添加类型定义**

在 `frontend/src/types/article.ts` 中添加：
```typescript
export interface TocItem {
  id: string;
  text: string;
  level: number;
}
```

- [ ] **Step 3: 编译验证**

```bash
cd frontend && npm run build
```

- [ ] **Step 4: 提交**

```bash
git add frontend/src/components/article/ArticleContent.tsx frontend/src/types/article.ts
git commit -m "feat: add heading IDs to ArticleContent for TOC"
```

#### Task 6: 创建目录组件

- [ ] **Step 1: 创建 TableOfContents 组件**

新建 `frontend/src/components/article/TableOfContents.tsx`：
```tsx
import { useState, useEffect, useRef } from 'react';
import { TocItem } from '@/types/article';

interface Props {
  headings: TocItem[];
}

export const TableOfContents: React.FC<Props> = ({ headings }) => {
  const [activeId, setActiveId] = useState<string>('');
  const containerRef = useRef<HTMLDivElement>(null);

  // 点击锚点跳转
  const handleClick = (id: string) => {
    const element = document.getElementById(id);
    if (element) {
      element.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  };

  // 监听滚动，高亮当前章节
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setActiveId(entry.target.id);
          }
        });
      },
      {
        rootMargin: '-80px 0px -80% 0px',
        threshold: 0,
      }
    );

    headings.forEach((heading) => {
      const element = document.getElementById(heading.id);
      if (element) {
        observer.observe(element);
      }
    });

    return () => observer.disconnect();
  }, [headings]);

  if (headings.length === 0) {
    return null;
  }

  return (
    <div
      ref={containerRef}
      className="sticky top-20 max-h-[calc(100vh-120px)] overflow-y-auto pr-4"
    >
      <h4 className="text-sm font-semibold text-neutral-500 mb-3 uppercase tracking-wider">
        目录
      </h4>
      <ul className="space-y-2">
        {headings.map((heading) => (
          <li
            key={heading.id}
            style={{ paddingLeft: `${(heading.level - 1) * 12}px` }}
          >
            <button
              onClick={() => handleClick(heading.id)}
              className={`text-left text-sm transition-colors duration-200 ${
                activeId === heading.id
                  ? 'text-primary-600 font-medium'
                  : 'text-neutral-500 hover:text-neutral-700'
              }`}
            >
              {heading.text}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
};
```

- [ ] **Step 2: 编译验证**

```bash
cd frontend && npm run build
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/components/article/TableOfContents.tsx
git commit -m "feat: add TableOfContents component"
```

#### Task 7: ArticleDetail 布局调整

- [ ] **Step 1: 修改 ArticleDetail 页面**

修改 `frontend/src/pages/ArticleDetail.tsx`：
```tsx
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { articleApi } from '@/api';
import { Layout, Loading } from '@/components/common';
import { ArticleContent, extractHeadings } from '@/components/article';
import { TableOfContents } from '@/components/article/TableOfContents';
import { CommentList } from '@/components/comment';
import { formatDate } from '@/utils';
import { useAuth } from '@/hooks';
import { useMemo } from 'react';

export const ArticleDetail = () => {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { isAdmin } = useAuth();

  const { data: article, isLoading } = useQuery({
    queryKey: ['article', slug],
    queryFn: () => articleApi.getArticleBySlug(slug!),
    enabled: !!slug,
  });

  // 提取目录
  const headings = useMemo(() => {
    if (!article?.content) return [];
    return extractHeadings(article.content);
  }, [article?.content]);

  const likeMutation = useMutation({
    mutationFn: () => articleApi.likeArticle(article!.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['article', slug] });
    },
  });

  if (isLoading) {
    return (
      <Layout>
        <Loading />
      </Layout>
    );
  }

  if (!article) {
    return (
      <Layout>
        <div className="max-w-reading mx-auto px-4 py-12 text-center">
          <h1 className="text-2xl font-semibold text-neutral-700">文章不存在</h1>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="max-w-7xl mx-auto px-4 py-12">
        <div className="flex gap-8">
          {/* 左侧目录 */}
          <aside className="hidden lg:block w-60 shrink-0">
            <TableOfContents headings={headings} />
          </aside>

          {/* 右侧文章 */}
          <article className="flex-1 min-w-0 max-w-reading">
            {/* 文章头部 */}
            <header className="mb-8">
              {/* 返回按钮 */}
              <button
                onClick={() => navigate(-1)}
                className="flex items-center gap-1 text-neutral-500 hover:text-primary-600 mb-4 transition-colors"
              >
                <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                </svg>
                返回
              </button>

              {/* 分类标签 */}
              <div className="mb-4">
                <span className="text-sm px-3 py-1 bg-primary-50 text-primary-600 rounded">
                  {article.category.name}
                </span>
              </div>

              {/* 标题 */}
              <div className="flex items-center justify-between gap-4 mb-4">
                <h1 className="text-4xl font-bold text-neutral-700">{article.title}</h1>
                {isAdmin && (
                  <Link
                    to={`/admin/articles/edit/${article.id}`}
                    className="btn btn-secondary text-sm whitespace-nowrap"
                  >
                    编辑文章
                  </Link>
                )}
              </div>

              {/* 元信息 */}
              <div className="flex items-center gap-6 text-neutral-500">
                <span className="flex items-center gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                  </svg>
                  {article.author.username}
                </span>
                <span className="flex items-center gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  {formatDate(article.created_at)}
                </span>
                <span className="flex items-center gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                  </svg>
                  {article.view_count}
                </span>
                <button
                  onClick={() => likeMutation.mutate()}
                  disabled={likeMutation.isPending}
                  className="flex items-center gap-1 hover:text-primary-600 transition-colors"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
                  </svg>
                  {article.like_count}
                </button>
              </div>
            </header>

            {/* 封面图 */}
            {article.coverImage && (
              <div className="w-full mb-8 rounded-xl overflow-hidden">
                <img
                  src={article.coverImage}
                  alt={article.title}
                  className="w-full h-auto"
                />
              </div>
            )}

            {/* 文章内容 */}
            <ArticleContent content={article.content} />

            {/* 分隔线 */}
            <hr className="my-12 border-neutral-200" />

            {/* 评论区 */}
            <CommentList articleId={article.id} />
          </article>
        </div>
      </div>
    </Layout>
  );
};
```

- [ ] **Step 2: 编译验证**

```bash
cd frontend && npm run build
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/pages/ArticleDetail.tsx
git commit -m "feat: add TOC sidebar to article detail page"
```

---

## 验证步骤

### 双 Token 验证

1. 启动后端服务：`cd backend && ./wenDao-server`
2. 启动前端服务：`cd frontend && npm run dev`
3. 登录账户，检查：
   - LocalStorage 有 `access_token`
   - Cookie 有 `refresh_token`（HttpOnly）
4. 等待 1 小时或修改配置缩短时间测试 Access Token 过期
5. 发起请求应自动刷新 Token
6. 登出后 Token 应失效

### 目录验证

1. 访问包含 Markdown 标题的文章
2. 左侧应显示目录
3. 点击目录项应平滑滚动到对应位置
4. 滚动文章时目录应同步高亮
5. 移动端目录应隐藏

---

**Plan complete and saved to `docs/superpowers/plans/2026-04-07-double-token-and-toc-plan.md`. Two execution options:**

1. **Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

2. **Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
