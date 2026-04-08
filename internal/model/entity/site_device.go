package entity

// SiteDevice 表示Site和Device的多对多关系
type SiteDevice struct {
	SiteID   uint `gorm:"primaryKey;comment:场所ID"`
	DeviceID uint `gorm:"primaryKey;comment:设备ID"`

	Site   Site   `gorm:"foreignKey:SiteID;constraint:OnDelete:CASCADE;"`
	Device Device `gorm:"foreignKey:DeviceID;constraint:OnDelete:CASCADE;"`
}
