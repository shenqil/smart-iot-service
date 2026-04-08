package contextx

import (
	"context"

	"rbac-admin/internal/model/entity"

	"github.com/gin-gonic/gin"
)

// 使用私有类型作为键，避免冲突
type userRoleKeyType struct{}

var userRoleKey = userRoleKeyType{}

// SetUserRoleToRequestContext 将用户角色存入 gin.Context 的 Request.Context
func SetUserRoleToRequestContext(c *gin.Context, role *entity.UserRole) {
	if c == nil || c.Request == nil {
		return
	}

	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	newCtx := context.WithValue(ctx, userRoleKey, role)
	c.Request = c.Request.WithContext(newCtx)
}

// GetUserRoleFromRequestContext 从 gin.Context 的 Request.Context 获取用户角色
func GetUserRoleFromRequestContext(c *gin.Context) (*entity.UserRole, bool) {
	if c == nil || c.Request == nil {
		return nil, false
	}

	ctx := c.Request.Context()
	if ctx == nil {
		return nil, false
	}

	val := ctx.Value(userRoleKey)
	if val == nil {
		return nil, false
	}

	role, ok := val.(*entity.UserRole)
	return role, ok
}
