package dao

import (
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// repositoryManager 数据仓库管理器
type repositoryManager struct {
	db             *gorm.DB
	redisClient    *redis.Client
	stockRepo      StockRepository         // 股票数据仓库
	marketDataRepo MarketDataRepository    // 行情数据仓库
	financialRepo  FinancialDataRepository // 财务数据仓库
	macroDataRepo  MacroDataRepository     // 宏观数据仓库
	dataTaskRepo   DataTaskRepository      // 数据任务仓库
	newsRepo       NewsRepository          // 新闻数据仓库
	sentimentRepo  SentimentRepository     // 市场情绪数据仓库
}

// NewRepositoryManager 创建数据仓库管理器实例
func NewRepositoryManager(db *gorm.DB, redisClient *redis.Client) RepositoryManager {
	return &repositoryManager{
		db:             db,
		redisClient:    redisClient,
		stockRepo:      NewStockRepository(db),
		marketDataRepo: NewMarketDataRepository(db),
		financialRepo:  NewFinancialDataRepository(db),
		macroDataRepo:  NewMacroDataRepository(db),
		dataTaskRepo:   NewDataTaskRepository(db),
		newsRepo:       NewNewsRepository(db, redisClient),
		sentimentRepo:  NewSentimentRepository(db),
	}
}

// Stock 获取股票数据仓库
func (r *repositoryManager) Stock() StockRepository {
	return r.stockRepo
}

// MarketData 获取行情数据仓库
func (r *repositoryManager) MarketData() MarketDataRepository {
	return r.marketDataRepo
}

// FinancialData 获取财务数据仓库
func (r *repositoryManager) FinancialData() FinancialDataRepository {
	return r.financialRepo
}

// MacroData 获取宏观数据仓库
func (r *repositoryManager) MacroData() MacroDataRepository {
	return r.macroDataRepo
}

// DataTask 获取数据任务仓库
func (r *repositoryManager) DataTask() DataTaskRepository {
	return r.dataTaskRepo
}

// News 获取新闻数据仓库
func (r *repositoryManager) News() NewsRepository {
	return r.newsRepo
}

// Sentiment 获取市场情绪数据仓库
func (r *repositoryManager) Sentiment() SentimentRepository {
	return r.sentimentRepo
}

// Close 关闭仓库管理器
func (r *repositoryManager) Close() error {
	// 这里可以添加清理逻辑
	return nil
}
