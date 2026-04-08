package repository

import (
	"context"
	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/errx"

	"gorm.io/gorm"
)

type UserSiteRepository struct {
	db *gorm.DB
}

func NewUserSiteRepository(db *gorm.DB) *UserSiteRepository {
	return &UserSiteRepository{
		db: db,
	}
}

func (r *UserSiteRepository) BatchCreate(ctx context.Context, userSites []*entity.UserSite) error {
	err := r.db.WithContext(ctx).Create(userSites).Error
	if err != nil {
		return errx.New(constant.InternalServerError, "批量添加园区用户失败", err)
	}
	return nil
}

func (r *UserSiteRepository) BatchDelete(ctx context.Context, userIDs []uint, siteID uint) error {
	err := r.db.WithContext(ctx).Where("site_id = ? AND user_id IN ?", siteID, userIDs).Delete(&entity.UserSite{}).Error
	if err != nil {
		return errx.New(constant.InternalServerError, "批量删除园区用户失败", err)
	}
	return nil
}

func (r *UserSiteRepository) ListUsersBySiteID(ctx context.Context, siteID uint, page, pageSize int) ([]*entity.User, int64, error) {
	var users []*entity.User
	var total int64

	// Subquery to get user IDs from the user_sites table
	subQuery := r.db.Model(&entity.UserSite{}).Select("user_id").Where("site_id = ?", siteID)

	// Count total users
	err := r.db.Model(&entity.User{}).Where("id IN (?)", subQuery).Count(&total).Error
	if err != nil {
		return nil, 0, errx.New(constant.InternalServerError, "查询园区用户总数失败", err)
	}

	// Get paginated users
	err = r.db.WithContext(ctx).Where("id IN (?)", subQuery).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error
	if err != nil {
		return nil, 0, errx.New(constant.InternalServerError, "查询园区用户列表失败", err)
	}

	return users, total, nil
}

func (r *UserSiteRepository) ListUsersNotInSite(ctx context.Context, siteID uint, companyID uint, page, pageSize int) ([]*entity.User, int64, error) {
	var users []*entity.User
	var total int64

	// Subquery to get user IDs from the current site
	subQuerySiteUsers := r.db.Model(&entity.UserSite{}).Select("user_id").Where("site_id = ?", siteID)

	// Subquery to get user IDs from the current company with the specified role
	subQueryCompanyUsers := r.db.Model(&entity.UserCompany{}).Select("user_id").
		Where("company_id = ? AND role <= ?", companyID, entity.UserRoleManager)

	// Users in the company but not in the site
	query := r.db.Model(&entity.User{}).
		Where("id IN (?)", subQueryCompanyUsers).
		Where("id NOT IN (?)", subQuerySiteUsers)

	// Count total matching users
	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, errx.New(constant.InternalServerError, "查询未分配园区用户总数失败", err)
	}

	// Get paginated users
	err = query.WithContext(ctx).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error
	if err != nil {
		return nil, 0, errx.New(constant.InternalServerError, "查询未分配园区用户列表失败", err)
	}

	return users, total, nil
}

func (r *UserSiteRepository) GetUserSiteIDs(ctx context.Context, userID uint, companyID uint) ([]uint, error) {
	var siteIDs []uint
	err := r.db.WithContext(ctx).
		Model(&entity.UserSite{}).
		Joins("JOIN sites ON sites.id = user_sites.site_id").
		Where("user_sites.user_id = ? AND sites.company_id = ?", userID, companyID).
		Pluck("site_id", &siteIDs).Error
	if err != nil {
		return nil, errx.New(constant.InternalServerError, "查询用户园区失败", err)
	}
	return siteIDs, nil
}
