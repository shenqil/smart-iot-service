package entity

// UserCompany 表示User和Company的多对多关系，包含用户在公司中的角色
type UserCompany struct {
	UserID    uint     `gorm:"primaryKey;comment:用户ID"`
	CompanyID uint     `gorm:"primaryKey;comment:公司ID"`
	Role      UserRole `gorm:"type:tinyint;not null;index;comment:用户在公司中的角色(Root/SuperAdmin/Admin/Manager/User)"`

	User    User    `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	Company Company `gorm:"foreignKey:CompanyID;constraint:OnDelete:CASCADE;"`
}
