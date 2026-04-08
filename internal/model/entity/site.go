package entity

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SiteStatus 场所状态类型
type SiteStatus string

const (
	SiteStatusActive            SiteStatus = "active"
	SiteStatusInactive          SiteStatus = "inactive"
	SiteStatusUnderConstruction SiteStatus = "under_construction"
)

// Site 表示园区、基地或公司场所
type Site struct {
	gorm.Model

	// 基本信息
	Name        string `gorm:"size:100;not null;comment:场所名称"`
	Title       string `gorm:"size:100;comment:场所标题"`
	Code        string `gorm:"size:50;uniqueIndex;comment:场所编码"`
	Description string `gorm:"type:text;comment:场所描述"`

	// 地理信息
	Address    string  `gorm:"size:200;comment:详细地址"`
	Province   string  `gorm:"size:50;comment:省份"`
	City       string  `gorm:"size:50;comment:城市"`
	District   string  `gorm:"size:50;comment:区县"`
	PostalCode string  `gorm:"size:20;comment:邮政编码"`
	Longitude  float64 `gorm:"type:decimal(10,7);comment:经度"`
	Latitude   float64 `gorm:"type:decimal(10,7);comment:纬度"`

	// 分类信息
	Type   string     `gorm:"size:50;index;comment:类型(园区/基地/公司等)"`
	Status SiteStatus `gorm:"size:20;default:'active';index;comment:状态(active/inactive/under_construction)"`

	// 图片信息
	CoverImage string         `gorm:"size:255;comment:封面图片URL"`
	Images     datatypes.JSON `gorm:"type:json;comment:图片列表JSON数组"`

	// 扩展信息
	Properties datatypes.JSON `gorm:"type:json;comment:自定义属性JSON"`

	// 公司关联
	CompanyID *uint    `gorm:"index;comment:所属公司ID"`
	Company   *Company `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	// 关联设备
	Devices []Device `gorm:"many2many:site_devices;"`

	Users []User `gorm:"many2many:user_sites;"`
}

// SiteQuery 查询条件结构体
type SiteQuery struct {
	Q          *string     // 模糊搜索字段（名称或编码）
	Name       *string     // 名称模糊匹配（保留原字段，向后兼容）
	Code       *string     // 编码精确匹配
	Type       *string     // 类型精确匹配
	Status     *SiteStatus // 状态精确匹配
	Province   *string     // 省份
	City       *string     // 城市
	District   *string     // 区县
	DeviceID   *uint       // 关联设备ID
	IDs        *[]uint     // 园区ID列表
	DeviceType *string     // 新增：关联设备类型过滤
	DeviceName *string     // 新增：关联设备名称模糊搜索
	CompanyID  *uint       // 公司ID
	Pagination             // 分页参数
}
