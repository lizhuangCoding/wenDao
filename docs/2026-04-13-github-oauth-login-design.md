# GitHub OAuth 登录设计文档

**日期**: 2026-04-13
**项目**: wenDao 个人博客平台
**主题**: GitHub OAuth 登录

## 1. 概述

本文档描述 wenDao 项目中 GitHub OAuth 登录能力的设计与当前实现，覆盖：

- 后端 GitHub OAuth 授权入口与回调流程；
- 前端登录/注册页中的 GitHub 登录入口；
- Cookie 为主的登录态恢复机制；
- GitHub 头像默认使用与后续同步策略；
- 本地开发配置项与联调要求；
- 当前实现的范围边界与后续可扩展点。

当前实现目标是：**在不引入前端专用 OAuth 回调页的前提下，让用户能够通过 GitHub 完成登录，并在前端自动恢复登录态，同时将 GitHub 头像纳入用户头像体系。**

---

## 2. 设计目标

### 2.1 核心目标

1. 支持用户通过 GitHub OAuth 完成登录；
2. 登录成功后后端通过 Cookie 写入认证状态；
3. 前端回到首页后，能够基于 Cookie 主动恢复登录态；
4. GitHub OAuth 新用户默认使用 GitHub 头像；
5. 后续再次 GitHub 登录时，按头像来源规则决定是否同步最新 GitHub 头像；
6. 登录页和注册页都提供 GitHub 登录入口，交互和视觉保持统一。

### 2.2 非目标

本次设计不包含以下能力：

- 本地账号与 GitHub 账号绑定/解绑；
- 一个用户同时绑定多个第三方 OAuth 提供商；
- 专门的前端 OAuth callback 页面；
- 第三方登录失败专属错误页；
- 用户自己手动配置“是否同步 GitHub 头像”；
- 完整 OAuth 账户中心。

---

## 3. 整体方案

采用 **“后端完成 OAuth 授权与回调 + Cookie 写入登录态 + 前端基于 `/auth/me` 主动恢复”** 的方案。

### 3.1 方案说明

1. 用户在前端登录页或注册页点击 GitHub 登录按钮；
2. 前端跳转到后端 `/api/auth/github`；
3. 后端生成 `state` 并重定向到 GitHub 授权页；
4. GitHub 授权后回调后端 `/api/auth/github/callback`；
5. 后端完成：
   - `state` 校验；
   - GitHub code 换 token；
   - 拉取 GitHub 用户资料；
   - 创建或更新本地用户；
   - 写入 `token` / `refresh_token` Cookie；
   - 跳转回前端站点 `site.url`；
6. 前端应用启动时主动调用 `/api/auth/me`；
7. 如果后端 Cookie 有效，则恢复当前用户状态并进入已登录视图。

### 3.2 选择该方案的原因

- 安全性更好，不通过 URL query 传递 access token；
- 与现有后端 JWT + Refresh Token 机制一致；
- 不需要额外新增前端 callback 页；
- 与当前项目的简单架构更匹配，改动更集中。

---

## 4. 后端设计

### 4.1 路由设计

GitHub OAuth 相关路由如下：

- `GET /api/auth/github`：发起 GitHub 授权；
- `GET /api/auth/github/callback`：处理 GitHub 回调；
- `GET /api/auth/me`：返回当前登录用户资料；
- `POST /api/auth/refresh`：刷新 access token；
- `POST /api/auth/logout`：登出。

相关代码位置：
- `backend/cmd/server/main.go`

### 4.2 配置设计

后端使用如下配置项：

```yaml
site:
  url: "http://localhost:3000"

oauth:
  github:
    client_id: ""
    client_secret: ""
    callback_url: "http://localhost:8089/api/auth/github/callback"
```

同时支持环境变量：

- `SITE_URL`
- `GITHUB_CLIENT_ID`
- `GITHUB_CLIENT_SECRET`
- `GITHUB_CALLBACK_URL`

相关代码位置：
- `backend/config/config.go`
- `backend/config/config.yaml`
- `backend/config/.env`

### 4.3 OAuth 授权入口

`GET /api/auth/github` 的处理流程：

1. 检查 GitHub OAuth 必需配置是否齐全；
2. 生成随机 `state`；
3. 将 `state` 写入 `oauth_state` Cookie；
4. 重定向到 GitHub 授权 URL。

这里 `oauth_state` 用于 CSRF 防护。Cookie 在 `release` 模式下会启用 `Secure`。

相关代码位置：
- `backend/internal/handler/user.go`
- `backend/internal/service/oauth.go`

### 4.4 OAuth 回调处理

`GET /api/auth/github/callback` 的处理流程：

1. 从 query 中获取 `code` 和 `state`；
2. 校验 Cookie 中的 `oauth_state`；
3. 清除 `oauth_state` Cookie；
4. 调用 GitHub OAuth 服务换取用户信息；
5. 调用用户服务完成本地用户创建/查询/更新；
6. 生成 access token；
7. 生成 refresh token；
8. 将 `token` 与 `refresh_token` 写入 Cookie；
9. 重定向回 `site.url`。

该流程中：
- 不再通过 URL query 传 token；
- 前端只需要在回到首页后调用 `/auth/me` 即可恢复状态。

相关代码位置：
- `backend/internal/handler/user.go`
- `backend/internal/service/user.go`
- `backend/internal/service/oauth.go`

### 4.5 GitHub 用户创建与登录规则

#### 首次 GitHub 登录
若系统中不存在 `(provider=github, oauthID=<github id>)` 对应用户，则创建新用户：

- `username`：默认使用 GitHub `login`；
- `email`：使用 GitHub 主邮箱；
- `oauth_provider` / `oauth_id`：写入第三方身份；
- `avatar_url`：使用 GitHub 头像；
- `avatar_source`：标记为 `github`；
- `role`：默认 `user`；
- `status`：默认 `active`。

#### 已有 GitHub 用户再次登录
若用户已存在：

- 校验账号状态；
- 根据头像来源判断是否同步 GitHub 最新头像；
- 重新生成 access token 并走正常登录返回流程。

### 4.6 GitHub 邮箱与唯一性处理

为了避免 OAuth 用户创建时依赖数据库报错来驱动业务逻辑，当前实现补了以下规则：

1. **如果 GitHub 用户资料无法获得邮箱，则直接失败**；
2. **如果该邮箱已被已有本地账户占用，则直接返回明确错误**；
3. **如果 GitHub `login` 与现有用户名冲突，则自动生成一个带 GitHub ID 后缀的用户名**；
4. 不做本地账号与 GitHub 账号自动合并。

这样可以保证：
- 错误更可控；
- 用户创建路径不依赖数据库唯一索引碰撞作为业务分支；
- 当前范围内不会引入账号绑定复杂度。

相关代码位置：
- `backend/internal/service/user.go`

---

## 5. 用户头像策略

### 5.1 头像字段

当前用户模型中与头像相关的核心字段：

- `avatar_url`
- `avatar_source`

其中 `avatar_source` 当前取值：

- `default`
- `github`
- `custom`

相关代码位置：
- `backend/internal/model/user.go`

### 5.2 默认头像规则

#### 普通邮箱注册用户
- 系统生成默认头像；
- `avatar_source = default`

#### GitHub OAuth 用户
- 默认使用 GitHub 头像；
- `avatar_source = github`

#### 用户手动上传站内头像
- 上传成功后更新 `avatar_url`；
- 同时标记 `avatar_source = custom`

### 5.3 GitHub 头像同步规则

后续用户再次通过 GitHub OAuth 登录时：

- 如果 `avatar_source == github`：同步最新 GitHub 头像；
- 如果 `avatar_source == default`：同步 GitHub 头像；
- 如果 `avatar_source == custom`：不覆盖用户自己上传的头像；
- 如果是历史数据且 `avatar_source` 为空：按可同步处理。

该规则保证了：
- GitHub 用户默认头像体验更合理；
- 用户自己上传头像后拥有更高优先级；
- 不会因再次 GitHub 登录而覆盖站内自定义头像。

---

## 6. 前端设计

### 6.1 登录/注册入口

前端在以下页面都提供 GitHub 登录入口：

- 登录页 `/login`
- 注册页 `/register`

入口位置统一为：

1. 表单字段
2. 主提交按钮
3. 分隔线
4. GitHub 登录按钮
5. 页面底部切换链接

这种布局更接近常见产品形态：
- 主表单保留主视觉优先级；
- GitHub 作为替代登录方式放在表单下方；
- 登录页与注册页视觉保持统一。

相关代码位置：
- `frontend/src/pages/Login.tsx`
- `frontend/src/pages/Register.tsx`
- `frontend/src/components/common/GitHubAuthButton.tsx`

### 6.2 GitHub 按钮组件

当前项目将 GitHub 入口抽成一个小的共享组件：

- `frontend/src/components/common/GitHubAuthButton.tsx`

职责：
- 渲染 GitHub 图标；
- 接收按钮文案；
- 接收点击事件；
- 统一边框、hover、深色模式和间距风格。

这样做的好处：
- 登录页和注册页样式一致；
- 页面代码更简洁；
- 后续若扩展其他第三方登录，更容易保持一致风格。

### 6.3 前端 OAuth 触发逻辑

前端不自己生成 GitHub 授权 URL，而是直接跳转到后端 OAuth 入口：

- `GET /api/auth/github`

由后端统一构造 GitHub 授权链接并发起重定向。

相关代码位置：
- `frontend/src/api/auth.ts`

---

## 7. 登录态恢复设计

### 7.1 问题背景

GitHub OAuth 回调成功后，后端通过 Cookie 设置登录态并跳转回前端。

如果前端只依赖本地 `localStorage access_token` 来判断是否已登录，那么 GitHub OAuth 完成后用户回到首页时，前端可能仍然显示未登录状态。

### 7.2 当前恢复策略

当前前端启动时会主动尝试请求 `/api/auth/me`：

- 如果已有 token，走正常的用户恢复逻辑；
- 如果没有 token，也会进行一次静默恢复；
- 如果后端 Cookie 仍有效，则恢复用户状态；
- 如果失败，则保持未登录，不触发跳转循环。

这样就实现了：
- Cookie 为主的 OAuth 登录态接管；
- 前端无需额外的 OAuth callback 页面；
- GitHub 登录后回到首页即可恢复已登录状态。

相关代码位置：
- `frontend/src/App.tsx`
- `frontend/src/store/authStore.ts`
- `frontend/src/api/client.ts`

### 7.3 Refresh Token 行为

前端仍保留 refresh token 刷新 access token 的逻辑：

- access token 失效后调用 `/api/auth/refresh`
- refresh token 通过 Cookie 续签
- 返回新的 access token 后继续重试请求

在实现中，已修复响应数据结构读取错误，避免 refresh 成功时因前端解包逻辑错误而误判失败。

相关代码位置：
- `frontend/src/api/client.ts`

---

## 8. 流式聊天与 Cookie 登录兼容

项目中存在一个特殊点：

- 大部分 API 通过 Axios 发起，并统一设置 `withCredentials`；
- AI 流式聊天使用原生 `fetch`。

若只依赖本地 `access_token` 而不携带 Cookie，则 GitHub OAuth 用户在“看起来已登录”的情况下，流式聊天仍可能因未认证而失败。

为此当前实现已补齐：

- 流式聊天 `fetch` 请求带上 `credentials: 'include'`；
- 若本地 token 存在，则保留 `Authorization` 头；
- 这样既兼容普通登录，也兼容 GitHub OAuth 的 Cookie 登录态。

相关代码位置：
- `frontend/src/api/chat.ts`

---

## 9. 安全与鲁棒性设计

### 9.1 state 校验

通过 `oauth_state` Cookie 与 GitHub 回调中的 `state` 做比对，防止 CSRF。

### 9.2 Cookie 安全

- 登录相关 Cookie 使用 `HttpOnly`；
- 在 `release` 模式下启用 `Secure`；
- 不通过 URL query 暴露 access token。

### 9.3 配置完整性校验

发起 GitHub OAuth 前，后端会检查：

- `client_id`
- `client_secret`
- `callback_url`

缺失时直接返回明确错误，而不是跳到一个无效的 GitHub 授权 URL。

### 9.4 头像同步保护

通过 `avatar_source` 避免第三方头像覆盖用户自己上传的站内头像。

### 9.5 用户唯一性保护

- 缺少邮箱直接失败；
- 邮箱冲突直接失败；
- 用户名冲突自动生成可用用户名。

---

## 10. 本地开发配置说明

本地开发推荐配置如下：

### 前端
- 前端地址：`http://localhost:3000`
- API Base URL：`http://localhost:8089/api`

### 后端
- 后端地址：`http://localhost:8089`
- GitHub callback URL：`http://localhost:8089/api/auth/github/callback`
- Site URL：`http://localhost:3000`

### GitHub OAuth App 后台
需要填写：

- Homepage URL：`http://localhost:3000`
- Authorization callback URL：`http://localhost:8089/api/auth/github/callback`

注意：
- GitHub 后台中的 callback URL 必须和后端配置中的 `GITHUB_CALLBACK_URL` 完全一致；
- 本地开发阶段使用 `localhost` 是可以的；
- callback URL 必须指向后端，而不是前端。

---

## 11. 相关文件

### 后端
- `backend/cmd/server/main.go`
- `backend/config/config.go`
- `backend/config/config.yaml`
- `backend/config/.env`
- `backend/internal/model/user.go`
- `backend/internal/handler/user.go`
- `backend/internal/handler/auth.go`
- `backend/internal/service/oauth.go`
- `backend/internal/service/user.go`
- `backend/internal/repository/user.go`

### 前端
- `frontend/src/App.tsx`
- `frontend/src/api/auth.ts`
- `frontend/src/api/client.ts`
- `frontend/src/api/chat.ts`
- `frontend/src/store/authStore.ts`
- `frontend/src/pages/Login.tsx`
- `frontend/src/pages/Register.tsx`
- `frontend/src/components/common/GitHubAuthButton.tsx`
- `frontend/src/i18n.ts`

### 测试
- `backend/internal/service/user_oauth_test.go`
- `backend/internal/handler/user_avatar_test.go`
- `backend/config/config_test.go`
- `backend/internal/repository/user_oauth_columns_test.go`

---

## 12. 当前实现结论

当前 GitHub OAuth 登录已经形成完整闭环：

1. 前端登录/注册页都有 GitHub 登录入口；
2. 后端可以正确发起 GitHub 授权并处理回调；
3. 登录成功后后端以 Cookie 写入登录态；
4. 前端首页可以基于 `/auth/me` 自动恢复登录状态；
5. GitHub 用户默认使用 GitHub 头像；
6. GitHub 头像同步规则已与用户自定义头像兼容；
7. 流式聊天与 Cookie 登录态已兼容；
8. 本地开发与 GitHub OAuth App 配置方式已经明确。

该设计在当前项目阶段兼顾了：
- 实现复杂度；
- 安全性；
- 用户体验；
- 与现有认证与头像体系的兼容性。

后续如需进一步扩展，可在此基础上增加：
- OAuth 失败页；
- 本地账号与 GitHub 账号绑定；
- 多第三方登录提供商；
- 用户自主控制第三方头像同步策略。
