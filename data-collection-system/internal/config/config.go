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
}