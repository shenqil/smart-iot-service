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

type SiteHandler struct {
	SiteSer *service.SiteService
}

// Create 创建场所
// @Summary 创建场所
// @Tags 场所管理
// @Accept json
// @Produce json
// @Param request body dto.SiteCreateRequest true "创建参数"
// @Success 200 {object} dto.Response{data=dto.SiteResponse}
// @Router /api/v1/auth/sites [post]
func (h *SiteHandler) Create(c *gin.Context) {
	var req dto.SiteCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	req.CompanyID = companyID

	resp, err := h.SiteSer.Create(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Get 获取场所详情
// @Summary 获取场所详情
// @Tags 场所管理
// @Produce json
// @Param id path int true "场所ID"
// @Success 200 {object} dto.Response{data=dto.SiteResponse}
// @Router /api/v1/auth/sites/{id} [get]
func (h *SiteHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.Int("req_Id", id))

	resp, err := h.SiteSer.Get(ctx, uint(id))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Update 更新场所
// @Summary 更新场所
// @Tags 场所管理
// @Accept json
// @Produce json
// @Param id path int true "场所ID"
// @Param request body dto.SiteUpdateRequest true "更新参数"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/sites/{id} [put]
func (h *SiteHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.SiteUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.Int("req_Id", id),
		slog.String("req_body", utils.LogStruct(req)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	req.CompanyID = companyID

	if err := h.SiteSer.Update(ctx, uint(id), req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Delete 删除场所
// @Summary 删除场所
// @Tags 场所管理
// @Produce json
// @Param id path int true "场所ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/sites/{id} [delete]
func (h *SiteHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.SiteSer.Delete(ctx, uint(id), companyID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// List 分页列表
// @Summary 获取场所列表
// @Tags 场所管理
// @Produce json
// @Param q query string false "综合查询条件"
// @Param name query string false "名称"
// @Param code query string false "编码"
// @Param type query string false "类型"
// @Param status query string false "状态"
// @Param province query string false "省份"
// @Param city query string false "城市"
// @Param district query string false "区县"
// @Param minLng query number false "最小经度"
// @Param maxLng query number false "最大经度"
// @Param minLat query number false "最小纬度"
// @Param maxLat query number false "最大纬度"
// @Param deviceId query int false "设备ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.SiteResponse}}
// @Router /api/v1/auth/sites [get]
func (h *SiteHandler) List(c *gin.Context) {
	var filter dto.SiteFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "查询参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.String("req_filter", utils.LogStruct(filter)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter.CompanyID = companyID

	list, total, err := h.SiteSer.List(ctx, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// BatchDelete 批量删除
// @Summary 批量删除场所
// @Tags 场所管理
// @Accept json
// @Produce json
// @Param request body dto.SiteDeleteRequest true "删除ID列表"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/sites/batch [delete]
func (h *SiteHandler) BatchDelete(c *gin.Context) {
	var req dto.SiteDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.String("req_body", utils.LogStruct(req)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	req.CompanyID = companyID

	if err := h.SiteSer.BatchDelete(ctx, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// AssignDevices 园区分配设备
// @Summary 园区分配设备
// @Tags 场所管理
// @Accept json
// @Produce json
// @Param request body dto.SiteDeviceAssignRequest true "分配设备参数"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/sites/devices/assign [post]
func (h *SiteHandler) AssignDevices(c *gin.Context) {
	var req dto.SiteDeviceAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.String("req_body", utils.LogStruct(req)))

	if err := h.SiteSer.AssignDevices(ctx, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// RemoveDevices 园区删除设备
// @Summary 园区删除设备
// @Tags 场所管理
// @Accept json
// @Produce json
// @Param request body dto.SiteDeviceRemoveRequest true "删除设备参数"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/sites/devices/remove [delete]
func (h *SiteHandler) RemoveDevices(c *gin.Context) {
	var req dto.SiteDeviceRemoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.String("req_body", utils.LogStruct(req)))

	if err := h.SiteSer.RemoveDevices(ctx, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// AddUsersToSite 园区批量添加用户
// @Summary 园区批量添加用户
// @Tags 场所管理
// @Accept json
// @Produce json
// @Param id path int true "场所ID"
// @Param request body dto.SiteUserAddRequest true "添加用户参数"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/sites/{id}/users [post]
func (h *SiteHandler) AddUsersToSite(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.SiteUserAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.SiteSer.AddUsersToSite(ctx, uint(id), req, companyID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// RemoveUsersFromSite 园区批量移除用户
// @Summary 园区批量移除用户
// @Tags 场所管理
// @Accept json
// @Produce json
// @Param id path int true "场所ID"
// @Param request body dto.SiteUserRemoveRequest true "移除用户参数"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/sites/{id}/users [delete]
func (h *SiteHandler) RemoveUsersFromSite(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.SiteUserRemoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.SiteSer.RemoveUsersFromSite(ctx, uint(id), req, companyID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// ListSiteUsers 获取园区用户列表
// @Summary 获取园区用户列表
// @Tags 场所管理
// @Produce json
// @Param id path int true "场所ID"
// @Param pageNumber query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.SiteUserResponse}}
// @Router /api/v1/auth/sites/{id}/users [get]
func (h *SiteHandler) ListSiteUsers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var filter dto.SiteUserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.Error(c, errx.New(constant.BadRequest, "查询参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))

	list, total, err := h.SiteSer.ListSiteUsers(ctx, uint(id), filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// ListUsersNotInSite 获取未分配到园区的用户列表
// @Summary 获取未分配到园区的用户列表
// @Tags 场所管理
// @Produce json
// @Param id path int true "场所ID"
// @Param pageNumber query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.SiteUserResponse}}
// @Router /api/v1/auth/sites/{id}/users/not-in [get]
func (h *SiteHandler) ListUsersNotInSite(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var filter dto.SiteUserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.Error(c, errx.New(constant.BadRequest, "查询参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	list, total, err := h.SiteSer.ListUsersNotInSite(ctx, uint(id), filter, companyID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}
