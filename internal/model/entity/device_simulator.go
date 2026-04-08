package entity

import (
	"gorm.io/gorm"
)

// DeviceSimulator 设备模拟器表
type DeviceSimulator struct {
	gorm.Model
	DeviceSN string `gorm:"size:100;uniqueIndex:uidx_device_sn,where:deleted_at IS NULL;not null;comment:设备SN号"`
}

// DeviceSimulatorQuery 设备模拟器查询条件
type DeviceSimulatorQuery struct {
	DeviceSN   *string // 设备SN号模糊搜索
	Q          *string // 综合查询条件，模糊匹配设备SN号
	Pagination         // 分页参数
}
