package dao

import (
	"context"
	"time"

	"data-collection-system/model"
)

// 类型别名
// type News = model.NewsData  // 注释掉，避免与model.News冲突

// NewsQueryParams 新闻查询参数
type NewsQueryParams struct {
	Page      int       `json:"page"`
	PageSize  int       `json:"page_size"`
	Category  string    `json:"category"`
	Source    string    `json:"source"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Keyword   string    `json:"keyword"`
	OrderBy   string    `json:"order_by"`  // published_at, created_at, view_count
	OrderDir  string    `json:"order_dir"` // asc, desc
	Status    int       `json:"status"`    // 0: 全部, 1: 已发布, 2: 草稿
}

// NewsSearchParams 新闻搜索参数
type NewsSearchParams struct {
	Keyword   string    `json:"keyword"`
	Category  string    `json:"category"`
	Source    string    `json:"source"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
}

// NewsStats 新闻统计信息
type NewsStats struct {
	TotalCount     int64            `json:"total_count"`
	TodayCount     int64            `json:"today_count"`
	WeekCount      int64            `json:"week_count"`
	MonthCount     int64            `json:"month_count"`
	CategoryStats  map[string]int64 `json:"category_stats"`
	SourceStats    map[string]int64 `json:"source_stats"`
	LastUpdateTime time.Time        `json:"last_update_time"`
}

// 移除所有DAO接口，统一使用Repository接口

// StockRepository 股票数据仓库接口
type StockRepository interface {
	Create(ctx context.Context, stock *model.Stock) error
	BatchCreate(ctx context.Context, stocks []*model.Stock) error
	GetByID(ctx context.Context, id uint64) (*model.Stock, error)
	GetBySymbol(ctx context.Context, symbol string) (*model.Stock, error)
	List(ctx context.Context, offset, limit int) ([]*model.Stock, error)
	ListWithPagination(ctx context.Context, page, pageSize int) ([]*model.Stock, int64, error)
	Update(ctx context.Context, stock *model.Stock) error
	Delete(ctx context.Context, id uint64) error
	GetByExchange(ctx context.Context, exchange string) ([]*model.Stock, error)
	GetByIndustry(ctx context.Context, industry string) ([]*model.Stock, error)
	GetBySector(ctx context.Context, sector string) ([]*model.Stock, error)
	GetActiveStocks(ctx context.Context) ([]*model.Stock, error)
	Search(ctx context.Context, keyword string) ([]*model.Stock, error)
	Count(ctx context.Context) (int64, error)
}

// MarketDataRepository 行情数据仓库接口
type MarketDataRepository interface {
	Create(ctx context.Context, data *model.MarketData) error
	BatchCreate(ctx context.Context, dataList []*model.MarketData) error
	GetByID(ctx context.Context, id uint64) (*model.MarketData, error)
	GetBySymbolAndTime(ctx context.Context, symbol string, tradeTime time.Time, period string) (*model.MarketData, error)
	GetBySymbolAndDate(ctx context.Context, symbol string, date time.Time) (*model.MarketData, error)
	GetByTimeRange(ctx context.Context, symbol string, startTime, endTime time.Time, period string) ([]*model.MarketData, error)
	GetByDateRange(ctx context.Context, symbol string, startDate, endDate time.Time) ([]*model.MarketData, error)
	GetLatest(ctx context.Context, symbol string, period string) (*model.MarketData, error)
	Update(ctx context.Context, data *model.MarketData) error
	Delete(ctx context.Context, id uint64) error
	DeleteByTimeRange(ctx context.Context, symbol string, startTime, endTime time.Time, period string) error
	Count(ctx context.Context) (int64, error)
}

// FinancialDataRepository 财务数据仓库接口
type FinancialDataRepository interface {
	Create(ctx context.Context, data *model.FinancialData) error
	BatchCreate(ctx context.Context, dataList []*model.FinancialData) error
	GetByID(ctx context.Context, id uint64) (*model.FinancialData, error)
	GetBySymbolAndReport(ctx context.Context, symbol string, reportDate time.Time, reportType string) (*model.FinancialData, error)
	GetBySymbolAndPeriod(ctx context.Context, symbol string, reportDate time.Time, reportType string) (*model.FinancialData, error)
	GetBySymbol(ctx context.Context, symbol string, limit int) ([]*model.FinancialData, error)
	GetByReportType(ctx context.Context, reportType string, limit int) ([]*model.FinancialData, error)
	GetLatest(ctx context.Context, symbol string) (*model.FinancialData, error)
	Update(ctx context.Context, data *model.FinancialData) error
	Delete(ctx context.Context, id uint64) error
	Count(ctx context.Context) (int64, error)
}

// MacroDataRepository 宏观数据仓库接口
type MacroDataRepository interface {
	Create(ctx context.Context, data *model.MacroData) error
	BatchCreate(ctx context.Context, dataList []*model.MacroData) error
	GetByID(ctx context.Context, id uint64) (*model.MacroData, error)
	GetByIndicatorAndDate(ctx context.Context, indicatorCode string, dataDate time.Time) (*model.MacroData, error)
	GetByIndicator(ctx context.Context, indicatorCode string, startDate, endDate time.Time) ([]*model.MacroData, error)
	GetByIndicatorWithLimit(ctx context.Context, indicatorCode string, limit int) ([]*model.MacroData, error)
	GetByPeriodType(ctx context.Context, periodType string, limit int) ([]*model.MacroData, error)
	GetByTimeRange(ctx context.Context, indicatorCode string, startDate, endDate time.Time) ([]*model.MacroData, error)
	GetLatest(ctx context.Context, indicatorCode string) (*model.MacroData, error)
	Update(ctx context.Context, data *model.MacroData) error
	Delete(ctx context.Context, id uint64) error
	Count(ctx context.Context) (int64, error)
}

// DataTaskRepository 数据任务仓库接口
type DataTaskRepository interface {
	Create(ctx context.Context, task *model.DataTask) error
	GetByID(ctx context.Context, id uint64) (*model.DataTask, error)
	GetByName(ctx context.Context, taskName string) (*model.DataTask, error)
	List(ctx context.Context, page, pageSize int) ([]*model.DataTask, int64, error)
	GetByType(ctx context.Context, taskType string) ([]*model.DataTask, error)
	GetByStatus(ctx context.Context, status int8) ([]*model.DataTask, error)
	GetPendingTasks(ctx context.Context) ([]*model.DataTask, error)
	Update(ctx context.Context, task *model.DataTask) error
	Delete(ctx context.Context, id uint64) error
	Count(ctx context.Context) (int64, error)
}

// StockMatchingStats 股票匹配统计信息
type StockMatchingStats struct {
	NewsWithStocks int64   `json:"news_with_stocks"`
	TotalNews      int64   `json:"total_news"`
	MatchingRate   float64 `json:"matching_rate"`
}

// ImportanceStats 重要程度统计信息
type ImportanceStats struct {
	TotalCount    int64   `json:"total_count"`
	VeryLowCount  int64   `json:"very_low_count"`
	LowCount      int64   `json:"low_count"`
	MediumCount   int64   `json:"medium_count"`
	HighCount     int64   `json:"high_count"`
	VeryHighCount int64   `json:"very_high_count"`
	VeryLowRate   float64 `json:"very_low_rate"`
	LowRate       float64 `json:"low_rate"`
	MediumRate    float64 `json:"medium_rate"`
	HighRate      float64 `json:"high_rate"`
	VeryHighRate  float64 `json:"very_high_rate"`
}

// DuplicationStats 去重统计信息
type DuplicationStats struct {
	TotalNews         int64   `json:"total_news"`
	DuplicateNews     int64   `json:"duplicate_news"`
	DeduplicationRate float64 `json:"deduplication_rate"`
}

// NewsRepository 新闻数据仓库接口（新爬虫系统）
type NewsRepository interface {
	Create(ctx context.Context, news *model.NewsData) error
	BatchCreate(ctx context.Context, newsList []*model.NewsData) error
	GetByID(ctx context.Context, id uint) (*model.NewsData, error)
	GetByURL(ctx context.Context, url string) (*model.NewsData, error)
	Exists(ctx context.Context, url string) (bool, error)
	ExistsByURL(ctx context.Context, url string) (bool, error)
	List(ctx context.Context, params *NewsQueryParams) ([]*model.NewsData, int64, error)
	GetByTimeRange(ctx context.Context, startTime, endTime time.Time, limit, offset int) ([]*model.NewsData, error)
	Update(ctx context.Context, news *model.NewsData) error
	Delete(ctx context.Context, id uint) error
	Search(ctx context.Context, params *NewsSearchParams) ([]*model.NewsData, int64, error)
	GetHotNews(ctx context.Context, limit int, hours int) ([]*model.NewsData, error)
	GetLatestNews(ctx context.Context, limit int) ([]*model.NewsData, error)
	GetByCategory(ctx context.Context, category string, limit int, offset int) ([]*model.NewsData, int64, error)
	GetBySource(ctx context.Context, source string, limit int, offset int) ([]*model.NewsData, int64, error)
	GetStats(ctx context.Context) (*NewsStats, error)
	CleanExpired(ctx context.Context, days int) (int64, error)
	// 新增统计方法
	GetStockMatchingStats(ctx context.Context) (*StockMatchingStats, error)
	GetImportanceStats(ctx context.Context) (*ImportanceStats, error)
	GetDuplicationStats(ctx context.Context) (*DuplicationStats, error)
}

// SentimentRepository 市场情绪数据仓库接口
type SentimentRepository interface {
	// 市场情绪数据
	CreateMarketSentiment(ctx context.Context, data *model.MarketSentimentData) error
	BatchCreateMarketSentiment(ctx context.Context, data []interface{}) error
	GetMarketSentimentByDate(ctx context.Context, date time.Time) ([]*model.MarketSentimentData, error)
	GetMarketSentimentByDateRange(ctx context.Context, startDate, endDate time.Time, dataType string) ([]*model.MarketSentimentData, error)
	
	// 资金流向数据
	CreateMoneyFlow(ctx context.Context, data *model.MoneyFlowData) error
	BatchCreateMoneyFlow(ctx context.Context, data []interface{}) error
	GetMoneyFlowBySymbolAndDate(ctx context.Context, symbol string, date time.Time) (*model.MoneyFlowData, error)
	GetMoneyFlowByDateRange(ctx context.Context, symbol string, startDate, endDate time.Time) ([]*model.MoneyFlowData, error)
	
	// 北向资金数据
	CreateNorthboundFund(ctx context.Context, data *model.NorthboundFundData) error
	BatchCreateNorthboundFund(ctx context.Context, dataList []*model.NorthboundFundData) error
	GetNorthboundFundByDate(ctx context.Context, tradeDate time.Time) ([]*model.NorthboundFundData, error)
	GetNorthboundFundByDateRange(ctx context.Context, startDate, endDate time.Time) ([]*model.NorthboundFundData, error)
	
	// 北向资金十大成交股数据
	CreateNorthboundTopStock(ctx context.Context, data *model.NorthboundTopStockData) error
	BatchCreateNorthboundTopStock(ctx context.Context, dataList []*model.NorthboundTopStockData) error
	GetNorthboundTopStocksByDate(ctx context.Context, tradeDate time.Time, market string) ([]*model.NorthboundTopStockData, error)
	
	// 融资融券数据
	CreateMarginTrading(ctx context.Context, data *model.MarginTradingData) error
	BatchCreateMarginTrading(ctx context.Context, data []interface{}) error
	GetMarginTradingByDate(ctx context.Context, date time.Time, exchange string) ([]*model.MarginTradingData, error)
	GetMarginTradingByDateRange(ctx context.Context, startDate, endDate time.Time, exchange string) ([]*model.MarginTradingData, error)
	
	// ETF数据
	CreateETF(ctx context.Context, data *model.ETFData) error
	BatchCreateETF(ctx context.Context, dataList []*model.ETFData) error
	GetETFByMarket(ctx context.Context, market string) ([]*model.ETFData, error)
	GetETFBySymbol(ctx context.Context, symbol string) (*model.ETFData, error)
	GetETFList(ctx context.Context, market string) ([]*model.ETFData, error)
}

// RepositoryManager 数据仓库管理器接口
type RepositoryManager interface {
	Stock() StockRepository                   // 股票数据仓库
	MarketData() MarketDataRepository         // 行情数据仓库
	FinancialData() FinancialDataRepository   // 财务数据仓库
	MacroData() MacroDataRepository           // 宏观数据仓库
	DataTask() DataTaskRepository             // 数据任务仓库
	News() NewsRepository                     // 新闻数据仓库
	Sentiment() SentimentRepository           // 市场情绪数据仓库
	Close() error
}
