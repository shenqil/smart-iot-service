package repository

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"rbac-admin/pkg/pubsubRedis"

	"gorm.io/gorm"
)

type DeviceRepository struct {
	Db             *gorm.DB
	onlineStatus   map[uint32]bool // 设备在线状态缓存
	statusMutex    sync.RWMutex    // 保护在线状态缓存的读写锁
	eventCollector *pubsubRedis.EventCollector
}

// NewDeviceRepository 创建新的设备仓库实例
func NewDeviceRepository(db *gorm.DB, eventCollector *pubsubRedis.EventCollector) *DeviceRepository {
	repo := &DeviceRepository{
		Db:             db,
		onlineStatus:   make(map[uint32]bool),
		eventCollector: eventCollector,
	}

	// 订阅通用设备告警事件，利用 device_online/device_offline 维护在线缓存
	if eventCollector != nil {
		eventCollector.SubscribeDeviceAlarm(func(payload pubsubRedis.DeviceAlarmPayload) {
			switch payload.AlarmType {
			case entity.AlarmTypeDeviceOnline:
				if id, err := strconv.ParseUint(payload.DeviceId, 10, 32); err == nil {
					repo.updateOnlineStatus(uint32(id), true)
				}
			case entity.AlarmTypeDeviceOffline:
				if id, err := strconv.ParseUint(payload.DeviceId, 10, 32); err == nil {
					repo.updateOnlineStatus(uint32(id), false)
				}
			}
		})
	}

	return repo
}

// updateOnlineStatus 更新设备在线状态缓存
func (r *DeviceRepository) updateOnlineStatus(deviceID uint32, online bool) {
	r.statusMutex.Lock()
	defer r.statusMutex.Unlock()
	r.onlineStatus[deviceID] = online
}

func (r *DeviceRepository) GetByField(ctx context.Context, field string, value interface{}) (*entity.Device, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var device entity.Device

	err := r.Db.WithContext(ctx).
		Where(fmt.Sprintf("%s = ?", field), value).
		First(&device).Error

	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, errx.New(constant.NotFound, "设备不存在", nil)
	case err != nil:
		logger.Error(fmt.Sprintf("查询设备失败[%s]: %v", field, err))
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return &device, nil
}

func (r *DeviceRepository) ExistsByField(ctx context.Context, field string, value interface{}) (bool, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var count int64

	err := r.Db.WithContext(ctx).
		Model(&entity.Device{}).
		Where(fmt.Sprintf("%s = ?", field), value).
		Count(&count).Error

	if err != nil {
		logger.Error(fmt.Sprintf("设备存在性检查失败[%s]: %v", field, err))
		return false, errx.New(constant.InternalServerError, "检查失败", err)
	}

	return count > 0, nil
}

func (r *DeviceRepository) Create(ctx context.Context, device *entity.Device) error {
	logger := contextx.GetLoggerFromContext(ctx)
	if device == nil {
		return errx.New(constant.BadRequest, "参数不能为空", nil)
	}

	err := r.Db.WithContext(ctx).Create(device).Error
	if err != nil {
		logger.Error(fmt.Sprintf("创建设备失败: %v", err))
		return errx.New(constant.InternalServerError, "创建失败", err)
	}

	return nil
}

func (r *DeviceRepository) Update(ctx context.Context, id uint, updates map[string]interface{}) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Model(&entity.Device{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		logger.Error(fmt.Sprintf("更新设备失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "更新失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "设备不存在", nil)
	}

	return nil
}

func (r *DeviceRepository) Delete(ctx context.Context, id uint) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		Delete(&entity.Device{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("删除设备失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "删除失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "设备不存在", nil)
	}

	return nil
}

func (r *DeviceRepository) BatchDelete(ctx context.Context, ids []uint) error {
	logger := contextx.GetLoggerFromContext(ctx)
	result := r.Db.WithContext(ctx).
		Where("id IN (?) AND deleted_at IS NULL", ids).
		Delete(&entity.Device{})

	if result.Error != nil {
		logger.Error(fmt.Sprintf("批量删除设备失败: %v", result.Error))
		return errx.New(constant.InternalServerError, "批量删除失败", result.Error)
	}
	if result.RowsAffected == 0 {
		return errx.New(constant.NotFound, "无有效设备", nil)
	}

	return nil
}

func (r *DeviceRepository) List(ctx context.Context, query entity.DeviceQuery) ([]*entity.Device, int64, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var devices []*entity.Device
	var total int64

	db := r.Db.WithContext(ctx).Model(&entity.Device{}).Where("devices.deleted_at IS NULL")

	// 处理站点ID过滤
	if query.SiteID != nil {
		db = db.Joins("JOIN site_devices ON site_devices.device_id = devices.id").
			Where("site_devices.site_id = ?", *query.SiteID)
	}

	// 处理排除站点ID过滤 - 查询不在指定站点下的设备
	if query.ExcludeSiteID != nil {
		db = db.Where("devices.id NOT IN (SELECT device_id FROM site_devices WHERE site_id = ?)", *query.ExcludeSiteID)
	}

	// 处理园区编码过滤
	if len(query.SiteCodes) > 0 {
		db = db.Joins("JOIN site_devices ON site_devices.device_id = devices.id").
			Joins("JOIN sites ON sites.id = site_devices.site_id").
			Where("sites.code IN (?)", query.SiteCodes)
	}

	if query.Q != nil && *query.Q != "" {
		q := "%" + *query.Q + "%"
		if id, err := strconv.Atoi(*query.Q); err == nil {
			// 数字+文本混合搜索
			db = db.Where(
				"device_num = ? OR name LIKE ? OR sim_card LIKE ?",
				id, q, q,
			)
		} else {
			// 仅文本字段搜索
			db = db.Where("name LIKE ? OR sim_card LIKE ?", q, q)
		}
	}

	// 其他精确匹配条件
	if query.MACAddress != nil {
		db = db.Where("mac_address = ?", *query.MACAddress)
	}
	if query.ReadInterval != nil {
		db = db.Where("read_interval = ?", *query.ReadInterval)
	}
	if query.CreatorID != nil {
		db = db.Where("creator_id = ?", *query.CreatorID)
	}

	// 枚举类型条件
	if len(query.DeviceTypes) > 0 {
		db = db.Where("device_type IN (?)", query.DeviceTypes)
	}
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}
	if query.IsActive != nil {
		db = db.Where("is_active = ?", *query.IsActive)
	}
	if query.CompanyID != nil {
		db = db.Where("company_id = ?", *query.CompanyID)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		logger.Error(fmt.Sprintf("统计设备总数失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	// 排序（使用 id DESC 确保排序稳定）
	db = db.Order("devices.id DESC")

	// 分页
	if query.PageSize > 0 {
		offset := (query.PageNumber - 1) * query.PageSize
		db = db.Offset(offset).Limit(query.PageSize)
	}

	// 执行查询
	if err := db.Find(&devices).Error; err != nil {
		logger.Error(fmt.Sprintf("查询设备列表失败: %v", err))
		return nil, 0, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return devices, total, nil
}

func (r *DeviceRepository) GetByID(ctx context.Context, id uint) (*entity.Device, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var device entity.Device

	err := r.Db.WithContext(ctx).Where("id = ?", id).First(&device).Error

	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, errx.New(constant.NotFound, "设备不存在", nil)
	case err != nil:
		logger.Error(fmt.Sprintf("查询失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return &device, nil
}

func (r *DeviceRepository) ExistsByID(ctx context.Context, id uint) (bool, error) {
	return r.ExistsByField(ctx, "id", id)
}

// GetByDeviceNumFromDB 从数据库获取设备（仅数据库操作，不涉及缓存）
func (r *DeviceRepository) GetByDeviceNumFromDB(ctx context.Context, deviceNum uint) (*entity.Device, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var device entity.Device

	err := r.Db.WithContext(ctx).
		Where("device_num = ?", deviceNum).
		Where("deleted_at IS NULL").
		First(&device).Error

	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, errx.New(constant.NotFound, "设备不存在", nil)
	case err != nil:
		logger.Error(fmt.Sprintf("查询设备失败[device_num]: %v", err))
		return nil, errx.New(constant.InternalServerError, "查询失败", err)
	}

	return &device, nil
}

func (r *DeviceRepository) ExistsByDeviceNum(ctx context.Context, deviceNum uint) (bool, error) {
	return r.ExistsByField(ctx, "device_num", deviceNum)
}

// BatchCheckOnlineStatus 批量检查设备在线状态
func (r *DeviceRepository) BatchCheckOnlineStatus(deviceIDs []uint32) map[uint32]bool {
	r.statusMutex.RLock()
	defer r.statusMutex.RUnlock()

	result := make(map[uint32]bool)
	for _, deviceID := range deviceIDs {
		if online, exists := r.onlineStatus[deviceID]; exists {
			result[deviceID] = online
		} else {
			// 如果缓存中没有该设备的状态，默认为离线
			result[deviceID] = false
		}
	}

	return result
}

// GetOnlineStatus 获取单个设备在线状态
func (r *DeviceRepository) GetOnlineStatus(deviceID uint32) bool {
	r.statusMutex.RLock()
	defer r.statusMutex.RUnlock()

	if online, exists := r.onlineStatus[deviceID]; exists {
		return online
	}
	return false
}

// GetDeviceNumsBySiteCode 根据园区编码获取设备编号列表
func (r *DeviceRepository) GetDeviceNumsBySiteCode(ctx context.Context, siteCode string) ([]uint, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var deviceNums []uint

	err := r.Db.WithContext(ctx).
		Table("devices").
		Select("devices.device_num").
		Joins("JOIN site_devices ON site_devices.device_id = devices.id").
		Joins("JOIN sites ON sites.id = site_devices.site_id").
		Where("sites.code = ?", siteCode).
		Where("sites.deleted_at IS NULL").
		Where("devices.deleted_at IS NULL").
		Pluck("devices.device_num", &deviceNums).Error

	if err != nil {
		logger.Error(fmt.Sprintf("根据园区编码查询设备编号列表失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "查询园区设备失败", err)
	}

	return deviceNums, nil
}
