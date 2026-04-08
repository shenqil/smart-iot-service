package entity

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// DeviceStatus 设备状态枚举
type DeviceStatus string

const (
	DeviceStatusActive   DeviceStatus = "active"
	DeviceStatusInactive DeviceStatus = "inactive"
	DeviceStatusFault    DeviceStatus = "fault"
)

// Device 设备主表
type Device struct {
	gorm.Model
	DeviceNum           uint           `gorm:"uniqueIndex:uidx_device_num,where:deleted_at IS NULL;not null;comment:设备编号"`
	Name                *string        `gorm:"size:100;comment:设备名称"`
	Description         string         `gorm:"type:text;comment:设备描述"`
	DeviceType          string         `gorm:"size:17;not null;comment:设备类型"`
	Location            datatypes.JSON `gorm:"type:json;comment:地理坐标"`
	ReadInterval        *int           `gorm:"default:60;comment:读取间隔(秒)"`
	Address             int            `gorm:"default:0;comment:设备地址"`
	MACAddress          *string        `gorm:"size:17;comment:MAC地址"`
	SIMCard             string         `gorm:"size:20;comment:SIM卡号"`
	IsActive            *bool          `gorm:"default:true;comment:是否激活"`
	CameraConfig        datatypes.JSON `gorm:"type:json;comment:摄像头配置"`
	Sensors             datatypes.JSON `gorm:"type:json;not null;comment:传感器配置"`
	Relays              datatypes.JSON `gorm:"type:json;not null;comment:继电器配置"`
	WaterFertilizerInfo datatypes.JSON `gorm:"type:json;comment:水肥信息"`
	CreatorID           uint           `gorm:"not null;index;comment:创建者ID"`
	Status              DeviceStatus   `gorm:"size:20;default:'active';comment:设备状态"`
	CoverImage          string         `gorm:"size:255;comment:封面图片URL"`
	CompanyID           *uint          `gorm:"index;comment:所属公司ID"`
	Company             *Company       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	Sites []Site `gorm:"many2many:site_devices;"`
}

// DeviceQuery 设备查询条件
type DeviceQuery struct {
	DeviceNum     *uint         // 设备编号精确匹配
	DeviceTypes   []string      // 设备类型数组，支持多选
	DeviceName    *string       // 设备名称模糊搜索
	SiteCodes     []string      // 园区编码数组，支持多选
	SiteID        *uint         // 站点ID过滤
	ExcludeSiteID *uint         // 排除指定站点ID的设备
	MACAddress    *string       // MAC地址精确匹配
	IsActive      *bool         // 是否激活
	Status        *DeviceStatus // 状态精确匹配
	ReadInterval  *int          // 读取间隔精确匹配
	CreatorID     *uint         // 创建者ID精确匹配
	CompanyID     *uint         // 公司ID精确匹配
	Q             *string       // 模糊搜索关键字，用于DeviceNum、Name、SIMCard
	Pagination                  // 分页参数
}

// DeviceTypeStats 设备类型统计结构
type DeviceTypeStats struct {
	DeviceType string `json:"deviceType"` // 设备类型
	Count      int64  `json:"count"`      // 设备数量
}
