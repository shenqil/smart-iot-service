package cache

import (
	"github.com/google/wire"
)

// ProviderSet is a set of providers for the cache module.
var ProviderSet = wire.NewSet(
	NewRedisClient,
	NewRedsync,
	NewDeviceDataCache,
	NewDeviceConfigCache,
	NewDeviceRelayRecordCache,
	NewUserRoleCache,
	NewUserSiteCache,
	NewDeviceTypesCache,
	NewStandardSensorsCache,
	NewStandardRelaysCache,
)
