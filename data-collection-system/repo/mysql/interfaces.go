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
	List(ctx context.Context, page, pageSize int) ([]*model.Stock, int64, error)
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
	GetByTimeRange(ctx context.Context, symbol string, startTime, endTime time.Time, period string) ([]*model.MarketData, error)
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
	GetByIndicator(ctx context.Context, indicatorCode string, limit int) ([]*model.MacroData, error)
	GetByPeriodType(ctx context.Context, periodType string, limit int) ([]*model.MacroData, error)
	GetByTimeRange(ctx context.Context, indicatorCode string, startDate, endDate time.Time) ([]*model.MacroData, error)
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

// NewsRepository 新闻数据仓库接口（新爬虫系统）
type NewsRepository interface {
	Create(ctx context.Context, news *model.NewsData) error
	BatchCreate(ctx context.Context, newsList []*model.NewsData) error
	GetByID(ctx context.Context, id uint) (*model.NewsData, error)
	GetByURL(ctx context.Context, url string) (*model.NewsData, error)
	Exists(ctx context.Context, url string) (bool, error)
	List(ctx context.Context, params *NewsQueryParams) ([]*model.NewsData, int64, error)
	Update(ctx context.Context, news *model.NewsData) error
	Delete(ctx context.Context, id uint) error
	Search(ctx context.Context, params *NewsSearchParams) ([]*model.NewsData, int64, error)
	GetHotNews(ctx context.Context, limit int, hours int) ([]*model.NewsData, error)
	GetLatestNews(ctx context.Context, limit int) ([]*model.NewsData, error)
	GetByCategory(ctx context.Context, category string, limit int, offset int) ([]*model.NewsData, int64, error)
	GetBySource(ctx context.Context, source string, limit int, offset int) ([]*model.NewsData, int64, error)
	GetStats(ctx context.Context) (*NewsStats, error)
	CleanExpired(ctx context.Context, days int) (int64, error)
}

// RepositoryManager 数据仓库管理器接口
type RepositoryManager interface {
	Stock() StockRepository                   // 股票数据仓库
	MarketData() MarketDataRepository         // 行情数据仓库
	FinancialData() FinancialDataRepository   // 财务数据仓库
	MacroData() MacroDataRepository           // 宏观数据仓库
	DataTask() DataTaskRepository             // 数据任务仓库
	News() NewsRepository                     // 新闻数据仓库
	Close() error
}
