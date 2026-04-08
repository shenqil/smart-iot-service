package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"rbac-admin/internal/cache"
	"rbac-admin/internal/model/dto"
	"rbac-admin/pkg/logger"
	"sync"
)

// DeviceConfig 设备配置结构
type DeviceConfig struct {
	CompanyID    uint
	DeviceName   string
	Sensors      []dto.SensorConfig
	Relays       []dto.RelayConfig
	ReadInterval int
}

// DeviceConfigProvider 负责从缓存中获取设备配置。
type DeviceConfigProvider struct {
	deviceConfigCache *cache.DeviceConfigCache
	configCache       map[uint32]*DeviceConfig // Changed to pointer
	cacheLock         sync.RWMutex
}

// NewDeviceConfigProvider 创建一个新的 DeviceConfigProvider 实例。
func NewDeviceConfigProvider(deviceConfigCache *cache.DeviceConfigCache) *DeviceConfigProvider {
	return &DeviceConfigProvider{
		deviceConfigCache: deviceConfigCache,
		configCache:       make(map[uint32]*DeviceConfig), // Changed to pointer
	}
}

// filter 通用过滤函数
func filter[S ~[]E, E any](s S, f func(E) bool) S {
	result := make(S, 0, len(s))

	for _, v := range s {
		if f(v) {
			result = append(result, v)
		}
	}
	return result
}

// LoadAndCacheDeviceConfig 从 DeviceConfigCache 加载并缓存设备配置。
func (p *DeviceConfigProvider) LoadAndCacheDeviceConfig(deviceNum uint32) (*DeviceConfig, error) {
	if p.deviceConfigCache == nil {
		return nil, fmt.Errorf("DeviceConfigCache未初始化")
	}

	// 1. 尝试从 DeviceConfigProvider 自身的内存缓存获取
	if cfg, ok := p.getCachedDeviceConfig(deviceNum); ok {
		logger.Logger.Debug(fmt.Sprintf("从 DeviceConfigProvider 内存缓存获取设备 %d 配置", deviceNum))
		return cfg, nil
	}

	// 2. 内存缓存未命中，从 DeviceConfigCache (Redis/DB) 获取
	ctx := context.Background()
	device, err := p.deviceConfigCache.GetConfig(ctx, deviceNum)
	if err != nil {
		return nil, fmt.Errorf("从 DeviceConfigCache 获取设备 %d 配置失败: %w", deviceNum, err)
	}

	if device == nil {
		return nil, fmt.Errorf("设备 %d 不存在", deviceNum)
	}

	// 3. 检查是否已删除
	if device.DeletedAt.Valid {
		return nil, fmt.Errorf("设备 %d 已被删除", deviceNum)
	}

	// 4. 检查是否激活
	if device.IsActive == nil || !*device.IsActive {
		return nil, fmt.Errorf("设备 %d 未激活，忽略配置缓存", deviceNum)
	}

	// 5. 解析配置
	var sensors []dto.SensorConfig
	var relays []dto.RelayConfig
	if err := json.Unmarshal(device.Sensors, &sensors); err != nil {
		return nil, fmt.Errorf("解析传感器配置失败: %w", err)
	}
	if err := json.Unmarshal(device.Relays, &relays); err != nil {
		return nil, fmt.Errorf("解析继电器配置失败: %w", err)
	}

	// 6. 缓存到 DeviceConfigProvider 的内存中
	p.cacheLock.Lock()
	defer p.cacheLock.Unlock()

	deviceName := ""
	if device.Name != nil {
		deviceName = *device.Name
	}

	var companyID uint
	if device.CompanyID != nil {
		companyID = *device.CompanyID
	}

	readInterval := 60 // 默认值
	if device.ReadInterval != nil {
		readInterval = *device.ReadInterval
	}

	cfg := &DeviceConfig{ // Store pointer
		CompanyID:  companyID,
		DeviceName: deviceName,
		Sensors: filter(sensors, func(s dto.SensorConfig) bool {
			return s.StandardID != nil && *s.StandardID != 0
		}),
		Relays: filter(relays, func(s dto.RelayConfig) bool {
			return s.StandardID != nil && *s.StandardID != 0
		}),
		ReadInterval: readInterval,
	}
	p.configCache[deviceNum] = cfg

	return cfg, nil
}

// removeDeviceConfig 移除设备配置缓存。
func (p *DeviceConfigProvider) RemoveDeviceConfig(deviceNum uint32) {
	p.cacheLock.Lock()
	defer p.cacheLock.Unlock()
	delete(p.configCache, deviceNum)
	logger.Logger.Info(fmt.Sprintf("设备 %d 配置已从 DeviceConfigProvider 内存缓存中移除", deviceNum))
}

// getCachedDeviceConfig 获取缓存的设备配置。
func (p *DeviceConfigProvider) getCachedDeviceConfig(deviceNum uint32) (*DeviceConfig, bool) {
	p.cacheLock.RLock()
	defer p.cacheLock.RUnlock()
	cfg, ok := p.configCache[deviceNum]
	return cfg, ok
}

// Close 关闭 DeviceConfigProvider，清空内存缓存。
func (p *DeviceConfigProvider) Close() error {
	p.cacheLock.Lock()
	p.configCache = make(map[uint32]*DeviceConfig)
	p.cacheLock.Unlock()
	logger.Logger.Info("DeviceConfigProvider 内存缓存已清空")
	return nil
}