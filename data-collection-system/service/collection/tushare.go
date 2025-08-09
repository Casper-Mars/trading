package collection

import (
	"context"
	"fmt"
	"log"
	"time"

	"data-collection-system/model"
	"data-collection-system/repo/external/tushare"
)

// TushareService Tushare数据采集业务服务
type TushareService struct {
	collector    *tushare.Collector
	stockRepo    StockRepository
	marketRepo   MarketRepository
	financialRepo FinancialRepository
	macroRepo    MacroRepository
}

// StockRepository 股票数据仓储接口
type StockRepository interface {
	Create(ctx context.Context, stock *model.Stock) error
	BatchCreate(ctx context.Context, stocks []*model.Stock) error
	GetBySymbol(ctx context.Context, symbol string) (*model.Stock, error)
	Update(ctx context.Context, stock *model.Stock) error
	List(ctx context.Context, offset, limit int) ([]*model.Stock, error)
}

// MarketRepository 市场数据仓储接口
type MarketRepository interface {
	Create(ctx context.Context, data *model.MarketData) error
	BatchCreate(ctx context.Context, data []*model.MarketData) error
	GetBySymbolAndDate(ctx context.Context, symbol string, date time.Time) (*model.MarketData, error)
	GetByDateRange(ctx context.Context, symbol string, startDate, endDate time.Time) ([]*model.MarketData, error)
	GetLatest(ctx context.Context, symbol string) (*model.MarketData, error)
}

// FinancialRepository 财务数据仓储接口
type FinancialRepository interface {
	Create(ctx context.Context, data *model.FinancialData) error
	BatchCreate(ctx context.Context, data []*model.FinancialData) error
	GetBySymbolAndPeriod(ctx context.Context, symbol string, reportDate time.Time, reportType string) (*model.FinancialData, error)
	GetBySymbol(ctx context.Context, symbol string, limit int) ([]*model.FinancialData, error)
}

// MacroRepository 宏观数据仓储接口
type MacroRepository interface {
	Create(ctx context.Context, data *model.MacroData) error
	BatchCreate(ctx context.Context, data []*model.MacroData) error
	GetByIndicator(ctx context.Context, indicatorCode string, startDate, endDate time.Time) ([]*model.MacroData, error)
	GetLatest(ctx context.Context, indicatorCode string) (*model.MacroData, error)
}

// NewTushareService 创建Tushare数据采集服务
func NewTushareService(
	collector *tushare.Collector,
	stockRepo StockRepository,
	marketRepo MarketRepository,
	financialRepo FinancialRepository,
	macroRepo MacroRepository,
) *TushareService {
	return &TushareService{
		collector:     collector,
		stockRepo:     stockRepo,
		marketRepo:    marketRepo,
		financialRepo: financialRepo,
		macroRepo:     macroRepo,
	}
}

// CollectStockBasicData 采集股票基础数据
func (s *TushareService) CollectStockBasicData(ctx context.Context) error {
	log.Println("开始采集股票基础数据...")

	// 从Tushare获取股票列表
	stocks, err := s.collector.GetStockCollector().GetStockBasic(ctx, "L")
	if err != nil {
		return fmt.Errorf("获取股票基础数据失败: %w", err)
	}

	log.Printf("从Tushare获取到 %d 只股票数据", len(stocks))

	// 批量保存到数据库
	if err := s.stockRepo.BatchCreate(ctx, stocks); err != nil {
		return fmt.Errorf("保存股票基础数据失败: %w", err)
	}

	log.Printf("成功保存 %d 只股票基础数据", len(stocks))
	return nil
}

// CollectDailyMarketData 采集日线行情数据
func (s *TushareService) CollectDailyMarketData(ctx context.Context, tradeDate string) error {
	log.Printf("开始采集 %s 的日线行情数据...", tradeDate)

	// 检查是否为交易日
	isTradeDay, err := s.collector.GetStockCollector().IsTradeDay(ctx, tradeDate)
	if err != nil {
		return fmt.Errorf("检查交易日失败: %w", err)
	}

	if !isTradeDay {
		log.Printf("%s 不是交易日，跳过采集", tradeDate)
		return nil
	}

	// 从Tushare获取当日行情数据
	marketData, err := s.collector.GetMarketCollector().GetDailyDataByDate(ctx, tradeDate)
	if err != nil {
		return fmt.Errorf("获取日线行情数据失败: %w", err)
	}

	log.Printf("从Tushare获取到 %d 条行情数据", len(marketData))

	// 批量保存到数据库
	if err := s.marketRepo.BatchCreate(ctx, marketData); err != nil {
		return fmt.Errorf("保存日线行情数据失败: %w", err)
	}

	log.Printf("成功保存 %d 条日线行情数据", len(marketData))
	return nil
}

// CollectStockHistoryData 采集指定股票的历史行情数据
func (s *TushareService) CollectStockHistoryData(ctx context.Context, symbol, startDate, endDate string) error {
	log.Printf("开始采集股票 %s 从 %s 到 %s 的历史行情数据...", symbol, startDate, endDate)

	// 从Tushare获取历史行情数据
	marketData, err := s.collector.GetMarketCollector().GetDailyData(ctx, symbol, startDate, endDate)
	if err != nil {
		return fmt.Errorf("获取历史行情数据失败: %w", err)
	}

	log.Printf("从Tushare获取到股票 %s 的 %d 条历史行情数据", symbol, len(marketData))

	// 批量保存到数据库
	if err := s.marketRepo.BatchCreate(ctx, marketData); err != nil {
		return fmt.Errorf("保存历史行情数据失败: %w", err)
	}

	log.Printf("成功保存股票 %s 的 %d 条历史行情数据", symbol, len(marketData))
	return nil
}

// CollectFinancialData 采集财务数据
func (s *TushareService) CollectFinancialData(ctx context.Context, symbol, period string) error {
	log.Printf("开始采集股票 %s 期间 %s 的财务数据...", symbol, period)

	var allFinancialData []*model.FinancialData

	// 采集利润表数据
	incomeData, err := s.collector.GetFinancialCollector().GetIncomeStatement(ctx, symbol, period, "", "")
	if err != nil {
		log.Printf("获取利润表数据失败: %v", err)
	} else {
		allFinancialData = append(allFinancialData, incomeData...)
		log.Printf("从Tushare获取到 %d 条利润表数据", len(incomeData))
	}

	// 采集资产负债表数据
	balanceData, err := s.collector.GetFinancialCollector().GetBalanceSheet(ctx, symbol, period, "", "")
	if err != nil {
		log.Printf("获取资产负债表数据失败: %v", err)
	} else {
		allFinancialData = append(allFinancialData, balanceData...)
		log.Printf("从Tushare获取到 %d 条资产负债表数据", len(balanceData))
	}

	// 采集现金流量表数据
	cashFlowData, err := s.collector.GetFinancialCollector().GetCashFlow(ctx, symbol, period, "", "")
	if err != nil {
		log.Printf("获取现金流量表数据失败: %v", err)
	} else {
		allFinancialData = append(allFinancialData, cashFlowData...)
		log.Printf("从Tushare获取到 %d 条现金流量表数据", len(cashFlowData))
	}

	// 批量保存财务数据
	if len(allFinancialData) > 0 {
		if err := s.financialRepo.BatchCreate(ctx, allFinancialData); err != nil {
			return fmt.Errorf("保存财务数据失败: %w", err)
		}
		log.Printf("成功保存股票 %s 的 %d 条财务数据", symbol, len(allFinancialData))
	}

	return nil
}

// CollectMacroData 采集宏观经济数据
func (s *TushareService) CollectMacroData(ctx context.Context, startDate, endDate string) error {
	log.Printf("开始采集 %s 到 %s 的宏观经济数据...", startDate, endDate)

	var allMacroData []*model.MacroData

	// 采集GDP数据
	gdpData, err := s.collector.GetMacroCollector().GetGDPData(ctx, startDate, endDate)
	if err != nil {
		log.Printf("获取GDP数据失败: %v", err)
	} else {
		allMacroData = append(allMacroData, gdpData...)
		log.Printf("从Tushare获取到 %d 条GDP数据", len(gdpData))
	}

	// 采集CPI数据
	cpiData, err := s.collector.GetMacroCollector().GetCPIData(ctx, startDate, endDate)
	if err != nil {
		log.Printf("获取CPI数据失败: %v", err)
	} else {
		allMacroData = append(allMacroData, cpiData...)
		log.Printf("从Tushare获取到 %d 条CPI数据", len(cpiData))
	}

	// 采集PPI数据
	ppiData, err := s.collector.GetMacroCollector().GetPPIData(ctx, startDate, endDate)
	if err != nil {
		log.Printf("获取PPI数据失败: %v", err)
	} else {
		allMacroData = append(allMacroData, ppiData...)
		log.Printf("从Tushare获取到 %d 条PPI数据", len(ppiData))
	}

	// 采集M2数据
	m2Data, err := s.collector.GetMacroCollector().GetM2Data(ctx, startDate, endDate)
	if err != nil {
		log.Printf("获取M2数据失败: %v", err)
	} else {
		allMacroData = append(allMacroData, m2Data...)
		log.Printf("从Tushare获取到 %d 条M2数据", len(m2Data))
	}

	// 采集PMI数据
	pmiData, err := s.collector.GetMacroCollector().GetPMIData(ctx, startDate, endDate)
	if err != nil {
		log.Printf("获取PMI数据失败: %v", err)
	} else {
		allMacroData = append(allMacroData, pmiData...)
		log.Printf("从Tushare获取到 %d 条PMI数据", len(pmiData))
	}

	// 采集Shibor数据
	shiborData, err := s.collector.GetMacroCollector().GetShiborData(ctx, startDate, endDate)
	if err != nil {
		log.Printf("获取Shibor数据失败: %v", err)
	} else {
		allMacroData = append(allMacroData, shiborData...)
		log.Printf("从Tushare获取到 %d 条Shibor数据", len(shiborData))
	}

	// 采集LPR数据
	lprData, err := s.collector.GetMacroCollector().GetLPRData(ctx, startDate, endDate)
	if err != nil {
		log.Printf("获取LPR数据失败: %v", err)
	} else {
		allMacroData = append(allMacroData, lprData...)
		log.Printf("从Tushare获取到 %d 条LPR数据", len(lprData))
	}

	// 采集汇率数据
	exchangeData, err := s.collector.GetMacroCollector().GetExchangeRateData(ctx, startDate, endDate)
	if err != nil {
		log.Printf("获取汇率数据失败: %v", err)
	} else {
		allMacroData = append(allMacroData, exchangeData...)
		log.Printf("从Tushare获取到 %d 条汇率数据", len(exchangeData))
	}

	// 批量保存宏观数据
	if len(allMacroData) > 0 {
		if err := s.macroRepo.BatchCreate(ctx, allMacroData); err != nil {
			return fmt.Errorf("保存宏观数据失败: %w", err)
		}
		log.Printf("成功保存 %d 条宏观经济数据", len(allMacroData))
	}

	return nil
}

// CollectRealtimeData 采集实时行情数据
func (s *TushareService) CollectRealtimeData(ctx context.Context, symbols []string) error {
	log.Printf("开始采集 %d 只股票的实时行情数据...", len(symbols))

	var allRealtimeData []*model.MarketData

	// 分批采集实时数据
	for _, symbol := range symbols {
		realtimeData, err := s.collector.GetMarketCollector().GetRealtimeData(ctx, []string{symbol})
		if err != nil {
			log.Printf("获取股票 %s 实时数据失败: %v", symbol, err)
			continue
		}

		allRealtimeData = append(allRealtimeData, realtimeData...)

		// 避免请求过于频繁
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("从Tushare获取到 %d 条实时行情数据", len(allRealtimeData))

	// 批量保存实时数据
	if len(allRealtimeData) > 0 {
		if err := s.marketRepo.BatchCreate(ctx, allRealtimeData); err != nil {
			return fmt.Errorf("保存实时行情数据失败: %w", err)
		}
		log.Printf("成功保存 %d 条实时行情数据", len(allRealtimeData))
	}

	return nil
}

// BatchCollectStockData 批量采集股票数据
func (s *TushareService) BatchCollectStockData(ctx context.Context, symbols []string, startDate, endDate string) error {
	log.Printf("开始批量采集 %d 只股票的数据...", len(symbols))

	for i, symbol := range symbols {
		log.Printf("正在采集第 %d/%d 只股票: %s", i+1, len(symbols), symbol)

		// 采集历史行情数据
		if err := s.CollectStockHistoryData(ctx, symbol, startDate, endDate); err != nil {
			log.Printf("采集股票 %s 历史数据失败: %v", symbol, err)
		}

		// 采集最新财务数据（最近一个报告期）
		currentYear := time.Now().Year()
		period := fmt.Sprintf("%d1231", currentYear-1) // 上一年年报
		if err := s.CollectFinancialData(ctx, symbol, period); err != nil {
			log.Printf("采集股票 %s 财务数据失败: %v", symbol, err)
		}

		// 避免请求过于频繁
		time.Sleep(200 * time.Millisecond)
	}

	log.Println("批量采集完成")
	return nil
}

// GetCollectorStatus 获取采集器状态
func (s *TushareService) GetCollectorStatus(ctx context.Context) (map[string]interface{}, error) {
	// 检查健康状态
	if err := s.collector.HealthCheck(ctx); err != nil {
		return map[string]interface{}{
			"status":  "unhealthy",
			"error":   err.Error(),
			"time":    time.Now(),
		}, nil
	}

	// 获取配额信息
	quota, err := s.collector.GetQuota(ctx)
	if err != nil {
		log.Printf("获取配额信息失败: %v", err)
		quota = map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"status":           "healthy",
		"time":             time.Now(),
		"quota":            quota,
		"supported_apis":   s.collector.GetSupportedAPIs(),
	}, nil
}

// SyncStockList 同步股票列表
func (s *TushareService) SyncStockList(ctx context.Context) error {
	log.Println("开始同步股票列表...")

	// 获取最新的股票列表
	stocks, err := s.collector.GetStockCollector().GetStockBasic(ctx, "L")
	if err != nil {
		return fmt.Errorf("获取股票列表失败: %w", err)
	}

	// 逐个检查和更新
	var newStocks []*model.Stock
	var updatedCount int

	for _, stock := range stocks {
		// 检查股票是否已存在
		existingStock, err := s.stockRepo.GetBySymbol(ctx, stock.Symbol)
		if err != nil {
			// 股票不存在，添加到新股票列表
			newStocks = append(newStocks, stock)
			continue
		}

		// 股票已存在，检查是否需要更新
		if s.needUpdateStock(existingStock, stock) {
			stock.ID = existingStock.ID
			stock.CreatedAt = existingStock.CreatedAt
			if err := s.stockRepo.Update(ctx, stock); err != nil {
				log.Printf("更新股票 %s 失败: %v", stock.Symbol, err)
			} else {
				updatedCount++
			}
		}
	}

	// 批量插入新股票
	if len(newStocks) > 0 {
		if err := s.stockRepo.BatchCreate(ctx, newStocks); err != nil {
			return fmt.Errorf("保存新股票失败: %w", err)
		}
	}

	log.Printf("股票列表同步完成: 新增 %d 只，更新 %d 只", len(newStocks), updatedCount)
	return nil
}

// needUpdateStock 检查股票信息是否需要更新
func (s *TushareService) needUpdateStock(existing, new *model.Stock) bool {
	return existing.Name != new.Name ||
		existing.Industry != new.Industry ||
		existing.Sector != new.Sector ||
		existing.Status != new.Status
}