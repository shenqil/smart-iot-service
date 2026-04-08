package service

import (
	"context"
	"fmt"
	"time"

	"rbac-admin/internal/cache"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/model/entity"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/errx"
)

type CompanyService struct {
	CompanyRepo     *repository.CompanyRepository
	UserRepo        *repository.UserRepository
	UserCompanyRepo *repository.UserCompanyRepository
	UserRoleCache   *cache.UserRoleCache // Changed from UserCache
}

// Create 创建公司
func (s *CompanyService) Create(ctx context.Context, req dto.CompanyCreateRequest) (*dto.CompanyResponse, error) {

	// 唯一字段检查
	if exists, err := s.CompanyRepo.ExistsByField(ctx, "name", req.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errx.New(constant.Conflict, "公司名称已存在", nil)
	}

	if exists, err := s.CompanyRepo.ExistsByField(ctx, "code", req.Code); err != nil {
		return nil, err
	} else if exists {
		return nil, errx.New(constant.Conflict, "公司编码已存在", nil)
	}

	// DTO转换
	company, err := req.ConvertToEntity()
	if err != nil {
		return nil, errx.New(constant.BadRequest, "参数转换失败", err)
	}

	// 创建记录（GORM会自动填充ID、CreatedAt、UpdatedAt等字段到company对象）
	if err := s.CompanyRepo.Create(ctx, company); err != nil {
		return nil, err
	}

	return dto.ConvertToCompanyResponse(company)
}

// Get 获取公司详情
func (s *CompanyService) Get(ctx context.Context, id uint) (*dto.CompanyResponse, error) {
	company, err := s.CompanyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.ConvertToCompanyResponse(company)
}

// Update 更新公司
func (s *CompanyService) Update(ctx context.Context, id uint, req dto.CompanyUpdateRequest) error {

	updates := make(map[string]interface{})
	current, err := s.CompanyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 1. 处理唯一字段（公司名称）
	if req.Name != nil && *req.Name != current.Name {
		if exists, err := s.CompanyRepo.ExistsByField(ctx, "name", *req.Name); err != nil {
			return err
		} else if exists {
			return errx.New(constant.Conflict, "公司名称已存在", nil)
		}
		updates["name"] = *req.Name
	}

	// 2. 处理唯一字段（公司编码）
	if req.Code != nil && *req.Code != current.Code {
		if exists, err := s.CompanyRepo.ExistsByField(ctx, "code", *req.Code); err != nil {
			return err
		} else if exists {
			return errx.New(constant.Conflict, "公司编码已存在", nil)
		}
		updates["code"] = *req.Code
	}

	// 3. 处理普通字段
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Logo != nil {
		updates["logo"] = *req.Logo
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.ContactPerson != nil {
		updates["contact_person"] = *req.ContactPerson
	}
	if req.ContactPhone != nil {
		updates["contact_phone"] = *req.ContactPhone
	}
	if req.ContactEmail != nil {
		updates["contact_email"] = *req.ContactEmail
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.Properties != nil {
		updates["properties"] = *req.Properties
	}
	if req.ExpiredAt != nil {
		expiredAt, err := time.Parse(time.RFC3339, *req.ExpiredAt)
		if err != nil {
			return errx.New(constant.BadRequest, "过期时间格式错误", err)
		}
		updates["expired_at"] = expiredAt
	}

	// 无有效更新时返回
	if len(updates) == 0 {
		return errx.New(constant.BadRequest, "未提供有效更新字段", nil)
	}

	if err := s.CompanyRepo.Update(ctx, id, updates); err != nil {
		return err
	}

	return nil
}

// Delete 删除公司
func (s *CompanyService) Delete(ctx context.Context, id uint) error {
	if err := s.CompanyRepo.Delete(ctx, id); err != nil {
		return err
	}

	// 删除相关角色缓存（异步执行，不阻塞删除操作）
	if s.UserRoleCache != nil {
		go s.UserRoleCache.DelCompanyRoles(context.Background(), id)
	}

	return nil
}

// List 分页查询公司列表
func (s *CompanyService) List(ctx context.Context, filter dto.CompanyFilter) ([]*dto.CompanyResponse, int64, error) {
	query := filter.ConvertToQuery()
	companies, total, err := s.CompanyRepo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.CompanyResponse, 0, len(companies))
	for _, company := range companies {
		resp, err := dto.ConvertToCompanyResponse(company)
		if err != nil {
			return nil, 0, err
		}
		responses = append(responses, resp)
	}

	return responses, total, nil
}

// BatchDelete 批量删除公司
func (s *CompanyService) BatchDelete(ctx context.Context, req dto.CompanyDeleteRequest) error {
	if len(req.IDs) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}

	if err := s.CompanyRepo.BatchDelete(ctx, req.IDs); err != nil {
		return err
	}

	// 删除相关角色缓存（异步执行，不阻塞删除操作）
	if s.UserRoleCache != nil {
		for _, id := range req.IDs {
			go s.UserRoleCache.DelCompanyRoles(context.Background(), id)
		}
	}

	return nil
}

// ============================================================================
// 企业管理用户相关方法
// ============================================================================

// ListCompanyUsers 分页查询某个企业的用户列表（含角色，可模糊搜索用户名）
func (s *CompanyService) ListCompanyUsers(ctx context.Context, companyID uint, filter dto.CompanyUserFilter) ([]*dto.CompanyUserBriefResponse, int64, error) {
	if companyID == 0 {
		return nil, 0, errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	// 验证公司是否存在
	exists, err := s.CompanyRepo.ExistsByField(ctx, "id", companyID)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, errx.New(constant.NotFound, "公司不存在", nil)
	}

	// 查询企业用户列表
	userCompanies, total, err := s.UserCompanyRepo.ListCompanyUsers(
		ctx,
		companyID,
		filter.Q,
		filter.GetPageNumber(),
		filter.GetPageSize(),
	)
	if err != nil {
		return nil, 0, err
	}

	// 转换为响应格式
	responses := make([]*dto.CompanyUserBriefResponse, 0, len(userCompanies))
	for _, uc := range userCompanies {
		resp := &dto.CompanyUserBriefResponse{
			ID:       uc.User.ID,
			Username: uc.User.Username,
			Nickname: uc.User.Nickname,
			Email:    uc.User.Email,
			Phone:    uc.User.Phone,
			Avatar:   uc.User.Avatar,
			Role:     dto.RoleNumericToString(uc.Role),
		}
		responses = append(responses, resp)
	}

	return responses, total, nil
}

// ListUsersNotInCompany 分页查询某个企业不存在的用户列表（可模糊搜索用户名）
func (s *CompanyService) ListUsersNotInCompany(ctx context.Context, companyID uint, filter dto.CompanyUserNotInFilter) ([]*dto.CompanyUserBriefResponse, int64, error) {
	if companyID == 0 {
		return nil, 0, errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	// 验证公司是否存在
	exists, err := s.CompanyRepo.ExistsByField(ctx, "id", companyID)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, errx.New(constant.NotFound, "公司不存在", nil)
	}

	// 查询不存在企业的用户列表
	users, total, err := s.UserCompanyRepo.ListUsersNotInCompany(
		ctx,
		companyID,
		filter.Q,
		filter.GetPageNumber(),
		filter.GetPageSize(),
	)
	if err != nil {
		return nil, 0, err
	}

	// 转换为响应格式（这些用户不在企业中，所以没有角色）
	responses := make([]*dto.CompanyUserBriefResponse, 0, len(users))
	for _, user := range users {
		resp := &dto.CompanyUserBriefResponse{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Email:    user.Email,
			Phone:    user.Phone,
			Avatar:   user.Avatar,
			Role:     "", // 不在企业中的用户没有角色
		}
		responses = append(responses, resp)
	}

	return responses, total, nil
}

// AddUserToCompany 为某个企业新增用户（含角色）
func (s *CompanyService) AddUserToCompany(ctx context.Context, companyID uint, req dto.CompanyUserAddRequest) error {
	if companyID == 0 {
		return errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	// 验证公司是否存在
	exists, err := s.CompanyRepo.ExistsByField(ctx, "id", companyID)
	if err != nil {
		return err
	}
	if !exists {
		return errx.New(constant.NotFound, "公司不存在", nil)
	}

	// 验证用户是否存在
	exists, err = s.UserRepo.ExistsByField(ctx, "id", req.UserID)
	if err != nil {
		return err
	}
	if !exists {
		return errx.New(constant.NotFound, "用户不存在", nil)
	}

	// 检查用户是否已经在该企业中
	exists, err = s.UserCompanyRepo.Exists(ctx, req.UserID, companyID)
	if err != nil {
		return err
	}
	if exists {
		return errx.New(constant.Conflict, "用户已在该企业中", nil)
	}

	// 创建用户企业关联
	userCompany := &entity.UserCompany{
		UserID:    req.UserID,
		CompanyID: companyID,
		Role:      dto.RoleStringToNumeric(req.Role),
	}

	if err := s.UserCompanyRepo.Create(ctx, userCompany); err != nil {
		return err
	}

	// 删除该用户的角色缓存（因为新增了关联，缓存需要更新）
	if s.UserRoleCache != nil {
		s.UserRoleCache.DelRole(ctx, req.UserID, companyID)
	}

	return nil
}

// BatchAddUsersToCompany 批量为某个企业新增用户（含角色）
func (s *CompanyService) BatchAddUsersToCompany(ctx context.Context, companyID uint, req dto.CompanyUserBatchAddRequest) error {
	if companyID == 0 {
		return errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	if len(req.Users) == 0 {
		return errx.New(constant.BadRequest, "用户列表不能为空", nil)
	}

	// 验证公司是否存在
	exists, err := s.CompanyRepo.ExistsByField(ctx, "id", companyID)
	if err != nil {
		return err
	}
	if !exists {
		return errx.New(constant.NotFound, "公司不存在", nil)
	}

	// 检查请求中是否有重复的用户ID
	userIDMap := make(map[uint]bool)
	for _, userReq := range req.Users {
		if userReq.UserID == 0 {
			return errx.New(constant.BadRequest, "用户ID不能为空", nil)
		}
		if userIDMap[userReq.UserID] {
			return errx.New(constant.BadRequest, fmt.Sprintf("请求中存在重复的用户ID: userId=%d", userReq.UserID), nil)
		}
		userIDMap[userReq.UserID] = true
	}

	// 验证所有用户是否存在
	for _, userReq := range req.Users {
		exists, err := s.UserRepo.ExistsByField(ctx, "id", userReq.UserID)
		if err != nil {
			return err
		}
		if !exists {
			return errx.New(constant.NotFound, fmt.Sprintf("用户不存在: userId=%d", userReq.UserID), nil)
		}
	}

	// 检查哪些用户已经在该企业中（检测到第一个存在用户立即抛出错误）
	for _, userReq := range req.Users {
		exists, err := s.UserCompanyRepo.Exists(ctx, userReq.UserID, companyID)
		if err != nil {
			return err
		}
		if exists {
			return errx.New(constant.Conflict, fmt.Sprintf("用户已在该企业中: userId=%d", userReq.UserID), nil)
		}
	}

	// 构建批量创建的用户企业关联列表
	userCompanies := make([]*entity.UserCompany, 0, len(req.Users))
	userIDs := make([]uint, 0, len(req.Users))
	for _, userReq := range req.Users {
		userCompanies = append(userCompanies, &entity.UserCompany{
			UserID:    userReq.UserID,
			CompanyID: companyID,
			Role:      dto.RoleStringToNumeric(userReq.Role),
		})
		userIDs = append(userIDs, userReq.UserID)
	}

	// 批量创建用户企业关联
	if err := s.UserCompanyRepo.BatchCreate(ctx, userCompanies); err != nil {
		return err
	}

	// 删除所有用户的角色缓存（因为新增了关联，缓存需要更新）
	if s.UserRoleCache != nil {
		go func() {
			for _, userID := range userIDs {
				s.UserRoleCache.DelRole(context.Background(), userID, companyID)
			}
		}()
	}

	return nil
}

// RemoveUserFromCompany 删除企业与用户的关联关系
func (s *CompanyService) RemoveUserFromCompany(ctx context.Context, companyID uint, userID uint) error {
	if companyID == 0 {
		return errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}
	if userID == 0 {
		return errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}

	// 验证公司是否存在
	exists, err := s.CompanyRepo.ExistsByField(ctx, "id", companyID)
	if err != nil {
		return err
	}
	if !exists {
		return errx.New(constant.NotFound, "公司不存在", nil)
	}

	// 删除用户企业关联
	if err := s.UserCompanyRepo.Delete(ctx, userID, companyID); err != nil {
		return err
	}

	// 删除该用户的角色缓存（因为删除了关联，缓存需要删除）
	if s.UserRoleCache != nil {
		s.UserRoleCache.DelRole(ctx, userID, companyID)
	}

	return nil
}

// BatchRemoveUsersFromCompany 批量删除某个企业的用户
func (s *CompanyService) BatchRemoveUsersFromCompany(ctx context.Context, companyID uint, req dto.CompanyUserBatchDeleteRequest) error {
	if companyID == 0 {
		return errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}

	if len(req.UserIDs) == 0 {
		return errx.New(constant.BadRequest, "用户ID列表不能为空", nil)
	}

	// 验证公司是否存在
	exists, err := s.CompanyRepo.ExistsByField(ctx, "id", companyID)
	if err != nil {
		return err
	}
	if !exists {
		return errx.New(constant.NotFound, "公司不存在", nil)
	}

	// 验证所有用户ID
	for _, userID := range req.UserIDs {
		if userID == 0 {
			return errx.New(constant.BadRequest, "用户ID不能为空", nil)
		}
	}

	// 构建批量删除的 pairs
	pairs := make([]struct {
		UserID    uint
		CompanyID uint
	}, len(req.UserIDs))
	for i, userID := range req.UserIDs {
		pairs[i] = struct {
			UserID    uint
			CompanyID uint
		}{UserID: userID, CompanyID: companyID}
	}

	// 批量删除用户企业关联
	if err := s.UserCompanyRepo.BatchDelete(ctx, pairs); err != nil {
		return err
	}

	// 删除所有用户在该公司的角色缓存（异步执行）
	if s.UserRoleCache != nil {
		go func() {
			for _, userID := range req.UserIDs {
				s.UserRoleCache.DelRole(context.Background(), userID, companyID)
			}
		}()
	}

	return nil
}

// UpdateUserRoleInCompany 更新某个用户在某个企业的角色
func (s *CompanyService) UpdateUserRoleInCompany(ctx context.Context, companyID uint, userID uint, req dto.CompanyUserUpdateRoleRequest) error {
	if companyID == 0 {
		return errx.New(constant.BadRequest, "公司ID不能为空", nil)
	}
	if userID == 0 {
		return errx.New(constant.BadRequest, "用户ID不能为空", nil)
	}

	// 验证公司是否存在
	exists, err := s.CompanyRepo.ExistsByField(ctx, "id", companyID)
	if err != nil {
		return err
	}
	if !exists {
		return errx.New(constant.NotFound, "公司不存在", nil)
	}

	// 验证用户是否存在
	exists, err = s.UserRepo.ExistsByField(ctx, "id", userID)
	if err != nil {
		return err
	}
	if !exists {
		return errx.New(constant.NotFound, "用户不存在", nil)
	}

	// 检查用户企业关联是否存在
	exists, err = s.UserCompanyRepo.Exists(ctx, userID, companyID)
	if err != nil {
		return err
	}
	if !exists {
		return errx.New(constant.NotFound, "用户不在该企业中", nil)
	}

	// 更新角色
	updates := map[string]interface{}{
		"role": dto.RoleStringToNumeric(req.Role),
	}

	if err := s.UserCompanyRepo.Update(ctx, userID, companyID, updates); err != nil {
		return err
	}

	// 删除该用户的角色缓存（因为更新了角色，缓存需要更新）
	if s.UserRoleCache != nil {
		s.UserRoleCache.DelRole(ctx, userID, companyID)
	}

	return nil
}
