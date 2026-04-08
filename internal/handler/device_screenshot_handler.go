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
)

type DeviceScreenshotHandler struct {
	Ser *service.DeviceScreenshotService
}

// Create
// @Summary 创建截图记录
// @Tags 设备截图
// @Accept json
// @Produce json
// @Param request body dto.DeviceScreenshotCreateRequest true "创建参数"
// @Success 200 {object} dto.Response{data=dto.DeviceScreenshotResponse}
// @Router /api/v1/auth/device-screenshots [post]
func (h *DeviceScreenshotHandler) Create(c *gin.Context) {
	var req dto.DeviceScreenshotCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}
	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))
	resp, err := h.Ser.Create(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, resp)
}

// Get
// @Summary 获取截图详情
// @Tags 设备截图
// @Produce json
// @Param id path int true "ID"
// @Success 200 {object} dto.Response{data=dto.DeviceScreenshotResponse}
// @Router /api/v1/auth/device-screenshots/{id} [get]
func (h *DeviceScreenshotHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}
	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_id", id))
	resp, err := h.Ser.Get(ctx, uint(id))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, resp)
}

// Update
// @Summary 更新截图
// @Tags 设备截图
// @Accept json
// @Produce json
// @Param id path int true "ID"
// @Param request body dto.DeviceScreenshotUpdateRequest true "更新参数"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/device-screenshots/{id} [put]
func (h *DeviceScreenshotHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}
	var req dto.DeviceScreenshotUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}
	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_id", id), slog.String("req_body", utils.LogStruct(req)))
	if err := h.Ser.Update(ctx, uint(id), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

// Delete
// @Summary 删除截图
// @Tags 设备截图
// @Produce json
// @Param id path int true "ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/device-screenshots/{id} [delete]
func (h *DeviceScreenshotHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}
	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_id", id))
	if err := h.Ser.Delete(ctx, uint(id)); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}

// List
// @Summary 获取截图列表
// @Tags 设备截图
// @Produce json
// @Param deviceSn query int false "设备序列号(8位纯数字)"
// @Param startTime query int false "开始时间(秒)"
// @Param endTime query int false "结束时间(秒)"
// @Param pageNumber query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.DeviceScreenshotResponse}}
// @Router /api/v1/auth/device-screenshots [get]
func (h *DeviceScreenshotHandler) List(c *gin.Context) {
	var filter dto.DeviceScreenshotFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.Error(c, errx.New(constant.BadRequest, "查询参数错误", err))
		return
	}
	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))
	list, total, err := h.Ser.List(ctx, filter)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginate(c, list, total)
}

// BatchDelete
// @Summary 批量删除截图
// @Tags 设备截图
// @Accept json
// @Produce json
// @Param request body dto.DeviceScreenshotDeleteRequest true "ID 列表"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/device-screenshots/batch [delete]
func (h *DeviceScreenshotHandler) BatchDelete(c *gin.Context) {
	var req dto.DeviceScreenshotDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}
	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))
	if err := h.Ser.BatchDelete(ctx, req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, nil)
}
