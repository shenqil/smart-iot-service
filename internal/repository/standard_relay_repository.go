package repository

import (
	"context"

	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"

	"fmt"
	"gorm.io/gorm"
)

type StandardRelayRepository struct {
	Db *gorm.DB
}

func NewStandardRelayRepository(db *gorm.DB) *StandardRelayRepository {
	return &StandardRelayRepository{
		Db: db,
	}
}

// GetByID 通过ID查询
func (r *StandardRelayRepository) GetByID(ctx context.Context, id uint) (*entity.StandardRelay, error) {
	var relay entity.StandardRelay
	if err := r.Db.WithContext(ctx).First(&relay, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errx.New(constant.NotFound, "记录不存在", nil)
		}
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}
	return &relay, nil
}

// ExistsByField 检查字段是否存在
func (r *StandardRelayRepository) ExistsByField(ctx context.Context, field string, value interface{}) (bool, error) {
	var count int64
	query := r.Db.WithContext(ctx).Model(&entity.StandardRelay{})

	switch field {
	case "name":
		query = query.Where("name = ?", value)
	case "id":
		query = query.Where("id = ?", value)
	default:
		return false, errx.New(constant.BadRequest, "不支持的字段", nil)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return count > 0, nil
}

// ExistsByName 检查名称是否存在（排除指定ID）
func (r *StandardRelayRepository) ExistsByName(ctx context.Context, name string, excludeID uint) (bool, error) {
	var count int64
	if err := r.Db.WithContext(ctx).
		Model(&entity.StandardRelay{}).
		Where("name = ? AND id != ?", name, excludeID).
		Count(&count).Error; err != nil {
		return false, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return count > 0, nil
}

// Create 创建记录
func (r *StandardRelayRepository) Create(ctx context.Context, relay *entity.StandardRelay) error {
	logger := contextx.GetLoggerFromContext(ctx)

	// 基本验证
	if relay.Name == "" {
		return errx.New(constant.BadRequest, "名称不能为空", nil)
	}

	// 创建记录
	if err := r.Db.WithContext(ctx).Create(relay).Error; err != nil {
		logger.Error(fmt.Sprintf("创建记录失败: %v", err))
		return errx.New(constant.InternalServerError, "创建失败", err)
	}

	return nil
}

// Update 更新记录
func (r *StandardRelayRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	logger := contextx.GetLoggerFromContext(ctx)

	if len(updates) == 0 {
		return errx.New(constant.BadRequest, "没有提供更新字段", nil)
	}

	// 更新数据库
	result := r.Db.WithContext(ctx).
		Model(&entity.StandardRelay{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		logger.Error(fmt.Sprintf("更新记录失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "更新失败", result.Error)
	}

	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "记录不存在", nil)
	}

	return nil
}

// Delete 删除记录
func (r *StandardRelayRepository) Delete(ctx context.Context, id uint) error {
	logger := contextx.GetLoggerFromContext(ctx)

	// 硬删除数据库记录
	result := r.Db.WithContext(ctx).
		Unscoped().
		Where("id = ?", id).
		Delete(&entity.StandardRelay{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除记录失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "删除失败", result.Error)
	}

	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "记录不存在", nil)
	}

	return nil
}

// List 分页查询
func (r *StandardRelayRepository) List(ctx context.Context, query entity.StandardRelayQuery) ([]*entity.StandardRelay, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var relays []*entity.StandardRelay
	var total int64

	db := r.Db.WithContext(ctx).Model(&entity.StandardRelay{}).Where("deleted_at IS NULL")

	// 名称查询优先
	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	} else if query.Q != "" {
		// Name 不存在时才使用 Q 搜索，仅搜索 name 字段
		db = db.Where("name LIKE ?", "%"+query.Q+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计总数失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	// 排序（使用 id ASC 确保排序稳定）
	db = db.Order("id ASC")

	// 分页
	if query.PageSize > 0 {
		offset := (query.PageNumber - 1) * query.PageSize
		db = db.Offset(offset).Limit(query.PageSize)
	}

	// 执行查询
	if err := db.Find(&relays).Error; err != nil {
		logger.Error(fmt.Sprintf("查询列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return relays, total, nil
}

// BatchDelete 批量删除
func (r *StandardRelayRepository) BatchDelete(ctx context.Context, ids []uint) error {
	logger := contextx.GetLoggerFromContext(ctx)

	if len(ids) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}

	// 批量硬删除数据库记录
	result := r.Db.WithContext(ctx).
		Unscoped().
		Where("id IN (?)", ids).
		Delete(&entity.StandardRelay{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("批量删除失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "批量删除失败", result.Error)
	}

	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "无有效记录", nil)
	}

	return nil
}
