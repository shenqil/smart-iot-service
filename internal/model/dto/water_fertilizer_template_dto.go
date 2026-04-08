package dto

import (
	"rbac-admin/internal/model/entity"
)

// WaterFertilizerTemplateCreateRequest 创建请求
type WaterFertilizerTemplateCreateRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=100"`
	Template string `json:"template" binding:"required"`
}

// WaterFertilizerTemplateUpdateRequest 更新请求
type WaterFertilizerTemplateUpdateRequest struct {
	Name     *string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Template *string `json:"template,omitempty" binding:"omitempty"`
}

// WaterFertilizerTemplateDeleteRequest 批量删除请求
type WaterFertilizerTemplateDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// WaterFertilizerTemplateFilter 查询过滤器
type WaterFertilizerTemplateFilter struct {
	Q string `form:"q,omitempty"`
	PaginationRequest
}

// WaterFertilizerTemplateResponse 响应结构
type WaterFertilizerTemplateResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Template  string `json:"template"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// ConvertToEntity 将创建请求转换为实体
func (req *WaterFertilizerTemplateCreateRequest) ConvertToEntity() *entity.WaterFertilizerTemplate {
	return &entity.WaterFertilizerTemplate{
		Name:     req.Name,
		Template: req.Template,
	}
}

// ConvertToQuery 将过滤器转换为实体查询
func (f *WaterFertilizerTemplateFilter) ConvertToQuery() entity.WaterFertilizerTemplateQuery {
	return entity.WaterFertilizerTemplateQuery{
		Q: f.Q,
		Pagination: entity.Pagination{
			PageNumber: f.GetPageNumber(),
			PageSize:   f.GetPageSize(),
		},
	}
}

// ConvertToWaterFertilizerTemplateResponse 将实体转换为响应
func ConvertToWaterFertilizerTemplateResponse(template *entity.WaterFertilizerTemplate) *WaterFertilizerTemplateResponse {
	return &WaterFertilizerTemplateResponse{
		ID:        template.ID,
		Name:      template.Name,
		Template:  template.Template,
		CreatedAt: template.CreatedAt.Unix(),
		UpdatedAt: template.UpdatedAt.Unix(),
	}
}

// GetPageNumber 安全获取页码
func (f *WaterFertilizerTemplateFilter) GetPageNumber() int {
	if f.PageNumber <= 0 {
		return 1
	}
	return f.PageNumber
}

// GetPageSize 安全获取每页大小
func (f *WaterFertilizerTemplateFilter) GetPageSize() int {
	if f.PageSize <= 0 {
		return 10
	}
	return f.PageSize
}

// BuildUpdateFields 构建更新字段映射
func (req *WaterFertilizerTemplateUpdateRequest) BuildUpdateFields() map[string]interface{} {
	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Template != nil {
		updates["template"] = *req.Template
	}

	return updates
}
