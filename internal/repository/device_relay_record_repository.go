package repository

import (
	"context"
	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"time"

	"gorm.io/gorm"
	"fmt"
)

type DeviceRelayRecordRepository struct {
	Db *gorm.DB
}

// NewDeviceRelayRecordRepository 创建新的设备继电器记录仓库实例
func NewDeviceRelayRecordRepository(db *gorm.DB) *DeviceRelayRecordRepository {
	return &DeviceRelayRecordRepository{
		Db: db,
	}
}

// Create 新增设备继电器记录（ID会自动填充到record.ID中）
func (r *DeviceRelayRecordRepository) Create(ctx context.Context, record *entity.DeviceRelayRecord) error {
	logger := contextx.GetLoggerFromContext(ctx)
	if record == nil {
		return errx.New(constant.BadRequest, "参数不能为空", nil)
	}

	err := r.Db.WithContext(ctx).Create(record).Error
	if err != nil {
		logger.Error(fmt.Sprintf("创建设备继电器记录失败: %v", err))
		return errx.New(constant.InternalServerError, "创建失败", err)
	}

	return nil
}

// GetByID 通过ID查询设备继电器记录
func (r *DeviceRelayRecordRepository) GetByID(ctx context.Context, id uint) (*entity.DeviceRelayRecord, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var record entity.DeviceRelayRecord

	err := r.Db.WithContext(ctx).First(&record, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, errx.New(constant.NotFound, "设备继电器记录不存在", nil)
	}
	if err != nil {
		logger.Error(fmt.Sprintf("查询设备继电器记录失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return &record, nil
}

// Update 更新设备继电器记录
func (r *DeviceRelayRecordRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Model(&entity.DeviceRelayRecord{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		logger.Error(fmt.Sprintf("更新设备继电器记录失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "更新失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "设备继电器记录不存在", nil)
	}

	return nil
}

// List 查询设备继电器记录列表
func (r *DeviceRelayRecordRepository) List(ctx context.Context, query entity.DeviceRelayRecordQuery) ([]*entity.DeviceRelayRecord, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var records []*entity.DeviceRelayRecord
	var total int64

	db := r.Db.WithContext(ctx).Model(&entity.DeviceRelayRecord{}).Where("deleted_at IS NULL")

	// 设备号精确匹配
	if query.DeviceSN != nil {
		db = db.Where("device_sn = ?", *query.DeviceSN)
	}

	// 通道号精确匹配
	if query.Channel != nil {
		db = db.Where("channel = ?", *query.Channel)
	}

	// 操作者模糊匹配
	if query.Operator != nil && *query.Operator != "" {
		db = db.Where("operator LIKE ?", "%"+*query.Operator+"%")
	}

	// 状态精确匹配
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}

	// 开始时间过滤（按创建时间）
	if query.StartTime != nil {
		db = db.Where("created_at >= ?", *query.StartTime)
	}

	// 结束时间过滤（按创建时间）
	if query.EndTime != nil {
		db = db.Where("created_at <= ?", *query.EndTime)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计设备继电器记录总数失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	// 排序（按创建时间倒序）
	db = db.Order("created_at DESC")

	// 分页
	pageSize := query.GetPageSize()
	if pageSize > 0 {
		offset := (query.GetPageNumber() - 1) * pageSize
		db = db.Offset(offset).Limit(pageSize)
	}

	// 执行查询
	if err := db.Find(&records).Error; err != nil {
		logger.Error(fmt.Sprintf("查询设备继电器记录列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return records, total, nil
}

// Delete 删除设备继电器记录（软删除）
func (r *DeviceRelayRecordRepository) Delete(ctx context.Context, id uint) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		Delete(&entity.DeviceRelayRecord{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除设备继电器记录失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "删除失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "设备继电器记录不存在", nil)
	}

	return nil
}

// DeleteBeforeMonths 完全删除多少个月之前的记录（物理删除）
func (r *DeviceRelayRecordRepository) DeleteBeforeMonths(ctx context.Context, months int) (int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	if months <= 0 {
		return 0, errx.New(constant.BadRequest, "月数必须大于0", nil)
	}

	// 计算截止时间
	cutoffTime := time.Now().AddDate(0, -months, 0)

	// 物理删除指定时间之前的所有记录（包括已软删除的）
	result := r.Db.WithContext(ctx).
		Unscoped().
		Where("created_at < ?", cutoffTime).
		Delete(&entity.DeviceRelayRecord{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除设备继电器记录失败: %v", result.Error))
		return 0, errx.New(constant.InternalServerError, "删除失败", result.Error)
	}

	return result.RowsAffected, nil
}
