package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORS 创建跨域处理中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")

		// 设置允许的请求源
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		// 设置允许的请求头
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
		// 设置允许的请求方法
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		// 允许浏览器发送凭证（Cookie）
		c.Header("Access-Control-Allow-Credentials", "true")
		// 预检请求缓存时间（24小时）
		c.Header("Access-Control-Max-Age", "86400")

		// 处理预检请求
		if method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
