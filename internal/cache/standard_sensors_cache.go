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
	standardSensorsCacheKey = "config:standard-sensors:all"
	standardSensorsLockKey  = "lock:config:standard-sensors"

	// 缓存有效期 (配置类数据，有效期较长)
	standardSensorsTTL = 24 * time.Hour

	// 【统一配置】锁的生命周期
	standardSensorsLockTTL = 5 * time.Second
	// 【统一配置】轮询间隔
	standardSensorsRetryDelay = 1 * time.Second
	// 【统一配置】最大重试次数
	standardSensorsMaxRetries = 3
)

// StandardSensorsCache 处理标准传感器缓存
type StandardSensorsCache struct {
	client *redis.Client
	rs     *redsync.Redsync
	repo   *repository.StandardSensorRepository
}

func NewStandardSensorsCache(client *redis.Client, rs *redsync.Redsync, repo *repository.StandardSensorRepository) *StandardSensorsCache {
	return &StandardSensorsCache{
		client: client,
		rs:     rs,
		repo:   repo,
	}
}

// GetStandardSensors 获取标准传感器列表
// 模式：Manual Loop + TryLock (Anti-Stampede)
func (c *StandardSensorsCache) GetStandardSensors(ctx context.Context) ([]*dto.StandardSensorResponse, error) {
	logger := contextx.GetLoggerFromContext(ctx)

	// --- 循环重试机制 ---
	for i := 0; i < standardSensorsMaxRetries; i++ {
		// 1. 尝试从缓存获取
		data, err := c.getFromCache(ctx)
		if err == nil {
			return data, nil
		} else if err != redis.Nil {
			logger.Warn(fmt.Sprintf("failed to get standard sensors from cache: %v", err))
		}

		// 2. 缓存未命中，尝试获取锁 (TryLock)
		// WithTries(1): 不排队，抢不到立即返回错误
		mutex := c.rs.NewMutex(standardSensorsLockKey,
			redsync.WithExpiry(standardSensorsLockTTL),
			redsync.WithTries(1),
		)

		if err := mutex.Lock(); err != nil {
			// 3. 抢锁失败
			// 说明有人正在重建缓存，休眠 1s 后重试
			logger.Debug(fmt.Sprintf("lock busy for standard sensors, sleeping... (attempt %d)", i+1))
			time.Sleep(standardSensorsRetryDelay)
			continue
		}

		// 4. 抢锁成功，执行重建逻辑
		var result []*dto.StandardSensorResponse
		var dbErr error

		func() {
			// 确保释放锁
			defer func() {
				if _, err := mutex.Unlock(); err != nil {
					logger.Warn(fmt.Sprintf("failed to unlock standard sensors: %v", err))
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
	logger.Warn("cache retry limit reached, fetching standard sensors from db directly")
	return c.loadAllFromDB(ctx)
}

// UpdateStandardSensors 数据变更时调用
// 策略：Cache-Aside (直接删除缓存)
func (c *StandardSensorsCache) UpdateStandardSensors(ctx context.Context) error {
	return c.client.Del(ctx, standardSensorsCacheKey).Err()
}

// RefreshCache forces a refresh of the cache from the database.
func (c *StandardSensorsCache) RefreshCache(ctx context.Context) ([]*dto.StandardSensorResponse, error) {
	return c.fetchAndCache(ctx)
}

// --- 私有辅助方法 ---

// getFromCache 读取缓存 + 脏数据清理
func (c *StandardSensorsCache) getFromCache(ctx context.Context) ([]*dto.StandardSensorResponse, error) {
	val, err := c.client.Get(ctx, standardSensorsCacheKey).Bytes()
	if err != nil {
		return nil, err
	}

	var responses []*dto.StandardSensorResponse
	if err := json.Unmarshal(val, &responses); err != nil {
		// 脏数据自愈
		c.client.Del(ctx, standardSensorsCacheKey)
		return nil, fmt.Errorf("invalid cache data deleted: %w", err)
	}

	return responses, nil
}

// fetchAndCache 查库 + 序列化 + 写缓存
func (c *StandardSensorsCache) fetchAndCache(ctx context.Context) ([]*dto.StandardSensorResponse, error) {
	// 1. 查库
	responses, err := c.loadAllFromDB(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 序列化
	jsonData, err := json.Marshal(responses)
	if err != nil {
		contextx.GetLoggerFromContext(ctx).Error(fmt.Sprintf("failed to serialize standard sensors: %v", err))
		return responses, nil
	}

	// 3. 写缓存 (非阻塞)
	err = c.client.Set(ctx, standardSensorsCacheKey, jsonData, standardSensorsTTL).Err()
	if err != nil {
		contextx.GetLoggerFromContext(ctx).Warn(fmt.Sprintf("failed to write cache for standard sensors: %v", err))
	}

	return responses, nil
}

// loadAllFromDB 分页加载所有数据
func (c *StandardSensorsCache) loadAllFromDB(ctx context.Context) ([]*dto.StandardSensorResponse, error) {

	var allResponses []*dto.StandardSensorResponse
	pageNumber := 1
	const pageSize = 500 // 调大 PageSize，减少数据库交互次数

	for {
		query := entity.StandardSensorQuery{
			Pagination: entity.Pagination{
				PageNumber: pageNumber,
				PageSize:   pageSize,
			},
		}

		entities, total, err := c.repo.List(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("load db failed: %w", err)
		}

		for _, e := range entities {
			allResponses = append(allResponses, dto.ConvertToStandardSensorResponse(e))
		}

		if int64(len(allResponses)) >= total {
			break
		}
		if len(entities) == 0 {
			break
		}
		pageNumber++
	}

	// 确保不返回 nil slice，避免 JSON 序列化为 "null"
	if allResponses == nil {
		allResponses = []*dto.StandardSensorResponse{}
	}

	return allResponses, nil
}
