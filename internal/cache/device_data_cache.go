package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"rbac-admin/internal/model/dto"
	"rbac-admin/pkg/contextx"

	"github.com/go-redis/redis/v8"
)

const (
	deviceDataKey = "device:%d:data"
	// 乐观锁最大重试次数 (CAS Loop)
	maxRetries = 3
)

// DeviceDataCache 处理设备实时数据的缓存。
// 采用无数据库回退策略（Cache-Only），适用于高频变化的实时数据。
type DeviceDataCache struct {
	client *redis.Client
}

// NewDeviceDataCache 创建一个新的 DeviceDataCache 实例。
func NewDeviceDataCache(client *redis.Client) *DeviceDataCache {
	return &DeviceDataCache{client: client}
}

// GetDeviceDataKey 返回设备实时数据的 Redis 键。
func GetDeviceDataKey(deviceID uint32) string {
	return fmt.Sprintf(deviceDataKey, deviceID)
}

// SetData 覆盖设置设备的实时数据。
func (c *DeviceDataCache) SetData(ctx context.Context, deviceID uint32, data *dto.DeviceRealTimeData) error {
	key := GetDeviceDataKey(deviceID)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化设备数据失败: %w", err)
	}
	// 实时数据通常不需要过期时间，或者设置一个较长的过期时间（如24h）防止僵尸数据
	return c.client.Set(ctx, key, jsonData, 0).Err()
}

// GetData 获取设备的实时数据。
func (c *DeviceDataCache) GetData(ctx context.Context, deviceID uint32) (*dto.DeviceRealTimeData, error) {
	key := GetDeviceDataKey(deviceID)
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var data dto.DeviceRealTimeData
	if err := json.Unmarshal(val, &data); err != nil {
		// 【优化】: 脏数据自愈
		// 如果缓存数据已损坏（非 JSON 格式），主动删除，防止后续请求持续报错
		contextx.GetLoggerFromContext(ctx).Warn(fmt.Sprintf("found invalid cache data for device %d, deleting: %v", deviceID, err))
		c.client.Del(ctx, key)
		return nil, fmt.Errorf("invalid cache data deleted: %w", err)
	}
	return &data, nil
}

// UpdateRelayStatus 线程安全地更新继电器状态。
// 使用 Redis WATCH + Retry Loop (乐观锁) 机制防止并发覆盖问题。
func (c *DeviceDataCache) UpdateRelayStatus(ctx context.Context, deviceID uint32, relayNum uint, state byte, timestamp time.Time) error {
	// 输入验证
	if relayNum > 31 {
		return fmt.Errorf("invalid relay num: %d (max 31)", relayNum)
	}

	key := GetDeviceDataKey(deviceID)
	relayKey := fmt.Sprintf("%d", relayNum+1)

	// 【完美优化】: 增加重试循环
	// 乐观锁必须配合重试机制，否则高并发下失败率极高
	for i := 0; i < maxRetries; i++ {
		// 每次重试前检查 Context 是否已取消，避免无意义的重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := c.client.Watch(ctx, func(tx *redis.Tx) error {
			// 1. 在事务内读取当前数据
			val, err := tx.Get(ctx, key).Bytes()
			if err != nil && err != redis.Nil {
				return err
			}

			var cacheData dto.DeviceRealTimeData

			// 2. 解析数据 (如果是新 Key 或数据为空，初始化默认结构)
			if len(val) > 0 {
				if err := json.Unmarshal(val, &cacheData); err != nil {
					// 遇到脏数据，尝试覆盖修复
					contextx.GetLoggerFromContext(ctx).Warn(fmt.Sprintf("invalid json during watch update for device %d: %v", deviceID, err))
					cacheData = c.initEmptyData(timestamp)
				}
			} else {
				cacheData = c.initEmptyData(timestamp)
			}

			// 3. 确保所有 Map 已初始化 (防止 nil map panic，包含 SensorData)
			if cacheData.RelayData == nil {
				cacheData.RelayData = make(map[string]dto.RelayData)
			}
			if cacheData.SensorData == nil {
				cacheData.SensorData = make(map[string]interface{})
			}

			// 4. 更新内存中的状态
			cacheData.RelayData[relayKey] = dto.RelayData{
				Value:     state,
				Timestamp: timestamp.Unix(),
			}
			cacheData.Timestamp = timestamp.Unix()

			// 5. 序列化
			newData, err := json.Marshal(cacheData)
			if err != nil {
				return err
			}

			// 6. 执行事务 (TxPipelined)
			// 如果在 Watch 期间 key 被其他人修改了，Exec 会返回 redis.TxFailedErr
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.Set(ctx, key, newData, 0)
				return nil
			})
			return err
		}, key)

		// 处理 Watch 结果
		if err == nil {
			// 成功
			return nil
		}
		if err == redis.TxFailedErr {
			// 发生并发冲突，进入下一次循环重试
			// 可选：在高并发极端的场景下，这里可以加一个微小的随机 sleep (jitter)
			continue
		}

		// 其他系统错误，直接返回
		return fmt.Errorf("failed to update relay status: %w", err)
	}

	return fmt.Errorf("update relay status failed after %d retries due to concurrent modifications", maxRetries)
}

// initEmptyData 辅助函数：初始化空的数据结构
func (c *DeviceDataCache) initEmptyData(ts time.Time) dto.DeviceRealTimeData {
	return dto.DeviceRealTimeData{
		SensorData: make(map[string]interface{}),
		RelayData:  make(map[string]dto.RelayData),
		Timestamp:  ts.Unix(),
	}
}
