package repository

import (
	"context"
	"strconv"

	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"

	"fmt"
	"gorm.io/gorm"
)

type WaterFertilizerTemplateRepository struct {
	Db *gorm.DB
}

func NewWaterFertilizerTemplateRepository(db *gorm.DB) *WaterFertilizerTemplateRepository {
	return &WaterFertilizerTemplateRepository{
		Db: db,
	}
}

// GetByID 直接从数据库获取数据
func (r *WaterFertilizerTemplateRepository) GetByID(ctx context.Context, id uint) (*entity.WaterFertilizerTemplate, error) {
	var template entity.WaterFertilizerTemplate
	if err := r.Db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errx.New(constant.NotFound, "水肥模板不存在", nil)
		}
		return nil, errx.New(constant.InternalServerError, "查询水肥模板失败", err)
	}
	return &template, nil
}

// GetByField 直接从数据库获取数据
func (r *WaterFertilizerTemplateRepository) GetByField(ctx context.Context, field string, value interface{}) (*entity.WaterFertilizerTemplate, error) {
	var template entity.WaterFertilizerTemplate
	query := r.Db.WithContext(ctx).Where("deleted_at IS NULL")

	switch field {
	case "name":
		query = query.Where("name = ?", value)
	default:
		return nil, errx.New(constant.BadRequest, "不支持的字段查询", nil)
	}

	if err := query.First(&template).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errx.New(constant.NotFound, "水肥模板不存在", nil)
		}
		return nil, errx.New(constant.InternalServerError, "查询水肥模板失败", err)
	}

	return &template, nil
}

// ExistsByField 直接从数据库检查字段是否存在
func (r *WaterFertilizerTemplateRepository) ExistsByField(ctx context.Context, field string, value interface{}) (bool, error) {
	query := r.Db.WithContext(ctx).Model(&entity.WaterFertilizerTemplate{}).Where("deleted_at IS NULL")

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
func (r *WaterFertilizerTemplateRepository) Create(ctx context.Context, template *entity.WaterFertilizerTemplate) error {
	if err := r.Db.WithContext(ctx).Create(template).Error; err != nil {
		return errx.New(constant.InternalServerError, "创建水肥模板失败", err)
	}
	return nil
}

// Update 更新记录
func (r *WaterFertilizerTemplateRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	result := r.Db.WithContext(ctx).
		Model(&entity.WaterFertilizerTemplate{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates)

	if result.Error != nil {
		return errx.New(constant.InternalServerError, "更新水肥模板失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "水肥模板不存在", nil)
	}

	return nil
}

// Delete 删除记录
func (r *WaterFertilizerTemplateRepository) Delete(ctx context.Context, id uint) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Unscoped().
		Where("id = ?", id).
		Delete(&entity.WaterFertilizerTemplate{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除水肥模板失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "删除水肥模板失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "水肥模板不存在", nil)
	}

	return nil
}

// BatchDelete 批量删除
func (r *WaterFertilizerTemplateRepository) BatchDelete(ctx context.Context, ids []uint) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Unscoped().
		Where("id IN (?)", ids).
		Delete(&entity.WaterFertilizerTemplate{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("批量删除水肥模板失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "批量删除水肥模板失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "无有效记录", nil)
	}

	return nil
}

// List 直接走数据库查询（带分页和查询条件）
func (r *WaterFertilizerTemplateRepository) List(ctx context.Context, query entity.WaterFertilizerTemplateQuery) ([]*entity.WaterFertilizerTemplate, int64, error) {
	var templates []*entity.WaterFertilizerTemplate
	var total int64

	db := r.Db.WithContext(ctx).Model(&entity.WaterFertilizerTemplate{}).Where("deleted_at IS NULL")

	// 关键词查询
	if query.Q != "" {
		if id, err := strconv.Atoi(query.Q); err == nil {
			db = db.Where("id = ? OR name LIKE ?", id, "%"+query.Q+"%")
		} else {
			db = db.Where("name LIKE ?", "%"+query.Q+"%")
		}
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
	if err := db.Find(&templates).Error; err != nil {
		return nil, 0, errx.New(constant.InternalServerError, "查询列表失败", err)
	}

	return templates, total, nil
}
