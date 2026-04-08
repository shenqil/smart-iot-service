package core

import (
	"context"
	"fmt"
	"rbac-admin/pkg/logger"
	"sync"
)

// Collector 采集器核心实现
type Collector struct {
	ctx    context.Context
	cancel context.CancelFunc

	// 服务管理
	services   map[string]ServiceInterface
	servicesMu sync.RWMutex

	// 配置
	config *CollectorConfig
}

// CollectorConfig 采集器配置
type CollectorConfig struct {
	EventChanSize int
}

// NewCollectorWithServices 创建新的采集器并直接初始化所有服务
func NewCollectorWithServices(config *CollectorConfig, services ...ServiceInterface) *Collector {
	ctx, cancel := context.WithCancel(context.Background())

	collector := &Collector{
		ctx:      ctx,
		cancel:   cancel,
		services: make(map[string]ServiceInterface),
		config:   config,
	}

	// 直接注册所有服务
	for _, service := range services {
		collector.services[service.GetName()] = service
		logger.Logger.Info(fmt.Sprintf("直接注册服务: %s", service.GetName()))
	}

	return collector
}

// GetService 获取服务
func (c *Collector) GetService(serviceName string) (ServiceInterface, bool) {
	c.servicesMu.RLock()
	defer c.servicesMu.RUnlock()

	service, exists := c.services[serviceName]
	return service, exists
}

// Start 启动采集器
func (c *Collector) Start() error {
	logger.Logger.Info("启动采集器...")

	c.servicesMu.RLock()
	services := make([]ServiceInterface, 0, len(c.services))
	for _, service := range c.services {
		services = append(services, service)
	}
	c.servicesMu.RUnlock()

	// 启动所有服务
	var wg sync.WaitGroup
	for _, service := range services {
		wg.Add(1)
		go func(svc ServiceInterface) {
			defer wg.Done()
			if err := svc.Start(); err != nil {
				logger.Logger.Error(fmt.Sprintf("启动服务 %s 失败: %v", svc.GetName(), err))
			}
		}(service)
	}

	// 等待所有服务启动
	go func() {
		wg.Wait()
		logger.Logger.Info("所有服务启动完成")
	}()

	return nil
}

// Stop 停止采集器
func (c *Collector) Stop() error {
	logger.Logger.Info("停止采集器...")

	// 取消上下文
	c.cancel()

	// 停止所有服务
	c.servicesMu.RLock()
	services := make([]ServiceInterface, 0, len(c.services))
	for _, service := range c.services {
		services = append(services, service)
	}
	c.servicesMu.RUnlock()

	for _, service := range services {
		if err := service.Stop(); err != nil {
			logger.Logger.Error(fmt.Sprintf("停止服务 %s 失败: %v", service.GetName(), err))
		}
	}

	logger.Logger.Info("采集器已停止")
	return nil
}

// DisconnectedDevice 断开设备连接
func (c *Collector) DisconnectedDevice(deviceID uint32) {
	c.servicesMu.RLock()
	defer c.servicesMu.RUnlock()

	// 尝试所有服务，如果找到存在的服务则执行并停止
	for _, service := range c.services {
		if exists := service.DisconnectedDevice(deviceID); exists {
			return // 服务存在并已处理，停止执行
		}
	}

	// 所有服务都不存在，记录警告
	logger.Logger.Warn(fmt.Sprintf("设备 %d 不存在", deviceID))
}

// DisconnectOtherConnections 断开同一设备ID的其他连接
func (c *Collector) DisconnectOtherConnections(deviceID uint32, currentConnectionID string, currentTime int64) {
	c.servicesMu.RLock()
	defer c.servicesMu.RUnlock()

	// 尝试所有服务，如果找到存在的服务则执行并停止
	for _, service := range c.services {
		service.DisconnectOtherConnections(deviceID, currentConnectionID, currentTime)
	}
}

// RelaySwitch 控制继电器
func (c *Collector) RelaySwitch(deviceID uint32, relayIndex int, isOpen bool) error {
	c.servicesMu.RLock()
	defer c.servicesMu.RUnlock()

	// 尝试所有服务，如果找到存在的服务则执行并停止
	for _, service := range c.services {
		if err, exists := service.RelaySwitch(deviceID, relayIndex, isOpen); exists {
			return err // 服务存在，返回执行结果（成功或错误）
		}
	}

	// 所有服务都不存在，返回错误
	return fmt.Errorf("设备 %d 不存在", deviceID)
}

// SetDeviceReadInterval 设置设备读取间隔
func (c *Collector) SetDeviceReadInterval(deviceID uint32, intervalMinutes int) error {
	c.servicesMu.RLock()
	defer c.servicesMu.RUnlock()

	// 尝试所有服务，如果找到存在的服务则执行并停止
	for _, service := range c.services {
		if err, exists := service.SetDeviceReadInterval(deviceID, intervalMinutes); exists {
			return err // 服务存在，返回执行结果（成功或错误）
		}
	}

	// 所有服务都不存在，返回错误
	return fmt.Errorf("设备 %d 不存在", deviceID)
}
