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

type MiniProgramMenuRepository struct {
	Db *gorm.DB
}

func NewMiniProgramMenuRepository(db *gorm.DB) *MiniProgramMenuRepository {
	return &MiniProgramMenuRepository{
		Db: db,
	}
}

// GetByID 通过ID查询
func (r *MiniProgramMenuRepository) GetByID(ctx context.Context, id uint) (*entity.MiniProgramMenu, error) {
	var menu entity.MiniProgramMenu
	if err := r.Db.WithContext(ctx).First(&menu, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errx.New(constant.NotFound, "记录不存在", nil)
		}
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}
	return &menu, nil
}

// ExistsByField 检查字段是否存在
func (r *MiniProgramMenuRepository) ExistsByField(ctx context.Context, field string, value interface{}) (bool, error) {
	var count int64
	query := r.Db.WithContext(ctx).Model(&entity.MiniProgramMenu{})

	switch field {
	case "code":
		query = query.Where("code = ?", value)
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

// ExistsByCode 检查代码是否存在（排除指定ID）
func (r *MiniProgramMenuRepository) ExistsByCode(ctx context.Context, code string, excludeID uint) (bool, error) {
	var count int64
	if err := r.Db.WithContext(ctx).
		Model(&entity.MiniProgramMenu{}).
		Where("code = ? AND id != ?", code, excludeID).
		Count(&count).Error; err != nil {
		return false, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return count > 0, nil
}

// Create 创建记录
func (r *MiniProgramMenuRepository) Create(ctx context.Context, menu *entity.MiniProgramMenu) error {
	logger := contextx.GetLoggerFromContext(ctx)

	// 基本验证
	if menu.Title == "" {
		return errx.New(constant.BadRequest, "标题不能为空", nil)
	}
	if menu.Code == "" {
		return errx.New(constant.BadRequest, "功能代码不能为空", nil)
	}

	// 创建记录
	if err := r.Db.WithContext(ctx).Create(menu).Error; err != nil {
		logger.Error(fmt.Sprintf("创建记录失败: %v", err))
		return errx.New(constant.InternalServerError, "创建失败", err)
	}

	return nil
}

// Update 更新记录
func (r *MiniProgramMenuRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	logger := contextx.GetLoggerFromContext(ctx)

	if len(updates) == 0 {
		return errx.New(constant.BadRequest, "没有提供更新字段", nil)
	}

	// 更新数据库
	result := r.Db.WithContext(ctx).
		Model(&entity.MiniProgramMenu{}).
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

// Delete 彻底删除记录（不使用软删除）
func (r *MiniProgramMenuRepository) Delete(ctx context.Context, id uint) error {
	logger := contextx.GetLoggerFromContext(ctx)

	// 彻底删除数据库记录
	result := r.Db.WithContext(ctx).
		Unscoped().
		Where("id = ?", id).
		Delete(&entity.MiniProgramMenu{})

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
func (r *MiniProgramMenuRepository) List(ctx context.Context, query entity.MiniProgramMenuQuery) ([]*entity.MiniProgramMenu, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var menus []*entity.MiniProgramMenu
	var total int64

	db := r.Db.WithContext(ctx).Model(&entity.MiniProgramMenu{})

	// 综合查询条件
	if query.Q != nil && *query.Q != "" {
		keywords := *query.Q
		db = db.Where("title LIKE ? OR code LIKE ?", "%"+keywords+"%", "%"+keywords+"%")
	}

	// 标题模糊匹配
	if query.Title != nil && *query.Title != "" {
		db = db.Where("title LIKE ?", "%"+*query.Title+"%")
	}

	// 代码精确匹配
	if query.Code != nil && *query.Code != "" {
		db = db.Where("code = ?", *query.Code)
	}

	// 状态精确匹配
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计总数失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	// 排序（按sort字段升序，相同sort按id升序）
	db = db.Order("sort ASC, id ASC")

	// 分页
	if query.PageSize > 0 {
		offset := (query.PageNumber - 1) * query.PageSize
		db = db.Offset(offset).Limit(query.PageSize)
	}

	// 执行查询
	if err := db.Find(&menus).Error; err != nil {
		logger.Error(fmt.Sprintf("查询列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return menus, total, nil
}

// ListAll 查询所有列表（不分页，用于受保护路由）
func (r *MiniProgramMenuRepository) ListAll(ctx context.Context) ([]*entity.MiniProgramMenu, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var menus []*entity.MiniProgramMenu

	// 只查询启用状态的记录，按sort排序
	if err := r.Db.WithContext(ctx).
		Model(&entity.MiniProgramMenu{}).
		Where("status = ?", entity.MiniProgramMenuStatusEnabled).
		Order("sort ASC, id ASC").
		Find(&menus).Error; err != nil {
		logger.Error(fmt.Sprintf("查询所有列表失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return menus, nil
}

// BatchDelete 批量彻底删除
func (r *MiniProgramMenuRepository) BatchDelete(ctx context.Context, ids []uint) error {
	logger := contextx.GetLoggerFromContext(ctx)

	if len(ids) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}

	// 批量彻底删除数据库记录
	result := r.Db.WithContext(ctx).
		Unscoped().
		Where("id IN (?)", ids).
		Delete(&entity.MiniProgramMenu{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("批量删除失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "批量删除失败", result.Error)
	}

	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "无有效记录", nil)
	}

	return nil
}

