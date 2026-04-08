package service

import (
	"context"

	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/model/entity"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"fmt"
)

type WaterFertilizerTemplateService struct {
	TemplateRepo *repository.WaterFertilizerTemplateRepository
}

// Create 创建水肥模板
func (s *WaterFertilizerTemplateService) Create(ctx context.Context, req dto.WaterFertilizerTemplateCreateRequest) (*dto.WaterFertilizerTemplateResponse, error) {
	// 检查名称唯一性
	if exists, err := s.TemplateRepo.ExistsByField(ctx, "name", req.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errx.New(constant.Conflict, "水肥模板名称已存在", nil)
	}

	// 转换DTO到Entity
	entity := req.ConvertToEntity()

	// 创建记录
	if err := s.TemplateRepo.Create(ctx, entity); err != nil {
		return nil, err
	}

	// 转换为响应
	return dto.ConvertToWaterFertilizerTemplateResponse(entity), nil
}

// Get 获取水肥模板详情
func (s *WaterFertilizerTemplateService) Get(ctx context.Context, id uint) (*dto.WaterFertilizerTemplateResponse, error) {
	entity, err := s.TemplateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.ConvertToWaterFertilizerTemplateResponse(entity), nil
}

// Update 更新水肥模板
func (s *WaterFertilizerTemplateService) Update(ctx context.Context, id uint, req dto.WaterFertilizerTemplateUpdateRequest) error {
	// 先获取当前数据
	current, err := s.TemplateRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查名称唯一性（仅在名称变更时检查）
	if req.Name != nil && *req.Name != current.Name {
		if exists, err := s.TemplateRepo.ExistsByField(ctx, "name", *req.Name); err != nil {
			return err
		} else if exists {
			return errx.New(constant.Conflict, "水肥模板名称已存在", nil)
		}
	}

	// 构建更新字段
	updates := req.BuildUpdateFields()
	if len(updates) == 0 {
		logger := contextx.GetLoggerFromContext(ctx)
		logger.Info(fmt.Sprintf("水肥模板更新请求无有效字段，模板ID: %d", id))
		return nil
	}

	if err := s.TemplateRepo.Update(ctx, id, updates); err != nil {
		return err
	}

	return nil
}

// Delete 删除水肥模板
func (s *WaterFertilizerTemplateService) Delete(ctx context.Context, id uint) error {
	if err := s.TemplateRepo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

// List 带分页和查询条件的水肥模板列表
func (s *WaterFertilizerTemplateService) List(ctx context.Context, filter dto.WaterFertilizerTemplateFilter) ([]*dto.WaterFertilizerTemplateResponse, int64, error) {
	query := filter.ConvertToQuery()
	entities, total, err := s.TemplateRepo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	return s.convertEntitiesToResponses(entities), total, nil
}

// convertEntitiesToResponses 将实体列表转换为响应列表
func (s *WaterFertilizerTemplateService) convertEntitiesToResponses(entities []*entity.WaterFertilizerTemplate) []*dto.WaterFertilizerTemplateResponse {
	responses := make([]*dto.WaterFertilizerTemplateResponse, 0, len(entities))
	for _, e := range entities {
		responses = append(responses, dto.ConvertToWaterFertilizerTemplateResponse(e))
	}
	return responses
}

// BatchDelete 批量删除水肥模板
func (s *WaterFertilizerTemplateService) BatchDelete(ctx context.Context, req dto.WaterFertilizerTemplateDeleteRequest) error {
	if len(req.IDs) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}
	if err := s.TemplateRepo.BatchDelete(ctx, req.IDs); err != nil {
		return err
	}

	return nil
}
