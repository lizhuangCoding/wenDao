# Article Planet Hero Design

## 概览
把首页首屏改造成一个沉浸式 3D 文章星球。用户进入首页后第一眼看到可自动旋转、可拖拽的星球；每篇已发布文章都是星球表面的一个发光节点。用户悬停或点按节点可以预览文章信息，点击节点后跳转到现有文章详情页 `/article/:slug`。

实现优先使用成熟 3D 组件和 React 生态库，不手搓 WebGL 核心逻辑。推荐依赖：

- `three`
- `@react-three/fiber`
- `@react-three/drei`

第一版不引入复杂自定义 shader。视觉冲击力主要来自 3D 星球结构、发光节点、分类色彩、星云背景、缓慢自转、交互焦点和动效组合。

## 目标
- 首页首屏成为强视觉主入口，而不是普通文章列表页。
- 星球包含所有已发布文章，不受当前文章列表分页接口限制。
- 点击星球节点可以直接进入文章详情页。
- 支持桌面拖拽旋转、滚轮缩放和移动端触控旋转。
- 保留当前首页的标语、搜索、分类筛选和文章卡片列表，避免牺牲可读性和低端设备可用性。
- 3D 代码保持组件化、可替换、可测试，不把数据请求、3D 场景和覆盖层 UI 混在一个大文件里。

## 非目标
- 不重写文章详情页。
- 不改变文章 Markdown 渲染。
- 不做后台文章编辑器改动。
- 不做真实地理地球或地图投影。
- 不为第一版实现复杂图谱关系、AI 聚类或物理模拟。
- 不手写 WebGL 渲染循环、相机控制器或底层 shader 系统。

## 当前状态
前端首页在 `frontend/src/pages/Home.tsx`：

- 使用 `articleApi.getArticles` 获取分页文章。
- 每页展示 9 篇文章卡片。
- Hero 区是大标题、搜索框和装饰背景。
- 分类栏在 Hero 下方。
- 文章详情路由已经是 `/article/:slug`。

文章 API 在 `frontend/src/api/article.ts`：

- `getArticles` 调用 `/articles`。
- `getArticleBySlug` 调用 `/articles/slug/:slug`。

后端公开文章路由在 `backend/cmd/server/bootstrap_http.go`：

- `GET /api/articles`
- `GET /api/articles/:id`
- `GET /api/articles/slug/:slug`

后端文章读取链路为：

- handler: `backend/internal/handler/article/article.go`
- service: `backend/internal/service/article/article_read.go`
- repository: `backend/internal/repository/article/article.go`

当前分页服务会把 `pageSize` 限制到 100。首页星球要承载“所有文章”，因此不应复用当前分页列表接口直接拉大页码。

## 推荐方案
采用 `three` + `@react-three/fiber` + `@react-three/drei`。

原因：

- 与 React 组件模型匹配，适合现有 Vite React 项目。
- `Canvas`、相机、帧循环、事件拾取由成熟库处理。
- `drei` 提供 `OrbitControls`、`Html`、`Billboard`、`Stars`、`AdaptiveDpr`、`Preload`、`Instances` 等可复用组件。
- 可以做到足够炫酷，同时保持代码边界清晰。
- 后续如果要扩展后处理、贴图、曲线连线或动画，也能自然接入 Three 生态。

不选择 `react-globe.gl` 的原因是它更偏地图地球，文章星球的材质、节点层级、覆盖 UI 和自定义交互会受限。不选择纯 CSS 或 2D Canvas，因为视觉上限、深度排序和交互手感都不足。

## 用户体验设计
### 首屏布局
首页首屏使用接近全视口高度的沉浸区：

- 3D Canvas 铺满首屏，作为主视觉。
- 顶部继续由现有 `Header` 固定悬浮。
- 标语、搜索框、分类入口作为轻量覆盖层放在左下或左侧中部。
- 星球主体在桌面端偏右居中，保留标题阅读空间。
- 移动端标题覆盖层上移，星球居中并降低缩放，避免控件遮挡节点。
- 首屏底部露出下一段文章列表的起始区域，提示用户还能继续滚动。

### 星球视觉
星球由四层组成：

- 核心球体：半透明深色或玻璃质感的球体，用 `meshStandardMaterial` 或 `meshPhysicalMaterial` 实现。
- 文章节点：每篇文章一个发光点，贴在球面或略高于球面。
- 分类轨迹：按分类颜色生成轻量环线或短弧线，增强“知识星球”感。
- 背景星场：使用 `drei` 的 `Stars` 或轻量粒子背景，不使用装饰性 CSS 光球。

节点样式：

- 分类决定颜色。
- 置顶文章节点更大、更亮。
- 浏览量和评论数影响节点亮度或外圈脉冲，但不影响点击面积。
- 知识文档来源文章可以使用不同节点形态，例如带细外环。

### 交互
桌面端：

- 星球自动慢速旋转。
- 鼠标进入 Canvas 后仍保持低速旋转。
- 拖拽可旋转星球。
- 滚轮可在有限范围内缩放。
- 悬停节点显示浮层。
- 点击节点跳转 `/article/:slug`。

移动端：

- 单指拖拽旋转。
- 点按节点显示浮层。
- 再点一次浮层或节点进入文章详情。
- 缩放范围更保守，避免页面滚动和 3D 手势冲突。

键盘和可访问性：

- 3D Canvas 不作为唯一入口。
- 下方保留文章卡片列表。
- 搜索和分类按钮仍是普通 DOM 控件。
- 节点浮层标题和链接使用可读文本。

## 数据设计
### 新增公开接口
新增：

```http
GET /api/articles/orbit
```

用途：为首页 3D 星球返回所有已发布文章的轻量数据。

响应示例：

```json
{
  "data": [
    {
      "id": 1,
      "title": "文章标题",
      "slug": "article-slug",
      "summary": "摘要",
      "cover_image": "/uploads/example.png",
      "view_count": 120,
      "comment_count": 5,
      "is_top": false,
      "source_type": "manual",
      "category": {
        "id": 2,
        "name": "AI",
        "slug": "ai"
      },
      "created_at": "2026-05-24T12:00:00Z"
    }
  ],
  "total": 1
}
```

接口只返回星球需要的字段，不返回文章正文、作者完整信息或 Markdown 内容。

### 后端实现边界
新增 repository/service/handler 方法，而不是让前端用分页接口循环拉取。

建议命名：

- repository: `ListOrbitArticles() ([]*model.Article, error)`
- service: `ListOrbitArticles() ([]*model.Article, error)`
- handler: `ListOrbitArticles(c *gin.Context)`

查询条件：

- `status = published`
- 排序：`is_top DESC, published_at DESC, created_at DESC`
- 预加载 `Category`
- 不预加载 `Author`
- 不读取 `content`

如果 GORM 模型默认会读取所有列，repository 应使用 `Select` 明确选择轻量字段，减少首页 payload。

路由注意事项：

- `/api/articles/orbit` 需要加到公开文章路由中。
- 必须增加路由测试，确认它不会被 `/api/articles/:id` 参数路由误伤。

### 前端 API
在 `frontend/src/api/article.ts` 增加：

```ts
getArticleOrbit: () => request.get<ArticleOrbitResponse>('/articles/orbit')
```

新增类型：

- `ArticleOrbitItem`
- `ArticleOrbitResponse`

这些类型不复用完整 `Article`，避免组件误以为有 `content` 或完整 `author`。

## 前端架构
### 文件结构
建议新增：

- `frontend/src/components/home/ArticlePlanetHero.tsx`
- `frontend/src/components/home/ArticlePlanetScene.tsx`
- `frontend/src/components/home/ArticlePlanetNode.tsx`
- `frontend/src/components/home/ArticlePlanetOverlay.tsx`
- `frontend/src/components/home/articlePlanetLayout.ts`
- `frontend/src/components/home/articlePlanetLayout.test.mjs`

`Home.tsx` 只负责编排：

- 请求 slogan、categories、分页文章、orbit articles。
- 管理搜索、分类和分页状态。
- 把 orbit 数据传给 `ArticlePlanetHero`。
- 继续渲染原文章列表作为首屏后内容。

### 组件职责
`ArticlePlanetHero`：

- 首页首屏容器。
- 接收 slogan、分类、当前搜索状态、orbit articles。
- 组织 3D 场景和覆盖层。
- 处理 loading/error/empty 的首屏展示。

`ArticlePlanetScene`：

- 包含 `Canvas`、相机、灯光、`OrbitControls`、星场、核心球体和文章节点集合。
- 不直接请求 API。
- 不处理搜索表单提交。

`ArticlePlanetNode`：

- 渲染单个文章节点。
- 负责 hover/focus/click 事件。
- 点击时调用传入的 `onSelect(article)`，由上层使用 `useNavigate` 跳转。

`ArticlePlanetOverlay`：

- 渲染标语、搜索框、分类筛选、当前选中文章浮层。
- 是普通 DOM，不放进 Three 场景里。
- 保证移动端和暗色模式可读。

`articlePlanetLayout.ts`：

- 纯函数，把文章列表映射为稳定的 3D 点位和视觉权重。
- 适合用现有 `.test.mjs` 模式单测。

### 点位算法
使用 Fibonacci sphere 生成均匀球面点位：

- 输入：文章数组。
- 输出：每篇文章的 `x/y/z` 坐标、半径、颜色、权重。
- 按文章排序后的 index 和 id 生成稳定点位。
- 分类只影响颜色和轻微轨道偏移，不破坏整体均匀分布。

权重建议：

```text
base = 1
topBonus = is_top ? 0.7 : 0
viewBonus = min(log10(view_count + 1) * 0.25, 0.8)
commentBonus = min(log10(comment_count + 1) * 0.2, 0.5)
weight = base + topBonus + viewBonus + commentBonus
```

视觉大小和发光强度从 `weight` 派生。点击目标应设置最小尺寸，避免低浏览文章难以点击。

## 性能策略
### 加载
- `ArticlePlanetHero` 使用 React lazy 或组件内动态 import，避免后台页和文章详情页加载 3D 代码。
- Vite `manualChunks` 可把 `three`、`@react-three/fiber`、`@react-three/drei` 分到 `three-vendor`。
- 3D 场景数据与当前分页文章并行请求。

### 渲染
- 节点数量较少时可以直接渲染节点组件。
- 节点数量较多时使用 `drei` 的 `Instances`/`Instance` 批量渲染节点。
- 移动端降低星场密度，关闭重型发光效果。
- 使用 `AdaptiveDpr` 控制高 DPI 设备压力。
- Canvas 设置稳定尺寸，防止布局抖动。

### 降级
- 如果 WebGL 不可用，首屏显示静态渐变/星场背景和文章入口，不阻塞文章列表。
- 如果 orbit 接口失败，显示当前 slogan、搜索和文章列表，不让首页空白。
- 如果没有文章，星球区域显示空态，不渲染空 Canvas。

## 错误处理
- `GET /articles/orbit` 失败：首屏展示轻量错误状态，保留下方分页文章列表。
- 分类为空：使用默认颜色。
- 文章缺少摘要：浮层显示“暂无摘要”。
- 文章缺少 slug：节点不跳转，并在开发环境给出 console warning；正常数据应由后端保证 slug 存在。
- 3D 依赖加载失败：回退到普通 Hero。

## 测试计划
### 后端
新增或扩展 handler/service/repository 测试：

- `GET /api/articles/orbit` 只返回已发布文章。
- 返回字段不包含 `content`。
- 返回分类信息。
- 置顶文章排序在前。
- 路由测试确认 `/api/articles/orbit` 命中 orbit handler，而不是 `/api/articles/:id`。

运行：

```bash
cd backend
go test ./...
```

### 前端
新增纯函数测试：

- Fibonacci sphere 点位数量等于文章数量。
- 同一输入生成稳定坐标。
- 节点权重随置顶、浏览量、评论数增加。
- 分类颜色稳定映射。
- 空数组返回空布局。

运行：

```bash
cd frontend
npm run build
npm run lint
node src/components/home/articlePlanetLayout.test.mjs
```

如果新增测试脚本整合到现有模式，也可以把 Node 测试加入统一测试命令。

### 手动验收
- 首页首屏能看到 3D 文章星球。
- 星球自动旋转。
- 鼠标拖拽和移动端触控可以旋转星球。
- 悬停或点按节点能看到文章信息。
- 点击节点跳转到正确的 `/article/:slug`。
- 搜索和分类筛选仍能驱动下方文章列表。
- 暗色模式可读。
- 移动端没有文字重叠，首屏控件不遮挡主要节点。
- orbit 接口失败时首页仍能浏览文章。

## 验收标准
- 首页第一屏以 3D 文章星球为主视觉。
- 星球包含所有已发布文章的节点。
- 节点能稳定映射文章，并能跳转到现有文章详情页。
- 3D 实现基于 `three`、`@react-three/fiber`、`@react-three/drei`，不手搓底层 WebGL。
- 新接口只返回轻量文章数据，不返回正文。
- 原文章列表、搜索、分类筛选和分页不被移除。
- 桌面和移动端都有可用交互。
- 前后端构建和测试通过。

## 实施顺序建议
1. 增加 orbit 数据接口和后端测试。
2. 增加前端类型、API 方法和点位纯函数测试。
3. 安装 3D 依赖并配置 Vite chunk。
4. 实现 `ArticlePlanetHero` 和 3D 场景组件。
5. 将首屏 Hero 接入 `Home.tsx`，保留文章列表。
6. 做移动端、暗色模式、加载失败和 WebGL 降级验证。
