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
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	Category   string    `json:"category"`
	Source     string    `json:"source"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Keyword    string    `json:"keyword"`
	OrderBy    string    `json:"order_by"`    // published_at, created_at, view_count
	OrderDir   string    `json:"order_dir"`   // asc, desc
	Status     int       `json:"status"`      // 0: 全部, 1: 已发布, 2: 草稿
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

// StockDAO 股票数据访问接口
type StockDAO interface {
	Create(ctx context.Context, stock *model.Stock) error
	GetBySymbol(ctx context.Context, symbol string) (*model.Stock, error)
	GetByExchange(ctx context.Context, exchange string, limit, offset int) ([]*model.Stock, error)
	GetByIndustry(ctx context.Context, industry string, limit, offset int) ([]*model.Stock, error)
	GetActiveStocks(ctx context.Context, limit, offset int) ([]*model.Stock, error)
	Update(ctx context.Context, stock *model.Stock) error
	Delete(ctx context.Context, symbol string) error
	Count(ctx context.Context) (int64, error)
}

// MarketDataDAO 行情数据访问接口
type MarketDataDAO interface {
	Create(ctx context.Context, data *model.MarketData) error
	BatchCreate(ctx context.Context, data []*model.MarketData) error
	GetBySymbol(ctx context.Context, symbol string, startDate, endDate time.Time, period string, limit, offset int) ([]*model.MarketData, error)
	GetLatest(ctx context.Context, symbol string, period string) (*model.MarketData, error)
	GetByDate(ctx context.Context, date time.Time, period string, limit, offset int) ([]*model.MarketData, error)
	Update(ctx context.Context, data *model.MarketData) error
	Delete(ctx context.Context, id uint64) error
	DeleteBySymbolAndDateRange(ctx context.Context, symbol string, startDate, endDate time.Time) error
	Count(ctx context.Context) (int64, error)
}

// FinancialDataDAO 财务数据访问接口
type FinancialDataDAO interface {
	Create(ctx context.Context, data *model.FinancialData) error
	BatchCreate(ctx context.Context, data []*model.FinancialData) error
	GetBySymbol(ctx context.Context, symbol string, startDate, endDate time.Time, reportType string, limit, offset int) ([]*model.FinancialData, error)
	GetLatest(ctx context.Context, symbol string, reportType string) (*model.FinancialData, error)
	GetByReportDate(ctx context.Context, reportDate time.Time, reportType string, limit, offset int) ([]*model.FinancialData, error)
	Update(ctx context.Context, data *model.FinancialData) error
	Delete(ctx context.Context, id uint64) error
	Count(ctx context.Context) (int64, error)
}

// NewsDataDAO 新闻数据访问接口
type NewsDataDAO interface {
	Create(ctx context.Context, news *model.NewsData) error
	BatchCreate(ctx context.Context, news []*model.NewsData) error
	GetByTimeRange(ctx context.Context, startTime, endTime time.Time, limit, offset int) ([]*model.NewsData, error)
	GetByCategory(ctx context.Context, category string, limit, offset int) ([]*model.NewsData, error)
	GetBySentiment(ctx context.Context, sentiment int8, limit, offset int) ([]*model.NewsData, error)
	GetByImportance(ctx context.Context, minLevel int8, limit, offset int) ([]*model.NewsData, error)
	GetByRelatedStock(ctx context.Context, symbol string, limit, offset int) ([]*model.NewsData, error)
	SearchByKeyword(ctx context.Context, keyword string, limit, offset int) ([]*model.NewsData, error)
	Update(ctx context.Context, news *model.NewsData) error
	Delete(ctx context.Context, id uint64) error
	Count(ctx context.Context) (int64, error)
}

// MacroDataDAO 宏观数据访问接口
type MacroDataDAO interface {
	Create(ctx context.Context, data *model.MacroData) error
	BatchCreate(ctx context.Context, data []*model.MacroData) error
	GetByIndicator(ctx context.Context, indicatorCode string, startDate, endDate time.Time, limit, offset int) ([]*model.MacroData, error)
	GetLatest(ctx context.Context, indicatorCode string) (*model.MacroData, error)
	GetByPeriodType(ctx context.Context, periodType string, limit, offset int) ([]*model.MacroData, error)
	GetByDateRange(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]*model.MacroData, error)
	Update(ctx context.Context, data *model.MacroData) error
	Delete(ctx context.Context, id uint64) error
	Count(ctx context.Context) (int64, error)
}

// DataTaskDAO 任务数据访问接口
type DataTaskDAO interface {
	Create(ctx context.Context, task *model.DataTask) error
	GetByID(ctx context.Context, id uint64) (*model.DataTask, error)
	GetByName(ctx context.Context, taskName string) (*model.DataTask, error)
	GetByType(ctx context.Context, taskType string, limit, offset int) ([]*model.DataTask, error)
	GetByStatus(ctx context.Context, status int8, limit, offset int) ([]*model.DataTask, error)
	GetEnabledTasks(ctx context.Context) ([]*model.DataTask, error)
	GetTasksToRun(ctx context.Context) ([]*model.DataTask, error)
	GetAll(ctx context.Context, limit, offset int) ([]*model.DataTask, error)
	Update(ctx context.Context, task *model.DataTask) error
	UpdateStatus(ctx context.Context, id uint64, status int8) error
	UpdateRunStats(ctx context.Context, id uint64, lastRunAt, nextRunAt *time.Time, runCount, successCount, failureCount uint64) error
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

// DAOManager DAO管理器接口
type DAOManager interface {
	Stock() StockDAO
	MarketData() MarketDataDAO
	FinancialData() FinancialDataDAO
	NewsData() NewsDataDAO
	News() NewsRepository  // 新爬虫系统的新闻仓库
	MacroData() MacroDataDAO
	DataTask() DataTaskDAO
	Close() error
}