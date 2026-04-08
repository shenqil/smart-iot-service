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
	"fmt"
)

// DeviceTypeHandler 设备类型处理器
type DeviceTypeHandler struct {
	DeviceTypeSer *service.DeviceTypeService
}

// CreateDeviceType 创建设备类型
// @Summary 创建设备类型
// @Tags 设备类型管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body dto.DeviceTypeCreateRequest true "创建参数"
// @Success 200 {object} dto.Response{data=dto.DeviceTypeResponse}
// @Failure 400 {object} dto.Response
// @Failure 409 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/device-types [post]
func (h *DeviceTypeHandler) CreateDeviceType(c *gin.Context) {
	var req dto.DeviceTypeCreateRequest
	logger := contextx.GetLoggerFromContext(c.Request.Context())

	// 参数绑定
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(fmt.Sprintf("参数绑定失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, constant.MsgInvalidInput, err))
		return
	}

	// DTO层验证
	if err := req.Validate(); err != nil {
		logger.Error(fmt.Sprintf("参数验证失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, err.Error(), nil))
		return
	}

	logger.Info(fmt.Sprintf("CreateDeviceType request parameters: %s", utils.LogStruct(req)))

	resp, err := h.DeviceTypeSer.CreateDeviceType(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// GetDeviceType 获取设备类型详情（路由已删除，方法保留）
// @Summary 获取设备类型详情
// @Tags 设备类型管理
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "设备类型ID"
// @Success 200 {object} dto.Response{data=dto.DeviceTypeResponse}
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/device-types/{id} [get]
func (h *DeviceTypeHandler) GetDeviceType(c *gin.Context) {
	idStr := c.Param("id")
	logger := contextx.GetLoggerFromContext(c.Request.Context())

	// 参数验证
	if idStr == "" {
		logger.Error("ID 参数为空")
		response.Error(c, errx.New(constant.BadRequest, "ID参数不能为空", nil))
		return
	}

	logger.Info(fmt.Sprintf("GetDeviceType request ID: %s", idStr))

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Error(fmt.Sprintf("ID 格式错误: %v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	resp, err := h.DeviceTypeSer.GetDeviceType(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// GetDeviceTypeByType 根据类型标识获取设备类型详情（路由已删除，方法保留）
// @Summary 根据类型标识获取设备类型详情
// @Tags 设备类型管理
// @Security ApiKeyAuth
// @Produce json
// @Param type path string true "设备类型标识"
// @Success 200 {object} dto.Response{data=dto.DeviceTypeResponse}
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/device-types/type/{type} [get]
func (h *DeviceTypeHandler) GetDeviceTypeByType(c *gin.Context) {
	typ := c.Param("type")
	logger := contextx.GetLoggerFromContext(c.Request.Context())

	// 参数验证
	if typ == "" {
		logger.Error("类型参数为空")
		response.Error(c, errx.New(constant.BadRequest, "类型参数不能为空", nil))
		return
	}

	logger.Info(fmt.Sprintf("GetDeviceTypeByType request type: %s", typ))

	resp, err := h.DeviceTypeSer.GetDeviceTypeByType(c.Request.Context(), typ)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// ListDeviceTypes 分页查询设备类型列表
// @Summary 分页查询设备类型列表
// @Tags 设备类型管理
// @Security ApiKeyAuth
// @Produce json
// @Param name query string false "名称模糊匹配"
// @Param q query string false "全部搜索参数"
// @Param pageNumber query int false "页码" default(1)
// @Param pageSize query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.DeviceTypeResponse}}
// @Failure 400 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/device-types [get]
func (h *DeviceTypeHandler) ListDeviceTypes(c *gin.Context) {
	var filter dto.DeviceTypeFilter
	logger := contextx.GetLoggerFromContext(c.Request.Context())

	if err := c.ShouldBindQuery(&filter); err != nil {
		logger.Error(fmt.Sprintf("查询参数错误：%v", err))
		response.Error(c, errx.New(constant.BadRequest, constant.MsgInvalidInput, nil))
		return
	}

	logger.Info(fmt.Sprintf("ListDeviceTypes request parameters: %s", utils.LogStruct(filter)))

	list, total, err := h.DeviceTypeSer.ListDeviceTypes(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// UpdateDeviceType 更新设备类型
// @Summary 更新设备类型
// @Tags 设备类型管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "设备类型ID"
// @Param request body dto.DeviceTypeUpdateRequest true "更新参数"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 409 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/device-types/{id} [put]
func (h *DeviceTypeHandler) UpdateDeviceType(c *gin.Context) {
	idStr := c.Param("id")
	logger := contextx.GetLoggerFromContext(c.Request.Context())

	// 参数验证
	if idStr == "" {
		logger.Error("ID 参数为空")
		response.Error(c, errx.New(constant.BadRequest, "ID参数不能为空", nil))
		return
	}

	logger.Info(fmt.Sprintf("UpdateDeviceType request ID: %s", idStr))

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Error(fmt.Sprintf("ID 格式错误: %v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.DeviceTypeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(fmt.Sprintf("参数绑定失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, constant.MsgInvalidInput, err))
		return
	}

	// DTO层验证
	if err := req.Validate(); err != nil {
		logger.Error(fmt.Sprintf("参数验证失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, err.Error(), nil))
		return
	}

	logger.Info(fmt.Sprintf("UpdateDeviceType request parameters: %s", utils.LogStruct(req)))

	if err := h.DeviceTypeSer.UpdateDeviceType(c.Request.Context(), uint(id), req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// DeleteDeviceType 删除设备类型
// @Summary 删除设备类型
// @Tags 设备类型管理
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "设备类型ID"
// @Success 200 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/device-types/{id} [delete]
func (h *DeviceTypeHandler) DeleteDeviceType(c *gin.Context) {
	idStr := c.Param("id")
	logger := contextx.GetLoggerFromContext(c.Request.Context())

	// 参数验证
	if idStr == "" {
		logger.Error("ID 参数为空")
		response.Error(c, errx.New(constant.BadRequest, "ID参数不能为空", nil))
		return
	}

	logger.Info(fmt.Sprintf("DeleteDeviceType request ID: %s", idStr))

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		logger.Error(fmt.Sprintf("ID 格式错误: %v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	if err := h.DeviceTypeSer.DeleteDeviceType(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// BatchDeleteDeviceTypes 批量删除设备类型
// @Summary 批量删除设备类型
// @Tags 设备类型管理
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body dto.DeviceTypeDeleteRequest true "删除ID列表"
// @Success 200 {object} dto.Response
// @Failure 400 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/v1/admin-auth/device-types [delete]
func (h *DeviceTypeHandler) BatchDeleteDeviceTypes(c *gin.Context) {
	var req dto.DeviceTypeDeleteRequest
	logger := contextx.GetLoggerFromContext(c.Request.Context())

	// 参数绑定
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(fmt.Sprintf("参数绑定失败：%v", err))
		response.Error(c, errx.New(constant.BadRequest, constant.MsgInvalidInput, err))
		return
	}

	// 参数验证
	if len(req.IDs) == 0 {
		logger.Error("删除ID列表为空")
		response.Error(c, errx.New(constant.BadRequest, "删除ID列表不能为空", nil))
		return
	}

	logger.Info(fmt.Sprintf("BatchDeleteDeviceTypes request IDs: %v", req.IDs))

	if err := h.DeviceTypeSer.BatchDeleteDeviceTypes(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// ListAllDeviceTypes 获取所有设备类型列表（不包含ID）
// @Summary 获取所有设备类型列表
// @Tags 设备类型管理
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} dto.Response{data=[]dto.DeviceTypeSimpleResponse}
// @Failure 500 {object} dto.Response
// @Router /api/v1/auth/device-types/all [get]
func (h *DeviceTypeHandler) ListAllDeviceTypes(c *gin.Context) {
	ctx := c.Request.Context()
	logger := contextx.GetLoggerFromContext(ctx)

	deviceTypes, err := h.DeviceTypeSer.ListAllDeviceTypes(ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("获取设备类型列表失败: %v", err))
		response.Error(c, errx.New(constant.InternalServerError, "获取设备类型列表失败", err))
		return
	}

	response.Success(c, deviceTypes)
}
