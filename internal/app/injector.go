package app

import (
	"rbac-admin/pkg/collector"
	"rbac-admin/pkg/devicesimulator"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

// InjectorSet 注入Injector
var InjectorSet = wire.NewSet(wire.Struct(new(Injector), "*"))

// Injector 注入器(用于初始化完成之后的引用)
type Injector struct {
	Engine                 *gin.Engine
	Collector              *collector.CollectorApp
	DeviceSimulatorManager *devicesimulator.DeviceSimulatorManager
	AppScheduler           *AppScheduler
}
