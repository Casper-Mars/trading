package models

import (
	"gorm.io/gorm"
)

// AllModels 返回所有需要迁移的模型
func AllModels() []interface{} {
	return []interface{}{
		&Stock{},
		&MarketData{},
		&FinancialData{},
		&NewsData{},
		&MacroData{},
		&DataTask{},
	}
}

// AutoMigrate 自动迁移所有模型
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(AllModels()...)
}

// CreateIndexes 创建额外的索引
func CreateIndexes(db *gorm.DB) error {
	// 为market_data表创建复合索引
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_market_data_symbol_date_period 
		ON market_data(symbol, trade_date, period)
	`).Error; err != nil {
		return err
	}

	// 为financial_data表创建复合索引
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_financial_data_symbol_date 
		ON financial_data(symbol, report_date)
	`).Error; err != nil {
		return err
	}

	// 为news_data表创建复合索引
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_news_data_publish_category 
		ON news_data(publish_time, category)
	`).Error; err != nil {
		return err
	}

	// 为macro_data表创建复合索引
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_macro_data_indicator_date 
		ON macro_data(indicator_code, data_date)
	`).Error; err != nil {
		return err
	}

	return nil
}