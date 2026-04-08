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

// parseUserID 解析用户ID参数
func parseUserID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		return 0, errx.New(constant.BadRequest, "ID格式错误", err)
	}
	return uint(id), nil
}

// bindAndValidate 绑定并验证请求参数
func bindAndValidate[T any](c *gin.Context, req *T) error {
	if err := c.ShouldBindJSON(req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		return errx.New(constant.BadRequest, "参数错误", err)
	}
	return nil
}

// bindQueryAndValidate 绑定并验证查询参数
func bindQueryAndValidate[T any](c *gin.Context, req *T) error {
	if err := c.ShouldBindQuery(req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		return errx.New(constant.BadRequest, "查询参数错误", err)
	}
	return nil
}

type UserHandler struct {
	UserSer        *service.UserService
	CompanyService *service.CompanyService
}

// ============================================================================
// 普通用户管理方法（需要CompanyID和Role，用于/api/v1/auth路由组）
// ============================================================================

// Create 创建用户
// @Summary 创建用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body dto.UserCreateRequest true "创建参数"
// @Success 200 {object} dto.Response{data=dto.UserResponse}
// @Router /api/v1/auth/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req dto.UserCreateRequest
	if err := bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	// 从上下文获取企业ID
	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	req.CompanyID = companyID

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))

	resp, err := h.UserSer.Create(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Update 更新用户
// @Summary 更新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body dto.UserUpdateRequest true "更新参数"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := parseUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.UserUpdateRequest
	if err := bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	// 从上下文获取企业ID
	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	req.CompanyID = companyID

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Uint64("req_Id", uint64(id)), slog.String("req_body", utils.LogStruct(req)))

	if err := h.UserSer.Update(ctx, id, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Delete 删除用户
// @Summary 删除用户
// @Tags 用户管理
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := parseUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 从上下文获取企业ID
	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Uint64("req_Id", uint64(id)))

	if err := h.UserSer.Delete(ctx, id, companyID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// List 分页查询用户列表
// @Summary 获取用户列表
// @Tags 用户管理
// @Produce json
// @Param q query string false "综合查询条件"
// @Param username query string false "用户名"
// @Param nickname query string false "昵称"
// @Param email query string false "邮箱"
// @Param phone query string false "手机号"
// @Param status query int false "状态(0:禁用,1:启用)"
// @Param gender query string false "性别(male/female/unknown)"
// @Param minAge query int false "最小年龄"
// @Param maxAge query int false "最大年龄"
// @Param roleId query int false "角色ID"
// @Param siteId query int false "园区ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.UserResponse}}
// @Router /api/v1/auth/users [get]
func (h *UserHandler) List(c *gin.Context) {
	var filter dto.UserFilter
	if err := bindQueryAndValidate(c, &filter); err != nil {
		response.Error(c, err)
		return
	}

	// 从上下文获取企业ID
	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter.CompanyID = &companyID

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))

	list, total, err := h.UserSer.List(ctx, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// Get 获取用户详情
// @Summary 获取用户详情
// @Tags 用户管理
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} dto.Response{data=dto.UserResponse}
// @Router /api/v1/auth/users/{id} [get]
func (h *UserHandler) Get(c *gin.Context) {
	id, err := parseUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 从上下文获取企业ID
	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Uint64("req_Id", uint64(id)))

	resp, err := h.UserSer.Get(ctx, id, companyID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// BatchDelete 批量删除用户
// @Summary 批量删除用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body dto.UserDeleteRequest true "删除ID列表"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/users/batch [delete]
func (h *UserHandler) BatchDelete(c *gin.Context) {
	var req dto.UserDeleteRequest
	if err := bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	// 从上下文获取企业ID
	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	req.CompanyID = companyID

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))

	if err := h.UserSer.BatchDelete(ctx, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// ============================================================================
// 当前用户相关方法（操作当前登录用户自身）
// ============================================================================

// GetCurrentUser 获取当前用户
// @Summary 获取当前用户
// @Description 获取当前登录用户详细信息和企业列表。此接口不校验角色和企业。
// @Tags 用户管理
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} dto.Response{data=dto.CurrentUserInfoResponse} "获取成功"
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "未授权访问"
// @Failure 404 {object} dto.Response "用户不存在"
// @Failure 500 {object} dto.Response "内部服务器错误"
// @Router /api/v1/auth/users/current [get]
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	// 从上下文获取用户ID（已通过JWTTokenMiddleware验证并设置）
	userID, exists := contextx.GetUserIDFromRequestContext(c)
	if !exists {
		logger := contextx.GetLoggerFromContext(c.Request.Context())
		logger.Error("从上下文中获取用户ID失败")
		response.Error(c, errx.New(constant.Unauthorized, "未授权访问", nil))
		return
	}

	// 获取用户信息
	currentUser, err := h.UserSer.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 通过关联表查询企业列表和角色
	companies, err := h.UserSer.GetUserCompaniesWithRoles(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 返回用户信息和企业列表
	response.Success(c, dto.CurrentUserInfoResponse{
		Info:      currentUser,
		Companies: companies,
	})
}

// UpdateProfile 更新当前用户个人信息
// @Summary 更新当前用户个人信息
// @Description 更新当前登录用户的个人信息（不包含密码）
// @Tags 用户管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body dto.UserProfileUpdateRequest true "更新参数"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response "请求参数错误"
// @Failure 401 {object} dto.Response "未授权"
// @Failure 409 {object} dto.Response "冲突（如邮箱/手机号已存在）"
// @Failure 500 {object} dto.Response "内部服务器错误"
// @Router /api/v1/auth/users/current/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	id, exists := contextx.GetUserIDFromRequestContext(c)
	if !exists {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("从上下文中获取用户ID失败")
		response.Error(c, errx.New(constant.Unauthorized, "未授权访问", nil))
		return
	}

	var req dto.UserProfileUpdateRequest
	if err := bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.Uint64("req_Id", uint64(id)),
		slog.String("req_body", utils.LogStruct(req)))

	if err := h.UserSer.UpdateProfile(ctx, id, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdatePassword 更新当前用户密码
// @Summary 更新当前用户密码
// @Description 更新当前登录用户的密码
// @Tags 用户管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body dto.UserPasswordUpdateRequest true "密码更新参数"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response "请求参数错误或旧密码不正确"
// @Failure 401 {object} dto.Response "未授权"
// @Failure 500 {object} dto.Response "内部服务器错误"
// @Router /api/v1/auth/users/current/password [put]
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	id, exists := contextx.GetUserIDFromRequestContext(c)
	if !exists {
		contextx.GetLoggerFromContext(c.Request.Context()).Error("从上下文中获取用户ID失败")
		response.Error(c, errx.New(constant.Unauthorized, "未授权访问", nil))
		return
	}

	var req dto.UserPasswordUpdateRequest
	if err := bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.Uint64("req_Id", uint64(id)),
		slog.String("req_body", "password_update_redacted")) // 不记录密码明文

	if err := h.UserSer.UpdatePassword(ctx, id, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// ============================================================================
// 管理员专用方法（不需要CompanyID和Role，用于/api/v1/admin-auth路由组）
// 这些方法直接操作User实体，不涉及公司关联和角色管理
// ============================================================================

// CreateByAdmin 管理员创建用户（不创建公司关联）
// @Summary 管理员创建用户
// @Tags 管理员-用户管理
// @Accept json
// @Produce json
// @Param request body dto.UserCreateRequest true "创建参数（不需要Role和CompanyID）"
// @Success 200 {object} dto.Response{data=dto.UserResponse}
// @Router /api/v1/admin-auth/users [post]
func (h *UserHandler) CreateByAdmin(c *gin.Context) {
	var req dto.UserCreateRequest
	if err := bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	// 管理员路由不需要 Role 和 CompanyID，忽略这些字段
	req.Role = ""
	req.CompanyID = 0

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))

	resp, err := h.UserSer.CreateByAdmin(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// UpdateByAdmin 管理员更新用户（不检查公司，不更新角色）
// @Summary 管理员更新用户
// @Tags 管理员-用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body dto.UserUpdateRequest true "更新参数（不需要Role和CompanyID）"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/users/{id} [put]
func (h *UserHandler) UpdateByAdmin(c *gin.Context) {
	id, err := parseUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.UserUpdateRequest
	if err := bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	// 管理员路由不需要 Role 和 CompanyID，忽略这些字段
	req.Role = nil
	req.CompanyID = 0

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Uint64("req_Id", uint64(id)), slog.String("req_body", utils.LogStruct(req)))

	if err := h.UserSer.UpdateByAdmin(ctx, id, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// DeleteByAdmin 管理员删除用户（直接删除用户实体）
// @Summary 管理员删除用户
// @Tags 管理员-用户管理
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/users/{id} [delete]
func (h *UserHandler) DeleteByAdmin(c *gin.Context) {
	id, err := parseUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Uint64("req_Id", uint64(id)))

	if err := h.UserSer.DeleteByAdmin(ctx, id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// ListByAdmin 管理员查询用户列表（不按公司过滤）
// @Summary 管理员获取用户列表
// @Tags 管理员-用户管理
// @Produce json
// @Param q query string false "综合查询条件"
// @Param username query string false "用户名"
// @Param nickname query string false "昵称"
// @Param email query string false "邮箱"
// @Param phone query string false "手机号"
// @Param status query int false "状态(0:禁用,1:启用)"
// @Param gender query string false "性别(male/female/unknown)"
// @Param minAge query int false "最小年龄"
// @Param maxAge query int false "最大年龄"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.UserResponse}}
// @Router /api/v1/admin-auth/users [get]
func (h *UserHandler) ListByAdmin(c *gin.Context) {
	var filter dto.UserFilter
	if err := bindQueryAndValidate(c, &filter); err != nil {
		response.Error(c, err)
		return
	}

	// 管理员路由不需要 CompanyID，确保为 nil
	filter.CompanyID = nil

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))

	list, total, err := h.UserSer.ListByAdmin(ctx, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// BatchDeleteByAdmin 管理员批量删除用户（直接删除用户实体）
// @Summary 管理员批量删除用户
// @Tags 管理员-用户管理
// @Accept json
// @Produce json
// @Param request body dto.UserDeleteRequest true "删除ID列表（不需要CompanyID）"
// @Success 200 {object} dto.Response
// @Router /api/v1/admin-auth/users [delete]
func (h *UserHandler) BatchDeleteByAdmin(c *gin.Context) {
	var req dto.UserDeleteRequest
	if err := bindAndValidate(c, &req); err != nil {
		response.Error(c, err)
		return
	}

	// 管理员路由不需要 CompanyID
	req.CompanyID = 0

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))

	if err := h.UserSer.BatchDeleteByAdmin(ctx, req.IDs); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}
