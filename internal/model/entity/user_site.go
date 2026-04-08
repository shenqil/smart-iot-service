package entity

// UserSite 是用户和园区的多对多关联表
type UserSite struct {
	UserID uint `gorm:"primaryKey"`
	SiteID uint `gorm:"primaryKey"`
}
