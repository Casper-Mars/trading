package tushare

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Collector Tushare数据采集器主入口
type Collector struct {
	client     *Client
	stock      *StockCollector
	market     *MarketCollector
	financial  *FinancialCollector
	macro      *MacroCollector
	config     *Config
}

// NewCollector 创建新的Tushare数据采集器
func NewCollector(config *Config) (*Collector, error) {
	if config.Token == "" {
		return nil, fmt.Errorf("tushare token is required")
	}

	client := NewClient(config)

	return &Collector{
		client:    client,
		stock:     NewStockCollector(client),
		market:    NewMarketCollector(client),
		financial: NewFinancialCollector(client),
		macro:     NewMacroCollector(client),
		config:    config,
	}, nil
}

// GetClient 获取底层客户端
func (c *Collector) GetClient() *Client {
	return c.client
}

// GetStockCollector 获取股票数据采集器
func (c *Collector) GetStockCollector() *StockCollector {
	return c.stock
}

// GetMarketCollector 获取市场数据采集器
func (c *Collector) GetMarketCollector() *MarketCollector {
	return c.market
}

// GetFinancialCollector 获取财务数据采集器
func (c *Collector) GetFinancialCollector() *FinancialCollector {
	return c.financial
}

// GetMacroCollector 获取宏观数据采集器
func (c *Collector) GetMacroCollector() *MacroCollector {
	return c.macro
}

// HealthCheck 健康检查
func (c *Collector) HealthCheck(ctx context.Context) error {
	return c.client.IsHealthy(ctx)
}

// CollectAllStockBasic 采集所有股票基础信息
func (c *Collector) CollectAllStockBasic(ctx context.Context) error {
	log.Println("开始采集股票基础信息...")

	// 采集上市股票
	stocks, err := c.stock.GetStockBasic(ctx, "L")
	if err != nil {
		return fmt.Errorf("采集上市股票失败: %w", err)
	}

	log.Printf("成功采集 %d 只上市股票信息", len(stocks))

	// 这里可以添加数据存储逻辑
	// 例如: return c.stockRepo.BatchInsert(ctx, stocks)

	return nil
}

// CollectDailyMarketData 采集指定日期的日线行情数据
func (c *Collector) CollectDailyMarketData(ctx context.Context, tradeDate string) error {
	log.Printf("开始采集 %s 的日线行情数据...", tradeDate)

	// 检查是否为交易日
	isTradeDay, err := c.stock.IsTradeDay(ctx, tradeDate)
	if err != nil {
		return fmt.Errorf("检查交易日失败: %w", err)
	}

	if !isTradeDay {
		log.Printf("%s 不是交易日，跳过采集", tradeDate)
		return nil
	}

	// 采集当日所有股票行情
	marketData, err := c.market.GetDailyDataByDate(ctx, tradeDate)
	if err != nil {
		return fmt.Errorf("采集日线数据失败: %w", err)
	}

	log.Printf("成功采集 %d 条日线行情数据", len(marketData))

	// 这里可以添加数据存储逻辑
	// 例如: return c.marketRepo.BatchInsert(ctx, marketData)

	return nil
}

// CollectStockFinancialData 采集指定股票的财务数据
func (c *Collector) CollectStockFinancialData(ctx context.Context, symbol, period string) error {
	log.Printf("开始采集股票 %s 期间 %s 的财务数据...", symbol, period)

	// 采集利润表
	incomeData, err := c.financial.GetIncomeStatement(ctx, symbol, period, "", "")
	if err != nil {
		log.Printf("采集利润表失败: %v", err)
	} else {
		log.Printf("成功采集 %d 条利润表数据", len(incomeData))
	}

	// 采集资产负债表
	balanceData, err := c.financial.GetBalanceSheet(ctx, symbol, period, "", "")
	if err != nil {
		log.Printf("采集资产负债表失败: %v", err)
	} else {
		log.Printf("成功采集 %d 条资产负债表数据", len(balanceData))
	}

	// 采集现金流量表
	cashFlowData, err := c.financial.GetCashFlow(ctx, symbol, period, "", "")
	if err != nil {
		log.Printf("采集现金流量表失败: %v", err)
	} else {
		log.Printf("成功采集 %d 条现金流量表数据", len(cashFlowData))
	}

	// 这里可以添加数据存储逻辑

	return nil
}

// CollectMacroData 采集宏观经济数据
func (c *Collector) CollectMacroData(ctx context.Context, startDate, endDate string) error {
	log.Printf("开始采集 %s 到 %s 的宏观经济数据...", startDate, endDate)

	// 采集GDP数据
	gdpData, err := c.macro.GetGDPData(ctx, startDate, endDate)
	if err != nil {
		log.Printf("采集GDP数据失败: %v", err)
	} else {
		log.Printf("成功采集 %d 条GDP数据", len(gdpData))
	}

	// 采集CPI数据
	cpiData, err := c.macro.GetCPIData(ctx, startDate, endDate)
	if err != nil {
		log.Printf("采集CPI数据失败: %v", err)
	} else {
		log.Printf("成功采集 %d 条CPI数据", len(cpiData))
	}

	// 采集PPI数据
	ppiData, err := c.macro.GetPPIData(ctx, startDate, endDate)
	if err != nil {
		log.Printf("采集PPI数据失败: %v", err)
	} else {
		log.Printf("成功采集 %d 条PPI数据", len(ppiData))
	}

	// 采集M2数据
	m2Data, err := c.macro.GetM2Data(ctx, startDate, endDate)
	if err != nil {
		log.Printf("采集M2数据失败: %v", err)
	} else {
		log.Printf("成功采集 %d 条M2数据", len(m2Data))
	}

	// 采集PMI数据
	pmiData, err := c.macro.GetPMIData(ctx, startDate, endDate)
	if err != nil {
		log.Printf("采集PMI数据失败: %v", err)
	} else {
		log.Printf("成功采集 %d 条PMI数据", len(pmiData))
	}

	// 这里可以添加数据存储逻辑

	return nil
}

// CollectRealtimeData 采集实时行情数据
func (c *Collector) CollectRealtimeData(ctx context.Context, symbols []string) error {
	log.Printf("开始采集 %d 只股票的实时行情数据...", len(symbols))

	// 分批采集，避免单次请求过多
	batchSize := 50
	for i := 0; i < len(symbols); i += batchSize {
		end := i + batchSize
		if end > len(symbols) {
			end = len(symbols)
		}

		batch := symbols[i:end]
		for _, symbol := range batch {
			realtimeData, err := c.market.GetRealtimeData(ctx, []string{symbol})
			if err != nil {
				log.Printf("采集股票 %s 实时数据失败: %v", symbol, err)
				continue
			}

			if len(realtimeData) > 0 {
				log.Printf("成功采集股票 %s 实时数据", symbol)
				// 这里可以添加数据存储逻辑
			}

			// 避免请求过于频繁
			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil
}

// CollectHistoricalData 采集历史数据（股票基础信息+行情数据）
func (c *Collector) CollectHistoricalData(ctx context.Context, symbol, startDate, endDate string) error {
	log.Printf("开始采集股票 %s 从 %s 到 %s 的历史数据...", symbol, startDate, endDate)

	// 采集股票基础信息
	stock, err := c.stock.GetStockBySymbol(ctx, symbol)
	if err != nil {
		return fmt.Errorf("获取股票基础信息失败: %w", err)
	}
	log.Printf("成功获取股票 %s (%s) 基础信息", stock.Symbol, stock.Name)

	// 采集日线行情数据
	marketData, err := c.market.GetDailyData(ctx, symbol, startDate, endDate)
	if err != nil {
		return fmt.Errorf("采集日线数据失败: %w", err)
	}
	log.Printf("成功采集 %d 条日线行情数据", len(marketData))

	// 这里可以添加数据存储逻辑

	return nil
}

// GetQuota 获取API配额信息
func (c *Collector) GetQuota(ctx context.Context) (map[string]interface{}, error) {
	data, err := c.client.Call(ctx, "user", nil, "")
	if err != nil {
		return nil, fmt.Errorf("获取配额信息失败: %w", err)
	}

	if data == nil || len(data.Items) == 0 {
		return nil, fmt.Errorf("配额信息为空")
	}

	// 创建字段索引映射
	fieldMap := make(map[string]int)
	for i, field := range data.Fields {
		fieldMap[field] = i
	}

	result := make(map[string]interface{})
	item := data.Items[0]
	for field, idx := range fieldMap {
		if idx < len(item) {
			result[field] = item[idx]
		}
	}

	return result, nil
}

// Close 关闭采集器
func (c *Collector) Close() error {
	// 这里可以添加清理逻辑
	log.Println("Tushare数据采集器已关闭")
	return nil
}

// BatchCollectStocks 批量采集股票数据
func (c *Collector) BatchCollectStocks(ctx context.Context, symbols []string, startDate, endDate string) error {
	log.Printf("开始批量采集 %d 只股票的数据...", len(symbols))

	for i, symbol := range symbols {
		log.Printf("正在采集第 %d/%d 只股票: %s", i+1, len(symbols), symbol)

		if err := c.CollectHistoricalData(ctx, symbol, startDate, endDate); err != nil {
			log.Printf("采集股票 %s 数据失败: %v", symbol, err)
			continue
		}

		// 避免请求过于频繁
		time.Sleep(200 * time.Millisecond)
	}

	log.Println("批量采集完成")
	return nil
}

// GetSupportedAPIs 获取支持的API列表
func (c *Collector) GetSupportedAPIs() []string {
	return []string{
		// 股票基础数据
		"stock_basic",    // 股票列表
		"stock_company",  // 上市公司基本信息
		"trade_cal",      // 交易日历
		
		// 行情数据
		"daily",          // 日线行情
		"stk_mins",       // 分钟行情
		"index_daily",    // 指数日线行情
		"moneyflow",      // 资金流向
		
		// 财务数据
		"income",         // 利润表
		"balancesheet",   // 资产负债表
		"cashflow",       // 现金流量表
		"fina_indicator", // 财务指标
		"dividend",       // 分红送股
		
		// 宏观数据
		"cn_gdp",         // GDP数据
		"cn_cpi",         // CPI数据
		"cn_ppi",         // PPI数据
		"cn_m",           // 货币供应量
		"cn_pmi",         // PMI数据
		"shibor",         // Shibor利率
		"cn_lpr",         // LPR利率
		"cn_fx",          // 汇率数据
	}
}