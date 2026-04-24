package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应格式
type Response struct {
	Code    int         `json:"code"` // 0=成功，非0=错误码
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 错误码定义
const (
	CodeSuccess            = 0
	CodeInvalidParams      = 40001
	CodeUnauthorized       = 40100
	CodeForbidden          = 40300
	CodeNotFound           = 40400
	CodeTooManyReq         = 42900
	CodeServiceUnavailable = 50300
	CodeInternalError      = 50000
)

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	statusCode := http.StatusOK
	switch code {
	case CodeInvalidParams:
		statusCode = http.StatusBadRequest
	case CodeUnauthorized:
		statusCode = http.StatusUnauthorized
	case CodeForbidden:
		statusCode = http.StatusForbidden
	case CodeNotFound:
		statusCode = http.StatusNotFound
	case CodeTooManyReq:
		statusCode = http.StatusTooManyRequests
	case CodeServiceUnavailable:
		statusCode = http.StatusServiceUnavailable
	case CodeInternalError:
		statusCode = http.StatusInternalServerError
	}

	c.JSON(statusCode, Response{
		Code:    code,
		Message: message,
	})
}

// InvalidParams 参数错误
func InvalidParams(c *gin.Context, message string) {
	Error(c, CodeInvalidParams, message)
}

// Unauthorized 未授权
func Unauthorized(c *gin.Context, message string) {
	Error(c, CodeUnauthorized, message)
}

// Forbidden 禁止访问
func Forbidden(c *gin.Context, message string) {
	Error(c, CodeForbidden, message)
}

// NotFound 资源未找到
func NotFound(c *gin.Context, message string) {
	Error(c, CodeNotFound, message)
}

// TooManyRequests 请求过多
func TooManyRequests(c *gin.Context, message string) {
	Error(c, CodeTooManyReq, message)
}

// ServiceUnavailable 服务不可用
func ServiceUnavailable(c *gin.Context, message string) {
	Error(c, CodeServiceUnavailable, message)
}

// InternalError 服务器内部错误
func InternalError(c *gin.Context, message string) {
	Error(c, CodeInternalError, message)
}
