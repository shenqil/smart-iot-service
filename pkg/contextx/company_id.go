package contextx

import (
	"context"

	"github.com/gin-gonic/gin"
)

// 使用私有类型作为键，避免冲突
type companyIDKeyType struct{}

var companyIDKey = companyIDKeyType{}

// SetCompanyIDToRequestContext 将公司ID存入 gin.Context 的 Request.Context
func SetCompanyIDToRequestContext(c *gin.Context, companyID *uint) {
	if c == nil || c.Request == nil {
		return
	}

	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	newCtx := context.WithValue(ctx, companyIDKey, companyID)
	c.Request = c.Request.WithContext(newCtx)
}

// GetCompanyIDFromRequestContext 从 gin.Context 的 Request.Context 获取公司ID
func GetCompanyIDFromRequestContext(c *gin.Context) (*uint, bool) {
	if c == nil || c.Request == nil {
		return nil, false
	}

	ctx := c.Request.Context()
	if ctx == nil {
		return nil, false
	}

	val := ctx.Value(companyIDKey)
	if val == nil {
		return nil, false
	}

	companyID, ok := val.(*uint)
	return companyID, ok
}
