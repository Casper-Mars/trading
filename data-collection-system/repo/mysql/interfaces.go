package dao

import (
	"context"
	"time"

	"data-collection-system/model"
)

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

// DAOManager DAO管理器接口
type DAOManager interface {
	Stock() StockDAO
	MarketData() MarketDataDAO
	FinancialData() FinancialDataDAO
	NewsData() NewsDataDAO
	MacroData() MacroDataDAO
	DataTask() DataTaskDAO
	Close() error
}