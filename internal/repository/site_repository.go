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

type SiteRepository struct {
	Db *gorm.DB
}

func (r *SiteRepository) GetByField(ctx context.Context, field string, value interface{}) (*entity.Site, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var site entity.Site

	err := r.Db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ?", field), value).
		First(&site).Error

	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, errx.New(constant.NotFound, "场所不存在", nil)
	case err != nil:
		logger.Error(fmt.Sprintf("查询失败[%s]: %v", field, err))
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return &site, nil
}

func (r *SiteRepository) ExistsByField(ctx context.Context, field string, value interface{}) (bool, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var count int64

	err := r.Db.WithContext(ctx).
		Model(&entity.Site{}).
		Where(fmt.Sprintf("%s = ?", field), value).
		Count(&count).Error

	if err != nil {
		logger.Error(fmt.Sprintf("存在性检查失败[%s]: %v", field, err))
		return false, errx.New(constant.InternalServerError, "检查失败", err)
	}

	return count > 0, nil
}

func (r *SiteRepository) Create(ctx context.Context, site *entity.Site) error {
	logger := contextx.GetLoggerFromContext(ctx)
	if site == nil {
		return errx.New(constant.BadRequest, "参数不能为空", nil)
	}

	err := r.Db.WithContext(ctx).Create(site).Error
	if err != nil {
		logger.Error(fmt.Sprintf("创建失败: %v", err))
		return errx.New(constant.InternalServerError, "创建失败", err)
	}

	return nil
}

func (r *SiteRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Model(&entity.Site{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		logger.Error(fmt.Sprintf("更新失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "更新失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "场所不存在", nil)
	}

	return nil
}

func (r *SiteRepository) Delete(ctx context.Context, id uint) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		Delete(&entity.Site{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "删除失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "场所不存在", nil)
	}

	return nil
}

func (r *SiteRepository) BatchDelete(ctx context.Context, ids []uint) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("id IN (?) AND deleted_at IS NULL", ids).
		Delete(&entity.Site{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("批量删除失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "批量删除失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "无有效记录", nil)
	}

	return nil
}

func (r *SiteRepository) List(ctx context.Context, query entity.SiteQuery) ([]*entity.Site, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var sites []*entity.Site
	var total int64

	db := r.Db.WithContext(ctx).Model(&entity.Site{}).Where("deleted_at IS NULL")

	// 处理Q字段（模糊搜索名称或编码）
	if query.Q != nil && *query.Q != "" {
		q := "%" + *query.Q + "%"
		db = db.Where("name LIKE ? OR code LIKE ?", q, q)
	} else {
		// 保持原有逻辑
		if query.Name != nil && *query.Name != "" {
			db = db.Where("name LIKE ?", "%"+*query.Name+"%")
		}
		if query.Code != nil {
			db = db.Where("code = ?", *query.Code)
		}
	}

	// 其他条件
	if query.Type != nil {
		db = db.Where("type = ?", *query.Type)
	}
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}
	if query.Province != nil {
		db = db.Where("province = ?", *query.Province)
	}
	if query.City != nil {
		db = db.Where("city = ?", *query.City)
	}
	if query.District != nil {
		db = db.Where("district = ?", *query.District)
	}
	if query.CompanyID != nil {
		db = db.Where("company_id = ?", *query.CompanyID)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计总数失败: %v", err))
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
	if err := db.Find(&sites).Error; err != nil {
		logger.Error(fmt.Sprintf("查询列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return sites, total, nil
}

func (r *SiteRepository) GetByID(ctx context.Context, id uint) (*entity.Site, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var site entity.Site

	err := r.Db.WithContext(ctx).Where("id = ?", id).First(&site).Error

	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, errx.New(constant.NotFound, "场所不存在", nil)
	case err != nil:
		logger.Error(fmt.Sprintf("查询失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return &site, nil
}

// AssignDevices 分配设备到园区（不报错如果已存在）
func (r *SiteRepository) AssignDevices(ctx context.Context, siteID uint, deviceIDs []uint) error {
	logger := contextx.GetLoggerFromContext(ctx)

	if len(deviceIDs) == 0 {
		return nil
	}

	// 开启事务
	tx := r.Db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查已存在的关联
	var existingDevices []entity.SiteDevice
	if err := tx.Where("site_id = ? AND device_id IN (?)", siteID, deviceIDs).Find(&existingDevices).Error; err != nil {
		tx.Rollback()
		logger.Error(fmt.Sprintf("查询现有设备关联失败: %v", err))
		return errx.New(constant.InternalServerError, "分配设备失败", err)
	}

	// 创建不存在的设备关联
	existingDeviceMap := make(map[uint]bool)
	for _, device := range existingDevices {
		existingDeviceMap[device.DeviceID] = true
	}

	var newDevices []entity.SiteDevice
	for _, deviceID := range deviceIDs {
		if !existingDeviceMap[deviceID] {
			newDevices = append(newDevices, entity.SiteDevice{
				SiteID:   siteID,
				DeviceID: deviceID,
			})
		}
	}

	if len(newDevices) > 0 {
		if err := tx.Create(&newDevices).Error; err != nil {
			tx.Rollback()
			logger.Error(fmt.Sprintf("创建新设备关联失败: %v", err))
			return errx.New(constant.InternalServerError, "分配设备失败", err)
		}
	}

	return tx.Commit().Error
}

// RemoveDevices 从园区删除设备（不报错如果不存在）
func (r *SiteRepository) RemoveDevices(ctx context.Context, siteID uint, deviceIDs []uint) error {
	logger := contextx.GetLoggerFromContext(ctx)

	if len(deviceIDs) == 0 {
		return nil
	}

	result := r.Db.WithContext(ctx).
		Where("site_id = ? AND device_id IN (?)", siteID, deviceIDs).
		Delete(&entity.SiteDevice{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除设备关联失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "删除设备失败", result.Error)
	}

	return nil
}
