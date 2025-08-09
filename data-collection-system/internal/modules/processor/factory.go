package processor

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"data-collection-system/pkg/logger"
)

// ProcessorConfig 处理器配置
type ProcessorConfig struct {
	// Redis配置
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	
	// 缓存配置
	CacheExpiration time.Duration
	CacheKeyPrefix  string
	
	// 处理配置
	BatchSize       int
	MaxRetries      int
	RetryDelay      time.Duration
	ProcessTimeout  time.Duration
	
	// 质量检查配置
	QualityThreshold float64
	EnableQualityCheck bool
}

// DefaultProcessorConfig 默认处理器配置
func DefaultProcessorConfig() *ProcessorConfig {
	return &ProcessorConfig{
		RedisAddr:          "localhost:6379",
		RedisPassword:      "",
		RedisDB:            0,
		CacheExpiration:    24 * time.Hour,
		CacheKeyPrefix:     "data_processor:",
		BatchSize:          100,
		MaxRetries:         3,
		RetryDelay:         time.Second,
		ProcessTimeout:     30 * time.Second,
		QualityThreshold:   0.8,
		EnableQualityCheck: true,
	}
}

// ProcessorFactory 处理器工厂
type ProcessorFactory struct {
	config      *ProcessorConfig
	redisClient *redis.Client
}

// NewProcessorFactory 创建处理器工厂
func NewProcessorFactory(config *ProcessorConfig, redisClient *redis.Client) (*ProcessorFactory, error) {
	if config == nil {
		config = DefaultProcessorConfig()
	}
	
	// 如果没有传入Redis客户端，则创建新的Redis客户端
	if redisClient == nil {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     config.RedisAddr,
			Password: config.RedisPassword,
			DB:       config.RedisDB,
		})
	}
	
	// 测试Redis连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warn("Redis connection failed, using memory deduplicator", "error", err)
		redisClient = nil
	}
	
	return &ProcessorFactory{
		config:      config,
		redisClient: redisClient,
	}, nil
}

// CreateProcessorManager 创建处理器管理器
func (f *ProcessorFactory) CreateProcessorManager() (*ProcessorManager, error) {
	manager := &ProcessorManager{
		processor:      f.CreateProcessor(),
		validator:      f.CreateValidator(),
		cleaner:        f.CreateCleaner(),
		deduplicator:   f.CreateDeduplicator(),
		qualityChecker: f.CreateQualityChecker(),
	}
	
	return manager, nil
}

// CreateValidator 创建数据验证器
func (f *ProcessorFactory) CreateValidator() DataValidator {
	return NewDefaultValidator()
}

// CreateCleaner 创建数据清洗器
func (f *ProcessorFactory) CreateCleaner() DataCleaner {
	return NewDefaultCleaner()
}

// CreateDeduplicator 创建去重器
func (f *ProcessorFactory) CreateDeduplicator() DataDeduplicator {
	if f.redisClient != nil {
		return NewRedisDeduplicator(f.redisClient)
	}
	
	// 如果Redis不可用，使用内存去重器
	return NewMemoryDeduplicator()
}

// CreateQualityChecker 创建质量检查器
func (f *ProcessorFactory) CreateQualityChecker() QualityChecker {
	return NewDefaultQualityChecker()
}

// CreateProcessor 创建数据处理器
func (f *ProcessorFactory) CreateProcessor() DataProcessor {
	return NewDefaultProcessor()
}

// CreateNewsProcessor 创建新闻处理器
func (f *ProcessorFactory) CreateNewsProcessor() *NewsProcessor {
	return NewNewsProcessor()
}

// Close 关闭工厂资源
func (f *ProcessorFactory) Close() error {
	if f.redisClient != nil {
		return f.redisClient.Close()
	}
	return nil
}

// GetConfig 获取配置
func (f *ProcessorFactory) GetConfig() *ProcessorConfig {
	return f.config
}

// UpdateConfig 更新配置
func (f *ProcessorFactory) UpdateConfig(config *ProcessorConfig) {
	f.config = config
}

// HealthCheck 健康检查
func (f *ProcessorFactory) HealthCheck(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"factory_status": "healthy",
		"timestamp":      time.Now(),
	}
	
	// 检查Redis连接
	if f.redisClient != nil {
		if err := f.redisClient.Ping(ctx).Err(); err != nil {
			health["redis_status"] = "unhealthy"
			health["redis_error"] = err.Error()
		} else {
			health["redis_status"] = "healthy"
		}
	} else {
		health["redis_status"] = "disabled"
	}
	
	// 添加配置信息
	health["config"] = map[string]interface{}{
		"batch_size":         f.config.BatchSize,
		"max_retries":        f.config.MaxRetries,
		"quality_threshold":  f.config.QualityThreshold,
		"enable_quality_check": f.config.EnableQualityCheck,
	}
	
	return health
}