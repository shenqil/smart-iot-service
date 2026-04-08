package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"rbac-admin/internal/model/dto"
	"rbac-admin/internal/model/entity"
	"rbac-admin/internal/repository"
	"rbac-admin/pkg/contextx"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
)

const (
	standardRelaysCacheKey = "config:standard-relays:all"
	standardRelaysLockKey  = "lock:config:standard-relays"

	// 缓存有效期 (配置类数据，有效期较长)
	standardRelaysTTL = 24 * time.Hour

	// 【统一配置】锁的生命周期
	standardRelaysLockTTL = 5 * time.Second
	// 【统一配置】轮询间隔
	standardRelaysRetryDelay = 1 * time.Second
	// 【统一配置】最大重试次数
	standardRelaysMaxRetries = 3
)

// StandardRelaysCache 处理标准继电器缓存
type StandardRelaysCache struct {
	client *redis.Client
	rs     *redsync.Redsync
	repo   *repository.StandardRelayRepository
}

func NewStandardRelaysCache(client *redis.Client, rs *redsync.Redsync, repo *repository.StandardRelayRepository) *StandardRelaysCache {
	return &StandardRelaysCache{
		client: client,
		rs:     rs,
		repo:   repo,
	}
}

// GetStandardRelays 获取标准继电器列表
// 模式：Manual Loop + TryLock (Anti-Stampede)
func (c *StandardRelaysCache) GetStandardRelays(ctx context.Context) ([]*dto.StandardRelayResponse, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// --- 循环重试机制 ---
	for i := 0; i < standardRelaysMaxRetries; i++ {
		// 1. 尝试从缓存获取
		data, err := c.getFromCache(ctx)
		if err == nil {
			return data, nil
		} else if err != redis.Nil {
			logger.Warn(fmt.Sprintf("failed to get standard relays from cache: %v", err))
		}

		// 2. 缓存未命中，尝试获取锁 (TryLock)
		// WithTries(1): 不排队，抢不到立即返回错误
		mutex := c.rs.NewMutex(standardRelaysLockKey,
			redsync.WithExpiry(standardRelaysLockTTL),
			redsync.WithTries(1),
		)

		if err := mutex.Lock(); err != nil {
			// 3. 抢锁失败
			// 说明有人正在重建缓存，休眠 1s 后重试
			logger.Debug(fmt.Sprintf("lock busy for standard relays, sleeping... (attempt %d)", i+1))
			time.Sleep(standardRelaysRetryDelay)
			continue
		}

		// 4. 抢锁成功，执行重建逻辑
		var result []*dto.StandardRelayResponse
		var dbErr error

		func() {
			// 确保释放锁
			defer func() {
				if _, err := mutex.Unlock(); err != nil {
					logger.Warn(fmt.Sprintf("failed to unlock standard relays: %v", err))
				}
			}()

			// 4.1 双重检查 (Double Check)
			// 防止在抢锁瞬间别人已经写入了
			if cached, err := c.getFromCache(ctx); err == nil {
				result = cached
				return
			}

			// 4.2 查库并回写
			result, dbErr = c.fetchAndCache(ctx)
		}()

		if dbErr != nil {
			return nil, dbErr
		}
		return result, nil
	}

	// 5. 兜底策略
	// 重试次数用尽，降级查库
	logger.Warn("cache retry limit reached, fetching standard relays from db directly")
	return c.loadAllFromDB(ctx)
}

// UpdateStandardRelays 数据变更时调用
// 策略：Cache-Aside (直接删除缓存)
func (c *StandardRelaysCache) UpdateStandardRelays(ctx context.Context) error {
	return c.client.Del(ctx, standardRelaysCacheKey).Err()
}

// RefreshCache forces a refresh of the cache from the database.
func (c *StandardRelaysCache) RefreshCache(ctx context.Context) ([]*dto.StandardRelayResponse, error) {
	return c.fetchAndCache(ctx)
}

// --- 私有辅助方法 ---

// getFromCache 读取缓存 + 脏数据清理
func (c *StandardRelaysCache) getFromCache(ctx context.Context) ([]*dto.StandardRelayResponse, error) {
	val, err := c.client.Get(ctx, standardRelaysCacheKey).Bytes()
	if err != nil {
		return nil, err
	}

	var responses []*dto.StandardRelayResponse
	if err := json.Unmarshal(val, &responses); err != nil {
		c.client.Del(ctx, standardRelaysCacheKey)
		return nil, fmt.Errorf("invalid cache data deleted: %w", err)
	}

	return responses, nil
}

// fetchAndCache 查库 + 序列化 + 写缓存
func (c *StandardRelaysCache) fetchAndCache(ctx context.Context) ([]*dto.StandardRelayResponse, error) {
	// 1. 查库
	responses, err := c.loadAllFromDB(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 序列化
	jsonData, err := json.Marshal(responses)
	if err != nil {
		contextx.GetLoggerFromContext(ctx).Error(fmt.Sprintf("failed to serialize standard relays: %v", err))
		return responses, nil
	}

	// 3. 写缓存 (非阻塞)
	err = c.client.Set(ctx, standardRelaysCacheKey, jsonData, standardRelaysTTL).Err()
	if err != nil {
		contextx.GetLoggerFromContext(ctx).Warn(fmt.Sprintf("failed to write cache for standard relays: %v", err))
	}

	return responses, nil
}

// loadAllFromDB 分页加载所有数据
func (c *StandardRelaysCache) loadAllFromDB(ctx context.Context) ([]*dto.StandardRelayResponse, error) {
	var allEntities []*entity.StandardRelay
	pageNumber := 1
	const pageSize = 500 // 调大 PageSize

	for {
		query := entity.StandardRelayQuery{
			Pagination: entity.Pagination{
				PageNumber: pageNumber,
				PageSize:   pageSize,
			},
		}
		entities, total, err := c.repo.List(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("load db failed: %w", err)
		}
		allEntities = append(allEntities, entities...)

		if int64(len(allEntities)) >= total {
			break
		}
		if len(entities) == 0 {
			break
		}
		pageNumber++
	}

	// 转换为 DTO
	// 假设 dto.ConvertToStandardRelayResponseList 处理了 nil 检查
	// 如果没有该批量转换方法，可在此处遍历转换
	result, err := dto.ConvertToStandardRelayResponseList(allEntities)
	if err != nil {
		return nil, err
	}

	// 确保不返回 nil slice
	if result == nil {
		result = []*dto.StandardRelayResponse{}
	}

	return result, nil
}
