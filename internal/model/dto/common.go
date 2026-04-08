package dto

import (
	"encoding/json"

	"gorm.io/datatypes"
)

// PaginationRequest 分页请求参数（通用）
type PaginationRequest struct {
	PageNumber int `form:"pageNumber" binding:"omitempty,min=1" default:"1"`        // 当前页码
	PageSize   int `form:"pageSize" binding:"omitempty,min=1,max=100" default:"10"` // 每页大小
}

// GetPageNumber 安全获取页码
func (p *PaginationRequest) GetPageNumber() int {
	if p.PageNumber <= 0 {
		return 1
	}
	return p.PageNumber
}

// GetPageSize 安全获取每页大小
func (p *PaginationRequest) GetPageSize() int {
	if p.PageSize <= 0 {
		return 10
	}
	return p.PageSize
}

// PaginationResponse 分页响应结构（通用）
type PaginationResponse struct {
	Records    interface{} `json:"records"`    // 数据列表
	TotalCount int64       `json:"totalCount"` // 总记录数
}

// Response 统一响应结构（通用）
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ToJSON 将结构体转换为datatypes.JSON
func ToJSON(v interface{}) (datatypes.JSON, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(data), nil
}

// FromJSON 从datatypes.JSON解析到结构体
func FromJSON(data datatypes.JSON, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		return err
	}
	return nil
}
