package repository

import (
	"context"
	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"strings"

	"fmt"
	"gorm.io/gorm"
)

type DeviceTypeRepository struct {
	Db *gorm.DB
}

// ExistsByID 检查ID是否存在
func (r *DeviceTypeRepository) ExistsByID(ctx context.Context, id uint) (bool, error) {
	if id == 0 {
		return false, errx.New(constant.BadRequest, "ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var count int64
	err := r.Db.WithContext(ctx).Model(&entity.DeviceType{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Count(&count).Error
	if err != nil {
		logger.Error(fmt.Sprintf("检查ID是否存在失败: %v", err))
		return false, errx.New(constant.InternalServerError, constant.MsgInternalServerErr, err)
	}
	return count > 0, nil
}

// ExistsByType 检查类型是否存在
func (r *DeviceTypeRepository) ExistsByType(ctx context.Context, typ string) (bool, error) {
	if strings.TrimSpace(typ) == "" {
		return false, errx.New(constant.BadRequest, "类型标识不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var count int64
	err := r.Db.WithContext(ctx).Model(&entity.DeviceType{}).
		Where("type = ? AND deleted_at IS NULL", typ).
		Count(&count).Error
	if err != nil {
		logger.Error(fmt.Sprintf("检查类型是否存在失败: %v", err))
		return false, errx.New(constant.InternalServerError, constant.MsgInternalServerErr, err)
	}
	return count > 0, nil
}

// Create 创建设备类型
func (r *DeviceTypeRepository) Create(ctx context.Context, deviceType *entity.DeviceType) error {
	if deviceType == nil {
		return errx.New(constant.BadRequest, constant.MsgInvalidInput, nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	err := r.Db.WithContext(ctx).Create(deviceType).Error
	if err != nil {
		logger.Error(fmt.Sprintf("创建 DeviceType 失败: %v", err))
		return errx.New(constant.InternalServerError, constant.MsgInternalServerErr, err)
	}
	return nil
}

// GetByID 根据ID获取设备类型
func (r *DeviceTypeRepository) GetByID(ctx context.Context, id uint) (*entity.DeviceType, error) {
	if id == 0 {
		return nil, errx.New(constant.BadRequest, "ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var deviceType entity.DeviceType
	err := r.Db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&deviceType).Error
	if err != nil {
		logger.Error(fmt.Sprintf("获取 DeviceType 失败: %v", err))
		if err == gorm.ErrRecordNotFound {
			return nil, errx.New(constant.NotFound, constant.MsgNotFound, nil)
		}
		return nil, errx.New(constant.InternalServerError, constant.MsgInternalServerErr, err)
	}
	return &deviceType, nil
}

// GetByType 根据类型标识获取设备类型
func (r *DeviceTypeRepository) GetByType(ctx context.Context, typ string) (*entity.DeviceType, error) {
	if strings.TrimSpace(typ) == "" {
		return nil, errx.New(constant.BadRequest, "类型标识不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var deviceType entity.DeviceType
	err := r.Db.WithContext(ctx).Where("type = ? AND deleted_at IS NULL", typ).First(&deviceType).Error
	if err != nil {
		logger.Error(fmt.Sprintf("根据类型获取 DeviceType 失败: %v", err))
		if err == gorm.ErrRecordNotFound {
			return nil, errx.New(constant.NotFound, constant.MsgNotFound, nil)
		}
		return nil, errx.New(constant.InternalServerError, constant.MsgInternalServerErr, err)
	}
	return &deviceType, nil
}

// Delete 删除设备类型
func (r *DeviceTypeRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return errx.New(constant.BadRequest, "ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Unscoped().
		Where("id = ?", id).
		Delete(&entity.DeviceType{})
	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除 DeviceType 失败: %v", result.Error))
		return errx.New(constant.InternalServerError, constant.MsgInternalServerErr, result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, constant.MsgNotFound, nil)
	}
	return nil
}

// BatchDelete 批量删除设备类型
func (r *DeviceTypeRepository) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return errx.New(constant.BadRequest, "删除ID列表不能为空", nil)
	}

	// 验证ID列表
	for _, id := range ids {
		if id == 0 {
			return errx.New(constant.BadRequest, "ID不能为空", nil)
		}
	}

	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Unscoped().
		Where("id IN (?)", ids).
		Delete(&entity.DeviceType{})
	if result.Error != nil {
		logger.Error(fmt.Sprintf("批量删除 DeviceType 失败: %v", result.Error))
		return errx.New(constant.InternalServerError, constant.MsgInternalServerErr, result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, constant.MsgNotFound, nil)
	}
	return nil
}

// List 分页查询设备类型列表
func (r *DeviceTypeRepository) List(ctx context.Context, query entity.DeviceTypeQuery) ([]*entity.DeviceType, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var deviceTypes []*entity.DeviceType
	var total int64

	// 构建基础查询
	db := r.Db.WithContext(ctx).Model(&entity.DeviceType{}).Where("deleted_at IS NULL")

	// 应用过滤条件
	db = r.applyFilters(db, query)

	// 统计总数
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计 DeviceType 总数失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, constant.MsgInternalServerErr, err)
	}

	// 排序（使用 id DESC 确保排序稳定）
	db = db.Order("id DESC")

	// 应用分页
	if query.Pagination.PageSize > 0 {
		offset := (query.Pagination.PageNumber - 1) * query.Pagination.PageSize
		db = db.Offset(offset).Limit(query.Pagination.PageSize)
	}

	// 执行查询
	if err := db.Find(&deviceTypes).Error; err != nil {
		logger.Error(fmt.Sprintf("查询 DeviceType 列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, constant.MsgInternalServerErr, err)
	}

	return deviceTypes, total, nil
}

// applyFilters 应用过滤条件
func (r *DeviceTypeRepository) applyFilters(db *gorm.DB, query entity.DeviceTypeQuery) *gorm.DB {
	if strings.TrimSpace(query.Name) != "" {
		db = db.Where("name LIKE ?", "%"+strings.TrimSpace(query.Name)+"%")
	} else if strings.TrimSpace(query.Q) != "" {
		// Name 不存在时才使用 Q 搜索，搜索 name 字段（模糊匹配）或 type 字段（精确匹配）
		q := strings.TrimSpace(query.Q)
		db = db.Where("name LIKE ? OR type = ?", "%"+q+"%", q)
	}
	return db
}

// Update 更新设备类型
func (r *DeviceTypeRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	if id == 0 {
		return errx.New(constant.BadRequest, "ID不能为空", nil)
	}
	if len(updates) == 0 {
		return errx.New(constant.BadRequest, "更新字段不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).Model(&entity.DeviceType{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates)
	if result.Error != nil {
		logger.Error(fmt.Sprintf("更新 DeviceType 失败: %v", result.Error))
		return errx.New(constant.InternalServerError, constant.MsgInternalServerErr, result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, constant.MsgNotFound, nil)
	}
	return nil
}
