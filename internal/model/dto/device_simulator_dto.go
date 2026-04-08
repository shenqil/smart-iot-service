package dto

import (
	"strings"

	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/errx"
)

// DeviceSimulatorCreateRequest 创建请求
type DeviceSimulatorCreateRequest struct {
	DeviceSN string `json:"deviceSN" binding:"required,min=1,max=100" example:"SN123456789"`
}

// Validate 验证创建请求
func (req *DeviceSimulatorCreateRequest) Validate() error {
	deviceSN := strings.TrimSpace(req.DeviceSN)
	if deviceSN == "" {
		return errx.New(constant.BadRequest, "设备SN号不能为空", nil)
	}
	if len(deviceSN) > 100 {
		return errx.New(constant.BadRequest, "设备SN号长度不能超过100个字符", nil)
	}
	req.DeviceSN = deviceSN
	return nil
}

// ConvertToEntity 将创建请求转换为实体
func (req *DeviceSimulatorCreateRequest) ConvertToEntity() (*entity.DeviceSimulator, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	return &entity.DeviceSimulator{
		DeviceSN: req.DeviceSN,
	}, nil
}

// DeviceSimulatorResponse 响应结构
type DeviceSimulatorResponse struct {
	ID        uint   `json:"id" example:"1"`
	DeviceSN  string `json:"deviceSN" example:"SN123456789"`
	CreatedAt string `json:"createdAt" example:"2024-01-01T00:00:00Z"`
	UpdatedAt string `json:"updatedAt" example:"2024-01-01T00:00:00Z"`
}

// ConvertToDeviceSimulatorResponse 将实体转换为响应
func ConvertToDeviceSimulatorResponse(simulator *entity.DeviceSimulator) (*DeviceSimulatorResponse, error) {
	if simulator == nil {
		return nil, errx.New(constant.InternalServerError, "数据为空", nil)
	}

	return &DeviceSimulatorResponse{
		ID:        simulator.ID,
		DeviceSN:  simulator.DeviceSN,
		CreatedAt: simulator.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: simulator.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// DeviceSimulatorListRequest 列表查询请求
type DeviceSimulatorListRequest struct {
	PaginationRequest
	DeviceSN *string `form:"deviceSN" example:"SN123"` // 设备SN号模糊搜索
	Q        *string `form:"q" example:"SN123"`         // 综合查询条件
}

// ConvertToQuery 转换为查询条件
func (req *DeviceSimulatorListRequest) ConvertToQuery() entity.DeviceSimulatorQuery {
	query := entity.DeviceSimulatorQuery{
		Pagination: entity.Pagination{
			PageNumber: req.PageNumber,
			PageSize:   req.PageSize,
		},
	}

	if req.DeviceSN != nil && strings.TrimSpace(*req.DeviceSN) != "" {
		deviceSN := strings.TrimSpace(*req.DeviceSN)
		query.DeviceSN = &deviceSN
	}

	if req.Q != nil && strings.TrimSpace(*req.Q) != "" {
		q := strings.TrimSpace(*req.Q)
		query.Q = &q
	}

	return query
}

// DeviceSimulatorBatchDeleteRequest 批量删除请求
type DeviceSimulatorBatchDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1" example:"[1,2,3]"`
}

