package entity

import "gorm.io/gorm"

// StandardSensor 标准传感器模型
type StandardSensor struct {
	gorm.Model

	Name       string  `gorm:"size:100;unique;not null;comment:传感器名称"`
	MinValue   float64 `gorm:"not null;comment:最小值"`
	MaxValue   float64 `gorm:"not null;comment:最大值"`
	Resolution float64 `gorm:"not null;comment:分辨率"`
	Unit       string  `gorm:"size:20;not null;comment:单位"`
	URL        string  `gorm:"size:255;comment:URL地址"`
}

// StandardSensorQuery 查询条件
type StandardSensorQuery struct {
	Name       string // 名称搜索参数
	Q          string // 全部搜索参数
	Pagination        // 分页参数
}
