package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger 创建日志记录中间件
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算耗时
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		method := c.Request.Method

		// 构建日志字段
		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
		}

		// 只在出错时记录额外信息
		if statusCode >= 400 || latency > 500*time.Millisecond {
			fields = append(fields, zap.String("ip", c.ClientIP()))
		}

		// 记录错误信息（如果有）
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// 根据状态码和耗时选择日志级别
		switch {
		case statusCode >= 500:
			logger.Error("Server error", fields...)
		case statusCode >= 400:
			logger.Warn("Client error", fields...)
		case latency > 500*time.Millisecond:
			logger.Warn("Slow request", fields...)
		// 默认不记录 INFO 日志，只记录错误和警告
		}
	}
}
