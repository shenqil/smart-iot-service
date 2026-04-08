package storage

import (
	"context" // Added import
	"fmt"
	"math"
	"math/rand"
	"rbac-admin/internal/cache"
	"rbac-admin/internal/model/dto"
	"rbac-admin/pkg/collector/singletons"
	"rbac-admin/pkg/logger"
	"rbac-admin/pkg/pubsubRedis"
	"time" // Added import

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// DataProcessor 负责处理设备数据，包括解析、告警检查和写入 InfluxDB。
// 它协调 DeviceConfigProvider, AlertManager 和 InfluxWriter。
type DataProcessor struct {
	configProvider *DeviceConfigProvider
	alertManager   *AlertManager
	influxWriter   *InfluxWriter
	deviceDataCache *cache.DeviceDataCache // Used for caching real-time data
	eventPublisher *pubsubRedis.EventCollector // Used for publishing alerts
}

// NewDataProcessor 创建一个新的 DataProcessor 实例。
func NewDataProcessor(
	configProvider *DeviceConfigProvider,
	alertManager *AlertManager,
	influxWriter *InfluxWriter,
	deviceDataCache *cache.DeviceDataCache,
	eventPublisher *pubsubRedis.EventCollector,
) *DataProcessor {
	return &DataProcessor{
		configProvider: configProvider,
		alertManager:   alertManager,
		influxWriter:   influxWriter,
		deviceDataCache: deviceDataCache,
		eventPublisher: eventPublisher,
	}
}

// InsertDeviceData 插入设备数据到时序数据库，这是 DataProcessor 的核心入口。
func (dp *DataProcessor) InsertDeviceData(data *singletons.DeviceData) error {
	if dp.influxWriter == nil {
		return fmt.Errorf("InfluxWriter 未初始化")
	}

	deviceData := data

	// 验证数据
	if len(deviceData.Data) != 64 {
		logger.Logger.Error(fmt.Sprintf("数据长度错误 %d-%d-%d", deviceData.ID, deviceData.Type, len(deviceData.Data)))
		return fmt.Errorf("数据长度错误")
	}

	// 获取设备配置
	config, err := dp.configProvider.LoadAndCacheDeviceConfig(deviceData.ID)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("获取设备配置失败 %d: %v", deviceData.ID, err))
		return err
	}

	timestamp := deviceData.LastSeen.UTC().Truncate(time.Second)

	// 处理传感器数据
	sensorFields := dp.processSensorData(deviceData, config)
	if len(sensorFields) > 0 {
		tags := map[string]string{"device_sn": fmt.Sprintf("%d", deviceData.ID)}
		sensorPoint := influxdb3.NewPoint("sensor_data", tags, sensorFields, timestamp)
		dp.influxWriter.AddDataPoint(sensorPoint)
	}

	// 处理继电器状态
	relayFields := make(map[string]dto.RelayData)
	relayOffset := uint(32) - 1
	for _, relay := range config.Relays {
		if relay.StandardID == nil || relay.ChannelNum < 1 || relay.ChannelNum > 32 {
			continue
		}

		value := deviceData.Data[relayOffset+relay.ChannelNum]
		key := fmt.Sprintf("%d", relay.ChannelNum)
		relayFields[key] = dto.RelayData{
			Value:     value,
			Timestamp: timestamp.Unix(),
		}
	}

	// 更新Redis缓存 (实时数据)
	if dp.deviceDataCache != nil {
		cacheData := &dto.DeviceRealTimeData{
			SensorData: sensorFields,
			RelayData:  relayFields,
			Timestamp:  timestamp.Unix(),
		}

		// context.Background() is needed here for cache operations
		if err := dp.deviceDataCache.SetData(context.Background(), deviceData.ID, cacheData); err != nil {
			logger.Logger.Error(fmt.Sprintf("Redis实时数据缓存更新失败: %v", err))
		}
	}

	return nil
}

// LoadAndCacheDeviceConfig 暴露配置提供者的加载方法。
func (dp *DataProcessor) LoadAndCacheDeviceConfig(deviceID uint32) (*DeviceConfig, error) {
	return dp.configProvider.LoadAndCacheDeviceConfig(deviceID)
}

// RemoveDeviceConfig 暴露配置提供者的移除方法。
func (dp *DataProcessor) RemoveDeviceConfig(deviceID uint32) {
	dp.configProvider.RemoveDeviceConfig(deviceID)
}

// Close 关闭 DataProcessor 及其内部组件。
func (dp *DataProcessor) Close() error {
	var errs []error

	if dp.influxWriter != nil {
		if err := dp.influxWriter.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭 InfluxWriter 失败: %w", err))
		}
	}
	if dp.configProvider != nil {
		if err := dp.configProvider.Close(); err != nil {
			errs = append(errs, fmt.Errorf("关闭 DeviceConfigProvider 失败: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("关闭 DataProcessor 失败: %v", errs)
	}
	logger.Logger.Info("DataProcessor 已关闭")
	return nil
}

// processSensorData 处理传感器数据。
func (dp *DataProcessor) processSensorData(data *singletons.DeviceData, config *DeviceConfig) map[string]interface{} {
	sensorFields := make(map[string]interface{})

	for _, sensor := range config.Sensors {
		if sensor.StandardID == nil || *sensor.StandardID == 0 || sensor.ChannelNum < 1 || sensor.ChannelNum > 32 {
			continue
		}

		raw := float64(0)
		offset := (sensor.ChannelNum - 1) * 2
		raw = float64(int16(data.Data[offset])<<8 | int16(data.Data[offset+1]))

		processedValue := dp.parseData(raw, &sensor)
		sensorFields[fmt.Sprintf("S-%d", sensor.ChannelNum)] = processedValue

		// 检查传感器告警
		dp.alertManager.CheckSensorAlerts(data.ID, config.CompanyID, config.DeviceName, &sensor, processedValue)
	}

	return sensorFields
}

// parseData 解析传感器数据。
func (dp *DataProcessor) parseData(raw float64, sensor *dto.SensorConfig) float64 {
	// 1. 解析出原始数据
	switch sensor.Resolution {
	case 0.1:
		raw = raw / 10
	case 0.01:
		raw = raw / 100
	case 0.001:
		raw = raw / 1000
	case 0.0001:
		raw = raw / 10000
	}

	// 2. 纠正数据
	if sensor.Correction != nil {
		beforeCorrection := raw
		switch sensor.Correction.Mode {
		case "coefficient": // 数据*系数+偏差值
			raw = raw*sensor.Correction.CoefficientValue + sensor.Correction.ErrorValue
			logger.Logger.Debug(fmt.Sprintf("数据矫正 [通道#%d-%s] 模式=%s, 矫正前=%.4f, 系数=%.4f, 偏差值=%.4f, 矫正后=%.4f",
				sensor.ChannelNum, sensor.DisplayName, sensor.Correction.Mode, beforeCorrection,
				sensor.Correction.CoefficientValue, sensor.Correction.ErrorValue, raw))
		case "custom": // 使用自定义值+随机值
			raw = sensor.Correction.CustomValue + rand.Float64()*sensor.Correction.FloatValue
			logger.Logger.Debug(fmt.Sprintf("数据矫正 [通道#%d-%s] 模式=%s, 矫正前=%.4f, 自定义值=%.4f, 随机范围=%.4f, 矫正后=%.4f",
				sensor.ChannelNum, sensor.DisplayName, sensor.Correction.Mode, beforeCorrection,
				sensor.Correction.CustomValue, sensor.Correction.FloatValue, raw))
		}

		// 3. 根据Resolution保留精度
		switch sensor.Resolution {
		case 0.1:
			raw = math.Round(raw*10) / 10
		case 0.01:
			raw = math.Round(raw*100) / 100
		case 0.001:
			raw = math.Round(raw*1000) / 1000
		case 0.0001:
			raw = math.Round(raw*10000) / 10000
		}
	}

	return raw
}
