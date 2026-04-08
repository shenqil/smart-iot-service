package cache

import (
	"context"
	"fmt"
	"rbac-admin/pkg/config"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
)

// NewRedisClient creates and returns a new Redis client instance.
func NewRedisClient() (*redis.Client, error) {

	confRedis := config.Conf.Redis

	client := redis.NewClient(&redis.Options{
		Addr:         confRedis.Addr,
		Password:     confRedis.Password,
		DB:           confRedis.DB,
		PoolSize:     50,
		MinIdleConns: 10,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test the connection.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}

// NewRedsync creates a new redsync instance.
func NewRedsync(client *redis.Client) *redsync.Redsync {
	pool := goredis.NewPool(client)
	return redsync.New(pool)
}
