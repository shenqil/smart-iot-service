package handler

import (
	"rbac-admin/internal/handler/response"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/service"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"

	"github.com/gin-gonic/gin"
	"fmt"
)

// AliyunOSSHandlerstruct 用户接口处理器
type AliyunOSSHandler struct {
	AliyunService *service.AliyunOSSService
}

// GetPolicyToken 获取上传所需的签名和凭证
// @Summary 获取所需的签名和凭证
// @Description OSS 上传所需的签名和凭证
// @Tags OSS 管理
// @Produce json
// @Param fileType query string true "文件类型"
// @Param customSubDir query string false "可选子目录"
// @Success 200 {object} dto.Response{data=dto.PolicyTokenResponse} "获取成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 404 {object} dto.Response "应用版本不存在"
// @Failure 500 {object} dto.Response "内部服务器错误"
// @Router /api/v1/auth/oss/sts [get]
func (h *AliyunOSSHandler) GetPolicyToken(c *gin.Context) {

	var req dto.PolicyTokenRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger := contextx.GetLoggerFromContext(c.Request.Context())
		logger.Error(fmt.Sprintf("参数绑定失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, constant.MsgInvalidInput, nil))
		return
	}

	resp, err := h.AliyunService.GetPolicyToken(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}
