package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Log      LogConfig      `mapstructure:"log"`
	Tushare  TushareConfig  `mapstructure:"tushare"`
	Crawler  CrawlerConfig  `mapstructure:"crawler"`
	BaiduAI  BaiduAIConfig  `mapstructure:"baidu_ai"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	Charset  string `mapstructure:"charset"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	// 连接池配置
	PoolSize     int `mapstructure:"pool_size"`
	MinIdleConns int `mapstructure:"min_idle_conns"`
	MaxRetries   int `mapstructure:"max_retries"`
	// 超时配置（秒）
	DialTimeout        int `mapstructure:"dial_timeout"`
	ReadTimeout        int `mapstructure:"read_timeout"`
	WriteTimeout       int `mapstructure:"write_timeout"`
	PoolTimeout        int `mapstructure:"pool_timeout"`
	IdleTimeout        int `mapstructure:"idle_timeout"`
	IdleCheckFrequency int `mapstructure:"idle_check_frequency"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// TushareConfig Tushare配置
type TushareConfig struct {
	Token   string `mapstructure:"token"`
	BaseURL string `mapstructure:"base_url"`
}

// CrawlerConfig 爬虫配置
type CrawlerConfig struct {
	UserAgent   string   `mapstructure:"user_agent"`
	Delay       int      `mapstructure:"delay"`
	Parallelism int      `mapstructure:"parallelism"`
	TargetURLs  []string `mapstructure:"target_urls"`
}

// BaiduAIConfig 百度AI配置
type BaiduAIConfig struct {
	AppID     string `mapstructure:"app_id"`
	APIKey    string `mapstructure:"api_key"`
	SecretKey string `mapstructure:"secret_key"`
	BaseURL   string `mapstructure:"base_url"`
	// 缓存配置
	CacheEnabled bool `mapstructure:"cache_enabled"`
	CacheTTL     int  `mapstructure:"cache_ttl"`
	// 限流配置
	QPS     int `mapstructure:"qps"`
	Timeout int `mapstructure:"timeout"`
}

// Load 加载配置
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// 设置环境变量前缀
	viper.SetEnvPrefix("DCS")
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件未找到，使用默认值
			fmt.Println("Config file not found, using default values")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// 从环境变量覆盖敏感配置
	if dbPassword := os.Getenv("DCS_DATABASE_PASSWORD"); dbPassword != "" {
		config.Database.Password = dbPassword
	}
	if redisPassword := os.Getenv("DCS_REDIS_PASSWORD"); redisPassword != "" {
		config.Redis.Password = redisPassword
	}
	if tushareToken := os.Getenv("DCS_TUSHARE_TOKEN"); tushareToken != "" {
		config.Tushare.Token = tushareToken
	}
	if baiduAppID := os.Getenv("DCS_BAIDU_AI_APP_ID"); baiduAppID != "" {
		config.BaiduAI.AppID = baiduAppID
	}
	if baiduAPIKey := os.Getenv("DCS_BAIDU_AI_API_KEY"); baiduAPIKey != "" {
		config.BaiduAI.APIKey = baiduAPIKey
	}
	if baiduSecretKey := os.Getenv("DCS_BAIDU_AI_SECRET_KEY"); baiduSecretKey != "" {
		config.BaiduAI.SecretKey = baiduSecretKey
	}

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults() {
	// 服务器默认配置
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")

	// 数据库默认配置
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.user", "root")
	viper.SetDefault("database.dbname", "trading_data")
	viper.SetDefault("database.charset", "utf8mb4")

	// Redis默认配置
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)
	// Redis连接池默认配置
	viper.SetDefault("redis.pool_size", 20)
	viper.SetDefault("redis.min_idle_conns", 5)
	viper.SetDefault("redis.max_retries", 3)
	// Redis超时默认配置（秒）
	viper.SetDefault("redis.dial_timeout", 10)
	viper.SetDefault("redis.read_timeout", 5)
	viper.SetDefault("redis.write_timeout", 5)
	viper.SetDefault("redis.pool_timeout", 10)
	viper.SetDefault("redis.idle_timeout", 300)
	viper.SetDefault("redis.idle_check_frequency", 60)

	// 日志默认配置
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", "stdout")

	// Tushare默认配置
	viper.SetDefault("tushare.base_url", "http://api.tushare.pro")

	// 爬虫默认配置
	viper.SetDefault("crawler.user_agent", "Mozilla/5.0 (compatible; DataCollector/1.0)")
	viper.SetDefault("crawler.delay", 1000)
	viper.SetDefault("crawler.parallelism", 2)

	// 百度AI默认配置
	viper.SetDefault("baidu_ai.base_url", "https://aip.baidubce.com")
	viper.SetDefault("baidu_ai.cache_enabled", true)
	viper.SetDefault("baidu_ai.cache_ttl", 3600)
	viper.SetDefault("baidu_ai.qps", 5)
	viper.SetDefault("baidu_ai.timeout", 30)
}
