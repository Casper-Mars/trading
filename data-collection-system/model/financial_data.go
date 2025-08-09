package model

import (
	"time"

	"gorm.io/gorm"
)

// FinancialData 财务数据模型
type FinancialData struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Symbol       string         `gorm:"type:varchar(20);not null;uniqueIndex:uk_symbol_report,priority:1" json:"symbol" binding:"required"`
	ReportDate   time.Time      `gorm:"type:date;not null;uniqueIndex:uk_symbol_report,priority:2;index:idx_report_date" json:"report_date" binding:"required"`
	ReportType   string         `gorm:"type:varchar(5);not null;uniqueIndex:uk_symbol_report,priority:3;comment:报告类型: Q1,Q2,Q3,A" json:"report_type" binding:"required"`
	Revenue      *float64       `gorm:"type:decimal(15,2);comment:营业收入" json:"revenue"`
	NetProfit    *float64       `gorm:"type:decimal(15,2);comment:净利润" json:"net_profit"`
	TotalAssets  *float64       `gorm:"type:decimal(15,2);comment:总资产" json:"total_assets"`
	TotalEquity  *float64       `gorm:"type:decimal(15,2);comment:股东权益" json:"total_equity"`
	ROE          *float64       `gorm:"type:decimal(8,4);comment:ROE" json:"roe"`
	ROA          *float64       `gorm:"type:decimal(8,4);comment:ROA" json:"roa"`
	GrossMargin  *float64       `gorm:"type:decimal(8,4);comment:毛利率" json:"gross_margin"`
	NetMargin    *float64       `gorm:"type:decimal(8,4);comment:净利率" json:"net_margin"`
	CurrentRatio *float64       `gorm:"type:decimal(8,4);comment:流动比率" json:"current_ratio"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (FinancialData) TableName() string {
	return "financial_data"
}

// ReportType 报告类型常量
const (
	ReportTypeQ1     = "Q1" // 一季报
	ReportTypeQ2     = "Q2" // 半年报
	ReportTypeQ3     = "Q3" // 三季报
	ReportTypeAnnual = "A"  // 年报
)

// ValidReportTypes 有效的报告类型
var ValidReportTypes = []string{
	ReportTypeQ1,
	ReportTypeQ2,
	ReportTypeQ3,
	ReportTypeAnnual,
}

// IsValidReportType 检查报告类型是否有效
func IsValidReportType(reportType string) bool {
	for _, rt := range ValidReportTypes {
		if rt == reportType {
			return true
		}
	}
	return false
}

// IsAnnualReport 检查是否为年报
func (f *FinancialData) IsAnnualReport() bool {
	return f.ReportType == ReportTypeAnnual
}

// IsQuarterlyReport 检查是否为季报
func (f *FinancialData) IsQuarterlyReport() bool {
	return f.ReportType != ReportTypeAnnual
}

// GetDebtToAssetRatio 计算资产负债率
func (f *FinancialData) GetDebtToAssetRatio() *float64 {
	if f.TotalAssets == nil || f.TotalEquity == nil || *f.TotalAssets == 0 {
		return nil
	}
	debt := *f.TotalAssets - *f.TotalEquity
	ratio := debt / *f.TotalAssets
	return &ratio
}

// GetEquityToAssetRatio 计算股东权益比率
func (f *FinancialData) GetEquityToAssetRatio() *float64 {
	if f.TotalAssets == nil || f.TotalEquity == nil || *f.TotalAssets == 0 {
		return nil
	}
	ratio := *f.TotalEquity / *f.TotalAssets
	return &ratio
}

// HasProfitability 检查是否盈利
func (f *FinancialData) HasProfitability() bool {
	return f.NetProfit != nil && *f.NetProfit > 0
}