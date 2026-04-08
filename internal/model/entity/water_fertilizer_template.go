package entity

import "gorm.io/gorm"

// WaterFertilizerTemplate 水肥模板模型
type WaterFertilizerTemplate struct {
	gorm.Model

	Name     string `gorm:"size:100;not null;comment:模板名称"`
	Template string `gorm:"type:text;not null;comment:模板内容(JSON字符串)"`
}

// WaterFertilizerTemplateQuery 查询条件
type WaterFertilizerTemplateQuery struct {
	Q          string // 模糊匹配Name
	Pagination        // 分页参数
}
