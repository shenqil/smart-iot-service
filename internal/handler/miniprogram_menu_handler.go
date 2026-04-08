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

type MiniProgramMenuHandler struct {
	MiniProgramMenuSer *service.MiniProgramMenuService
}

// Create 创建记录
// @Summary 创建小程序功能栏配置
// @Tags 小程序功能栏管理
// @Accept json
// @Produce json
// @Param request body dto.MiniProgramMenuCreateRequest true "创建参数"
// @Success 200 {object} dto.Response{data=dto.MiniProgramMenuResponse}
// @Failure 400 {object} dto.Response
// @Failure 409 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/miniprogram-menus [post]
func (h *MiniProgramMenuHandler) Create(c *gin.Context) {
	var req dto.MiniProgramMenuCreateRequest
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

	resp, err := h.MiniProgramMenuSer.Create(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Get 获取详情
// @Summary 获取小程序功能栏配置详情
// @Tags 小程序功能栏管理
// @Produce json
// @Param id path int true "配置ID"
// @Success 200 {object} dto.Response{data=dto.MiniProgramMenuResponse}
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/miniprogram-menus/{id} [get]
func (h *MiniProgramMenuHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("ID格式错误", slog.Any("error", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.Uint64("req_id", uint64(id)))

	resp, err := h.MiniProgramMenuSer.Get(ctx, uint(id))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Update 更新记录
// @Summary 更新小程序功能栏配置
// @Tags 小程序功能栏管理
// @Accept json
// @Produce json
// @Param id path int true "配置ID"
// @Param request body dto.MiniProgramMenuUpdateRequest true "更新参数"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 409 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/miniprogram-menus/{id} [put]
func (h *MiniProgramMenuHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("ID格式错误", slog.Any("error", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.MiniProgramMenuUpdateRequest
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
		slog.Uint64("req_id", uint64(id)),
		slog.String("req_body", utils.LogStruct(req)))

	if err := h.MiniProgramMenuSer.Update(ctx, uint(id), req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Delete 删除记录（彻底删除）
// @Summary 删除小程序功能栏配置
// @Tags 小程序功能栏管理
// @Produce json
// @Param id path int true "配置ID"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/miniprogram-menus/{id} [delete]
func (h *MiniProgramMenuHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("ID格式错误", slog.Any("error", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.Uint64("req_id", uint64(id)))

	if err := h.MiniProgramMenuSer.Delete(ctx, uint(id)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// List 分页列表
// @Summary 获取小程序功能栏配置列表
// @Tags 小程序功能栏管理
// @Produce json
// @Param q query string false "综合查询关键词"
// @Param title query string false "标题"
// @Param code query string false "功能代码"
// @Param status query int false "状态(0:禁用,1:启用)"
// @Param pageNumber query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.MiniProgramMenuResponse}}
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/miniprogram-menus [get]
func (h *MiniProgramMenuHandler) List(c *gin.Context) {
	var filter dto.MiniProgramMenuFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("查询参数绑定失败", slog.Any("error", err))
		response.Error(c, errx.New(constant.BadRequest, "查询参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.String("req_filter", utils.LogStruct(filter)))

	list, total, err := h.MiniProgramMenuSer.List(ctx, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// ListAll 获取所有列表（直接从数据库查询，不使用缓存）
// @Summary 获取所有小程序功能栏配置（无分页）
// @Tags 小程序功能栏管理
// @Produce json
// @Success 200 {object} dto.Response{data=[]dto.MiniProgramMenuResponse}
// @Failure 500 {object} dto.Response
// @Router /api/v1/auth/miniprogram-menus/all [get]
func (h *MiniProgramMenuHandler) ListAll(c *gin.Context) {
	ctx := contextx.SetLoggerToContext(c.Request.Context())

	list, err := h.MiniProgramMenuSer.ListAll(ctx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, list)
}

// BatchDelete 批量删除（彻底删除）
// @Summary 批量删除小程序功能栏配置
// @Tags 小程序功能栏管理
// @Accept json
// @Produce json
// @Param request body dto.MiniProgramMenuDeleteRequest true "删除ID列表"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/miniprogram-menus [delete]
func (h *MiniProgramMenuHandler) BatchDelete(c *gin.Context) {
	var req dto.MiniProgramMenuDeleteRequest
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

	if err := h.MiniProgramMenuSer.BatchDelete(ctx, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}
