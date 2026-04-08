package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"rbac-admin/internal/model/entity"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/contextx"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
)

const (
	deviceConfigKey     = "device:%d:config"
	deviceConfigLockKey = "lock:device:%d:config"

	deviceConfigTTL = 24 * time.Hour

	// 【统一配置】锁的生命周期 (防止死锁)
	deviceConfigLockTTL = 5 * time.Second
	// 【统一配置】抢锁失败后的休眠时间 (轮询间隔)
	deviceConfigRetryDelay = 1 * time.Second
	// 【统一配置】最大重试次数 (3次 * 1s < 5s，确保逻辑闭环)
	deviceConfigMaxRetries = 3
)

// DeviceConfigCache handles caching for device configuration.
type DeviceConfigCache struct {
	client *redis.Client
	rs     *redsync.Redsync
	repo   *repository.DeviceRepository
}

// NewDeviceConfigCache creates a new DeviceConfigCache.
func NewDeviceConfigCache(client *redis.Client, rs *redsync.Redsync, repo *repository.DeviceRepository) *DeviceConfigCache {
	return &DeviceConfigCache{
		client: client,
		rs:     rs,
		repo:   repo,
	}
}

// GetDeviceConfigKey returns the Redis key for device configuration.
func GetDeviceConfigKey(deviceID uint32) string {
	return fmt.Sprintf(deviceConfigKey, deviceID)
}

// DelConfig deletes the configuration for a device from the cache.
func (c *DeviceConfigCache) DelConfig(ctx context.Context, deviceID uint32) error {
	key := GetDeviceConfigKey(deviceID)
	return c.client.Del(ctx, key).Err()
}

// GetConfig 获取设备配置信息。
// 采用 Anti-Stampede (Manual Loop + TryLock) 模式。
func (c *DeviceConfigCache) GetConfig(ctx context.Context, deviceID uint32) (*entity.Device, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	key := GetDeviceConfigKey(deviceID)
	lockKey := fmt.Sprintf(deviceConfigLockKey, deviceID)

	// --- 循环重试机制 ---
	for i := 0; i < deviceConfigMaxRetries; i++ {
		// 1. 尝试从缓存获取
		config, err := c.getFromCache(ctx, key)
		if err == nil {
			return config, nil
		} else if err != redis.Nil {
			logger.Warn(fmt.Sprintf("failed to get device config from cache: %v", err))
		}

		// 2. 缓存未命中，尝试获取锁 (TryLock)
		// WithTries(1): 只尝试一次，如果锁被占用立即返回错误，不排队
		mutex := c.rs.NewMutex(lockKey,
			redsync.WithExpiry(deviceConfigLockTTL),
			redsync.WithTries(1),
		)

		if err := mutex.Lock(); err != nil {
			// 3. 抢锁失败
			// 说明别人正在查库，不需要排队，只需要睡一会，等下一次循环直接读缓存
			logger.Debug(fmt.Sprintf("lock busy for device %d, sleeping... (attempt %d)", deviceID, i+1))
			time.Sleep(deviceConfigRetryDelay)
			continue
		}

		// 4. 抢锁成功，执行查库逻辑
		var result *entity.Device
		var dbErr error

		func() {
			// 确保锁释放
			defer func() {
				if _, err := mutex.Unlock(); err != nil {
					logger.Warn(fmt.Sprintf("failed to unlock device %d: %v", deviceID, err))
				}
			}()

			// 4.1 双重检查 (Double Check)
			// 防止在抢锁瞬间别人已经写入了缓存
			if cached, err := c.getFromCache(ctx, key); err == nil {
				result = cached
				return
			}

			// 4.2 查库并回写
			result, dbErr = c.fetchAndCache(ctx, deviceID, key)
		}()

		if dbErr != nil {
			return nil, dbErr
		}
		return result, nil
	}

	// 5. 兜底策略
	// 如果重试次数用尽（约 3 秒）还没拿到数据，降级直接查库，保证业务可用性
	logger.Warn("cache retry limit reached, fetching device config from db directly")
	return c.fetchFromDB(ctx, deviceID)
}

// getFromCache 封装了读取和反序列化逻辑，并包含脏数据清理功能
func (c *DeviceConfigCache) getFromCache(ctx context.Context, key string) (*entity.Device, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var config entity.Device
	// 【优化关键】: JSON 反序列化与脏数据自愈
	if err := json.Unmarshal(val, &config); err != nil {
		c.client.Del(ctx, key)
		return nil, fmt.Errorf("invalid cache data deleted: %w", err)
	}

	return &config, nil
}

// fetchFromDB 仅负责查库
func (c *DeviceConfigCache) fetchFromDB(ctx context.Context, deviceID uint32) (*entity.Device, error) {
	return c.repo.GetByDeviceNumFromDB(ctx, uint(deviceID))
}

// fetchAndCache 负责查库并回写缓存
func (c *DeviceConfigCache) fetchAndCache(ctx context.Context, deviceID uint32, key string) (*entity.Device, error) {
	config, err := c.fetchFromDB(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	// 序列化
	jsonData, err := json.Marshal(config)
	if err != nil {
		contextx.GetLoggerFromContext(ctx).Error(fmt.Sprintf("failed to serialize device config: %v", err))
		return config, nil
	}

	// 【优化关键】: 非阻塞写入
	// 缓存写入失败仅记录日志，优先保证业务数据返回
	err = c.client.Set(ctx, key, jsonData, deviceConfigTTL).Err()
	if err != nil {
		contextx.GetLoggerFromContext(ctx).Warn(fmt.Sprintf("failed to write cache for key %s: %v", key, err))
	}

	return config, nil
}
