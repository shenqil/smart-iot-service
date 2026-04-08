package service

import (
	"context"

	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	pubsub "rbac-admin/pkg/pubsubRedis"

	"log/slog"
)

type DeviceSimulatorService struct {
	SimulatorRepo *repository.DeviceSimulatorRepository
	EventCollector *pubsub.EventCollector
}

// Create 创建记录
func (s *DeviceSimulatorService) Create(ctx context.Context, req dto.DeviceSimulatorCreateRequest) (*dto.DeviceSimulatorResponse, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 验证请求数据
	if err := req.Validate(); err != nil {
		logger.Error("创建请求验证失败", slog.Any("error", err))
		return nil, err
	}

	// 检查设备SN号唯一性
	if exists, err := s.SimulatorRepo.ExistsByDeviceSN(ctx, req.DeviceSN); err != nil {
		logger.Error("检查设备SN号唯一性失败", slog.Any("error", err))
		return nil, err
	} else if exists {
		return nil, errx.New(constant.Conflict, "设备SN号已存在", nil)
	}

	// DTO转换为实体
	entity, err := req.ConvertToEntity()
	if err != nil {
		logger.Error("DTO转换失败", slog.Any("error", err))
		return nil, err
	}

	// 创建记录
	if err := s.SimulatorRepo.Create(ctx, entity); err != nil {
		logger.Error("创建记录失败", slog.Any("error", err))
		return nil, err
	}

	// 转换为响应
	resp, err := dto.ConvertToDeviceSimulatorResponse(entity)
	if err != nil {
		logger.Error("转换响应失败", slog.Any("error", err))
		return nil, err
	}

	logger.Info("成功创建设备模拟器", slog.Uint64("id", uint64(entity.ID)), slog.String("deviceSN", entity.DeviceSN))

	// 派发增加设备事件
	if s.EventCollector != nil {
		payload := pubsub.DeviceSimulatorChangePayload{
			Type:      pubsub.DeviceSimulatorChangeTypeAdd,
			DeviceSNs: []string{entity.DeviceSN},
		}
		if err := s.EventCollector.PublishDeviceSimulatorChange(payload); err != nil {
			logger.Error("派发增加设备事件失败", slog.Any("error", err))
			// 事件派发失败不影响创建结果，只记录日志
		} else {
			logger.Info("已派发增加设备事件", slog.String("deviceSN", entity.DeviceSN))
		}
	}

	return resp, nil
}

// List 分页查询列表
func (s *DeviceSimulatorService) List(ctx context.Context, req dto.DeviceSimulatorListRequest) (*dto.PaginationResponse, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 转换为查询条件
	query := req.ConvertToQuery()

	// 查询列表
	simulators, total, err := s.SimulatorRepo.List(ctx, query)
	if err != nil {
		logger.Error("查询列表失败", slog.Any("error", err))
		return nil, err
	}

	// 转换为响应列表
	records := make([]*dto.DeviceSimulatorResponse, 0, len(simulators))
	for _, simulator := range simulators {
		resp, err := dto.ConvertToDeviceSimulatorResponse(simulator)
		if err != nil {
			logger.Error("转换响应失败", slog.Any("error", err))
			continue
		}
		records = append(records, resp)
	}

	return &dto.PaginationResponse{
		Records:    records,
		TotalCount: total,
	}, nil
}

// BatchDelete 批量删除
func (s *DeviceSimulatorService) BatchDelete(ctx context.Context, req dto.DeviceSimulatorBatchDeleteRequest) error {
	logger := contextx.GetLoggerFromContext(ctx)

	if len(req.IDs) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}

	// 先查询要删除的设备SN列表
	deviceSNs := make([]string, 0)
	for _, id := range req.IDs {
		simulator, err := s.SimulatorRepo.GetByID(ctx, id)
		if err != nil {
			logger.Warn("查询设备模拟器失败，跳过", slog.Uint64("id", uint64(id)), slog.Any("error", err))
			continue
		}
		deviceSNs = append(deviceSNs, simulator.DeviceSN)
	}

	// 批量删除
	if err := s.SimulatorRepo.BatchDelete(ctx, req.IDs); err != nil {
		logger.Error("批量删除失败", slog.Any("error", err), slog.Any("ids", req.IDs))
		return err
	}

	logger.Info("成功批量删除设备模拟器", slog.Int("count", len(req.IDs)), slog.Any("ids", req.IDs))

	// 派发减少设备事件
	if s.EventCollector != nil && len(deviceSNs) > 0 {
		payload := pubsub.DeviceSimulatorChangePayload{
			Type:      pubsub.DeviceSimulatorChangeTypeRemove,
			DeviceSNs: deviceSNs,
		}
		if err := s.EventCollector.PublishDeviceSimulatorChange(payload); err != nil {
			logger.Error("派发减少设备事件失败", slog.Any("error", err))
			// 事件派发失败不影响删除结果，只记录日志
		} else {
			logger.Info("已派发减少设备事件", slog.Int("count", len(deviceSNs)), slog.Any("deviceSNs", deviceSNs))
		}
	}

	return nil
}

