package entity

import (
	"time"

	"gorm.io/gorm"
)

// UserStatus 用户状态枚举
type UserStatus int8

const (
	UserStatusDisabled UserStatus = 0
	UserStatusEnabled  UserStatus = 1
)

// Gender 性别枚举
type Gender string

const (
	GenderMale    Gender = "male"
	GenderFemale  Gender = "female"
	GenderUnknown Gender = "unknown"
)

// UserRole 用户角色枚举（数值类型）
type UserRole int8

const (
	UserRoleRoot       UserRole = 120 // 根管理员（最高权限）
	UserRoleSuperAdmin UserRole = 100 // 超级管理员
	UserRoleAdmin      UserRole = 99  // 管理员
	UserRoleManager    UserRole = 98  // 经理
	UserRoleUser       UserRole = 0   // 普通用户（默认角色）
)

// GetRoleName 获取角色名称
func (r UserRole) GetRoleName() string {
	switch r {
	case UserRoleRoot:
		return "Root"
	case UserRoleSuperAdmin:
		return "SuperAdmin"
	case UserRoleAdmin:
		return "Admin"
	case UserRoleManager:
		return "Manager"
	case UserRoleUser:
		return "User"
	default:
		return "User" // 默认为普通用户
	}
}

// User 用户模型
type User struct {
	gorm.Model
	Username        string     `gorm:"size:50;not null;uniqueIndex;comment:用户名"`
	Password        string     `gorm:"size:255;not null;comment:密码"`
	Nickname        string     `gorm:"size:50;comment:昵称"`
	Email           *string    `gorm:"size:100;uniqueIndex;comment:邮箱"`
	Phone           *string    `gorm:"size:20;uniqueIndex;comment:手机号"`
	Status          UserStatus `gorm:"type:tinyint;default:1;index;comment:状态(0:禁用,1:启用)"`
	Gender          Gender     `gorm:"size:10;default:'unknown';comment:性别(male/female/unknown)"`
	Age             *int       `gorm:"type:tinyint unsigned;comment:年龄"`
	Birthday        *time.Time `gorm:"type:date;comment:生日"`
	Description     string     `gorm:"size:255;comment:描述"`
	Avatar          string     `gorm:"size:255;comment:头像URL"`
	LastLoginAt     *time.Time `gorm:"index;comment:最后登录时间"`
	LastLoginIP     string     `gorm:"size:50;comment:最后登录IP"`
	WechatUnionID   *string    `gorm:"size:128;index;comment:微信UnionID(用于关联小程序和网页登录的同一用户)"`
	WechatOpenID    *string    `gorm:"size:128;index;comment:微信OpenID(小程序)"`
	WechatWebOpenID *string    `gorm:"size:128;index;comment:微信OpenID(网页扫码登录)"`
}

// UserQuery 用户查询条件
type UserQuery struct {
	Q           *string    // 综合查询条件，同时模糊匹配用户名和昵称，精确匹配邮箱和手机号
	Username    *string    // 用户名模糊匹配
	Nickname    *string    // 昵称模糊匹配
	Email       *string    // 邮箱精确匹配
	Phone       *string    // 手机号精确匹配
	LastLoginAt *time.Time // 最后登录时间之后
	CompanyID   *uint      // 公司ID（用于过滤用户）
	Pagination             // 分页参数
}
