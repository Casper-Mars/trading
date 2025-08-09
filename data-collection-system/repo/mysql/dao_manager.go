package dao

import (
	"gorm.io/gorm"
)

// daoManager DAO管理器实现
type daoManager struct {
	db           *gorm.DB
	stockDAO     StockDAO
	marketDAO    MarketDataDAO
	financialDAO FinancialDataDAO
	newsDAO      NewsDataDAO
	newsRepo     NewsRepository  // 新爬虫系统的新闻仓库
	macroDAO     MacroDataDAO
	taskDAO      DataTaskDAO
}

// NewDAOManager 创建DAO管理器实例
func NewDAOManager(db *gorm.DB) DAOManager {
	return &daoManager{
		db:           db,
		stockDAO:     NewStockDAO(db),
		marketDAO:    NewMarketDataDAO(db),
		financialDAO: NewFinancialDataDAO(db),
		newsDAO:      NewNewsDataDAO(db),
		newsRepo:     NewNewsRepository(db),  // 新爬虫系统的新闻仓库
		macroDAO:     NewMacroDataDAO(db),
		taskDAO:      NewDataTaskDAO(db),
	}
}

// Stock 获取股票DAO
func (d *daoManager) Stock() StockDAO {
	return d.stockDAO
}

// MarketData 获取行情数据DAO
func (d *daoManager) MarketData() MarketDataDAO {
	return d.marketDAO
}

// FinancialData 获取财务数据DAO
func (d *daoManager) FinancialData() FinancialDataDAO {
	return d.financialDAO
}

// NewsData 获取新闻数据DAO
func (d *daoManager) NewsData() NewsDataDAO {
	return d.newsDAO
}

// News 获取新闻仓库（新爬虫系统）
func (d *daoManager) News() NewsRepository {
	return d.newsRepo
}

// MacroData 获取宏观数据DAO
func (d *daoManager) MacroData() MacroDataDAO {
	return d.macroDAO
}

// DataTask 获取任务DAO
func (d *daoManager) DataTask() DataTaskDAO {
	return d.taskDAO
}

// Close 关闭DAO管理器
func (d *daoManager) Close() error {
	// 这里可以添加清理逻辑
	return nil
}