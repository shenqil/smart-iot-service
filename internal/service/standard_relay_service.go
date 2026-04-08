package service

import (
	"context"
	"fmt"
	"log/slog"

	"rbac-admin/internal/cache"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
)

type StandardRelayService struct {
	StandardRelayRepo *repository.StandardRelayRepository
	ConfigCache       *cache.StandardRelaysCache
}

// Create 创建记录
func (s *StandardRelayService) Create(ctx context.Context, req dto.StandardRelayCreateRequest) (*dto.StandardRelayResponse, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 验证请求数据
	if err := req.Validate(); err != nil {
		logger.Error("创建请求验证失败", slog.Any("error", err))
		return nil, err
	}

	// 检查名称唯一性
	if exists, err := s.StandardRelayRepo.ExistsByField(ctx, "name", req.Name); err != nil {
		logger.Error("检查名称唯一性失败", slog.Any("error", err))
		return nil, err
	} else if exists {
		return nil, errx.New(constant.Conflict, "名称已存在", nil)
	}

	// DTO转换为实体
	entity, err := req.ConvertToEntity()
	if err != nil {
		logger.Error("DTO转换失败", slog.Any("error", err))
		return nil, err
	}

	// 创建记录
	if err := s.StandardRelayRepo.Create(ctx, entity); err != nil {
		logger.Error("创建记录失败", slog.Any("error", err))
		return nil, err
	}

	// 更新缓存
	if s.ConfigCache != nil {
		go s.ConfigCache.UpdateStandardRelays(context.Background())
	}

	// 转换为响应
	resp, err := dto.ConvertToStandardRelayResponse(entity)
	if err != nil {
		logger.Error("转换响应失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功创建标准继电器", slog.Uint64("id", uint64(entity.ID)), slog.String("name", entity.Name))
	return resp, nil
}

// Get 获取单条记录
func (s *StandardRelayService) Get(ctx context.Context, id uint) (*dto.StandardRelayResponse, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	if id == 0 {
		return nil, errx.New(constant.BadRequest, "ID不能为空", nil)
	}

	entity, err := s.StandardRelayRepo.GetByID(ctx, id)
	if err != nil {
		logger.Error("获取记录失败", slog.Uint64("id", uint64(id)), slog.Any("error", err))
		return nil, err
	}

	resp, err := dto.ConvertToStandardRelayResponse(entity)
	if err != nil {
		logger.Error("转换响应失败", slog.Any("error", err))
		return nil, err
	}

	return resp, nil
}

// Update 更新记录
func (s *StandardRelayService) Update(ctx context.Context, id uint, req dto.StandardRelayUpdateRequest) error {
	logger := contextx.GetLoggerFromContext(ctx)

	if id == 0 {
		return errx.New(constant.BadRequest, "ID不能为空", nil)
	}

	// 验证请求数据
	if err := req.Validate(); err != nil {
		logger.Error("更新请求验证失败", slog.Any("error", err))
		return err
	}

	// 检查记录是否存在
	current, err := s.StandardRelayRepo.GetByID(ctx, id)
	if err != nil {
		logger.Error("获取当前记录失败", slog.Uint64("id", uint64(id)), slog.Any("error", err))
		return err
	}

	// 转换为更新映射
	updates, err := req.ConvertToUpdateMap()
	if err != nil {
		logger.Error("转换更新映射失败", slog.Any("error", err))
		return err
	}

	// 无有效更新时返回
	if len(updates) == 0 {
		logger.Info(fmt.Sprintf("标准继电器更新请求无有效字段，继电器ID: %d", id))
		return nil
	}

	// 处理唯一字段（名称）
	if req.Name != nil && *req.Name != current.Name {
		if exists, err := s.StandardRelayRepo.ExistsByName(ctx, *req.Name, id); err != nil {
			logger.Error("检查名称唯一性失败", slog.Any("error", err))
			return err
		} else if exists {
			return errx.New(constant.Conflict, "名称已存在", nil)
		}
	}

	// 执行更新
	if err := s.StandardRelayRepo.Update(ctx, id, updates); err != nil {
		logger.Error("更新记录失败", slog.Uint64("id", uint64(id)), slog.Any("error", err))
		return err
	}

	// 更新缓存
	if s.ConfigCache != nil {
		go s.ConfigCache.UpdateStandardRelays(context.Background())
	}

	logger.Info("成功更新标准继电器", slog.Uint64("id", uint64(id)))
	return nil
}

// Delete 删除记录
func (s *StandardRelayService) Delete(ctx context.Context, id uint) error {
	logger := contextx.GetLoggerFromContext(ctx)

	if id == 0 {
		return errx.New(constant.BadRequest, "ID不能为空", nil)
	}

	// 检查记录是否存在
	if _, err := s.StandardRelayRepo.GetByID(ctx, id); err != nil {
		logger.Error("获取记录失败", slog.Uint64("id", uint64(id)), slog.Any("error", err))
		return err
	}

	// 执行删除
	if err := s.StandardRelayRepo.Delete(ctx, id); err != nil {
		logger.Error("删除记录失败", slog.Uint64("id", uint64(id)), slog.Any("error", err))
		return err
	}

	// 更新缓存
	if s.ConfigCache != nil {
		go s.ConfigCache.UpdateStandardRelays(context.Background())
	}

	logger.Info("成功删除标准继电器", slog.Uint64("id", uint64(id)))
	return nil
}

// List 分页列表
func (s *StandardRelayService) List(ctx context.Context, filter dto.StandardRelayFilter) ([]*dto.StandardRelayResponse, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 转换查询条件
	query := filter.ConvertToQuery()

	// 查询实体列表
	entities, total, err := s.StandardRelayRepo.List(ctx, query)
	if err != nil {
		logger.Error("查询列表失败", slog.Any("error", err))
		return nil, 0, err
	}

	// 批量转换响应
	responses, err := dto.ConvertToStandardRelayResponseList(entities)
	if err != nil {
		logger.Error("转换响应列表失败", slog.Any("error", err))
		return nil, 0, err
	}

	logger.Info("成功查询标准继电器列表", slog.Int("count", len(responses)), slog.Int64("total", total))
	return responses, total, nil
}

// ListAll 获取所有数据（走缓存）
func (s *StandardRelayService) ListAll(ctx context.Context) ([]*dto.StandardRelayResponse, error) {
	if s.ConfigCache == nil {
		return nil, errx.New(constant.InternalServerError, "配置缓存服务未初始化", nil)
	}
	return s.ConfigCache.GetStandardRelays(ctx)
}

// BatchDelete 批量删除
func (s *StandardRelayService) BatchDelete(ctx context.Context, req dto.StandardRelayDeleteRequest) error {
	logger := contextx.GetLoggerFromContext(ctx)

	// 验证请求数据
	if err := req.Validate(); err != nil {
		logger.Error("批量删除请求验证失败", slog.Any("error", err))
		return err
	}

	// 执行批量删除
	if err := s.StandardRelayRepo.BatchDelete(ctx, req.IDs); err != nil {
		logger.Error("批量删除失败", slog.Any("error", err))
		return err
	}

	// 更新缓存
	if s.ConfigCache != nil {
		go s.ConfigCache.UpdateStandardRelays(context.Background())
	}

	logger.Info("成功批量删除标准继电器", slog.Int("count", len(req.IDs)))
	return nil
}
