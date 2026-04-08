package handler

import (
	"strconv"

	"rbac-admin/internal/handler/response"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/service"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"rbac-admin/pkg/utils"

	"github.com/gin-gonic/gin"
	"log/slog"
	"fmt"
)

type StandardSensorHandler struct {
	SensorSer *service.StandardSensorService
}

// parseID 解析路径参数中的ID
func (h *StandardSensorHandler) parseID(c *gin.Context) (uint, error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		return 0, errx.New(constant.BadRequest, "ID格式错误", err)
	}
	return uint(id), nil
}

// bindAndValidate 绑定并验证请求参数
func (h *StandardSensorHandler) bindAndValidate(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindJSON(req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		return errx.New(constant.BadRequest, "参数错误", err)
	}
	return nil
}

// Create 创建传感器
// @Summary 创建标准传感器
// @Tags 标准传感器管理
// @Accept json
// @Produce json
// @Param request body dto.StandardSensorCreateRequest true "创建参数"
// @Success 200 {object} dto.Response{data=dto.StandardSensorResponse}
// @Router /api/v1/admin-auth/standard-sensors [post]
func (h *StandardSensorHandler) Create(c *gin.Context) {
	var req dto.StandardSensorCreateRequest
	if err := h.bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))
	resp, err := h.SensorSer.Create(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Get 获取传感器详情
// @Summary 获取传感器详情
// @Tags 标准传感器管理
// @Produce json
// @Param id path int true "传感器ID"
// @Success 200 {object} dto.Response{data=dto.StandardSensorResponse}
// @Router /api/v1/admin-auth/standard-sensors/{id} [get]
func (h *StandardSensorHandler) Get(c *gin.Context) {
	id, err := h.parseID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Uint64("req_Id", uint64(id)))
	resp, err := h.SensorSer.Get(ctx, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Update 更新传感器
// @Summary 更新传感器
// @Tags 标准传感器管理
// @Accept json
// @Produce json
// @Param id path int true "传感器ID"
// @Param request body dto.StandardSensorUpdateRequest true "更新参数"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/standard-sensors/{id} [put]
func (h *StandardSensorHandler) Update(c *gin.Context) {
	id, err := h.parseID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.StandardSensorUpdateRequest
	if err := h.bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Uint64("req_Id", uint64(id)), slog.String("req_body", utils.LogStruct(req)))
	if err := h.SensorSer.Update(ctx, id, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Delete 删除传感器
// @Summary 删除传感器
// @Tags 标准传感器管理
// @Produce json
// @Param id path int true "传感器ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/standard-sensors/{id} [delete]
func (h *StandardSensorHandler) Delete(c *gin.Context) {
	id, err := h.parseID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Uint64("req_Id", uint64(id)))
	if err := h.SensorSer.Delete(ctx, id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// List 分页列表（走数据库查询）
// @Summary 获取标准传感器列表（带分页）
// @Tags 标准传感器管理
// @Produce json
// @Param name query string false "名称搜索参数"
// @Param q query string false "全部搜索参数"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.StandardSensorResponse}}
// @Router /api/v1/admin-auth/standard-sensors [get]
func (h *StandardSensorHandler) List(c *gin.Context) {
	var filter dto.StandardSensorFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "查询参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))
	list, total, err := h.SensorSer.List(ctx, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// ListAll 获取所有传感器（走缓存）
// @Summary 获取所有标准传感器（无分页）
// @Tags 标准传感器管理
// @Produce json
// @Success 200 {object} dto.Response{data=[]dto.StandardSensorResponse}
// @Router /api/v1/auth/standard-sensors/all [get]
func (h *StandardSensorHandler) ListAll(c *gin.Context) {
	list, err := h.SensorSer.ListAll(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, list)
}

// BatchDelete 批量删除
// @Summary 批量删除传感器
// @Tags 标准传感器管理
// @Accept json
// @Produce json
// @Param request body dto.StandardSensorDeleteRequest true "删除ID列表"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/standard-sensors [delete]
func (h *StandardSensorHandler) BatchDelete(c *gin.Context) {
	var req dto.StandardSensorDeleteRequest
	if err := h.bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))
	if err := h.SensorSer.BatchDelete(ctx, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}
