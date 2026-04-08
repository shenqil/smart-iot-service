package repository

import (
	"context"
	"fmt"

	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"

	"gorm.io/gorm"
)

type CompanyRepository struct {
	Db *gorm.DB
}

// GetByField 通用字段查询方法
func (r *CompanyRepository) GetByField(ctx context.Context, field string, value interface{}) (*entity.Company, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var company entity.Company

	err := r.Db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND deleted_at IS NULL", field), value).
		First(&company).Error

	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, errx.New(constant.NotFound, "公司不存在", nil)
	case err != nil:
		logger.Error(fmt.Sprintf("查询公司失败[%s]: %v", field, err))
		return nil, errx.New(constant.InternalServerError, "查询公司失败", err)
	}

	return &company, nil
}

// GetByID 通过ID查询公司
func (r *CompanyRepository) GetByID(ctx context.Context, id uint) (*entity.Company, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var company entity.Company

	err := r.Db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&company).Error

	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, errx.New(constant.NotFound, "公司不存在", nil)
	case err != nil:
		logger.Error(fmt.Sprintf("查询公司失败[ID=%d]: %v", id, err))
		return nil, errx.New(constant.InternalServerError, "查询公司失败", err)
	}

	return &company, nil
}

// ExistsByField 通用存在性检查
func (r *CompanyRepository) ExistsByField(ctx context.Context, field string, value interface{}) (bool, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var count int64

	err := r.Db.WithContext(ctx).
		Model(&entity.Company{}).
		Where(fmt.Sprintf("%s = ? AND deleted_at IS NULL", field), value).
		Count(&count).Error

	if err != nil {
		logger.Error(fmt.Sprintf("公司存在性检查失败[%s]: %v", field, err))
		return false, errx.New(constant.InternalServerError, "检查失败", err)
	}

	return count > 0, nil
}

// Create 创建公司
func (r *CompanyRepository) Create(ctx context.Context, company *entity.Company) error {
	logger := contextx.GetLoggerFromContext(ctx)
	if company == nil {
		return errx.New(constant.BadRequest, "公司参数不能为空", nil)
	}

	err := r.Db.WithContext(ctx).Create(company).Error
	if err != nil {
		logger.Error(fmt.Sprintf("创建公司失败: %v", err))
		return errx.New(constant.InternalServerError, "创建公司失败", err)
	}

	return nil
}

// Update 更新公司
func (r *CompanyRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	logger := contextx.GetLoggerFromContext(ctx)

	result := r.Db.WithContext(ctx).
		Model(&entity.Company{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates)

	if result.Error != nil {
		logger.Error(fmt.Sprintf("更新公司失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "更新公司失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "公司不存在", nil)
	}

	return nil
}

// Delete 删除公司
func (r *CompanyRepository) Delete(ctx context.Context, id uint) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		Delete(&entity.Company{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除公司失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "删除公司失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "公司不存在", nil)
	}

	return nil
}

// BatchDelete 批量删除公司
func (r *CompanyRepository) BatchDelete(ctx context.Context, ids []uint) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("id IN (?) AND deleted_at IS NULL", ids).
		Delete(&entity.Company{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("批量删除公司失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "批量删除公司失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "无有效公司", nil)
	}

	return nil
}

// List 分页查询公司列表
func (r *CompanyRepository) List(ctx context.Context, query entity.CompanyQuery) ([]*entity.Company, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var companies []*entity.Company
	var total int64

	db := r.Db.WithContext(ctx).Model(&entity.Company{}).Where("deleted_at IS NULL")

	// 综合查询条件
	if query.Q != nil && *query.Q != "" {
		q := *query.Q
		db = db.Where("name LIKE ? OR code LIKE ?", "%"+q+"%", "%"+q+"%")
	} else {
		// 单独字段查询条件
		if query.Name != nil && *query.Name != "" {
			db = db.Where("name LIKE ?", "%"+*query.Name+"%")
		}
		if query.Code != nil && *query.Code != "" {
			db = db.Where("code = ?", *query.Code)
		}
	}

	// 基础条件
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计公司总数失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	// 排序（使用 id DESC 确保排序稳定）
	db = db.Order("id DESC")

	// 分页
	if query.PageSize > 0 {
		offset := (query.PageNumber - 1) * query.PageSize
		db = db.Offset(offset).Limit(query.PageSize)
	}

	// 执行查询
	if err := db.Find(&companies).Error; err != nil {
		logger.Error(fmt.Sprintf("查询公司列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return companies, total, nil
}
