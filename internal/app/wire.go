//go:build wireinject
// +build wireinject

package app

import (
	"rbac-admin/internal/cache"
	"rbac-admin/internal/handler"
	"rbac-admin/internal/repository"
	"rbac-admin/internal/router"
	"rbac-admin/internal/service"
	"rbac-admin/pkg/collector"
	"rbac-admin/pkg/database"
	"rbac-admin/pkg/devicesimulator"
	"rbac-admin/pkg/influxdb"
	"rbac-admin/pkg/jwt"
	"rbac-admin/pkg/pubsubRedis"
	"rbac-admin/pkg/scheduler"

	"github.com/google/wire"
)

// InitializeApp 初始化应用依赖
func InitializeApp() (*Injector, func(), error) {
	wire.Build(
		database.InitDatabase,
		influxdb.NewInfluxdb3,
		pubsubRedis.NewEventCollector,
		repository.RepoSet,
		cache.ProviderSet,
		service.ServiceSet,
		handler.HandlerSet,
		router.RouterSet,
		router.SetupRouter,
		InjectorSet,
		collector.NewCollectorApp,
		devicesimulator.NewDeviceSimulatorManager,
		jwt.NewJWT,
		scheduler.NewScheduler,
		SchedulerSet,
	)
	return new(Injector), func() {}, nil
}
