package dto

import (
	"rbac-admin/internal/model/entity"
)

// StandardSensorCreateRequest 创建请求
type StandardSensorCreateRequest struct {
	Name       string  `json:"name" binding:"required,min=1,max=100"`
	MinValue   float64 `json:"minValue" binding:"omitempty"`
	MaxValue   float64 `json:"maxValue" binding:"omitempty"`
	Resolution float64 `json:"resolution" binding:"omitempty"`
	Unit       string  `json:"unit" binding:"omitempty,max=20"`
	URL        string  `json:"url" binding:"omitempty,max=255"`
}

// StandardSensorUpdateRequest 更新请求
type StandardSensorUpdateRequest struct {
	Name       *string  `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	MinValue   *float64 `json:"minValue,omitempty" binding:"omitempty"`
	MaxValue   *float64 `json:"maxValue,omitempty" binding:"omitempty"`
	Resolution *float64 `json:"resolution,omitempty" binding:"omitempty"`
	Unit       *string  `json:"unit,omitempty" binding:"omitempty,max=20"`
	URL        *string  `json:"url,omitempty" binding:"omitempty,max=255"`
}

// StandardSensorDeleteRequest 批量删除请求
type StandardSensorDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// StandardSensorFilter 查询过滤器
type StandardSensorFilter struct {
	Name     string `form:"name,omitempty"` // 名称搜索参数
	Q        string `form:"q,omitempty"`    // 全部搜索参数
	PaginationRequest
}

// StandardSensorResponse 响应结构
type StandardSensorResponse struct {
	ID         uint    `json:"id"`
	Name       string  `json:"name"`
	MinValue   float64 `json:"minValue"`
	MaxValue   float64 `json:"maxValue"`
	Resolution float64 `json:"resolution"`
	Unit       string  `json:"unit"`
	URL        string  `json:"url,omitempty"`
	CreatedAt  int64   `json:"createdAt"`
	UpdatedAt  int64   `json:"updatedAt"`
}

// ConvertToEntity 将创建请求转换为实体
func (req *StandardSensorCreateRequest) ConvertToEntity() *entity.StandardSensor {
	return &entity.StandardSensor{
		Name:       req.Name,
		MinValue:   req.MinValue,
		MaxValue:   req.MaxValue,
		Resolution: req.Resolution,
		Unit:       req.Unit,
		URL:        req.URL,
	}
}

// ConvertToQuery 将过滤器转换为实体查询
func (f *StandardSensorFilter) ConvertToQuery() entity.StandardSensorQuery {
	return entity.StandardSensorQuery{
		Name: f.Name,
		Q:    f.Q,
		Pagination: entity.Pagination{
			PageNumber: f.GetPageNumber(),
			PageSize:   f.GetPageSize(),
		},
	}
}

// ConvertToStandardSensorResponse 将实体转换为响应
func ConvertToStandardSensorResponse(sensor *entity.StandardSensor) *StandardSensorResponse {
	return &StandardSensorResponse{
		ID:         sensor.ID,
		Name:       sensor.Name,
		MinValue:   sensor.MinValue,
		MaxValue:   sensor.MaxValue,
		Resolution: sensor.Resolution,
		Unit:       sensor.Unit,
		URL:        sensor.URL,
		CreatedAt:  sensor.CreatedAt.Unix(),
		UpdatedAt:  sensor.UpdatedAt.Unix(),
	}
}

// GetPageNumber 安全获取页码
func (f *StandardSensorFilter) GetPageNumber() int {
	if f.PageNumber <= 0 {
		return 1
	}
	return f.PageNumber
}

// GetPageSize 安全获取每页大小
func (f *StandardSensorFilter) GetPageSize() int {
	if f.PageSize <= 0 {
		return 10
	}
	return f.PageSize
}

// BuildUpdateFields 构建更新字段映射
func (req *StandardSensorUpdateRequest) BuildUpdateFields() map[string]interface{} {
	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.MinValue != nil {
		updates["min_value"] = *req.MinValue
	}
	if req.MaxValue != nil {
		updates["max_value"] = *req.MaxValue
	}
	if req.Resolution != nil {
		updates["resolution"] = *req.Resolution
	}
	if req.Unit != nil {
		updates["unit"] = *req.Unit
	}
	if req.URL != nil {
		updates["url"] = *req.URL
	}

	return updates
}
