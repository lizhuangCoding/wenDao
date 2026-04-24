# 问道博客平台 - 双 Token 认证与文章目录功能设计

## 概述

为问道博客平台实现两个功能：
1. 双 Token 认证机制（Access Token + Refresh Token）
2. 文章页面左侧目录侧边栏

## 功能一：双 Token 认证机制

### 1.1 架构设计

```
登录成功 → 返回 Access Token + Refresh Token (HttpOnly Cookie)
请求 API → 使用 Access Token（Header: Authorization: Bearer <token>）
Token 过期 → 使用 Refresh Token 换取新 Access Token
登出 → 清除 Refresh Token
```

### 1.2 配置变更

修改 `backend/config/config.yaml`：
```yaml
jwt:
  secret: ${JWT_SECRET}
  access_expire_hours: 1    # Access Token 过期时间：1小时
  refresh_expire_days: 7     # Refresh Token 过期时间：7天
```

### 1.3 后端实现

#### 1.3.1 JWT 工具增强

修改 `backend/internal/pkg/jwt/jwt.go`：
- 新增 `GenerateAccessToken()` - 生成 Access Token（短期）
- 新增 `GenerateRefreshToken()` - 生成 Refresh Token（长期）
- 新增 `ParseRefreshToken()` - 解析 Refresh Token
- 新增 `ValidateRefreshToken()` - 验证 Refresh Token（检查是否在黑名单）

#### 1.3.2 新增端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/auth/refresh` | POST | 使用 Refresh Token 换取新 Access Token |
| `/api/auth/logout` | POST | 登出（清除 Refresh Token） |

#### 1.3.3 登录响应变更

修改登录成功响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600
  }
}
```
Refresh Token 通过 HttpOnly Cookie 返回，Cookie 名称：`refresh_token`

#### 1.3.4 中间件调整

修改 `backend/internal/middleware/auth.go`：
- 保持现有 Access Token 验证逻辑
- 移除 Refresh Token 相关逻辑（单独处理）

### 1.4 前端实现

#### 1.4.1 认证状态管理

修改 `frontend/src/store/auth.ts`：
- 保持 LocalStorage 存储 Access Token
- Refresh Token 由后端通过 HttpOnly Cookie 管理

#### 1.4.2 请求拦截器

修改 `frontend/src/api/client.ts`：
- 保持从 LocalStorage 读取 Access Token
- 401 响应时自动调用刷新 Token 接口
- 刷新失败则跳转登录页

#### 1.4.3 登录/登出流程

- 登录成功后，后端设置 Refresh Token 到 HttpOnly Cookie
- 登出时调用 `/api/auth/logout`

---

## 功能二：文章目录侧边栏

### 2.1 布局设计

```
+------------------+----------------------------------------+
|                  |                                        |
|     目录侧边栏     |              文章内容                    |
|   (sticky left)  |           (max-width: reading)          |
|                  |                                        |
|  - 标题1         |  # 标题1                                |
|    - 标题1.1     |     内容...                             |
|    - 标题1.2     |                                        |
|  - 标题2         |  ## 标题1.1                             |
|    - 标题2.1     |     内容...                             |
|                  |                                        |
+------------------+----------------------------------------+
```

### 2.2 目录生成逻辑

从 Markdown 内容解析所有级别标题：
```typescript
interface TocItem {
  id: string;      // 锚点ID（slugify 标题文字）
  text: string;    // 标题文字
  level: number;   // 标题级别（1-6）
}
```

解析正则：
```typescript
const headingRegex = /^(#{1,6})\s+(.+)$/gm;
```

### 2.3 组件设计

#### 2.3.1 ArticleDetail 布局调整

修改 `frontend/src/pages/ArticleDetail.tsx`：
- 使用 Grid 布局：左侧目录 + 右侧文章
- 目录宽度：240px（桌面端）
- 移动端隐藏目录

```tsx
<div className="flex">
  {/* 目录侧边栏 */}
  <aside className="hidden lg:block w-60 shrink-0">
    <TableOfContents headings={headings} />
  </aside>

  {/* 文章内容 */}
  <article className="flex-1 max-w-reading">
    ...
  </article>
</div>
```

#### 2.3.2 TableOfContents 组件

新建 `frontend/src/components/article/TableOfContents.tsx`：

**Props：**
```typescript
interface Props {
  headings: TocItem[];
}
```

**功能：**
1. **粘性定位**：`position: sticky; top: 100px`
2. **独立滚动**：`max-height: calc(100vh - 200px); overflow-y: auto`
3. **锚点跳转**：点击后平滑滚动到对应标题
4. **同步高亮**：监听滚动，使用 IntersectionObserver 检测当前可见标题
5. **层级缩进**：根据 level 计算 padding-left

**高亮逻辑：**
```typescript
// 使用 IntersectionObserver 监听所有标题元素
// 设置 activeId 状态
// 目录项根据 activeId 高亮
```

#### 2.3.3 ArticleContent 组件调整

修改 `frontend/src/components/article/ArticleContent.tsx`：
- 为每个标题添加 ID（用于锚点跳转）
- 导出 headings 供父组件使用

```tsx
// 渲染标题时添加 ID
const renderHeading = (level: number, text: string) => {
  const id = slugify(text);
  return <h{level} id={id}>{text}</h{level}>;
};
```

---

## 关键技术决策

### 双 Token

1. **Refresh Token 存储**：HttpOnly Cookie，更安全
2. **刷新机制**：前端 401 时自动刷新，用户无感知
3. **黑名单机制**：Refresh Token 也加入黑名单，登出立即失效

### 目录

1. **解析时机**：客户端解析（从 ArticleContent 导出）
2. **锚点生成**：使用 slugify 将标题转为 ID
3. **滚动平滑**：`scroll-behavior: smooth`
4. **响应式**：移动端隐藏目录

---

## 验证方案

### 双 Token

1. 登录后检查 LocalStorage 有 Access Token
2. 检查 Cookie 有 refresh_token（HttpOnly）
3. 等待 Access Token 过期（可修改配置缩短时间测试）
4. 发起请求应自动刷新 Token
5. 登出后 Token 应失效

### 目录

1. 访问有 Markdown 标题的文章
2. 左侧应显示目录
3. 点击目录项应平滑滚动到对应位置
4. 滚动文章时目录应同步高亮
5. 移动端目录应隐藏
