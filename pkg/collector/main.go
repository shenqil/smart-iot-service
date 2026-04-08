package collector

import (
	"context"
	"fmt"
	"time"

	"rbac-admin/internal/cache"
	"rbac-admin/pkg/collector/core"
	services "rbac-admin/pkg/collector/services"
	"rbac-admin/pkg/collector/singletons"
	"rbac-admin/pkg/collector/storage"
	"rbac-admin/pkg/config"
	"rbac-admin/pkg/logger"
	pubsub "rbac-admin/pkg/pubsubRedis"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// CollectorApp 结构体用于封装采集器相关的逻辑和依赖
type CollectorApp struct {
	influxClient      *influxdb3.Client
	deviceConfigCache *cache.DeviceConfigCache
	deviceDataCache   *cache.DeviceDataCache // This will be used for UpdateRelayStatus
	eventCollector    *pubsub.EventCollector
	coreCollector     *core.Collector
	dataProcessor     *storage.DataProcessor // Changed from influxStorage
	eventPublisher    *singletons.EventPublisher
}

// NewCollectorApp 创建新的 CollectorApp 实例
func NewCollectorApp(
	influxClient *influxdb3.Client,
	deviceConfigCache *cache.DeviceConfigCache,
	deviceDataCache *cache.DeviceDataCache, // This will be passed
	eventCollector *pubsub.EventCollector,
) *CollectorApp {
	// 初始化事件发布器
	eventPublisher := singletons.GetEventPublisher()
	logger.Logger.Info("事件发布器已初始化")

	// Initialize new storage components
	deviceConfigProvider := storage.NewDeviceConfigProvider(deviceConfigCache)
	alertManager := storage.NewAlertManager(eventCollector)
	influxWriter := storage.NewInfluxWriter(influxClient)
	dataProcessor := storage.NewDataProcessor(deviceConfigProvider, alertManager, influxWriter, deviceDataCache, eventCollector)
	logger.Logger.Info("DataProcessor 已初始化")

	// 创建配置
	confCollector := config.Conf.Collector
	collectorConfig := &core.CollectorConfig{
		EventChanSize: confCollector.ChanSize,
	}

	// 创建三个服务
	service1 := services.NewActiveReportService(confCollector.Addr1)
	service2 := services.NewControlService(confCollector.Addr2)
	service3 := services.NewIoTService(confCollector.Addr3)

	// 直接创建采集器并注册所有服务
	coreCollector := core.NewCollectorWithServices(collectorConfig, service1, service2, service3)

	return &CollectorApp{
		influxClient:      influxClient,
		deviceConfigCache: deviceConfigCache,
		deviceDataCache:   deviceDataCache, // Assign deviceDataCache
		eventCollector:    eventCollector,
		coreCollector:     coreCollector,
		dataProcessor:     dataProcessor, // Assign dataProcessor
		eventPublisher:    eventPublisher,
	}
}

// Start 启动采集器应用
func (app *CollectorApp) Start() {
	app.setupRedisEventSubscriptions()
	app.setupInternalEventSubscriptions()

	app.coreCollector.Start()

	logger.Logger.Info("采集器应用启动完成")
}

// setupRedisEventSubscriptions 设置Redis事件订阅
func (app *CollectorApp) setupRedisEventSubscriptions() {
	if app.eventCollector == nil {
		logger.Logger.Error("EventCollector 实例为空，无法设置事件订阅")
		return
	}

	// 订阅继电器切换事件
	app.eventCollector.SubscribeRelaySwitch(func(payload pubsub.RelaySwitchPayload) {
		logger.Logger.Info(fmt.Sprintf("收到继电器切换事件: 设备ID=%d, 继电器=%d, 状态=%v, 操作者=%s",
			payload.DeviceID, payload.RelayIndex, payload.IsOpen, payload.Operator))

		// 直接调用 core 的 RelaySwitch 方法
		if err := app.coreCollector.RelaySwitch(payload.DeviceID, payload.RelayIndex, payload.IsOpen); err != nil {
			logger.Logger.Error(fmt.Sprintf("继电器切换失败: %v", err))
		} else {
			logger.Logger.Info(fmt.Sprintf("继电器切换成功: 设备ID=%d, 继电器=%d", payload.DeviceID, payload.RelayIndex))
		}
	})

	// 订阅重启设备事件
	app.eventCollector.SubscribeRestartDevice(func(payload pubsub.RestartDevicePayload) {
		logger.Logger.Info(fmt.Sprintf("收到重启设备事件: 设备ID=%d", payload.DeviceID))

		// 直接调用 core 的 DisconnectedDevice 方法
		app.coreCollector.DisconnectedDevice(payload.DeviceID)
		logger.Logger.Info(fmt.Sprintf("设备重启处理完成: 设备ID=%d", payload.DeviceID))
	})

	logger.Logger.Info("已订阅 EventCollector 事件")
}

// setupInternalEventSubscriptions 设置内部事件订阅
func (app *CollectorApp) setupInternalEventSubscriptions() {
	// 使用Subscribe订阅所有事件，在回调中根据事件类型进行分发
	app.eventPublisher.Subscribe(func(event singletons.Event) {
		switch event.Type {
		case singletons.EventStatus:
			app.handleDeviceStatusEvent(&event)
		case singletons.EventData:
			app.handleDeviceDataEvent(&event)
		case singletons.EventRelayStatus:
			app.handleRelayStatusEvent(&event)
		default:
			// 记录未知事件类型的警告
			logger.Logger.Warn(fmt.Sprintf("收到未知事件类型: %d", event.Type))
		}
	})

	logger.Logger.Info("内部事件订阅已设置")
}

// handleDeviceStatusEvent 处理设备状态事件
func (app *CollectorApp) handleDeviceStatusEvent(event *singletons.Event) {
	status, ok := event.Data.(*singletons.DeviceStatus)
	if !ok {
		logger.Logger.Error(fmt.Sprintf("设备状态事件数据类型错误: expected *singletons.DeviceStatus, got %T", event.Data))
		return
	}

	deviceID := status.ID
	timestamp := status.Time.Unix()
	isOnline := status.Online == 1

	// Attempt to load device config. This config is crucial for subsequent operations.
	config, err := app.dataProcessor.LoadAndCacheDeviceConfig(deviceID)
	if err != nil {
		// If config loading fails, log and handle disconnection if device is online.
		// Further processing depending on config cannot proceed safely.
		if isOnline {
			logger.Logger.Error(fmt.Sprintf("设备 %d 上线时加载配置失败，断开连接: %v", deviceID, err))
			app.coreCollector.DisconnectedDevice(deviceID)
		} else {
			logger.Logger.Error(fmt.Sprintf("设备 %d 下线时加载配置失败: %v", deviceID, err))
		}
		// In either case (online or offline), if config loading failed, we cannot proceed with
		// config-dependent logic (like publishing a detailed alarm or setting read interval).
		// We can publish a generic alarm here if desired, but for now, we'll return.
		return // Exit early if config cannot be loaded
	}

	// Logic for device status based on whether it's online or offline
	if isOnline {
		logger.Logger.Info(fmt.Sprintf("设备上线: ID=%d, 连接ID=%s", deviceID, status.ConnectionID))
		app.coreCollector.DisconnectOtherConnections(deviceID, status.ConnectionID, timestamp)
		if config.ReadInterval > 0 {
			if err := app.coreCollector.SetDeviceReadInterval(deviceID, config.ReadInterval); err != nil {
				logger.Logger.Error(fmt.Sprintf("设备 %d 设置读取间隔失败: %v", deviceID, err))
			} else {
				logger.Logger.Info(fmt.Sprintf("设备 %d 读取间隔已设置为: %d 秒", deviceID, config.ReadInterval))
			}
		}
		logger.Logger.Info(fmt.Sprintf("设备 %d 上线处理完成", deviceID))
	} else {
		logger.Logger.Info(fmt.Sprintf("设备下线: ID=%d", deviceID))
		app.dataProcessor.RemoveDeviceConfig(deviceID)
		logger.Logger.Info(fmt.Sprintf("设备 %d 下线处理完成", deviceID))
	}

	// Determine alarm type and description based on online status
	var alarmType, description string
	if isOnline {
		alarmType = "device_online"
		description = "设备上线"
	} else {
		alarmType = "device_offline"
		description = "设备下线"
	}

	// Publish device status alarm
	alarmPayload := pubsub.DeviceAlarmPayload{
		AlarmLevel:   4, // Info
		AlarmType:    alarmType,
		EnterpriseId: fmt.Sprintf("%d", config.CompanyID), // config is guaranteed to be non-nil here
		DeviceId:     fmt.Sprintf("%d", deviceID),
		DeviceName:   config.DeviceName, // config is guaranteed to be non-nil here
		Description:  description,
		Value:        "",
		Threshold:    "",
		Timestamp:    timestamp,
	}

	if err := app.eventCollector.PublishDeviceAlarm(alarmPayload); err != nil {
		logger.Logger.Error(fmt.Sprintf("发布设备告警失败: 设备ID=%d, 类型=%s, 错误=%v", deviceID, alarmType, err))
	}
}

// handleDeviceDataEvent 处理设备数据事件
func (app *CollectorApp) handleDeviceDataEvent(event *singletons.Event) {
	data, ok := event.Data.(*singletons.DeviceData)
	if !ok {
		logger.Logger.Error(fmt.Sprintf("设备数据事件数据类型错误: %T", event.Data))
		return
	}

	// 插入设备数据到InfluxDB
	if err := app.dataProcessor.InsertDeviceData(data); err != nil { // Changed call
		logger.Logger.Error(fmt.Sprintf("插入设备数据失败: %v", err))
	}
}

// handleRelayStatusEvent 处理继电器状态事件
func (app *CollectorApp) handleRelayStatusEvent(event *singletons.Event) {
	relay, ok := event.Data.(*singletons.RelayStatus)
	if !ok {
		logger.Logger.Error(fmt.Sprintf("继电器状态事件数据类型错误: %T", event.Data))
		return
	}

	// 发布继电器切换ASK事件
	if app.eventCollector != nil {
		askPayload := pubsub.RelaySwitchAskPayload{
			DeviceID:   relay.ID,
			RelayIndex: int(relay.RelayNum), // RelayNum是0-31
			State:      relay.State,
			Time:       time.Now().Unix(),
		}
		if err := app.eventCollector.PublishRelaySwitchAsk(askPayload); err != nil {
			logger.Logger.Error(fmt.Sprintf("发布继电器切换ASK事件失败: %v", err))
		}
	}

	// 更新继电器状态到Redis缓存 (using deviceDataCache.UpdateRelayStatus)
	if app.deviceDataCache != nil {
		// The `timestamp` for UpdateRelayStatus is taken from the current time.
		if err := app.deviceDataCache.UpdateRelayStatus(context.Background(), relay.ID, uint(relay.RelayNum), relay.State, time.Now()); err != nil {
			logger.Logger.Error(fmt.Sprintf("更新继电器状态到Redis失败[deviceID=%d, relayNum=%d]: %v", relay.ID, relay.RelayNum, err))
		}
	}
}
