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

type DeviceSimulatorRepository struct {
	Db *gorm.DB
}

func NewDeviceSimulatorRepository(db *gorm.DB) *DeviceSimulatorRepository {
	return &DeviceSimulatorRepository{
		Db: db,
	}
}

// GetByID 通过ID查询
func (r *DeviceSimulatorRepository) GetByID(ctx context.Context, id uint) (*entity.DeviceSimulator, error) {
	var simulator entity.DeviceSimulator
	if err := r.Db.WithContext(ctx).First(&simulator, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errx.New(constant.NotFound, "记录不存在", nil)
		}
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}
	return &simulator, nil
}

// ExistsByDeviceSN 检查设备SN号是否存在
func (r *DeviceSimulatorRepository) ExistsByDeviceSN(ctx context.Context, deviceSN string) (bool, error) {
	var count int64
	if err := r.Db.WithContext(ctx).
		Model(&entity.DeviceSimulator{}).
		Where("device_sn = ?", deviceSN).
		Count(&count).Error; err != nil {
		return false, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return count > 0, nil
}

// Create 创建记录
func (r *DeviceSimulatorRepository) Create(ctx context.Context, simulator *entity.DeviceSimulator) error {
	logger := contextx.GetLoggerFromContext(ctx)

	// 基本验证
	if simulator.DeviceSN == "" {
		return errx.New(constant.BadRequest, "设备SN号不能为空", nil)
	}

	// 创建记录
	if err := r.Db.WithContext(ctx).Create(simulator).Error; err != nil {
		logger.Error(fmt.Sprintf("创建记录失败: %v", err))
		return errx.New(constant.InternalServerError, "创建失败", err)
	}

	return nil
}

// List 分页查询
func (r *DeviceSimulatorRepository) List(ctx context.Context, query entity.DeviceSimulatorQuery) ([]*entity.DeviceSimulator, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var simulators []*entity.DeviceSimulator
	var total int64

	db := r.Db.WithContext(ctx).Model(&entity.DeviceSimulator{})

	// 综合查询条件
	if query.Q != nil && *query.Q != "" {
		keywords := *query.Q
		db = db.Where("device_sn LIKE ?", "%"+keywords+"%")
	}

	// 设备SN号模糊匹配
	if query.DeviceSN != nil && *query.DeviceSN != "" {
		db = db.Where("device_sn LIKE ?", "%"+*query.DeviceSN+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计总数失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	// 排序（按id降序，最新的在前）
	db = db.Order("id DESC")

	// 分页
	if query.PageSize > 0 {
		offset := (query.PageNumber - 1) * query.PageSize
		db = db.Offset(offset).Limit(query.PageSize)
	}

	// 执行查询
	if err := db.Find(&simulators).Error; err != nil {
		logger.Error(fmt.Sprintf("查询列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return simulators, total, nil
}

// BatchDelete 批量彻底删除（物理删除）
func (r *DeviceSimulatorRepository) BatchDelete(ctx context.Context, ids []uint) error {
	logger := contextx.GetLoggerFromContext(ctx)

	if len(ids) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}

	// 批量彻底删除数据库记录
	result := r.Db.WithContext(ctx).
		Unscoped().
		Where("id IN (?)", ids).
		Delete(&entity.DeviceSimulator{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("批量删除失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "批量删除失败", result.Error)
	}

	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "无有效记录", nil)
	}

	return nil
}
