package service

import (
	"context"
	"fmt"

	"rbac-admin/internal/cache"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/model/entity"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
)

type DeviceTypeService struct {
	Repo        *repository.DeviceTypeRepository
	ConfigCache *cache.DeviceTypesCache
}

// CreateDeviceType 创建设备类型
func (s *DeviceTypeService) CreateDeviceType(ctx context.Context, req dto.DeviceTypeCreateRequest) (*dto.DeviceTypeResponse, error) {
	// 检查类型是否已存在
	exists, err := s.Repo.ExistsByType(ctx, req.Type)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errx.New(constant.Conflict, "类型已存在", nil)
	}

	// 转换并创建实体
	deviceType := req.ConvertToEntity()
	if err := s.Repo.Create(ctx, deviceType); err != nil {
		return nil, err
	}

	// 更新缓存
	if s.ConfigCache != nil {
		go s.ConfigCache.UpdateDeviceTypes(context.Background())
	}

	return dto.ConvertToDeviceTypeResponse(deviceType), nil
}

// GetDeviceType 根据ID获取设备类型
func (s *DeviceTypeService) GetDeviceType(ctx context.Context, id uint) (*dto.DeviceTypeResponse, error) {
	if id == 0 {
		return nil, errx.New(constant.BadRequest, "设备类型ID不能为空", nil)
	}

	deviceType, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.ConvertToDeviceTypeResponse(deviceType), nil
}

// GetDeviceTypeByType 根据类型标识获取设备类型
func (s *DeviceTypeService) GetDeviceTypeByType(ctx context.Context, typ string) (*dto.DeviceTypeResponse, error) {
	if typ == "" {
		return nil, errx.New(constant.BadRequest, "类型标识不能为空", nil)
	}

	deviceType, err := s.Repo.GetByType(ctx, typ)
	if err != nil {
		return nil, err
	}
	return dto.ConvertToDeviceTypeResponse(deviceType), nil
}

// UpdateDeviceType 更新设备类型
func (s *DeviceTypeService) UpdateDeviceType(ctx context.Context, id uint, req dto.DeviceTypeUpdateRequest) error {
	if id == 0 {
		return errx.New(constant.BadRequest, "设备类型ID不能为空", nil)
	}

	// 获取现有记录
	old, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 构建更新字段
	updates := s.buildUpdateFields(req, old)
	if len(updates) == 0 {
		logger := contextx.GetLoggerFromContext(ctx)
		logger.Info(fmt.Sprintf("设备类型更新请求无有效字段，设备类型ID: %d", id))
		return nil
	}

	// 检查类型冲突
	if req.Type != nil && *req.Type != old.Type {
		exists, err := s.Repo.ExistsByType(ctx, *req.Type)
		if err != nil {
			return err
		}
		if exists {
			return errx.New(constant.Conflict, "类型已存在", nil)
		}
	}

	if err := s.Repo.Update(ctx, id, updates); err != nil {
		return err
	}

	// 更新缓存
	if s.ConfigCache != nil {
		go s.ConfigCache.UpdateDeviceTypes(context.Background())
	}

	return nil
}

// buildUpdateFields 构建更新字段
func (s *DeviceTypeService) buildUpdateFields(req dto.DeviceTypeUpdateRequest, old *entity.DeviceType) map[string]interface{} {
	updates := make(map[string]interface{})

	if req.Type != nil && *req.Type != old.Type {
		updates["type"] = *req.Type
	}
	if req.Name != nil && *req.Name != old.Name {
		updates["name"] = *req.Name
	}
	if req.Description != nil && *req.Description != old.Description {
		updates["description"] = *req.Description
	}
	if req.Icon != nil && *req.Icon != old.Icon {
		updates["icon"] = *req.Icon
	}
	if req.Schema != nil {
		updates["schema"] = *req.Schema
	}

	return updates
}

// DeleteDeviceType 删除设备类型
func (s *DeviceTypeService) DeleteDeviceType(ctx context.Context, id uint) error {
	if id == 0 {
		return errx.New(constant.BadRequest, "设备类型ID不能为空", nil)
	}
	if err := s.Repo.Delete(ctx, id); err != nil {
		return err
	}

	// 更新缓存
	if s.ConfigCache != nil {
		go s.ConfigCache.UpdateDeviceTypes(context.Background())
	}

	return nil
}

// BatchDeleteDeviceTypes 批量删除设备类型
func (s *DeviceTypeService) BatchDeleteDeviceTypes(ctx context.Context, req dto.DeviceTypeDeleteRequest) error {
	if len(req.IDs) == 0 {
		return errx.New(constant.BadRequest, "无有效删除元素", nil)
	}

	// 验证ID列表
	for _, id := range req.IDs {
		if id == 0 {
			return errx.New(constant.BadRequest, "设备类型ID不能为空", nil)
		}
	}

	if err := s.Repo.BatchDelete(ctx, req.IDs); err != nil {
		return err
	}

	// 更新缓存
	if s.ConfigCache != nil {
		go s.ConfigCache.UpdateDeviceTypes(context.Background())
	}

	return nil
}

// ListDeviceTypes 分页查询设备类型列表
func (s *DeviceTypeService) ListDeviceTypes(ctx context.Context, filter dto.DeviceTypeFilter) ([]*dto.DeviceTypeResponse, int64, error) {
	// 验证分页参数
	if filter.PageNumber <= 0 {
		filter.PageNumber = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 10
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100 // 限制最大页面大小
	}

	query := filter.ConvertToQuery()
	list, total, err := s.Repo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	// 预分配切片容量以提高性能
	resp := make([]*dto.DeviceTypeResponse, 0, len(list))
	for _, item := range list {
		resp = append(resp, dto.ConvertToDeviceTypeResponse(item))
	}
	return resp, total, nil
}

// ListAllDeviceTypes 获取所有未删除的设备类型列表（不包含ID）
// 优先从缓存读取，如果缓存未命中，则从数据库查询并回写缓存。
func (s *DeviceTypeService) ListAllDeviceTypes(ctx context.Context) ([]*dto.DeviceTypeSimpleResponse, error) {
	if s.ConfigCache == nil {
		return nil, errx.New(constant.InternalServerError, "配置缓存服务未初始化", nil)
	}
	return s.ConfigCache.GetDeviceTypes(ctx)
}
