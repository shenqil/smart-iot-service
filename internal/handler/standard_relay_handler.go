package handler

import (
	"log/slog"
	"strconv"

	"rbac-admin/internal/handler/response"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/service"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"rbac-admin/pkg/utils"

	"github.com/gin-gonic/gin"
)

type StandardRelayHandler struct {
	StandardRelaySer *service.StandardRelayService
}

// Create 创建记录
// @Summary 创建标准继电器
// @Tags 标准继电器管理
// @Accept json
// @Produce json
// @Param request body dto.StandardRelayCreateRequest true "创建参数"
// @Success 200 {object} dto.Response{data=dto.StandardRelayResponse}
// @Failure 400 {object} dto.Response
// @Failure 409 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/standard-relays [post]
func (h *StandardRelayHandler) Create(c *gin.Context) {
	var req dto.StandardRelayCreateRequest
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

	resp, err := h.StandardRelaySer.Create(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Get 获取详情
// @Summary 获取标准继电器详情
// @Tags 标准继电器管理
// @Produce json
// @Param id path int true "标准继电器ID"
// @Success 200 {object} dto.Response{data=dto.StandardRelayResponse}
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/standard-relays/{id} [get]
func (h *StandardRelayHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("ID格式错误", slog.Any("error", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.Uint64("req_id", uint64(id)))

	resp, err := h.StandardRelaySer.Get(ctx, uint(id))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Update 更新记录
// @Summary 更新标准继电器
// @Tags 标准继电器管理
// @Accept json
// @Produce json
// @Param id path int true "标准继电器ID"
// @Param request body dto.StandardRelayUpdateRequest true "更新参数"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 409 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/standard-relays/{id} [put]
func (h *StandardRelayHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("ID格式错误", slog.Any("error", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.StandardRelayUpdateRequest
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

	if err := h.StandardRelaySer.Update(ctx, uint(id), req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Delete 删除记录
// @Summary 删除标准继电器
// @Tags 标准继电器管理
// @Produce json
// @Param id path int true "标准继电器ID"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/standard-relays/{id} [delete]
func (h *StandardRelayHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("ID格式错误", slog.Any("error", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.Uint64("req_id", uint64(id)))

	if err := h.StandardRelaySer.Delete(ctx, uint(id)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// List 分页列表
// @Summary 获取标准继电器列表
// @Tags 标准继电器管理
// @Produce json
// @Param name query string false "名称搜索参数"
// @Param q query string false "全部搜索参数"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.StandardRelayResponse}}
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/standard-relays [get]
func (h *StandardRelayHandler) List(c *gin.Context) {
	var filter dto.StandardRelayFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("查询参数绑定失败", slog.Any("error", err))
		response.Error(c, errx.New(constant.BadRequest, "查询参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.String("req_filter", utils.LogStruct(filter)))

	list, total, err := h.StandardRelaySer.List(ctx, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// ListAll 获取所有继电器（走缓存）
// @Summary 获取所有标准继电器（无分页）
// @Tags 标准继电器管理
// @Produce json
// @Success 200 {object} dto.Response{data=[]dto.StandardRelayResponse}
// @Failure 500 {object} dto.Response
// @Router /api/v1/auth/standard-relays/all [get]
func (h *StandardRelayHandler) ListAll(c *gin.Context) {
	ctx := contextx.SetLoggerToContext(c.Request.Context())

	list, err := h.StandardRelaySer.ListAll(ctx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, list)
}

// BatchDelete 批量删除
// @Summary 批量删除标准继电器
// @Tags 标准继电器管理
// @Accept json
// @Produce json
// @Param request body dto.StandardRelayDeleteRequest true "删除ID列表"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/standard-relays [delete]
func (h *StandardRelayHandler) BatchDelete(c *gin.Context) {
	var req dto.StandardRelayDeleteRequest
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

	if err := h.StandardRelaySer.BatchDelete(ctx, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}
