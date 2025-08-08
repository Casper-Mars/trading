package models

import (
	"time"

	"gorm.io/gorm"
)

// MarketData 行情数据模型
type MarketData struct {
	ID         uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Symbol     string         `gorm:"type:varchar(20);not null;uniqueIndex:uk_symbol_time_period,priority:1" json:"symbol" binding:"required"`
	TradeDate  time.Time      `gorm:"type:date;not null;index:idx_symbol_date,priority:2" json:"trade_date" binding:"required"`
	Period     string         `gorm:"type:varchar(10);not null;uniqueIndex:uk_symbol_time_period,priority:3;comment:周期: 1m,5m,15m,30m,1h,1d" json:"period" binding:"required"`
	TradeTime  time.Time      `gorm:"type:datetime;not null;uniqueIndex:uk_symbol_time_period,priority:2;index:idx_trade_time" json:"trade_time" binding:"required"`
	OpenPrice  float64        `gorm:"type:decimal(10,3);not null" json:"open_price" binding:"required"`
	HighPrice  float64        `gorm:"type:decimal(10,3);not null" json:"high_price" binding:"required"`
	LowPrice   float64        `gorm:"type:decimal(10,3);not null" json:"low_price" binding:"required"`
	ClosePrice float64        `gorm:"type:decimal(10,3);not null" json:"close_price" binding:"required"`
	Volume     int64          `gorm:"type:bigint;not null" json:"volume" binding:"required"`
	Amount     float64        `gorm:"type:decimal(15,2);not null" json:"amount" binding:"required"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (MarketData) TableName() string {
	return "market_data"
}

// Period 周期类型常量
const (
	Period1Min  = "1m"
	Period5Min  = "5m"
	Period15Min = "15m"
	Period30Min = "30m"
	Period1Hour = "1h"
	Period1Day  = "1d"
)

// ValidPeriods 有效的周期类型
var ValidPeriods = []string{
	Period1Min,
	Period5Min,
	Period15Min,
	Period30Min,
	Period1Hour,
	Period1Day,
}

// IsValidPeriod 检查周期是否有效
func IsValidPeriod(period string) bool {
	for _, p := range ValidPeriods {
		if p == period {
			return true
		}
	}
	return false
}

// GetPriceChange 计算价格变化
func (m *MarketData) GetPriceChange() float64 {
	return m.ClosePrice - m.OpenPrice
}

// GetPriceChangePercent 计算价格变化百分比
func (m *MarketData) GetPriceChangePercent() float64 {
	if m.OpenPrice == 0 {
		return 0
	}
	return (m.ClosePrice - m.OpenPrice) / m.OpenPrice * 100
}

// GetTurnover 计算换手率（需要流通股本数据）
func (m *MarketData) GetTurnover(circulatingShares int64) float64 {
	if circulatingShares == 0 {
		return 0
	}
	return float64(m.Volume) / float64(circulatingShares) * 100
}