package dto

import (
	"time"

	"rbac-admin/internal/model/entity"
)

// CompanyStatus 公司状态枚举
type CompanyStatus int8

const (
	CompanyStatusDisabled CompanyStatus = 0
	CompanyStatusEnabled  CompanyStatus = 1
)

// CompanyCreateRequest 创建公司请求
type CompanyCreateRequest struct {
	Name          string        `json:"name" binding:"required,min=2,max=100"`
	Code          string        `json:"code" binding:"required,min=2,max=50"`
	Description   string        `json:"description" binding:"max=500"`
	Logo          string        `json:"logo" binding:"omitempty,max=255"`
	Status        CompanyStatus `json:"status" binding:"oneof=0 1"`
	ContactPerson string        `json:"contactPerson" binding:"max=50"`
	ContactPhone  string        `json:"contactPhone" binding:"max=20"`
	ContactEmail  string        `json:"contactEmail" binding:"omitempty,email,max=100"`
	Address       string        `json:"address" binding:"max=200"`
	Properties    string        `json:"properties" binding:"omitempty"`
	ExpiredAt     *string       `json:"expiredAt" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

// CompanyUpdateRequest 更新公司请求
type CompanyUpdateRequest struct {
	Name          *string        `json:"name" binding:"omitempty,min=2,max=100"`
	Code          *string        `json:"code" binding:"omitempty,min=2,max=50"`
	Description   *string        `json:"description" binding:"omitempty,max=500"`
	Logo          *string        `json:"logo" binding:"omitempty,max=255"`
	Status        *CompanyStatus `json:"status" binding:"omitempty,oneof=0 1"`
	ContactPerson *string        `json:"contactPerson" binding:"omitempty,max=50"`
	ContactPhone  *string        `json:"contactPhone" binding:"omitempty,max=20"`
	ContactEmail  *string        `json:"contactEmail" binding:"omitempty,email,max=100"`
	Address       *string        `json:"address" binding:"omitempty,max=200"`
	Properties    *string        `json:"properties" binding:"omitempty"`
	ExpiredAt     *string        `json:"expiredAt" binding:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

// CompanyDeleteRequest 删除公司请求
type CompanyDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// CompanyFilter 公司查询过滤器
type CompanyFilter struct {
	Q      *string        `form:"q" binding:"omitempty,max=100"`
	Name   *string        `form:"name" binding:"omitempty,max=100"`
	Code   *string        `form:"code" binding:"omitempty,max=50"`
	Status *CompanyStatus `form:"status" binding:"omitempty,oneof=0 1"`
	PaginationRequest
}

// CompanyResponse 公司响应
type CompanyResponse struct {
	ID            uint          `json:"id"`
	Name          string        `json:"name"`
	Code          string        `json:"code"`
	Description   string        `json:"description"`
	Logo          string        `json:"logo"`
	Status        CompanyStatus `json:"status"`
	ContactPerson string        `json:"contactPerson"`
	ContactPhone  string        `json:"contactPhone"`
	ContactEmail  string        `json:"contactEmail,omitempty"`
	Address       string        `json:"address"`
	Properties    string        `json:"properties"`
	ExpiredAt     *int64        `json:"expiredAt,omitempty"`
	CreatedAt     int64         `json:"createdAt"`
	UpdatedAt     int64         `json:"updatedAt"`
}

// CompanyWithRoleResponse 公司响应（包含用户在该公司的角色）
type CompanyWithRoleResponse struct {
	CompanyResponse
	Role string `json:"role"` // 用户在该公司的角色
}

// ConvertToEntity 将创建请求转换为实体
func (req *CompanyCreateRequest) ConvertToEntity() (*entity.Company, error) {
	company := &entity.Company{
		Name:          req.Name,
		Code:          req.Code,
		Description:   req.Description,
		Logo:          req.Logo,
		Status:        entity.CompanyStatus(req.Status),
		ContactPerson: req.ContactPerson,
		ContactPhone:  req.ContactPhone,
		ContactEmail:  req.ContactEmail,
		Address:       req.Address,
		Properties:    req.Properties,
	}

	if req.ExpiredAt != nil {
		expiredAt, err := time.Parse(time.RFC3339, *req.ExpiredAt)
		if err != nil {
			return nil, err
		}
		company.ExpiredAt = &expiredAt
	}

	return company, nil
}

// ConvertToQuery 将过滤器转换为实体查询
func (f *CompanyFilter) ConvertToQuery() entity.CompanyQuery {
	query := entity.CompanyQuery{
		Q:    f.Q,
		Name: f.Name,
		Code: f.Code,
		Pagination: entity.Pagination{
			PageNumber: f.GetPageNumber(),
			PageSize:   f.GetPageSize(),
		},
	}

	if f.Status != nil {
		status := entity.CompanyStatus(*f.Status)
		query.Status = &status
	}

	return query
}

// ConvertToCompanyResponse 将实体转换为响应
func ConvertToCompanyResponse(company *entity.Company) (*CompanyResponse, error) {
	resp := &CompanyResponse{
		ID:            company.ID,
		Name:          company.Name,
		Code:          company.Code,
		Description:   company.Description,
		Logo:          company.Logo,
		Status:        CompanyStatus(company.Status),
		ContactPerson: company.ContactPerson,
		ContactPhone:  company.ContactPhone,
		ContactEmail:  company.ContactEmail,
		Address:       company.Address,
		Properties:    company.Properties,
		CreatedAt:     company.CreatedAt.Unix(),
		UpdatedAt:     company.UpdatedAt.Unix(),
	}

	if company.ExpiredAt != nil {
		expiredAt := company.ExpiredAt.Unix()
		resp.ExpiredAt = &expiredAt
	}

	return resp, nil
}

// GetPageNumber 安全获取页码
func (f *CompanyFilter) GetPageNumber() int {
	if f.PageNumber <= 0 {
		return 1
	}
	return f.PageNumber
}

// GetPageSize 安全获取每页大小
func (f *CompanyFilter) GetPageSize() int {
	if f.PageSize <= 0 {
		return 10
	}
	return f.PageSize
}

// ============================================================================
// 企业管理用户相关 DTO
// ============================================================================

// CompanyUserBriefResponse 企业用户简要信息响应
type CompanyUserBriefResponse struct {
	ID       uint    `json:"id"`
	Username string  `json:"username"`
	Nickname string  `json:"nickname"`
	Email    *string `json:"email,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Avatar   string  `json:"avatar"`
	Role     string  `json:"role"` // 用户在该企业的角色
}

// CompanyUserFilter 企业用户查询过滤器
type CompanyUserFilter struct {
	Q *string `form:"q,omitempty" binding:"omitempty,max=100"` // 综合查询条件，同时模糊匹配用户名、手机号和邮箱
	PaginationRequest
}

// CompanyUserNotInFilter 企业不存在用户查询过滤器
type CompanyUserNotInFilter struct {
	Q *string `form:"q,omitempty" binding:"omitempty,max=100"` // 综合查询条件，同时模糊匹配用户名、手机号和邮箱
	PaginationRequest
}

// CompanyUserAddRequest 为企业添加用户请求
type CompanyUserAddRequest struct {
	UserID uint   `json:"userId" binding:"required,min=1"`
	Role   string `json:"role" binding:"required,oneof=Root SuperAdmin Admin Manager User"`
}

// CompanyUserUpdateRoleRequest 更新用户在企业中的角色请求
type CompanyUserUpdateRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=Root SuperAdmin Admin Manager User"`
}

// CompanyUserBatchDeleteRequest 批量删除企业用户请求
type CompanyUserBatchDeleteRequest struct {
	UserIDs []uint `json:"userIds" binding:"required,min=1"`
}

// CompanyUserBatchAddRequest 批量为企业添加用户请求
type CompanyUserBatchAddRequest struct {
	Users []CompanyUserAddRequest `json:"users" binding:"required,min=1,dive"`
}

// GetPageNumber 安全获取页码
func (f *CompanyUserFilter) GetPageNumber() int {
	if f.PageNumber <= 0 {
		return 1
	}
	return f.PageNumber
}

// GetPageSize 安全获取每页大小
func (f *CompanyUserFilter) GetPageSize() int {
	if f.PageSize <= 0 {
		return 10
	}
	return f.PageSize
}

// GetPageNumber 安全获取页码
func (f *CompanyUserNotInFilter) GetPageNumber() int {
	if f.PageNumber <= 0 {
		return 1
	}
	return f.PageNumber
}

// GetPageSize 安全获取每页大小
func (f *CompanyUserNotInFilter) GetPageSize() int {
	if f.PageSize <= 0 {
		return 10
	}
	return f.PageSize
}