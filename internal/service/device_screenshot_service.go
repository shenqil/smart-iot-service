package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
)

type DeviceScreenshotService struct {
	Repo *repository.DeviceScreenshotRepository
	OSS  *repository.OSSClient
}

// Create
func (s *DeviceScreenshotService) Create(ctx context.Context, req dto.DeviceScreenshotCreateRequest) (*dto.DeviceScreenshotResponse, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	entity, err := req.ConvertToEntity()
	if err != nil {
		return nil, err
	}

	// 处理图片上传
	if req.ImageURL != "" && isValidURL(req.ImageURL) {
		uploadedURL, err := s.uploadImageToOSS(req.ImageURL, req.DeviceSN)
		if err != nil {
			logger.Error(fmt.Sprintf("%v", err))
			return nil, errx.New(constant.InternalServerError, "图片上传失败", err)
		}
		entity.ImageURL = uploadedURL
	}

	if err := s.Repo.Create(ctx, entity); err != nil {
		return nil, err
	}

	return dto.ConvertToDeviceScreenshotResponse(entity), nil
}

// Get
func (s *DeviceScreenshotService) Get(ctx context.Context, id uint) (*dto.DeviceScreenshotResponse, error) {
	entity, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.ConvertToDeviceScreenshotResponse(entity), nil
}

// Update
func (s *DeviceScreenshotService) Update(ctx context.Context, id uint, req dto.DeviceScreenshotUpdateRequest) error {
	logger := contextx.GetLoggerFromContext(ctx)
	updates := make(map[string]interface{})
	if req.CaptureTime != nil {
		if *req.CaptureTime <= 0 {
			return errx.New(constant.BadRequest, "截图时间无效", nil)
		}
		updates["capture_time"] = *req.CaptureTime
	}

	// 处理图片上传
	if req.ImageURL != nil && *req.ImageURL != "" && isValidURL(*req.ImageURL) {
		// 如果提供了新的图片URL，上传到OSS
		uploadedURL, err := s.uploadImageToOSS(*req.ImageURL, uint32(id))
		if err != nil {
			logger.Error(fmt.Sprintf("%v", err))
			return errx.New(constant.InternalServerError, "图片上传失败", err)
		}
		updates["image_url"] = uploadedURL
	}
	if req.Extra != nil {
		updates["extra"] = *req.Extra
	}
	if len(updates) == 0 {
		return errx.New(constant.BadRequest, "未提供有效更新字段", nil)
	}
	return s.Repo.Update(ctx, id, updates)
}

// Delete
func (s *DeviceScreenshotService) Delete(ctx context.Context, id uint) error {
	return s.Repo.Delete(ctx, id)
}

// BatchDelete
func (s *DeviceScreenshotService) BatchDelete(ctx context.Context, req dto.DeviceScreenshotDeleteRequest) error {
	if len(req.IDs) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}
	return s.Repo.BatchDelete(ctx, req.IDs)
}

// List
func (s *DeviceScreenshotService) List(ctx context.Context, filter dto.DeviceScreenshotFilter) ([]*dto.DeviceScreenshotResponse, int64, error) {
	query := filter.ConvertToQuery()
	entities, total, err := s.Repo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	responses := make([]*dto.DeviceScreenshotResponse, 0, len(entities))
	for _, e := range entities {
		responses = append(responses, dto.ConvertToDeviceScreenshotResponse(e))
	}
	return responses, total, nil
}

// uploadImageToOSS 上传图片到OSS
func (s *DeviceScreenshotService) uploadImageToOSS(imageURL string, id uint32) (string, error) {
	// 生成OSS路径
	objectPath := s.generateOSSPath(imageURL, id)

	// 上传文件
	return s.OSS.UploadFromURL(imageURL, objectPath)
}

// generateOSSPath 生成OSS存储路径
func (s *DeviceScreenshotService) generateOSSPath(imageURL string, id uint32) string {
	now := time.Now()
	yearMonth := now.Format("200601")
	millis := now.UnixMilli()

	// 常见图片扩展名
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"}

	// 简单匹配图片扩展名
	for _, ext := range imageExts {
		if strings.Contains(strings.ToLower(imageURL), ext) {
			return fmt.Sprintf("device_screenshot/%s/%d/%d%s", yearMonth, id, millis, ext)
		}
	}

	// 如果没有匹配到图片扩展名，使用默认的.jpg
	return fmt.Sprintf("device_screenshot/%s/%d/%d.jpg", yearMonth, id, millis)
}

// isValidURL 检查是否为有效URL
func isValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
