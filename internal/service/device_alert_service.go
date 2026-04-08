package service

import (
	"context"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/repository"
	"strconv"
)

type DeviceAlarmService struct {
	DeviceAlarmRepo *repository.DeviceAlarmRepository
	DeviceRepo      *repository.DeviceRepository
}

// NewDeviceAlarmService 创建新的设备告警服务实例
func NewDeviceAlarmService(deviceAlarmRepo *repository.DeviceAlarmRepository, deviceRepo *repository.DeviceRepository) *DeviceAlarmService {
	return &DeviceAlarmService{
		DeviceAlarmRepo: deviceAlarmRepo,
		DeviceRepo:      deviceRepo,
	}
}

// List 获取告警列表（DeviceAlarm）
func (s *DeviceAlarmService) List(ctx context.Context, req dto.DeviceAlarmSearchRequest) ([]*dto.DeviceAlarmResponse, int64, error) {
	// 转换查询条件
	query := req.ToQuery()

	// 如果给定了园区 code，查询该园区下的所有设备 IDs
	if query.SiteCode != nil && *query.SiteCode != "" {
		deviceIds, err := s.getDeviceIdsBySiteCode(ctx, *query.SiteCode)
		if err != nil {
			return nil, 0, err
		}
		// 如果该园区下没有设备，直接返回空结果
		if len(deviceIds) == 0 {
			return []*dto.DeviceAlarmResponse{}, 0, nil
		}
		query.DeviceIds = deviceIds
	}

	// 查询数据
	companyIDPtr := &req.CompanyID
	alarms, total, err := s.DeviceAlarmRepo.List(ctx, query, companyIDPtr)
	if err != nil {
		return nil, 0, err
	}

	// 转换为响应格式
	var responses []*dto.DeviceAlarmResponse
	for _, a := range alarms {
		resp := dto.ConvertToDeviceAlarmResponse(a)
		responses = append(responses, resp)
	}

	return responses, total, nil
}

// getDeviceIdsBySiteCode 根据园区编码获取设备编号列表（作为字符串）
func (s *DeviceAlarmService) getDeviceIdsBySiteCode(ctx context.Context, siteCode string) ([]string, error) {
	// 调用 repository 层方法查询设备编号列表
	deviceNums, err := s.DeviceRepo.GetDeviceNumsBySiteCode(ctx, siteCode)
	if err != nil {
		return nil, err
	}

	// 将 uint 转换为 string
	deviceNumStrings := make([]string, 0, len(deviceNums))
	for _, num := range deviceNums {
		deviceNumStrings = append(deviceNumStrings, strconv.FormatUint(uint64(num), 10))
	}

	return deviceNumStrings, nil
}

// BatchProcess 批量处理告警（DeviceAlarm）
func (s *DeviceAlarmService) BatchProcess(ctx context.Context, req dto.DeviceAlarmBatchProcessRequest) (*dto.DeviceAlarmBatchProcessResponse, error) {
	// UserID 和 CompanyID 已在 handler 层验证并填充
	processedBy := req.UserID

	var count int64
	var err error

	companyIDPtr := &req.CompanyID
	if len(req.IDs) == 0 {
		// 一键处理所有告警
		count, err = s.DeviceAlarmRepo.ProcessAllAlerts(ctx, processedBy, req.ProcessorName, req.ProcessRemark, companyIDPtr)
	} else if len(req.IDs) == 1 {
		// 处理单个告警
		err = s.DeviceAlarmRepo.ProcessAlert(ctx, req.IDs[0], processedBy, req.ProcessorName, req.ProcessRemark, companyIDPtr)
		if err == nil {
			count = 1
		}
	} else {
		// 批量处理告警
		count, err = s.DeviceAlarmRepo.ProcessBatchAlerts(ctx, req.IDs, processedBy, req.ProcessorName, req.ProcessRemark, companyIDPtr)
	}

	if err != nil {
		return nil, err
	}

	return &dto.DeviceAlarmBatchProcessResponse{
		ProcessedCount: count,
	}, nil
}
