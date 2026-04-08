package dto

import (
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/errx"
)

type ComparisonType string

const (
	ComparisonTypeDay   ComparisonType = "day"
	ComparisonTypeMonth ComparisonType = "month"
)

type RelayData struct {
	Value     byte  `json:"value"`
	Timestamp int64 `json:"timestamp"` // 数据时间(秒级时间戳)
}

type DeviceRealTimeData struct {
	SensorData map[string]interface{} `json:"sensor_data"`
	RelayData  map[string]RelayData   `json:"relay_data"`
	Timestamp  int64                  `json:"timestamp"` // 数据时间(秒级时间戳)
}

type SensorHistoryFilter struct {
	DeviceSN   string             `form:"deviceSN" binding:"required"` // 设备序列号
	StartTime  int64              `form:"startTime,omitempty"`         // 开始时间(秒级时间戳)
	EndTime    int64              `form:"endTime,omitempty"`           // 结束时间(秒级时间戳)
	Columns    []string           `form:"columns,omitempty"`           // 需要查询的传感器列名(可选)
	Interval   *int               `form:"interval,omitempty"`          // 时间间隔(分钟)
	Pagination *PaginationRequest `form:"pagination,omitempty"`        // 分页参数（可选）
}

// Validate 验证过滤器参数
func (f *SensorHistoryFilter) Validate() error {
	if f.DeviceSN == "" {
		return errx.New(constant.BadRequest, "设备序列号不能为空", nil)
	}
	// 当同时提供时间范围时进行校验（允许为空以便服务层回填默认值）
	if f.StartTime != 0 && f.EndTime != 0 {
		if f.StartTime > f.EndTime {
			return errx.New(constant.BadRequest, "开始时间不能大于结束时间", nil)
		}
		const maxRange int64 = 31 * 24 * 60 * 60
		if f.EndTime-f.StartTime > maxRange {
			return errx.New(constant.BadRequest, "查询时间范围不能超过31天", nil)
		}
	}
	return nil
}

type RelayOperationsFilter struct {
	DeviceSN     string             `form:"deviceSN" binding:"required"` // 设备序列号
	RelayChannel *int               `form:"relayChannel,omitempty"`      // 继电器通道(可选)
	StartTime    int64              `form:"startTime,omitempty" `        // 开始时间(秒级时间戳)
	EndTime      int64              `form:"endTime,omitempty" `          // 结束时间(秒级时间戳)
	Pagination   *PaginationRequest `form:"pagination,omitempty"`        // 分页参数(可选)
}

// Validate 验证继电器操作过滤器
func (f *RelayOperationsFilter) Validate() error {
	if f.DeviceSN == "" {
		return errx.New(constant.BadRequest, "设备序列号不能为空", nil)
	}
	if f.StartTime != 0 && f.EndTime != 0 {
		if f.StartTime > f.EndTime {
			return errx.New(constant.BadRequest, "开始时间不能大于结束时间", nil)
		}
		const maxRange int64 = 30 * 24 * 60 * 60
		if f.EndTime-f.StartTime > maxRange {
			return errx.New(constant.BadRequest, "查询时间范围不能超过30天", nil)
		}
	}
	return nil
}

type SensorComparisonFilter struct {
	DeviceSN  string         `form:"deviceSN" binding:"required"`  // 设备序列号
	Type      ComparisonType `form:"type" binding:"required"`      // 对比类型: day/month
	Timestamp int64          `form:"timestamp" binding:"required"` // 基准时间戳(毫秒级)
	Columns   []string       `form:"columns" binding:"required"`   // 需要对比的传感器列名
}

// Validate 验证对比过滤器
func (f *SensorComparisonFilter) Validate() error {
	if f.DeviceSN == "" {
		return errx.New(constant.BadRequest, "设备序列号不能为空", nil)
	}
	if f.Type != ComparisonTypeDay && f.Type != ComparisonTypeMonth {
		return errx.New(constant.BadRequest, "无效的对比类型", nil)
	}
	if f.Timestamp <= 0 {
		return errx.New(constant.BadRequest, "无效的时间戳", nil)
	}
	if len(f.Columns) == 0 {
		return errx.New(constant.BadRequest, "列名不能为空", nil)
	}
	return nil
}

type SensorComparisonResult struct {
	CurrentData []map[string]interface{} `json:"currentData"` // 当前周期数据
	CompareData []map[string]interface{} `json:"compareData"` // 对比周期数据
}
