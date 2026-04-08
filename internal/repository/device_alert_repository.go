package repository

import (
	"context"
	"fmt"
	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"rbac-admin/pkg/pubsubRedis"
	"time"

	"gorm.io/gorm"
)

type DeviceAlarmRepository struct {
	Db             *gorm.DB
	eventCollector *pubsubRedis.EventCollector
}

// NewDeviceAlarmRepository 创建新的设备告警仓库实例
func NewDeviceAlarmRepository(db *gorm.DB, eventCollector *pubsubRedis.EventCollector) *DeviceAlarmRepository {
	repo := &DeviceAlarmRepository{
		Db:             db,
		eventCollector: eventCollector,
	}

	// 订阅通用设备告警事件
	if eventCollector != nil {
		eventCollector.SubscribeDeviceAlarm(func(payload pubsubRedis.DeviceAlarmPayload) {
			repo.handleDeviceAlarmEvent(payload)
		})
	}

	return repo
}

// handleDeviceAlarmEvent 处理通用设备告警事件
func (r *DeviceAlarmRepository) handleDeviceAlarmEvent(payload pubsubRedis.DeviceAlarmPayload) {
	alarm := &entity.DeviceAlarm{
		AlarmLevel:   entity.AlarmLevel(payload.AlarmLevel),
		AlarmType:    payload.AlarmType,
		AlarmSubType: payload.AlarmSubType,
		EnterpriseId: payload.EnterpriseId,
		DeviceId:     payload.DeviceId,
		DeviceName:   payload.DeviceName,
		Description:  payload.Description,
		Value:        payload.Value,
		Threshold:    payload.Threshold,
		IsProcessed:  entity.ProcessStatusUnprocessed,
		CreatedAt:    time.Unix(payload.Timestamp, 0),
	}

	if err := r.Db.Create(alarm).Error; err != nil {
		fmt.Printf("Failed to create device alarm: %v\n", err)
	}
}

// List 获取告警列表（DeviceAlarm）
func (r *DeviceAlarmRepository) List(ctx context.Context, query entity.DeviceAlarmQuery, companyID *uint) ([]*entity.DeviceAlarm, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 验证CompanyID
	if companyID == nil {
		return nil, 0, errx.New(constant.Forbidden, "未获取到公司ID，无法查询告警", nil)
	}

	enterpriseId := fmt.Sprint(*companyID)
	db := r.Db.WithContext(ctx).Model(&entity.DeviceAlarm{}).Where("enterprise_id = ?", enterpriseId)

	// 应用过滤条件
	if query.Q != nil && *query.Q != "" {
		db = db.Where("device_name LIKE ? OR device_id = ?", "%"+*query.Q+"%", *query.Q)
	}
	if query.DeviceId != nil {
		db = db.Where("device_id = ?", *query.DeviceId)
	}
	// 如果给定了设备 ID 列表（通过园区编码查询得到），使用这些设备 IDs 进行过滤
	if len(query.DeviceIds) > 0 {
		db = db.Where("device_id IN (?)", query.DeviceIds)
	}
	if query.AlarmType != nil {
		db = db.Where("alarm_type = ?", *query.AlarmType)
	}
	if query.AlarmLevel != nil {
		db = db.Where("alarm_level = ?", *query.AlarmLevel)
	}
	if query.IsProcessed != nil {
		db = db.Where("is_processed = ?", *query.IsProcessed)
	}
	if query.ProcessedBy != nil {
		db = db.Where("processed_by = ?", *query.ProcessedBy)
	}
	if query.StartTime != nil {
		db = db.Where("created_at >= ?", *query.StartTime)
	}
	if query.EndTime != nil {
		db = db.Where("created_at <= ?", *query.EndTime)
	}

	// 获取总数
	var total int64
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("获取告警总数失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	// 分页查询
	var alerts []*entity.DeviceAlarm
	offset := (query.GetPageNumber() - 1) * query.GetPageSize()
	err := db.Order("is_processed ASC, created_at DESC").
		Offset(offset).
		Limit(query.GetPageSize()).
		Find(&alerts).Error

	if err != nil {
		logger.Error(fmt.Sprintf("获取告警列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return alerts, total, nil
}

// ProcessAlert 处理单个告警（DeviceAlarm）
func (r *DeviceAlarmRepository) ProcessAlert(ctx context.Context, id uint, processedBy uint, processorName, remark string, companyID *uint) error {
	logger := contextx.GetLoggerFromContext(ctx)

	// 验证CompanyID
	if companyID == nil {
		return errx.New(constant.Forbidden, "未获取到公司ID，无法处理告警", nil)
	}

	result := r.Db.WithContext(ctx).Model(&entity.DeviceAlarm{}).
		Where("id = ? AND enterprise_id = ? AND is_processed = ?", id, fmt.Sprint(*companyID), 0).
		Updates(map[string]interface{}{
			"is_processed":   entity.ProcessStatusProcessed,
			"processed_by":   processedBy,
			"processor_name": processorName,
			"process_remark": remark,
			"processed_at":   time.Now(),
		})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("处理告警失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "处理失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "告警记录不存在或已处理", nil)
	}
	return nil
}

// ProcessAllAlerts 处理所有未处理的告警（DeviceAlarm）
func (r *DeviceAlarmRepository) ProcessAllAlerts(ctx context.Context, processedBy uint, processorName, remark string, companyID *uint) (int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 验证CompanyID
	if companyID == nil {
		return 0, errx.New(constant.Forbidden, "未获取到公司ID，无法处理告警", nil)
	}

	result := r.Db.WithContext(ctx).Model(&entity.DeviceAlarm{}).
		Where("enterprise_id = ? AND is_processed = ?", fmt.Sprint(*companyID), 0).
		Updates(map[string]interface{}{
			"is_processed":   entity.ProcessStatusProcessed,
			"processed_by":   processedBy,
			"processor_name": processorName,
			"process_remark": remark,
			"processed_at":   time.Now(),
		})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("处理所有告警失败: %v", result.Error))
		return 0, errx.New(constant.InternalServerError, "处理失败", result.Error)
	}

	return result.RowsAffected, nil
}

// ProcessBatchAlerts 批量处理告警（DeviceAlarm）
func (r *DeviceAlarmRepository) ProcessBatchAlerts(ctx context.Context, alertIDs []uint, processedBy uint, processorName, remark string, companyID *uint) (int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 验证CompanyID
	if companyID == nil {
		return 0, errx.New(constant.Forbidden, "未获取到公司ID，无法处理告警", nil)
	}

	if len(alertIDs) == 0 {
		return 0, errx.New(constant.BadRequest, "告警ID列表不能为空", nil)
	}

	result := r.Db.WithContext(ctx).Model(&entity.DeviceAlarm{}).
		Where("id IN ? AND enterprise_id = ? AND is_processed = ?", alertIDs, fmt.Sprint(*companyID), 0).
		Updates(map[string]interface{}{
			"is_processed":   entity.ProcessStatusProcessed,
			"processed_by":   processedBy,
			"processor_name": processorName,
			"process_remark": remark,
			"processed_at":   time.Now(),
		})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("批量处理告警失败: %v", result.Error))
		return 0, errx.New(constant.InternalServerError, "处理失败", result.Error)
	}

	return result.RowsAffected, nil
}
