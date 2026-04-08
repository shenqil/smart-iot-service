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

	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
)

type WaterFertilizerTemplateHandler struct {
	TemplateSer *service.WaterFertilizerTemplateService
}

// parseID 解析路径参数中的ID
func (h *WaterFertilizerTemplateHandler) parseID(c *gin.Context) (uint, error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		return 0, errx.New(constant.BadRequest, "ID格式错误", err)
	}
	return uint(id), nil
}

// bindAndValidate 绑定并验证请求参数
func (h *WaterFertilizerTemplateHandler) bindAndValidate(c *gin.Context, req interface{}) error {
	if err := c.ShouldBindJSON(req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		return errx.New(constant.BadRequest, "参数错误", err)
	}
	return nil
}

// Create 创建水肥模板
// @Summary 创建水肥模板
// @Tags 水肥模板管理
// @Accept json
// @Produce json
// @Param request body dto.WaterFertilizerTemplateCreateRequest true "创建参数"
// @Success 200 {object} dto.Response{data=dto.WaterFertilizerTemplateResponse}
// @Router /api/v1/admin-auth/water-fertilizer-templates [post]
// @Router /api/v1/auth/water-fertilizer-templates [post]
func (h *WaterFertilizerTemplateHandler) Create(c *gin.Context) {
	var req dto.WaterFertilizerTemplateCreateRequest
	if err := h.bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))
	resp, err := h.TemplateSer.Create(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Get 获取水肥模板详情
// @Summary 获取水肥模板详情
// @Tags 水肥模板管理
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} dto.Response{data=dto.WaterFertilizerTemplateResponse}
// @Router /api/v1/admin-auth/water-fertilizer-templates/{id} [get]
func (h *WaterFertilizerTemplateHandler) Get(c *gin.Context) {
	id, err := h.parseID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Uint64("req_Id", uint64(id)))
	resp, err := h.TemplateSer.Get(ctx, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Update 更新水肥模板
// @Summary 更新水肥模板
// @Tags 水肥模板管理
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param request body dto.WaterFertilizerTemplateUpdateRequest true "更新参数"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/water-fertilizer-templates/{id} [put]
func (h *WaterFertilizerTemplateHandler) Update(c *gin.Context) {
	id, err := h.parseID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.WaterFertilizerTemplateUpdateRequest
	if err := h.bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Uint64("req_Id", uint64(id)), slog.String("req_body", utils.LogStruct(req)))
	if err := h.TemplateSer.Update(ctx, id, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Delete 删除水肥模板
// @Summary 删除水肥模板
// @Tags 水肥模板管理
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/water-fertilizer-templates/{id} [delete]
func (h *WaterFertilizerTemplateHandler) Delete(c *gin.Context) {
	id, err := h.parseID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Uint64("req_Id", uint64(id)))
	if err := h.TemplateSer.Delete(ctx, id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// List 分页列表
// @Summary 获取水肥模板列表（带分页）
// @Tags 水肥模板管理
// @Produce json
// @Param keywords query string false "关键词"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.WaterFertilizerTemplateResponse}}
// @Router /api/v1/admin-auth/water-fertilizer-templates [get]
// @Router /api/v1/auth/water-fertilizer-templates [get]
func (h *WaterFertilizerTemplateHandler) List(c *gin.Context) {
	var filter dto.WaterFertilizerTemplateFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "查询参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))
	list, total, err := h.TemplateSer.List(ctx, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// BatchDelete 批量删除
// @Summary 批量删除水肥模板
// @Tags 水肥模板管理
// @Accept json
// @Produce json
// @Param request body dto.WaterFertilizerTemplateDeleteRequest true "删除ID列表"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/water-fertilizer-templates [delete]
func (h *WaterFertilizerTemplateHandler) BatchDelete(c *gin.Context) {
	var req dto.WaterFertilizerTemplateDeleteRequest
	if err := h.bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))
	if err := h.TemplateSer.BatchDelete(ctx, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}
