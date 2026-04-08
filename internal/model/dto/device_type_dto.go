package dto

import (
	"errors"
	"rbac-admin/internal/model/entity"
	"strings"

	"gorm.io/datatypes"
)

// DeviceTypeDeleteRequest 批量删除请求
type DeviceTypeDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// DeviceTypeFilter 查询过滤器
type DeviceTypeFilter struct {
	Name              string           `form:"name,omitempty"`        // 名称模糊匹配
	Q                 string           `form:"q,omitempty"`           // 全部搜索参数
	PaginationRequest `form:",inline"` // 分页参数
}

// ConvertToQuery 将过滤器转换为实体查询
func (f *DeviceTypeFilter) ConvertToQuery() entity.DeviceTypeQuery {
	pageNumber := f.PageNumber
	if pageNumber <= 0 {
		pageNumber = 1
	}

	pageSize := f.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	return entity.DeviceTypeQuery{
		Name: f.Name,
		Q:    f.Q,
		Pagination: entity.Pagination{
			PageNumber: pageNumber,
			PageSize:   pageSize,
		},
	}
}

// DeviceTypeCreateRequest 创建请求
type DeviceTypeCreateRequest struct {
	Type        string         `json:"type" binding:"required,min=1,max=50"`
	Name        string         `json:"name" binding:"required,min=1,max=100"`
	Description string         `json:"description" binding:"max=500"`
	Icon        string         `json:"icon" binding:"max=255"`
	Schema      datatypes.JSON `json:"schema" binding:"required"`
}

// DeviceTypeUpdateRequest 更新请求
type DeviceTypeUpdateRequest struct {
	Type        *string         `json:"type,omitempty" binding:"omitempty,min=1,max=50"`
	Name        *string         `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Description *string         `json:"description,omitempty" binding:"omitempty,max=500"`
	Icon        *string         `json:"icon,omitempty" binding:"omitempty,max=255"`
	Schema      *datatypes.JSON `json:"schema,omitempty"`
}

// DeviceTypeResponse 响应结构
type DeviceTypeResponse struct {
	ID          uint           `json:"id"`
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Icon        string         `json:"icon"`
	Schema      datatypes.JSON `json:"schema"`
	CreatedAt   int64          `json:"createdAt"` // 秒级时间戳
	UpdatedAt   int64          `json:"updatedAt"` // 秒级时间戳
}

// DeviceTypeSimpleResponse 设备类型简单响应结构（不包含ID）
type DeviceTypeSimpleResponse struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Icon        string         `json:"icon"`
	Schema      datatypes.JSON `json:"schema"`
	CreatedAt   int64          `json:"createdAt"`
	UpdatedAt   int64          `json:"updatedAt"`
}

// ConvertToEntity 将创建请求转换为实体
func (req *DeviceTypeCreateRequest) ConvertToEntity() *entity.DeviceType {
	return &entity.DeviceType{
		Type:        strings.TrimSpace(req.Type),
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Icon:        strings.TrimSpace(req.Icon),
		Schema:      req.Schema,
	}
}

// Validate 验证创建请求
func (req *DeviceTypeCreateRequest) Validate() error {
	if strings.TrimSpace(req.Type) == "" {
		return errors.New("类型标识不能为空")
	}
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("类型名称不能为空")
	}
	if len(req.Type) > 50 {
		return errors.New("类型标识长度不能超过50个字符")
	}
	if len(req.Name) > 100 {
		return errors.New("类型名称长度不能超过100个字符")
	}
	if len(req.Description) > 500 {
		return errors.New("描述长度不能超过500个字符")
	}
	if len(req.Icon) > 255 {
		return errors.New("图标URL长度不能超过255个字符")
	}
	return nil
}

// ConvertToDeviceTypeResponse 将实体转换为响应
func ConvertToDeviceTypeResponse(deviceType *entity.DeviceType) *DeviceTypeResponse {
	return &DeviceTypeResponse{
		ID:          deviceType.ID,
		Type:        deviceType.Type,
		Name:        deviceType.Name,
		Description: deviceType.Description,
		Icon:        deviceType.Icon,
		Schema:      deviceType.Schema,
		CreatedAt:   deviceType.CreatedAt.Unix(),
		UpdatedAt:   deviceType.UpdatedAt.Unix(),
	}
}

// Validate 验证更新请求
func (req *DeviceTypeUpdateRequest) Validate() error {
	if req.Type != nil {
		if strings.TrimSpace(*req.Type) == "" {
			return errors.New("类型标识不能为空")
		}
		if len(*req.Type) > 50 {
			return errors.New("类型标识长度不能超过50个字符")
		}
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return errors.New("类型名称不能为空")
		}
		if len(*req.Name) > 100 {
			return errors.New("类型名称长度不能超过100个字符")
		}
	}
	if req.Description != nil && len(*req.Description) > 500 {
		return errors.New("描述长度不能超过500个字符")
	}
	if req.Icon != nil && len(*req.Icon) > 255 {
		return errors.New("图标URL长度不能超过255个字符")
	}
	return nil
}

// ConvertToDeviceTypeSimpleResponse 将实体转换为简单响应（不包含ID）
func ConvertToDeviceTypeSimpleResponse(deviceType *entity.DeviceType) *DeviceTypeSimpleResponse {
	return &DeviceTypeSimpleResponse{
		Type:        deviceType.Type,
		Name:        deviceType.Name,
		Description: deviceType.Description,
		Icon:        deviceType.Icon,
		Schema:      deviceType.Schema,
		CreatedAt:   deviceType.CreatedAt.Unix(),
		UpdatedAt:   deviceType.UpdatedAt.Unix(),
	}
}
