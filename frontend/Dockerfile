# 依赖阶段：安装前端依赖。
FROM node:22-alpine AS deps

WORKDIR /app

# 国内服务器访问 npm 官方源较慢，默认使用 npmmirror。
# 如需覆盖，可在 docker compose build 时传入 --build-arg NPM_REGISTRY=...
ARG NPM_REGISTRY=https://registry.npmmirror.com
RUN npm config set registry ${NPM_REGISTRY}

# 先复制依赖清单，利用 Docker 缓存避免每次都重新 npm ci。
COPY package.json package-lock.json ./
RUN npm ci

# 构建阶段：把 React/Vite 项目构建成静态文件。
FROM deps AS builder

# 前端请求后端 API 的基础路径。默认 /api，交给 Nginx/Compose 代理。
ARG VITE_API_BASE_URL=/api
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

COPY . .
RUN npm run build

# 运行阶段：用 Nginx 提供静态文件和 API 反向代理。
FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html

# 前端容器内监听 80；是否暴露到公网由 compose 文件决定。
EXPOSE 80
