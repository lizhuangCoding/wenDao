# 用户自定义头像上传设计

## 概述
在现有博客系统中，用户注册后已经具备默认头像，OAuth 用户也会继承第三方头像；同时，后台文章编辑已经具备完整的图片上传能力，包括文件类型校验、大小限制、可选压缩、本地落盘以及 `uploads` 表记录。

本设计的目标是在**不破坏现有图片模块**的前提下，补齐“用户在个人中心自定义上传头像”的闭环：
- 用户初始仍然拥有默认头像；
- 用户可在个人中心上传并替换头像；
- 头像上传继续复用现有图片上传模块；
- 上传后的头像地址写回 `users.avatar_url`；
- 前端刷新页面后头像状态仍然正确。

## 目标
实现一个最小但完整的头像上传方案，使普通登录用户可以在个人中心替换头像，同时保持与现有图片上传模块的兼容性。

成功标准：
1. 新用户仍然拥有默认头像；
2. 登录用户可以在个人中心上传新头像；
3. 上传文件继续写入现有 `/uploads/...` 存储路径；
4. 上传记录继续写入 `uploads` 表；
5. 用户头像字段继续使用 `users.avatar_url`；
6. 前端刷新后，Header 与其他头像显示位置能继续展示新头像。

## 当前现状

### 已有能力
1. 用户模型已经有头像字段 `avatar_url`
   - `backend/internal/model/user.go`
2. 注册用户会自动写入默认头像
   - `backend/internal/service/user.go`
3. GitHub OAuth 用户会写入 GitHub 头像
   - `backend/internal/service/user.go`
4. 现有上传模块已经支持：
   - multipart 图片上传；
   - 文件大小校验；
   - MIME 类型校验；
   - 图片压缩与缩放；
   - 本地文件落盘；
   - `uploads` 表记录。
   - 相关代码：`backend/internal/handler/upload.go`、`backend/internal/service/upload.go`、`backend/internal/repository/upload.go`
5. 前端 `User` 类型和多个 UI 位置已经使用 `avatar_url`
   - `frontend/src/types/index.ts`
   - `frontend/src/components/common/Header.tsx`
   - `frontend/src/components/comment/CommentItem.tsx`
   - `frontend/src/pages/AIChat.tsx`

### 缺失能力
1. 目前只有管理员图片上传接口：`/api/admin/upload/image`
2. 普通用户没有个人中心页面，也没有头像上传入口
3. 后端虽然已经有 `UserService.UpdateAvatar(userID, avatarURL)`，但没有对应路由调用
4. 当前 `/api/auth/me` 返回字段不完整，不包含 `avatar_url`，前端刷新后无法可靠恢复头像状态

## 设计总结
采用“**复用现有上传模块 + 新增用户头像上传接口 + 新增个人中心页面**”的方案。

核心思路：
- 不重写上传逻辑；
- 不新增头像专用存储表；
- 不改变现有管理员图片上传能力；
- 头像上传只是把现有上传服务的结果进一步同步到 `users.avatar_url`；
- 个人中心只做最小闭环，不扩展成完整资料编辑系统。

## 范围

### 包含
- 为普通登录用户新增头像上传接口
- 为普通登录用户新增个人中心页面
- 补全 `/api/auth/me` 响应中的头像字段
- 上传成功后更新当前用户的 `avatar_url`
- 继续复用现有图片上传模块和上传记录表

### 不包含
- 头像裁剪
- 拖拽上传
- 头像历史版本管理
- 删除旧头像文件
- 用户名/邮箱编辑
- 头像审核
- 对象存储/CDN 改造

## 后端设计

### 1. 路由设计
保留现有管理员上传接口不变：
- `POST /api/admin/upload/image`

新增用户头像上传接口：
- `POST /api/users/me/avatar`

该接口应位于登录态用户可访问的受保护路由组下，而不是管理员路由组下。

### 2. `/api/auth/me` 响应补全
当前 `/api/auth/me` 已经用于前端启动时恢复当前登录用户状态，因此它必须返回完整的用户基础资料，而不仅仅是局部字段。

建议返回：
- `id`
- `email`
- `username`
- `role`
- `avatar_url`
- `expires_in`

这样可以保证：
- 用户更换头像后刷新页面仍能看到新头像；
- Header、评论区、聊天页等读取用户头像的 UI 不会因页面刷新退回默认展示逻辑。

### 3. 头像上传处理流程
`POST /api/users/me/avatar` 的处理流程如下：

1. 从请求中读取 multipart 文件字段 `file`；
2. 从认证上下文中获取当前 `user_id`；
3. 调用现有 `UploadService.UploadImage(file, header, userID)`；
4. 由上传服务完成：
   - 大小校验；
   - MIME 校验；
   - 可选压缩/缩放；
   - 文件保存到本地；
   - 在 `uploads` 表中创建记录；
5. 取得上传结果中的图片 URL；
6. 调用现有 `UserService.UpdateAvatar(userID, url)` 更新当前用户头像；
7. 返回最新头像地址，最好同时返回最新用户资料，便于前端直接同步状态。

### 4. 服务层复用策略
头像上传不新增独立的图片处理服务，优先复用现有能力：
- `UploadService.UploadImage(...)`
- `UserService.UpdateAvatar(...)`

实现上可以由 handler 顺序调用这两个 service。

原因：
- 改动最小；
- 保持现有上传模块边界不变；
- 避免为了单一需求引入不必要的新抽象。

### 5. 数据模型与存储兼容性
本设计不新增数据库表，也不修改已有头像字段设计。

继续使用：
- `users.avatar_url`：保存当前头像 URL；
- `uploads` 表：记录头像文件上传元数据；
- `/uploads/...`：作为静态文件访问路径。

这样做的结果是：
- 头像图片与后台文章图片共享同一上传设施；
- 运维、存储、清理策略仍可统一处理；
- 兼容现有图片模块的要求得到满足。

### 6. 依赖注入修正
当前代码中，`UserHandler` 已经声明了 `uploadService` 依赖，但创建 `UserHandler` 的入口尚未与该签名保持一致。这说明用户头像上传链路很可能只做了一半。

因此实现该需求时需要一并修正：
- `UserHandler` 构造参数与实际注入保持一致；
- 让用户 handler 可以真正访问上传服务。

这是完成头像上传的必要前提，而不是额外重构。

## 前端设计

### 1. 路由设计
新增一个用户个人中心页面，例如：
- `/profile`

该页面仅承担当前需求所需的最小资料展示与头像上传职责。

### 2. 页面功能
个人中心页面第一版仅包含：
- 当前头像预览；
- 用户名展示；
- 邮箱展示；
- “更换头像”上传入口；
- 上传结果反馈。

用户名和邮箱在本次设计中保持只读。

### 3. 上传交互
采用“选择即上传”的简单交互：
1. 用户点击上传按钮；
2. 选择本地图片；
3. 立即发起上传；
4. 上传成功后立即更新头像预览；
5. 同步刷新全局登录用户状态。

不引入“选择后再保存”的二次确认流程，避免增加不必要复杂度。

### 4. API 设计
前端保留现有管理员上传 API 不变。

新增用户头像上传 API 方法，例如：
- `uploadAvatar(file)`

其职责是：
- 发送 `multipart/form-data` 请求；
- 调用新的用户头像上传接口；
- 返回最新头像数据或最新用户资料。

### 5. 状态同步策略
头像上传成功后，推荐采用以下任一策略：

优先推荐：
- 后端直接返回最新用户资料；
- 前端直接更新 auth store 中的 `user`。

同时要求：
- `/api/auth/me` 也必须返回 `avatar_url`，保证页面刷新或重新进入应用时状态一致。

这样可以让以下位置自动受益：
- Header 用户头像
- 评论区用户头像
- AI 聊天页用户头像
- 其他依赖 `avatar_url` 的组件

### 6. 入口设计
用户进入个人中心的入口建议放在 Header 用户区域，例如：
- 点击头像/用户名进入；
- 或在用户菜单中增加“个人中心”。

第一版不需要新增多个入口，只保留一个稳定入口即可。

## 错误处理

### 后端
头像上传接口沿用现有上传模块的错误分类：
- 缺少文件 -> 参数错误；
- 文件类型不允许 -> 参数错误；
- 文件过大 -> 参数错误；
- 文件落盘失败 -> 服务端错误；
- 上传记录写入失败 -> 服务端错误；
- 头像字段更新失败 -> 服务端错误。

如果文件上传成功但数据库写入上传记录失败，继续沿用现有清理策略，删除已写入的本地文件。

### 前端
页面最小反馈策略：
- 上传中：禁用按钮，显示上传中状态；
- 上传成功：提示“头像更新成功”；
- 上传失败：
  - 文件类型不支持 -> 明确提示；
  - 文件过大 -> 明确提示；
  - 其他错误 -> 统一提示“头像上传失败，请重试”。

不引入复杂回滚逻辑。

## 默认头像策略
保持现有默认头像逻辑不变：
- 普通注册用户继续使用当前默认头像生成逻辑；
- GitHub OAuth 用户继续使用 GitHub 返回头像；
- 当用户主动上传头像后，用新的站内 `/uploads/...` 地址覆盖 `avatar_url`。

这样可以兼容现有存量数据，也避免无意义迁移。

## 影响文件

### 后端
建议修改：
- `backend/cmd/server/main.go`
- `backend/internal/handler/user.go`
- `backend/internal/handler/auth.go`
- `backend/internal/service/user.go`（如需补充组合逻辑或返回值处理）

预计复用，不做结构性改造：
- `backend/internal/handler/upload.go`
- `backend/internal/service/upload.go`
- `backend/internal/repository/upload.go`
- `backend/internal/model/user.go`
- `backend/internal/model/upload.go`

### 前端
建议新增或修改：
- `frontend/src/router.tsx`
- `frontend/src/api/auth.ts` 或新增用户资料 API 文件
- `frontend/src/api/upload.ts` 或新增用户头像上传 API 文件
- `frontend/src/store/authStore.ts`
- `frontend/src/components/common/Header.tsx`
- 新增个人中心页面文件（例如 `frontend/src/pages/Profile.tsx`）

## 测试计划

### 后端验证
1. 登录用户上传合法头像图片成功；
2. 文件成功写入本地存储目录；
3. `uploads` 表新增记录；
4. `users.avatar_url` 成功更新；
5. 非法 MIME 类型被拒绝；
6. 超过大小限制的文件被拒绝；
7. 未登录访问头像上传接口返回未授权；
8. `/api/auth/me` 返回中包含 `avatar_url`。

### 前端验证
1. 用户进入个人中心可看到当前头像；
2. 上传新头像后页面立即显示新头像；
3. Header 中头像同步更新；
4. 刷新页面后头像仍然正确；
5. 上传失败时展示明确错误提示；
6. 原有管理员文章图片上传功能保持可用。

## 验收标准
当满足以下条件时，设计视为成功：
- 新用户注册后仍有默认头像；
- 普通登录用户可以在个人中心上传头像；
- 头像上传继续经过现有图片上传模块；
- 上传记录继续保存在 `uploads` 表；
- 当前用户 `avatar_url` 成功更新；
- 页面刷新后前端仍能恢复并展示新头像；
- 现有管理员图片上传功能不受影响。
