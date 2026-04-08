package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"

	"gorm.io/gorm"
)

type UserRepository struct {
	Db *gorm.DB
}

// GetByField 通用字段查询方法
func (r *UserRepository) GetByField(ctx context.Context, field string, value interface{}) (*entity.User, error) {
	if field == "" || value == nil {
		return nil, errx.New(constant.BadRequest, "查询字段和值不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var user entity.User

	err := r.Db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ? AND deleted_at IS NULL", field), value).
		First(&user).Error

	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, errx.New(constant.NotFound, "用户不存在", nil)
	case err != nil:
		logger.Error(fmt.Sprintf("查询用户失败[%s]: %v", field, err))
		return nil, errx.New(constant.InternalServerError, "查询用户失败", err)
	}

	return &user, nil
}

// GetByID 通过ID查询用户
func (r *UserRepository) GetByID(ctx context.Context, id uint) (*entity.User, error) {
	if id == 0 {
		return nil, errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}

	return r.GetByField(ctx, "id", id)
}

// GetByUsernameOrEmailOrPhone 通过用户名、邮箱或手机号查询用户（用于登录）
func (r *UserRepository) GetByUsernameOrEmailOrPhone(ctx context.Context, identifier string) (*entity.User, error) {
	if strings.TrimSpace(identifier) == "" {
		return nil, errx.New(constant.BadRequest, "登录标识不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var user entity.User

	err := r.Db.WithContext(ctx).
		Where("(username = ? OR email = ? OR phone = ?) AND deleted_at IS NULL", identifier, identifier, identifier).
		First(&user).Error

	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, errx.New(constant.NotFound, "用户不存在", nil)
	case err != nil:
		logger.Error(fmt.Sprintf("查询用户失败[identifier=%s]: %v", identifier, err))
		return nil, errx.New(constant.InternalServerError, "查询用户失败", err)
	}

	return &user, nil
}

// GetByIDs 批量通过ID查询用户
func (r *UserRepository) GetByIDs(ctx context.Context, ids []uint) ([]*entity.User, error) {
	if len(ids) == 0 {
		return []*entity.User{}, nil
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var users []*entity.User

	err := r.Db.WithContext(ctx).
		Where("id IN (?) AND deleted_at IS NULL", ids).
		Find(&users).Error

	if err != nil {
		logger.Error(fmt.Sprintf("批量查询用户失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "查询用户失败", err)
	}

	return users, nil
}

// ExistsByField 通用存在性检查
func (r *UserRepository) ExistsByField(ctx context.Context, field string, value interface{}) (bool, error) {
	if field == "" || value == nil {
		return false, errx.New(constant.BadRequest, "查询字段和值不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	var count int64

	err := r.Db.WithContext(ctx).
		Model(&entity.User{}).
		Where(fmt.Sprintf("%s = ? AND deleted_at IS NULL", field), value).
		Count(&count).Error

	if err != nil {
		logger.Error(fmt.Sprintf("用户存在性检查失败[%s]: %v", field, err))
		return false, errx.New(constant.InternalServerError, "检查失败", err)
	}

	return count > 0, nil
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	if user == nil {
		return errx.New(constant.BadRequest, "用户参数不能为空", nil)
	}

	// 验证必填字段
	if err := r.validateUser(user); err != nil {
		return err
	}

	logger := contextx.GetLoggerFromContext(ctx)
	err := r.Db.WithContext(ctx).Create(user).Error

	if err != nil {
		logger.Error(fmt.Sprintf("创建用户失败: %v", err))
		return errx.New(constant.InternalServerError, "创建用户失败", err)
	}

	return nil
}

// validateUser 验证用户实体
func (r *UserRepository) validateUser(user *entity.User) error {
	if strings.TrimSpace(user.Username) == "" {
		return errx.New(constant.BadRequest, "用户名不能为空", nil)
	}
	if len(user.Username) < 3 || len(user.Username) > 50 {
		return errx.New(constant.BadRequest, "用户名长度必须在3-50个字符之间", nil)
	}
	if strings.TrimSpace(user.Password) == "" {
		return errx.New(constant.BadRequest, "密码不能为空", nil)
	}
	return nil
}

// Update 更新用户
func (r *UserRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	if id == 0 {
		return errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}
	if len(updates) == 0 {
		return errx.New(constant.BadRequest, "更新字段不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Model(&entity.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates)

	if result.Error != nil {
		logger.Error(fmt.Sprintf("更新用户失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "更新用户失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "用户不存在", nil)
	}

	return nil
}

// Delete 删除用户
func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		Delete(&entity.User{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除用户失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "删除用户失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "用户不存在", nil)
	}

	return nil
}

// BatchDelete 批量删除用户
func (r *UserRepository) BatchDelete(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return errx.New(constant.BadRequest, "删除ID列表不能为空", nil)
	}

	// 验证ID列表
	for _, id := range ids {
		if id == 0 {
			return errx.New(constant.BadRequest, "用户ID不能为空", nil)
		}
	}

	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("id IN (?) AND deleted_at IS NULL", ids).
		Delete(&entity.User{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("批量删除用户失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "批量删除用户失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "无有效用户", nil)
	}

	return nil
}

// List 分页查询用户列表
func (r *UserRepository) List(ctx context.Context, query entity.UserQuery) ([]*entity.User, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var users []*entity.User
	var total int64

	// 构建基础查询
	db := r.Db.WithContext(ctx).Model(&entity.User{}).Where("users.deleted_at IS NULL")

	// 如果指定了企业ID，需要通过 UserCompany 表过滤
	if query.CompanyID != nil && *query.CompanyID > 0 {
		db = db.Joins("JOIN user_companies ON user_companies.user_id = users.id").
			Where("user_companies.company_id = ?", *query.CompanyID)
	}

	// 应用查询条件
	if query.Q != nil && *query.Q != "" {
		q := *query.Q
		db = db.Where("users.username LIKE ? OR users.nickname LIKE ? OR users.email = ? OR users.phone = ?",
			"%"+q+"%", "%"+q+"%", q, q)
	} else {
		// 单独字段查询条件
		if query.Username != nil {
			db = db.Where("users.username LIKE ?", "%"+*query.Username+"%")
		}
		if query.Nickname != nil {
			db = db.Where("users.nickname LIKE ?", "%"+*query.Nickname+"%")
		}
		if query.Email != nil {
			db = db.Where("users.email = ?", *query.Email)
		}
		if query.Phone != nil {
			db = db.Where("users.phone = ?", *query.Phone)
		}
	}

	// 时间范围
	if query.LastLoginAt != nil {
		db = db.Where("users.last_login_at >= ?", *query.LastLoginAt)
	}

	// 获取总数（使用相同的查询条件）
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计用户总数失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	// 排序（使用 id DESC 确保排序稳定）
	db = db.Order("users.id DESC")

	// 分页
	if query.PageSize > 0 {
		offset := (query.PageNumber - 1) * query.PageSize
		db = db.Offset(offset).Limit(query.PageSize)
	}

	// 执行查询
	if err := db.Find(&users).Error; err != nil {
		logger.Error(fmt.Sprintf("查询用户列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return users, total, nil
}

// UpdateLastLogin 更新用户最后登录信息
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id uint, loginAt time.Time, loginIP string) error {
	if id == 0 {
		return errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}

	logger := contextx.GetLoggerFromContext(ctx)
	updates := map[string]interface{}{
		"last_login_at": loginAt,
		"last_login_ip": loginIP,
	}

	result := r.Db.WithContext(ctx).Model(&entity.User{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(updates)

	if result.Error != nil {
		logger.Error(fmt.Sprintf("更新用户最后登录信息失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "更新登录信息失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "用户不存在", nil)
	}
	return nil
}
