package handler

import (
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

type DeviceAlarmHandler struct {
	DeviceAlarmSer *service.DeviceAlarmService
}

// List 获取告警列表（DeviceAlarm）
// @Summary 获取设备告警列表
// @Tags 设备告警
// @Accept json
// @Produce json
// @Param q query string false "模糊搜索：设备名称、设备ID"
// @Param enterpriseId query string false "企业ID"
// @Param deviceId query string false "设备ID"
// @Param alarmType query string false "告警类型"
// @Param alarmLevel query int false "告警级别"
// @Param isProcessed query int false "是否已处理(0/1)"
// @Param processedBy query uint false "处理人ID"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Param pageNum query int false "页码" default(1)
// @Param pageSize query int false "每页大小" default(10)
// @Success 200 {object} dto.Response{data=[]dto.DeviceAlarmResponse}
// @Router /api/v1/auth/device-alarms [get]
func (h *DeviceAlarmHandler) List(c *gin.Context) {
	var req dto.DeviceAlarmSearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		contextx.GetLoggerFromContext(c.Request.Context()).Error(fmt.Sprintf("%v", err))
		response.Error(c, errx.New(constant.BadRequest, "参数错误", err))
		return
	}

	// 校验时间范围
	if req.StartTime != nil && req.EndTime != nil && req.StartTime.After(*req.EndTime) {
		response.Error(c, errx.New(constant.BadRequest, "开始时间不能大于结束时间", nil))
		return
	}

	ctx := contextx.SetLoggerToContext(c.Request.Context(), slog.String("req_query", utils.LogStruct(req)))

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	req.CompanyID = companyID

	alerts, total, err := h.DeviceAlarmSer.List(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginate(c, alerts, total)
}

// BatchProcess 批量处理告警（DeviceAlarm）
// @Summary 批量处理设备告警
// @Tags 设备告警
// @Accept json
// @Produce json
// @Param request body dto.DeviceAlarmBatchProcessRequest true "处理请求"
// @Success 200 {object} dto.Response{data=dto.DeviceAlarmBatchProcessResponse}
// @Router /api/v1/auth/device-alarms/batch-process [post]
func (h *DeviceAlarmHandler) BatchProcess(c *gin.Context) {
	var req dto.DeviceAlarmBatchProcessRequest
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
	req.UserID = userID

	companyID, err := getRequiredCompanyID(c)
	if err != nil {
		response.Error(c, err)
		return
	}
	req.CompanyID = companyID

	resp, err := h.DeviceAlarmSer.BatchProcess(ctx, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp)
}
