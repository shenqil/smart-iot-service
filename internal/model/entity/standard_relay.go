package entity

import "gorm.io/gorm"

// StandardRelay 标准继电器模型
type StandardRelay struct {
	gorm.Model
	Name string `gorm:"size:100;unique;not null;comment:名称"`
	URL  string `gorm:"size:255;comment:URL地址"`
}

// StandardRelayQuery 查询条件
type StandardRelayQuery struct {
	Name       string // 名称搜索参数
	Q          string // 全部搜索参数
	Pagination        // 分页参数
}
