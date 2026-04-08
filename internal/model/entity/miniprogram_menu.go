package entity

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// MiniProgramMenuStatus 小程序功能栏状态枚举
type MiniProgramMenuStatus int8

const (
	MiniProgramMenuStatusDisabled MiniProgramMenuStatus = 0 // 禁用
	MiniProgramMenuStatusEnabled  MiniProgramMenuStatus = 1 // 启用
)

// MiniProgramMenu 小程序功能栏配置模型
type MiniProgramMenu struct {
	gorm.Model
	Title    string                `gorm:"size:100;not null;comment:标题" json:"title"`
	Image    string                `gorm:"size:255;comment:图片URL" json:"image"`
	Code     string                `gorm:"size:50;uniqueIndex;not null;comment:功能代码" json:"code"`
	Sort     int                   `gorm:"default:0;comment:排序" json:"sort"`
	SubTitle string                `gorm:"size:200;comment:副标题" json:"subTitle"`
	Status   MiniProgramMenuStatus `gorm:"type:tinyint;default:1;index;comment:状态(0:禁用,1:启用)" json:"status"`
	// 扩展信息
	Properties datatypes.JSON `gorm:"type:json;comment:自定义属性JSON" json:"properties"`
}

// MiniProgramMenuQuery 小程序功能栏查询条件
type MiniProgramMenuQuery struct {
	Q          *string                // 综合查询条件，同时模糊匹配标题和代码
	Title      *string                // 标题模糊匹配
	Code       *string                // 代码精确匹配
	Status     *MiniProgramMenuStatus // 状态精确匹配
	Pagination                        // 分页参数
}
