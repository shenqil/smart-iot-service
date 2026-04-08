package handler

import (
	"fmt"
	"rbac-admin/internal/handler/response"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/service"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"rbac-admin/pkg/utils"

	"log/slog"

	"github.com/gin-gonic/gin"
)

type MyManagementHandler struct {
	Service                  *service.MyManagementService
	DeviceDataService        *service.DeviceDataService
	DeviceRelayRecordService *service.DeviceRelayRecordService
}

// ListMySites 获取我的园区列表
// @Summary 获取我的园区列表
// @Tags 我的管理
// @Produce json
// @Param siteName query string false "园区名称模糊搜索"
// @Param siteCode query string false "园区编码模糊搜索"
// @Param pageNumber query int false "页码" default(1)
// @Param pageSize query int false "每页大小" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.MySiteSimpleResponse}}
// @Router /api/v1/auth/my/sites [get]
func (h *MyManagementHandler) ListMySites(c *gin.Context) {
	var filter dto.MySiteFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}
	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter.CompanyID = companyID

	userID, err := getRequiredUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter.UserID = userID

	list, total, err := h.Service.ListMySites(ctx, &filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// ListMyDevices 获取我的设备列表
// @Summary 获取我的设备列表
// @Tags 我的管理
// @Produce json
// @Param siteCode query []string false "园区编码数组" collectionFormat(multi)
// @Param deviceType query []string false "设备类型数组" collectionFormat(multi)
// @Param deviceName query string false "设备名称模糊搜索"
// @Param q query string false "综合查询条件"
// @Param pageNumber query int false "页码" default(1)
// @Param pageSize query int false "每页大小" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.MyDeviceResponse}}
// @Router /api/v1/auth/my/devices [get]
func (h *MyManagementHandler) ListMyDevices(c *gin.Context) {
	var filter dto.MyDeviceFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter.CompanyID = companyID

	list, total, err := h.Service.ListMyDevices(ctx, &filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, list, total)
}

// GetRealTimeData 获取设备实时数据
// @Summary 获取设备实时数据
// @Tags 我的管理
// @Produce json
// @Param deviceSN path string true "设备序列号"
// @Success 200 {object} dto.Response{data=dto.DeviceRealTimeData}
// @Router /api/v1/auth/my/device/{deviceSN}/realtime [get]
func (h *MyManagementHandler) GetRealTimeData(c *gin.Context) {
	deviceSN := c.Param("deviceSN")
	if deviceSN == "" {
		response.Error(c, errx.New(constant.BadRequest, "设备序列号不能为空", nil))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("deviceSN", deviceSN))

	data, err := h.DeviceDataService.GetRealTimeData(ctx, deviceSN)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, data)
}

// GetSensorHistoryData 获取传感器历史数据
// @Summary 获取传感器历史数据
// @Tags 我的管理
// @Produce json
// @Param deviceSN query string true "设备序列号"
// @Param startTime query int false "开始时间(秒级时间戳)"
// @Param endTime query int false "结束时间(秒级时间戳)"
// @Param interval query int false "时间间隔(分钟)"
// @Param columns query []string false "需要对比的传感器列名" collectionFormat(multi)
// @Param pagination query dto.PaginationRequest false "分页参数"
// @Success 200 {object} dto.Response{data=[]map[string]interface{}}
// @Router /api/v1/auth/my/device/sensors/history [get]
func (h *MyManagementHandler) GetSensorHistoryData(c *gin.Context) {
	var filter dto.SensorHistoryFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))

	dataPoints, err := h.DeviceDataService.GetSensorHistoryData(ctx, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dataPoints)
}

// SwitchRelay 控制继电器开关
// @Summary 控制继电器开关状态
// @Tags 设备控制
// @Accept json
// @Produce json
// @Param request body dto.RelaySwitchRequest true "继电器控制参数"
// @Success 200 {object} dto.Response{data=string}
// @Router /api/v1/auth/my/device/relay/switch [post]
func (h *MyManagementHandler) SwitchRelay(c *gin.Context) {
	var req dto.RelaySwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_body", utils.LogStruct(req)))

	// 从上下文获取用户ID（必填）
	userID, err := getRequiredUserID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	req.OperatorUserID = userID

	// Service层会进行完整的参数验证，这里只做基本的JSON绑定验证
	if err := h.DeviceRelayRecordService.SwitchRelay(ctx, &req); err != nil {
		response.Error(c, err)
		return
	}

	action := "关闭"
	if req.IsOpen {
		action = "开启"
	}
	response.Success(c, fmt.Sprintf("成功%s继电器%d", action, req.RelayIndex))
}

// ListDeviceRelayRecords 获取设备继电器操作记录列表
// @Summary 获取设备继电器操作记录列表
// @Tags 我的管理
// @Produce json
// @Param deviceSN query int false "设备号"
// @Param channel query int false "通道号(0-31)"
// @Param operator query string false "操作者"
// @Param status query string false "操作状态"
// @Param startTime query int false "开始时间(秒级时间戳)"
// @Param endTime query int false "结束时间(秒级时间戳)"
// @Param pageNumber query int false "页码" default(1)
// @Param pageSize query int false "每页大小" default(10)
// @Success 200 {object} dto.Response{data=dto.PaginationResponse{records=[]dto.DeviceRelayRecordResponse}}
// @Router /api/v1/auth/my/device/relay/records [get]
func (h *MyManagementHandler) ListDeviceRelayRecords(c *gin.Context) {
	var filter dto.DeviceRelayRecordFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))

	records, total, err := h.DeviceRelayRecordService.ListDeviceRelayRecords(ctx, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, records, total)
}

// GetSensorComparison 获取传感器日环比/月环比数据
// @Summary 获取传感器日环比/月环比数据
// @Tags 我的管理
// @Produce json
// @Param deviceSN query string true "设备序列号"
// @Param type query string true "对比类型: day/month"
// @Param timestamp query int true "基准时间戳(毫秒级)"
// @Param columns query []string true "需要对比的传感器列名" collectionFormat(multi)
// @Success 200 {object} dto.Response{data=dto.SensorComparisonResult}
// @Router /api/v1/auth/my/device/sensors/comparison [get]
func (h *MyManagementHandler) GetSensorComparison(c *gin.Context) {
	var filter dto.SensorComparisonFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	// Validate columns
	if len(filter.Columns) == 0 {
		response.Error(c, errx.New(constant.BadRequest, "至少需要指定一个传感器列名", nil))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(),
		slog.String("req_filter", utils.LogStruct(filter)))

	result, err := h.DeviceDataService.GetSensorComparison(ctx, filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// GetMyDeviceTypeStats 获取我的设备类型统计
// @Summary 获取我的设备类型统计
// @Tags 我的管理
// @Produce json
// @Param siteCode query []string false "园区编码数组，为空则查询所有园区" collectionFormat(multi)
// @Success 200 {object} dto.Response{data=[]dto.MyDeviceTypeStatsResponse}
// @Router /api/v1/auth/my/devices/type-stats [get]
func (h *MyManagementHandler) GetMyDeviceTypeStats(c *gin.Context) {
	var filter dto.MyDeviceTypeStatsFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_filter", utils.LogStruct(filter)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	filter.CompanyID = companyID

	stats, err := h.Service.GetMyDeviceTypeStats(ctx, &filter)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, stats)
}
