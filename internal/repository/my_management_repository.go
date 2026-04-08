package repository

import (
	"context"
	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"strconv"

	"gorm.io/gorm"
	"fmt"
)

type MyManagementRepository struct {
	Db *gorm.DB
}

// ListMySites 获取用户管理的园区列表（不包含设备）
func (r *MyManagementRepository) ListMySites(ctx context.Context, filter *entity.SiteQuery) ([]*entity.Site, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var sites []*entity.Site
	var total int64

	db := r.Db.WithContext(ctx).
		Model(&entity.Site{}).
		Where("sites.deleted_at IS NULL").
		Where("sites.status = ?", entity.SiteStatusActive)

	// 如果 filter 不为 nil 且包含企业ID，则添加企业ID过滤
	if filter != nil && filter.CompanyID != nil && *filter.CompanyID > 0 {
		db = db.Where("sites.company_id = ?", *filter.CompanyID)
	}

	// 如果 filter 不为 nil 且包含园区ID列表，则添加园区ID过滤
	if filter != nil && filter.IDs != nil && len(*filter.IDs) > 0 {
		db = db.Where("sites.id IN (?)", *filter.IDs)
	}

	// 添加园区名称模糊搜索
	if filter != nil {
		if filter.Q != nil && *filter.Q != "" {
			db = db.Where("sites.name LIKE ?", "%"+*filter.Q+"%")
		}

		// 精确匹配园区编码
		if filter.Code != nil && *filter.Code != "" {
			db = db.Where("sites.code = ?", *filter.Code)
		}
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计园区总数失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	// 排序（使用 id DESC 确保排序稳定）
	db = db.Order("sites.id DESC")

	// 分页
	if filter != nil && filter.PageSize > 0 {
		offset := (filter.PageNumber - 1) * filter.PageSize
		db = db.Offset(offset).Limit(filter.PageSize)
	}

	// 执行查询
	if err := db.Find(&sites).Error; err != nil {
		logger.Error(fmt.Sprintf("查询园区列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return sites, total, nil
}

// ListMyDevices 获取用户管理的设备列表
func (r *MyManagementRepository) ListMyDevices(ctx context.Context, filter *entity.DeviceQuery) ([]*entity.Device, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var devices []*entity.Device
	var total int64

	// 检查是否有可查询的园区编码
	hasSiteCodes := filter != nil && len(filter.SiteCodes) > 0

	db := r.Db.WithContext(ctx).
		Model(&entity.Device{}).
		Joins("JOIN site_devices ON site_devices.device_id = devices.id").
		Joins("JOIN sites ON sites.id = site_devices.site_id").
		Where("sites.deleted_at IS NULL").
		Where("sites.status = ?", entity.SiteStatusActive).
		Where("devices.deleted_at IS NULL").
		Where("devices.is_active = ?", true)

	// 如果 filter 不为 nil 且包含企业ID，则添加企业ID过滤
	if filter != nil && filter.CompanyID != nil && *filter.CompanyID > 0 {
		db = db.Where("sites.company_id = ?", *filter.CompanyID)
	}

	// 添加园区编码过滤条件
	if filter != nil {
		if hasSiteCodes {
			db = db.Where("sites.code IN (?)", filter.SiteCodes)
		}
		// 如果没有指定 SiteCodes，则查询当前企业下所有园区的设备

		if len(filter.DeviceTypes) > 0 {
			db = db.Where("devices.device_type IN (?)", filter.DeviceTypes)
		}
		if filter.DeviceName != nil && *filter.DeviceName != "" {
			db = db.Where("devices.name LIKE ?", "%"+*filter.DeviceName+"%")
		}

		// 综合查询条件，用于DeviceNum、Name
		if filter.Q != nil && *filter.Q != "" {
			q := "%" + *filter.Q + "%"
			if id, err := strconv.Atoi(*filter.Q); err == nil {
				// 数字+文本混合搜索
				db = db.Where(
					"devices.device_num = ? OR devices.name LIKE ?",
					id, q,
				)
			} else {
				// 仅文本字段搜索
				db = db.Where("devices.name LIKE ?", q)
			}
		}
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计设备总数失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	// 排序（使用 id DESC 确保排序稳定）
	db = db.Order("devices.id DESC")

	// 分页
	if filter != nil && filter.PageSize > 0 {
		offset := (filter.PageNumber - 1) * filter.PageSize
		db = db.Offset(offset).Limit(filter.PageSize)
	}

	// 执行查询
	if err := db.Find(&devices).Error; err != nil {
		logger.Error(fmt.Sprintf("查询设备列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return devices, total, nil
}

// GetMyDeviceTypeStats 获取我的设备类型统计
func (r *MyManagementRepository) GetMyDeviceTypeStats(ctx context.Context, filter *entity.DeviceQuery) ([]*entity.DeviceTypeStats, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var stats []*entity.DeviceTypeStats

	db := r.Db.WithContext(ctx).
		Model(&entity.Device{}).
		Select("devices.device_type, COUNT(*) as count").
		Joins("JOIN site_devices ON site_devices.device_id = devices.id").
		Joins("JOIN sites ON sites.id = site_devices.site_id").
		Where("sites.deleted_at IS NULL").
		Where("sites.status = ?", entity.SiteStatusActive).
		Where("devices.deleted_at IS NULL").
		Where("devices.is_active = ?", true).
		Group("devices.device_type").
		Order("count DESC")

	// 如果 filter 不为 nil 且包含企业ID，则添加企业ID过滤
	if filter != nil && filter.CompanyID != nil && *filter.CompanyID > 0 {
		db = db.Where("sites.company_id = ?", *filter.CompanyID)
	}

	// 添加园区编码过滤条件（如果指定了园区编码）
	if filter != nil && len(filter.SiteCodes) > 0 {
		db = db.Where("sites.code IN (?)", filter.SiteCodes)
	}

	// 执行查询
	if err := db.Find(&stats).Error; err != nil {
		logger.Error(fmt.Sprintf("查询设备类型统计失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return stats, nil
}
