package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"data-collection-system/pkg/config"
	"data-collection-system/pkg/logger"

	"github.com/go-redis/redis/v8"
)

var rdb *redis.Client

// CacheManager Redis缓存管理器
type CacheManager struct {
	client *redis.Client
}

// 缓存键命名规范常量
const (
	// 键前缀
	KeyPrefixStock    = "stock"     // 股票数据
	KeyPrefixNews     = "news"      // 新闻数据
	KeyPrefixSentiment = "sentiment" // 情感分析数据
	KeyPrefixIndicator = "indicator" // 技术指标数据
	KeyPrefixUser     = "user"      // 用户数据
	KeyPrefixSession  = "session"   // 会话数据

	// 分隔符
	KeySeparator = ":"
)

// TTL策略常量
const (
	TTLShort   = 5 * time.Minute   // 短期缓存：5分钟
	TTLMedium  = 30 * time.Minute  // 中期缓存：30分钟
	TTLLong    = 2 * time.Hour     // 长期缓存：2小时
	TTLDaily   = 24 * time.Hour    // 日缓存：24小时
	TTLWeekly  = 7 * 24 * time.Hour // 周缓存：7天
	TTLSession = 30 * time.Minute  // 会话缓存：30分钟
)

// Init 初始化Redis连接
func Init(cfg config.RedisConfig) (*redis.Client, error) {
	// 创建Redis客户端，配置连接池和超时参数
	rdb = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		// 连接池配置
		PoolSize:     cfg.PoolSize,     // 连接池大小
		MinIdleConns: cfg.MinIdleConns, // 最小空闲连接数
		MaxRetries:   cfg.MaxRetries,   // 最大重试次数
		// 超时配置
		DialTimeout:  time.Duration(cfg.DialTimeout) * time.Second,  // 连接超时
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,  // 读取超时
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second, // 写入超时
		PoolTimeout:  time.Duration(cfg.PoolTimeout) * time.Second,  // 连接池超时
		// 空闲连接检查
		IdleTimeout:       time.Duration(cfg.IdleTimeout) * time.Second,      // 空闲连接超时
		IdleCheckFrequency: time.Duration(cfg.IdleCheckFrequency) * time.Second, // 空闲连接检查频率
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("Redis connected successfully with enhanced configuration")
	return rdb, nil
}

// NewCacheManager 创建缓存管理器实例
func NewCacheManager() *CacheManager {
	return &CacheManager{
		client: rdb,
	}
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

// BuildKey 构建缓存键，遵循命名规范
// 格式：prefix:module:identifier[:subkey]
func BuildKey(prefix, module, identifier string, subkeys ...string) string {
	parts := []string{prefix, module, identifier}
	if len(subkeys) > 0 {
		parts = append(parts, subkeys...)
	}
	return strings.Join(parts, KeySeparator)
}

// BuildStockKey 构建股票数据缓存键
func BuildStockKey(stockCode, dataType string) string {
	return BuildKey(KeyPrefixStock, dataType, stockCode)
}

// BuildNewsKey 构建新闻数据缓存键
func BuildNewsKey(newsID, dataType string) string {
	return BuildKey(KeyPrefixNews, dataType, newsID)
}

// BuildSentimentKey 构建情感分析缓存键
func BuildSentimentKey(sourceID, analysisType string) string {
	return BuildKey(KeyPrefixSentiment, analysisType, sourceID)
}

// BuildIndicatorKey 构建技术指标缓存键
func BuildIndicatorKey(stockCode, indicatorType string) string {
	return BuildKey(KeyPrefixIndicator, indicatorType, stockCode)
}

// BuildUserKey 构建用户数据缓存键
func BuildUserKey(userID, dataType string) string {
	return BuildKey(KeyPrefixUser, dataType, userID)
}

// BuildSessionKey 构建会话缓存键
func BuildSessionKey(sessionID string) string {
	return BuildKey(KeyPrefixSession, "data", sessionID)
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

// ========== CacheManager 增强方法 ==========

// SetJSON 设置JSON格式缓存
func (cm *CacheManager) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if cm.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	
	return cm.client.Set(ctx, key, data, ttl).Err()
}

// GetJSON 获取JSON格式缓存
func (cm *CacheManager) GetJSON(ctx context.Context, key string, dest interface{}) error {
	if cm.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	
	data, err := cm.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	
	return json.Unmarshal([]byte(data), dest)
}

// SetWithTTL 根据数据类型设置合适的TTL
func (cm *CacheManager) SetWithTTL(ctx context.Context, key string, value interface{}, dataType string) error {
	var ttl time.Duration
	
	// 根据数据类型选择合适的TTL
	switch dataType {
	case "realtime", "quote":
		ttl = TTLShort // 实时数据：5分钟
	case "news", "sentiment":
		ttl = TTLMedium // 新闻和情感数据：30分钟
	case "indicator", "analysis":
		ttl = TTLLong // 技术指标：2小时
	case "daily", "historical":
		ttl = TTLDaily // 历史数据：24小时
	case "weekly", "report":
		ttl = TTLWeekly // 周报数据：7天
	case "session":
		ttl = TTLSession // 会话数据：30分钟
	default:
		ttl = TTLMedium // 默认：30分钟
	}
	
	return cm.SetJSON(ctx, key, value, ttl)
}

// MSet 批量设置缓存
func (cm *CacheManager) MSet(ctx context.Context, pairs map[string]interface{}) error {
	if cm.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	
	args := make([]interface{}, 0, len(pairs)*2)
	for key, value := range pairs {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON for key %s: %w", key, err)
		}
		args = append(args, key, data)
	}
	
	return cm.client.MSet(ctx, args...).Err()
}

// MGet 批量获取缓存
func (cm *CacheManager) MGet(ctx context.Context, keys []string) (map[string]string, error) {
	if cm.client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}
	
	results, err := cm.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	
	data := make(map[string]string)
	for i, result := range results {
		if result != nil {
			data[keys[i]] = result.(string)
		}
	}
	
	return data, nil
}

// DeleteByPattern 根据模式删除缓存键
func (cm *CacheManager) DeleteByPattern(ctx context.Context, pattern string) error {
	if cm.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	
	keys, err := cm.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	
	if len(keys) > 0 {
		return cm.client.Del(ctx, keys...).Err()
	}
	
	return nil
}

// RefreshTTL 刷新缓存过期时间
func (cm *CacheManager) RefreshTTL(ctx context.Context, key string, ttl time.Duration) error {
	if cm.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	
	return cm.client.Expire(ctx, key, ttl).Err()
}

// GetTTL 获取缓存剩余过期时间
func (cm *CacheManager) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	if cm.client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}
	
	return cm.client.TTL(ctx, key).Result()
}

// IsExists 检查缓存键是否存在
func (cm *CacheManager) IsExists(ctx context.Context, key string) (bool, error) {
	if cm.client == nil {
		return false, fmt.Errorf("redis client not initialized")
	}
	
	count, err := cm.client.Exists(ctx, key).Result()
	return count > 0, err
}

// GetSize 获取缓存值大小（字节）
func (cm *CacheManager) GetSize(ctx context.Context, key string) (int64, error) {
	if cm.client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}
	
	return cm.client.StrLen(ctx, key).Result()
}