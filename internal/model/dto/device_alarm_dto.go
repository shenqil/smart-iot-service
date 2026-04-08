package dto

import (
	"rbac-admin/internal/model/entity"
	"time"
)

// DeviceAlarmSearchRequest 告警搜索请求
type DeviceAlarmSearchRequest struct {
	Q                 *string               `form:"q,omitempty"`                                            // 模糊搜索：设备名称、设备ID
	EnterpriseId      *string               `form:"enterpriseId,omitempty"`                                 // 企业ID
	DeviceId          *string               `form:"deviceId,omitempty"`                                     // 设备ID
	SiteCode          *string               `form:"siteCode,omitempty"`                                     // 园区编码
	AlarmType         *string               `form:"alarmType,omitempty"`                                    // 告警类型
	AlarmLevel        *entity.AlarmLevel    `form:"alarmLevel,omitempty" binding:"omitempty,oneof=1 2 3 4"` // 告警级别
	IsProcessed       *entity.ProcessStatus `form:"isProcessed,omitempty" binding:"omitempty,oneof=0 1"`    // 是否已处理(0/1)
	ProcessedBy       *uint                 `form:"processedBy,omitempty"`                                  // 处理人ID
	StartTime         *time.Time            `form:"startTime,omitempty"`                                    // 开始时间（创建时间）
	EndTime           *time.Time            `form:"endTime,omitempty"`                                      // 结束时间（创建时间）
	CompanyID         uint                  `form:"-" json:"-"`                                             // 公司ID（从上下文获取，不暴露给用户）
	PaginationRequest `form:",inline"`      // 分页
}

// GetPageNumber 安全获取页码
func (r *DeviceAlarmSearchRequest) GetPageNumber() int {
	if r.PageNumber <= 0 {
		return 1
	}
	return r.PageNumber
}

// GetPageSize 安全获取每页大小
func (r *DeviceAlarmSearchRequest) GetPageSize() int {
	if r.PageSize <= 0 {
		return 10
	}
	if r.PageSize > 100 {
		return 100
	}
	return r.PageSize
}

// ToQuery 转换为实体查询
func (r *DeviceAlarmSearchRequest) ToQuery() entity.DeviceAlarmQuery {
	return entity.DeviceAlarmQuery{
		Q:            r.Q,
		EnterpriseId: r.EnterpriseId,
		DeviceId:     r.DeviceId,
		SiteCode:     r.SiteCode,
		AlarmType:    r.AlarmType,
		AlarmLevel:   r.AlarmLevel,
		IsProcessed:  r.IsProcessed,
		ProcessedBy:  r.ProcessedBy,
		StartTime:    r.StartTime,
		EndTime:      r.EndTime,
		Pagination: entity.Pagination{
			PageNumber: r.GetPageNumber(),
			PageSize:   r.GetPageSize(),
		},
	}
}

// DeviceAlarmResponse 告警响应
type DeviceAlarmResponse struct {
	ID            uint       `json:"id"`
	AlarmLevel    uint8      `json:"alarmLevel"`
	AlarmType     string     `json:"alarmType"`
	AlarmSubType  string     `json:"alarmSubType"`
	EnterpriseId  string     `json:"enterpriseId"`
	DeviceId      string     `json:"deviceId"`
	DeviceName    string     `json:"deviceName"`
	Description   string     `json:"description"`
	Value         string     `json:"value"`
	Threshold     string     `json:"threshold"`
	IsProcessed   uint8      `json:"isProcessed"`
	ProcessedBy   *uint      `json:"processedBy,omitempty"`
	ProcessorName string     `json:"processorName"`
	ProcessRemark string     `json:"processRemark"`
	ProcessedAt   *time.Time `json:"processedAt,omitempty"`
	CreatedAt     int64      `json:"createdAt"`
}

// DeviceAlarmBatchProcessRequest 批量处理告警请求
type DeviceAlarmBatchProcessRequest struct {
	IDs           []uint `json:"ids"`                              // 告警ID列表，为空时一键处理所有
	ProcessorName string `json:"processorName" binding:"required"` // 处理人姓名
	ProcessRemark string `json:"processRemark"`                    // 处理备注
	UserID        uint   `json:"-" form:"-"`                       // 用户ID（从上下文获取，不暴露给用户）
	CompanyID     uint   `json:"-" form:"-"`                       // 公司ID（从上下文获取，不暴露给用户）
}

// DeviceAlarmBatchProcessResponse 批量处理告警响应
type DeviceAlarmBatchProcessResponse struct {
	ProcessedCount int64 `json:"processedCount"`
}

// ConvertToDeviceAlarmResponse 将实体转换为响应
func ConvertToDeviceAlarmResponse(a *entity.DeviceAlarm) *DeviceAlarmResponse {
	resp := &DeviceAlarmResponse{
		ID:            a.ID,
		AlarmLevel:    uint8(a.AlarmLevel),
		AlarmType:     a.AlarmType,
		AlarmSubType:  a.AlarmSubType,
		EnterpriseId:  a.EnterpriseId,
		DeviceId:      a.DeviceId,
		DeviceName:    a.DeviceName,
		Description:   a.Description,
		Value:         a.Value,
		Threshold:     a.Threshold,
		IsProcessed:   uint8(a.IsProcessed),
		ProcessedBy:   a.ProcessedBy,
		ProcessorName: a.ProcessorName,
		ProcessRemark: a.ProcessRemark,
		ProcessedAt:   a.ProcessedAt,
		CreatedAt:     a.CreatedAt.Unix(),
	}
	return resp
}
