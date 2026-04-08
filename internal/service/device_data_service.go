package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"rbac-admin/internal/cache"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
	"rbac-admin/pkg/pubsubRedis"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// defaultFilterInterval 默认过滤间隔
var defaultFilterInterval int64 = 24 * 60 * 60

// DeviceDataService 设备数据服务
type DeviceDataService struct {
	InfluxClient      *influxdb3.Client
	DeviceDataCache   *cache.DeviceDataCache
	EventCollector    *pubsubRedis.EventCollector
	DeviceRepo        *repository.DeviceRepository
	DeviceConfigCache *cache.DeviceConfigCache
}

// GetRealTimeData 获取设备实时数据（直接从Redis获取）
func (s *DeviceDataService) GetRealTimeData(ctx context.Context, deviceSN string) (*dto.DeviceRealTimeData, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 解析设备ID
	var deviceID uint32
	if _, err := fmt.Sscanf(deviceSN, "%d", &deviceID); err != nil {
		return nil, errx.New(constant.BadRequest, "无效的设备序列号", err)
	}

	// 直接从Redis获取实时数据
	data, err := s.DeviceDataCache.GetData(ctx, deviceID)
	if err != nil {
		logger.Error(fmt.Sprintf("从Redis获取设备实时数据失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "获取设备数据失败", err)
	}

	// 如果没有数据，返回一个只包含当前时间戳的空结构
	if data == nil {
		return &dto.DeviceRealTimeData{
			Timestamp:  time.Now().Unix(),
			SensorData: make(map[string]interface{}),
			RelayData:  make(map[string]dto.RelayData),
		}, nil
	}

	return data, nil
}

// GetSensorHistoryData 获取传感器历史数据
func (s *DeviceDataService) GetSensorHistoryData(
	ctx context.Context,
	filter dto.SensorHistoryFilter,
) ([]map[string]interface{}, error) {
	// 验证参数
	if err := filter.Validate(); err != nil {
		return nil, errx.New(constant.BadRequest, err.Error(), nil)
	}

	// 解析设备ID
	var deviceID uint32
	if _, err := fmt.Sscanf(filter.DeviceSN, "%d", &deviceID); err != nil {
		return nil, errx.New(constant.BadRequest, "无效的设备序列号", err)
	}

	// 使用最新数据拉取最近24小时数据
	if filter.StartTime == 0 || filter.EndTime == 0 {
		realData, _ := s.DeviceDataCache.GetData(ctx, deviceID)
		if realData == nil {
			filter.EndTime = time.Now().Unix()
		} else {
			filter.EndTime = realData.Timestamp
		}
		filter.StartTime = filter.EndTime - defaultFilterInterval

	}

	// 计算时间范围（秒）
	timeRange := filter.EndTime - filter.StartTime
	oneDaySeconds := int64(24 * 60 * 60)

	// 如果时间范围 <= 1天，直接查询（使用左闭右开区间）
	if timeRange <= oneDaySeconds {
		return s.queryHistoryData(ctx, filter)
	}

	// 超过1天，按24小时拆分查询
	var allResults []map[string]interface{}
	currentStart := filter.StartTime

	for currentStart < filter.EndTime {
		// 计算当前查询的结束时间（24小时后，但不超出总结束时间）
		currentEnd := currentStart + oneDaySeconds
		if currentEnd > filter.EndTime {
			currentEnd = filter.EndTime
		}

		// 统一使用左闭右开区间 [start, end)，避免数据重合
		// 第一个时间段：[currentStart, currentEnd)
		// 后续时间段：[currentStart, currentEnd)，其中 currentStart = 上一个的 currentEnd
		queryFilter := filter
		queryFilter.StartTime = currentStart
		queryFilter.EndTime = currentEnd

		results, err := s.queryHistoryData(ctx, queryFilter)
		if err != nil {
			return nil, err
		}

		// 合并结果
		allResults = append(allResults, results...)

		// 移动到下一个时间段
		currentStart = currentEnd
	}

	return allResults, nil
}

// queryHistoryData 查询指定时间范围的历史数据
// 统一使用左闭右开区间 [start, end)，使用 filter 中的 StartTime 和 EndTime
func (s *DeviceDataService) queryHistoryData(
	ctx context.Context,
	filter dto.SensorHistoryFilter,
) ([]map[string]interface{}, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 构建查询
	query, err := s.buildOptimizedHistoryQuery(ctx, filter)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	if query != "" {
		logger.Info(fmt.Sprintf("执行查询 [%d - %d]: %s", filter.StartTime, filter.EndTime, query))

		// 执行查询
		iterator, err := s.InfluxClient.Query(ctx, query)
		if err != nil {
			logger.Error(fmt.Sprintf("查询传感器历史数据失败: %v", err))
			return nil, errx.New(constant.InternalServerError, "查询传感器历史数据失败", err)
		}

		// 处理结果
		for iterator.Next() {
			results = append(results, iterator.Value())
		}
	}

	return results, nil
}

// getSensorColumns 获取传感器数据的数字列名
func (s *DeviceDataService) getSensorColumns(ctx context.Context, deviceSN string) ([]string, error) {

	// 解析设备ID
	var deviceID uint
	if _, err := fmt.Sscanf(deviceSN, "%d", &deviceID); err != nil {
		return nil, errx.New(constant.BadRequest, "无效的设备序列号", err)
	}

	// 从DeviceConfigCache获取设备配置信息
	if s.DeviceConfigCache == nil {
		return nil, errx.New(constant.InternalServerError, "设备配置缓存服务未初始化", nil)
	}
	device, err := s.DeviceConfigCache.GetConfig(ctx, uint32(deviceID))
	if err != nil {
		return nil, err
	}
	if device == nil {
		return nil, errx.New(constant.NotFound, "未查询到传感器信息", nil)
	}

	var sensors []dto.SensorConfig
	if device.Sensors != nil {
		_ = dto.FromJSON(device.Sensors, &sensors)
	}

	columns := make([]string, 0, 32)
	for _, sensor := range sensors {
		if sensor.StandardID != nil && *sensor.StandardID > 0 {
			columns = append(columns, fmt.Sprintf("S-%d", sensor.ChannelNum))
		}
	}

	return columns, nil
}

// buildOptimizedHistoryQuery 构建优化的历史数据查询
// 统一使用左闭右开区间 [start, end)，使用 filter 中的 StartTime 和 EndTime
func (s *DeviceDataService) buildOptimizedHistoryQuery(ctx context.Context, filter dto.SensorHistoryFilter) (string, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	start := time.Unix(filter.StartTime, 0).UTC().Format(time.RFC3339)
	end := time.Unix(filter.EndTime, 0).UTC().Format(time.RFC3339)

	// 统一使用左闭右开区间 [start, end)
	timeCondition := fmt.Sprintf("time >= '%s' AND time < '%s'", start, end)

	isInterval := filter.Interval != nil && *filter.Interval > 0

	// 1) 列选择：优先使用传入列，否则按设备配置推断
	columns := filter.Columns
	if len(columns) == 0 {
		cls, err := s.getSensorColumns(ctx, filter.DeviceSN)
		if err != nil {
			logger.Error(fmt.Sprintf("获取传感器列失败: %v", err))
			return "", errx.New(constant.InternalServerError, "获取传感器列失败", err)
		}
		columns = cls
	}
	if len(columns) == 0 {
		logger.Warn("无传感器列")
		return "", nil
	}

	// 2) 构建选择器
	selectors := make([]string, len(columns))
	for i, c := range columns {
		if isInterval {
			selectors[i] = fmt.Sprintf(`selector_first("%s", time)['value'] AS "%s"`, c, c)
		} else {
			selectors[i] = fmt.Sprintf(`"%s"`, c)
		}
	}
	// 连接选择器
	sensorSelectors := ""
	for i, v := range selectors {
		if i > 0 {
			sensorSelectors += ", "
		}
		sensorSelectors += v
	}

	// 3) 生成 SQL
	var sql string
	if isInterval {
		sql = fmt.Sprintf(`
SELECT
  date_bin(INTERVAL '%d minutes', time) AS bin_time,
  to_unixtime(MIN(time)) AS time_s,
  device_sn,
  %s
FROM sensor_data
WHERE device_sn = '%s'
  AND %s
GROUP BY bin_time, device_sn
ORDER BY time_s ASC`, *filter.Interval, sensorSelectors, filter.DeviceSN, timeCondition)
	} else {
		sql = fmt.Sprintf(`
SELECT
  to_unixtime(time) AS time_s,
  device_sn,
  %s
FROM sensor_data
WHERE device_sn = '%s'
  AND %s
ORDER BY time ASC`, sensorSelectors, filter.DeviceSN, timeCondition)
	}

	return sql, nil
}

// GetSensorComparison 获取传感器日环比/月环比数据
func (s *DeviceDataService) GetSensorComparison(
	ctx context.Context,
	filter dto.SensorComparisonFilter,
) (*dto.SensorComparisonResult, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 设置上海时区
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return nil, errx.New(constant.InternalServerError, "加载时区失败", err)
	}

	baseTime := time.Unix(filter.Timestamp/1000, 0).In(loc)

	var currentStart, currentEnd, compareStart, compareEnd time.Time

	switch filter.Type {
	case dto.ComparisonTypeDay:
		now := time.Now().In(loc)
		currentStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		currentEnd = currentStart.Add(24 * time.Hour)
		compareStart = time.Date(baseTime.Year(), baseTime.Month(), baseTime.Day(), 0, 0, 0, 0, loc)
		compareEnd = compareStart.Add(24 * time.Hour)

		curr, err := s.getComparisonDataHourly(ctx, filter.DeviceSN, currentStart, currentEnd, filter.Columns[0])
		if err != nil {
			logger.Error(fmt.Sprintf("获取当前周期数据失败: %v", err))
			return nil, err
		}
		comp, err := s.getComparisonDataHourly(ctx, filter.DeviceSN, compareStart, compareEnd, filter.Columns[0])
		if err != nil {
			logger.Error(fmt.Sprintf("获取对比周期数据失败: %v", err))
			return nil, err
		}
		return &dto.SensorComparisonResult{CurrentData: curr, CompareData: comp}, nil

	case dto.ComparisonTypeMonth:
		now := time.Now().In(loc)
		currentStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		currentEnd = currentStart.AddDate(0, 1, 0)
		c := baseTime
		compareStart = time.Date(c.Year(), c.Month(), 1, 0, 0, 0, 0, loc)
		compareEnd = compareStart.AddDate(0, 1, 0)

		curr, err := s.getComparisonDataDaily(ctx, filter.DeviceSN, currentStart, currentEnd, filter.Columns[0])
		if err != nil {
			logger.Error(fmt.Sprintf("获取当前周期数据失败: %v", err))
			return nil, err
		}
		comp, err := s.getComparisonDataDaily(ctx, filter.DeviceSN, compareStart, compareEnd, filter.Columns[0])
		if err != nil {
			logger.Error(fmt.Sprintf("获取对比周期数据失败: %v", err))
			return nil, err
		}
		return &dto.SensorComparisonResult{CurrentData: curr, CompareData: comp}, nil

	default:
		return nil, errx.New(constant.BadRequest, "无效的对比类型", nil)
	}
}

func (s *DeviceDataService) getComparisonDataHourly(
	ctx context.Context,
	deviceSN string,
	start, end time.Time,
	column string,
) ([]map[string]interface{}, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// 每小时仅取第一条数据
	query := fmt.Sprintf(`
SELECT
  date_bin(INTERVAL '1 hour', time) AS bin_time,
  to_unixtime(selector_first("%[1]s", time)['time']) AS time_s,
  device_sn,
  selector_first("%[1]s", time)['value'] AS "%[1]s"
FROM sensor_data
WHERE device_sn = '%[2]s'
  AND time BETWEEN '%[3]s' AND '%[4]s'
GROUP BY bin_time, device_sn
ORDER BY time_s ASC
`, column, deviceSN, start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))

	logger.Info("优化的小时对比查询", slog.String("query", query))

	iter, err := s.InfluxClient.Query(ctx, query)
	if err != nil {
		logger.Error(fmt.Sprintf("查询传感器按小时对比数据失败: %v", err))
		return nil, errx.New(constant.InternalServerError, "查询传感器对比数据失败", err)
	}

	var results []map[string]interface{}
	for iter.Next() {
		results = append(results, iter.Value())
	}
	return results, nil
}

func (s *DeviceDataService) getComparisonDataDaily(
	ctx context.Context,
	deviceSN string,
	start, end time.Time,
	column string,
) ([]map[string]interface{}, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	var allResults []map[string]interface{}

	// 设置上海时区
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return nil, errx.New(constant.InternalServerError, "加载时区失败", err)
	}

	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		// 调整为上海时区的中午12点到18点
		dayStart := time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, loc)
		dayEnd := time.Date(day.Year(), day.Month(), day.Day(), 18, 0, 0, 0, loc)

		query := fmt.Sprintf(`
SELECT
  to_unixtime(time) AS time_s,
  device_sn,
  "%[1]s"
FROM sensor_data
WHERE device_sn = '%[2]s'
  AND time >= '%[3]s'
  AND time <= '%[4]s'
ORDER BY time_s ASC
LIMIT 1
`, column, deviceSN, dayStart.UTC().Format(time.RFC3339Nano), dayEnd.UTC().Format(time.RFC3339Nano))

		logger.Info("月度中午12点数据查询", slog.String("query", query))

		iter, err := s.InfluxClient.Query(ctx, query)
		if err != nil {
			logger.Error(fmt.Sprintf("查询传感器按天对比数据失败: %v", err))
			return nil, errx.New(constant.InternalServerError, "查询传感器对比数据失败", err)
		}
		if iter.Next() {
			allResults = append(allResults, iter.Value())
		}
	}
	return allResults, nil
}
