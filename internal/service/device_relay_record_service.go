package service

import (
	"context"
	"fmt"
	"time"

	"rbac-admin/internal/cache"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/model/entity"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"rbac-admin/pkg/logger"
	"rbac-admin/pkg/pubsubRedis"
)

// DeviceRelayRecordService 设备继电器记录服务
type DeviceRelayRecordService struct {
	DeviceRelayRecordRepo  *repository.DeviceRelayRecordRepository
	DeviceRelayRecordCache *cache.DeviceRelayRecordCache
	EventCollector         *pubsubRedis.EventCollector
	DeviceRepo             *repository.DeviceRepository
}

// NewDeviceRelayRecordService 创建新的设备继电器记录服务实例
func NewDeviceRelayRecordService(
	deviceRelayRecordRepo *repository.DeviceRelayRecordRepository,
	deviceRelayRecordCache *cache.DeviceRelayRecordCache,
	eventCollector *pubsubRedis.EventCollector,
	deviceRepo *repository.DeviceRepository,
) *DeviceRelayRecordService {
	service := &DeviceRelayRecordService{
		DeviceRelayRecordRepo:  deviceRelayRecordRepo,
		DeviceRelayRecordCache: deviceRelayRecordCache,
		EventCollector:         eventCollector,
		DeviceRepo:             deviceRepo,
	}

	// 订阅继电器切换ASK事件
	if eventCollector != nil {
		eventCollector.SubscribeRelaySwitchAsk(func(payload pubsubRedis.RelaySwitchAskPayload) {
			service.handleRelaySwitchAsk(payload)
		})
		logger.Logger.Info("设备继电器记录服务已订阅继电器切换ASK事件")
	}

	return service
}

// handleRelaySwitchAsk 处理继电器切换ASK事件
func (s *DeviceRelayRecordService) handleRelaySwitchAsk(payload pubsubRedis.RelaySwitchAskPayload) {
	ctx := context.Background()
	logger := contextx.GetLoggerFromContext(ctx)

	// 从Redis读取设备继电器记录ID、状态和时间
	cachedState, recordID, recordTime, err := s.DeviceRelayRecordCache.GetRelayRecord(ctx, payload.DeviceID, uint(payload.RelayIndex))
	if err != nil {
		logger.Error(fmt.Sprintf("获取继电器操作状态失败[deviceID=%d, relayIndex=%d]: %v", payload.DeviceID, payload.RelayIndex, err))
		return
	}

	// 匹配到发送缓存，立即删除缓存
	if err := s.DeviceRelayRecordCache.DeleteRelayRecord(ctx, payload.DeviceID, uint(payload.RelayIndex)); err != nil {
		logger.Warn(fmt.Sprintf("删除设备继电器记录缓存失败[deviceID=%d, relayIndex=%d]: %v", payload.DeviceID, payload.RelayIndex, err))
		// 删除失败不影响主流程，只记录警告
	} else {
		logger.Debug(fmt.Sprintf("设备继电器记录缓存已删除[deviceID=%d, relayIndex=%d]", payload.DeviceID, payload.RelayIndex))
	}

	// 如果不存在记录ID，说明这不是我们发起的操作，可能是设备主动上报的状态变化
	if recordID == 0 {
		logger.Debug(fmt.Sprintf("未找到对应的继电器操作记录[deviceID=%d, relayIndex=%d]，可能是设备主动上报", payload.DeviceID, payload.RelayIndex))
		return
	}

	// 验证状态一致性：payload中的状态应该与缓存中的状态一致
	if cachedState != payload.State {
		logger.Warn(fmt.Sprintf("继电器状态不一致[deviceID=%d, relayIndex=%d]: 缓存状态=%d, 实际状态=%d",
			payload.DeviceID, payload.RelayIndex, cachedState, payload.State))
		// 继续处理，使用payload中的实际状态
	}

	// 根据payload中的实际状态确定成功状态
	var successStatus entity.OperationStatus
	if payload.State == 1 {
		successStatus = entity.OperationStatusOpenSuccess
	} else {
		successStatus = entity.OperationStatusCloseSuccess
	}

	// 计算耗时（毫秒）：从记录时间到设备响应时间
	// 使用payload中的时间戳，更接近设备实际响应时间
	responseTime := time.Unix(payload.Time, 0)
	var duration int
	if !recordTime.IsZero() {
		duration = int(responseTime.Sub(recordTime).Milliseconds())
		// 如果计算出的耗时为负数，说明时间戳有问题，设置为0
		if duration < 0 {
			logger.Warn(fmt.Sprintf("计算出的耗时为负数[recordID=%d]，设置为0", recordID))
			duration = 0
		}
	}

	updates := map[string]interface{}{
		"status":   successStatus,
		"duration": duration,
	}

	// 更新设备继电器记录
	if err := s.DeviceRelayRecordRepo.Update(ctx, recordID, updates); err != nil {
		logger.Error(fmt.Sprintf("更新设备继电器记录失败[recordID=%d]: %v", recordID, err))
	} else {
		logger.Info(fmt.Sprintf("设备继电器记录更新成功[recordID=%d, status=%s, duration=%dms]", recordID, successStatus, duration))
	}
}

// SwitchRelay 控制继电器开关状态（使用发布订阅模式）
// ctx: 上下文，包含日志等信息
// req: 继电器控制请求，其中RelayIndex范围为0-31
func (s *DeviceRelayRecordService) SwitchRelay(ctx context.Context, req *dto.RelaySwitchRequest) error {
	logger := contextx.GetLoggerFromContext(ctx)

	// 1. 参数验证
	if err := s.validateSwitchRelayRequest(req); err != nil {
		return err
	}

	// 2. 解析设备ID
	deviceId, err := s.parseDeviceID(req.DeviceSN)
	if err != nil {
		logger.Error(fmt.Sprintf("解析设备序列号失败(SN:%s): %v", req.DeviceSN, err))
		return errx.New(constant.NotFound, "设备不存在", err)
	}

	// 3. 检查上一次操作是否还未返回结果
	_, recordID, recordTime, err := s.DeviceRelayRecordCache.GetRelayRecord(ctx, deviceId, uint(req.RelayIndex))
	if err != nil {
		logger.Warn(fmt.Sprintf("获取继电器操作记录失败[deviceID=%d, relayIndex=%d]: %v", deviceId, req.RelayIndex, err))
		// 获取失败不影响主流程，继续执行
	} else if recordID != 0 && !recordTime.IsZero() {
		// 如果存在记录且时间有效，检查时间间隔
		timeSinceLastOp := time.Since(recordTime)
		if timeSinceLastOp < 3*time.Second {
			logger.Warn(fmt.Sprintf("上一次操作还未返回结果[deviceID=%d, relayIndex=%d, 间隔=%v]", deviceId, req.RelayIndex, timeSinceLastOp))
			return errx.New(constant.Conflict, "上一次操作还未返回结果", nil)
		}
	}

	// 4. 验证设备在线状态
	if !s.DeviceRepo.GetOnlineStatus(deviceId) {
		logger.Warn(fmt.Sprintf("设备不在线(SN:%s, ID:%d)", req.DeviceSN, deviceId))
		return errx.New(constant.Conflict, "设备不在线", nil)
	}

	// 5. 先创建设备继电器记录（用于追踪操作结果）
	operatorUserIDPtr := &req.OperatorUserID
	record, err := s.createRelayRecord(ctx, req, deviceId, req.RelayIndex, operatorUserIDPtr)
	if err != nil {
		logger.Error(fmt.Sprintf("创建设备继电器记录失败[deviceID=%d, relayIndex=%d]: %v", deviceId, req.RelayIndex, err))
		return errx.New(constant.InternalServerError, "创建设备继电器记录失败", err)
	}

	// 6. 写入Redis缓存（用于后续ASK事件匹配）
	state := byte(0)
	if req.IsOpen {
		state = 1
	}
	if err := s.DeviceRelayRecordCache.SetRelayRecord(ctx, deviceId, uint(req.RelayIndex), state, record.ID); err != nil {
		logger.Error(fmt.Sprintf("写入设备继电器记录到Redis失败[deviceID=%d, relayIndex=%d, recordID=%d]: %v", deviceId, req.RelayIndex, record.ID, err))
		// Redis写入失败，需要删除已创建的数据库记录
		if delErr := s.DeviceRelayRecordRepo.Delete(ctx, record.ID); delErr != nil {
			logger.Error(fmt.Sprintf("删除设备继电器记录失败[recordID=%d]: %v", record.ID, delErr))
		}
		return errx.New(constant.InternalServerError, "写入Redis缓存失败", err)
	}

	// 7. 数据库和Redis都写入成功，发布继电器切换事件
	relayPayload := pubsubRedis.RelaySwitchPayload{
		DeviceID:   deviceId,
		RelayIndex: req.RelayIndex, // RelayIndex范围为0-31
		IsOpen:     req.IsOpen,
		Operator:   req.Operator,
	}
	if err := s.EventCollector.PublishRelaySwitch(relayPayload); err != nil {
		logger.Error(fmt.Sprintf("发布继电器切换事件失败[deviceID=%d, relayIndex=%d, isOpen=%t]: %v", deviceId, req.RelayIndex, req.IsOpen, err))

		// 事件发布失败，更新记录状态为失败，并删除缓存
		var failedStatus entity.OperationStatus
		if req.IsOpen {
			failedStatus = entity.OperationStatusOpenFailed
		} else {
			failedStatus = entity.OperationStatusCloseFailed
		}
		updates := map[string]interface{}{
			"status": failedStatus,
		}
		if updateErr := s.DeviceRelayRecordRepo.Update(ctx, record.ID, updates); updateErr != nil {
			logger.Error(fmt.Sprintf("更新设备继电器记录状态失败[recordID=%d]: %v", record.ID, updateErr))
		}

		// 删除缓存
		if delErr := s.DeviceRelayRecordCache.DeleteRelayRecord(ctx, deviceId, uint(req.RelayIndex)); delErr != nil {
			logger.Warn(fmt.Sprintf("删除设备继电器记录缓存失败[deviceID=%d, relayIndex=%d]: %v", deviceId, req.RelayIndex, delErr))
		}

		return errx.New(constant.InternalServerError, "事件发布失败", err)
	}

	logger.Info(fmt.Sprintf("继电器切换事件已发布[deviceID=%d, relayIndex=%d, isOpen=%t, recordID=%d]",
		deviceId, req.RelayIndex, req.IsOpen, record.ID))

	return nil
}

// validateSwitchRelayRequest 验证继电器控制请求参数
func (s *DeviceRelayRecordService) validateSwitchRelayRequest(req *dto.RelaySwitchRequest) error {
	if req == nil {
		return errx.New(constant.BadRequest, "请求参数不能为空", nil)
	}
	if req.Operator == "" {
		return errx.New(constant.BadRequest, "操作者不能为空", nil)
	}
	if req.RelayIndex < 0 || req.RelayIndex > 31 {
		return errx.New(constant.BadRequest, fmt.Sprintf("无效的继电器索引: %d，必须在0-31范围内", req.RelayIndex), nil)
	}
	if req.DeviceSN == "" {
		return errx.New(constant.BadRequest, "设备序列号不能为空", nil)
	}
	return nil
}

// parseDeviceID 解析设备序列号为设备ID
func (s *DeviceRelayRecordService) parseDeviceID(deviceSN string) (uint32, error) {
	var deviceId uint32
	_, err := fmt.Sscanf(deviceSN, "%d", &deviceId)
	if err != nil {
		return 0, err
	}
	if deviceId == 0 {
		return 0, fmt.Errorf("设备ID不能为0")
	}
	return deviceId, nil
}

// createRelayRecord 创建设备继电器记录
func (s *DeviceRelayRecordService) createRelayRecord(ctx context.Context, req *dto.RelaySwitchRequest, deviceId uint32, relayIndex int, operatorUserID *uint) (*entity.DeviceRelayRecord, error) {
	// 确定操作状态
	operationStatus := entity.OperationStatusClosing
	if req.IsOpen {
		operationStatus = entity.OperationStatusOpening
	}

	// 创建设备继电器记录
	record := &entity.DeviceRelayRecord{
		DeviceSN:       deviceId,   // 使用设备ID作为设备号
		Channel:        relayIndex, // relayIndex是0-31，channel也是0-31
		Operator:       req.Operator,
		OperatorUserID: operatorUserID, // 操作者用户ID（如果存在）
		Status:         operationStatus,
	}

	if err := s.DeviceRelayRecordRepo.Create(ctx, record); err != nil {
		return nil, err
	}

	return record, nil
}

// ListDeviceRelayRecords 查询设备继电器记录列表
func (s *DeviceRelayRecordService) ListDeviceRelayRecords(ctx context.Context, filter dto.DeviceRelayRecordFilter) ([]*dto.DeviceRelayRecordResponse, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 转换过滤条件为实体查询条件
	query := entity.DeviceRelayRecordQuery{
		DeviceSN: filter.DeviceSN,
		Channel:  filter.Channel,
		Operator: filter.Operator,
		Pagination: entity.Pagination{
			PageNumber: filter.PageNumber,
			PageSize:   filter.PageSize,
		},
	}

	// 转换状态
	if filter.Status != nil && *filter.Status != "" {
		status := entity.OperationStatus(*filter.Status)
		query.Status = &status
	}

	// 转换时间
	if filter.StartTime != nil && *filter.StartTime > 0 {
		startTime := time.Unix(*filter.StartTime, 0)
		query.StartTime = &startTime
	}
	if filter.EndTime != nil && *filter.EndTime > 0 {
		endTime := time.Unix(*filter.EndTime, 0)
		query.EndTime = &endTime
	}

	// 查询记录
	records, total, err := s.DeviceRelayRecordRepo.List(ctx, query)
	if err != nil {
		logger.Error(fmt.Sprintf("查询设备继电器记录列表失败: %v", err))
		return nil, 0, err
	}

	// 转换为响应结构
	responses := make([]*dto.DeviceRelayRecordResponse, 0, len(records))
	for _, record := range records {
		responses = append(responses, &dto.DeviceRelayRecordResponse{
			ID:        record.ID,
			DeviceSN:  record.DeviceSN,
			Channel:   record.Channel,
			Operator:  record.Operator,
			Status:    string(record.Status),
			Duration:  record.Duration,
			Remark:    record.Remark,
			CreatedAt: record.CreatedAt.Unix(),
			UpdatedAt: record.UpdatedAt.Unix(),
		})
	}

	return responses, total, nil
}
