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

type DeviceHandler struct {
	DeviceSer *service.DeviceService
}

// Create 创建设备
// @Summary 创建设备
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param request body dto.DeviceCreateRequest true "创建设备参数"
// @Success 200 {object} dto.Response{data=dto.DeviceResponse}
// @Router /api/v1/auth/devices [post]
func (h *DeviceHandler) Create(c *gin.Context) {
	var req dto.DeviceCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))

	userID, err := getRequiredUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	req.CreatorID = userID

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	req.CompanyID = companyID

	resp, err := h.DeviceSer.Create(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Get 获取设备详情
// @Summary 获取设备详情
// @Tags 设备管理
// @Produce json
// @Param id path int true "设备ID"
// @Success 200 {object} dto.Response{data=dto.DeviceResponse}
// @Router /api/v1/auth/devices/{id} [get]
func (h *DeviceHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id))

	resp, err := h.DeviceSer.Get(ctx, uint(id))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// GetByDeviceNum 通过设备编号获取详情
// @Summary 通过设备编号获取详情
// @Tags 设备管理
// @Produce json
// @Param deviceNum path int true "设备编号"
// @Success 200 {object} dto.Response{data=dto.DeviceResponse}
// @Router /api/v1/auth/devices/by-number/{deviceNum} [get]
func (h *DeviceHandler) GetByDeviceNum(c *gin.Context) {
	deviceNum, err := strconv.Atoi(c.Param("deviceNum"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "设备编号格式错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_deviceNum", deviceNum))

	resp, err := h.DeviceSer.GetByDeviceNum(ctx, uint(deviceNum))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// Update 更新设备
// @Summary 更新设备
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param id path int true "设备ID"
// @Param request body dto.DeviceUpdateRequest true "更新参数"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/devices/{id} [put]
func (h *DeviceHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.DeviceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id), slog.String("req_body", utils.LogStruct(req)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	req.CompanyID = companyID

	if err := h.DeviceSer.Update(ctx, uint(id), req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// Delete 删除设备
// @Summary 删除设备
// @Tags 设备管理
// @Produce json
// @Param id path int true "设备ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/devices/{id} [delete]
func (h *DeviceHandler) Delete(c *gin.Context) {
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

	if err := h.DeviceSer.Delete(ctx, uint(id), companyID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// List 分页列表
// @Summary 获取设备列表
// @Tags 设备管理
// @Produce json
// @Param q query string false "综合查询条件"
// @Param deviceNum query int false "设备编号"
// @Param name query string false "设备名称"
// @Param deviceType query string false "设备类型" Enums(sensor, relay, gateway)
// @Param macAddress query string false "MAC地址"
// @Param simCard query string false "SIM卡号"
// @Param isActive query bool false "是否激活"
// @Param status query string false "状态" Enums(active, inactive, fault)
// @Param readInterval query int false "读取间隔"
// @Param creatorId query int false "创建者ID"
// @Param siteId query int false "园区ID"
// @Param lng query float64 false "中心经度"
// @Param lat query float64 false "中心纬度"
// @Param radius query float64 false "搜索半径(米)"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.DeviceResponse}}
// @Router /api/v1/auth/devices [get]
func (h *DeviceHandler) List(c *gin.Context) {
	var filter dto.DeviceFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "查询参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter.CompanyID = companyID

	list, total, err := h.DeviceSer.List(ctx, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// BatchDelete 批量删除
// @Summary 批量删除设备
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param request body dto.DeviceDeleteRequest true "删除ID列表"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/devices/batch [delete]
func (h *DeviceHandler) BatchDelete(c *gin.Context) {
	var req dto.DeviceDeleteRequest
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

	if err := h.DeviceSer.BatchDelete(ctx, req); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateCameraConfig 更新摄像头配置
// @Summary 更新摄像头配置
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param id path int true "设备ID"
// @Param request body dto.UpdateCameraConfigRequest true "摄像头配置"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/devices/{id}/camera-config [put]
func (h *DeviceHandler) UpdateCameraConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.UpdateCameraConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id), slog.String("req_body", utils.LogStruct(req)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.DeviceSer.UpdateCameraConfig(ctx, uint(id), req, companyID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateSensors 更新传感器配置
// @Summary 更新传感器配置
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param id path int true "设备ID"
// @Param request body dto.UpdateSensorsRequest true "传感器配置"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/devices/{id}/sensors [put]
func (h *DeviceHandler) UpdateSensors(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.UpdateSensorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id), slog.String("req_body", utils.LogStruct(req)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.DeviceSer.UpdateSensors(ctx, uint(id), req, companyID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateRelays 更新继电器配置
// @Summary 更新继电器配置
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param id path int true "设备ID"
// @Param request body dto.UpdateRelaysRequest true "继电器配置"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/devices/{id}/relays [put]
func (h *DeviceHandler) UpdateRelays(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.UpdateRelaysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id), slog.String("req_body", utils.LogStruct(req)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.DeviceSer.UpdateRelays(ctx, uint(id), req, companyID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateLocation 更新地理坐标
// @Summary 更新地理坐标
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param id path int true "设备ID"
// @Param request body dto.UpdateLocationRequest true "地理坐标"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/devices/{id}/location [put]
func (h *DeviceHandler) UpdateLocation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id), slog.String("req_body", utils.LogStruct(req)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.DeviceSer.UpdateLocation(ctx, uint(id), req, companyID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateBasicInfo 更新设备基础信息
// @Summary 更新设备基础信息
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param id path int true "设备ID"
// @Param request body dto.UpdateBasicInfoRequest true "基础信息"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/devices/{id}/basic-info [put]
func (h *DeviceHandler) UpdateBasicInfo(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.UpdateBasicInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id), slog.String("req_body", utils.LogStruct(req)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.DeviceSer.UpdateBasicInfo(ctx, uint(id), req, companyID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// UpdateWaterFertilizerInfo 更新水肥信息
// @Summary 更新水肥信息
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param id path int true "设备ID"
// @Param request body dto.UpdateWaterFertilizerInfoRequest true "水肥信息"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/devices/{id}/water-fertilizer-info [put]
func (h *DeviceHandler) UpdateWaterFertilizerInfo(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "ID格式错误", err))
		return
	}

	var req dto.UpdateWaterFertilizerInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.Int("req_Id", id), slog.String("req_body", utils.LogStruct(req)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.DeviceSer.UpdateWaterFertilizerInfo(ctx, uint(id), req, companyID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}

// GetWaterFertilizerInfo 查询水肥信息
// @Summary 查询水肥信息
// @Tags 设备管理
// @Produce json
// @Param id path int true "设备ID"
// @Success 200 {object} dto.Response{data=dto.GetWaterFertilizerInfoResponse}
// @Router /api/v1/auth/devices/{id}/water-fertilizer-info [get]
func (h *DeviceHandler) GetWaterFertilizerInfo(c *gin.Context) {
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

	resp, err := h.DeviceSer.GetWaterFertilizerInfo(ctx, uint(id), companyID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}

// DeleteWaterFertilizerInfo 删除水肥信息
// @Summary 删除水肥信息
// @Tags 设备管理
// @Produce json
// @Param id path int true "设备ID"
// @Success 200 {object} dto.Response
// @Router /api/v1/auth/devices/{id}/water-fertilizer-info [delete]
func (h *DeviceHandler) DeleteWaterFertilizerInfo(c *gin.Context) {
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

	if err := h.DeviceSer.DeleteWaterFertilizerInfo(ctx, uint(id), companyID); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, nil)
}
