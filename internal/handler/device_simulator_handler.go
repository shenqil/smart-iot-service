package handler

import (
	"rbac-admin/internal/handler/response"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/service"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"rbac-admin/pkg/utils"

	"github.com/gin-gonic/gin"
	"log/slog"
)

type DeviceSimulatorHandler struct {
	DeviceSimulatorSer *service.DeviceSimulatorService
}

// Create 创建记录
// @Summary 创建设备模拟器
// @Tags 设备模拟器管理
// @Accept json
// @Produce json
// @Param request body dto.DeviceSimulatorCreateRequest true "创建参数"
// @Success 200 {object} dto.Response{data=dto.DeviceSimulatorResponse}
// @Failure 400 {object} dto.Response
// @Failure 409 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/device-simulators [post]
func (h *DeviceSimulatorHandler) Create(c *gin.Context) {
	var req dto.DeviceSimulatorCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("参数绑定失败", slog.Any("error", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	// 验证请求数据
	if err := req.Validate(); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("请求验证失败", slog.Any("error", err))
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.String("req_body", utils.LogStruct(req)))

	resp, err := h.DeviceSimulatorSer.Create(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// List 分页查询列表
// @Summary 分页查询设备模拟器列表
// @Tags 设备模拟器管理
// @Produce json
// @Param pageNumber query int false "页码" default(1)
// @Param pageSize query int false "每页大小" default(10)
// @Param deviceSN query string false "设备SN号模糊搜索"
// @Param q query string false "综合查询条件"
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.DeviceSimulatorResponse}}
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/device-simulators [get]
func (h *DeviceSimulatorHandler) List(c *gin.Context) {
	var req dto.DeviceSimulatorListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("参数绑定失败", slog.Any("error", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.String("req_query", utils.LogStruct(req)))

	resp, err := h.DeviceSimulatorSer.List(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// BatchDelete 批量删除
// @Summary 批量删除设备模拟器
// @Tags 设备模拟器管理
// @Accept json
// @Produce json
// @Param request body dto.DeviceSimulatorBatchDeleteRequest true "删除参数"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/device-simulators [delete]
func (h *DeviceSimulatorHandler) BatchDelete(c *gin.Context) {
	var req dto.DeviceSimulatorBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("参数绑定失败", slog.Any("error", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.String("req_body", utils.LogStruct(req)))

	if err := h.DeviceSimulatorSer.BatchDelete(ctx, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

