package pubsubRedis

import (
	"context"
	"encoding/json"
	"fmt"
	"rbac-admin/pkg/config"
	"rbac-admin/pkg/logger"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// 定义事件类型
type EventType string

const (
	channelCollector = "collector" // 统一的事件通道

	EventRelaySwitch    EventType = "relay_switch"     // 继电器切换专用事件
	EventRelaySwitchAsk EventType = "relay_switch_ask" // 继电器切换ASK事件
	EventRestartDevice  EventType = "restart_device"   // 重启设备

	EventDeviceAlarm           EventType = "device_alarm"            // 通用设备告警事件
	EventDeviceSimulatorChange EventType = "device_simulator_change" // 模拟设备增加/减少事件
)

// RelaySwitchPayload 继电器切换事件负载
type RelaySwitchPayload struct {
	DeviceID   uint32 `json:"device_id"`
	RelayIndex int    `json:"relay_index"`
	IsOpen     bool   `json:"is_open"`
	Operator   string `json:"operator"`
}

// RestartDevicePayload 重启设备事件负载
type RestartDevicePayload struct {
	DeviceID uint32 `json:"device_id"`
}

// DeviceAlarmPayload 通用设备告警事件负载（对齐 entity.DeviceAlarm）
type DeviceAlarmPayload struct {
	AlarmLevel   uint8  `json:"alarm_level"` // 1,2,3,4
	AlarmType    string `json:"alarm_type"`
	AlarmSubType string `json:"alarm_sub_type,omitempty"`
	EnterpriseId string `json:"enterprise_id"`
	DeviceId     string `json:"device_id"`
	DeviceName   string `json:"device_name"`
	Description  string `json:"description"`
	Value        string `json:"value"`
	Threshold    string `json:"threshold"`
	Timestamp    int64  `json:"timestamp"`
}

// RelaySwitchHandler 继电器切换事件回调函数类型
type RelaySwitchHandler func(payload RelaySwitchPayload)

// RelaySwitchAskPayload 继电器切换ASK事件负载
type RelaySwitchAskPayload struct {
	DeviceID   uint32 `json:"device_id"`
	RelayIndex int    `json:"relay_index"`
	State      byte   `json:"state"`
	Time       int64  `json:"time"` // Unix时间戳
}

// RelaySwitchAskHandler 继电器切换ASK事件回调函数类型
type RelaySwitchAskHandler func(payload RelaySwitchAskPayload)

// RestartDeviceHandler 重启设备事件回调函数类型
type RestartDeviceHandler func(payload RestartDevicePayload)

// DeviceAlarmHandler 通用设备告警事件回调函数类型
type DeviceAlarmHandler func(payload DeviceAlarmPayload)

// DeviceSimulatorChangeType 模拟设备变更类型
type DeviceSimulatorChangeType string

const (
	DeviceSimulatorChangeTypeAdd    DeviceSimulatorChangeType = "add"    // 增加设备
	DeviceSimulatorChangeTypeRemove DeviceSimulatorChangeType = "remove" // 减少设备
)

// DeviceSimulatorChangePayload 模拟设备变更事件负载
type DeviceSimulatorChangePayload struct {
	Type      DeviceSimulatorChangeType `json:"type"`      // 变更类型：add 或 remove
	DeviceSNs []string                  `json:"deviceSNs"` // 设备SN号数组
}

// DeviceSimulatorChangeHandler 模拟设备变更事件回调函数类型
type DeviceSimulatorChangeHandler func(payload DeviceSimulatorChangePayload)

// EventCollector 事件收集器，封装 Redis 发布订阅功能
type EventCollector struct {
	client        *redis.Client
	redisPubSub   *redis.PubSub
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	eventHandlers map[EventType][]interface{} // 事件类型到监听函数的映射
}

// getRedisOptions 获取优化的 Redis 连接选项
func getRedisOptions() *redis.Options {
	confPubsub := config.Conf.Pubsub
	confRedis := config.Conf.Redis

	return &redis.Options{
		Addr:     confRedis.Addr,
		Password: confRedis.Password,
		DB:       confPubsub.DB,
		// 连接池优化配置
		PoolSize:     50, // 增加连接池大小，支持更多并发
		MinIdleConns: 10, // 增加最小空闲连接，减少连接创建开销
		MaxRetries:   5,  // 增加重试次数，提高可靠性
		// 超时配置优化
		DialTimeout:  10 * time.Second, // 增加连接超时时间
		ReadTimeout:  5 * time.Second,  // 增加读取超时时间
		WriteTimeout: 5 * time.Second,  // 增加写入超时时间
		PoolTimeout:  30 * time.Second, // 连接池超时
		IdleTimeout:  5 * time.Minute,  // 空闲连接超时
		// 连接保活配置
		IdleCheckFrequency: 1 * time.Minute, // 空闲连接检查频率
		// 网络配置
		Network: "tcp", // 明确指定网络类型
	}
}

// NewEventCollector 创建新的事件收集器（保持向后兼容）
// 注意：此方法现在返回单例实例，多次调用返回同一个实例
func NewEventCollector() (*EventCollector, error) {
	// 使用优化的 Redis 选项创建客户端
	client := redis.NewClient(getRedisOptions())

	ctx := context.Background()

	// 测试连接
	if _, err := client.Ping(ctx).Result(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// 订阅 collector 频道
	redisPubSub := client.Subscribe(ctx, channelCollector)

	ctx, cancel := context.WithCancel(context.Background())

	collector := &EventCollector{
		client:        client,
		redisPubSub:   redisPubSub,
		ctx:           ctx,
		cancel:        cancel,
		eventHandlers: make(map[EventType][]interface{}),
	}

	// 启动事件分发器
	go collector.startDispatcher()

	logger.Logger.Info("EventCollector 单例实例创建成功")
	return collector, nil
}

// startDispatcher 启动事件分发器
func (ec *EventCollector) startDispatcher() {
	defer func() {
		if err := recover(); err != nil {
			logger.Logger.Error(fmt.Sprintf("Event collector dispatcher panic recovered: %v", err))
		}
	}()

	ch := ec.redisPubSub.Channel()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				logger.Logger.Error("Redis pubsub channel closed, stopping dispatcher")
				return
			}
			ec.dispatchEvent(msg.Payload)

		case <-ec.ctx.Done():
			logger.Logger.Error("Context cancelled, stopping dispatcher")
			return
		}
	}
}

// dispatchEvent 分发事件给对应的监听函数
func (ec *EventCollector) dispatchEvent(data string) {
	// 查找第一个冒号的位置
	colonIndex := -1
	for i, char := range data {
		if char == ':' {
			colonIndex = i
			break
		}
	}

	if colonIndex == -1 {
		logger.Logger.Error("Invalid message format, no colon found: " + data)
		return
	}

	// 提取事件类型
	eventTypeStr := data[:colonIndex]
	payloadStr := data[colonIndex+1:]

	// 转换为 EventType
	eventType := EventType(eventTypeStr)

	ec.mu.RLock()
	handlers, exists := ec.eventHandlers[eventType]
	ec.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		logger.Logger.Debug(fmt.Sprintf("No handlers for event type: %s", eventType))
		return
	}

	// 触发对应事件类型的所有监听函数
	for _, handler := range handlers {
		go ec.safeCallHandler(eventType, handler, payloadStr)
	}
}

// safeCallHandler 安全调用事件处理函数
func (ec *EventCollector) safeCallHandler(eventType EventType, handler interface{}, payloadStr string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Error(fmt.Sprintf("Event handler panic for type %s: %v", eventType, r))
		}
	}()

	switch eventType {
	case EventRelaySwitch:
		if h, ok := handler.(RelaySwitchHandler); ok {
			var payload RelaySwitchPayload
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to unmarshal RelaySwitchPayload: %v", err))
				return
			}
			h(payload)
		}
	case EventRelaySwitchAsk:
		if h, ok := handler.(RelaySwitchAskHandler); ok {
			var payload RelaySwitchAskPayload
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to unmarshal RelaySwitchAskPayload: %v", err))
				return
			}
			h(payload)
		}
	case EventRestartDevice:
		if h, ok := handler.(RestartDeviceHandler); ok {
			var payload RestartDevicePayload
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to unmarshal RestartDevicePayload: %v", err))
				return
			}
			h(payload)
		}

	case EventDeviceAlarm:
		if h, ok := handler.(DeviceAlarmHandler); ok {
			var payload DeviceAlarmPayload
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to unmarshal DeviceAlarmPayload: %v", err))
				return
			}
			h(payload)
		}
	case EventDeviceSimulatorChange:
		if h, ok := handler.(DeviceSimulatorChangeHandler); ok {
			var payload DeviceSimulatorChangePayload
			if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to unmarshal DeviceSimulatorChangePayload: %v", err))
				return
			}
			h(payload)
		}
	}
}

// publishEvent 通用的发布事件方法
func (ec *EventCollector) publishEvent(eventType EventType, payload interface{}) error {
	// 将 payload 序列化为 JSON 字符串
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// 拼接格式：EventType:Payload
	message := string(eventType) + ":" + string(data)

	logger.Logger.Debug(fmt.Sprintf("Publishing to channel %s: %s", channelCollector, message))
	return ec.client.Publish(ec.ctx, channelCollector, message).Err()
}

// PublishRelaySwitch 发布继电器切换事件
func (ec *EventCollector) PublishRelaySwitch(payload RelaySwitchPayload) error {
	return ec.publishEvent(EventRelaySwitch, payload)
}

// PublishRelaySwitchAsk 发布继电器切换ASK事件
func (ec *EventCollector) PublishRelaySwitchAsk(payload RelaySwitchAskPayload) error {
	return ec.publishEvent(EventRelaySwitchAsk, payload)
}

// PublishRestartDevice 发布重启设备事件
func (ec *EventCollector) PublishRestartDevice(payload RestartDevicePayload) error {
	return ec.publishEvent(EventRestartDevice, payload)
}

// PublishDeviceAlarm 发布通用设备告警事件
func (ec *EventCollector) PublishDeviceAlarm(payload DeviceAlarmPayload) error {
	return ec.publishEvent(EventDeviceAlarm, payload)
}

// PublishDeviceSimulatorChange 发布模拟设备变更事件
func (ec *EventCollector) PublishDeviceSimulatorChange(payload DeviceSimulatorChangePayload) error {
	return ec.publishEvent(EventDeviceSimulatorChange, payload)
}

// SubscribeRelaySwitch 订阅继电器切换事件
func (ec *EventCollector) SubscribeRelaySwitch(handler RelaySwitchHandler) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.eventHandlers[EventRelaySwitch] = append(ec.eventHandlers[EventRelaySwitch], handler)
}

// SubscribeRelaySwitchAsk 订阅继电器切换ASK事件
func (ec *EventCollector) SubscribeRelaySwitchAsk(handler RelaySwitchAskHandler) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.eventHandlers[EventRelaySwitchAsk] = append(ec.eventHandlers[EventRelaySwitchAsk], handler)
}

// SubscribeRestartDevice 订阅重启设备事件
func (ec *EventCollector) SubscribeRestartDevice(handler RestartDeviceHandler) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.eventHandlers[EventRestartDevice] = append(ec.eventHandlers[EventRestartDevice], handler)
}

// SubscribeDeviceAlarm 订阅通用设备告警事件
func (ec *EventCollector) SubscribeDeviceAlarm(handler DeviceAlarmHandler) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.eventHandlers[EventDeviceAlarm] = append(ec.eventHandlers[EventDeviceAlarm], handler)
}

// SubscribeDeviceSimulatorChange 订阅模拟设备变更事件
func (ec *EventCollector) SubscribeDeviceSimulatorChange(handler DeviceSimulatorChangeHandler) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.eventHandlers[EventDeviceSimulatorChange] = append(ec.eventHandlers[EventDeviceSimulatorChange], handler)
}

// Close 关闭事件收集器
func (ec *EventCollector) Close() error {
	ec.cancel()

	if ec.redisPubSub != nil {
		if err := ec.redisPubSub.Close(); err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to close redis pubsub: %v", err))
		}
	}

	if ec.client != nil {
		if err := ec.client.Close(); err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to close redis client: %v", err))
			return err
		}
	}

	return nil
}
