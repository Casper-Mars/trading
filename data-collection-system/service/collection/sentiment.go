package collection

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"data-collection-system/model"
	"data-collection-system/repo/external/tushare"
	"data-collection-system/repo/mysql"
)

// SentimentService 市场情绪和资金流向数据采集服务
type SentimentService struct {
	collector     *tushare.Collector
	sentimentRepo dao.SentimentRepository
}



// NewSentimentService 创建市场情绪和资金流向数据采集服务
func NewSentimentService(
	collector *tushare.Collector,
	sentimentRepo dao.SentimentRepository,
) *SentimentService {
	return &SentimentService{
		collector:     collector,
		sentimentRepo: sentimentRepo,
	}
}

// CollectMoneyFlowData 采集个股资金流向数据
func (s *SentimentService) CollectMoneyFlowData(ctx context.Context, symbol, tradeDate string) error {
	log.Printf("开始采集股票 %s 在 %s 的资金流向数据...", symbol, tradeDate)

	// 从Tushare获取资金流向数据
	rawData, err := s.collector.GetMarketCollector().GetMoneyFlow(ctx, symbol, tradeDate, tradeDate)
	if err != nil {
		return fmt.Errorf("获取资金流向数据失败: %w", err)
	}

	if len(rawData) == 0 {
		log.Printf("股票 %s 在 %s 没有资金流向数据", symbol, tradeDate)
		return nil
	}

	// 转换数据格式
	moneyFlowData := s.convertToMoneyFlowData(rawData)

	// 保存到数据库
	// 转换为[]interface{}类型
	moneyFlowInterfaces := make([]interface{}, len(moneyFlowData))
	for i, data := range moneyFlowData {
		moneyFlowInterfaces[i] = data
	}
	if err := s.sentimentRepo.BatchCreateMoneyFlow(ctx, moneyFlowInterfaces); err != nil {
		return fmt.Errorf("保存资金流向数据失败: %w", err)
	}

	log.Printf("成功采集并保存 %d 条资金流向数据", len(moneyFlowData))
	return nil
}

// 数据转换函数

// convertToMoneyFlowData 转换资金流向数据
func (s *SentimentService) convertToMoneyFlowData(rawData []map[string]interface{}) []*model.MoneyFlowData {
	result := make([]*model.MoneyFlowData, 0, len(rawData))
	for _, item := range rawData {
		data := &model.MoneyFlowData{}
		
		// 基础字段
		if v, ok := item["ts_code"].(string); ok {
			data.Symbol = v
		}
		if v, ok := item["trade_date"].(string); ok {
			if tradeDate, err := time.Parse("20060102", v); err == nil {
				data.TradeDate = tradeDate
			}
		}
		
		// 小单数据
		data.BuySmallVol = int64(s.parseFloat64(item["buy_sm_vol"]))
		data.BuySmallAmount = s.parseFloat64(item["buy_sm_amount"])
		data.SellSmallVol = int64(s.parseFloat64(item["sell_sm_vol"]))
		data.SellSmallAmount = s.parseFloat64(item["sell_sm_amount"])
		
		// 中单数据
		data.BuyMediumVol = int64(s.parseFloat64(item["buy_md_vol"]))
		data.BuyMediumAmount = s.parseFloat64(item["buy_md_amount"])
		data.SellMediumVol = int64(s.parseFloat64(item["sell_md_vol"]))
		data.SellMediumAmount = s.parseFloat64(item["sell_md_amount"])
		
		// 大单数据
		data.BuyLargeVol = int64(s.parseFloat64(item["buy_lg_vol"]))
		data.BuyLargeAmount = s.parseFloat64(item["buy_lg_amount"])
		data.SellLargeVol = int64(s.parseFloat64(item["sell_lg_vol"]))
		data.SellLargeAmount = s.parseFloat64(item["sell_lg_amount"])
		
		// 特大单数据
		data.BuyExtraLargeVol = int64(s.parseFloat64(item["buy_elg_vol"]))
		data.BuyExtraLargeAmount = s.parseFloat64(item["buy_elg_amount"])
		data.SellExtraLargeVol = int64(s.parseFloat64(item["sell_elg_vol"]))
		data.SellExtraLargeAmount = s.parseFloat64(item["sell_elg_amount"])
		
		// 净流入数据
		data.NetFlowVol = int64(s.parseFloat64(item["net_mf_vol"]))
		data.NetFlowAmount = s.parseFloat64(item["net_mf_amount"])
		
		data.CreatedAt = time.Now()
		data.UpdatedAt = time.Now()
		
		result = append(result, data)
	}
	return result
}

// convertToNorthboundFundData 转换北向资金数据
func (s *SentimentService) convertToNorthboundFundData(rawData []map[string]interface{}) []*model.NorthboundFundData {
	result := make([]*model.NorthboundFundData, 0, len(rawData))
	for _, item := range rawData {
		data := &model.NorthboundFundData{}
		
		// 解析交易日期
		if tradeDateStr, ok := item["trade_date"].(string); ok {
			if tradeDate, err := time.Parse("20060102", tradeDateStr); err == nil {
				data.TradeDate = tradeDate
			}
		}
		
		// 市场类型
		data.MarketType = int(s.parseFloat64(item["market_type"]))
		
		// 沪股通数据
		data.HSGTBuyAmount = s.parseFloat64(item["hsgt_buy_amount"])
		data.HSGTSellAmount = s.parseFloat64(item["hsgt_sell_amount"])
		data.HSGTNetAmount = s.parseFloat64(item["hsgt_net_amount"])
		
		// 深股通数据
		data.SZGTBuyAmount = s.parseFloat64(item["szgt_buy_amount"])
		data.SZGTSellAmount = s.parseFloat64(item["szgt_sell_amount"])
		data.SZGTNetAmount = s.parseFloat64(item["szgt_net_amount"])
		
		// 港股通数据
		data.HKGTBuyAmount = s.parseFloat64(item["hkgt_buy_amount"])
		data.HKGTSellAmount = s.parseFloat64(item["hkgt_sell_amount"])
		data.HKGTNetAmount = s.parseFloat64(item["hkgt_net_amount"])
		
		// 总计数据
		data.TotalBuyAmount = s.parseFloat64(item["total_buy_amount"])
		data.TotalSellAmount = s.parseFloat64(item["total_sell_amount"])
		data.TotalNetAmount = s.parseFloat64(item["total_net_amount"])
		
		data.CreatedAt = time.Now()
		data.UpdatedAt = time.Now()
		
		result = append(result, data)
	}
	return result
}

// convertToNorthboundTopStockData 转换北向资金十大成交股数据
func (s *SentimentService) convertToNorthboundTopStockData(rawData []map[string]interface{}) []*model.NorthboundTopStockData {
	result := make([]*model.NorthboundTopStockData, 0, len(rawData))
	for _, item := range rawData {
		data := &model.NorthboundTopStockData{}
		
		// 解析交易日期
		if tradeDateStr, ok := item["trade_date"].(string); ok {
			if tradeDate, err := time.Parse("20060102", tradeDateStr); err == nil {
				data.TradeDate = tradeDate
			}
		}
		
		// 股票信息
		if symbol, ok := item["ts_code"].(string); ok {
			data.Symbol = symbol
		}
		if name, ok := item["name"].(string); ok {
			data.Name = name
		}
		
		// 价格和排名信息
		data.ClosePrice = s.parseFloat64(item["close"])
		data.Change = s.parseFloat64(item["change"])
		data.Rank = int(s.parseFloat64(item["rank"]))
		data.MarketType = int(s.parseFloat64(item["market_type"]))
		
		// 成交金额信息
		data.Amount = s.parseFloat64(item["amount"])
		data.NetAmount = s.parseFloat64(item["net_amount"])
		data.BuyAmount = s.parseFloat64(item["buy_amount"])
		data.SellAmount = s.parseFloat64(item["sell_amount"])
		
		data.CreatedAt = time.Now()
		data.UpdatedAt = time.Now()
		
		result = append(result, data)
	}
	return result
}

// convertToMarginTradingData 转换融资融券数据
func (s *SentimentService) convertToMarginTradingData(rawData []map[string]interface{}) []*model.MarginTradingData {
	result := make([]*model.MarginTradingData, 0, len(rawData))
	for _, item := range rawData {
		data := &model.MarginTradingData{}
		
		// 解析交易日期
		if tradeDateStr, ok := item["trade_date"].(string); ok {
			if tradeDate, err := time.Parse("20060102", tradeDateStr); err == nil {
				data.TradeDate = tradeDate
			}
		}
		
		// 交易所信息
		if exchangeID, ok := item["exchange_id"].(string); ok {
			data.ExchangeID = exchangeID
		}
		
		// 融资数据
		data.FinancingBalance = s.parseFloat64(item["rzye"])
		data.FinancingBuy = s.parseFloat64(item["rzmre"])
		data.FinancingRepay = s.parseFloat64(item["rzche"])
		
		// 融券数据
		data.SecuritiesBalance = s.parseFloat64(item["rqye"])
		data.SecuritiesSell = s.parseFloat64(item["rqmcl"])
		data.SecuritiesRepay = s.parseFloat64(item["rqchl"])
		
		// 总计数据
		data.TotalBalance = s.parseFloat64(item["rzrqye"])
		
		data.CreatedAt = time.Now()
		data.UpdatedAt = time.Now()
		
		result = append(result, data)
	}
	return result
}

// convertToETFData 转换ETF数据
func (s *SentimentService) convertToETFData(rawData []map[string]interface{}) []*model.ETFData {
	result := make([]*model.ETFData, 0, len(rawData))
	for _, item := range rawData {
		data := &model.ETFData{}
		
		// 基本信息
		if symbol, ok := item["ts_code"].(string); ok {
			data.Symbol = symbol
		}
		if shortName, ok := item["name"].(string); ok {
			data.ShortName = shortName
		}
		if extName, ok := item["ext_name"].(string); ok {
			data.ExtName = extName
		}
		if fullName, ok := item["full_name"].(string); ok {
			data.FullName = fullName
		}
		
		// 指数信息
		if indexCode, ok := item["index_code"].(string); ok {
			data.IndexCode = indexCode
		}
		if indexName, ok := item["index_name"].(string); ok {
			data.IndexName = indexName
		}
		
		// 日期信息
		if setupDateStr, ok := item["setup_date"].(string); ok {
			if setupDate, err := time.Parse("20060102", setupDateStr); err == nil {
				data.SetupDate = setupDate
			}
		}
		if listDateStr, ok := item["list_date"].(string); ok {
			if listDate, err := time.Parse("20060102", listDateStr); err == nil {
				data.ListDate = listDate
			}
		}
		
		// 状态和交易所信息
		if listStatus, ok := item["list_status"].(string); ok {
			data.ListStatus = listStatus
		}
		if exchange, ok := item["exchange"].(string); ok {
			data.Exchange = exchange
		}
		
		// 管理信息
		if managerName, ok := item["manager"].(string); ok {
			data.ManagerName = managerName
		}
		if custodName, ok := item["custodian"].(string); ok {
			data.CustodName = custodName
		}
		
		// 费用和类型信息
		data.MgtFee = s.parseFloat64(item["mgt_fee"])
		if etfType, ok := item["fund_type"].(string); ok {
			data.ETFType = etfType
		}
		
		data.CreatedAt = time.Now()
		data.UpdatedAt = time.Now()
		
		result = append(result, data)
	}
	return result
}

// parseFloat64 解析浮点数
func (s *SentimentService) parseFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0.0
}

// CollectNorthboundFundData 采集北向资金数据
func (s *SentimentService) CollectNorthboundFundData(ctx context.Context, tradeDate string) error {
	log.Printf("开始采集 %s 的北向资金数据...", tradeDate)

	// 从Tushare获取沪深港通资金流向数据
	rawData, err := s.collector.GetMarketCollector().GetNorthboundFundFlow(ctx, tradeDate, tradeDate)
	if err != nil {
		return fmt.Errorf("获取北向资金数据失败: %w", err)
	}

	if len(rawData) == 0 {
		log.Printf("%s 没有北向资金数据", tradeDate)
		return nil
	}

	// 转换数据格式
	northboundData := s.convertToNorthboundFundData(rawData)

	// 保存到数据库
	if err := s.sentimentRepo.BatchCreateNorthboundFund(ctx, northboundData); err != nil {
		return fmt.Errorf("保存北向资金数据失败: %w", err)
	}

	log.Printf("成功采集并保存 %d 条北向资金数据", len(northboundData))
	return nil
}

// CollectNorthboundTopStocksData 采集北向资金十大成交股数据
func (s *SentimentService) CollectNorthboundTopStocksData(ctx context.Context, tradeDate, market string) error {
	log.Printf("开始采集 %s 市场在 %s 的北向资金十大成交股数据...", market, tradeDate)

	// 从Tushare获取沪深股通十大成交股数据
	rawData, err := s.collector.GetMarketCollector().GetNorthboundTopStocks(ctx, tradeDate, market)
	if err != nil {
		return fmt.Errorf("获取北向资金十大成交股数据失败: %w", err)
	}

	if len(rawData) == 0 {
		log.Printf("%s 市场在 %s 没有北向资金十大成交股数据", market, tradeDate)
		return nil
	}

	// 转换数据格式
	topStocksData := s.convertToNorthboundTopStockData(rawData)

	// 保存到数据库
	if err := s.sentimentRepo.BatchCreateNorthboundTopStock(ctx, topStocksData); err != nil {
		return fmt.Errorf("保存北向资金十大成交股数据失败: %w", err)
	}

	log.Printf("成功采集并保存 %d 条北向资金十大成交股数据", len(topStocksData))
	return nil
}

// CollectMarginTradingData 采集融资融券数据
func (s *SentimentService) CollectMarginTradingData(ctx context.Context, tradeDate, exchange string) error {
	log.Printf("开始采集 %s 交易所在 %s 的融资融券数据...", exchange, tradeDate)

	// 从Tushare获取融资融券数据
	rawData, err := s.collector.GetMarketCollector().GetMarginTradingData(ctx, tradeDate, exchange)
	if err != nil {
		return fmt.Errorf("获取融资融券数据失败: %w", err)
	}

	if len(rawData) == 0 {
		log.Printf("%s 交易所在 %s 没有融资融券数据", exchange, tradeDate)
		return nil
	}

	// 转换数据格式
	marginData := s.convertToMarginTradingData(rawData)

	// 转换为[]interface{}类型
		data := make([]interface{}, len(marginData))
		for i, item := range marginData {
			data[i] = item
		}
		// 批量保存到数据库
		if err := s.sentimentRepo.BatchCreateMarginTrading(ctx, data); err != nil {
			return fmt.Errorf("保存融资融券数据失败: %v", err)
		}

	log.Printf("成功采集并保存 %d 条融资融券数据", len(marginData))
	return nil
}

// CollectETFBasicData 采集ETF基础数据
func (s *SentimentService) CollectETFBasicData(ctx context.Context, market string) error {
	log.Printf("开始采集 %s 市场的ETF基础数据...", market)

	// 从Tushare获取ETF基础数据
	rawData, err := s.collector.GetMarketCollector().GetETFBasicData(ctx, "L", market)
	if err != nil {
		return fmt.Errorf("获取ETF基础数据失败: %w", err)
	}

	if len(rawData) == 0 {
		log.Printf("%s 市场没有ETF基础数据", market)
		return nil
	}

	// 转换数据格式
	etfData := s.convertToETFData(rawData)

	// 保存到数据库
	if err := s.sentimentRepo.BatchCreateETF(ctx, etfData); err != nil {
		return fmt.Errorf("保存ETF基础数据失败: %w", err)
	}

	log.Printf("成功采集并保存 %d 条ETF基础数据", len(etfData))
	return nil
}

// CollectAllSentimentData 采集所有市场情绪和资金流向数据
func (s *SentimentService) CollectAllSentimentData(ctx context.Context, tradeDate string) error {
	log.Printf("开始采集 %s 的所有市场情绪和资金流向数据...", tradeDate)

	// 1. 采集北向资金数据
	if err := s.CollectNorthboundFundData(ctx, tradeDate); err != nil {
		log.Printf("采集北向资金数据失败: %v", err)
		// 不中断，继续采集其他数据
	}

	// 2. 采集沪股通十大成交股数据
	if err := s.CollectNorthboundTopStocksData(ctx, tradeDate, "SH"); err != nil {
		log.Printf("采集沪股通十大成交股数据失败: %v", err)
	}

	// 3. 采集深股通十大成交股数据
	if err := s.CollectNorthboundTopStocksData(ctx, tradeDate, "SZ"); err != nil {
		log.Printf("采集深股通十大成交股数据失败: %v", err)
	}

	// 4. 采集上交所融资融券数据
	if err := s.CollectMarginTradingData(ctx, tradeDate, "SSE"); err != nil {
		log.Printf("采集上交所融资融券数据失败: %v", err)
	}

	// 5. 采集深交所融资融券数据
	if err := s.CollectMarginTradingData(ctx, tradeDate, "SZSE"); err != nil {
		log.Printf("采集深交所融资融券数据失败: %v", err)
	}

	log.Printf("完成 %s 的市场情绪和资金流向数据采集", tradeDate)
	return nil
}

// CollectActiveStocksMoneyFlow 采集活跃股票的资金流向数据
func (s *SentimentService) CollectActiveStocksMoneyFlow(ctx context.Context, tradeDate string, symbols []string) error {
	log.Printf("开始采集 %d 只活跃股票在 %s 的资金流向数据...", len(symbols), tradeDate)

	successCount := 0
	failCount := 0

	for _, symbol := range symbols {
		if err := s.CollectMoneyFlowData(ctx, symbol, tradeDate); err != nil {
			log.Printf("采集股票 %s 资金流向数据失败: %v", symbol, err)
			failCount++
			continue
		}
		successCount++

		// 添加延时，避免频率限制
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("活跃股票资金流向数据采集完成: 成功 %d 只，失败 %d 只", successCount, failCount)
	return nil
}

// GetSentimentCollectorStatus 获取市场情绪数据采集器状态
func (s *SentimentService) GetSentimentCollectorStatus(ctx context.Context) (map[string]interface{}, error) {
	status := map[string]interface{}{
		"service_name":    "SentimentService",
		"status":          "running",
		"last_check_time": time.Now().Format("2006-01-02 15:04:05"),
		"supported_data_types": []string{
			"money_flow",        // 个股资金流向
			"northbound_fund",   // 北向资金
			"northbound_top",    // 北向资金十大成交股
			"margin_trading",    // 融资融券
			"etf_basic",         // ETF基础数据
		},
		"data_sources": []string{
			"tushare_moneyflow",      // Tushare个股资金流向
			"tushare_moneyflow_hsgt", // Tushare沪深港通资金流向
			"tushare_hsgt_top10",     // Tushare沪深股通十大成交股
			"tushare_margin",         // Tushare融资融券
			"tushare_etf_basic",      // Tushare ETF基础信息
		},
	}

	return status, nil
}

// ValidateSentimentCollectionTask 验证市场情绪数据采集任务参数
func (s *SentimentService) ValidateSentimentCollectionTask(taskType string, params map[string]interface{}) error {
	switch taskType {
	case "money_flow":
		if _, ok := params["symbol"]; !ok {
			return fmt.Errorf("money_flow任务缺少symbol参数")
		}
		if _, ok := params["trade_date"]; !ok {
			return fmt.Errorf("money_flow任务缺少trade_date参数")
		}
	case "northbound_fund":
		if _, ok := params["trade_date"]; !ok {
			return fmt.Errorf("northbound_fund任务缺少trade_date参数")
		}
	case "northbound_top":
		if _, ok := params["trade_date"]; !ok {
			return fmt.Errorf("northbound_top任务缺少trade_date参数")
		}
		if _, ok := params["market"]; !ok {
			return fmt.Errorf("northbound_top任务缺少market参数")
		}
	case "margin_trading":
		if _, ok := params["trade_date"]; !ok {
			return fmt.Errorf("margin_trading任务缺少trade_date参数")
		}
		if _, ok := params["exchange"]; !ok {
			return fmt.Errorf("margin_trading任务缺少exchange参数")
		}
	case "etf_basic":
		if _, ok := params["market"]; !ok {
			return fmt.Errorf("etf_basic任务缺少market参数")
		}
	case "all_sentiment":
		if _, ok := params["trade_date"]; !ok {
			return fmt.Errorf("all_sentiment任务缺少trade_date参数")
		}
	default:
		return fmt.Errorf("不支持的市场情绪数据采集任务类型: %s", taskType)
	}

	return nil
}

// ExecuteSentimentCollectionTask 执行市场情绪数据采集任务
func (s *SentimentService) ExecuteSentimentCollectionTask(ctx context.Context, taskType string, params map[string]interface{}) error {
	// 验证任务参数
	if err := s.ValidateSentimentCollectionTask(taskType, params); err != nil {
		return err
	}

	switch taskType {
	case "money_flow":
		symbol := params["symbol"].(string)
		tradeDate := params["trade_date"].(string)
		return s.CollectMoneyFlowData(ctx, symbol, tradeDate)

	case "northbound_fund":
		tradeDate := params["trade_date"].(string)
		return s.CollectNorthboundFundData(ctx, tradeDate)

	case "northbound_top":
		tradeDate := params["trade_date"].(string)
		market := params["market"].(string)
		return s.CollectNorthboundTopStocksData(ctx, tradeDate, market)

	case "margin_trading":
		tradeDate := params["trade_date"].(string)
		exchange := params["exchange"].(string)
		return s.CollectMarginTradingData(ctx, tradeDate, exchange)

	case "etf_basic":
		market := params["market"].(string)
		return s.CollectETFBasicData(ctx, market)

	case "all_sentiment":
		tradeDate := params["trade_date"].(string)
		return s.CollectAllSentimentData(ctx, tradeDate)

	default:
		return fmt.Errorf("不支持的市场情绪数据采集任务类型: %s", taskType)
	}
}