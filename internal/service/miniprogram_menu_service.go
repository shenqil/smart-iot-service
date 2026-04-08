package service

import (
	"context"

	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"

	"log/slog"
	"fmt"
)

type MiniProgramMenuService struct {
	MenuRepo *repository.MiniProgramMenuRepository
}

// Create 创建记录
func (s *MiniProgramMenuService) Create(ctx context.Context, req dto.MiniProgramMenuCreateRequest) (*dto.MiniProgramMenuResponse, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 验证请求数据
	if err := req.Validate(); err != nil {
		logger.Error("创建请求验证失败", slog.Any("error", err))
		return nil, err
	}

	// 检查代码唯一性
	if exists, err := s.MenuRepo.ExistsByField(ctx, "code", req.Code); err != nil {
		logger.Error("检查代码唯一性失败", slog.Any("error", err))
		return nil, err
	} else if exists {
		return nil, errx.New(constant.Conflict, "功能代码已存在", nil)
	}

	// DTO转换为实体
	entity, err := req.ConvertToEntity()
	if err != nil {
		logger.Error("DTO转换失败", slog.Any("error", err))
		return nil, err
	}

	// 创建记录
	if err := s.MenuRepo.Create(ctx, entity); err != nil {
		logger.Error("创建记录失败", slog.Any("error", err))
		return nil, err
	}

	// 转换为响应
	resp, err := dto.ConvertToMiniProgramMenuResponse(entity)
	if err != nil {
		logger.Error("转换响应失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功创建小程序功能栏配置", slog.Uint64("id", uint64(entity.ID)), slog.String("code", entity.Code))
	return resp, nil
}

// Get 获取单条记录
func (s *MiniProgramMenuService) Get(ctx context.Context, id uint) (*dto.MiniProgramMenuResponse, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	if id == 0 {
		return nil, errx.New(constant.BadRequest, "ID不能为空", nil)
	}

	entity, err := s.MenuRepo.GetByID(ctx, id)
	if err != nil {
		logger.Error("获取记录失败", slog.Uint64("id", uint64(id)), slog.Any("error", err))
		return nil, err
	}

	resp, err := dto.ConvertToMiniProgramMenuResponse(entity)
	if err != nil {
		logger.Error("转换响应失败", slog.Any("error", err))
		return nil, err
	}

	return resp, nil
}

// Update 更新记录
func (s *MiniProgramMenuService) Update(ctx context.Context, id uint, req dto.MiniProgramMenuUpdateRequest) error {
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
	current, err := s.MenuRepo.GetByID(ctx, id)
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
		logger.Info(fmt.Sprintf("小程序功能栏配置更新请求无有效字段，配置ID: %d", id))
		return nil
	}

	// 处理唯一字段（代码）
	if req.Code != nil && *req.Code != current.Code {
		if exists, err := s.MenuRepo.ExistsByCode(ctx, *req.Code, id); err != nil {
			logger.Error("检查代码唯一性失败", slog.Any("error", err))
			return err
		} else if exists {
			return errx.New(constant.Conflict, "功能代码已存在", nil)
		}
	}

	// 执行更新
	if err := s.MenuRepo.Update(ctx, id, updates); err != nil {
		logger.Error("更新记录失败", slog.Uint64("id", uint64(id)), slog.Any("error", err))
		return err
	}

	logger.Info("成功更新小程序功能栏配置", slog.Uint64("id", uint64(id)))
	return nil
}

// Delete 删除记录（彻底删除）
func (s *MiniProgramMenuService) Delete(ctx context.Context, id uint) error {
	logger := contextx.GetLoggerFromContext(ctx)

	if id == 0 {
		return errx.New(constant.BadRequest, "ID不能为空", nil)
	}

	// 检查记录是否存在
	if _, err := s.MenuRepo.GetByID(ctx, id); err != nil {
		logger.Error("获取记录失败", slog.Uint64("id", uint64(id)), slog.Any("error", err))
		return err
	}

	// 执行彻底删除
	if err := s.MenuRepo.Delete(ctx, id); err != nil {
		logger.Error("删除记录失败", slog.Uint64("id", uint64(id)), slog.Any("error", err))
		return err
	}

	logger.Info("成功删除小程序功能栏配置", slog.Uint64("id", uint64(id)))
	return nil
}

// List 分页列表
func (s *MiniProgramMenuService) List(ctx context.Context, filter dto.MiniProgramMenuFilter) ([]*dto.MiniProgramMenuResponse, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 转换查询条件
	query := filter.ConvertToQuery()

	// 查询实体列表
	entities, total, err := s.MenuRepo.List(ctx, query)
	if err != nil {
		logger.Error("查询列表失败", slog.Any("error", err))
		return nil, 0, err
	}

	// 批量转换响应
	responses, err := dto.ConvertToMiniProgramMenuResponseList(entities)
	if err != nil {
		logger.Error("转换响应列表失败", slog.Any("error", err))
		return nil, 0, err
	}

	logger.Info("成功查询小程序功能栏配置列表", slog.Int("count", len(responses)), slog.Int64("total", total))
	return responses, total, nil
}

// ListAll 获取所有列表（直接从数据库查询，不使用缓存）
func (s *MiniProgramMenuService) ListAll(ctx context.Context) ([]*dto.MiniProgramMenuResponse, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 直接从数据库查询所有启用的记录
	entities, err := s.MenuRepo.ListAll(ctx)
	if err != nil {
		logger.Error("从数据库获取所有小程序功能栏配置失败", slog.Any("error", err))
		return nil, err
	}

	// 批量转换响应
	responses, err := dto.ConvertToMiniProgramMenuResponseList(entities)
	if err != nil {
		logger.Error("转换响应列表失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功从数据库获取所有小程序功能栏配置", slog.Int("count", len(responses)))
	return responses, nil
}

// BatchDelete 批量删除（彻底删除）
func (s *MiniProgramMenuService) BatchDelete(ctx context.Context, req dto.MiniProgramMenuDeleteRequest) error {
	logger := contextx.GetLoggerFromContext(ctx)

	// 验证请求数据
	if err := req.Validate(); err != nil {
		logger.Error("批量删除请求验证失败", slog.Any("error", err))
		return err
	}

	// 执行批量彻底删除
	if err := s.MenuRepo.BatchDelete(ctx, req.IDs); err != nil {
		logger.Error("批量删除失败", slog.Any("error", err))
		return err
	}

	logger.Info("成功批量删除小程序功能栏配置", slog.Int("count", len(req.IDs)))
	return nil
}

