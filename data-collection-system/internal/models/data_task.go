package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TaskConfig 任务配置类型，用于JSON字段
type TaskConfig map[string]interface{}

// Value 实现driver.Valuer接口
func (tc TaskConfig) Value() (driver.Value, error) {
	if len(tc) == 0 {
		return nil, nil
	}
	return json.Marshal(tc)
}

// Scan 实现sql.Scanner接口
func (tc *TaskConfig) Scan(value interface{}) error {
	if value == nil {
		*tc = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, tc)
	case string:
		return json.Unmarshal([]byte(v), tc)
	default:
		return fmt.Errorf("cannot scan %T into TaskConfig", value)
	}
}

// DataTask 数据任务模型
type DataTask struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskName    string         `gorm:"type:varchar(100);not null;uniqueIndex" json:"task_name" binding:"required"`
	TaskType    string         `gorm:"type:varchar(50);not null;index:idx_task_type" json:"task_type" binding:"required"`
	Description string         `gorm:"type:text" json:"description"`
	CronExpr    string         `gorm:"type:varchar(100);not null" json:"cron_expr" binding:"required"`
	Status      int8           `gorm:"type:tinyint;not null;default:1;index:idx_status" json:"status"`
	Config      TaskConfig     `gorm:"type:json" json:"config"`
	LastRunAt   *time.Time     `gorm:"type:timestamp" json:"last_run_at"`
	NextRunAt   *time.Time     `gorm:"type:timestamp;index:idx_next_run" json:"next_run_at"`
	RunCount    uint64         `gorm:"default:0" json:"run_count"`
	SuccessCount uint64        `gorm:"default:0" json:"success_count"`
	FailureCount uint64        `gorm:"default:0" json:"failure_count"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (DataTask) TableName() string {
	return "data_tasks"
}

// TaskStatus 任务状态常量
const (
	TaskStatusDisabled = 0 // 禁用
	TaskStatusEnabled  = 1 // 启用
	TaskStatusRunning  = 2 // 运行中
	TaskStatusPaused   = 3 // 暂停
)

// TaskType 任务类型常量
const (
	TaskTypeStockData     = "stock_data"     // 股票数据采集
	TaskTypeMarketData    = "market_data"    // 行情数据采集
	TaskTypeFinancialData = "financial_data" // 财务数据采集
	TaskTypeNewsData      = "news_data"      // 新闻数据采集
	TaskTypeMacroData     = "macro_data"     // 宏观数据采集
	TaskTypeDataCleanup   = "data_cleanup"   // 数据清理
	TaskTypeDataBackup    = "data_backup"    // 数据备份
)

// ValidTaskTypes 有效的任务类型
var ValidTaskTypes = []string{
	TaskTypeStockData,
	TaskTypeMarketData,
	TaskTypeFinancialData,
	TaskTypeNewsData,
	TaskTypeMacroData,
	TaskTypeDataCleanup,
	TaskTypeDataBackup,
}

// IsValidTaskType 检查任务类型是否有效
func IsValidTaskType(taskType string) bool {
	for _, tt := range ValidTaskTypes {
		if tt == taskType {
			return true
		}
	}
	return false
}

// IsEnabled 检查任务是否启用
func (t *DataTask) IsEnabled() bool {
	return t.Status == TaskStatusEnabled
}

// IsDisabled 检查任务是否禁用
func (t *DataTask) IsDisabled() bool {
	return t.Status == TaskStatusDisabled
}

// IsRunning 检查任务是否正在运行
func (t *DataTask) IsRunning() bool {
	return t.Status == TaskStatusRunning
}

// IsPaused 检查任务是否暂停
func (t *DataTask) IsPaused() bool {
	return t.Status == TaskStatusPaused
}

// Enable 启用任务
func (t *DataTask) Enable() {
	t.Status = TaskStatusEnabled
}

// Disable 禁用任务
func (t *DataTask) Disable() {
	t.Status = TaskStatusDisabled
}

// SetRunning 设置任务为运行状态
func (t *DataTask) SetRunning() {
	t.Status = TaskStatusRunning
}

// Pause 暂停任务
func (t *DataTask) Pause() {
	t.Status = TaskStatusPaused
}

// IncrementRunCount 增加运行次数
func (t *DataTask) IncrementRunCount() {
	t.RunCount++
	now := time.Now()
	t.LastRunAt = &now
}

// IncrementSuccessCount 增加成功次数
func (t *DataTask) IncrementSuccessCount() {
	t.SuccessCount++
}

// IncrementFailureCount 增加失败次数
func (t *DataTask) IncrementFailureCount() {
	t.FailureCount++
}

// GetSuccessRate 获取成功率
func (t *DataTask) GetSuccessRate() float64 {
	if t.RunCount == 0 {
		return 0
	}
	return float64(t.SuccessCount) / float64(t.RunCount)
}

// GetFailureRate 获取失败率
func (t *DataTask) GetFailureRate() float64 {
	if t.RunCount == 0 {
		return 0
	}
	return float64(t.FailureCount) / float64(t.RunCount)
}

// ShouldRun 检查任务是否应该运行
func (t *DataTask) ShouldRun() bool {
	if !t.IsEnabled() {
		return false
	}
	if t.NextRunAt == nil {
		return false
	}
	return time.Now().After(*t.NextRunAt)
}

// SetNextRunTime 设置下次运行时间
func (t *DataTask) SetNextRunTime(nextTime time.Time) {
	t.NextRunAt = &nextTime
}

// GetConfigValue 获取配置值
func (t *DataTask) GetConfigValue(key string) (interface{}, bool) {
	if t.Config == nil {
		return nil, false
	}
	value, exists := t.Config[key]
	return value, exists
}

// SetConfigValue 设置配置值
func (t *DataTask) SetConfigValue(key string, value interface{}) {
	if t.Config == nil {
		t.Config = make(TaskConfig)
	}
	t.Config[key] = value
}