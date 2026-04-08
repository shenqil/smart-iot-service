package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"rbac-admin/internal/cache/common"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/contextx"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
)

const (
	userCompanySitesKey     = "user:%d:company:%d:sites"
	userCompanySitesLockKey = "lock:user:%d:company:%d:sites"

	// 统一配置：锁的生命周期 (防止死锁)
	userCompanySitesLockTTL = 5 * time.Second
	// 统一配置：抢锁失败后的休眠时间 (轮询间隔)
	userCompanySitesRetryDelay = 1 * time.Second
	// 统一配置：最大轮询次数 ( 3次 * 1s < 5s，确保安全)
	userCompanySitesMaxRetries = 3
)

type UserSiteCache struct {
	client       *redis.Client
	rs           *redsync.Redsync
	userSiteRepo *repository.UserSiteRepository
}

func NewUserSiteCache(
	client *redis.Client,
	rs *redsync.Redsync,
	userSiteRepo *repository.UserSiteRepository,
) *UserSiteCache {
	return &UserSiteCache{
		client:       client,
		rs:           rs,
		userSiteRepo: userSiteRepo,
	}
}

func GetUserSitesCacheKey(userID, companyID uint) string {
	return fmt.Sprintf(userCompanySitesKey, userID, companyID)
}

// GetUserSiteIDs 获取用户站点列表 (Anti-Stampede / 防击穿模式)
func (c *UserSiteCache) GetUserSiteIDs(ctx context.Context, userID, companyID uint) ([]uint, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	key := GetUserSitesCacheKey(userID, companyID)
	lockKey := fmt.Sprintf(userCompanySitesLockKey, userID, companyID)

	// --- 循环重试机制 ---
	// 相比 Redsync 内部重试，手动循环能避免请求串行排队，减少 Redis 压力
	for i := 0; i < userCompanySitesMaxRetries; i++ {

		// 1. 尝试从缓存获取
		siteIDs, err := c.getFromCache(ctx, key)
		if err == nil {
			return siteIDs, nil
		} else if err != redis.Nil {
			logger.Warn(fmt.Sprintf("failed to get site IDs from cache: %v", err))
		}

		// 2. 缓存未命中，尝试获取锁 (TryLock 模式)
		// WithTries(1): 只尝试一次，如果锁被占用立即返回错误，不排队
		mutex := c.rs.NewMutex(lockKey,
			redsync.WithExpiry(userCompanySitesLockTTL),
			redsync.WithTries(1),
		)

		if err := mutex.Lock(); err != nil {
			// 3. 抢锁失败
			// 说明别人正在查库，不需要排队，只需要睡一会，等下一次循环直接读缓存
			logger.Debug(fmt.Sprintf("lock busy for user %d site %d, sleeping... (attempt %d)", userID, companyID, i+1))
			time.Sleep(userCompanySitesRetryDelay)
			continue
		}

		// 4. 抢锁成功，执行查库逻辑
		var result []uint
		var dbErr error

		func() {
			// 确保锁释放
			defer func() {
				if _, err := mutex.Unlock(); err != nil {
					logger.Warn(fmt.Sprintf("failed to unlock user sites: %v", err))
				}
			}()

			// 4.1 双重检查 (Double Check)
			// 防止在抢锁瞬间别人已经写入了缓存
			if cached, err := c.getFromCache(ctx, key); err == nil {
				result = cached
				return
			}

			// 4.2 查库并回写
			result, dbErr = c.fetchAndCacheUserSiteIDs(ctx, userID, companyID, key)
		}()

		if dbErr != nil {
			return nil, dbErr
		}
		return result, nil
	}

	// 5. 兜底策略
	// 如果重试 3 次（约 3 秒）还没拿到数据，降级直接查库，保证业务可用性
	logger.Warn("cache retry limit reached, fetching from db directly")
	return c.fetchUserSiteIDsFromDB(ctx, userID, companyID)
}

// DelUserSites 删除缓存
func (c *UserSiteCache) DelUserSites(ctx context.Context, userID, companyID uint) error {
	key := GetUserSitesCacheKey(userID, companyID)
	return c.client.Del(ctx, key).Err()
}

// --- 私有辅助方法 ---

// getFromCache 封装读缓存 + 脏数据自愈
func (c *UserSiteCache) getFromCache(ctx context.Context, key string) ([]uint, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var siteIDs []uint
	if err := json.Unmarshal([]byte(val), &siteIDs); err != nil {
		// JSON 解析失败意味着缓存损坏，主动清理
		c.client.Del(ctx, key)
		return nil, fmt.Errorf("invalid cache data deleted: %w", err)
	}

	return siteIDs, nil
}

// fetchUserSiteIDsFromDB 纯查库逻辑 + 空值处理
func (c *UserSiteCache) fetchUserSiteIDsFromDB(ctx context.Context, userID, companyID uint) ([]uint, error) {
	siteIDs, err := c.userSiteRepo.GetUserSiteIDs(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}
	// 规范化：确保不返回 nil 切片，避免 json 序列化为 "null"
	if siteIDs == nil {
		siteIDs = []uint{}
	}
	return siteIDs, nil
}

// fetchAndCacheUserSiteIDs 查库 + 序列化 + 非阻塞写缓存
func (c *UserSiteCache) fetchAndCacheUserSiteIDs(ctx context.Context, userID, companyID uint, key string) ([]uint, error) {
	siteIDs, err := c.fetchUserSiteIDsFromDB(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}

	siteIDsJSON, err := json.Marshal(siteIDs)
	if err != nil {
		contextx.GetLoggerFromContext(ctx).Error(fmt.Sprintf("failed to marshal site IDs: %v", err))
		return siteIDs, nil
	}

	// 写入缓存，错误仅记录日志
	err = c.client.Set(ctx, key, siteIDsJSON, common.UserCacheTTL).Err()
	if err != nil {
		contextx.GetLoggerFromContext(ctx).Warn(fmt.Sprintf("failed to write cache for key %s: %v", key, err))
	}

	return siteIDs, nil
}
