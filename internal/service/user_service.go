package service

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"rbac-admin/internal/cache"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/model/entity"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/config"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/contextx"
	"rbac-admin/pkg/errx"
)

type UserService struct {
	UserRepo        *repository.UserRepository
	UserCompanyRepo *repository.UserCompanyRepository
	UserRoleCache   *cache.UserRoleCache
}

// getRole implements the cache-aside pattern for fetching a user's role.
func (s *UserService) getRole(ctx context.Context, userID uint, companyID uint) (entity.UserRole, error) {
	if s.UserRoleCache != nil {
		role, err := s.UserRoleCache.GetRole(ctx, userID, companyID)
		if err != nil {
			return 0, errx.New(constant.Forbidden, constant.MsgForbidden, err)
		}
		return role, nil
	}
	// Fallback if cache is not initialized (though it should be)
	userCompany, err := s.UserCompanyRepo.GetByUserAndCompany(ctx, userID, companyID)
	if err != nil {
		return 0, err
	}
	return userCompany.Role, nil
}

// ============================================================================
// 普通用户管理方法（需要CompanyID和Role，用于/api/v1/auth路由组）
// ============================================================================

// Create 创建用户
func (s *UserService) Create(ctx context.Context, req dto.UserCreateRequest) (*dto.UserResponse, error) {

	// 处理空字符串：Email 和 Phone 为空字符串时设置为 nil
	if req.Email != nil && *req.Email == "" {
		req.Email = nil
	}
	if req.Phone != nil && *req.Phone == "" {
		req.Phone = nil
	}

	// 批量检查唯一字段
	uniqueFields := map[string]interface{}{
		"username": req.Username,
	}
	if req.Email != nil {
		uniqueFields["email"] = *req.Email
	}
	if req.Phone != nil {
		uniqueFields["phone"] = *req.Phone
	}

	for field, value := range uniqueFields {
		if exists, err := s.UserRepo.ExistsByField(ctx, field, value); err != nil {
			return nil, err
		} else if exists {
			return nil, errx.New(constant.Conflict, getFieldConflictMessage(field), nil)
		}
	}

	// DTO转换
	user, err := req.ConvertToEntity()
	if err != nil {
		return nil, errx.New(constant.BadRequest, "参数转换失败", err)
	}

	// Encrypt password
	if user.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, errx.New(constant.InternalServerError, "密码加密失败", err)
		}
		user.Password = string(hashedPassword)
	}

	// 创建记录（GORM会自动填充ID、CreatedAt、UpdatedAt等字段到user对象）
	if err := s.UserRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// 创建用户企业关联
	// 如果未指定角色，默认为 User
	roleStr := req.Role
	if roleStr == "" {
		roleStr = "User"
	}
	role := dto.RoleStringToNumeric(roleStr)
	userCompany := &entity.UserCompany{
		UserID:    user.ID,
		CompanyID: req.CompanyID,
		Role:      role,
	}
	if err := s.UserCompanyRepo.Create(ctx, userCompany); err != nil {
		// 如果创建关联失败，删除已创建的用户
		_ = s.UserRepo.Delete(ctx, user.ID)
		return nil, err
	}

	return dto.ConvertToUserResponse(user)
}

// getFieldConflictMessage 获取字段冲突错误消息
func getFieldConflictMessage(field string) string {
	switch field {
	case "username":
		return "用户名已存在"
	case "email":
		return "邮箱已存在"
	case "phone":
		return "手机号已存在"
	default:
		return "字段已存在"
	}
}

// Get 获取用户详情（包含角色信息）
func (s *UserService) Get(ctx context.Context, id uint, companyID uint) (*dto.UserResponse, error) {
	// 查询用户信息
	user, err := s.UserRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 查询用户在该公司的角色信息
	role, err := s.getRole(ctx, id, companyID)
	if err != nil {
		return nil, err
	}

	// 转换响应，包含角色信息
	return dto.ConvertToUserResponseWithRole(user, &role)
}

// Update 更新用户
func (s *UserService) Update(ctx context.Context, id uint, req dto.UserUpdateRequest) error {
	// 校验用户是否属于当前企业
	_, err := s.getRole(ctx, id, req.CompanyID)
	if err != nil {
		return err
	}

	current, err := s.UserRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	updates := make(map[string]interface{})

	// 批量检查唯一字段
	uniqueFieldsToCheck := make(map[string]interface{})

	// 1. 处理唯一字段（用户名）
	if req.Username != nil && *req.Username != current.Username {
		uniqueFieldsToCheck["username"] = *req.Username
	}

	// 2. 处理可空唯一字段（邮箱）
	if req.Email != nil {
		switch {
		case *req.Email == "": // 清空字段
			updates["email"] = nil
		case current.Email == nil || *req.Email != *current.Email: // 新值且需要校验
			uniqueFieldsToCheck["email"] = *req.Email
		}
	}

	// 3. 处理可空唯一字段（手机号）
	if req.Phone != nil {
		switch {
		case *req.Phone == "": // 清空字段
			updates["phone"] = nil
		case current.Phone == nil || *req.Phone != *current.Phone: // 新值且需要校验
			uniqueFieldsToCheck["phone"] = *req.Phone
		}
	}

	// 批量检查唯一字段
	for field, value := range uniqueFieldsToCheck {
		if exists, err := s.UserRepo.ExistsByField(ctx, field, value); err != nil {
			return err
		} else if exists {
			return errx.New(constant.Conflict, getFieldConflictMessage(field), nil)
		}
		updates[field] = value
	}

	// 4. 处理密码
	if req.Password != nil && *req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return errx.New(constant.InternalServerError, "密码加密失败", err)
		}
		updates["password"] = string(hashedPassword)
	}

	// 5. 处理普通字段
	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Gender != nil {
		updates["gender"] = *req.Gender
	}
	if req.Age != nil {
		updates["age"] = *req.Age
	}
	if req.Birthday != nil {
		birthday, err := time.Parse("2006-01-02", *req.Birthday)
		if err != nil {
			return errx.New(constant.BadRequest, "生日格式错误", err)
		}
		updates["birthday"] = birthday
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}

	// 无有效更新时返回（但需要检查是否有角色更新）
	hasRoleUpdate := req.Role != nil
	if len(updates) == 0 && !hasRoleUpdate {
		logger := contextx.GetLoggerFromContext(ctx)
		logger.Info(fmt.Sprintf("用户更新请求无有效字段，用户ID: %d", id))
		return nil
	}

	// 更新用户基本信息
	if len(updates) > 0 {
		err = s.UserRepo.Update(ctx, id, updates)
		if err != nil {
			return err
		}
	}

	// 处理角色更新（如果提供了角色）
	if req.Role != nil {
		role := dto.RoleStringToNumeric(*req.Role)
		roleUpdates := map[string]interface{}{
			"role": role,
		}
		if err := s.UserCompanyRepo.Update(ctx, id, req.CompanyID, roleUpdates); err != nil {
			return err
		}
		// Invalidate cache since role has changed
		if s.UserRoleCache != nil {
			go func() {
				bgCtx := context.Background()
				if err := s.UserRoleCache.DelRole(bgCtx, id, req.CompanyID); err != nil {
					contextx.GetLoggerFromContext(bgCtx).Warn(fmt.Sprintf("delete user role cache failed: %v", err))
				}
			}()
		}
	}

	return nil
}

// Delete 删除用户
func (s *UserService) Delete(ctx context.Context, id uint, companyID uint) error {
	// 校验用户是否属于当前企业
	_, err := s.getRole(ctx, id, companyID)
	if err != nil {
		return err
	}

	// 删除用户企业关联
	if err := s.UserCompanyRepo.Delete(ctx, id, companyID); err != nil {
		return err
	}

	// 删除该用户在该公司的角色缓存
	if s.UserRoleCache != nil {
		go func() {
			bgCtx := context.Background()
			if err := s.UserRoleCache.DelRole(bgCtx, id, companyID); err != nil {
				contextx.GetLoggerFromContext(bgCtx).Warn(fmt.Sprintf("delete user role cache failed: %v", err))
			}
		}()
	}

	return nil
}

// List 分页查询用户列表
func (s *UserService) List(ctx context.Context, filter dto.UserFilter) ([]*dto.UserResponse, int64, error) {
	query := filter.ConvertToQuery()

	users, total, err := s.UserRepo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	companyID := *filter.CompanyID

	responses := make([]*dto.UserResponse, 0, len(users))
	for _, user := range users {
		// 获取用户在该公司的角色
		role, err := s.getRole(ctx, user.ID, companyID)
		if err != nil {
			// 如果获取某个用户的角色失败，记录错误并跳过该用户或返回整个错误
			// 这里选择返回整个错误，因为角色是响应的重要组成部分
			return nil, 0, fmt.Errorf("获取用户 %d 在公司 %d 的角色失败: %w", user.ID, companyID, err)
		}

		resp, err := dto.ConvertToUserResponseWithRole(user, &role)
		if err != nil {
			return nil, 0, err
		}
		responses = append(responses, resp)
	}

	return responses, total, nil
}

// BatchDelete 批量删除用户
func (s *UserService) BatchDelete(ctx context.Context, req dto.UserDeleteRequest) error {
	if len(req.IDs) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}

	// 校验所有用户是否属于当前企业
	for _, id := range req.IDs {
		_, err := s.getRole(ctx, id, req.CompanyID)
		if err != nil {
			return err
		}
	}

	// 批量删除用户企业关联
	pairs := make([]struct {
		UserID    uint
		CompanyID uint
	}, len(req.IDs))
	for i, id := range req.IDs {
		pairs[i] = struct {
			UserID    uint
			CompanyID uint
		}{UserID: id, CompanyID: req.CompanyID}
	}
	if err := s.UserCompanyRepo.BatchDelete(ctx, pairs); err != nil {
		return err
	}

	// 删除所有用户在该公司的角色缓存
	if s.UserRoleCache != nil {
		go func() {
			bgCtx := context.Background()
			for _, id := range req.IDs {
				if err := s.UserRoleCache.DelRole(bgCtx, id, req.CompanyID); err != nil {
					contextx.GetLoggerFromContext(bgCtx).Warn(fmt.Sprintf("delete user role cache for user %d failed: %v", id, err))
				}
			}
		}()
	}

	return nil
}

// ============================================================================
// 当前用户相关方法（操作当前登录用户自身）
// ============================================================================

func (s *UserService) GetCurrentUser(ctx context.Context, id uint) (*dto.UserResponse, error) {
	// 如果是Root角色，返回Root角色信息
	if id == rootUser.ID {
		return &dto.UserResponse{
			ID:          rootUser.ID,
			Username:    config.Conf.Root.UserName,
			Nickname:    config.Conf.Root.Nickname,
			Status:      dto.UserStatusEnabled,
			Gender:      dto.UserGenderUnknown,
			Description: "",
			Avatar:      "",
			LastLoginIP: "",
			CreatedAt:   0,
			UpdatedAt:   0,
		}, nil
	}

	user, err := s.UserRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.ConvertToUserResponse(user)
}

// GetUserCompaniesWithRoles 获取用户的所有企业及其角色
func (s *UserService) GetUserCompaniesWithRoles(ctx context.Context, userID uint) ([]*dto.CompanyWithRoleResponse, error) {
	// 如果是Root角色，返回空列表
	if userID == rootUser.ID {
		return []*dto.CompanyWithRoleResponse{}, nil
	}

	// 查询用户的所有企业关联（包含公司信息）
	userCompanies, err := s.UserCompanyRepo.GetByUserIDWithCompany(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 转换为响应格式
	companies := make([]*dto.CompanyWithRoleResponse, 0, len(userCompanies))
	for _, uc := range userCompanies {
		if uc.Company.ID == 0 {
			continue // 跳过无效的公司数据
		}

		companyResp, err := dto.ConvertToCompanyResponse(&uc.Company)
		if err != nil {
			return nil, err
		}

		companies = append(companies, &dto.CompanyWithRoleResponse{
			CompanyResponse: *companyResp,
			Role:            dto.RoleNumericToString(uc.Role),
		})
	}

	return companies, nil
}

// GetCurrentUserEntity 获取当前用户实体（用于需要完整用户信息的场景）
func (s *UserService) GetCurrentUserEntity(ctx context.Context, id uint) (*entity.User, error) {
	// 如果是Root角色，创建一个临时的 User 对象
	if id == rootUser.ID {
		return &entity.User{
			Model: gorm.Model{
				ID: id,
			},
			Username: config.Conf.Root.UserName,
			Nickname: config.Conf.Root.Nickname,
		}, nil
	}

	return s.UserRepo.GetByID(ctx, id)
}

func (s *UserService) UpdateLastLogin(ctx context.Context, id uint, loginAt time.Time, loginIP string) error {
	if id == config.Conf.Root.ID {
		return nil
	}

	return s.UserRepo.UpdateLastLogin(ctx, id, loginAt, loginIP)
}

// UpdateProfile 更新用户个人信息（不包含密码）
func (s *UserService) UpdateProfile(ctx context.Context, id uint, req dto.UserProfileUpdateRequest) error {
	if id == config.Conf.Root.ID {
		return nil
	}

	// 获取当前用户信息
	current, err := s.UserRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	updates := make(map[string]interface{})

	// 批量检查唯一字段
	uniqueFieldsToCheck := make(map[string]interface{})

	// 处理空字符串：Email 和 Phone 为空字符串时设置为 nil
	email := req.Email
	if email != nil && *email == "" {
		email = nil
	}
	phone := req.Phone
	if phone != nil && *phone == "" {
		phone = nil
	}

	// 1. 处理可空唯一字段（邮箱）
	if email != nil {
		switch {
		case current.Email == nil || *email != *current.Email: // 新值且需要校验
			uniqueFieldsToCheck["email"] = *email
		}
	}

	// 2. 处理可空唯一字段（手机号）
	if phone != nil {
		switch {
		case current.Phone == nil || *phone != *current.Phone: // 新值且需要校验
			uniqueFieldsToCheck["phone"] = *phone
		}
	}

	// 批量检查唯一字段
	for field, value := range uniqueFieldsToCheck {
		if exists, err := s.UserRepo.ExistsByField(ctx, field, value); err != nil {
			return err
		} else if exists {
			return errx.New(constant.Conflict, getFieldConflictMessage(field), nil)
		}
		updates[field] = value
	}

	// 3. 处理普通字段
	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}
	if req.Gender != nil {
		updates["gender"] = *req.Gender
	}
	if req.Age != nil {
		updates["age"] = *req.Age
	}
	if req.Birthday != nil {
		birthday, err := time.Parse("2006-01-02", *req.Birthday)
		if err != nil {
			return errx.New(constant.BadRequest, "生日格式错误", err)
		}
		updates["birthday"] = birthday
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}

	// 无有效更新时返回
	if len(updates) == 0 {
		logger := contextx.GetLoggerFromContext(ctx)
		logger.Info(fmt.Sprintf("用户更新请求无有效字段，用户ID: %d", id))
		return nil
	}

	err = s.UserRepo.Update(ctx, id, updates)
	if err != nil {
		return err
	}

	return nil
}

// UpdatePassword 更新用户密码
func (s *UserService) UpdatePassword(ctx context.Context, id uint, req dto.UserPasswordUpdateRequest) error {
	if id == config.Conf.Root.ID {
		return nil
	}

	// 获取目标用户信息
	user, err := s.UserRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return errx.New(constant.BadRequest, "旧密码不正确", nil)
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errx.New(constant.InternalServerError, "密码加密失败", err)
	}

	// 更新密码
	updates := map[string]interface{}{
		"password": string(hashedPassword),
	}
	return s.UserRepo.Update(ctx, id, updates)
}

// ============================================================================
// 管理员专用方法（不需要CompanyID和Role，用于/api/v1/admin-auth路由组）
// 这些方法直接操作User实体，不涉及公司关联和角色管理
// ============================================================================

// CreateByAdmin 管理员创建用户（不创建公司关联）
func (s *UserService) CreateByAdmin(ctx context.Context, req dto.UserCreateRequest) (*dto.UserResponse, error) {
	// 处理空字符串：Email 和 Phone 为空字符串时设置为 nil
	if req.Email != nil && *req.Email == "" {
		req.Email = nil
	}
	if req.Phone != nil && *req.Phone == "" {
		req.Phone = nil
	}

	// 批量检查唯一字段
	uniqueFields := map[string]interface{}{
		"username": req.Username,
	}
	if req.Email != nil {
		uniqueFields["email"] = *req.Email
	}
	if req.Phone != nil {
		uniqueFields["phone"] = *req.Phone
	}

	for field, value := range uniqueFields {
		if exists, err := s.UserRepo.ExistsByField(ctx, field, value); err != nil {
			return nil, err
		} else if exists {
			return nil, errx.New(constant.Conflict, getFieldConflictMessage(field), nil)
		}
	}

	// DTO转换
	user, err := req.ConvertToEntity()
	if err != nil {
		return nil, errx.New(constant.BadRequest, "参数转换失败", err)
	}

	// Encrypt password
	if user.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, errx.New(constant.InternalServerError, "密码加密失败", err)
		}
		user.Password = string(hashedPassword)
	}

	// 创建记录（不创建用户企业关联）
	if err := s.UserRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return dto.ConvertToUserResponse(user)
}

// UpdateByAdmin 管理员更新用户（不检查公司，不更新角色）
func (s *UserService) UpdateByAdmin(ctx context.Context, id uint, req dto.UserUpdateRequest) error {
	current, err := s.UserRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	updates := make(map[string]interface{})

	// 批量检查唯一字段
	uniqueFieldsToCheck := make(map[string]interface{})

	// 1. 处理唯一字段（用户名）
	if req.Username != nil && *req.Username != current.Username {
		uniqueFieldsToCheck["username"] = *req.Username
	}

	// 2. 处理可空唯一字段（邮箱）
	if req.Email != nil {
		switch {
		case *req.Email == "": // 清空字段
			updates["email"] = nil
		case current.Email == nil || *req.Email != *current.Email: // 新值且需要校验
			uniqueFieldsToCheck["email"] = *req.Email
		}
	}

	// 3. 处理可空唯一字段（手机号）
	if req.Phone != nil {
		switch {
		case *req.Phone == "": // 清空字段
			updates["phone"] = nil
		case current.Phone == nil || *req.Phone != *current.Phone: // 新值且需要校验
			uniqueFieldsToCheck["phone"] = *req.Phone
		}
	}

	// 批量检查唯一字段
	for field, value := range uniqueFieldsToCheck {
		if exists, err := s.UserRepo.ExistsByField(ctx, field, value); err != nil {
			return err
		} else if exists {
			return errx.New(constant.Conflict, getFieldConflictMessage(field), nil)
		}
		updates[field] = value
	}

	// 4. 处理密码
	if req.Password != nil && *req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return errx.New(constant.InternalServerError, "密码加密失败", err)
		}
		updates["password"] = string(hashedPassword)
	}

	// 5. 处理普通字段
	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Gender != nil {
		updates["gender"] = *req.Gender
	}
	if req.Age != nil {
		updates["age"] = *req.Age
	}
	if req.Birthday != nil {
		birthday, err := time.Parse("2006-01-02", *req.Birthday)
		if err != nil {
			return errx.New(constant.BadRequest, "生日格式错误", err)
		}
		updates["birthday"] = birthday
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}

	// 无有效更新时返回
	if len(updates) == 0 {
		logger := contextx.GetLoggerFromContext(ctx)
		logger.Info(fmt.Sprintf("用户更新请求无有效字段，用户ID: %d", id))
		return nil
	}

	// 更新用户基本信息（不更新角色）
	err = s.UserRepo.Update(ctx, id, updates)
	if err != nil {
		return err
	}

	return nil
}

// DeleteByAdmin 管理员删除用户（直接删除用户实体，同时删除所有公司关联）
func (s *UserService) DeleteByAdmin(ctx context.Context, id uint) error {
	// 先删除用户的所有公司关联
	if err := s.UserCompanyRepo.DeleteByUserID(ctx, id); err != nil {
		// 如果删除关联失败，记录日志但不阻止删除用户
		logger := contextx.GetLoggerFromContext(ctx)
		logger.Warn(fmt.Sprintf("删除用户公司关联失败，用户ID: %d, 错误: %v", id, err))
	}

	// 删除用户实体
	if err := s.UserRepo.Delete(ctx, id); err != nil {
		return err
	}

	// 异步清理该用户的所有角色缓存
	if s.UserRoleCache != nil {
		go func() {
			bgCtx := context.Background()
			if err := s.UserRoleCache.DelUserRoles(bgCtx, id); err != nil {
				contextx.GetLoggerFromContext(bgCtx).Warn(fmt.Sprintf("delete user roles cache for user %d failed: %v", id, err))
			}
		}()
	}

	return nil
}

// ListByAdmin 管理员查询用户列表（不按公司过滤）
func (s *UserService) ListByAdmin(ctx context.Context, filter dto.UserFilter) ([]*dto.UserResponse, int64, error) {
	// 确保 CompanyID 为 nil，不按公司过滤
	filter.CompanyID = nil
	query := filter.ConvertToQuery()

	users, total, err := s.UserRepo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.UserResponse, 0, len(users))
	for _, user := range users {
		resp, err := dto.ConvertToUserResponse(user)
		if err != nil {
			return nil, 0, err
		}
		responses = append(responses, resp)
	}

	return responses, total, nil
}

// BatchDeleteByAdmin 管理员批量删除用户（直接删除用户实体）
func (s *UserService) BatchDeleteByAdmin(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}

	// 先删除所有用户的所有公司关联
	for _, id := range ids {
		if err := s.UserCompanyRepo.DeleteByUserID(ctx, id); err != nil {
			// 如果删除关联失败，记录日志但不阻止删除用户
			logger := contextx.GetLoggerFromContext(ctx)
			logger.Warn(fmt.Sprintf("删除用户公司关联失败，用户ID: %d, 错误: %v", id, err))
		}
	}

	// 批量删除用户实体
	if err := s.UserRepo.BatchDelete(ctx, ids); err != nil {
		return err
	}

	// 异步清理缓存
	if s.UserRoleCache != nil {
		go func() {
			bgCtx := context.Background()
			for _, id := range ids {
				if err := s.UserRoleCache.DelUserRoles(bgCtx, id); err != nil {
					contextx.GetLoggerFromContext(bgCtx).Warn(fmt.Sprintf("delete user roles cache for user %d failed: %v", id, err))
				}
			}
		}()
	}

	return nil
}
