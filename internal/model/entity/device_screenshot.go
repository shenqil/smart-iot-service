package entity

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// DeviceScreenshot 截图记录实体
type DeviceScreenshot struct {
	gorm.Model

	// 设备序列号（8位纯数字）
	DeviceSN uint32 `gorm:"type:int unsigned;not null;index;comment:设备序列号"`

	// 截图时间
	CaptureTime time.Time `gorm:"type:timestamp;not null;comment:截图时间"`

	// OSS 图片地址
	ImageURL string `gorm:"type:varchar(255);not null;comment:截图存储URL"`

	// 扩展信息（JSON，留给用户自由扩展）
	Extra datatypes.JSON `gorm:"type:json;comment:用户扩展字段"`
}

// DeviceScreenshotQuery 查询条件结构体
type DeviceScreenshotQuery struct {
	DeviceSN   *uint32    // 设备序列号
	StartTime  *time.Time // 开始时间
	EndTime    *time.Time // 结束时间
	Pagination            // 分页参数
}
