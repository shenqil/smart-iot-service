package contextx

import (
	"context"

	"github.com/gin-gonic/gin"
)

// 使用私有类型作为键，避免冲突
type userIDKeyType struct{}

var userIDKey = userIDKeyType{}

// SetUserIDToRequestContext 将用户ID存入 gin.Context 的 Request.Context
func SetUserIDToRequestContext(c *gin.Context, userID uint) {
	if c == nil || c.Request == nil {
		return
	}

	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	newCtx := context.WithValue(ctx, userIDKey, userID)
	c.Request = c.Request.WithContext(newCtx)
}

// GetUserIDFromRequestContext 从 gin.Context 的 Request.Context 获取用户ID
func GetUserIDFromRequestContext(c *gin.Context) (uint, bool) {
	if c == nil || c.Request == nil {
		return 0, false
	}

	ctx := c.Request.Context()
	if ctx == nil {
		return 0, false
	}

	val := ctx.Value(userIDKey)
	if val == nil {
		return 0, false
	}

	userID, ok := val.(uint)
	return userID, ok
}
