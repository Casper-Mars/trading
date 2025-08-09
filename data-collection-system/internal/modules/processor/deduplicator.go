package processor
import (
	"context"
	"crypto/md5"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"data-collection-system/pkg/logger"
)

// RedisDeduplicator Redis数据去重器
type RedisDeduplicator struct {
	redisClient *redis.Client
	ttl         time.Duration // 缓存过期时间
	keyPrefix   string        // 键前缀
}

// NewRedisDeduplicator 创建Redis去重器
func NewRedisDeduplicator(redisClient *redis.Client) *RedisDeduplicator {
	return &RedisDeduplicator{
		redisClient: redisClient,
		ttl:         24 * time.Hour, // 默认24小时过期
		keyPrefix:   "data_processor:dedup:",
	}
}

// SetTTL 设置缓存过期时间
func (r *RedisDeduplicator) SetTTL(ttl time.Duration) {
	r.ttl = ttl
}

// SetKeyPrefix 设置键前缀
func (r *RedisDeduplicator) SetKeyPrefix(prefix string) {
	r.keyPrefix = prefix
}

// CheckDuplicate 检查数据是否重复
func (r *RedisDeduplicator) CheckDuplicate(ctx context.Context, dataType string, key string) (bool, error) {
	// 生成Redis键
	redisKey := r.generateRedisKey(dataType, key)
	
	// 检查键是否存在
	exists, err := r.redisClient.Exists(ctx, redisKey).Result()
	if err != nil {
		logger.Error("Failed to check duplicate in Redis", "error", err, "key", redisKey)
		return false, fmt.Errorf("redis check failed: %w", err)
	}
	
	return exists > 0, nil
}

// MarkProcessed 标记数据已处理
func (r *RedisDeduplicator) MarkProcessed(ctx context.Context, dataType string, key string) error {
	// 生成Redis键
	redisKey := r.generateRedisKey(dataType, key)
	
	// 设置键值，记录处理时间
	processedTime := time.Now().Unix()
	err := r.redisClient.Set(ctx, redisKey, processedTime, r.ttl).Err()
	if err != nil {
		logger.Error("Failed to mark as processed in Redis", "error", err, "key", redisKey)
		return fmt.Errorf("redis set failed: %w", err)
	}
	
	logger.Debug("Marked data as processed", "key", redisKey, "ttl", r.ttl)
	return nil
}

// GetProcessedTime 获取数据处理时间
func (r *RedisDeduplicator) GetProcessedTime(ctx context.Context, dataType string, key string) (time.Time, error) {
	// 生成Redis键
	redisKey := r.generateRedisKey(dataType, key)
	
	// 获取处理时间
	timestamp, err := r.redisClient.Get(ctx, redisKey).Int64()
	if err != nil {
		if err == redis.Nil {
			return time.Time{}, fmt.Errorf("data not found")
		}
		logger.Error("Failed to get processed time from Redis", "error", err, "key", redisKey)
		return time.Time{}, fmt.Errorf("redis get failed: %w", err)
	}
	
	return time.Unix(timestamp, 0), nil
}

// RemoveProcessed 移除已处理标记
func (r *RedisDeduplicator) RemoveProcessed(ctx context.Context, dataType string, key string) error {
	// 生成Redis键
	redisKey := r.generateRedisKey(dataType, key)
	
	// 删除键
	err := r.redisClient.Del(ctx, redisKey).Err()
	if err != nil {
		logger.Error("Failed to remove processed mark from Redis", "error", err, "key", redisKey)
		return fmt.Errorf("redis del failed: %w", err)
	}
	
	logger.Debug("Removed processed mark", "key", redisKey)
	return nil
}

// BatchCheckDuplicate 批量检查重复
func (r *RedisDeduplicator) BatchCheckDuplicate(ctx context.Context, dataType string, keys []string) (map[string]bool, error) {
	if len(keys) == 0 {
		return make(map[string]bool), nil
	}
	
	// 生成Redis键列表
	redisKeys := make([]string, len(keys))
	for i, key := range keys {
		redisKeys[i] = r.generateRedisKey(dataType, key)
	}
	
	// 批量检查存在性
	pipe := r.redisClient.Pipeline()
	cmds := make([]*redis.IntCmd, len(redisKeys))
	for i, redisKey := range redisKeys {
		cmds[i] = pipe.Exists(ctx, redisKey)
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		logger.Error("Failed to batch check duplicates in Redis", "error", err)
		return nil, fmt.Errorf("redis pipeline failed: %w", err)
	}
	
	// 构建结果映射
	result := make(map[string]bool, len(keys))
	for i, key := range keys {
		exists, err := cmds[i].Result()
		if err != nil {
			logger.Warn("Failed to get exists result for key", "error", err, "key", key)
			result[key] = false
		} else {
			result[key] = exists > 0
		}
	}
	
	return result, nil
}

// BatchMarkProcessed 批量标记已处理
func (r *RedisDeduplicator) BatchMarkProcessed(ctx context.Context, dataType string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	
	// 批量设置
	pipe := r.redisClient.Pipeline()
	processedTime := time.Now().Unix()
	
	for _, key := range keys {
		redisKey := r.generateRedisKey(dataType, key)
		pipe.Set(ctx, redisKey, processedTime, r.ttl)
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		logger.Error("Failed to batch mark as processed in Redis", "error", err)
		return fmt.Errorf("redis pipeline failed: %w", err)
	}
	
	logger.Debug("Batch marked data as processed", "count", len(keys), "dataType", dataType)
	return nil
}

// CleanExpired 清理过期的去重记录
func (r *RedisDeduplicator) CleanExpired(ctx context.Context, dataType string) error {
	// 获取所有相关键
	pattern := r.generateRedisKey(dataType, "*")
	keys, err := r.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		logger.Error("Failed to get keys for cleanup", "error", err, "pattern", pattern)
		return fmt.Errorf("redis keys failed: %w", err)
	}
	
	if len(keys) == 0 {
		return nil
	}
	
	// 检查每个键的TTL
	pipe := r.redisClient.Pipeline()
	ttlCmds := make([]*redis.DurationCmd, len(keys))
	for i, key := range keys {
		ttlCmds[i] = pipe.TTL(ctx, key)
	}
	
	_, err = pipe.Exec(ctx)
	if err != nil {
		logger.Error("Failed to check TTL for cleanup", "error", err)
		return fmt.Errorf("redis pipeline failed: %w", err)
	}
	
	// 收集需要删除的键
	expiredKeys := make([]string, 0)
	for i, key := range keys {
		ttl, err := ttlCmds[i].Result()
		if err != nil {
			logger.Warn("Failed to get TTL for key", "error", err, "key", key)
			continue
		}
		
		// TTL为-1表示没有过期时间，TTL为-2表示键不存在
		if ttl == -2 {
			expiredKeys = append(expiredKeys, key)
		}
	}
	
	// 删除过期键
	if len(expiredKeys) > 0 {
		err = r.redisClient.Del(ctx, expiredKeys...).Err()
		if err != nil {
			logger.Error("Failed to delete expired keys", "error", err, "count", len(expiredKeys))
			return fmt.Errorf("redis del failed: %w", err)
		}
		
		logger.Info("Cleaned expired deduplication records", "count", len(expiredKeys), "dataType", dataType)
	}
	
	return nil
}

// GetStats 获取去重统计信息
func (r *RedisDeduplicator) GetStats(ctx context.Context, dataType string) (*DeduplicationStats, error) {
	// 获取所有相关键
	pattern := r.generateRedisKey(dataType, "*")
	keys, err := r.redisClient.Keys(ctx, pattern).Result()
	if err != nil {
		logger.Error("Failed to get keys for stats", "error", err, "pattern", pattern)
		return nil, fmt.Errorf("redis keys failed: %w", err)
	}
	
	stats := &DeduplicationStats{
		DataType:    dataType,
		TotalKeys:   len(keys),
		ActiveKeys:  0,
		ExpiredKeys: 0,
		Timestamp:   time.Now(),
	}
	
	if len(keys) == 0 {
		return stats, nil
	}
	
	// 检查每个键的状态
	pipe := r.redisClient.Pipeline()
	ttlCmds := make([]*redis.DurationCmd, len(keys))
	for i, key := range keys {
		ttlCmds[i] = pipe.TTL(ctx, key)
	}
	
	_, err = pipe.Exec(ctx)
	if err != nil {
		logger.Error("Failed to check TTL for stats", "error", err)
		return stats, nil // 返回基础统计信息
	}
	
	// 统计活跃和过期键
	for i := range keys {
		ttl, err := ttlCmds[i].Result()
		if err != nil {
			continue
		}
		
		if ttl == -2 {
			stats.ExpiredKeys++
		} else {
			stats.ActiveKeys++
		}
	}
	
	return stats, nil
}

// generateRedisKey 生成Redis键
func (r *RedisDeduplicator) generateRedisKey(dataType string, key string) string {
	// 使用MD5哈希来处理过长的键
	if len(key) > 200 {
		hash := md5.Sum([]byte(key))
		key = fmt.Sprintf("%x", hash)
	}
	
	return fmt.Sprintf("%s%s:%s", r.keyPrefix, dataType, key)
}

// DeduplicationStats 去重统计信息
type DeduplicationStats struct {
	DataType    string    `json:"data_type"`
	TotalKeys   int       `json:"total_keys"`
	ActiveKeys  int       `json:"active_keys"`
	ExpiredKeys int       `json:"expired_keys"`
	Timestamp   time.Time `json:"timestamp"`
}

// MemoryDeduplicator 内存数据去重器（用于测试或小规模场景）
type MemoryDeduplicator struct {
	data   map[string]time.Time
	ttl    time.Duration
	mutex  sync.RWMutex
}

// NewMemoryDeduplicator 创建内存去重器
func NewMemoryDeduplicator() *MemoryDeduplicator {
	return &MemoryDeduplicator{
		data:   make(map[string]time.Time),
		ttl:    24 * time.Hour,
	}
}

// CheckDuplicate 检查数据是否重复
func (m *MemoryDeduplicator) CheckDuplicate(ctx context.Context, dataType string, key string) (bool, error) {
	fullKey := fmt.Sprintf("%s:%s", dataType, key)
	
	// 清理过期数据
	m.cleanExpired()
	
	_, exists := m.data[fullKey]
	return exists, nil
}

// MarkProcessed 标记数据已处理
func (m *MemoryDeduplicator) MarkProcessed(ctx context.Context, dataType string, key string) error {
	fullKey := fmt.Sprintf("%s:%s", dataType, key)
	m.data[fullKey] = time.Now()
	
	logger.Debug("Marked data as processed in memory", "key", fullKey)
	return nil
}

// cleanExpired 清理过期数据
func (m *MemoryDeduplicator) cleanExpired() {
	now := time.Now()
	for key, timestamp := range m.data {
		if now.Sub(timestamp) > m.ttl {
			delete(m.data, key)
		}
	}
}