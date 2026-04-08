package service

import (
	"context"

	"rbac-admin/internal/cache"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/model/entity"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/errx"
	"rbac-admin/pkg/pubsubRedis"
)

// ============================================================================
// DeviceService - 设备服务层，处理设备相关的业务逻辑
// ============================================================================

type DeviceService struct {
	DeviceRepo        *repository.DeviceRepository
	UserRepo          *repository.UserRepository
	EventCollector    *pubsubRedis.EventCollector
	DeviceConfigCache *cache.DeviceConfigCache
}

// ============================================================================
// 基础CRUD方法 - 处理设备的创建、查询、更新、删除等基本操作
// ============================================================================

// Create 创建设备 - 创建新的设备记录
func (s *DeviceService) Create(ctx context.Context, req dto.DeviceCreateRequest) (*dto.DeviceResponse, error) {
	// 检查设备编号是否已存在
	if exists, err := s.DeviceRepo.ExistsByDeviceNum(ctx, req.DeviceNum); err != nil {
		return nil, err
	} else if exists {
		return nil, errx.New(constant.Conflict, "设备编号已存在", nil)
	}

	// 检查MAC地址是否已存在
	if exists, err := s.DeviceRepo.ExistsByField(ctx, "mac_address", req.MACAddress); err != nil {
		return nil, err
	} else if exists {
		return nil, errx.New(constant.Conflict, "MAC地址已存在", nil)
	}

	// DTO转换（CompanyID 和 CreatorID 已在 ConvertToEntity 中验证）
	entity, err := req.ConvertToEntity()
	if err != nil {
		return nil, err
	}

	// 创建记录
	if err := s.DeviceRepo.Create(ctx, entity); err != nil {
		return nil, err
	}
	if s.DeviceConfigCache != nil {
		s.DeviceConfigCache.DelConfig(ctx, uint32(req.DeviceNum))
	}

	return dto.ConvertToDeviceResponse(entity, func(deviceNum uint) string {
		return s.getDeviceStatus(deviceNum)
	})
}

// Get 获取设备详情 - 根据ID获取单个设备的信息
func (s *DeviceService) Get(ctx context.Context, id uint) (*dto.DeviceResponse, error) {
	entity, err := s.DeviceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.ConvertToDeviceResponse(entity, func(deviceNum uint) string {
		return s.getDeviceStatus(deviceNum)
	})
}

// Update 更新设备 - 更新设备的所有可修改字段
func (s *DeviceService) Update(ctx context.Context, id uint, req dto.DeviceUpdateRequest) error {
	// 获取当前设备信息
	current, err := s.DeviceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查权限
	if err := s.checkCompanyPermission(current, req.CompanyID); err != nil {
		return err
	}

	updates := make(map[string]interface{})

	// 处理MAC地址
	if req.MACAddress != nil && *req.MACAddress != *current.MACAddress {
		if exists, err := s.DeviceRepo.ExistsByField(ctx, "mac_address", *req.MACAddress); err != nil {
			return err
		} else if exists {
			return errx.New(constant.Conflict, "MAC地址已存在", nil)
		}
		updates["mac_address"] = *req.MACAddress
	}

	// 处理地理坐标
	if req.Location != nil {
		if err := req.Location.Validate(); err != nil {
			return err
		}

		locationJSON, err := dto.ToJSON(*req.Location)
		if err != nil {
			return errx.New(constant.BadRequest, "经纬度配置格式错误", err)
		}
		updates["location"] = locationJSON
	}

	// 处理JSON字段（传感器配置）
	if req.Sensors != nil {
		sensorsJSON, err := dto.ToJSON(*req.Sensors)
		if err != nil {
			return errx.New(constant.BadRequest, "传感器配置格式错误", err)
		}
		updates["sensors"] = sensorsJSON
	}

	// 处理JSON字段（继电器配置）
	if req.Relays != nil {
		relaysJSON, err := dto.ToJSON(*req.Relays)
		if err != nil {
			return errx.New(constant.BadRequest, "继电器配置格式错误", err)
		}
		updates["relays"] = relaysJSON
	}

	// 处理摄像头配置
	if req.CameraConfig != nil {
		cameraJSON, err := dto.ToJSON(*req.CameraConfig)
		if err != nil {
			return errx.New(constant.BadRequest, "摄像头配置格式错误", err)
		}
		updates["camera_config"] = cameraJSON
	}

	// 处理水肥信息配置
	if req.WaterFertilizerInfo != nil {
		waterFertilizerJSON, err := dto.ToJSON(*req.WaterFertilizerInfo)
		if err != nil {
			return errx.New(constant.BadRequest, "水肥信息格式错误", err)
		}
		updates["water_fertilizer_info"] = waterFertilizerJSON
	}

	// 处理枚举状态
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	// 处理普通字段
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.DeviceType != nil {
		updates["device_type"] = *req.DeviceType
	}
	if req.ReadInterval != nil {
		updates["read_interval"] = *req.ReadInterval
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.SIMCard != nil {
		updates["sim_card"] = *req.SIMCard
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.CoverImage != nil {
		updates["cover_image"] = *req.CoverImage
	}

	// 无有效更新时返回
	if len(updates) == 0 {
		return errx.New(constant.BadRequest, "未提供有效更新字段", nil)
	}

	err = s.DeviceRepo.Update(ctx, id, updates)
	if err != nil {
		return err
	}

	// 处理配置更新后的清理工作
	s.handleConfigUpdate(ctx, current.DeviceNum)

	return nil
}

func (s *DeviceService) Delete(ctx context.Context, id uint, companyID uint) error {
	// 获取当前设备信息
	current, err := s.DeviceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查权限
	if err := s.checkCompanyPermission(current, companyID); err != nil {
		return err
	}

	err = s.DeviceRepo.Delete(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

// ============================================================================
// 批量操作方法 - 处理设备的批量查询和批量删除操作
// ============================================================================

// List 分页查询设备列表 - 根据过滤条件分页获取设备列表
func (s *DeviceService) List(ctx context.Context, filter dto.DeviceFilter) ([]*dto.DeviceResponse, int64, error) {
	query := filter.ConvertToQuery()

	entities, total, err := s.DeviceRepo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.DeviceResponse, 0, len(entities))
	for _, e := range entities {
		resp, err := dto.ConvertToDeviceResponse(e, func(deviceNum uint) string {
			return s.getDeviceStatus(deviceNum)
		})
		if err != nil {
			return nil, 0, err
		}
		responses = append(responses, resp)
	}

	return responses, total, nil
}

// BatchDelete 批量删除设备 - 批量删除多个设备记录
func (s *DeviceService) BatchDelete(ctx context.Context, req dto.DeviceDeleteRequest) error {
	if len(req.IDs) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}

	// 检查每个设备是否属于当前用户公司，并收集设备编号用于删除缓存
	var deviceNums []uint
	for _, id := range req.IDs {
		current, err := s.DeviceRepo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if err := s.checkCompanyPermission(current, req.CompanyID); err != nil {
			return err
		}
		deviceNums = append(deviceNums, current.DeviceNum)
	}

	err := s.DeviceRepo.BatchDelete(ctx, req.IDs)
	if err != nil {
		return err
	}

	return nil
}

// ============================================================================
// 缓存相关方法 - 处理设备缓存数据的查询操作
// ============================================================================

// GetByDeviceNum 通过设备编号获取设备信息
func (s *DeviceService) GetByDeviceNum(ctx context.Context, deviceNum uint) (*dto.DeviceResponse, error) {
	entity, err := s.DeviceRepo.GetByDeviceNumFromDB(ctx, deviceNum)
	if err != nil {
		return nil, err
	}

	return dto.ConvertToDeviceResponse(entity, func(num uint) string {
		return s.getDeviceStatus(num)
	})
}

// ============================================================================
// 私有辅助方法 - 服务内部使用的辅助方法
// ============================================================================

// checkCompanyPermission 检查公司权限 - 验证用户是否有权限操作指定公司的设备
func (s *DeviceService) checkCompanyPermission(current *entity.Device, companyID uint) error {
	if current.CompanyID == nil || *current.CompanyID != companyID {
		return errx.New(constant.Forbidden, "只能操作本公司设备", nil)
	}
	return nil
}

// publishDeviceRestart 发布设备重启事件 - 通知设备配置已更新，需要重启
func (s *DeviceService) publishDeviceRestart(deviceNum uint) {
	go func() {
		restartPayload := pubsubRedis.RestartDevicePayload{
			DeviceID: uint32(deviceNum),
		}
		s.EventCollector.PublishRestartDevice(restartPayload)
	}()
}

// handleConfigUpdate 处理配置更新 - 配置更新后的通用处理逻辑（重启设备和清理缓存）
func (s *DeviceService) handleConfigUpdate(ctx context.Context, deviceNum uint) {
	s.publishDeviceRestart(deviceNum)
	if s.DeviceConfigCache != nil {
		s.DeviceConfigCache.DelConfig(ctx, uint32(deviceNum))
	}
}

// getDeviceStatus 获取设备在线状态 - 根据设备编号判断设备是否在线
func (s *DeviceService) getDeviceStatus(deviceNum uint) string {
	isOnline := s.DeviceRepo.GetOnlineStatus(uint32(deviceNum))

	status := entity.DeviceStatusInactive
	if isOnline {
		status = entity.DeviceStatusActive
	}

	return string(status)
}

// ============================================================================
// 独立更新方法 - 处理设备各个配置项的单独更新操作
// ============================================================================

// UpdateCameraConfig 更新摄像头配置 - 单独更新设备的摄像头配置
func (s *DeviceService) UpdateCameraConfig(ctx context.Context, id uint, req dto.UpdateCameraConfigRequest, companyID uint) error {
	// 获取当前设备信息
	current, err := s.DeviceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查权限
	if err := s.checkCompanyPermission(current, companyID); err != nil {
		return err
	}

	updates := make(map[string]interface{})

	// 处理摄像头配置
	if req.CameraConfig != nil {
		cameraJSON, err := dto.ToJSON(*req.CameraConfig)
		if err != nil {
			return errx.New(constant.BadRequest, "摄像头配置格式错误", err)
		}
		updates["camera_config"] = cameraJSON
	} else {
		// 设置为空
		updates["camera_config"] = nil
	}

	err = s.DeviceRepo.Update(ctx, id, updates)
	if err != nil {
		return err
	}

	// 处理配置更新后的清理工作
	s.handleConfigUpdate(ctx, current.DeviceNum)

	return nil
}

// UpdateSensors 更新传感器配置 - 单独更新设备的传感器配置
func (s *DeviceService) UpdateSensors(ctx context.Context, id uint, req dto.UpdateSensorsRequest, companyID uint) error {
	// 获取当前设备信息
	current, err := s.DeviceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查权限
	if err := s.checkCompanyPermission(current, companyID); err != nil {
		return err
	}

	// 验证传感器配置
	for _, sensor := range req.Sensors {
		if err := sensor.Validate(); err != nil {
			return err
		}
	}

	updates := make(map[string]interface{})

	// 处理传感器配置
	sensorsJSON, err := dto.ToJSON(req.Sensors)
	if err != nil {
		return errx.New(constant.BadRequest, "传感器配置格式错误", err)
	}
	updates["sensors"] = sensorsJSON

	err = s.DeviceRepo.Update(ctx, id, updates)
	if err != nil {
		return err
	}

	// 处理配置更新后的清理工作
	s.handleConfigUpdate(ctx, current.DeviceNum)

	return nil
}

// UpdateRelays 更新继电器配置 - 单独更新设备的继电器配置
func (s *DeviceService) UpdateRelays(ctx context.Context, id uint, req dto.UpdateRelaysRequest, companyID uint) error {
	// 获取当前设备信息
	current, err := s.DeviceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查权限
	if err := s.checkCompanyPermission(current, companyID); err != nil {
		return err
	}

	updates := make(map[string]interface{})

	// 处理继电器配置
	relaysJSON, err := dto.ToJSON(req.Relays)
	if err != nil {
		return errx.New(constant.BadRequest, "继电器配置格式错误", err)
	}
	updates["relays"] = relaysJSON

	err = s.DeviceRepo.Update(ctx, id, updates)
	if err != nil {
		return err
	}

	// 处理配置更新后的清理工作
	s.handleConfigUpdate(ctx, current.DeviceNum)

	return nil
}

// UpdateLocation 更新地理坐标 - 单独更新设备的地理位置坐标
func (s *DeviceService) UpdateLocation(ctx context.Context, id uint, req dto.UpdateLocationRequest, companyID uint) error {
	// 获取当前设备信息
	current, err := s.DeviceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查权限：只能更新本公司设备
	if current.CompanyID == nil || *current.CompanyID != companyID {
		return errx.New(constant.Forbidden, "只能更新本公司设备", nil)
	}

	updates := make(map[string]interface{})

	// 处理地理坐标
	if req.Location != nil {
		if err := req.Location.Validate(); err != nil {
			return err
		}

		locationJSON, err := dto.ToJSON(*req.Location)
		if err != nil {
			return errx.New(constant.BadRequest, "经纬度配置格式错误", err)
		}
		updates["location"] = locationJSON
	} else {
		// 设置为空
		updates["location"] = nil
	}

	return s.DeviceRepo.Update(ctx, id, updates)
}

// UpdateBasicInfo 更新设备基础信息 - 更新设备的基本信息，不包含复杂的配置字段
func (s *DeviceService) UpdateBasicInfo(ctx context.Context, id uint, req dto.UpdateBasicInfoRequest, companyID uint) error {
	// 获取当前设备信息
	current, err := s.DeviceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查权限：只能更新本公司设备
	if current.CompanyID == nil || *current.CompanyID != companyID {
		return errx.New(constant.Forbidden, "只能更新本公司设备", nil)
	}

	updates := make(map[string]interface{})

	// 处理MAC地址
	if req.MACAddress != nil && *req.MACAddress != *current.MACAddress {
		if exists, err := s.DeviceRepo.ExistsByField(ctx, "mac_address", *req.MACAddress); err != nil {
			return err
		} else if exists {
			return errx.New(constant.Conflict, "MAC地址已存在", nil)
		}
		updates["mac_address"] = *req.MACAddress
	}

	// 处理地理坐标
	if req.Location != nil {
		if err := req.Location.Validate(); err != nil {
			return err
		}

		locationJSON, err := dto.ToJSON(*req.Location)
		if err != nil {
			return errx.New(constant.BadRequest, "经纬度配置格式错误", err)
		}
		updates["location"] = locationJSON
	}

	// 处理枚举状态
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	// 处理普通字段
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.DeviceType != nil {
		updates["device_type"] = *req.DeviceType
	}
	if req.ReadInterval != nil {
		updates["read_interval"] = *req.ReadInterval
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.SIMCard != nil {
		updates["sim_card"] = *req.SIMCard
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.CoverImage != nil {
		updates["cover_image"] = *req.CoverImage
	}

	// 无有效更新时返回
	if len(updates) == 0 {
		return errx.New(constant.BadRequest, "未提供有效更新字段", nil)
	}

	if err := s.DeviceRepo.Update(ctx, id, updates); err != nil {
		return err
	}
	if s.DeviceConfigCache != nil {
		s.DeviceConfigCache.DelConfig(ctx, uint32(current.DeviceNum))
	}
	return nil
}

// ============================================================================
// 水肥信息管理方法 - 处理设备水肥信息的增删改查操作
// ============================================================================

// UpdateWaterFertilizerInfo 更新水肥信息 - 单独更新设备的水肥信息配置
func (s *DeviceService) UpdateWaterFertilizerInfo(ctx context.Context, id uint, req dto.UpdateWaterFertilizerInfoRequest, companyID uint) error {
	// 获取当前设备信息
	current, err := s.DeviceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查权限
	if err := s.checkCompanyPermission(current, companyID); err != nil {
		return err
	}

	updates := make(map[string]interface{})

	// 处理水肥信息配置
	if req.WaterFertilizerInfo != nil {
		infoJSON, err := dto.ToJSON(*req.WaterFertilizerInfo)
		if err != nil {
			return errx.New(constant.BadRequest, "水肥信息格式错误", err)
		}
		updates["water_fertilizer_info"] = infoJSON
	} else {
		// 设置为空
		updates["water_fertilizer_info"] = nil
	}

	if err := s.DeviceRepo.Update(ctx, id, updates); err != nil {
		return err
	}
	if s.DeviceConfigCache != nil {
		s.DeviceConfigCache.DelConfig(ctx, uint32(current.DeviceNum))
	}
	return nil
}

// GetWaterFertilizerInfo 查询水肥信息 - 获取设备的水肥信息配置
func (s *DeviceService) GetWaterFertilizerInfo(ctx context.Context, id uint, companyID uint) (*dto.GetWaterFertilizerInfoResponse, error) {
	// 获取当前设备信息
	current, err := s.DeviceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 检查权限
	if err := s.checkCompanyPermission(current, companyID); err != nil {
		return nil, err
	}

	resp := &dto.GetWaterFertilizerInfoResponse{}

	// 水肥信息配置转换
	if current.WaterFertilizerInfo != nil {
		var waterFertilizerInfo dto.DeviceWaterFertilizerInfo
		if err := dto.FromJSON(current.WaterFertilizerInfo, &waterFertilizerInfo); err == nil {
			resp.WaterFertilizerInfo = &waterFertilizerInfo
		}
	}

	return resp, nil
}

// DeleteWaterFertilizerInfo 删除水肥信息 - 删除设备的水肥信息配置
func (s *DeviceService) DeleteWaterFertilizerInfo(ctx context.Context, id uint, companyID uint) error {
	// 获取当前设备信息
	current, err := s.DeviceRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查权限
	if err := s.checkCompanyPermission(current, companyID); err != nil {
		return err
	}

	updates := make(map[string]interface{})
	// 设置为空，删除水肥信息
	updates["water_fertilizer_info"] = nil

	if err := s.DeviceRepo.Update(ctx, id, updates); err != nil {
		return err
	}
	if s.DeviceConfigCache != nil {
		s.DeviceConfigCache.DelConfig(ctx, uint32(current.DeviceNum))
	}
	return nil
}
