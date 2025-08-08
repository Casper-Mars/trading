package dao

import (
	"context"
	"time"

	"data-collection-system/internal/models"
)

// StockDAO 股票数据访问接口
type StockDAO interface {
	Create(ctx context.Context, stock *models.Stock) error
	GetBySymbol(ctx context.Context, symbol string) (*models.Stock, error)
	GetByExchange(ctx context.Context, exchange string, limit, offset int) ([]*models.Stock, error)
	GetByIndustry(ctx context.Context, industry string, limit, offset int) ([]*models.Stock, error)
	GetActiveStocks(ctx context.Context, limit, offset int) ([]*models.Stock, error)
	Update(ctx context.Context, stock *models.Stock) error
	Delete(ctx context.Context, symbol string) error
	Count(ctx context.Context) (int64, error)
}

// MarketDataDAO 行情数据访问接口
type MarketDataDAO interface {
	Create(ctx context.Context, data *models.MarketData) error
	BatchCreate(ctx context.Context, data []*models.MarketData) error
	GetBySymbol(ctx context.Context, symbol string, startDate, endDate time.Time, period string, limit, offset int) ([]*models.MarketData, error)
	GetLatest(ctx context.Context, symbol string, period string) (*models.MarketData, error)
	GetByDate(ctx context.Context, date time.Time, period string, limit, offset int) ([]*models.MarketData, error)
	Update(ctx context.Context, data *models.MarketData) error
	Delete(ctx context.Context, id uint64) error
	DeleteBySymbolAndDateRange(ctx context.Context, symbol string, startDate, endDate time.Time) error
	Count(ctx context.Context) (int64, error)
}

// FinancialDataDAO 财务数据访问接口
type FinancialDataDAO interface {
	Create(ctx context.Context, data *models.FinancialData) error
	BatchCreate(ctx context.Context, data []*models.FinancialData) error
	GetBySymbol(ctx context.Context, symbol string, startDate, endDate time.Time, reportType string, limit, offset int) ([]*models.FinancialData, error)
	GetLatest(ctx context.Context, symbol string, reportType string) (*models.FinancialData, error)
	GetByReportDate(ctx context.Context, reportDate time.Time, reportType string, limit, offset int) ([]*models.FinancialData, error)
	Update(ctx context.Context, data *models.FinancialData) error
	Delete(ctx context.Context, id uint64) error
	Count(ctx context.Context) (int64, error)
}

// NewsDataDAO 新闻数据访问接口
type NewsDataDAO interface {
	Create(ctx context.Context, news *models.NewsData) error
	BatchCreate(ctx context.Context, news []*models.NewsData) error
	GetByTimeRange(ctx context.Context, startTime, endTime time.Time, limit, offset int) ([]*models.NewsData, error)
	GetByCategory(ctx context.Context, category string, limit, offset int) ([]*models.NewsData, error)
	GetBySentiment(ctx context.Context, sentiment int8, limit, offset int) ([]*models.NewsData, error)
	GetByImportance(ctx context.Context, minLevel int8, limit, offset int) ([]*models.NewsData, error)
	GetByRelatedStock(ctx context.Context, symbol string, limit, offset int) ([]*models.NewsData, error)
	SearchByKeyword(ctx context.Context, keyword string, limit, offset int) ([]*models.NewsData, error)
	Update(ctx context.Context, news *models.NewsData) error
	Delete(ctx context.Context, id uint64) error
	Count(ctx context.Context) (int64, error)
}

// MacroDataDAO 宏观数据访问接口
type MacroDataDAO interface {
	Create(ctx context.Context, data *models.MacroData) error
	BatchCreate(ctx context.Context, data []*models.MacroData) error
	GetByIndicator(ctx context.Context, indicatorCode string, startDate, endDate time.Time, limit, offset int) ([]*models.MacroData, error)
	GetLatest(ctx context.Context, indicatorCode string) (*models.MacroData, error)
	GetByPeriodType(ctx context.Context, periodType string, limit, offset int) ([]*models.MacroData, error)
	GetByDateRange(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]*models.MacroData, error)
	Update(ctx context.Context, data *models.MacroData) error
	Delete(ctx context.Context, id uint64) error
	Count(ctx context.Context) (int64, error)
}

// DataTaskDAO 任务数据访问接口
type DataTaskDAO interface {
	Create(ctx context.Context, task *models.DataTask) error
	GetByID(ctx context.Context, id uint64) (*models.DataTask, error)
	GetByName(ctx context.Context, taskName string) (*models.DataTask, error)
	GetByType(ctx context.Context, taskType string, limit, offset int) ([]*models.DataTask, error)
	GetByStatus(ctx context.Context, status int8, limit, offset int) ([]*models.DataTask, error)
	GetEnabledTasks(ctx context.Context) ([]*models.DataTask, error)
	GetTasksToRun(ctx context.Context) ([]*models.DataTask, error)
	GetAll(ctx context.Context, limit, offset int) ([]*models.DataTask, error)
	Update(ctx context.Context, task *models.DataTask) error
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