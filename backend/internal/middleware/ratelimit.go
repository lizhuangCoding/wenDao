package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"wenDao/internal/pkg/response"
)

// RateLimitType 限流类型
type RateLimitType int

const (
	GlobalLimit RateLimitType = iota // 全局限流（所有请求）
	IPLimit                          // IP 限流
	UserLimit                        // 用户限流（需要认证）
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Name    string                    // 限流场景名称，避免不同窗口共用同一个 Redis key
	Type    RateLimitType             // 限流类型
	Limit   int                       // 请求数
	Window  time.Duration             // 固定窗口大小
	Message string                    // 触发限流时返回给客户端的具体提示
	KeyFunc func(*gin.Context) string // 自定义 key 生成函数（可选）
}

var rateLimitFixedWindowScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return current
`)

// RateLimit 创建限流中间件
func RateLimit(rdb *redis.Client, config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rdb == nil || config.Limit <= 0 {
			c.Next()
			return
		}

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
			response.TooManyRequests(c, rateLimitExceededMessage(config))
			c.Abort()
			return
		}

		c.Next()
	}
}

func rateLimitExceededMessage(config RateLimitConfig) string {
	if message := strings.TrimSpace(config.Message); message != "" {
		return message
	}
	return "请求过于频繁，请稍后再试"
}

// generateRateLimitKey 生成限流 key
func generateRateLimitKey(c *gin.Context, config RateLimitConfig) string {
	scope := strings.TrimSpace(config.Name)
	if scope == "" {
		scope = rateLimitTypeName(config.Type)
	}

	// 如果提供了自定义 key 函数，使用自定义函数
	if config.KeyFunc != nil {
		return fmt.Sprintf("ratelimit:%s:%s", scope, config.KeyFunc(c))
	}

	// 根据类型生成默认 key
	switch config.Type {
	case GlobalLimit:
		return fmt.Sprintf("ratelimit:%s:global", scope)
	case IPLimit:
		return fmt.Sprintf("ratelimit:%s:ip:%s", scope, c.ClientIP())
	case UserLimit:
		// 从 context 中获取用户 ID（由 Auth 中间件注入）
		userID, exists := c.Get("user_id")
		if !exists {
			// 未登录用户按 IP 限流
			return fmt.Sprintf("ratelimit:%s:ip:%s", scope, c.ClientIP())
		}
		return fmt.Sprintf("ratelimit:%s:user:%v", scope, userID)
	default:
		return fmt.Sprintf("ratelimit:%s:unknown", scope)
	}
}

func rateLimitTypeName(limitType RateLimitType) string {
	switch limitType {
	case GlobalLimit:
		return "global"
	case IPLimit:
		return "ip"
	case UserLimit:
		return "user"
	default:
		return "unknown"
	}
}

// checkRateLimit 检查是否超过限流（原子固定窗口计数器）。
func checkRateLimit(ctx context.Context, rdb *redis.Client, key string, limit int, window time.Duration) (bool, error) {
	current, err := rateLimitFixedWindowScript.Run(ctx, rdb, []string{key}, window.Milliseconds()).Int64()
	if err != nil {
		return false, err
	}

	return current <= int64(limit), nil
}
