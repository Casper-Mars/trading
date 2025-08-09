package cron

import (
	"fmt"
	"time"
)

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	// 基础配置
	Enabled         bool          `yaml:"enabled" json:"enabled"`                   // 是否启用调度器
	MaxConcurrency  int           `yaml:"max_concurrency" json:"max_concurrency"`   // 最大并发任务数
	JobTimeout      time.Duration `yaml:"job_timeout" json:"job_timeout"`           // 任务超时时间
	RetryAttempts   int           `yaml:"retry_attempts" json:"retry_attempts"`     // 重试次数
	RetryInterval   time.Duration `yaml:"retry_interval" json:"retry_interval"`     // 重试间隔
	
	// 监控配置
	MetricsEnabled  bool          `yaml:"metrics_enabled" json:"metrics_enabled"`   // 是否启用指标收集
	HealthCheck     bool          `yaml:"health_check" json:"health_check"`         // 是否启用健康检查
	LogLevel        string        `yaml:"log_level" json:"log_level"`               // 日志级别
	
	// 存储配置
	PersistJobs     bool          `yaml:"persist_jobs" json:"persist_jobs"`         // 是否持久化任务
	JobHistoryLimit int           `yaml:"job_history_limit" json:"job_history_limit"` // 任务历史记录限制
	
	// 时区配置
	Timezone        string        `yaml:"timezone" json:"timezone"`                 // 时区设置
	
	// 任务配置
	DefaultJobs     []JobConfig   `yaml:"default_jobs" json:"default_jobs"`         // 默认任务配置
}

// JobConfig 任务配置
type JobConfig struct {
	ID          string                 `yaml:"id" json:"id"`                     // 任务ID
	Name        string                 `yaml:"name" json:"name"`                 // 任务名称
	Type        string                 `yaml:"type" json:"type"`                 // 任务类型
	Schedule    string                 `yaml:"schedule" json:"schedule"`         // Cron表达式
	Enabled     bool                   `yaml:"enabled" json:"enabled"`           // 是否启用
	Timeout     time.Duration          `yaml:"timeout" json:"timeout"`           // 任务超时时间
	Retries     int                    `yaml:"retries" json:"retries"`           // 重试次数
	Config      map[string]interface{} `yaml:"config" json:"config"`             // 任务配置参数
	Description string                 `yaml:"description" json:"description"`   // 任务描述
	Category    string                 `yaml:"category" json:"category"`         // 任务分类
}

// DefaultSchedulerConfig 默认调度器配置
func DefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		Enabled:         true,
		MaxConcurrency:  10,
		JobTimeout:      30 * time.Minute,
		RetryAttempts:   3,
		RetryInterval:   5 * time.Minute,
		MetricsEnabled:  true,
		HealthCheck:     true,
		LogLevel:        "info",
		PersistJobs:     true,
		JobHistoryLimit: 1000,
		Timezone:        "Asia/Shanghai",
		DefaultJobs:     getDefaultJobs(),
	}
}

// getDefaultJobs 获取默认任务配置
func getDefaultJobs() []JobConfig {
	return []JobConfig{
		// 股票数据采集任务
		{
			ID:          "stock_daily_collection",
			Name:        "每日股票数据采集",
			Type:        "stock_data",
			Schedule:    "0 18 * * 1-5", // 工作日18:00执行
			Enabled:     true,
			Timeout:     20 * time.Minute,
			Retries:     3,
			Category:    "stock",
			Description: "每日股票数据采集，包括基础信息、市场数据和财务数据",
			Config: map[string]interface{}{
				"data_types": []string{"basic", "market", "financial"},
				"batch_size": 100,
			},
		},
		// 新闻数据采集任务
		{
			ID:          "news_hourly_collection",
			Name:        "每小时新闻数据采集",
			Type:        "news_data",
			Schedule:    "0 * * * *", // 每小时执行
			Enabled:     true,
			Timeout:     10 * time.Minute,
			Retries:     2,
			Category:    "news",
			Description: "每小时新闻数据采集，获取最新财经新闻",
			Config: map[string]interface{}{
				"sources":    []string{"sina", "163", "tencent"},
				"batch_size": 50,
				"keywords":   []string{"股票", "财经", "投资"},
			},
		},
		// 市场数据实时采集
		{
			ID:          "market_realtime_collection",
			Name:        "市场数据实时采集",
			Type:        "market_data",
			Schedule:    "*/5 * * * 1-5", // 工作日每5分钟执行
			Enabled:     true,
			Timeout:     5 * time.Minute,
			Retries:     2,
			Category:    "stock",
			Description: "市场数据实时采集，包括价格、成交量等",
			Config: map[string]interface{}{
				"symbols":    []string{"000001.SZ", "000002.SZ", "600000.SH"},
				"batch_size": 200,
			},
		},
		// 数据清理任务
		{
			ID:          "data_cleanup_daily",
			Name:        "每日数据清理",
			Type:        "data_cleanup",
			Schedule:    "0 2 * * *", // 每天凌晨2点执行
			Enabled:     true,
			Timeout:     30 * time.Minute,
			Retries:     1,
			Category:    "system",
			Description: "每日数据清理，删除过期日志和临时数据",
			Config: map[string]interface{}{
				"retention_days": 30,
				"cleanup_types": []string{"logs", "temp_files", "cache"},
			},
		},
		// 系统监控任务
		{
			ID:          "system_monitor",
			Name:        "系统监控",
			Type:        "system_monitor",
			Schedule:    "*/10 * * * *", // 每10分钟执行
			Enabled:     true,
			Timeout:     2 * time.Minute,
			Retries:     1,
			Category:    "system",
			Description: "系统监控，监控CPU、内存、磁盘等系统指标",
			Config: map[string]interface{}{
				"metrics": []string{"cpu", "memory", "disk", "network"},
				"thresholds": map[string]float64{
					"cpu":    80.0,
					"memory": 85.0,
					"disk":   90.0,
				},
			},
		},
	}
}

// Validate 验证配置
func (c *SchedulerConfig) Validate() error {
	if c.MaxConcurrency <= 0 {
		return fmt.Errorf("max_concurrency must be greater than 0")
	}
	
	if c.JobTimeout <= 0 {
		return fmt.Errorf("job_timeout must be greater than 0")
	}
	
	if c.RetryAttempts < 0 {
		return fmt.Errorf("retry_attempts must be non-negative")
	}
	
	if c.RetryInterval <= 0 {
		return fmt.Errorf("retry_interval must be greater than 0")
	}
	
	if c.JobHistoryLimit <= 0 {
		return fmt.Errorf("job_history_limit must be greater than 0")
	}
	
	// 验证时区
	if c.Timezone != "" {
		if _, err := time.LoadLocation(c.Timezone); err != nil {
			return fmt.Errorf("invalid timezone: %s", c.Timezone)
		}
	}
	
	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log_level: %s", c.LogLevel)
	}
	
	// 验证默认任务配置
	for i, job := range c.DefaultJobs {
		if err := job.Validate(); err != nil {
			return fmt.Errorf("default job %d validation failed: %w", i, err)
		}
	}
	
	return nil
}

// Validate 验证任务配置
func (j *JobConfig) Validate() error {
	if j.ID == "" {
		return fmt.Errorf("job id cannot be empty")
	}
	
	if j.Name == "" {
		return fmt.Errorf("job name cannot be empty")
	}
	
	if j.Type == "" {
		return fmt.Errorf("job type cannot be empty")
	}
	
	if j.Schedule == "" {
		return fmt.Errorf("job schedule cannot be empty")
	}
	
	if j.Timeout <= 0 {
		return fmt.Errorf("job timeout must be greater than 0")
	}
	
	if j.Retries < 0 {
		return fmt.Errorf("job retries must be non-negative")
	}
	
	return nil
}

// GetLocation 获取时区位置
func (c *SchedulerConfig) GetLocation() (*time.Location, error) {
	if c.Timezone == "" {
		return time.Local, nil
	}
	return time.LoadLocation(c.Timezone)
}

// IsJobEnabled 检查任务是否启用
func (c *SchedulerConfig) IsJobEnabled(jobID string) bool {
	for _, job := range c.DefaultJobs {
		if job.ID == jobID {
			return job.Enabled
		}
	}
	return false
}

// GetJobConfig 获取任务配置
func (c *SchedulerConfig) GetJobConfig(jobID string) *JobConfig {
	for _, job := range c.DefaultJobs {
		if job.ID == jobID {
			return &job
		}
	}
	return nil
}

// AddJobConfig 添加任务配置
func (c *SchedulerConfig) AddJobConfig(job JobConfig) error {
	if err := job.Validate(); err != nil {
		return err
	}
	
	// 检查是否已存在相同ID的任务
	for _, existingJob := range c.DefaultJobs {
		if existingJob.ID == job.ID {
			return fmt.Errorf("job with id %s already exists", job.ID)
		}
	}
	
	c.DefaultJobs = append(c.DefaultJobs, job)
	return nil
}

// RemoveJobConfig 移除任务配置
func (c *SchedulerConfig) RemoveJobConfig(jobID string) bool {
	for i, job := range c.DefaultJobs {
		if job.ID == jobID {
			c.DefaultJobs = append(c.DefaultJobs[:i], c.DefaultJobs[i+1:]...)
			return true
		}
	}
	return false
}

// UpdateJobConfig 更新任务配置
func (c *SchedulerConfig) UpdateJobConfig(jobID string, updatedJob JobConfig) error {
	if err := updatedJob.Validate(); err != nil {
		return err
	}
	
	for i, job := range c.DefaultJobs {
		if job.ID == jobID {
			c.DefaultJobs[i] = updatedJob
			return nil
		}
	}
	
	return fmt.Errorf("job with id %s not found", jobID)
}

// GetJobsByCategory 根据分类获取任务
func (c *SchedulerConfig) GetJobsByCategory(category string) []JobConfig {
	var jobs []JobConfig
	for _, job := range c.DefaultJobs {
		if job.Category == category {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// GetEnabledJobs 获取启用的任务
func (c *SchedulerConfig) GetEnabledJobs() []JobConfig {
	var jobs []JobConfig
	for _, job := range c.DefaultJobs {
		if job.Enabled {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

// Clone 克隆配置
func (c *SchedulerConfig) Clone() *SchedulerConfig {
	cloned := *c
	cloned.DefaultJobs = make([]JobConfig, len(c.DefaultJobs))
	copy(cloned.DefaultJobs, c.DefaultJobs)
	return &cloned
}

// ToMap 转换为Map格式
func (c *SchedulerConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"enabled":           c.Enabled,
		"max_concurrency":   c.MaxConcurrency,
		"job_timeout":       c.JobTimeout.String(),
		"retry_attempts":    c.RetryAttempts,
		"retry_interval":    c.RetryInterval.String(),
		"metrics_enabled":   c.MetricsEnabled,
		"health_check":      c.HealthCheck,
		"log_level":         c.LogLevel,
		"persist_jobs":      c.PersistJobs,
		"job_history_limit": c.JobHistoryLimit,
		"timezone":          c.Timezone,
		"default_jobs":      len(c.DefaultJobs),
	}
}