package entity

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// DeviceType 设备类型模型
type DeviceType struct {
	gorm.Model
	Type        string         `gorm:"size:50;uniqueIndex;not null;comment:类型标识"`
	Name        string         `gorm:"size:100;not null;comment:类型名称"`
	Description string         `gorm:"type:text;comment:描述"`
	Icon        string         `gorm:"size:255;comment:图标URL"`
	Schema      datatypes.JSON `gorm:"type:json;comment:JSON Schema"`
}

// DeviceTypeQuery 查询条件
type DeviceTypeQuery struct {
	Name string // 名称模糊匹配
	Q    string // 全部搜索参数
	Pagination   // 分页参数
}
