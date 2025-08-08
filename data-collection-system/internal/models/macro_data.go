package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// MacroData 宏观经济数据模型
type MacroData struct {
	ID            uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	IndicatorCode string         `gorm:"type:varchar(50);not null;uniqueIndex:uk_indicator_date,priority:1" json:"indicator_code" binding:"required"`
	IndicatorName string         `gorm:"type:varchar(200);not null" json:"indicator_name" binding:"required"`
	PeriodType    string         `gorm:"type:varchar(10);not null;index:idx_period_type;comment:周期类型: daily,weekly,monthly,quarterly,yearly" json:"period_type" binding:"required"`
	DataDate      time.Time      `gorm:"type:date;not null;uniqueIndex:uk_indicator_date,priority:2;index:idx_data_date" json:"data_date" binding:"required"`
	Value         float64        `gorm:"type:decimal(20,6);not null" json:"value" binding:"required"`
	Unit          string         `gorm:"type:varchar(20)" json:"unit"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (MacroData) TableName() string {
	return "macro_data"
}

// PeriodType 周期类型常量
const (
	PeriodTypeDaily     = "daily"
	PeriodTypeWeekly    = "weekly"
	PeriodTypeMonthly   = "monthly"
	PeriodTypeQuarterly = "quarterly"
	PeriodTypeYearly    = "yearly"
)

// ValidPeriodTypes 有效的周期类型
var ValidPeriodTypes = []string{
	PeriodTypeDaily,
	PeriodTypeWeekly,
	PeriodTypeMonthly,
	PeriodTypeQuarterly,
	PeriodTypeYearly,
}

// IsValidPeriodType 检查周期类型是否有效
func IsValidPeriodType(periodType string) bool {
	for _, pt := range ValidPeriodTypes {
		if pt == periodType {
			return true
		}
	}
	return false
}

// IsDaily 检查是否为日度数据
func (m *MacroData) IsDaily() bool {
	return m.PeriodType == PeriodTypeDaily
}

// IsWeekly 检查是否为周度数据
func (m *MacroData) IsWeekly() bool {
	return m.PeriodType == PeriodTypeWeekly
}

// IsMonthly 检查是否为月度数据
func (m *MacroData) IsMonthly() bool {
	return m.PeriodType == PeriodTypeMonthly
}

// IsQuarterly 检查是否为季度数据
func (m *MacroData) IsQuarterly() bool {
	return m.PeriodType == PeriodTypeQuarterly
}

// IsYearly 检查是否为年度数据
func (m *MacroData) IsYearly() bool {
	return m.PeriodType == PeriodTypeYearly
}

// GetFormattedValue 获取格式化的数值（带单位）
func (m *MacroData) GetFormattedValue() string {
	if m.Unit != "" {
		return fmt.Sprintf("%.6f %s", m.Value, m.Unit)
	}
	return fmt.Sprintf("%.6f", m.Value)
}

// IsPositive 检查数值是否为正
func (m *MacroData) IsPositive() bool {
	return m.Value > 0
}

// IsNegative 检查数值是否为负
func (m *MacroData) IsNegative() bool {
	return m.Value < 0
}

// IsZero 检查数值是否为零
func (m *MacroData) IsZero() bool {
	return m.Value == 0
}