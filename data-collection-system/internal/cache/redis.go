package cache

import (
	"context"
	"fmt"
	"time"

	"data-collection-system/internal/config"
	"data-collection-system/pkg/logger"

	"github.com/go-redis/redis/v8"
)

var rdb *redis.Client

// Init 初始化Redis连接
func Init(cfg config.RedisConfig) (*redis.Client, error) {
	// 创建Redis客户端
	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: 10,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("Redis connected successfully")
	return rdb, nil
}

// GetRedis 获取Redis客户端实例
func GetRedis() *redis.Client {
	return rdb
}

// Close 关闭Redis连接
func Close() error {
	if rdb != nil {
		return rdb.Close()
	}
	return nil
}

// Set 设置缓存
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if rdb == nil {
		return fmt.Errorf("redis not initialized")
	}
	return rdb.Set(ctx, key, value, expiration).Err()
}

// Get 获取缓存
func Get(ctx context.Context, key string) (string, error) {
	if rdb == nil {
		return "", fmt.Errorf("redis not initialized")
	}
	return rdb.Get(ctx, key).Result()
}

// Del 删除缓存
func Del(ctx context.Context, keys ...string) error {
	if rdb == nil {
		return fmt.Errorf("redis not initialized")
	}
	return rdb.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func Exists(ctx context.Context, keys ...string) (int64, error) {
	if rdb == nil {
		return 0, fmt.Errorf("redis not initialized")
	}
	return rdb.Exists(ctx, keys...).Result()
}

// Expire 设置过期时间
func Expire(ctx context.Context, key string, expiration time.Duration) error {
	if rdb == nil {
		return fmt.Errorf("redis not initialized")
	}
	return rdb.Expire(ctx, key, expiration).Err()
}

// HSet 设置哈希字段
func HSet(ctx context.Context, key string, values ...interface{}) error {
	if rdb == nil {
		return fmt.Errorf("redis not initialized")
	}
	return rdb.HSet(ctx, key, values...).Err()
}

// HGet 获取哈希字段
func HGet(ctx context.Context, key, field string) (string, error) {
	if rdb == nil {
		return "", fmt.Errorf("redis not initialized")
	}
	return rdb.HGet(ctx, key, field).Result()
}

// HGetAll 获取所有哈希字段
func HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if rdb == nil {
		return nil, fmt.Errorf("redis not initialized")
	}
	return rdb.HGetAll(ctx, key).Result()
}