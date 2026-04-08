package singletons

import (
	"context"
	"fmt"
	"rbac-admin/pkg/logger"
	"sync"
	"time"
)

// EventType 事件类型
type EventType int

const (
	EventData        EventType = iota // 数据事件
	EventStatus                       // 状态事件
	EventRelayStatus                  // 继电器状态事件
)

// Event 事件结构
type Event struct {
	Type EventType
	Data interface{}
}

// DeviceData 设备数据结构
type DeviceData struct {
	ID       uint32    `json:"id"`        // 设备ID
	Type     uint32    `json:"type"`      // 数据包类型
	LastSeen time.Time `json:"last_seen"` // 最后通信时间
	Data     []byte    `json:"data"`      // 数据
}

// DeviceStatus 设备状态结构
type DeviceStatus struct {
	ID           uint32    `json:"id"`           // 设备ID
	Online       byte      `json:"online"`       // 在线状态
	Time         time.Time `json:"time"`         // 状态时间
	ConnectionID string    `json:"connectionId"` // 连接唯一ID（使用设备地址）
}

// RelayStatus 继电器状态结构
type RelayStatus struct {
	ID       uint32 `json:"id"`       // 设备ID
	RelayNum uint32 `json:"relayNum"` // (0-31)
	State    byte   `json:"state"`
}

// EventPublisher 事件发布器单例
type EventPublisher struct {
	subscribers map[string]func(Event)
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	eventCount  int64
}

var (
	eventPublisherInstance *EventPublisher
	eventPublisherOnce     sync.Once
)

// GetEventPublisher 获取事件发布器单例
func GetEventPublisher() *EventPublisher {
	eventPublisherOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		eventPublisherInstance = &EventPublisher{
			subscribers: make(map[string]func(Event)),
			ctx:         ctx,
			cancel:      cancel,
		}
	})
	return eventPublisherInstance
}

// Publish 发布事件（核心发布函数）
func (ep *EventPublisher) Publish(event Event) {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	// 统计事件数量
	ep.eventCount++

	// 发送给所有订阅者
	for subscriberID, callback := range ep.subscribers {
		select {
		case <-ep.ctx.Done():
			// 上下文已取消，停止发送
			return
		default:
			// 异步调用回调函数
			go func(id string, cb func(Event)) {
				defer func() {
					if r := recover(); r != nil {
						logger.Logger.Error(fmt.Sprintf("订阅者 %s 回调函数执行出错: %v", id, r))
					}
				}()
				cb(event)
			}(subscriberID, callback)
		}
	}
}

// Subscribe 订阅所有事件（回调函数模式）
func (ep *EventPublisher) Subscribe(callback func(Event)) string {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// 生成唯一的订阅者ID
	subscriberID := generateSubscriberID()

	// 注册新的订阅者
	ep.subscribers[subscriberID] = callback

	logger.Logger.Info(fmt.Sprintf("订阅者 %s 已订阅所有事件", subscriberID))
	return subscriberID
}

// SubscribeEventType 订阅指定类型的事件
func (ep *EventPublisher) SubscribeEventType(eventType EventType, callback func(Event)) string {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// 生成唯一的订阅者ID
	subscriberID := generateSubscriberID()

	// 创建带过滤器的回调函数
	filteredCallback := func(event Event) {
		if event.Type == eventType {
			callback(event)
		}
	}

	// 注册新的订阅者
	ep.subscribers[subscriberID] = filteredCallback

	logger.Logger.Info(fmt.Sprintf("订阅者 %s 已订阅事件类型 %d", subscriberID, eventType))
	return subscriberID
}

// Unsubscribe 取消订阅
func (ep *EventPublisher) Unsubscribe(subscriberID string) bool {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if _, exists := ep.subscribers[subscriberID]; exists {
		delete(ep.subscribers, subscriberID)
		logger.Logger.Info(fmt.Sprintf("订阅者 %s 已取消订阅", subscriberID))
		return true
	}
	return false
}

// PublishDeviceData 发布设备数据事件（快捷函数）
func (ep *EventPublisher) PublishDeviceData(deviceData *DeviceData) {
	event := Event{
		Type: EventData,
		Data: deviceData,
	}
	ep.Publish(event)
}

// PublishDeviceStatus 发布设备状态事件（快捷函数）
func (ep *EventPublisher) PublishDeviceStatus(deviceStatus *DeviceStatus) {
	event := Event{
		Type: EventStatus,
		Data: deviceStatus,
	}
	ep.Publish(event)
}

// PublishRelayStatus 发布继电器状态事件（快捷函数）
func (ep *EventPublisher) PublishRelayStatus(relayStatus *RelayStatus) {
	event := Event{
		Type: EventRelayStatus,
		Data: relayStatus,
	}
	ep.Publish(event)
}

// Close 关闭事件发布器
func (ep *EventPublisher) Close() {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	// 取消上下文
	ep.cancel()

	// 清空所有订阅者
	ep.subscribers = make(map[string]func(Event))

	logger.Logger.Info("事件发布器已关闭")
}

// generateSubscriberID 生成唯一的订阅者ID
func generateSubscriberID() string {
	return time.Now().Format("20060102150405") + "-" + time.Now().Format("000000000")
}
