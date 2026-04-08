package handler

import (
	"github.com/google/wire"
)

// HandlerSet 控制器层集合 注入
var HandlerSet = wire.NewSet(
	wire.NewSet(wire.Struct(new(AuthHandler), "*")),
	wire.NewSet(wire.Struct(new(UserHandler), "*")),
	wire.NewSet(wire.Struct(new(AliyunOSSHandler), "*")),
	wire.NewSet(wire.Struct(new(DeviceTypeHandler), "*")),
	wire.NewSet(wire.Struct(new(StandardSensorHandler), "*")),
	wire.NewSet(wire.Struct(new(StandardRelayHandler), "*")),
	wire.NewSet(wire.Struct(new(DeviceHandler), "*")),
	wire.NewSet(wire.Struct(new(SiteHandler), "*")),
	wire.NewSet(wire.Struct(new(MyManagementHandler), "*")),
	wire.NewSet(wire.Struct(new(WeatherHandler), "*")),
	wire.NewSet(wire.Struct(new(DeviceScreenshotHandler), "*")),
	wire.NewSet(wire.Struct(new(CompanyHandler), "*")),
	wire.NewSet(wire.Struct(new(DeviceAlarmHandler), "*")),
	wire.NewSet(wire.Struct(new(MiniProgramMenuHandler), "*")),
	wire.NewSet(wire.Struct(new(DeviceSimulatorHandler), "*")),
	wire.NewSet(wire.Struct(new(WaterFertilizerTemplateHandler), "*")),
	wire.NewSet(wire.Struct(new(DeployHandler), "*")),
)
