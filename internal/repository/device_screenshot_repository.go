package repository

import (
	"context"

	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"

	"gorm.io/gorm"
	"fmt"
)

type DeviceScreenshotRepository struct {
	Db *gorm.DB
}

// Create 创建
func (r *DeviceScreenshotRepository) Create(ctx context.Context, record *entity.DeviceScreenshot) error {
	logger := contextx.GetLoggerFromContext(ctx)
	if record == nil {
		return errx.New(constant.BadRequest, "参数不能为空", nil)
	}
	if err := r.Db.WithContext(ctx).Create(record).Error; err != nil {
		logger.Error(fmt.Sprintf("创建失败: %v", err))
		return errx.New(constant.InternalServerError, "创建失败", err)
	}
	return nil
}

// Update 更新
func (r *DeviceScreenshotRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Model(&entity.DeviceScreenshot{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		logger.Error(fmt.Sprintf("更新失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "更新失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "记录不存在", nil)
	}
	return nil
}

// Delete 删除
func (r *DeviceScreenshotRepository) Delete(ctx context.Context, id uint) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		Delete(&entity.DeviceScreenshot{})
	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "删除失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "记录不存在", nil)
	}
	return nil
}

// BatchDelete 批量删除
func (r *DeviceScreenshotRepository) BatchDelete(ctx context.Context, ids []uint) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("id IN (?) AND deleted_at IS NULL", ids).
		Delete(&entity.DeviceScreenshot{})
	if result.Error != nil {
		logger.Error(fmt.Sprintf("批量删除失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "批量删除失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "无有效记录", nil)
	}
	return nil
}

// GetByID
func (r *DeviceScreenshotRepository) GetByID(ctx context.Context, id uint) (*entity.DeviceScreenshot, error) {
	var record entity.DeviceScreenshot
	err := r.Db.WithContext(ctx).First(&record, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, errx.New(constant.NotFound, "记录不存在", nil)
	}
	if err != nil {
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}
	return &record, nil
}

// List 查询
func (r *DeviceScreenshotRepository) List(ctx context.Context, query entity.DeviceScreenshotQuery) ([]*entity.DeviceScreenshot, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var list []*entity.DeviceScreenshot
	var total int64

	db := r.Db.WithContext(ctx).Model(&entity.DeviceScreenshot{}).Where("deleted_at IS NULL")

	if query.DeviceSN != nil {
		db = db.Where("device_sn = ?", *query.DeviceSN)
	}
	if query.StartTime != nil {
		db = db.Where("capture_time >= ?", *query.StartTime)
	}
	if query.EndTime != nil {
		db = db.Where("capture_time <= ?", *query.EndTime)
	}

	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	// 排序（使用 capture_time DESC, id DESC 确保排序稳定）
	db = db.Order("capture_time DESC, id DESC")

	if query.PageSize > 0 {
		offset := (query.PageNumber - 1) * query.PageSize
		db = db.Offset(offset).Limit(query.PageSize)
	}

	if err := db.Find(&list).Error; err != nil {
		logger.Error(fmt.Sprintf("查询失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}
	return list, total, nil
}
