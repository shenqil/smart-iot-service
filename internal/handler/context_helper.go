package handler

import (
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"

	"github.com/gin-gonic/gin"
)

// getRequiredUserID 从上下文获取用户ID，如果不存在或为0则返回错误
func getRequiredUserID(c *gin.Context) (uint, error) {
	userID, exists := contextx.GetUserIDFromRequestContext(c)
	if !exists {
		return 0, errx.New(constant.Unauthorized, "未授权访问", nil)
	}
	return userID, nil
}

// getRequiredCompanyID 从上下文获取公司ID，如果不存在或为0则返回错误
func getRequiredCompanyID(c *gin.Context) (uint, error) {
	companyID, exists := contextx.GetCompanyIDFromRequestContext(c)
	if !exists || companyID == nil || *companyID == 0 {
		return 0, errx.New(constant.BadRequest, "请求必须包含公司ID", nil)
	}
	return *companyID, nil
}
