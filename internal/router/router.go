package router

import (
	"rbac-admin/internal/cache"
	"rbac-admin/internal/handler"
	"rbac-admin/internal/middleware"
	"rbac-admin/pkg/config"
	"rbac-admin/pkg/jwt"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	_ "rbac-admin/docs"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RouterSet 注入router
var RouterSet = wire.NewSet(wire.Struct(new(RouterDependencies), "*"))

// RouterDependencies 包含路由初始化所需的所有依赖处理器
type RouterDependencies struct {
	JWT                            *jwt.JWT
	UserRoleCache                  *cache.UserRoleCache // Replaced UserCache
	AuthHandler                    *handler.AuthHandler
	UserHandler                    *handler.UserHandler
	AliyunOSSHandler               *handler.AliyunOSSHandler
	DeviceTypeHandler              *handler.DeviceTypeHandler
	StandardSensorHandler          *handler.StandardSensorHandler
	StandardRelayHandler           *handler.StandardRelayHandler
	DeviceHandler                  *handler.DeviceHandler
	SiteHandler                    *handler.SiteHandler
	MyManagementHandler            *handler.MyManagementHandler
	WeatherHandler                 *handler.WeatherHandler
	DeviceScreenshotHandler        *handler.DeviceScreenshotHandler
	CompanyHandler                 *handler.CompanyHandler
	DeviceAlarmHandler             *handler.DeviceAlarmHandler
	MiniProgramMenuHandler         *handler.MiniProgramMenuHandler
	DeviceSimulatorHandler         *handler.DeviceSimulatorHandler
	WaterFertilizerTemplateHandler *handler.WaterFertilizerTemplateHandler
	DeployHandler                  *handler.DeployHandler
}

// SetupRouter 初始化主路由
func SetupRouter(deps *RouterDependencies) *gin.Engine {

	gin.SetMode(config.Conf.App.RunMode)

	// r := gin.Default()
	r := gin.New()

	// 日志
	r.Use(middleware.LoggerMiddleware())

	// 恢复中间件
	r.Use(gin.Recovery())

	// 默认配置（允许所有来源）
	r.Use(cors.Default())

	r.GET("/ping", func(ctx *gin.Context) {
		ctx.String(200, "pong")
	})

	// 公共路由（无需权限校验）
	public := r.Group("/api/v1/public")
	{
		public.POST("/login", deps.AuthHandler.Login)                                       // 用户登录
		public.POST("/logout", deps.AuthHandler.Logout)                                     // 用户注销
		public.POST("/miniprogram/login", deps.AuthHandler.MiniProgramLogin)                // 小程序登录
		public.POST("/miniprogram/complete-info", deps.AuthHandler.MiniProgramCompleteInfo) // 小程序注册（补充用户信息）
		public.GET("/wechat/qrcode", deps.AuthHandler.GenerateWechatQRCode)                 // 生成微信扫码登录二维码
		public.POST("/wechat/qrcode/status", deps.AuthHandler.GetWechatQRCodeStatus)        // 查询微信扫码状态
		public.GET("/wechat/callback", deps.AuthHandler.WechatCallback)                     // 微信扫码登录回调
	}

	// 受保护的路由（需要权限校验）
	apiGroup := r.Group("/api/v1/auth")
	// 第一个中间件：验证token，所有apiGroup都需要执行
	apiGroup.Use(middleware.JWTTokenMiddleware(deps.JWT))
	// 第二个中间件：获取用户权限和公司信息，可以跳过部分URL
	// 跳过 /api/v1/auth/users/current，因为该接口不校验角色和企业
	// 跳过 /api/v1/auth/oss/sts，因为该接口不校验角色和企业
	apiGroup.Use(middleware.UserPermissionMiddleware(deps.UserRoleCache, "/api/v1/auth/users/current", "/api/v1/auth/oss/sts")) // Changed UserCache to UserRoleCache
	{

		// ========================================================================
		// 用户管理路由（需要CompanyID和Role，受企业权限控制）
		// ========================================================================
		apiGroup.GET("/users/current", deps.UserHandler.GetCurrentUser)          // 获取当前用户详情（不校验角色和企业）
		apiGroup.GET("/users", deps.UserHandler.List)                            // 获取用户列表
		apiGroup.GET("/users/:id", deps.UserHandler.Get)                         // 获取用户详情（包含角色）
		apiGroup.POST("/users", deps.UserHandler.Create)                         // 创建用户
		apiGroup.PUT("/users/:id", deps.UserHandler.Update)                      // 更新用户信息
		apiGroup.DELETE("/users/:id", deps.UserHandler.Delete)                   // 删除用户
		apiGroup.DELETE("/users", deps.UserHandler.BatchDelete)                  // 删除用户
		apiGroup.PUT("/users/current/profile", deps.UserHandler.UpdateProfile)   // 更新当前用户个人信息
		apiGroup.PUT("/users/current/password", deps.UserHandler.UpdatePassword) // 更新当前用户密码

		// OSS 管理
		apiGroup.GET("/oss/sts", deps.AliyunOSSHandler.GetPolicyToken)

		// 设备类型（只保留 /all 路由）
		apiGroup.GET("/device-types/all", deps.DeviceTypeHandler.ListAllDeviceTypes) // 获取所有设备类型列表

		// 所有标准的传感器（只保留 /all 路由）
		apiGroup.GET("/standard-sensors/all", deps.StandardSensorHandler.ListAll) // 获取列表

		// 所有标准的继电器（只保留 /all 路由）
		apiGroup.GET("/standard-relays/all", deps.StandardRelayHandler.ListAll) // 获取所有（无分页）

		// 所有设备
		apiGroup.GET("/devices", deps.DeviceHandler.List)                                                   // 获取列表
		apiGroup.GET("/devices/:id", deps.DeviceHandler.Get)                                                // 获取详情
		apiGroup.POST("/devices", deps.DeviceHandler.Create)                                                // 创建
		apiGroup.PUT("/devices/:id", deps.DeviceHandler.Update)                                             // 更新
		apiGroup.PUT("/devices/:id/camera-config", deps.DeviceHandler.UpdateCameraConfig)                   // 更新摄像头配置
		apiGroup.PUT("/devices/:id/sensors", deps.DeviceHandler.UpdateSensors)                              // 更新传感器配置
		apiGroup.PUT("/devices/:id/relays", deps.DeviceHandler.UpdateRelays)                                // 更新继电器配置
		apiGroup.PUT("/devices/:id/location", deps.DeviceHandler.UpdateLocation)                            // 更新地理坐标
		apiGroup.PUT("/devices/:id/basic-info", deps.DeviceHandler.UpdateBasicInfo)                         // 更新基础信息
		apiGroup.PUT("/devices/:id/water-fertilizer-info", deps.DeviceHandler.UpdateWaterFertilizerInfo)    // 更新水肥信息
		apiGroup.GET("/devices/:id/water-fertilizer-info", deps.DeviceHandler.GetWaterFertilizerInfo)       // 查询水肥信息
		apiGroup.DELETE("/devices/:id/water-fertilizer-info", deps.DeviceHandler.DeleteWaterFertilizerInfo) // 删除水肥信息
		apiGroup.DELETE("/devices/:id", deps.DeviceHandler.Delete)                                          // 删除
		apiGroup.DELETE("/devices", deps.DeviceHandler.BatchDelete)                                         // 批量删除

		// 所有园区
		apiGroup.GET("/sites", deps.SiteHandler.List)                            // 获取列表
		apiGroup.GET("/sites/:id", deps.SiteHandler.Get)                         // 获取详情
		apiGroup.POST("/sites", deps.SiteHandler.Create)                         // 创建
		apiGroup.PUT("/sites/:id", deps.SiteHandler.Update)                      // 更新
		apiGroup.DELETE("/sites/:id", deps.SiteHandler.Delete)                   // 删除
		apiGroup.DELETE("/sites", deps.SiteHandler.BatchDelete)                  // 批量删除
		apiGroup.POST("/sites/devices/assign", deps.SiteHandler.AssignDevices)   // 分配设备
		apiGroup.DELETE("/sites/devices/remove", deps.SiteHandler.RemoveDevices) // 删除设备
		apiGroup.POST("/sites/:id/users", deps.SiteHandler.AddUsersToSite)       // 批量添加用户
		apiGroup.DELETE("/sites/:id/users", deps.SiteHandler.RemoveUsersFromSite) // 批量删除用户
		apiGroup.GET("/sites/:id/users", deps.SiteHandler.ListSiteUsers)         // 查询园区用户
		apiGroup.GET("/sites/:id/users/not-in", deps.SiteHandler.ListUsersNotInSite) // 查询园区未分配用户

		// 我的管理
		myGroup := apiGroup.Group("/my")
		myGroup.GET("/sites", deps.MyManagementHandler.ListMySites)                             // 获取列表
		myGroup.GET("/devices", deps.MyManagementHandler.ListMyDevices)                         // 获取详情
		myGroup.GET("/devices/type-stats", deps.MyManagementHandler.GetMyDeviceTypeStats)       // 获取设备类型统计
		myGroup.GET("/device-types", deps.DeviceTypeHandler.ListAllDeviceTypes)                 // 获取所有设备类型（不包含ID）
		myGroup.GET("/device/:deviceSN/realtime", deps.MyManagementHandler.GetRealTimeData)     // 获取列表
		myGroup.GET("/device/sensors/history", deps.MyManagementHandler.GetSensorHistoryData)   // 获取详情
		myGroup.POST("/device/relay/switch", deps.MyManagementHandler.SwitchRelay)              // 控制继电器开关
		myGroup.GET("/device/relay/records", deps.MyManagementHandler.ListDeviceRelayRecords)   // 获取设备继电器操作记录列表
		myGroup.GET("/device/sensors/comparison", deps.MyManagementHandler.GetSensorComparison) // 获取详情

		// 设备图片管理
		apiGroup.GET("/device-screenshots", deps.DeviceScreenshotHandler.List)           // 获取列表
		apiGroup.GET("/device-screenshots/:id", deps.DeviceScreenshotHandler.Get)        // 获取详情
		apiGroup.POST("/device-screenshots", deps.DeviceScreenshotHandler.Create)        // 创建
		apiGroup.PUT("/device-screenshots/:id", deps.DeviceScreenshotHandler.Update)     // 更新
		apiGroup.DELETE("/device-screenshots/:id", deps.DeviceScreenshotHandler.Delete)  // 删除
		apiGroup.DELETE("/device-screenshots", deps.DeviceScreenshotHandler.BatchDelete) // 批量删除

		// 获取当前天气
		apiGroup.GET("/weather", deps.WeatherHandler.GetWeather)

		// 设备告警管理
		apiGroup.GET("/device-alarms", deps.DeviceAlarmHandler.List)                        // 获取告警列表
		apiGroup.POST("/device-alarms/batch-process", deps.DeviceAlarmHandler.BatchProcess) // 批量处理告警

		// 水肥模板管理
		apiGroup.GET("/water-fertilizer-templates", deps.WaterFertilizerTemplateHandler.List) // 获取水肥模板列表

		// 小程序功能栏配置（查询所有列表，直接从数据库查询，不使用缓存）
		apiGroup.GET("/miniprogram-menus/all", deps.MiniProgramMenuHandler.ListAll) // 获取所有小程序功能栏配置列表
	}

	// 管理员路由（只需要token验证，不需要权限校验）
	adminAuthGroup := r.Group("/api/v1/admin-auth")
	// 只使用第一个中间件：验证token
	adminAuthGroup.Use(middleware.JWTTokenMiddleware(deps.JWT))
	{
		// 设备类型管理
		adminAuthGroup.GET("/device-types", deps.DeviceTypeHandler.ListDeviceTypes)           // 获取列表
		adminAuthGroup.POST("/device-types", deps.DeviceTypeHandler.CreateDeviceType)         // 创建
		adminAuthGroup.PUT("/device-types/:id", deps.DeviceTypeHandler.UpdateDeviceType)      // 更新
		adminAuthGroup.DELETE("/device-types/:id", deps.DeviceTypeHandler.DeleteDeviceType)   // 删除
		adminAuthGroup.DELETE("/device-types", deps.DeviceTypeHandler.BatchDeleteDeviceTypes) // 批量删除

		// 所有标准的传感器
		adminAuthGroup.GET("/standard-sensors", deps.StandardSensorHandler.List)           // 获取列表
		adminAuthGroup.GET("/standard-sensors/:id", deps.StandardSensorHandler.Get)        // 获取详情
		adminAuthGroup.POST("/standard-sensors", deps.StandardSensorHandler.Create)        // 创建
		adminAuthGroup.PUT("/standard-sensors/:id", deps.StandardSensorHandler.Update)     // 更新
		adminAuthGroup.DELETE("/standard-sensors/:id", deps.StandardSensorHandler.Delete)  // 删除
		adminAuthGroup.DELETE("/standard-sensors", deps.StandardSensorHandler.BatchDelete) // 批量删除

		// 所有标准的继电器
		adminAuthGroup.GET("/standard-relays", deps.StandardRelayHandler.List)           // 获取列表
		adminAuthGroup.GET("/standard-relays/:id", deps.StandardRelayHandler.Get)        // 获取详情
		adminAuthGroup.POST("/standard-relays", deps.StandardRelayHandler.Create)        // 创建
		adminAuthGroup.PUT("/standard-relays/:id", deps.StandardRelayHandler.Update)     // 更新
		adminAuthGroup.DELETE("/standard-relays/:id", deps.StandardRelayHandler.Delete)  // 删除
		adminAuthGroup.DELETE("/standard-relays", deps.StandardRelayHandler.BatchDelete) // 批量删除

		// 公司管理路由
		adminAuthGroup.GET("/companies", deps.CompanyHandler.List)           // 获取公司列表
		adminAuthGroup.GET("/companies/:id", deps.CompanyHandler.Get)        // 获取公司详情
		adminAuthGroup.POST("/companies", deps.CompanyHandler.Create)        // 创建公司
		adminAuthGroup.PUT("/companies/:id", deps.CompanyHandler.Update)     // 更新公司信息
		adminAuthGroup.DELETE("/companies/:id", deps.CompanyHandler.Delete)  // 删除公司
		adminAuthGroup.DELETE("/companies", deps.CompanyHandler.BatchDelete) // 批量删除公司

		// 企业管理用户路由
		adminAuthGroup.GET("/companies/:id/users", deps.CompanyHandler.ListCompanyUsers)                     // 获取企业用户列表
		adminAuthGroup.GET("/companies/:id/users/not-in", deps.CompanyHandler.ListUsersNotInCompany)         // 获取企业不存在用户列表
		adminAuthGroup.POST("/companies/:id/users", deps.CompanyHandler.AddUserToCompany)                    // 为企业添加用户
		adminAuthGroup.DELETE("/companies/:id/users/batch", deps.CompanyHandler.BatchRemoveUsersFromCompany) // 批量删除企业用户关联
		adminAuthGroup.DELETE("/companies/:id/users/:userId", deps.CompanyHandler.RemoveUserFromCompany)     // 删除企业用户关联
		adminAuthGroup.PUT("/companies/:id/users/:userId/role", deps.CompanyHandler.UpdateUserRoleInCompany) // 更新企业用户角色

		// ========================================================================
		// 用户管理路由（管理员专用，不涉及CompanyID和Role，直接操作User实体）
		// ========================================================================
		adminAuthGroup.GET("/users", deps.UserHandler.ListByAdmin)           // 获取用户列表
		adminAuthGroup.POST("/users", deps.UserHandler.CreateByAdmin)        // 创建用户
		adminAuthGroup.PUT("/users/:id", deps.UserHandler.UpdateByAdmin)     // 更新用户信息
		adminAuthGroup.DELETE("/users/:id", deps.UserHandler.DeleteByAdmin)  // 删除用户
		adminAuthGroup.DELETE("/users", deps.UserHandler.BatchDeleteByAdmin) // 批量删除用户

		// 小程序功能栏配置管理（增删改查，删除需要彻底删除）
		adminAuthGroup.GET("/miniprogram-menus", deps.MiniProgramMenuHandler.List)           // 获取列表
		adminAuthGroup.GET("/miniprogram-menus/:id", deps.MiniProgramMenuHandler.Get)        // 获取详情
		adminAuthGroup.POST("/miniprogram-menus", deps.MiniProgramMenuHandler.Create)        // 创建
		adminAuthGroup.PUT("/miniprogram-menus/:id", deps.MiniProgramMenuHandler.Update)     // 更新
		adminAuthGroup.DELETE("/miniprogram-menus/:id", deps.MiniProgramMenuHandler.Delete)  // 删除（彻底删除）
		adminAuthGroup.DELETE("/miniprogram-menus", deps.MiniProgramMenuHandler.BatchDelete) // 批量删除（彻底删除）

		// 设备模拟器管理
		adminAuthGroup.GET("/device-simulators", deps.DeviceSimulatorHandler.List)           // 分页查询列表
		adminAuthGroup.POST("/device-simulators", deps.DeviceSimulatorHandler.Create)        // 新增设备SN号
		adminAuthGroup.DELETE("/device-simulators", deps.DeviceSimulatorHandler.BatchDelete) // 批量删除设备SN号（物理删除）

		// 水肥模板管理
		adminAuthGroup.GET("/water-fertilizer-templates", deps.WaterFertilizerTemplateHandler.List)           // 获取水肥模板列表
		adminAuthGroup.GET("/water-fertilizer-templates/:id", deps.WaterFertilizerTemplateHandler.Get)        // 获取水肥模板详情
		adminAuthGroup.POST("/water-fertilizer-templates", deps.WaterFertilizerTemplateHandler.Create)        // 创建水肥模板
		adminAuthGroup.PUT("/water-fertilizer-templates/:id", deps.WaterFertilizerTemplateHandler.Update)     // 更新水肥模板
		adminAuthGroup.DELETE("/water-fertilizer-templates/:id", deps.WaterFertilizerTemplateHandler.Delete)  // 删除水肥模板
		adminAuthGroup.DELETE("/water-fertilizer-templates", deps.WaterFertilizerTemplateHandler.BatchDelete) // 批量删除水肥模板

		// 部署管理
		adminAuthGroup.POST("/deploy", deps.DeployHandler.Deploy) // 部署 dist 压缩包
	}

	if config.Conf.App.Swagger {
		// 注册 Swagger 文档路由
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	return r
}
