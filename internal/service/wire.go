package service

import (
	"github.com/google/wire"
)

// ServiceSet 服务层集合 注入
var ServiceSet = wire.NewSet(
	wire.NewSet(NewAuthService),
	wire.NewSet(wire.Struct(new(UserService), "*")),
	wire.NewSet(wire.Struct(new(AliyunOSSService), "*")),
	wire.NewSet(wire.Struct(new(DeviceTypeService), "*")),
	wire.NewSet(wire.Struct(new(StandardSensorService), "*")),
	wire.NewSet(wire.Struct(new(StandardRelayService), "*")),
	wire.NewSet(wire.Struct(new(DeviceService), "*")),
	wire.NewSet(NewSiteService),
	wire.NewSet(NewMyManagementService),
	wire.NewSet(wire.Struct(new(DeviceDataService), "*")),
	wire.NewSet(NewWeatherService),
	wire.NewSet(wire.Struct(new(DeviceScreenshotService), "*")),
	wire.NewSet(wire.Struct(new(CompanyService), "*")),
	wire.NewSet(NewDeviceAlarmService),
	wire.NewSet(NewDeviceRelayRecordService),
	wire.NewSet(wire.Struct(new(MiniProgramMenuService), "*")),
	wire.NewSet(wire.Struct(new(DeviceSimulatorService), "*")),
	wire.NewSet(wire.Struct(new(WaterFertilizerTemplateService), "*")),
)
