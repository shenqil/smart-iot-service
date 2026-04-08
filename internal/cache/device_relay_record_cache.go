package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	deviceRelayRecordKey = "device:%d:relay_record:%d"
	relayRecordTTL       = 1 * time.Hour
)

// DeviceRelayRecordCache handles caching for device relay operation records.
// It only interacts with Redis and does not contain any database fallback logic.
type DeviceRelayRecordCache struct {
	client *redis.Client
}

// NewDeviceRelayRecordCache creates a new DeviceRelayRecordCache.
func NewDeviceRelayRecordCache(client *redis.Client) *DeviceRelayRecordCache {
	return &DeviceRelayRecordCache{client: client}
}

// GetDeviceRelayRecordKey returns the Redis key for a device's relay record.
func GetDeviceRelayRecordKey(deviceID uint32, relayNum uint) string {
	return fmt.Sprintf(deviceRelayRecordKey, deviceID, relayNum)
}

// SetRelayRecord sets the cache for a device relay operation record.
func (c *DeviceRelayRecordCache) SetRelayRecord(ctx context.Context, deviceID uint32, relayNum uint, state byte, recordID uint) error {
	if recordID == 0 {
		return errors.New("recordID cannot be zero")
	}
	key := GetDeviceRelayRecordKey(deviceID, relayNum)
	recordData := map[string]interface{}{
		"record_id": recordID,
		"state":     state,
		"time":      time.Now().Unix(),
	}
	jsonData, err := json.Marshal(recordData)
	if err != nil {
		return fmt.Errorf("failed to serialize relay record: %w", err)
	}
	return c.client.Set(ctx, key, jsonData, relayRecordTTL).Err()
}

// GetRelayRecord retrieves a device relay operation record from the cache.
// Returns the state, record ID, timestamp, and an error.
// Returns redis.Nil if the key does not exist.
func (c *DeviceRelayRecordCache) GetRelayRecord(ctx context.Context, deviceID uint32, relayNum uint) (byte, uint, time.Time, error) {
	key := GetDeviceRelayRecordKey(deviceID, relayNum)
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return 0, 0, time.Time{}, err // Pass redis.Nil up
	}

	var recordData map[string]interface{}
	if err := json.Unmarshal([]byte(val), &recordData); err != nil {
		return 0, 0, time.Time{}, fmt.Errorf("failed to deserialize relay record: %w", err)
	}

	var state byte
	if stateVal, ok := recordData["state"].(float64); ok {
		state = byte(stateVal)
	}

	var recordID uint
	if recordIDVal, ok := recordData["record_id"].(float64); ok {
		recordID = uint(recordIDVal)
	}

	var recordTime time.Time
	if timeVal, ok := recordData["time"].(float64); ok {
		recordTime = time.Unix(int64(timeVal), 0)
	}

	return state, recordID, recordTime, nil
}

// DeleteRelayRecord deletes a device relay operation record from the cache.
func (c *DeviceRelayRecordCache) DeleteRelayRecord(ctx context.Context, deviceID uint32, relayNum uint) error {
	key := GetDeviceRelayRecordKey(deviceID, relayNum)
	return c.client.Del(ctx, key).Err()
}
