# 问道博客平台 - 前端

基于 React + TypeScript + Vite 构建的现代化技术博客前端应用。

## 技术栈

- **框架**: React 18 + TypeScript 5
- **构建工具**: Vite 5
- **样式**: Tailwind CSS 3
- **状态管理**: Zustand 4
- **数据请求**: TanStack Query v5 (React Query) + Axios
- **路由**: React Router v6
- **表单**: React Hook Form
- **Markdown**: react-markdown + rehype-highlight
- **动画**: Framer Motion

## 设计特点

- 🎨 自然绿白配色系统，清新自然
- 📖 680px 窄宽居中布局，优化阅读体验
- 🎭 智能隐藏导航栏，滚动自动收起
- ✨ 流畅的页面过渡和交互动画
- 🤖 内置 AI 技术助手
- 💬 两级评论系统
- 📱 响应式设计

## 快速开始

### 安装依赖

```bash
npm install
```

### 开发模式

```bash
npm run dev
```

访问 http://localhost:3000

### 构建生产版本

```bash
npm run build
```

### 预览生产构建

```bash
npm run preview
```

## 项目结构

```
src/
├── api/              # API 请求封装
├── components/       # 组件
│   ├── common/      # 通用组件
│   ├── article/     # 文章组件
│   ├── comment/     # 评论组件
│   └── admin/       # 管理后台组件
├── pages/           # 页面组件
├── store/           # Zustand 状态管理
├── hooks/           # 自定义 Hooks
├── utils/           # 工具函数
├── types/           # TypeScript 类型定义
├── styles/          # 全局样式
├── router.tsx       # 路由配置
├── App.tsx          # 应用根组件
└── main.tsx         # 入口文件
```

## 环境变量

复制 `.env.example` 为 `.env` 并配置：

```env
VITE_API_BASE_URL=http://localhost:8080/api
```

## 主要功能

### 用户端
- ✅ 文章浏览（分类筛选、分页）
- ✅ 文章详情（Markdown 渲染、点赞）
- ✅ 评论功能（两级评论）
- ✅ AI 技术助手（智能问答）
- ✅ 用户认证（登录、注册）

### 管理端
- 🚧 文章管理（待实现）
- 🚧 分类管理（待实现）
- 🚧 评论管理（待实现）

## 配色系统

### Primary（主色 - 绿色）
- `primary-500`: #22c55e（主要按钮、链接）
- `primary-600`: #16a34a（hover 状态）

### Neutral（中性色 - 带绿调的灰色）
- `neutral-50`: #fafbfa（页面背景）
- `neutral-700`: #2c3e2c（标题文字）
- `neutral-800`: #1a251a（正文文字）

## 动画规范

- **过渡时长**: 250ms（导航栏、页面切换）
- **卡片悬浮**: 150ms，向上移动 4px
- **按钮缩放**: hover 1.02x，active 0.98x
- **输入框焦点**: 200ms，绿色光晕

## API 集成

后端 API 基础路径：`/api`

主要端点：
- `GET /articles` - 获取文章列表
- `GET /articles/:slug` - 获取文章详情
- `POST /auth/login` - 用户登录
- `POST /auth/register` - 用户注册
- `POST /ai/chat` - AI 聊天
- `GET /articles/:id/comments` - 获取评论
- `POST /comments` - 创建评论

## 开发建议

1. 使用 React Query 进行数据请求和缓存
2. 使用 Zustand 管理全局状态（认证、UI）
3. 遵循 Tailwind CSS 的 utility-first 方法
4. 保持组件小而专注
5. 优先使用 TypeScript 类型安全

## 浏览器支持

- Chrome (最新版)
- Firefox (最新版)
- Safari (最新版)
- Edge (最新版)

## License

MIT
