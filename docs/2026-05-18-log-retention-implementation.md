# 日志保留与轮转技术实现说明

本文记录问道博客项目日志保留、日志轮转、AI 聊天日志写入和 Docker 容器日志限制的技术实现。重点说明代码如何控制日志增长，而不是只记录使用方法。

## 1. 背景与目标

项目目前有三类主要日志：

1. 后端应用日志：Gin 请求日志、业务错误日志、启动日志等。
2. AI 聊天日志：多 Agent 协作流程、工具调用、最终完成状态等结构化事件。
3. Docker 容器日志：容器 stdout/stderr 被 Docker 的 `json-file` 日志驱动保存。

这三类日志如果不限制，会持续占用云服务器磁盘。因此本次改动的目标是：

- 限制单个日志文件大小。
- 限制轮转备份数量。
- 压缩历史备份。
- 清理超过保留天数的历史日志。
- 限制 Docker 容器 stdout/stderr 日志大小。

## 2. 配置入口

后端日志配置结构在 `backend/config/config.go` 的 `LogConfig` 中：

```go
type LogConfig struct {
    Level      string `mapstructure:"level"`
    Format     string `mapstructure:"format"`
    Output     string `mapstructure:"output"`
    MaxSizeMB  int    `mapstructure:"max_size_mb"`
    MaxBackups int    `mapstructure:"max_backups"`
    MaxAgeDays int    `mapstructure:"max_age_days"`
    Compress   bool   `mapstructure:"compress"`
}
```

默认值有三层：

1. `backend/config/config.yaml` 中的默认配置：

```yaml
log:
  level: "info"
  format: "console"
  output: "log/"
  max_size_mb: 100
  max_backups: 7
  max_age_days: 28
  compress: true
```

2. `LoadConfig()` 中通过 Viper 设置的代码兜底：

```go
viper.SetDefault("log.max_size_mb", 100)
viper.SetDefault("log.max_backups", 7)
viper.SetDefault("log.max_age_days", 28)
viper.SetDefault("log.compress", true)
```

3. 配置读取后的兜底校验：如果 `max_size_mb`、`max_backups`、`max_age_days` 被配置成 `0` 或负数，会恢复为 `100 / 7 / 28`。

环境变量绑定如下：

```env
LOG_MAX_SIZE_MB=100
LOG_MAX_BACKUPS=7
LOG_MAX_AGE_DAYS=28
LOG_COMPRESS=true
```

线上使用 Docker Compose 时，`docker-compose.prod.yml` 也给这些变量提供了默认值：

```yaml
LOG_MAX_SIZE_MB: ${LOG_MAX_SIZE_MB:-100}
LOG_MAX_BACKUPS: ${LOG_MAX_BACKUPS:-7}
LOG_MAX_AGE_DAYS: ${LOG_MAX_AGE_DAYS:-28}
LOG_COMPRESS: ${LOG_COMPRESS:-true}
```

## 3. 什么是日志轮转

日志轮转是指：当当前日志文件达到指定大小后，不再继续无限追加，而是把当前文件改名保存为历史备份，并创建一个新的同名日志文件继续写入。

例如当前写入：

```text
2026-05-18.log
```

当文件超过 `LOG_MAX_SIZE_MB=100` 后，日志库会把旧文件重命名为带时间戳的备份文件，然后继续创建新的：

```text
2026-05-18.log
2026-05-18-2026-05-18T10-30-00.000.log.gz
```

轮转解决的是“单个日志文件无限变大”的问题。它不是定时任务，也不是每天自动切换文件。本项目当前的日志文件名是在服务启动时按当天日期生成的，如果服务跨天一直不重启，日志仍会继续写入启动当天对应的文件；下一次服务重启时才会使用新的日期文件名。

## 4. lumberjack 是什么

`lumberjack` 是 Go 生态里常用的日志文件轮转库，依赖路径是：

```go
gopkg.in/natefinch/lumberjack.v2
```

它不是日志框架本身，而是一个“可轮转的文件写入器”。

在本项目里：

- `zap` 负责生成日志内容、编码日志字段、控制日志级别。
- `lumberjack` 负责把日志写进文件，并在文件过大时执行轮转。

核心配置如下：

```go
&lumberjack.Logger{
    Filename:   fullPath,
    MaxSize:    cfg.MaxSizeMB,
    MaxBackups: cfg.MaxBackups,
    MaxAge:     cfg.MaxAgeDays,
    Compress:   cfg.Compress,
}
```

含义：

- `Filename`：当前正在写入的日志文件。
- `MaxSize`：单个日志文件最大大小，单位 MB。
- `MaxBackups`：按大小轮转出来的备份文件最多保留几个。
- `MaxAge`：轮转备份最多保留多少天。
- `Compress`：是否压缩轮转后的备份。

## 5. 后端应用日志实现

后端应用日志在 `backend/cmd/server/bootstrap_infra.go` 的 `initLogger()` 中初始化。

当 `LOG_OUTPUT=stdout` 或为空时：

- 日志写到标准输出。
- 文件轮转不由后端应用处理。
- Docker 容器日志限制负责控制 stdout/stderr 的磁盘占用。
- 后端仍会尝试清理 AI 聊天日志目录中的过期日志。

当 `LOG_OUTPUT` 是文件目录或文件路径时：

1. `logOutputDir()` 解析日志目录。
2. `os.MkdirAll()` 确保日志目录存在。
3. `pruneExpiredLogFiles()` 清理超过保留天数的历史日志。
4. 使用启动当天日期生成应用日志文件名：

```go
todayFilename := time.Now().Format("2006-01-02") + ".log"
fullPath := filepath.Join(dir, todayFilename)
```

5. 创建 `lumberjack.Logger` 并交给 `zapcore.NewCore()`：

```go
fileWriter := &lumberjack.Logger{
    Filename:   fullPath,
    MaxSize:    cfg.MaxSizeMB,
    MaxBackups: cfg.MaxBackups,
    MaxAge:     cfg.MaxAgeDays,
    Compress:   cfg.Compress,
}
```

最终写入链路是：

```text
业务代码 / Gin 中间件
  -> zap.Logger
  -> zapcore.Core
  -> lumberjack.Logger
  -> log/YYYY-MM-DD.log
```

## 6. AI 聊天日志实现

AI 聊天日志在 `backend/internal/service/ai/ai_log.go` 中实现。

原先实现是：

```text
json.Encoder -> os.File
```

这种方式只会持续追加写入，没有大小轮转，也没有备份保留控制。

现在改成：

```text
json.Encoder -> lumberjack.Logger
```

核心函数是 `NewAILoggerWithRotation()`：

```go
func NewAILoggerWithRotation(logDir string, rotation LogRotationConfig) (AILogger, error)
```

它会创建如下文件：

```text
log/YYYY-MM-DD-ai-chat.log
```

并通过 `lumberjack.Logger` 控制：

- 单文件最大大小。
- 备份数量。
- 备份保留天数。
- 是否压缩备份。

AI 日志的初始化入口在 `backend/cmd/server/bootstrap_services.go`：

```go
aiEventLogger, err := service.NewAILoggerWithRotation(
    aiLogDir(cfg.Log.Output),
    service.LogRotationConfig{
        MaxSizeMB:  cfg.Log.MaxSizeMB,
        MaxBackups: cfg.Log.MaxBackups,
        MaxAgeDays: cfg.Log.MaxAgeDays,
        Compress:   cfg.Log.Compress,
    },
)
```

也就是说，AI 聊天日志和后端应用文件日志共用同一套保留策略。

AI 日志写入链路是：

```text
ThinkTank 多 Agent 流程
  -> AILogger.LogStage / AILogger.LogError
  -> json.Encoder
  -> lumberjack.Logger
  -> log/YYYY-MM-DD-ai-chat.log
```

## 7. 历史日志最多保留 28 天如何实现

单靠 `lumberjack` 不能完整解决本项目的历史日志保留问题。原因是项目会按启动日期生成不同的基础日志文件：

```text
2026-05-01.log
2026-05-02.log
2026-05-03-ai-chat.log
```

`lumberjack` 更擅长管理“当前文件按大小轮转出来的备份”，但不会自动扫描并删除所有不同日期的基础日志文件。因此额外实现了 `pruneExpiredLogFiles()`。

该函数在后端启动时执行，逻辑是：

1. 扫描日志目录。
2. 只识别项目自己生成的日志文件名。
3. 从文件名开头解析日期。
4. 计算保留截止日期。
5. 删除早于截止日期的日志。

识别规则覆盖：

```text
YYYY-MM-DD.log
YYYY-MM-DD-ai-chat.log
YYYY-MM-DD-时间戳.log
YYYY-MM-DD-ai-chat-时间戳.log
*.log.gz
```

不会处理无法匹配的文件，例如：

```text
random.log
manual-backup.txt
mysql-error.log
```

这样可以避免误删用户手工放进日志目录的其他文件。

需要注意：清理发生在后端启动时，不是后台定时任务。如果服务长期不重启，过期基础日志不会在当天自动删除；下一次重启时会被清理。

## 8. Docker 容器日志实现

生产环境 `LOG_OUTPUT=stdout`，后端应用日志会写到容器标准输出。此时真正占用宿主机磁盘的是 Docker 的 `json-file` 日志。

因此在 `docker-compose.prod.yml` 中增加了统一日志配置：

```yaml
x-logging: &default-logging
  driver: json-file
  options:
    max-size: ${DOCKER_LOG_MAX_SIZE:-20m}
    max-file: "${DOCKER_LOG_MAX_FILE:-5}"
```

并应用到：

- `caddy`
- `frontend`
- `backend`
- `mysql`
- `redis`

含义是：

- 单个 Docker 容器日志文件最大 `20m`。
- 每个容器最多保留 `5` 个日志文件。
- 默认最多约 `100MB / 容器`。

这些配置控制的是 Docker 保存的 stdout/stderr 日志，通常位于宿主机 Docker 数据目录下，例如：

```text
/var/lib/docker/containers/<container-id>/<container-id>-json.log
```

Docker 日志配置只有在容器重新创建后才会生效。因此线上更新后需要执行 `docker compose up -d --build` 或等价的重建命令。

## 9. 配置优先级

生产运行时，日志配置大致按以下优先级生效：

```text
服务器 .env.production
  > docker-compose.prod.yml 中的 ${VAR:-default}
  > 后端 config.yaml
  > 后端代码兜底默认值
```

其中：

- `DOCKER_LOG_MAX_SIZE`、`DOCKER_LOG_MAX_FILE` 只影响 Docker 容器日志。
- `LOG_MAX_SIZE_MB`、`LOG_MAX_BACKUPS`、`LOG_MAX_AGE_DAYS`、`LOG_COMPRESS` 影响后端文件日志和 AI 聊天日志。
- 如果 `LOG_OUTPUT=stdout`，普通后端应用日志由 Docker 日志轮转控制；AI 聊天日志仍由后端的 `lumberjack` 控制。

## 10. 测试覆盖

本次增加了以下测试：

- `backend/config/config_test.go`
  - 验证日志配置可以通过环境变量覆盖。
- `backend/cmd/server/log_cleanup_test.go`
  - 验证过期日志会被删除。
  - 验证未匹配的文件不会被误删。
  - 验证保留天数禁用时不会清理。
  - 验证 `LOG_OUTPUT=stdout` 时 AI 日志目录默认使用 `log/`。
- `backend/internal/service/ai/ai_log_test.go`
  - 验证 AI 日志可以通过轮转日志器写入 JSON 行。

完整后端测试命令：

```bash
cd backend
go test -count=1 ./...
```

Docker Compose 配置验证命令：

```bash
docker compose --env-file .env.production.example -f docker-compose.prod.yml config
```

## 11. 当前限制与后续可选优化

当前实现已经能解决日志长期增长问题，但仍有几个明确边界：

1. 应用日志和 AI 日志的基础文件名按服务启动日期生成，不会在午夜自动切换到新日期文件。
2. 超过 `max_age_days` 的基础日志清理发生在服务启动时，不是常驻定时任务。
3. 生产环境如果 `LOG_OUTPUT=stdout`，普通后端应用日志不走应用层 `lumberjack`，而是走 Docker `json-file` 轮转。
4. 当前 `docker-compose.prod.yml` 没有把 `/app/log` 映射到宿主机 volume；如果希望持久化 AI 聊天日志，需要额外挂载日志目录。

后续如果需要更强的日志治理，可以考虑：

- 使用专门的日志采集系统，例如 Loki、ELK、阿里云 SLS。
- 使用定时任务或后台 goroutine 做运行时过期清理。
- 引入按日期自动切换文件的日志 writer。
- 将 AI 聊天过程日志改为数据库持久化为主、文件日志为辅。
