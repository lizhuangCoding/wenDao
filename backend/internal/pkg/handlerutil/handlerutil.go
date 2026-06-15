package handlerutil

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/response"
)

// MustGetInt64Param parses an int64 from the URL path parameter identified by name.
// On failure it writes a 400 error response via the response package and returns false.
func MustGetInt64Param(c *gin.Context, name string) (int64, bool) {
	val, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil {
		response.InvalidParams(c, "Invalid "+name)
		return 0, false
	}
	return val, true
}

// MustGetUserID retrieves the user_id set by the JWT middleware from gin.Context.
// On failure it writes a 401 error response and returns false.
func MustGetUserID(c *gin.Context) (int64, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "authentication required")
		return 0, false
	}
	userID, ok := value.(int64)
	if !ok {
		response.Unauthorized(c, "authentication required")
		return 0, false
	}
	return userID, true
}

// MustGetOptionalQueryInt reads an optional int query parameter.
// If the parameter is missing or cannot be parsed, defaultVal is returned.
func MustGetOptionalQueryInt(c *gin.Context, name string, defaultVal int) int {
	val := c.Query(name)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return n
}
