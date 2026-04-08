package repository

import (
	"github.com/google/wire"
)

// RepositorySet 仓储层集合 注入
var RepoSet = wire.NewSet(
	wire.NewSet(wire.Struct(new(UserRepository), "*")),
	wire.NewSet(wire.Struct(new(DeviceTypeRepository), "*")),
	wire.NewSet(NewStandardRelayRepository),
	wire.NewSet(NewStandardSensorRepository),
	wire.NewSet(NewDeviceRepository),
	wire.NewSet(wire.Struct(new(SiteRepository), "*")),
	wire.NewSet(wire.Struct(new(MyManagementRepository), "*")),
	wire.NewSet(wire.Struct(new(DeviceScreenshotRepository), "*")),
	wire.NewSet(wire.Struct(new(CompanyRepository), "*")),
	wire.NewSet(wire.Struct(new(UserCompanyRepository), "*")),
	wire.NewSet(NewUserSiteRepository),
	wire.NewSet(NewOSSClient),
	wire.NewSet(NewDeviceAlarmRepository),
	wire.NewSet(NewDeviceRelayRecordRepository),
	wire.NewSet(NewMiniProgramMenuRepository),
	wire.NewSet(NewDeviceSimulatorRepository),
	wire.NewSet(NewWaterFertilizerTemplateRepository),
)
