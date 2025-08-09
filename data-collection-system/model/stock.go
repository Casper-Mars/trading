package model

import (
	"time"

	"gorm.io/gorm"
)

// Stock 股票基础信息模型
type Stock struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Symbol    string         `gorm:"type:varchar(20);uniqueIndex:uk_symbol;not null" json:"symbol" binding:"required"`
	Name      string         `gorm:"type:varchar(100);not null" json:"name" binding:"required"`
	Exchange  string         `gorm:"type:varchar(10);index:idx_exchange;not null" json:"exchange" binding:"required"`
	Industry  string         `gorm:"type:varchar(50);index:idx_industry" json:"industry"`
	Sector    string         `gorm:"type:varchar(50)" json:"sector"`
	ListDate  *time.Time     `gorm:"type:date" json:"list_date"`
	Status    int8           `gorm:"type:tinyint;default:1;index:idx_status;comment:状态: 1-正常, 0-停牌" json:"status"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (Stock) TableName() string {
	return "stocks"
}

// StockStatus 股票状态常量
const (
	StockStatusSuspended = 0 // 停牌
	StockStatusActive    = 1 // 正常交易
	StockStatusDelisted  = 2 // 退市
)

// IsActive 检查股票是否正常交易
func (s *Stock) IsActive() bool {
	return s.Status == StockStatusActive
}

// IsSuspended 检查股票是否停牌
func (s *Stock) IsSuspended() bool {
	return s.Status == StockStatusSuspended
}