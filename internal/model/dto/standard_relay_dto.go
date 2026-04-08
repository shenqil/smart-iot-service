package dto

import (
	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/errx"
	"strings"
)

// StandardRelayCreateRequest 创建请求
type StandardRelayCreateRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100" example:"继电器名称"`
	URL  string `json:"url" binding:"omitempty,max=255" example:"http://example.com"`
}

// Validate 验证创建请求
func (req *StandardRelayCreateRequest) Validate() error {
	if strings.TrimSpace(req.Name) == "" {
		return errx.New(constant.BadRequest, "名称不能为空", nil)
	}
	if len(req.Name) > 100 {
		return errx.New(constant.BadRequest, "名称长度不能超过100个字符", nil)
	}
	if len(req.URL) > 255 {
		return errx.New(constant.BadRequest, "URL长度不能超过255个字符", nil)
	}
	return nil
}

// ConvertToEntity 将创建请求转换为实体
func (req *StandardRelayCreateRequest) ConvertToEntity() (*entity.StandardRelay, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	return &entity.StandardRelay{
		Name: strings.TrimSpace(req.Name),
		URL:  strings.TrimSpace(req.URL),
	}, nil
}

// StandardRelayUpdateRequest 更新请求
type StandardRelayUpdateRequest struct {
	Name *string `json:"name,omitempty" binding:"omitempty,min=1,max=100" example:"继电器名称"`
	URL  *string `json:"url,omitempty" binding:"omitempty,max=255" example:"http://example.com"`
}

// Validate 验证更新请求
func (req *StandardRelayUpdateRequest) Validate() error {
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return errx.New(constant.BadRequest, "名称不能为空", nil)
		}
		if len(name) > 100 {
			return errx.New(constant.BadRequest, "名称长度不能超过100个字符", nil)
		}
	}

	if req.URL != nil && len(*req.URL) > 255 {
		return errx.New(constant.BadRequest, "URL长度不能超过255个字符", nil)
	}

	return nil
}

// ConvertToUpdateMap 转换为更新映射
func (req *StandardRelayUpdateRequest) ConvertToUpdateMap() (map[string]interface{}, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.URL != nil {
		updates["url"] = strings.TrimSpace(*req.URL)
	}

	return updates, nil
}

// StandardRelayDeleteRequest 批量删除请求
type StandardRelayDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1" example:"[1,2,3]"`
}

// Validate 验证删除请求
func (req *StandardRelayDeleteRequest) Validate() error {
	if len(req.IDs) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}
	if len(req.IDs) > 100 {
		return errx.New(constant.BadRequest, "批量删除数量不能超过100个", nil)
	}
	return nil
}

// StandardRelayFilter 查询过滤器
type StandardRelayFilter struct {
	Name              string           `form:"name,omitempty" example:"名称搜索参数"`
	Q                 string           `form:"q,omitempty" example:"全部搜索参数"`
	PaginationRequest `form:",inline"` // 分页参数
}

// ConvertToQuery 将过滤器转换为实体查询
func (f *StandardRelayFilter) ConvertToQuery() entity.StandardRelayQuery {
	return entity.StandardRelayQuery{
		Name: strings.TrimSpace(f.Name),
		Q:    strings.TrimSpace(f.Q),
		Pagination: entity.Pagination{
			PageNumber: f.GetPageNumber(),
			PageSize:   f.GetPageSize(),
		},
	}
}

// StandardRelayResponse 响应结构
type StandardRelayResponse struct {
	ID        uint   `json:"id" example:"1"`
	Name      string `json:"name" example:"继电器名称"`
	URL       string `json:"url" example:"http://example.com"`
	CreatedAt int64  `json:"createdAt" example:"1640995200"` // 秒级时间戳
	UpdatedAt int64  `json:"updatedAt" example:"1640995200"` // 秒级时间戳
}

// ConvertToStandardRelayResponse 将实体转换为响应
func ConvertToStandardRelayResponse(relay *entity.StandardRelay) (*StandardRelayResponse, error) {
	if relay == nil {
		return nil, errx.New(constant.InternalServerError, "实体数据为空", nil)
	}

	return &StandardRelayResponse{
		ID:        relay.ID,
		Name:      relay.Name,
		URL:       relay.URL,
		CreatedAt: relay.CreatedAt.Unix(),
		UpdatedAt: relay.UpdatedAt.Unix(),
	}, nil
}

// ConvertToStandardRelayResponseList 批量转换实体为响应
func ConvertToStandardRelayResponseList(relays []*entity.StandardRelay) ([]*StandardRelayResponse, error) {
	if len(relays) == 0 {
		return []*StandardRelayResponse{}, nil
	}

	responses := make([]*StandardRelayResponse, 0, len(relays))
	for _, relay := range relays {
		resp, err := ConvertToStandardRelayResponse(relay)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// GetPageNumber 安全获取页码
func (f *StandardRelayFilter) GetPageNumber() int {
	if f.PageNumber <= 0 {
		return 1
	}
	return f.PageNumber
}

// GetPageSize 安全获取每页大小
func (f *StandardRelayFilter) GetPageSize() int {
	if f.PageSize <= 0 {
		return 10 // 默认每页10条
	}
	if f.PageSize > 100 {
		return 100 // 最大每页100条
	}
	return f.PageSize
}
