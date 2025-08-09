package dao

import (
	"context"
	"fmt"
	"time"

	"data-collection-system/model"
	"gorm.io/gorm"
)

// sentimentRepository 市场情绪数据仓库实现
type sentimentRepository struct {
	db *gorm.DB
}

// NewSentimentRepository 创建市场情绪数据仓库实例
func NewSentimentRepository(db *gorm.DB) SentimentRepository {
	return &sentimentRepository{db: db}
}

// CreateMarketSentiment 创建市场情绪数据
func (r *sentimentRepository) CreateMarketSentiment(ctx context.Context, data *model.MarketSentimentData) error {
	return r.db.WithContext(ctx).Create(data).Error
}

// BatchCreateMarketSentiment 批量创建市场情绪数据
func (r *sentimentRepository) BatchCreateMarketSentiment(ctx context.Context, data []interface{}) error {
	if len(data) == 0 {
		return nil
	}
	// 转换为具体类型
	dataList := make([]*model.MarketSentimentData, len(data))
	for i, item := range data {
		if sentimentData, ok := item.(*model.MarketSentimentData); ok {
			dataList[i] = sentimentData
		} else {
			return fmt.Errorf("invalid data type at index %d", i)
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(dataList, 100).Error
}

// GetMarketSentimentByDate 根据日期获取市场情绪数据
func (r *sentimentRepository) GetMarketSentimentByDate(ctx context.Context, date time.Time) ([]*model.MarketSentimentData, error) {
	var dataList []*model.MarketSentimentData
	err := r.db.WithContext(ctx).
		Where("trade_date = ?", date.Format("2006-01-02")).
		Order("created_at DESC").
		Find(&dataList).Error
	return dataList, err
}

// GetMarketSentimentByDateRange 根据日期范围获取市场情绪数据
func (r *sentimentRepository) GetMarketSentimentByDateRange(ctx context.Context, startDate, endDate time.Time, dataType string) ([]*model.MarketSentimentData, error) {
	var dataList []*model.MarketSentimentData
	query := r.db.WithContext(ctx).Where("trade_date >= ? AND trade_date <= ?", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if dataType != "" {
		query = query.Where("data_type = ?", dataType)
	}
	err := query.Order("trade_date DESC").Find(&dataList).Error
	return dataList, err
}

// 资金流向数据相关方法

// CreateMoneyFlow 创建资金流向数据
func (r *sentimentRepository) CreateMoneyFlow(ctx context.Context, data *model.MoneyFlowData) error {
	return r.db.WithContext(ctx).Create(data).Error
}

// BatchCreateMoneyFlow 批量创建资金流向数据
func (r *sentimentRepository) BatchCreateMoneyFlow(ctx context.Context, data []interface{}) error {
	if len(data) == 0 {
		return nil
	}
	// 转换为具体类型
	moneyFlowData := make([]*model.MoneyFlowData, len(data))
	for i, item := range data {
		if mfd, ok := item.(*model.MoneyFlowData); ok {
			moneyFlowData[i] = mfd
		} else {
			return fmt.Errorf("invalid data type at index %d", i)
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(moneyFlowData, 100).Error
}

// GetMoneyFlowBySymbolAndDate 根据股票代码和交易日期获取资金流向数据
func (r *sentimentRepository) GetMoneyFlowBySymbolAndDate(ctx context.Context, symbol string, tradeDate time.Time) (*model.MoneyFlowData, error) {
	var data model.MoneyFlowData
	err := r.db.WithContext(ctx).
		Where("symbol = ? AND trade_date = ?", symbol, tradeDate.Format("2006-01-02")).
		First(&data).Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// GetMoneyFlowByDateRange 根据日期范围获取资金流向数据
func (r *sentimentRepository) GetMoneyFlowByDateRange(ctx context.Context, symbol string, startDate, endDate time.Time) ([]*model.MoneyFlowData, error) {
	var dataList []*model.MoneyFlowData
	query := r.db.WithContext(ctx).
		Where("trade_date >= ? AND trade_date <= ?", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	
	if symbol != "" {
		query = query.Where("symbol = ?", symbol)
	}
	
	err := query.Order("trade_date DESC").Find(&dataList).Error
	return dataList, err
}

// 北向资金数据相关方法

// CreateNorthboundFund 创建北向资金数据
func (r *sentimentRepository) CreateNorthboundFund(ctx context.Context, data *model.NorthboundFundData) error {
	return r.db.WithContext(ctx).Create(data).Error
}

// BatchCreateNorthboundFund 批量创建北向资金数据
func (r *sentimentRepository) BatchCreateNorthboundFund(ctx context.Context, dataList []*model.NorthboundFundData) error {
	if len(dataList) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(dataList, 100).Error
}

// GetNorthboundFundByDate 根据交易日期获取北向资金数据
func (r *sentimentRepository) GetNorthboundFundByDate(ctx context.Context, tradeDate time.Time) ([]*model.NorthboundFundData, error) {
	var dataList []*model.NorthboundFundData
	err := r.db.WithContext(ctx).
		Where("trade_date = ?", tradeDate.Format("2006-01-02")).
		Find(&dataList).Error
	return dataList, err
}

// GetNorthboundFundByDateRange 根据日期范围获取北向资金数据
func (r *sentimentRepository) GetNorthboundFundByDateRange(ctx context.Context, startDate, endDate time.Time) ([]*model.NorthboundFundData, error) {
	var dataList []*model.NorthboundFundData
	err := r.db.WithContext(ctx).
		Where("trade_date >= ? AND trade_date <= ?", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Order("trade_date DESC").
		Find(&dataList).Error
	return dataList, err
}

// 北向资金十大成交股数据相关方法

// CreateNorthboundTopStock 创建北向资金十大成交股数据
func (r *sentimentRepository) CreateNorthboundTopStock(ctx context.Context, data *model.NorthboundTopStockData) error {
	return r.db.WithContext(ctx).Create(data).Error
}

// BatchCreateNorthboundTopStock 批量创建北向资金十大成交股数据
func (r *sentimentRepository) BatchCreateNorthboundTopStock(ctx context.Context, dataList []*model.NorthboundTopStockData) error {
	if len(dataList) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(dataList, 100).Error
}

// GetNorthboundTopStocksByDate 根据交易日期和市场获取北向资金十大成交股数据
func (r *sentimentRepository) GetNorthboundTopStocksByDate(ctx context.Context, tradeDate time.Time, market string) ([]*model.NorthboundTopStockData, error) {
	var dataList []*model.NorthboundTopStockData
	query := r.db.WithContext(ctx).Where("trade_date = ?", tradeDate.Format("2006-01-02"))
	
	if market != "" {
		query = query.Where("market = ?", market)
	}
	
	err := query.Order("rank ASC").Find(&dataList).Error
	return dataList, err
}

// 融资融券数据相关方法

// CreateMarginTrading 创建融资融券数据
func (r *sentimentRepository) CreateMarginTrading(ctx context.Context, data *model.MarginTradingData) error {
	return r.db.WithContext(ctx).Create(data).Error
}

// BatchCreateMarginTrading 批量创建融资融券数据
func (r *sentimentRepository) BatchCreateMarginTrading(ctx context.Context, data []interface{}) error {
	if len(data) == 0 {
		return nil
	}
	// 转换为具体类型
	dataList := make([]*model.MarginTradingData, len(data))
	for i, item := range data {
		if marginData, ok := item.(*model.MarginTradingData); ok {
			dataList[i] = marginData
		} else {
			return fmt.Errorf("invalid data type at index %d", i)
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(dataList, 100).Error
}

// GetMarginTradingByDate 根据交易日期和交易所获取融资融券数据
func (r *sentimentRepository) GetMarginTradingByDate(ctx context.Context, tradeDate time.Time, exchange string) ([]*model.MarginTradingData, error) {
	var dataList []*model.MarginTradingData
	query := r.db.WithContext(ctx).Where("trade_date = ?", tradeDate.Format("2006-01-02"))
	
	if exchange != "" {
		query = query.Where("exchange = ?", exchange)
	}
	
	err := query.Find(&dataList).Error
	return dataList, err
}

// GetMarginTradingByDateRange 根据日期范围获取融资融券数据
func (r *sentimentRepository) GetMarginTradingByDateRange(ctx context.Context, startDate, endDate time.Time, exchange string) ([]*model.MarginTradingData, error) {
	var dataList []*model.MarginTradingData
	query := r.db.WithContext(ctx).Where("trade_date >= ? AND trade_date <= ?", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if exchange != "" {
		query = query.Where("exchange = ?", exchange)
	}
	err := query.Order("trade_date DESC").Find(&dataList).Error
	return dataList, err
}

// ETF数据相关方法

// CreateETF 创建ETF数据
func (r *sentimentRepository) CreateETF(ctx context.Context, data *model.ETFData) error {
	return r.db.WithContext(ctx).Create(data).Error
}

// BatchCreateETF 批量创建ETF数据
func (r *sentimentRepository) BatchCreateETF(ctx context.Context, dataList []*model.ETFData) error {
	if len(dataList) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(dataList, 100).Error
}

// GetETFByMarket 根据市场获取ETF数据
func (r *sentimentRepository) GetETFByMarket(ctx context.Context, market string) ([]*model.ETFData, error) {
	var dataList []*model.ETFData
	query := r.db.WithContext(ctx)
	
	if market != "" {
		query = query.Where("market = ?", market)
	}
	
	err := query.Order("symbol ASC").Find(&dataList).Error
	return dataList, err
}

// GetETFBySymbol 根据代码获取ETF数据
func (r *sentimentRepository) GetETFBySymbol(ctx context.Context, symbol string) (*model.ETFData, error) {
	var etf model.ETFData
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&etf).Error
	if err != nil {
		return nil, err
	}
	return &etf, nil
}

// GetETFList 根据市场获取ETF列表
func (r *sentimentRepository) GetETFList(ctx context.Context, market string) ([]*model.ETFData, error) {
	var etfs []*model.ETFData
	query := r.db.WithContext(ctx)
	if market != "" {
		query = query.Where("market = ?", market)
	}
	err := query.Find(&etfs).Error
	return etfs, err
}