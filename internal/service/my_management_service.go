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

type MyManagementService struct {
	Repo          *repository.MyManagementRepository
	DeviceRepo    *repository.DeviceRepository
	UserRoleCache *cache.UserRoleCache
	UserSiteCache *cache.UserSiteCache
	SiteRepo      *repository.SiteRepository
}

func NewMyManagementService(
	repo *repository.MyManagementRepository,
	deviceRepo *repository.DeviceRepository,
	userRoleCache *cache.UserRoleCache,
	userSiteCache *cache.UserSiteCache,
	siteRepo *repository.SiteRepository,
) *MyManagementService {
	return &MyManagementService{
		Repo:          repo,
		DeviceRepo:    deviceRepo,
		UserRoleCache: userRoleCache,
		UserSiteCache: userSiteCache,
		SiteRepo:      siteRepo,
	}
}

// ListMySites 获取用户管理的园区列表（不包含设备）
func (s *MyManagementService) ListMySites(ctx context.Context, filter *dto.MySiteFilter) ([]*dto.MySiteSimpleResponse, int64, error) {
	var (
		sites []*entity.Site
		total int64
		err   error
	)

	siteQuery := filter.ToSiteQuery()
	if siteQuery == nil {
		return nil, 0, errx.New(constant.BadRequest, "过滤条件不能为空", nil)
	}

	// 1. 检查用户角色以确定数据范围
	role, roleErr := s.UserRoleCache.GetRole(ctx, filter.UserID, filter.CompanyID)
	if roleErr != nil {
		// 如果获取角色失败，直接抛出错误
		return nil, 0, errx.New(constant.InternalServerError, "获取用户角色失败", roleErr)
	}

	// 2. 如果是经理或以下角色，则走特殊逻辑
	if role <= entity.UserRoleManager {
		// 对于经理及以下角色，必须从缓存获取其管理的园区ID
		siteIDs, siteErr := s.UserSiteCache.GetUserSiteIDs(ctx, filter.UserID, filter.CompanyID)
		if siteErr != nil {
			// 如果获取缓存失败，直接抛出错误
			return nil, 0, errx.New(constant.InternalServerError, "获取用户园区失败", siteErr)
		}
		if len(siteIDs) == 0 {
			// 如果没有分配的园区，直接返回空
			return []*dto.MySiteSimpleResponse{}, 0, nil
		}
		// 使用获取到的园区ID列表进行查询
		siteQuery.IDs = &siteIDs
	}
	// 统一调用 s.Repo.ListMySites
	sites, total, err = s.Repo.ListMySites(ctx, siteQuery)

	if err != nil {
		return nil, 0, err
	}

	// 3. 转换为响应DTO
	responses := make([]*dto.MySiteSimpleResponse, 0, len(sites))
	for _, site := range sites {
		resp, err := dto.ConvertToMySiteSimpleResponse(site)
		if err != nil {
			return nil, 0, errx.New(constant.InternalServerError, "数据转换失败", err)
		}
		responses = append(responses, resp)
	}

	return responses, total, nil
}

// ListMyDevices 获取用户管理的设备列表
func (s *MyManagementService) ListMyDevices(ctx context.Context, filter *dto.MyDeviceFilter) ([]*dto.MyDeviceResponse, int64, error) {
	// 转换为设备查询条件
	deviceQuery := filter.ToDeviceQuery()
	if deviceQuery == nil {
		return nil, 0, errx.New(constant.BadRequest, "过滤条件不能为空", nil)
	}

	// 设置 SiteCodes 过滤条件（如果指定了则过滤，否则查询所有）

	devices, total, err := s.Repo.ListMyDevices(ctx, deviceQuery)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*dto.MyDeviceResponse, 0, len(devices))
	for _, device := range devices {
		resp, err := dto.ConvertToMyDeviceResponse(device, func(deviceNum uint) string {
			return s.getDeviceStatus(deviceNum)
		})
		if err != nil {
			return nil, 0, errx.New(constant.InternalServerError, "数据转换失败", err)
		}
		responses = append(responses, resp)
	}

	return responses, total, nil
}
func (s *MyManagementService) getDeviceStatus(deviceNum uint) string {
	isOnline := s.DeviceRepo.GetOnlineStatus(uint32(deviceNum))

	status := entity.DeviceStatusInactive
	if isOnline {
		status = entity.DeviceStatusActive
	}

	return string(status)
}

// GetMyDeviceTypeStats 获取我的设备类型统计
func (s *MyManagementService) GetMyDeviceTypeStats(ctx context.Context, filter *dto.MyDeviceTypeStatsFilter) ([]*dto.MyDeviceTypeStatsResponse, error) {
	// 转换为设备查询条件
	deviceQuery := filter.ToDeviceQuery()
	if deviceQuery == nil {
		return nil, errx.New(constant.BadRequest, "过滤条件不能为空", nil)
	}

	// 如果指定了 SiteCodes，则使用这些 SiteCodes 进行过滤
	// 如果没有指定 SiteCodes，则查询当前企业下所有园区的设备
	if len(filter.SiteCodes) > 0 {
		deviceQuery.SiteCodes = filter.SiteCodes
	}

	stats, err := s.Repo.GetMyDeviceTypeStats(ctx, deviceQuery)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.MyDeviceTypeStatsResponse, 0, len(stats))
	for _, stat := range stats {
		resp := &dto.MyDeviceTypeStatsResponse{
			DeviceType: stat.DeviceType,
			Count:      stat.Count,
		}
		responses = append(responses, resp)
	}

	return responses, nil
}
