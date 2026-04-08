package app

import (
	"context"
	"time"

	"rbac-admin/internal/cache"
	"rbac-admin/pkg/logger"
	"rbac-admin/pkg/scheduler"

	"github.com/google/wire"
)

var SchedulerSet = wire.NewSet(NewAppScheduler)

type AppScheduler struct {
	scheduler           *scheduler.Scheduler
	deviceTypesCache    *cache.DeviceTypesCache
	standardRelaysCache *cache.StandardRelaysCache
	standardSensorsCache *cache.StandardSensorsCache
}

func NewAppScheduler(
	scheduler *scheduler.Scheduler,
	deviceTypesCache *cache.DeviceTypesCache,
	standardRelaysCache *cache.StandardRelaysCache,
	standardSensorsCache *cache.StandardSensorsCache,
) *AppScheduler {
	return &AppScheduler{
		scheduler:           scheduler,
		deviceTypesCache:    deviceTypesCache,
		standardRelaysCache: standardRelaysCache,
		standardSensorsCache: standardSensorsCache,
	}
}

func (s *AppScheduler) Start() {
	// Schedule recurring jobs
	s.scheduleDailyJobs()

	// Schedule one-off jobs after startup
	s.scheduleStartupJobs()

	s.scheduler.Start()
	logger.Logger.Info("Scheduler started.")
}

func (s *AppScheduler) Stop() {
	s.scheduler.Stop()
	logger.Logger.Info("Scheduler stopped.")
}

func (s *AppScheduler) scheduleDailyJobs() {
	ctx := context.Background()
	// Schedule at 3:00 AM every day
	spec := "0 3 * * *"

	_, err := s.scheduler.AddFunc(spec, func() {
		logger.Logger.Info("Running daily job: Refresh DeviceTypesCache")
		_, refreshErr := s.deviceTypesCache.RefreshCache(ctx)
		if refreshErr != nil {
			logger.Logger.Error("Failed to refresh DeviceTypesCache", "error", refreshErr)
		}
	})
	if err != nil {
		logger.Logger.Error("Failed to schedule DeviceTypesCache refresh", "error", err)
	}

	_, err = s.scheduler.AddFunc(spec, func() {
		logger.Logger.Info("Running daily job: Refresh StandardRelaysCache")
		_, refreshErr := s.standardRelaysCache.RefreshCache(ctx)
		if refreshErr != nil {
			logger.Logger.Error("Failed to refresh StandardRelaysCache", "error", refreshErr)
		}
	})
	if err != nil {
		logger.Logger.Error("Failed to schedule StandardRelaysCache refresh", "error", err)
	}

	_, err = s.scheduler.AddFunc(spec, func() {
		logger.Logger.Info("Running daily job: Refresh StandardSensorsCache")
		_, refreshErr := s.standardSensorsCache.RefreshCache(ctx)
		if refreshErr != nil {
			logger.Logger.Error("Failed to refresh StandardSensorsCache", "error", refreshErr)
		}
	})
	if err != nil {
		logger.Logger.Error("Failed to schedule StandardSensorsCache refresh", "error", err)
	}
}

func (s *AppScheduler) scheduleStartupJobs() {
	ctx := context.Background()
	go func() {
		time.Sleep(1 * time.Minute)

		logger.Logger.Info("Running startup job: Refresh DeviceTypesCache")
		_, err := s.deviceTypesCache.RefreshCache(ctx)
		if err != nil {
			logger.Logger.Error("Failed to refresh DeviceTypesCache on startup", "error", err)
		}

		logger.Logger.Info("Running startup job: Refresh StandardRelaysCache")
		_, err = s.standardRelaysCache.RefreshCache(ctx)
		if err != nil {
			logger.Logger.Error("Failed to refresh StandardRelaysCache on startup", "error", err)
		}

		logger.Logger.Info("Running startup job: Refresh StandardSensorsCache")
		_, err = s.standardSensorsCache.RefreshCache(ctx)
		if err != nil {
			logger.Logger.Error("Failed to refresh StandardSensorsCache on startup", "error", err)
		}
	}()
}
