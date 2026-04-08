package storage

import (
	"fmt"
	"rbac-admin/internal/model/dto"
	"rbac-admin/pkg/logger"
	"rbac-admin/pkg/pubsubRedis"
	"sync"
	"time"
)

// AlertManager 负责管理传感器告警逻辑，包括告警规则检查、重复告警控制和事件发布。
type AlertManager struct {
	eventPublisher *pubsubRedis.EventCollector
	// alertCooldowns 记录每个告警的上次触发时间：map[deviceID]map[channelNum]map[alertIndex]lastAlertTime
	alertCooldowns map[uint32]map[uint]map[int]int64
	mu             sync.RWMutex // 保护 alertCooldowns
}

// NewAlertManager 创建一个新的 AlertManager 实例。
func NewAlertManager(eventPublisher *pubsubRedis.EventCollector) *AlertManager {
	return &AlertManager{
		eventPublisher: eventPublisher,
		alertCooldowns: make(map[uint32]map[uint]map[int]int64),
	}
}

// CheckSensorAlerts 检查传感器值是否触发告警，并处理告警事件发布和冷却。
func (am *AlertManager) CheckSensorAlerts(
	deviceID uint32,
	companyID uint,
	deviceName string,
	sensor *dto.SensorConfig,
	value float64,
) {
	if am.eventPublisher == nil {
		return
	}

	if len(sensor.AlertConfigs) == 0 {
		return
	}

	for alertIndex, alertConfig := range sensor.AlertConfigs {
		if !alertConfig.Enabled {
			continue
		}

		// 检查是否触发告警
		triggered := alertConfig.IsAlertTriggered(value)
		if !triggered {
			continue
		}

		// 检查重复告警间隔
		if alertConfig.RepeatAlert {
			shouldSend := am.shouldSendRepeatAlert(deviceID, sensor.ChannelNum, alertIndex, alertConfig.RepeatInterval)
			if !shouldSend {
				continue
			}
		}

		// 发布通用设备告警事件（对齐 DeviceAlarmPayload）
		thresholdStr := fmt.Sprintf("%v", alertConfig.Threshold)
		if alertConfig.AlertType == "range" {
			thresholdStr = fmt.Sprintf("%v~%v", alertConfig.MinValue, alertConfig.MaxValue)
		}
		var ruleSentence string
		switch alertConfig.AlertType {
		case "greater":
			ruleSentence = fmt.Sprintf("当前值为 %v，大于限定值 %s", value, thresholdStr)
		case "less":
			ruleSentence = fmt.Sprintf("当前值为 %v，小于限定值 %s", value, thresholdStr)
		case "equal":
			ruleSentence = fmt.Sprintf("当前值为 %v，等于限定值 %s", value, thresholdStr)
		case "range":
			ruleSentence = fmt.Sprintf("当前值为 %v，处于限定范围 %s", value, thresholdStr)
		default:
			ruleSentence = fmt.Sprintf("当前值为 %v，触发规则(%s)，限定值 %s", value, alertConfig.AlertType, thresholdStr)
		}
		desc := fmt.Sprintf("通道#%d（%s）%s", sensor.ChannelNum, sensor.DisplayName, ruleSentence)
		payload := pubsubRedis.DeviceAlarmPayload{
			AlarmLevel:   3, // Warning
			AlarmType:    "threshold_breach",
			AlarmSubType: alertConfig.AlertType,
			EnterpriseId: fmt.Sprint(companyID),
			DeviceId:     fmt.Sprint(deviceID),
			DeviceName:   deviceName,
			Description:  desc,
			Value:        fmt.Sprintf("%v", value),
			Threshold:    thresholdStr,
			Timestamp:    time.Now().Unix(),
		}

		logger.Logger.Info(fmt.Sprintf("告警触发: 设备ID=%d(%s), 通道#%d(%s)), 值=%.4f, 类型=%s",
			deviceID, deviceName, sensor.ChannelNum, sensor.DisplayName, value, alertConfig.AlertType))

		if err := am.eventPublisher.PublishDeviceAlarm(payload); err != nil {
			logger.Logger.Error(fmt.Sprintf("发布告警事件失败: 设备ID=%d, 通道#%d, 告警索引=%d, 错误=%v", deviceID, sensor.ChannelNum, alertIndex, err))
		}

		// 记录告警时间（用于重复告警控制）
		if alertConfig.RepeatAlert {
			am.recordAlertTime(deviceID, sensor.ChannelNum, alertIndex)
		}
	}
}

// shouldSendRepeatAlert 检查是否应该发送重复告警。
func (am *AlertManager) shouldSendRepeatAlert(deviceID uint32, channelNum uint, alertIndex int, repeatInterval int) bool {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if _, ok := am.alertCooldowns[deviceID]; !ok {
		return true
	}
	if _, ok := am.alertCooldowns[deviceID][channelNum]; !ok {
		return true
	}
	lastAlertTime, exists := am.alertCooldowns[deviceID][channelNum][alertIndex]
	if !exists {
		return true
	}

	currentTime := time.Now().Unix()
	return currentTime-lastAlertTime >= int64(repeatInterval)
}

// recordAlertTime 记录告警时间，用于重复告警控制。
func (am *AlertManager) recordAlertTime(deviceID uint32, channelNum uint, alertIndex int) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, ok := am.alertCooldowns[deviceID]; !ok {
		am.alertCooldowns[deviceID] = make(map[uint]map[int]int64)
	}
	if _, ok := am.alertCooldowns[deviceID][channelNum]; !ok {
		am.alertCooldowns[deviceID][channelNum] = make(map[int]int64)
	}

	am.alertCooldowns[deviceID][channelNum][alertIndex] = time.Now().Unix()
}
