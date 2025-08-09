package collector

import (
	"context"
	"data-collection-system/internal/models"
)

// DataCollector 数据采集器接口
type DataCollector interface {
	// ValidateConnection 验证连接
	ValidateConnection(ctx context.Context) error
	
	// Close 关闭采集器
	Close()
}

// StockDataCollector 股票数据采集器接口
type StockDataCollector interface {
	DataCollector
	
	// CollectStockBasic 采集股票基础数据
	CollectStockBasic(ctx context.Context, exchange string) ([]*models.Stock, error)
	
	// CollectMarketData 采集行情数据
	// period: 数据周期，支持 "1min", "5min", "15min", "30min", "60min", "daily"
	CollectMarketData(ctx context.Context, symbol, period, startDate, endDate string) ([]*models.MarketData, error)
}

// FinancialDataCollector 财务数据采集器接口
type FinancialDataCollector interface {
	DataCollector
	
	// CollectFinancialData 采集财务数据
	CollectFinancialData(ctx context.Context, symbol, reportType, startDate, endDate string) (*models.FinancialData, error)
}

// MacroDataCollector 宏观数据采集器接口
type MacroDataCollector interface {
	DataCollector
	
	// CollectMacroData 采集宏观数据
	CollectMacroData(ctx context.Context, indicator, period, startDate, endDate string) ([]*models.MacroData, error)
}

// ComprehensiveCollector 综合数据采集器接口
type ComprehensiveCollector interface {
	StockDataCollector
	FinancialDataCollector
	MacroDataCollector
}

// CollectorManager 采集器管理器
type CollectorManager struct {
	collectors map[string]DataCollector
}

// NewCollectorManager 创建采集器管理器
func NewCollectorManager() *CollectorManager {
	return &CollectorManager{
		collectors: make(map[string]DataCollector),
	}
}

// RegisterCollector 注册采集器
func (cm *CollectorManager) RegisterCollector(name string, collector DataCollector) {
	cm.collectors[name] = collector
}

// GetCollector 获取采集器
func (cm *CollectorManager) GetCollector(name string) (DataCollector, bool) {
	collector, exists := cm.collectors[name]
	return collector, exists
}

// GetStockCollector 获取股票数据采集器
func (cm *CollectorManager) GetStockCollector(name string) (StockDataCollector, bool) {
	collector, exists := cm.collectors[name]
	if !exists {
		return nil, false
	}
	if stockCollector, ok := collector.(StockDataCollector); ok {
		return stockCollector, true
	}
	return nil, false
}

// GetFinancialCollector 获取财务数据采集器
func (cm *CollectorManager) GetFinancialCollector(name string) (FinancialDataCollector, bool) {
	collector, exists := cm.collectors[name]
	if !exists {
		return nil, false
	}
	if financialCollector, ok := collector.(FinancialDataCollector); ok {
		return financialCollector, true
	}
	return nil, false
}

// GetMacroCollector 获取宏观数据采集器
func (cm *CollectorManager) GetMacroCollector(name string) (MacroDataCollector, bool) {
	collector, exists := cm.collectors[name]
	if !exists {
		return nil, false
	}
	if macroCollector, ok := collector.(MacroDataCollector); ok {
		return macroCollector, true
	}
	return nil, false
}

// GetComprehensiveCollector 获取综合数据采集器
func (cm *CollectorManager) GetComprehensiveCollector(name string) (ComprehensiveCollector, bool) {
	collector, exists := cm.collectors[name]
	if !exists {
		return nil, false
	}
	if comprehensiveCollector, ok := collector.(ComprehensiveCollector); ok {
		return comprehensiveCollector, true
	}
	return nil, false
}

// ValidateAllConnections 验证所有采集器连接
func (cm *CollectorManager) ValidateAllConnections(ctx context.Context) map[string]error {
	results := make(map[string]error)
	for name, collector := range cm.collectors {
		results[name] = collector.ValidateConnection(ctx)
	}
	return results
}

// CloseAll 关闭所有采集器
func (cm *CollectorManager) CloseAll() {
	for _, collector := range cm.collectors {
		collector.Close()
	}
}

// ListCollectors 列出所有注册的采集器
func (cm *CollectorManager) ListCollectors() []string {
	names := make([]string, 0, len(cm.collectors))
	for name := range cm.collectors {
		names = append(names, name)
	}
	return names
}