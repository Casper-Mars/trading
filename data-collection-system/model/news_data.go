package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// StringSlice 字符串切片类型，用于JSON字段
type StringSlice []string

// Value 实现driver.Valuer接口
func (s StringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan 实现sql.Scanner接口
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("cannot scan %T into StringSlice", value)
	}
}

// NewsData 新闻数据模型
type NewsData struct {
	ID                uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Title             string         `gorm:"type:varchar(500);not null" json:"title" binding:"required"`
	Content           string         `gorm:"type:text" json:"content"`
	Source            string         `gorm:"type:varchar(100)" json:"source"`
	PublishTime       time.Time      `gorm:"type:datetime;not null;index:idx_publish_time" json:"publish_time" binding:"required"`
	Category          string         `gorm:"type:varchar(50);index:idx_category" json:"category"`
	Sentiment         *int8          `gorm:"type:tinyint;index:idx_sentiment;comment:情感倾向: 1-正面, 0-中性, -1-负面" json:"sentiment"`
	SentimentScore    *float64       `gorm:"type:decimal(5,4);comment:情感得分" json:"sentiment_score"`
	ImportanceLevel   int8           `gorm:"type:tinyint;default:3;index:idx_importance;comment:重要程度: 1-5级" json:"importance_level"`
	RelatedStocks     StringSlice    `gorm:"type:json;comment:相关股票列表" json:"related_stocks"`
	RelatedIndustries StringSlice    `gorm:"type:json;comment:相关行业列表" json:"related_industries"`
	ProcessedAt       *time.Time     `gorm:"type:timestamp;comment:处理时间" json:"processed_at"`
	CreatedAt         time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定表名
func (NewsData) TableName() string {
	return "news_data"
}

// Sentiment 情感倾向常量
const (
	SentimentNegative = -1 // 负面
	SentimentNeutral  = 0  // 中性
	SentimentPositive = 1  // 正面
)

// ImportanceLevel 重要程度常量
const (
	ImportanceLevelVeryLow  = 1 // 很低
	ImportanceLevelLow      = 2 // 低
	ImportanceLevelMedium   = 3 // 中等
	ImportanceLevelHigh     = 4 // 高
	ImportanceLevelVeryHigh = 5 // 很高
)

// IsPositive 检查是否为正面新闻
func (n *NewsData) IsPositive() bool {
	return n.Sentiment != nil && *n.Sentiment == SentimentPositive
}

// IsNegative 检查是否为负面新闻
func (n *NewsData) IsNegative() bool {
	return n.Sentiment != nil && *n.Sentiment == SentimentNegative
}

// IsNeutral 检查是否为中性新闻
func (n *NewsData) IsNeutral() bool {
	return n.Sentiment != nil && *n.Sentiment == SentimentNeutral
}

// IsHighImportance 检查是否为高重要性新闻
func (n *NewsData) IsHighImportance() bool {
	return n.ImportanceLevel >= ImportanceLevelHigh
}

// IsProcessed 检查是否已处理
func (n *NewsData) IsProcessed() bool {
	return n.ProcessedAt != nil
}

// MarkAsProcessed 标记为已处理
func (n *NewsData) MarkAsProcessed() {
	now := time.Now()
	n.ProcessedAt = &now
}

// HasRelatedStocks 检查是否有相关股票
func (n *NewsData) HasRelatedStocks() bool {
	return len(n.RelatedStocks) > 0
}

// HasRelatedIndustries 检查是否有相关行业
func (n *NewsData) HasRelatedIndustries() bool {
	return len(n.RelatedIndustries) > 0
}

// AddRelatedStock 添加相关股票
func (n *NewsData) AddRelatedStock(symbol string) {
	for _, stock := range n.RelatedStocks {
		if stock == symbol {
			return // 已存在，不重复添加
		}
	}
	n.RelatedStocks = append(n.RelatedStocks, symbol)
}

// AddRelatedIndustry 添加相关行业
func (n *NewsData) AddRelatedIndustry(industry string) {
	for _, ind := range n.RelatedIndustries {
		if ind == industry {
			return // 已存在，不重复添加
		}
	}
	n.RelatedIndustries = append(n.RelatedIndustries, industry)
}