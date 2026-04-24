package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"wenDao/internal/pkg/response"
)

// RateLimitType 限流类型
type RateLimitType int

const (
	GlobalLimit RateLimitType = iota // 全局限流（所有请求）
	IPLimit                           // IP 限流
	UserLimit                         // 用户限流（需要认证）
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Type    RateLimitType             // 限流类型
	Limit   int                       // 请求数
	Window  time.Duration             // 时间窗口
	KeyFunc func(*gin.Context) string // 自定义 key 生成函数（可选）
}

// RateLimit 创建限流中间件
func RateLimit(rdb *redis.Client, config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()

		// 生成限流 key
		key := generateRateLimitKey(c, config)

		// 检查限流
		allowed, err := checkRateLimit(ctx, rdb, key, config.Limit, config.Window)
		if err != nil {
			// Redis 错误不影响业务，只记录日志
			c.Next()
			return
		}

		if !allowed {
			response.TooManyRequests(c, "Too many requests, please try again later")
			c.Abort()
			return
		}

		c.Next()
	}
}

// generateRateLimitKey 生成限流 key
func generateRateLimitKey(c *gin.Context, config RateLimitConfig) string {
	// 如果提供了自定义 key 函数，使用自定义函数
	if config.KeyFunc != nil {
		return fmt.Sprintf("ratelimit:%s", config.KeyFunc(c))
	}

	// 根据类型生成默认 key
	switch config.Type {
	case GlobalLimit:
		return "ratelimit:global"
	case IPLimit:
		return fmt.Sprintf("ratelimit:ip:%s", c.ClientIP())
	case UserLimit:
		// 从 context 中获取用户 ID（由 Auth 中间件注入）
		userID, exists := c.Get("user_id")
		if !exists {
			// 未登录用户按 IP 限流
			return fmt.Sprintf("ratelimit:ip:%s", c.ClientIP())
		}
		return fmt.Sprintf("ratelimit:user:%v", userID)
	default:
		return "ratelimit:unknown"
	}
}

// checkRateLimit 检查是否超过限流（滑动窗口算法）
func checkRateLimit(ctx context.Context, rdb *redis.Client, key string, limit int, window time.Duration) (bool, error) {
	// 使用 INCR + EXPIRE 实现简单的计数器
	// 更精确的实现可以使用 Lua 脚本或 Redis 的 ZSET
	current, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	// 第一次请求时设置过期时间
	if current == 1 {
		rdb.Expire(ctx, key, window)
	}

	return current <= int64(limit), nil
}
