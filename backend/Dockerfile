# 构建阶段：使用 Go 镜像下载依赖并编译后端二进制。
FROM golang:1.25-alpine AS builder

WORKDIR /src

# 阿里云等国内服务器访问 Go 官方源较慢，默认走国内代理。
# 如需覆盖，可在 docker compose build 时传入 --build-arg GOPROXY=...
ARG GOPROXY=https://goproxy.cn,direct
ARG GOSUMDB=sum.golang.google.cn
ENV GOPROXY=${GOPROXY}
ENV GOSUMDB=${GOSUMDB}

# 先复制依赖清单，利用 Docker 缓存避免每次都重新下载依赖。
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/wendao-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/wendao-init-admin ./scripts

# 运行阶段：只保留运行后端需要的最小环境，不带 Go 编译工具链。
FROM alpine:3.22

# ca-certificates 用于访问 HTTPS 外部服务，tzdata 用于正确处理时区。
# app 用户避免容器内以 root 身份运行应用。
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S app -G app

WORKDIR /app

COPY --from=builder /out/wendao-server /app/wendao-server
COPY --from=builder /out/wendao-init-admin /app/wendao-init-admin
COPY config/config.yaml /app/config/config.yaml

# uploads 和 log 是运行时目录；uploads 由 compose 挂载 volume 持久化。
RUN mkdir -p /app/uploads /app/log \
    && chown -R app:app /app

USER app

# 后端服务在容器内监听 8089。
EXPOSE 8089

CMD ["/app/wendao-server"]
