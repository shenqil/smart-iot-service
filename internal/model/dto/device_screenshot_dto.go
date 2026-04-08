package dto

import (
	"time"

	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/errx"

	"gorm.io/datatypes"
)

// DeviceScreenshotCreateRequest 创建请求
type DeviceScreenshotCreateRequest struct {
	DeviceSN    uint32         `json:"deviceSn" binding:"required"`
	CaptureTime int64          `json:"captureTime" binding:"required"`
	ImageURL    string         `json:"imageUrl" binding:"required,url"`
	Extra       datatypes.JSON `json:"extra,omitempty"`
}

// DeviceScreenshotUpdateRequest 更新请求
type DeviceScreenshotUpdateRequest struct {
	CaptureTime *int64          `json:"captureTime,omitempty"`
	ImageURL    *string         `json:"imageUrl,omitempty" binding:"omitempty,url"`
	Extra       *datatypes.JSON `json:"extra,omitempty"`
}

// DeviceScreenshotDeleteRequest 批量删除请求
type DeviceScreenshotDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// DeviceScreenshotFilter 查询过滤器
type DeviceScreenshotFilter struct {
	DeviceSN          *uint32 `form:"deviceSn,omitempty"`
	StartTime         *int64  `form:"startTime,omitempty"`
	EndTime           *int64  `form:"endTime,omitempty"`
	PaginationRequest `form:",inline"`
}

// DeviceScreenshotResponse 响应结构
type DeviceScreenshotResponse struct {
	ID          uint           `json:"id"`
	DeviceSN    uint32         `json:"deviceSn"`
	CaptureTime int64          `json:"captureTime"`
	ImageURL    string         `json:"imageUrl"`
	Extra       datatypes.JSON `json:"extra"`
	CreatedAt   int64          `json:"createdAt"`
	UpdatedAt   int64          `json:"updatedAt"`
}

// ConvertToEntity 将创建请求转换为实体
func (req *DeviceScreenshotCreateRequest) ConvertToEntity() (*entity.DeviceScreenshot, error) {
	if req.CaptureTime <= 0 {
		return nil, errx.New(constant.BadRequest, "截图时间无效", nil)
	}

	if len(req.Extra) == 0 {
		req.Extra = datatypes.JSON([]byte("{}"))
	}

	return &entity.DeviceScreenshot{
		DeviceSN:    req.DeviceSN,
		CaptureTime: time.Unix(req.CaptureTime, 0),
		ImageURL:    req.ImageURL,
		Extra:       req.Extra,
	}, nil
}

// ConvertToQuery 将过滤器转换为实体查询
func (f *DeviceScreenshotFilter) ConvertToQuery() entity.DeviceScreenshotQuery {
	query := entity.DeviceScreenshotQuery{
		DeviceSN: f.DeviceSN,
		Pagination: entity.Pagination{
			PageNumber: f.GetPageNumber(),
			PageSize:   f.GetPageSize(),
		},
	}
	if f.StartTime != nil && *f.StartTime > 0 {
		t := time.Unix(*f.StartTime, 0)
		query.StartTime = &t
	}
	if f.EndTime != nil && *f.EndTime > 0 {
		t := time.Unix(*f.EndTime, 0)
		query.EndTime = &t
	}
	return query
}

// ConvertToResponse 将实体转换为响应
func ConvertToDeviceScreenshotResponse(s *entity.DeviceScreenshot) *DeviceScreenshotResponse {
	return &DeviceScreenshotResponse{
		ID:          s.ID,
		DeviceSN:    s.DeviceSN,
		CaptureTime: s.CaptureTime.Unix(),
		ImageURL:    s.ImageURL,
		Extra:       s.Extra,
		CreatedAt:   s.CreatedAt.Unix(),
		UpdatedAt:   s.UpdatedAt.Unix(),
	}
}

// GetPageNumber 安全获取页码
func (f *DeviceScreenshotFilter) GetPageNumber() int {
	if f.PageNumber <= 0 {
		return 1
	}
	return f.PageNumber
}

// GetPageSize 安全获取每页大小
func (f *DeviceScreenshotFilter) GetPageSize() int {
	if f.PageSize <= 0 {
		return 10
	}
	return f.PageSize
}
