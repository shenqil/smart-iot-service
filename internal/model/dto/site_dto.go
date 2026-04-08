package dto

import (
	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/errx"

	"gorm.io/datatypes"
)

// SiteDeviceInfo 关联设备信息
type SiteDeviceInfo struct {
	ID        uint    `json:"id"`
	DeviceNum uint    `json:"deviceNum"`
	Name      *string `json:"name,omitempty"`
	Type      string  `json:"type"`
	IsActive  bool    `json:"isActive"`
	Status    string  `json:"status"`
}

// SiteCreateRequest 创建请求
type SiteCreateRequest struct {
	Name        string         `json:"name" binding:"required,max=100"`
	Title       string         `json:"title" binding:"max=100"`
	Code        string         `json:"code" binding:"required,max=50"`
	Description string         `json:"description" binding:"max=500"`
	Address     string         `json:"address" binding:"max=200"`
	Province    string         `json:"province" binding:"max=50"`
	City        string         `json:"city" binding:"max=50"`
	District    string         `json:"district" binding:"max=50"`
	PostalCode  string         `json:"postalCode" binding:"max=20"`
	Longitude   float64        `json:"longitude"`
	Latitude    float64        `json:"latitude"`
	Type        string         `json:"type" binding:"required,max=50"`
	Status      string         `json:"status" binding:"omitempty,oneof=active inactive under_construction"`
	CoverImage  string         `json:"coverImage" binding:"omitempty,max=255"`
	Images      []string       `json:"images" binding:"dive"`
	Properties  datatypes.JSON `json:"properties" binding:"required"`
	CompanyID   uint           `json:"-" form:"-"` // 公司ID（从上下文获取，不暴露给用户）
}

// SiteUpdateRequest 更新请求
type SiteUpdateRequest struct {
	Name        *string         `json:"name,omitempty" binding:"omitempty,max=100"`
	Title       *string         `json:"title,omitempty" binding:"omitempty,max=100"`
	Description *string         `json:"description,omitempty" binding:"omitempty,max=500"`
	Address     *string         `json:"address,omitempty" binding:"omitempty,max=200"`
	Province    *string         `json:"province,omitempty" binding:"omitempty,max=50"`
	City        *string         `json:"city,omitempty" binding:"omitempty,max=50"`
	District    *string         `json:"district,omitempty" binding:"omitempty,max=50"`
	PostalCode  *string         `json:"postalCode,omitempty" binding:"omitempty,max=20"`
	Longitude   *float64        `json:"longitude,omitempty"`
	Latitude    *float64        `json:"latitude,omitempty"`
	Type        *string         `json:"type,omitempty" binding:"omitempty,max=50"`
	Status      *string         `json:"status,omitempty" binding:"omitempty,oneof=active inactive under_construction"`
	CoverImage  *string         `json:"coverImage,omitempty" binding:"omitempty,max=255"`
	Images      *[]string       `json:"images,omitempty" binding:"omitempty,dive"`
	Properties  *datatypes.JSON `json:"properties,omitempty" binding:"omitempty,dive"`
	CompanyID   uint            `json:"-" form:"-"` // 公司ID（从上下文获取，不暴露给用户）
}

// SiteDeleteRequest 批量删除请求
type SiteDeleteRequest struct {
	IDs       []uint `json:"ids" binding:"required,min=1"`
	CompanyID uint   `json:"-"` // 公司ID（从上下文获取，不暴露给用户）
}

// SiteDeviceAssignRequest 园区分配设备请求
type SiteDeviceAssignRequest struct {
	SiteID    uint   `json:"siteId" binding:"required"`
	DeviceIDs []uint `json:"deviceIds" binding:"required,min=1"`
}

// SiteDeviceRemoveRequest 园区删除设备请求
type SiteDeviceRemoveRequest struct {
	SiteID    uint   `json:"siteId" binding:"required"`
	DeviceIDs []uint `json:"deviceIds" binding:"required,min=1"`
}

// SiteUserAddRequest adds users to a site.
type SiteUserAddRequest struct {
	UserIDs []uint `json:"userIds" binding:"required,min=1"`
}

// SiteUserRemoveRequest removes users from a site.
type SiteUserRemoveRequest struct {
	UserIDs []uint `json:"userIds" binding:"required,min=1"`
}

// SiteUserResponse represents a user assigned to a site.
type SiteUserResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

// SiteUserFilter is used to filter users for a site.
type SiteUserFilter struct {
	PaginationRequest
}


// SiteFilter 查询过滤器
type SiteFilter struct {
	Q                 *string `form:"q,omitempty"`
	Name              *string `form:"name,omitempty"`
	Code              *string `form:"code,omitempty"`
	Type              *string `form:"type,omitempty"`
	Status            *string `form:"status,omitempty"`
	Province          *string `form:"province,omitempty"`
	City              *string `form:"city,omitempty"`
	District          *string `form:"district,omitempty"`
	CompanyID         uint    `form:"-" json:"-"` // 公司ID（从上下文获取，不暴露给用户）
	PaginationRequest `form:",inline"`
}

// SiteCompany 园区公司信息
type SiteCompany struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// SiteResponse 响应结构
type SiteResponse struct {
	ID          uint           `json:"id"`
	Name        string         `json:"name"`
	Title       string         `json:"title"`
	Code        string         `json:"code"`
	Description string         `json:"description"`
	Address     string         `json:"address"`
	Province    string         `json:"province"`
	City        string         `json:"city"`
	District    string         `json:"district"`
	PostalCode  string         `json:"postalCode"`
	Longitude   float64        `json:"longitude"`
	Latitude    float64        `json:"latitude"`
	Type        string         `json:"type"`
	Status      string         `json:"status"`
	CoverImage  string         `json:"coverImage"`
	Images      []string       `json:"images"`
	Properties  datatypes.JSON `json:"properties"`
	Company     *SiteCompany   `json:"company,omitempty"`
	CreatedAt   int64          `json:"createdAt"`
	UpdatedAt   int64          `json:"updatedAt"`
}

// ConvertToEntity 将创建请求转换为实体
func (req *SiteCreateRequest) ConvertToEntity() (*entity.Site, error) {
	if req.Longitude < -180 || req.Longitude > 180 {
		return nil, errx.New(constant.BadRequest, "经度范围无效", nil)
	}
	if req.Latitude < -90 || req.Latitude > 90 {
		return nil, errx.New(constant.BadRequest, "纬度范围无效", nil)
	}

	site := &entity.Site{
		Name:        req.Name,
		Title:       req.Title,
		Code:        req.Code,
		Description: req.Description,
		Address:     req.Address,
		Province:    req.Province,
		City:        req.City,
		District:    req.District,
		PostalCode:  req.PostalCode,
		Longitude:   req.Longitude,
		Latitude:    req.Latitude,
		Type:        req.Type,
		Status:      entity.SiteStatus(req.Status),
		CoverImage:  req.CoverImage,
		Properties:  req.Properties,
		CompanyID:   &req.CompanyID,
	}

	// 处理图片
	if len(req.Images) > 0 {
		imagesJSON, err := ToJSON(req.Images)
		if err != nil {
			return nil, errx.New(constant.BadRequest, "图片格式错误", err)
		}
		site.Images = imagesJSON
	}

	return site, nil
}

// ConvertToQuery 将过滤器转换为实体查询
func (f *SiteFilter) ConvertToQuery() entity.SiteQuery {
	query := entity.SiteQuery{
		Q:         f.Q,
		Name:      f.Name,
		Code:      f.Code,
		Type:      f.Type,
		Province:  f.Province,
		City:      f.City,
		District:  f.District,
		CompanyID: &f.CompanyID,
		Pagination: entity.Pagination{
			PageNumber: f.GetPageNumber(),
			PageSize:   f.GetPageSize(),
		},
	}

	if f.Status != nil {
		status := entity.SiteStatus(*f.Status)
		query.Status = &status
	}

	return query
}

// ConvertToSiteResponse 将实体转换为响应
func ConvertToSiteResponse(site *entity.Site) (*SiteResponse, error) {
	resp := &SiteResponse{
		ID:          site.ID,
		Name:        site.Name,
		Title:       site.Title,
		Code:        site.Code,
		Description: site.Description,
		Address:     site.Address,
		Province:    site.Province,
		City:        site.City,
		District:    site.District,
		PostalCode:  site.PostalCode,
		Longitude:   site.Longitude,
		Latitude:    site.Latitude,
		Type:        site.Type,
		Status:      string(site.Status),
		CoverImage:  site.CoverImage,
		CreatedAt:   site.CreatedAt.Unix(),
		UpdatedAt:   site.UpdatedAt.Unix(),
		Properties:  site.Properties,
	}

	// 处理图片
	var images []string
	if err := FromJSON(site.Images, &images); err == nil {
		resp.Images = images
	}

	// 处理公司信息
	if site.Company != nil {
		resp.Company = &SiteCompany{
			ID:   site.Company.ID,
			Name: site.Company.Name,
			Code: site.Company.Code,
		}
	}

	return resp, nil
}


