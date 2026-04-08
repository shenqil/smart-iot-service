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

type StandardSensorRepository struct {
	Db *gorm.DB
}

func NewStandardSensorRepository(db *gorm.DB) *StandardSensorRepository {
	return &StandardSensorRepository{
		Db: db,
	}
}

// GetByID 直接从数据库获取数据
func (r *StandardSensorRepository) GetByID(ctx context.Context, id uint) (*entity.StandardSensor, error) {
	var sensor entity.StandardSensor
	if err := r.Db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&sensor).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errx.New(constant.NotFound, "传感器不存在", nil)
		}
		return nil, errx.New(constant.InternalServerError, "查询传感器失败", err)
	}
	return &sensor, nil
}

// GetByField 直接从数据库获取数据
func (r *StandardSensorRepository) GetByField(ctx context.Context, field string, value interface{}) (*entity.StandardSensor, error) {
	var sensor entity.StandardSensor
	query := r.Db.WithContext(ctx).Where("deleted_at IS NULL")

	switch field {
	case "name":
		query = query.Where("name = ?", value)
	default:
		return nil, errx.New(constant.BadRequest, "不支持的字段查询", nil)
	}

	if err := query.First(&sensor).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errx.New(constant.NotFound, "传感器不存在", nil)
		}
		return nil, errx.New(constant.InternalServerError, "查询传感器失败", err)
	}

	return &sensor, nil
}

// ExistsByField 直接从数据库检查字段是否存在
func (r *StandardSensorRepository) ExistsByField(ctx context.Context, field string, value interface{}) (bool, error) {
	query := r.Db.WithContext(ctx).Model(&entity.StandardSensor{}).Where("deleted_at IS NULL")

	switch field {
	case "name":
		query = query.Where("name = ?", value)
	default:
		return false, errx.New(constant.BadRequest, "不支持的字段查询", nil)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, errx.New(constant.InternalServerError, "检查字段存在性失败", err)
	}

	return count > 0, nil
}

// Create 创建记录
func (r *StandardSensorRepository) Create(ctx context.Context, sensor *entity.StandardSensor) error {
	if err := r.Db.WithContext(ctx).Create(sensor).Error; err != nil {
		return errx.New(constant.InternalServerError, "创建传感器失败", err)
	}
	return nil
}

// Update 更新记录
func (r *StandardSensorRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	result := r.Db.WithContext(ctx).
		Model(&entity.StandardSensor{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates)

	if result.Error != nil {
		return errx.New(constant.InternalServerError, "更新传感器失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "传感器不存在", nil)
	}

	return nil
}

// Delete 删除记录
func (r *StandardSensorRepository) Delete(ctx context.Context, id uint) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).Delete(&entity.StandardSensor{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除传感器失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "删除传感器失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "传感器不存在", nil)
	}

	return nil
}

// BatchDelete 批量删除
func (r *StandardSensorRepository) BatchDelete(ctx context.Context, ids []uint) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("id IN (?) AND deleted_at IS NULL", ids).
		Delete(&entity.StandardSensor{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("批量删除传感器失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "批量删除传感器失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "无有效记录", nil)
	}

	return nil
}

// List 直接走数据库查询（带分页和查询条件）
func (r *StandardSensorRepository) List(ctx context.Context, query entity.StandardSensorQuery) ([]*entity.StandardSensor, int64, error) {
	var sensors []*entity.StandardSensor
	var total int64

	db := r.Db.WithContext(ctx).Model(&entity.StandardSensor{}).Where("deleted_at IS NULL")

	// 名称查询优先
	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	} else if query.Q != "" {
		// Name 不存在时才使用 Q 搜索，仅搜索 name 字段
		db = db.Where("name LIKE ?", "%"+query.Q+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, errx.New(constant.InternalServerError, "查询总数失败", err)
	}

	// 排序（使用 id ASC 确保排序稳定）
	db = db.Order("id ASC")

	// 分页处理
	if query.PageSize > 0 {
		offset := (query.PageNumber - 1) * query.PageSize
		db = db.Offset(offset).Limit(query.PageSize)
	}

	// 执行查询
	if err := db.Find(&sensors).Error; err != nil {
		return nil, 0, errx.New(constant.InternalServerError, "查询列表失败", err)
	}

	return sensors, total, nil
}
