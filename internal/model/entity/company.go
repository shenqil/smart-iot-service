package entity

import (
	"time"

	"gorm.io/gorm"
)

// CompanyStatus 公司状态枚举
type CompanyStatus int8

const (
	CompanyStatusDisabled CompanyStatus = 0
	CompanyStatusEnabled  CompanyStatus = 1
)

// Company 公司模型
type Company struct {
	gorm.Model
	Name        string        `gorm:"size:100;not null;uniqueIndex;comment:公司名称"`
	Code        string        `gorm:"size:50;uniqueIndex;comment:公司编码"`
	Description string        `gorm:"type:text;comment:公司描述"`
	Logo        string        `gorm:"size:255;comment:公司logoURL"`
	Status      CompanyStatus `gorm:"type:tinyint;default:1;index;comment:状态(0:禁用,1:启用)"`

	// 联系信息
	ContactPerson string `gorm:"size:50;comment:联系人姓名"`
	ContactPhone  string `gorm:"size:20;comment:联系电话"`
	ContactEmail  string `gorm:"size:100;comment:联系邮箱"`
	Address       string `gorm:"size:200;comment:公司地址"`

	// 扩展信息
	Properties string     `gorm:"type:json;comment:自定义属性JSON"`
	ExpiredAt  *time.Time `gorm:"comment:过期时间"`
}

// CompanyQuery 公司查询条件
type CompanyQuery struct {
	Q          *string        // 综合查询条件，同时模糊匹配公司名称和编码
	Name       *string        // 公司名称模糊匹配
	Code       *string        // 公司编码精确匹配
	Status     *CompanyStatus // 状态精确匹配
	Pagination                // 分页参数
}
