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

type StandardSensorService struct {
	SensorRepo  *repository.StandardSensorRepository
	ConfigCache *cache.StandardSensorsCache
}

// Create 创建传感器
func (s *StandardSensorService) Create(ctx context.Context, req dto.StandardSensorCreateRequest) (*dto.StandardSensorResponse, error) {
	// 检查名称唯一性
	if exists, err := s.SensorRepo.ExistsByField(ctx, "name", req.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errx.New(constant.Conflict, "传感器名称已存在", nil)
	}

	// 转换DTO到Entity
	entity := req.ConvertToEntity()

	// 创建记录
	if err := s.SensorRepo.Create(ctx, entity); err != nil {
		return nil, err
	}

	// 更新缓存
	if s.ConfigCache != nil {
		go s.ConfigCache.UpdateStandardSensors(context.Background())
	}

	// 转换为响应
	return dto.ConvertToStandardSensorResponse(entity), nil
}

// Get 获取传感器详情
func (s *StandardSensorService) Get(ctx context.Context, id uint) (*dto.StandardSensorResponse, error) {
	entity, err := s.SensorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.ConvertToStandardSensorResponse(entity), nil
}

// Update 更新传感器
func (s *StandardSensorService) Update(ctx context.Context, id uint, req dto.StandardSensorUpdateRequest) error {
	// 先获取当前数据
	current, err := s.SensorRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查名称唯一性（仅在名称变更时检查）
	if req.Name != nil && *req.Name != current.Name {
		if exists, err := s.SensorRepo.ExistsByField(ctx, "name", *req.Name); err != nil {
			return err
		} else if exists {
			return errx.New(constant.Conflict, "传感器名称已存在", nil)
		}
	}

	// 构建更新字段
	updates := req.BuildUpdateFields()
	if len(updates) == 0 {
		logger := contextx.GetLoggerFromContext(ctx)
		logger.Info(fmt.Sprintf("标准传感器更新请求无有效字段，传感器ID: %d", id))
		return nil
	}

	if err := s.SensorRepo.Update(ctx, id, updates); err != nil {
		return err
	}

	// 更新缓存
	if s.ConfigCache != nil {
		go s.ConfigCache.UpdateStandardSensors(context.Background())
	}

	return nil
}

// Delete 删除传感器
func (s *StandardSensorService) Delete(ctx context.Context, id uint) error {
	if err := s.SensorRepo.Delete(ctx, id); err != nil {
		return err
	}

	// 更新缓存
	if s.ConfigCache != nil {
		go s.ConfigCache.UpdateStandardSensors(context.Background())
	}

	return nil
}

// List 带分页和查询条件的列表（走数据库）
func (s *StandardSensorService) List(ctx context.Context, filter dto.StandardSensorFilter) ([]*dto.StandardSensorResponse, int64, error) {
	query := filter.ConvertToQuery()
	entities, total, err := s.SensorRepo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	return s.convertEntitiesToResponses(entities), total, nil
}

// ListAll 获取所有数据（走缓存）
func (s *StandardSensorService) ListAll(ctx context.Context) ([]*dto.StandardSensorResponse, error) {
	if s.ConfigCache == nil {
		return nil, errx.New(constant.InternalServerError, "配置缓存服务未初始化", nil)
	}
	return s.ConfigCache.GetStandardSensors(ctx)
}

// convertEntitiesToResponses 将实体列表转换为响应列表
func (s *StandardSensorService) convertEntitiesToResponses(entities []*entity.StandardSensor) []*dto.StandardSensorResponse {
	responses := make([]*dto.StandardSensorResponse, 0, len(entities))
	for _, e := range entities {
		responses = append(responses, dto.ConvertToStandardSensorResponse(e))
	}
	return responses
}

// BatchDelete 批量删除传感器
func (s *StandardSensorService) BatchDelete(ctx context.Context, req dto.StandardSensorDeleteRequest) error {
	if len(req.IDs) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}
	if err := s.SensorRepo.BatchDelete(ctx, req.IDs); err != nil {
		return err
	}

	// 更新缓存
	if s.ConfigCache != nil {
		go s.ConfigCache.UpdateStandardSensors(context.Background())
	}

	return nil
}
