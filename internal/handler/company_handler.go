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

type CompanyHandler struct {
	CompanySer *service.CompanyService
}

// Create 创建公司
// @Summary 创建公司
// @Tags 公司管理
// @Accept json
// @Produce json
// @Param request body dto.CompanyCreateRequest true "创建参数"
// @Success 200 {object} dto.Response{data=dto.CompanyResponse}
// @Router /api/v1/admin-auth/companies [post]
func (h *CompanyHandler) Create(c *gin.Context) {
	var req dto.CompanyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))

	resp, err := h.CompanySer.Create(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Get 获取公司详情
// @Summary 获取公司详情
// @Tags 公司管理
// @Produce json
// @Param id path int true "公司ID"
// @Success 200 {object} dto.Response{data=dto.CompanyResponse}
// @Router /api/v1/admin-auth/companies/{id} [get]
func (h *CompanyHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id))

	resp, err := h.CompanySer.Get(ctx, uint(id))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Update 更新公司
// @Summary 更新公司
// @Tags 公司管理
// @Accept json
// @Produce json
// @Param id path int true "公司ID"
// @Param request body dto.CompanyUpdateRequest true "更新参数"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/companies/{id} [put]
func (h *CompanyHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.CompanyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id), slog.String("req_body", utils.LogStruct(req)))

	if err := h.CompanySer.Update(ctx, uint(id), req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Delete 删除公司
// @Summary 删除公司
// @Tags 公司管理
// @Produce json
// @Param id path int true "公司ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/companies/{id} [delete]
func (h *CompanyHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id))

	if err := h.CompanySer.Delete(ctx, uint(id)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// List 分页查询公司列表
// @Summary 获取公司列表
// @Tags 公司管理
// @Produce json
// @Param q query string false "综合查询条件"
// @Param name query string false "公司名称"
// @Param code query string false "公司编码"
// @Param status query int false "状态(0:禁用,1:启用)"
// @Param managerId query int false "负责人ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.CompanyResponse}}
// @Router /api/v1/admin-auth/companies [get]
func (h *CompanyHandler) List(c *gin.Context) {
	var filter dto.CompanyFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "查询参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))

	list, total, err := h.CompanySer.List(ctx, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// BatchDelete 批量删除公司
// @Summary 批量删除公司
// @Tags 公司管理
// @Accept json
// @Produce json
// @Param request body dto.CompanyDeleteRequest true "删除ID列表"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/companies [delete]
func (h *CompanyHandler) BatchDelete(c *gin.Context) {
	var req dto.CompanyDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))

	if err := h.CompanySer.BatchDelete(ctx, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// ============================================================================
// 企业管理用户相关方法
// ============================================================================

// ListCompanyUsers 分页查询某个企业的用户列表
// @Summary 获取企业用户列表
// @Tags 公司管理
// @Produce json
// @Param id path int true "公司ID"
// @Param q query string false "综合查询条件（模糊匹配用户名、手机号、邮箱）"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.CompanyUserBriefResponse}}
// @Router /api/v1/admin-auth/companies/{id}/users [get]
func (h *CompanyHandler) ListCompanyUsers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var filter dto.CompanyUserFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "查询参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id), slog.String("req_filter", utils.LogStruct(filter)))

	list, total, err := h.CompanySer.ListCompanyUsers(ctx, uint(id), filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// ListUsersNotInCompany 分页查询某个企业不存在的用户列表
// @Summary 获取企业不存在用户列表
// @Tags 公司管理
// @Produce json
// @Param id path int true "公司ID"
// @Param q query string false "综合查询条件（模糊匹配用户名、手机号、邮箱）"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.CompanyUserBriefResponse}}
// @Router /api/v1/admin-auth/companies/{id}/users/not-in [get]
func (h *CompanyHandler) ListUsersNotInCompany(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var filter dto.CompanyUserNotInFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "查询参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id), slog.String("req_filter", utils.LogStruct(filter)))

	list, total, err := h.CompanySer.ListUsersNotInCompany(ctx, uint(id), filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// AddUserToCompany 为某个企业新增用户（支持单个和批量）
// @Summary 为企业添加用户（支持批量）
// @Tags 公司管理
// @Accept json
// @Produce json
// @Param id path int true "公司ID"
// @Param request body dto.CompanyUserBatchAddRequest true "添加用户参数（支持批量）"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/companies/{id}/users [post]
func (h *CompanyHandler) AddUserToCompany(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.CompanyUserBatchAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id), slog.String("req_body", utils.LogStruct(req)))

	if err := h.CompanySer.BatchAddUsersToCompany(ctx, uint(id), req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// RemoveUserFromCompany 删除企业与用户的关联关系
// @Summary 删除企业用户关联
// @Tags 公司管理
// @Produce json
// @Param id path int true "公司ID"
// @Param userId path int true "用户ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/companies/{id}/users/{userId} [delete]
func (h *CompanyHandler) RemoveUserFromCompany(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "公司ID格式错误", err))
		return
	}

	userID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "用户ID格式错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_companyId", id), slog.Int("req_userId", userID))

	if err := h.CompanySer.RemoveUserFromCompany(ctx, uint(id), uint(userID)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// BatchRemoveUsersFromCompany 批量删除某个企业的用户
// @Summary 批量删除企业用户
// @Tags 公司管理
// @Accept json
// @Produce json
// @Param id path int true "公司ID"
// @Param request body dto.CompanyUserBatchDeleteRequest true "删除用户ID列表"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/companies/{id}/users/batch [delete]
func (h *CompanyHandler) BatchRemoveUsersFromCompany(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "公司ID格式错误", err))
		return
	}

	var req dto.CompanyUserBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_companyId", id), slog.String("req_body", utils.LogStruct(req)))

	if err := h.CompanySer.BatchRemoveUsersFromCompany(ctx, uint(id), req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateUserRoleInCompany 更新某个用户在某个企业的角色
// @Summary 更新企业用户角色
// @Tags 公司管理
// @Accept json
// @Produce json
// @Param id path int true "公司ID"
// @Param userId path int true "用户ID"
// @Param request body dto.CompanyUserUpdateRoleRequest true "更新角色参数"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/companies/{id}/users/{userId}/role [put]
func (h *CompanyHandler) UpdateUserRoleInCompany(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "公司ID格式错误", err))
		return
	}

	userID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "用户ID格式错误", err))
		return
	}

	var req dto.CompanyUserUpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_companyId", id), slog.Int("req_userId", userID), slog.String("req_body", utils.LogStruct(req)))

	if err := h.CompanySer.UpdateUserRoleInCompany(ctx, uint(id), uint(userID), req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}
