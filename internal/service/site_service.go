package service

import (
	"context"
	"rbac-admin/internal/cache"
	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/model/entity"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/constant"
	"rbac-admin/pkg/errx"
)

type SiteService struct {
	SiteRepo        *repository.SiteRepository
	UserRepo        *repository.UserRepository
	UserSiteRepo    *repository.UserSiteRepository
	UserCompanyRepo *repository.UserCompanyRepository
	UserSiteCache   *cache.UserSiteCache
}

func NewSiteService(
	siteRepo *repository.SiteRepository,
	userRepo *repository.UserRepository,
	userSiteRepo *repository.UserSiteRepository,
	userCompanyRepo *repository.UserCompanyRepository,
	userSiteCache *cache.UserSiteCache,
) *SiteService {
	return &SiteService{
		SiteRepo:        siteRepo,
		UserRepo:        userRepo,
		UserSiteRepo:    userSiteRepo,
		UserCompanyRepo: userCompanyRepo,
		UserSiteCache:   userSiteCache,
	}
}

func (s *SiteService) Create(ctx context.Context, req dto.SiteCreateRequest) (*dto.SiteResponse, error) {
	// 检查编码唯一性
	if exists, err := s.SiteRepo.ExistsByField(ctx, "code", req.Code); err != nil {
		return nil, err
	} else if exists {
		return nil, errx.New(constant.Conflict, "场所编码已存在", nil)
	}

	// DTO转换
	site, err := req.ConvertToEntity()
	if err != nil {
		return nil, err
	}

	// CompanyID 已在 ConvertToEntity 中验证
	site.CompanyID = &req.CompanyID

	// 创建记录
	if err := s.SiteRepo.Create(ctx, site); err != nil {
		return nil, err
	}

	// 重新查询获取完整数据
	fullSite, err := s.SiteRepo.GetByID(ctx, site.ID)
	if err != nil {
		return nil, err
	}

	return dto.ConvertToSiteResponse(fullSite)
}

func (s *SiteService) Get(ctx context.Context, id uint) (*dto.SiteResponse, error) {
	site, err := s.SiteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.ConvertToSiteResponse(site)
}

func (s *SiteService) Update(ctx context.Context, id uint, req dto.SiteUpdateRequest) error {
	// 获取当前场所信息
	currentSite, err := s.SiteRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查权限：只能更新本公司场所
	if currentSite.CompanyID == nil || *currentSite.CompanyID != req.CompanyID {
		return errx.New(constant.Forbidden, "只能更新本公司场所", nil)
	}

	updates := make(map[string]interface{})

	// 处理普通字段
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.Province != nil {
		updates["province"] = *req.Province
	}
	if req.City != nil {
		updates["city"] = *req.City
	}
	if req.District != nil {
		updates["district"] = *req.District
	}
	if req.PostalCode != nil {
		updates["postal_code"] = *req.PostalCode
	}
	if req.Longitude != nil {
		if *req.Longitude < -180 || *req.Longitude > 180 {
			return errx.New(constant.BadRequest, "经度范围无效", nil)
		}
		updates["longitude"] = *req.Longitude
	}
	if req.Latitude != nil {
		if *req.Latitude < -90 || *req.Latitude > 90 {
			return errx.New(constant.BadRequest, "纬度范围无效", nil)
		}
		updates["latitude"] = *req.Latitude
	}
	if req.Type != nil {
		updates["type"] = *req.Type
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.CoverImage != nil {
		updates["cover_image"] = *req.CoverImage
	}

	// 处理JSON字段（图片）
	if req.Images != nil {
		if len(*req.Images) == 0 {
			updates["images"] = nil
		} else {
			imagesJSON, err := dto.ToJSON(*req.Images)
			if err != nil {
				return errx.New(constant.BadRequest, "图片格式错误", err)
			}
			updates["images"] = imagesJSON
		}
	}

	// 处理JSON字段（属性）
	if req.Properties != nil {
		updates["properties"] = *req.Properties
	}

	// 执行更新
	if len(updates) > 0 {
		if err := s.SiteRepo.Update(ctx, id, updates); err != nil {
			return err
		}
	}

	return nil
}

func (s *SiteService) Delete(ctx context.Context, id uint, companyID uint) error {
	// 获取当前场所信息用于验证
	current, err := s.SiteRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 检查权限：只能删除本公司场所
	if current.CompanyID == nil || *current.CompanyID != companyID {
		return errx.New(constant.Forbidden, "只能删除本公司场所", nil)
	}

	return s.SiteRepo.Delete(ctx, id)
}

func (s *SiteService) List(ctx context.Context, filter dto.SiteFilter) ([]*dto.SiteResponse, int64, error) {
	query := filter.ConvertToQuery()

	sites, total, err := s.SiteRepo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.SiteResponse, 0, len(sites))
	for _, s := range sites {
		resp, err := dto.ConvertToSiteResponse(s)
		if err != nil {
			return nil, 0, err
		}
		responses = append(responses, resp)
	}

	return responses, total, nil
}

func (s *SiteService) BatchDelete(ctx context.Context, req dto.SiteDeleteRequest) error {
	if len(req.IDs) == 0 {
		return errx.New(constant.BadRequest, "ID列表不能为空", nil)
	}

	// 检查每个场所是否属于当前用户公司
	for _, id := range req.IDs {
		currentSite, err := s.SiteRepo.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if currentSite.CompanyID == nil || *currentSite.CompanyID != req.CompanyID {
			return errx.New(constant.Forbidden, "只能删除本公司场所", nil)
		}
	}

	return s.SiteRepo.BatchDelete(ctx, req.IDs)
}

// AssignDevices 园区分配设备
func (s *SiteService) AssignDevices(ctx context.Context, req dto.SiteDeviceAssignRequest) error {
	if len(req.DeviceIDs) == 0 {
		return errx.New(constant.BadRequest, "设备ID列表不能为空", nil)
	}

	// 检查场所是否存在
	if exists, err := s.SiteRepo.ExistsByField(ctx, "id", req.SiteID); err != nil {
		return err
	} else if !exists {
		return errx.New(constant.NotFound, "场所不存在", nil)
	}

	return s.SiteRepo.AssignDevices(ctx, req.SiteID, req.DeviceIDs)
}

// RemoveDevices 园区删除设备
func (s *SiteService) RemoveDevices(ctx context.Context, req dto.SiteDeviceRemoveRequest) error {
	if len(req.DeviceIDs) == 0 {
		return errx.New(constant.BadRequest, "设备ID列表不能为空", nil)
	}

	// 检查场所是否存在
	if exists, err := s.SiteRepo.ExistsByField(ctx, "id", req.SiteID); err != nil {
		return err
	} else if !exists {
		return errx.New(constant.NotFound, "场所不存在", nil)
	}

	return s.SiteRepo.RemoveDevices(ctx, req.SiteID, req.DeviceIDs)
}

func (s *SiteService) AddUsersToSite(ctx context.Context, siteID uint, req dto.SiteUserAddRequest, companyID uint) error {
	site, err := s.SiteRepo.GetByID(ctx, siteID)
	if err != nil {
		return err
	}
	if site.CompanyID == nil || *site.CompanyID != companyID {
		return errx.New(constant.Forbidden, "不能操作非本公司的园区", nil)
	}

	// 优化：一次性检查所有用户是否都属于该公司
	count, err := s.UserCompanyRepo.CountUsersInCompany(ctx, req.UserIDs, companyID)
	if err != nil {
		return err
	}
	if count != int64(len(req.UserIDs)) {
		return errx.New(constant.BadRequest, "一个或多个用户不属于该公司", nil)
	}

	var userSites []*entity.UserSite
	for _, userID := range req.UserIDs {
		userSites = append(userSites, &entity.UserSite{
			UserID: userID,
			SiteID: siteID,
		})
	}

	if err := s.UserSiteRepo.BatchCreate(ctx, userSites); err != nil {
		return err
	}

	// Invalidate cache for each user
	for _, userID := range req.UserIDs {
		go s.UserSiteCache.DelUserSites(context.Background(), userID, companyID)
	}

	return nil
}

func (s *SiteService) RemoveUsersFromSite(ctx context.Context, siteID uint, req dto.SiteUserRemoveRequest, companyID uint) error {
	site, err := s.SiteRepo.GetByID(ctx, siteID)
	if err != nil {
		return err
	}
	if site.CompanyID == nil || *site.CompanyID != companyID {
		return errx.New(constant.Forbidden, "不能操作非本公司的园区", nil)
	}

	if err := s.UserSiteRepo.BatchDelete(ctx, req.UserIDs, siteID); err != nil {
		return err
	}

	// Invalidate cache for each user
	for _, userID := range req.UserIDs {
		go s.UserSiteCache.DelUserSites(context.Background(), userID, companyID)
	}

	return nil
}

func (s *SiteService) ListSiteUsers(ctx context.Context, siteID uint, filter dto.SiteUserFilter) ([]*dto.SiteUserResponse, int64, error) {
	users, total, err := s.UserSiteRepo.ListUsersBySiteID(ctx, siteID, filter.GetPageNumber(), filter.GetPageSize())
	if err != nil {
		return nil, 0, err
	}

	var responses []*dto.SiteUserResponse
	for _, user := range users {
		email := ""
		if user.Email != nil {
			email = *user.Email
		}
		phone := ""
		if user.Phone != nil {
			phone = *user.Phone
		}
		responses = append(responses, &dto.SiteUserResponse{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Email:    email,
			Phone:    phone,
		})
	}

	return responses, total, nil
}

func (s *SiteService) ListUsersNotInSite(ctx context.Context, siteID uint, filter dto.SiteUserFilter, companyID uint) ([]*dto.SiteUserResponse, int64, error) {
	users, total, err := s.UserSiteRepo.ListUsersNotInSite(ctx, siteID, companyID, filter.GetPageNumber(), filter.GetPageSize())
	if err != nil {
		return nil, 0, err
	}

	var responses []*dto.SiteUserResponse
	for _, user := range users {
		email := ""
		if user.Email != nil {
			email = *user.Email
		}
		phone := ""
		if user.Phone != nil {
			phone = *user.Phone
		}
		responses = append(responses, &dto.SiteUserResponse{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Email:    email,
			Phone:    phone,
		})
	}

	return responses, total, nil
}
