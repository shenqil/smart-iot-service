package devicesimulator

import (
	"context"
	"fmt"
	"log/slog"
	"rbac-admin/internal/model/entity"
	"rbac-admin/pkg/logger"
	pubsub "rbac-admin/pkg/pubsubRedis"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"
)

// DeviceSimulatorManager 设备模拟器管理器
type DeviceSimulatorManager struct {
	db             *gorm.DB
	eventCollector *pubsub.EventCollector
	serverAddr     string
	logger         *slog.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
	clients        map[string]*IoTDeviceClient // map[deviceSN]*IoTDeviceClient
	wg             sync.WaitGroup              // 等待所有客户端停止
}

// NewDeviceSimulatorManager 创建设备模拟器管理器
func NewDeviceSimulatorManager(db *gorm.DB, eventCollector *pubsub.EventCollector) *DeviceSimulatorManager {
	ctx, cancel := context.WithCancel(context.Background())

	// 获取IoT服务地址

	// 创建日志器
	logger := createLogger()

	manager := &DeviceSimulatorManager{
		db:             db,
		eventCollector: eventCollector,
		serverAddr:     "127.0.0.1:9898",
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
		clients:        make(map[string]*IoTDeviceClient),
	}
	manager.logger.Info("创建设备模拟器管理器")
	return manager
}

// Start 启动管理器
func (m *DeviceSimulatorManager) Start() error {
	m.logger.Info("启动设备模拟器管理器")

	// 订阅模拟设备变更事件
	m.eventCollector.SubscribeDeviceSimulatorChange(m.handleDeviceSimulatorChange)

	// 启动时加载所有需要模拟的设备
	if err := m.loadAllDevices(); err != nil {
		m.logger.Error("加载设备列表失败", slog.Any("error", err))
		return err
	}

	m.logger.Info("设备模拟器管理器启动成功")
	return nil
}

// Stop 停止管理器并等待所有客户端完全停止
func (m *DeviceSimulatorManager) Stop() error {
	m.logger.Info("停止设备模拟器管理器")

	m.cancel()

	// 停止所有客户端
	m.mu.Lock()
	clients := make([]*IoTDeviceClient, 0, len(m.clients))
	deviceSNs := make([]string, 0, len(m.clients))
	for deviceSN, client := range m.clients {
		clients = append(clients, client)
		deviceSNs = append(deviceSNs, deviceSN)
	}
	m.clients = make(map[string]*IoTDeviceClient) // 清空映射
	m.mu.Unlock()

	// 并发停止所有客户端
	for i, client := range clients {
		m.wg.Add(1)
		go func(deviceSN string, c *IoTDeviceClient) {
			defer m.wg.Done()
			c.Stop()
			m.logger.Info(fmt.Sprintf("停止设备模拟器: %s", deviceSN))
		}(deviceSNs[i], client)
	}

	// 等待所有客户端停止完成，设置超时避免无限等待
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		m.logger.Info("所有设备模拟器已停止")
	case <-time.After(10 * time.Second):
		m.logger.Warn("等待设备模拟器停止超时")
	}

	m.logger.Info("设备模拟器管理器已停止")
	return nil
}

// loadAllDevices 加载所有需要模拟的设备
func (m *DeviceSimulatorManager) loadAllDevices() error {
	m.logger.Info("开始加载所有需要模拟的设备")

	// 分页读取所有设备
	pageNumber := 1
	pageSize := 100

	for {
		var simulators []entity.DeviceSimulator
		var total int64

		// 直接使用 GORM 查询
		db := m.db.Model(&entity.DeviceSimulator{})

		// 获取总数
		if err := db.Count(&total).Error; err != nil {
			return fmt.Errorf("统计设备总数失败: %w", err)
		}

		// 分页查询
		offset := (pageNumber - 1) * pageSize
		if err := db.Order("id DESC").
			Offset(offset).
			Limit(pageSize).
			Find(&simulators).Error; err != nil {
			return fmt.Errorf("查询设备列表失败: %w", err)
		}

		// 启动当前页的设备
		for _, simulator := range simulators {
			m.addDevice(simulator.DeviceSN)
		}

		m.logger.Info(fmt.Sprintf("已加载第 %d 页设备，共 %d 条，总计 %d 条", pageNumber, len(simulators), total))

		// 如果当前页数据少于pageSize，说明已经是最后一页
		if len(simulators) < pageSize {
			break
		}

		// 如果已经加载完所有数据
		if int64(pageNumber*pageSize) >= total {
			break
		}

		pageNumber++
	}

	m.logger.Info(fmt.Sprintf("设备加载完成，共启动 %d 个设备模拟器", len(m.clients)))
	return nil
}

// handleDeviceSimulatorChange 处理模拟设备变更事件
func (m *DeviceSimulatorManager) handleDeviceSimulatorChange(payload pubsub.DeviceSimulatorChangePayload) {
	m.logger.Info(fmt.Sprintf("收到模拟设备变更事件: type=%s, deviceSNs=%v", payload.Type, payload.DeviceSNs))

	switch payload.Type {
	case pubsub.DeviceSimulatorChangeTypeAdd:
		// 增加设备
		for _, deviceSN := range payload.DeviceSNs {
			m.addDevice(deviceSN)
		}
	case pubsub.DeviceSimulatorChangeTypeRemove:
		// 减少设备
		for _, deviceSN := range payload.DeviceSNs {
			m.removeDevice(deviceSN)
		}
	default:
		m.logger.Warn(fmt.Sprintf("未知的变更类型: %s", payload.Type))
	}
}

// addDevice 添加设备模拟器
func (m *DeviceSimulatorManager) addDevice(deviceSN string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查设备是否已存在
	if _, exists := m.clients[deviceSN]; exists {
		m.logger.Warn(fmt.Sprintf("设备模拟器已存在: %s", deviceSN))
		return
	}

	// 将设备SN转换为设备ID（假设设备SN就是设备ID的字符串形式）
	deviceID, err := strconv.ParseUint(deviceSN, 10, 32)
	if err != nil {
		// 如果无法转换为数字，记录错误并返回
		m.logger.Error(fmt.Sprintf("设备SN %s 无法转换为数字，跳过添加: %v", deviceSN, err))
		return
	}

	m.logger.Info(fmt.Sprintf("准备添加设备模拟器: SN=%s, ID=%d, 服务器地址=%s", deviceSN, deviceID, m.serverAddr))

	// 创建并启动客户端（传递日志器，在goroutine中运行）
	client := NewIoTDeviceClient(uint32(deviceID), m.serverAddr, m.logger)
	client.Start() // Start() 现在在内部使用goroutine，不会阻塞

	m.clients[deviceSN] = client
	m.logger.Info(fmt.Sprintf("已添加设备模拟器: SN=%s, ID=%d, 当前总设备数: %d", deviceSN, deviceID, len(m.clients)))
}

// removeDevice 移除设备模拟器
func (m *DeviceSimulatorManager) removeDevice(deviceSN string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[deviceSN]
	if !exists {
		m.logger.Warn(fmt.Sprintf("设备模拟器不存在: %s", deviceSN))
		return
	}

	// 停止客户端
	client.Stop()

	// 从映射中删除
	delete(m.clients, deviceSN)
	m.logger.Info(fmt.Sprintf("已移除设备模拟器: %s", deviceSN))
}

// createLogger 创建日志器
func createLogger() *slog.Logger {
	// 使用项目中的 logger 包创建服务日志器
	return logger.NewServiceLogger("device_simulator_manager")
}
