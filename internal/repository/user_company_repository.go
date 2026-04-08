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

type UserCompanyRepository struct {
	Db *gorm.DB
}

// Create 创建用户公司关联
func (r *UserCompanyRepository) Create(ctx context.Context, userCompany *entity.UserCompany) error {
	if userCompany == nil {
		return errx.New(constant.BadRequest, "用户公司关联参数不能为空", nil)
	}

	if userCompany.UserID == 0 {
		return errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}

	if userCompany.CompanyID == 0 {
		return errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	err := r.Db.WithContext(ctx).Create(userCompany).Error

	if err != nil {
		logger.Error(fmt.Sprintf("创建用户公司关联失败: %v", err))
		return errx.New(constant.InternalServerError, "创建用户公司关联失败", err)
	}

	return nil
}

// BatchCreate 批量创建用户公司关联
func (r *UserCompanyRepository) BatchCreate(ctx context.Context, userCompanies []*entity.UserCompany) error {
	if len(userCompanies) == 0 {
		return errx.New(constant.BadRequest, "用户公司关联列表不能为空", nil)
	}

	// 验证所有关联
	for _, userCompany := range userCompanies {
		if userCompany == nil {
			return errx.New(constant.BadRequest, "用户公司关联参数不能为空", nil)
		}
		if userCompany.UserID == 0 {
			return errx.New(constant.BadRequest, "用户ID不能为空", nil)
		}
		if userCompany.CompanyID == 0 {
			return errx.New(constant.BadRequest, "公司ID不能为空", nil)
		}
	}

	logger := contextx.GetLoggerFromContext(ctx)
	err := r.Db.WithContext(ctx).Create(userCompanies).Error

	if err != nil {
		logger.Error(fmt.Sprintf("批量创建用户公司关联失败: %v", err))
		return errx.New(constant.InternalServerError, "批量创建用户公司关联失败", err)
	}

	return nil
}

// GetByUserID 获取用户的所有公司关联
func (r *UserCompanyRepository) GetByUserID(ctx context.Context, userID uint) ([]*entity.UserCompany, error) {
	if userID == 0 {
		return nil, errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var userCompanies []*entity.UserCompany

	err := r.Db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&userCompanies).Error

	if err != nil {
		logger.Error(fmt.Sprintf("查询用户公司关联失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return userCompanies, nil
}

// GetByUserIDWithCompany 获取用户的所有公司关联（包含公司信息）
func (r *UserCompanyRepository) GetByUserIDWithCompany(ctx context.Context, userID uint) ([]*entity.UserCompany, error) {
	if userID == 0 {
		return nil, errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var userCompanies []*entity.UserCompany

	err := r.Db.WithContext(ctx).
		Preload("Company").
		Where("user_id = ?", userID).
		Find(&userCompanies).Error

	if err != nil {
		logger.Error(fmt.Sprintf("查询用户公司关联失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return userCompanies, nil
}

// GetByCompanyID 获取公司的所有用户关联
func (r *UserCompanyRepository) GetByCompanyID(ctx context.Context, companyID uint) ([]*entity.UserCompany, error) {
	if companyID == 0 {
		return nil, errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var userCompanies []*entity.UserCompany

	err := r.Db.WithContext(ctx).
		Where("company_id = ?", companyID).
		Find(&userCompanies).Error

	if err != nil {
		logger.Error(fmt.Sprintf("查询公司用户关联失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return userCompanies, nil
}

// GetByUserAndCompany 获取特定用户和公司的关联
func (r *UserCompanyRepository) GetByUserAndCompany(ctx context.Context, userID uint, companyID uint) (*entity.UserCompany, error) {
	if userID == 0 {
		return nil, errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}

	if companyID == 0 {
		return nil, errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var userCompany entity.UserCompany

	err := r.Db.WithContext(ctx).
		Where("user_id = ? AND company_id = ?", userID, companyID).
		First(&userCompany).Error

	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, errx.New(constant.NotFound, "用户公司关联不存在", nil)
	case err != nil:
		logger.Error(fmt.Sprintf("查询用户公司关联失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return &userCompany, nil
}

// Update 更新用户公司关联（主要用于更新角色）
func (r *UserCompanyRepository) Update(ctx context.Context, userID uint, companyID uint, updates map[string]interface{}) error {
	if userID == 0 {
		return errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}

	if companyID == 0 {
		return errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	if len(updates) == 0 {
		return errx.New(constant.BadRequest, "更新字段不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Model(&entity.UserCompany{}).
		Where("user_id = ? AND company_id = ?", userID, companyID).
		Updates(updates)

	if result.Error != nil {
		logger.Error(fmt.Sprintf("更新用户公司关联失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "更新用户公司关联失败", result.Error)
	}

	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "用户公司关联不存在", nil)
	}

	return nil
}

// Delete 删除用户公司关联
func (r *UserCompanyRepository) Delete(ctx context.Context, userID uint, companyID uint) error {
	if userID == 0 {
		return errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}

	if companyID == 0 {
		return errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("user_id = ? AND company_id = ?", userID, companyID).
		Delete(&entity.UserCompany{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除用户公司关联失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "删除用户公司关联失败", result.Error)
	}

	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "用户公司关联不存在", nil)
	}

	return nil
}

// DeleteByUserID 删除用户的所有公司关联
func (r *UserCompanyRepository) DeleteByUserID(ctx context.Context, userID uint) error {
	if userID == 0 {
		return errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&entity.UserCompany{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除用户所有公司关联失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "删除失败", result.Error)
	}

	return nil
}

// DeleteByCompanyID 删除公司的所有用户关联
func (r *UserCompanyRepository) DeleteByCompanyID(ctx context.Context, companyID uint) error {
	if companyID == 0 {
		return errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("company_id = ?", companyID).
		Delete(&entity.UserCompany{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除公司所有用户关联失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "删除失败", result.Error)
	}

	return nil
}

// BatchDelete 批量删除用户公司关联
// pairs: 用户ID和公司ID的配对列表
func (r *UserCompanyRepository) BatchDelete(ctx context.Context, pairs []struct {
	UserID    uint
	CompanyID uint
}) error {
	if len(pairs) == 0 {
		return errx.New(constant.BadRequest, "删除列表不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)

	// 验证所有配对
	for _, pair := range pairs {
		if pair.UserID == 0 || pair.CompanyID == 0 {
			return errx.New(constant.BadRequest, "用户ID和公司ID不能为空", nil)
		}
	}

	// 构建批量删除条件（使用 OR 条件）
	db := r.Db.WithContext(ctx).Model(&entity.UserCompany{})
	for i, pair := range pairs {
		if i == 0 {
			db = db.Where("(user_id = ? AND company_id = ?)", pair.UserID, pair.CompanyID)
		} else {
			db = db.Or("(user_id = ? AND company_id = ?)", pair.UserID, pair.CompanyID)
		}
	}

	result := db.Delete(&entity.UserCompany{})
	if result.Error != nil {
		logger.Error(fmt.Sprintf("批量删除用户公司关联失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "批量删除失败", result.Error)
	}

	return nil
}

// Exists 检查用户公司关联是否存在
func (r *UserCompanyRepository) Exists(ctx context.Context, userID uint, companyID uint) (bool, error) {
	if userID == 0 {
		return false, errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}

	if companyID == 0 {
		return false, errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var count int64

	err := r.Db.WithContext(ctx).
		Model(&entity.UserCompany{}).
		Where("user_id = ? AND company_id = ?", userID, companyID).
		Count(&count).Error

	if err != nil {
		logger.Error(fmt.Sprintf("检查用户公司关联存在性失败: %v", err))
		return false, errx.New(constant.InternalServerError, "检查失败", err)
	}

	return count > 0, nil
}

// CountUsersInCompany counts how many users from a list are in a specific company.
func (r *UserCompanyRepository) CountUsersInCompany(ctx context.Context, userIDs []uint, companyID uint) (int64, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}
	var count int64
	err := r.Db.WithContext(ctx).
		Model(&entity.UserCompany{}).
		Where("company_id = ? AND user_id IN ?", companyID, userIDs).
		Count(&count).Error
	if err != nil {
		return 0, errx.New(constant.InternalServerError, "查询用户是否属于公司失败", err)
	}
	return count, nil
}

// ListCompanyUsers 分页查询某个企业的用户列表（含角色，可模糊搜索用户名、手机号、邮箱）
func (r *UserCompanyRepository) ListCompanyUsers(ctx context.Context, companyID uint, q *string, pageNumber, pageSize int) ([]*entity.UserCompany, int64, error) {
	if companyID == 0 {
		return nil, 0, errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var userCompanies []*entity.UserCompany
	var total int64

	// 构建基础查询
	db := r.Db.WithContext(ctx).
		Model(&entity.UserCompany{}).
		Preload("User").
		Where("company_id = ?", companyID)

	// 如果提供了查询条件，进行模糊搜索（同时匹配用户名、手机号、邮箱）
	if q != nil && *q != "" {
		db = db.Joins("JOIN users ON users.id = user_companies.user_id").
			Where("(users.username LIKE ? OR users.phone LIKE ? OR users.email LIKE ?) AND users.deleted_at IS NULL",
				"%"+*q+"%", "%"+*q+"%", "%"+*q+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计企业用户总数失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	// 排序和分页
	if q != nil && *q != "" {
		// 如果使用了 JOIN，需要指定表名
		db = db.Order("users.id DESC")
	} else {
		db = db.Order("user_id DESC")
	}

	if pageSize > 0 {
		offset := (pageNumber - 1) * pageSize
		db = db.Offset(offset).Limit(pageSize)
	}

	// 执行查询
	if err := db.Find(&userCompanies).Error; err != nil {
		logger.Error(fmt.Sprintf("查询企业用户列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return userCompanies, total, nil
}

// ListUsersNotInCompany 分页查询某个企业不存在的用户列表（可模糊搜索用户名、手机号、邮箱）
func (r *UserCompanyRepository) ListUsersNotInCompany(ctx context.Context, companyID uint, q *string, pageNumber, pageSize int) ([]*entity.User, int64, error) {
	if companyID == 0 {
		return nil, 0, errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var users []*entity.User
	var total int64

	// 构建基础查询：查询所有用户，排除已在该企业的用户
	db := r.Db.WithContext(ctx).
		Model(&entity.User{}).
		Where("users.deleted_at IS NULL").
		Where("users.id NOT IN (SELECT user_id FROM user_companies WHERE company_id = ?)", companyID)

	// 如果提供了查询条件，进行模糊搜索（同时匹配用户名、手机号、邮箱）
	if q != nil && *q != "" {
		db = db.Where("users.username LIKE ? OR users.phone LIKE ? OR users.email LIKE ?",
			"%"+*q+"%", "%"+*q+"%", "%"+*q+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计不存在企业用户总数失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	// 排序和分页
	db = db.Order("users.id DESC")

	if pageSize > 0 {
		offset := (pageNumber - 1) * pageSize
		db = db.Offset(offset).Limit(pageSize)
	}

	// 执行查询
	if err := db.Find(&users).Error; err != nil {
		logger.Error(fmt.Sprintf("查询不存在企业用户列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return users, total, nil
}