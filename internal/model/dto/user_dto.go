package dto

import (
	"fmt"
	"time"

	"rbac-admin/internal/model/entity"
)

// UserGender 性别枚举
type UserGender string

const (
	UserGenderMale    UserGender = "male"
	UserGenderFemale  UserGender = "female"
	UserGenderUnknown UserGender = "unknown"
)

// UserStatus 状态枚举
type UserStatus int8

const (
	UserStatusDisabled UserStatus = 0
	UserStatusEnabled  UserStatus = 1
)

// RoleStringToNumeric 将角色字符串转换为数值
func RoleStringToNumeric(roleStr string) entity.UserRole {
	switch roleStr {
	case "Root":
		return entity.UserRoleRoot
	case "SuperAdmin":
		return entity.UserRoleSuperAdmin
	case "Admin":
		return entity.UserRoleAdmin
	case "Manager":
		return entity.UserRoleManager
	case "User":
		return entity.UserRoleUser
	default:
		return entity.UserRoleUser // 默认为普通用户
	}
}

// RoleNumericToString 将角色数值转换为字符串
func RoleNumericToString(role entity.UserRole) string {
	return role.GetRoleName()
}

// UserCreateRequest 创建用户请求
type UserCreateRequest struct {
	Username    string     `json:"username" binding:"required,min=3,max=50"`
	Password    string     `json:"password" binding:"required,min=6,max=255"`
	Nickname    string     `json:"nickname" binding:"max=50"`
	Email       *string    `json:"email" binding:"omitempty,max=100"`
	Phone       *string    `json:"phone" binding:"omitempty"`
	Status      UserStatus `json:"status" binding:"oneof=0 1"`
	Gender      UserGender `json:"gender" binding:"oneof=male female unknown"`
	Age         *int       `json:"age" binding:"omitempty,min=0,max=150"`
	Birthday    *string    `json:"birthday" binding:"omitempty,datetime=2006-01-02"`
	Description string     `json:"description" binding:"max=255"`
	Avatar      string     `json:"avatar" binding:"omitempty,max=255"`
	Role        string     `json:"role" binding:"omitempty,oneof=Root SuperAdmin Admin Manager User"`
	CompanyID   uint       `json:"-" form:"-"` // 公司ID（从上下文获取，不暴露给用户）
}

// UserUpdateRequest 更新用户请求
type UserUpdateRequest struct {
	Username    *string     `json:"username" binding:"omitempty,min=3,max=50"`
	Password    *string     `json:"password" binding:"omitempty"`
	Nickname    *string     `json:"nickname" binding:"omitempty,max=50"`
	Email       *string     `json:"email" binding:"omitempty,max=100"`
	Phone       *string     `json:"phone" binding:"omitempty"`
	Status      *UserStatus `json:"status" binding:"omitempty,oneof=0 1"`
	Gender      *UserGender `json:"gender" binding:"omitempty,oneof=male female unknown"`
	Age         *int        `json:"age" binding:"omitempty,min=0,max=150"`
	Birthday    *string     `json:"birthday" binding:"omitempty,datetime=2006-01-02"`
	Description *string     `json:"description" binding:"omitempty,max=255"`
	Avatar      *string     `json:"avatar" binding:"omitempty,max=255"`
	Role        *string     `json:"role" binding:"omitempty,oneof=Root SuperAdmin Admin Manager User"`
	CompanyID   uint        `json:"-" form:"-"` // 公司ID（从上下文获取，不暴露给用户）
}

// UserDeleteRequest 删除用户请求
type UserDeleteRequest struct {
	IDs       []uint `json:"ids" binding:"required,min=1"`
	CompanyID uint   `json:"-" form:"-"` // 公司ID（从上下文获取，不暴露给用户）
}

// UserFilter 用户查询过滤器
type UserFilter struct {
	Q           *string `form:"q" binding:"omitempty,max=100"`           // 综合查询条件，同时模糊匹配用户名和昵称，精确匹配邮箱和手机号
	Username    *string `form:"username" binding:"omitempty,max=50"`     // 用户名模糊匹配
	Nickname    *string `form:"nickname" binding:"omitempty,max=50"`     // 昵称模糊匹配
	Email       *string `form:"email" binding:"omitempty,email,max=100"` // 邮箱精确匹配
	Phone       *string `form:"phone" binding:"omitempty,e164"`          // 手机号精确匹配
	LastLoginAt *int64  `form:"lastLoginAt" binding:"omitempty"`         // 最后登录时间之后（秒级时间戳）
	CompanyID   *uint   `form:"-" json:"-"`                              // 公司ID（从上下文获取，不暴露给用户）
	PaginationRequest
}

// UserResponse 用户响应
type UserResponse struct {
	ID          uint       `json:"id"`
	Username    string     `json:"username"`
	Nickname    string     `json:"nickname"`
	Email       *string    `json:"email,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	Status      UserStatus `json:"status"`
	Gender      UserGender `json:"gender"`
	Age         *int       `json:"age,omitempty"`
	Birthday    *string    `json:"birthday,omitempty"` // datetime=2006-01-02
	Description string     `json:"description"`
	Avatar      string     `json:"avatar"`
	Role        *string    `json:"role,omitempty"`
	LastLoginAt *int64     `json:"lastLoginAt,omitempty"`
	LastLoginIP string     `json:"lastLoginIP"`
	CreatedAt   int64      `json:"createdAt"`
	UpdatedAt   int64      `json:"updatedAt"`
}

// CurrentUserInfoResponse 当前用户信息响应（包含用户信息和管理的企业列表）
type CurrentUserInfoResponse struct {
	Info      *UserResponse              `json:"info"`      // 当前用户信息
	Companies []*CompanyWithRoleResponse `json:"companies"` // 管理的企业列表（包含角色）
}

// UserProfileUpdateRequest 更新用户个人信息请求
type UserProfileUpdateRequest struct {
	Nickname    *string     `json:"nickname" binding:"omitempty,max=50"`
	Email       *string     `json:"email" binding:"omitempty,max=100"`
	Phone       *string     `json:"phone" binding:"omitempty"`
	Gender      *UserGender `json:"gender" binding:"omitempty,oneof=male female unknown"`
	Age         *int        `json:"age" binding:"omitempty,min=0,max=150"`
	Birthday    *string     `json:"birthday" binding:"omitempty,datetime=2006-01-02"`
	Description *string     `json:"description" binding:"omitempty,max=255"`
	Avatar      *string     `json:"avatar" binding:"omitempty,max=255"`
}

// UserPasswordUpdateRequest 更新用户密码请求
type UserPasswordUpdateRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// ConvertToEntity 将创建请求转换为实体
func (req *UserCreateRequest) ConvertToEntity() (*entity.User, error) {
	user := &entity.User{
		Username:    req.Username,
		Password:    req.Password,
		Nickname:    req.Nickname,
		Email:       req.Email,
		Phone:       req.Phone,
		Status:      entity.UserStatus(req.Status),
		Gender:      entity.Gender(req.Gender),
		Age:         req.Age,
		Description: req.Description,
		Avatar:      req.Avatar,
	}

	if req.Birthday != nil {
		birthday, err := time.Parse("2006-01-02", *req.Birthday)
		if err != nil {
			return nil, err
		}
		user.Birthday = &birthday
	}

	return user, nil
}

// ConvertToQuery 将过滤器转换为实体查询
func (f *UserFilter) ConvertToQuery() entity.UserQuery {
	query := entity.UserQuery{
		Q:         f.Q, // 新增的Q参数
		Username:  f.Username,
		Nickname:  f.Nickname,
		Email:     f.Email,
		Phone:     f.Phone,
		CompanyID: f.CompanyID,
		Pagination: entity.Pagination{
			PageNumber: f.GetPageNumber(),
			PageSize:   f.GetPageSize(),
		},
	}

	if f.LastLoginAt != nil {
		t := time.Unix(*f.LastLoginAt, 0)
		query.LastLoginAt = &t
	}

	return query
}

// ConvertToUserResponse 将实体转换为响应
func ConvertToUserResponse(user *entity.User) (*UserResponse, error) {
	return ConvertToUserResponseWithRole(user, nil)
}

// ConvertToUserResponseWithRole 将实体转换为响应（支持角色参数）
func ConvertToUserResponseWithRole(user *entity.User, role *entity.UserRole) (*UserResponse, error) {
	if user == nil {
		return nil, nil
	}

	resp := &UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Nickname:    user.Nickname,
		Email:       user.Email,
		Phone:       user.Phone,
		Status:      UserStatus(user.Status),
		Gender:      UserGender(user.Gender),
		Age:         user.Age,
		Description: user.Description,
		Avatar:      user.Avatar,
		LastLoginIP: user.LastLoginIP,
		CreatedAt:   user.CreatedAt.Unix(),
		UpdatedAt:   user.UpdatedAt.Unix(),
	}

	// 处理可选字段
	if user.LastLoginAt != nil {
		lastLoginAt := user.LastLoginAt.Unix()
		resp.LastLoginAt = &lastLoginAt
	}

	if user.Birthday != nil {
		birthday := user.Birthday.Format("2006-01-02")
		resp.Birthday = &birthday
	}

	// 处理角色信息
	if role != nil {
		roleStr := RoleNumericToString(*role)
		resp.Role = &roleStr
	}

	return resp, nil
}

// GetPageNumber 安全获取页码
func (f *UserFilter) GetPageNumber() int {
	if f.PageNumber <= 0 {
		return 1
	}
	return f.PageNumber
}

// GetPageSize 安全获取每页大小
func (f *UserFilter) GetPageSize() int {
	if f.PageSize <= 0 {
		return 10
	}
	// 限制最大页面大小，防止性能问题
	if f.PageSize > 100 {
		return 100
	}
	return f.PageSize
}

// Validate 验证用户创建请求
func (req *UserCreateRequest) Validate() error {
	// 可以添加额外的业务逻辑验证
	if req.Username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if len(req.Username) < 3 {
		return fmt.Errorf("用户名长度不能少于3个字符")
	}
	if req.Password == "" {
		return fmt.Errorf("密码不能为空")
	}
	if len(req.Password) < 6 {
		return fmt.Errorf("密码长度不能少于6个字符")
	}
	return nil
}

// Validate 验证用户更新请求
func (req *UserUpdateRequest) Validate() error {
	// 检查是否有任何字段被更新
	hasUpdate := req.Username != nil || req.Password != nil || req.Nickname != nil ||
		req.Email != nil || req.Phone != nil || req.Status != nil || req.Gender != nil ||
		req.Age != nil || req.Birthday != nil || req.Description != nil ||
		req.Avatar != nil

	if !hasUpdate {
		return fmt.Errorf("至少需要提供一个更新字段")
	}

	// 验证密码强度
	if req.Password != nil && *req.Password != "" && len(*req.Password) < 6 {
		return fmt.Errorf("密码长度不能少于6个字符")
	}

	return nil
}
