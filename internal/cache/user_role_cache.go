package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"rbac-admin/internal/cache/common"
	"rbac-admin/internal/model/entity"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/contextx"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
)

const (
	userCompanyRoleKey = "user:%d:company:%d:role"
	userRoleLockKey    = "lock:user:%d:company:%d:role"

	// 【统一配置】锁的生命周期 (防止死锁)
	userRoleLockTTL = 5 * time.Second
	// 【统一配置】抢锁失败后的休眠时间 (轮询间隔)
	userRoleRetryDelay = 1 * time.Second
	// 【统一配置】最大重试次数 (3次 * 1s < 5s，确保逻辑闭环)
	userRoleMaxRetries = 3
)

// UserRoleCache handles caching for user role information.
type UserRoleCache struct {
	client          *redis.Client
	rs              *redsync.Redsync
	userCompanyRepo *repository.UserCompanyRepository
}

// NewUserRoleCache creates a new UserRoleCache.
func NewUserRoleCache(
	client *redis.Client,
	rs *redsync.Redsync,
	userCompanyRepo *repository.UserCompanyRepository,
) *UserRoleCache {
	return &UserRoleCache{
		client:          client,
		rs:              rs,
		userCompanyRepo: userCompanyRepo,
	}
}

// GetRoleCacheKey returns the Redis key for a user's role in a company.
func GetRoleCacheKey(userID, companyID uint) string {
	return fmt.Sprintf(userCompanyRoleKey, userID, companyID)
}

// GetRole retrieves a user's role for a specific company from the cache.
// Uses the Anti-Stampede pattern (Manual Loop + TryLock).
func (c *UserRoleCache) GetRole(ctx context.Context, userID, companyID uint) (entity.UserRole, error) {
	logger := contextx.GetLoggerFromContext(ctx)
	key := GetRoleCacheKey(userID, companyID)
	lockKey := fmt.Sprintf(userRoleLockKey, userID, companyID)

	// --- 循环重试机制 ---
	for i := 0; i < userRoleMaxRetries; i++ {
		// 1. 尝试从缓存获取
		role, err := c.getFromCache(ctx, key)
		if err == nil {
			return role, nil
		} else if err != redis.Nil {
			logger.Warn(fmt.Sprintf("failed to get role from cache: %v", err))
		}

		// 2. 缓存未命中，尝试获取锁 (TryLock)
		// WithTries(1): 只尝试一次，如果锁被占用立即返回错误，不排队
		mutex := c.rs.NewMutex(lockKey,
			redsync.WithExpiry(userRoleLockTTL),
			redsync.WithTries(1),
		)

		if err := mutex.Lock(); err != nil {
			// 3. 抢锁失败
			// 说明别人正在查库，不需要排队，只需要睡一会，等下一次循环直接读缓存
			logger.Debug(fmt.Sprintf("lock busy for user %d role, sleeping... (attempt %d)", userID, i+1))
			time.Sleep(userRoleRetryDelay)
			continue
		}

		// 4. 抢锁成功，执行查库逻辑
		var result entity.UserRole
		var dbErr error

		func() {
			// 确保锁释放
			defer func() {
				if _, err := mutex.Unlock(); err != nil {
					logger.Warn(fmt.Sprintf("failed to unlock user role: %v", err))
				}
			}()

			// 4.1 双重检查 (Double Check)
			// 防止在抢锁瞬间别人已经写入了缓存
			if cached, err := c.getFromCache(ctx, key); err == nil {
				result = cached
				return
			}

			// 4.2 查库并回写
			result, dbErr = c.fetchAndCacheRole(ctx, userID, companyID, key)
		}()

		if dbErr != nil {
			return 0, dbErr
		}
		return result, nil
	}

	// 5. 兜底策略
	// 如果重试 3 次（约 3 秒）还没拿到数据，降级直接查库，保证业务可用性
	logger.Warn("cache retry limit reached, fetching role from db directly")
	return c.fetchRoleFromDB(ctx, userID, companyID)
}

// getFromCache encapsulates the parsing logic and cleans up dirty data.
func (c *UserRoleCache) getFromCache(ctx context.Context, key string) (entity.UserRole, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	roleInt, err := strconv.ParseInt(val, 10, 8)
	if err != nil {
		// 【优化】: 数据格式错误（脏数据），主动删除，防止后续一直报错
		c.client.Del(ctx, key)
		return 0, fmt.Errorf("invalid cache data deleted: %w", err)
	}

	return entity.UserRole(roleInt), nil
}

// DelRole deletes the role cache for a specific user and company.
func (c *UserRoleCache) DelRole(ctx context.Context, userID, companyID uint) error {

	key := GetRoleCacheKey(userID, companyID)
	return c.client.Del(ctx, key).Err()
}

// DelUserRoles deletes all role caches for a specific user across all companies.
func (c *UserRoleCache) DelUserRoles(ctx context.Context, userID uint) error {
	pattern := fmt.Sprintf("user:%d:company:*:role", userID)
	return c.deleteByPattern(ctx, pattern)
}

// DelCompanyRoles deletes all role caches for a specific company across all users.
func (c *UserRoleCache) DelCompanyRoles(ctx context.Context, companyID uint) error {
	pattern := fmt.Sprintf("user:*:company:%d:role", companyID)
	return c.deleteByPattern(ctx, pattern)
}

// deleteByPattern helps delete keys matching a pattern using Pipeline for performance.
func (c *UserRoleCache) deleteByPattern(ctx context.Context, pattern string) error {
	var keys []string
	var cursor uint64
	var err error

	// 1. Scan keys (collect all keys first)
	// 假设单个 User 或 Company 关联的角色数量在可控范围内（几千以内），SCAN 是安全的。
	for {
		var batch []string
		batch, cursor, err = c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		return nil
	}

	// 2. Batch Delete using Pipeline
	// 【优化】: 使用 Pipeline 减少网络 RTT
	pipe := c.client.Pipeline()
	for _, k := range keys {
		pipe.Del(ctx, k)
	}
	_, err = pipe.Exec(ctx)

	return err
}

func (c *UserRoleCache) fetchRoleFromDB(ctx context.Context, userID, companyID uint) (entity.UserRole, error) {
	userCompany, err := c.userCompanyRepo.GetByUserAndCompany(ctx, userID, companyID)
	if err != nil {
		return 0, err
	}
	return userCompany.Role, nil
}

func (c *UserRoleCache) fetchAndCacheRole(ctx context.Context, userID, companyID uint, key string) (entity.UserRole, error) {
	role, err := c.fetchRoleFromDB(ctx, userID, companyID)
	if err != nil {
		return 0, err
	}
	roleStr := strconv.FormatInt(int64(role), 10)

	// 写入缓存，错误仅记录日志，不影响主流程
	err = c.client.Set(ctx, key, roleStr, common.UserCacheTTL).Err()
	if err != nil {
		contextx.GetLoggerFromContext(ctx).Warn(fmt.Sprintf("failed to write cache for key %s: %v", key, err))
	}

	return role, nil
}
