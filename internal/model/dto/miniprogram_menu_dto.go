package dto

import (
	"strings"

	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/errx"

	"gorm.io/datatypes"
)

// MiniProgramMenuStatus 小程序功能栏状态枚举
type MiniProgramMenuStatus int8

const (
	MiniProgramMenuStatusDisabled MiniProgramMenuStatus = 0 // 禁用
	MiniProgramMenuStatusEnabled  MiniProgramMenuStatus = 1 // 启用
)

// MiniProgramMenuCreateRequest 创建请求
type MiniProgramMenuCreateRequest struct {
	Title      string                `json:"title" binding:"required,min=1,max=100" example:"功能标题"`
	Image      string                `json:"image" binding:"omitempty,max=255" example:"https://example.com/image.png"`
	Code       string                `json:"code" binding:"required,min=1,max=50" example:"menu_code"`
	Sort       int                   `json:"sort" binding:"omitempty" example:"0"`
	SubTitle   string                `json:"subTitle" binding:"omitempty,max=200" example:"副标题"`
	Status     MiniProgramMenuStatus `json:"status" binding:"omitempty,oneof=0 1" example:"1"`
	Properties datatypes.JSON        `json:"properties" binding:"omitempty"`
}

// Validate 验证创建请求
func (req *MiniProgramMenuCreateRequest) Validate() error {
	if strings.TrimSpace(req.Title) == "" {
		return errx.New(constant.BadRequest, "标题不能为空", nil)
	}
	if len(req.Title) > 100 {
		return errx.New(constant.BadRequest, "标题长度不能超过100个字符", nil)
	}
	if strings.TrimSpace(req.Code) == "" {
		return errx.New(constant.BadRequest, "功能代码不能为空", nil)
	}
	if len(req.Code) > 50 {
		return errx.New(constant.BadRequest, "功能代码长度不能超过50个字符", nil)
	}
	if len(req.Image) > 255 {
		return errx.New(constant.BadRequest, "图片URL长度不能超过255个字符", nil)
	}
	if len(req.SubTitle) > 200 {
		return errx.New(constant.BadRequest, "副标题长度不能超过200个字符", nil)
	}
	return nil
}

// ConvertToEntity 将创建请求转换为实体
func (req *MiniProgramMenuCreateRequest) ConvertToEntity() (*entity.MiniProgramMenu, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	menu := &entity.MiniProgramMenu{
		Title:    strings.TrimSpace(req.Title),
		Image:    strings.TrimSpace(req.Image),
		Code:     strings.TrimSpace(req.Code),
		Sort:     req.Sort,
		SubTitle: strings.TrimSpace(req.SubTitle),
		Status:   entity.MiniProgramMenuStatus(req.Status),
	}

	// 如果Status未设置，默认为启用
	if req.Status == 0 {
		menu.Status = entity.MiniProgramMenuStatusEnabled
	}

	// 处理Properties
	menu.Properties = req.Properties

	return menu, nil
}

// MiniProgramMenuUpdateRequest 更新请求
type MiniProgramMenuUpdateRequest struct {
	Title      *string                `json:"title,omitempty" binding:"omitempty,min=1,max=100" example:"功能标题"`
	Image      *string                `json:"image,omitempty" binding:"omitempty,max=255" example:"https://example.com/image.png"`
	Code       *string                `json:"code,omitempty" binding:"omitempty,min=1,max=50" example:"menu_code"`
	Sort       *int                   `json:"sort,omitempty" example:"0"`
	SubTitle   *string                `json:"subTitle,omitempty" binding:"omitempty,max=200" example:"副标题"`
	Status     *MiniProgramMenuStatus `json:"status,omitempty" binding:"omitempty,oneof=0 1" example:"1"`
	Properties *datatypes.JSON        `json:"properties,omitempty" binding:"omitempty,dive"`
}

// Validate 验证更新请求
func (req *MiniProgramMenuUpdateRequest) Validate() error {
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return errx.New(constant.BadRequest, "标题不能为空", nil)
		}
		if len(title) > 100 {
			return errx.New(constant.BadRequest, "标题长度不能超过100个字符", nil)
		}
	}

	if req.Code != nil {
		code := strings.TrimSpace(*req.Code)
		if code == "" {
			return errx.New(constant.BadRequest, "功能代码不能为空", nil)
		}
		if len(code) > 50 {
			return errx.New(constant.BadRequest, "功能代码长度不能超过50个字符", nil)
		}
	}

	if req.Image != nil && len(*req.Image) > 255 {
		return errx.New(constant.BadRequest, "图片URL长度不能超过255个字符", nil)
	}

	if req.SubTitle != nil && len(*req.SubTitle) > 200 {
		return errx.New(constant.BadRequest, "副标题长度不能超过200个字符", nil)
	}

	return nil
}

// ConvertToUpdateMap 转换为更新映射
func (req *MiniProgramMenuUpdateRequest) ConvertToUpdateMap() (map[string]interface{}, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Title != nil {
		updates["title"] = strings.TrimSpace(*req.Title)
	}
	if req.Image != nil {
		updates["image"] = strings.TrimSpace(*req.Image)
	}
	if req.Code != nil {
		updates["code"] = strings.TrimSpace(*req.Code)
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	if req.SubTitle != nil {
		updates["sub_title"] = strings.TrimSpace(*req.SubTitle)
	}
	if req.Status != nil {
		updates["status"] = entity.MiniProgramMenuStatus(*req.Status)
	}
	if req.Properties != nil {
		updates["properties"] = *req.Properties
	}

	return updates, nil
}

// MiniProgramMenuDeleteRequest 批量删除请求
type MiniProgramMenuDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1" example:"[1,2,3]"`
}

// Validate 验证删除请求
func (req *MiniProgramMenuDeleteRequest) Validate() error {
	if len(req.IDs) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}
	if len(req.IDs) > 100 {
		return errx.New(constant.BadRequest, "批量删除数量不能超过100个", nil)
	}
	return nil
}

// MiniProgramMenuFilter 查询过滤器
type MiniProgramMenuFilter struct {
	Q                 *string                `form:"q,omitempty" binding:"omitempty,max=100" example:"搜索关键词"`
	Title             *string                `form:"title,omitempty" binding:"omitempty,max=100" example:"标题"`
	Code              *string                `form:"code,omitempty" binding:"omitempty,max=50" example:"功能代码"`
	Status            *MiniProgramMenuStatus `form:"status,omitempty" binding:"omitempty,oneof=0 1" example:"1"`
	PaginationRequest `form:",inline"`       // 分页参数
}

// ConvertToQuery 将过滤器转换为实体查询
func (f *MiniProgramMenuFilter) ConvertToQuery() entity.MiniProgramMenuQuery {
	query := entity.MiniProgramMenuQuery{
		Q:     f.Q,
		Title: f.Title,
		Code:  f.Code,
		Pagination: entity.Pagination{
			PageNumber: f.GetPageNumber(),
			PageSize:   f.GetPageSize(),
		},
	}

	if f.Status != nil {
		status := entity.MiniProgramMenuStatus(*f.Status)
		query.Status = &status
	}

	return query
}

// MiniProgramMenuResponse 响应结构
type MiniProgramMenuResponse struct {
	ID         uint                  `json:"id" example:"1"`
	Title      string                `json:"title" example:"功能标题"`
	Image      string                `json:"image" example:"https://example.com/image.png"`
	Code       string                `json:"code" example:"menu_code"`
	Sort       int                   `json:"sort" example:"0"`
	SubTitle   string                `json:"subTitle" example:"副标题"`
	Status     MiniProgramMenuStatus `json:"status" example:"1"`
	Properties datatypes.JSON        `json:"properties"`
	CreatedAt  int64                 `json:"createdAt" example:"1640995200"` // 秒级时间戳
	UpdatedAt  int64                 `json:"updatedAt" example:"1640995200"` // 秒级时间戳
}

// ConvertToMiniProgramMenuResponse 将实体转换为响应
func ConvertToMiniProgramMenuResponse(menu *entity.MiniProgramMenu) (*MiniProgramMenuResponse, error) {
	if menu == nil {
		return nil, errx.New(constant.InternalServerError, "实体数据为空", nil)
	}

	resp := &MiniProgramMenuResponse{
		ID:         menu.ID,
		Title:      menu.Title,
		Image:      menu.Image,
		Code:       menu.Code,
		Sort:       menu.Sort,
		SubTitle:   menu.SubTitle,
		Status:     MiniProgramMenuStatus(menu.Status),
		Properties: menu.Properties,
		CreatedAt:  menu.CreatedAt.Unix(),
		UpdatedAt:  menu.UpdatedAt.Unix(),
	}

	return resp, nil
}

// ConvertToMiniProgramMenuResponseList 批量转换实体为响应
func ConvertToMiniProgramMenuResponseList(menus []*entity.MiniProgramMenu) ([]*MiniProgramMenuResponse, error) {
	if len(menus) == 0 {
		return []*MiniProgramMenuResponse{}, nil
	}

	responses := make([]*MiniProgramMenuResponse, 0, len(menus))
	for _, menu := range menus {
		resp, err := ConvertToMiniProgramMenuResponse(menu)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// GetPageNumber 安全获取页码
func (f *MiniProgramMenuFilter) GetPageNumber() int {
	if f.PageNumber <= 0 {
		return 1
	}
	return f.PageNumber
}

// GetPageSize 安全获取每页大小
func (f *MiniProgramMenuFilter) GetPageSize() int {
	if f.PageSize <= 0 {
		return 10 // 默认每页10条
	}
	if f.PageSize > 100 {
		return 100 // 最大每页100条
	}
	return f.PageSize
}
