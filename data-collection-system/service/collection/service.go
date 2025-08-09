package collection

import (
	"context"
	"fmt"
	"log"
	"time"

	"data-collection-system/model"
	"data-collection-system/repo/external/tushare"
)

// Service 数据采集业务服务
type Service struct {
	tushareService *TushareService
	// newsService    *NewsService    // 新闻采集服务（预留）
	// crawlerService *CrawlerService // 爬虫采集服务（预留）
}

// Config 采集服务配置
type Config struct {
	TushareToken string `yaml:"tushare_token"`
	TushareURL   string `yaml:"tushare_url"`
	RateLimit    int    `yaml:"rate_limit"`
	RetryCount   int    `yaml:"retry_count"`
	Timeout      int    `yaml:"timeout"`
}

// NewService 创建数据采集服务
func NewService(
	config *Config,
	stockRepo StockRepository,
	marketRepo MarketRepository,
	financialRepo FinancialRepository,
	macroRepo MacroRepository,
) (*Service, error) {
	// 创建Tushare采集器
	tushareCollector, err := tushare.NewCollector(&tushare.Config{
		Token:      config.TushareToken,
		BaseURL:    config.TushareURL,
		RateLimit:  config.RateLimit,
		RetryCount: config.RetryCount,
		Timeout:    time.Duration(config.Timeout) * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("创建Tushare采集器失败: %w", err)
	}

	// 创建Tushare业务服务
	tushareService := NewTushareService(
		tushareCollector,
		stockRepo,
		marketRepo,
		financialRepo,
		macroRepo,
	)

	return &Service{
		tushareService: tushareService,
	}, nil
}

// CollectStockBasicData 采集股票基础数据
func (s *Service) CollectStockBasicData(ctx context.Context) error {
	return s.tushareService.CollectStockBasicData(ctx)
}

// CollectDailyMarketData 采集日线行情数据
func (s *Service) CollectDailyMarketData(ctx context.Context, tradeDate string) error {
	return s.tushareService.CollectDailyMarketData(ctx, tradeDate)
}

// CollectStockHistoryData 采集指定股票的历史行情数据
func (s *Service) CollectStockHistoryData(ctx context.Context, symbol, startDate, endDate string) error {
	return s.tushareService.CollectStockHistoryData(ctx, symbol, startDate, endDate)
}

// CollectFinancialData 采集财务数据
func (s *Service) CollectFinancialData(ctx context.Context, symbol, period string) error {
	return s.tushareService.CollectFinancialData(ctx, symbol, period)
}

// CollectMacroData 采集宏观经济数据
func (s *Service) CollectMacroData(ctx context.Context, startDate, endDate string) error {
	return s.tushareService.CollectMacroData(ctx, startDate, endDate)
}

// CollectRealtimeData 采集实时行情数据
func (s *Service) CollectRealtimeData(ctx context.Context, symbols []string) error {
	return s.tushareService.CollectRealtimeData(ctx, symbols)
}

// BatchCollectStockData 批量采集股票数据
func (s *Service) BatchCollectStockData(ctx context.Context, symbols []string, startDate, endDate string) error {
	return s.tushareService.BatchCollectStockData(ctx, symbols, startDate, endDate)
}

// SyncStockList 同步股票列表
func (s *Service) SyncStockList(ctx context.Context) error {
	return s.tushareService.SyncStockList(ctx)
}

// GetCollectorStatus 获取采集器状态
func (s *Service) GetCollectorStatus(ctx context.Context) (map[string]interface{}, error) {
	return s.tushareService.GetCollectorStatus(ctx)
}

// CollectTodayData 采集今日数据（综合任务）
func (s *Service) CollectTodayData(ctx context.Context) error {
	log.Println("开始执行今日数据采集任务...")

	// 获取今日日期
	today := time.Now().Format("20060102")

	// 1. 采集今日行情数据
	if err := s.CollectDailyMarketData(ctx, today); err != nil {
		log.Printf("采集今日行情数据失败: %v", err)
		// 不返回错误，继续执行其他任务
	}

	// 2. 同步股票列表（每日更新）
	if err := s.SyncStockList(ctx); err != nil {
		log.Printf("同步股票列表失败: %v", err)
	}

	// 3. 采集宏观数据（每日更新）
	startDate := time.Now().AddDate(0, 0, -7).Format("20060102") // 最近7天
	endDate := today
	if err := s.CollectMacroData(ctx, startDate, endDate); err != nil {
		log.Printf("采集宏观数据失败: %v", err)
	}

	log.Println("今日数据采集任务完成")
	return nil
}

// CollectWeeklyData 采集周度数据（综合任务）
func (s *Service) CollectWeeklyData(ctx context.Context) error {
	log.Println("开始执行周度数据采集任务...")

	// 获取活跃股票列表（示例：获取前100只股票）
	stocks, err := s.getActiveStocks(ctx, 100)
	if err != nil {
		return fmt.Errorf("获取活跃股票列表失败: %w", err)
	}

	// 提取股票代码
	var symbols []string
	for _, stock := range stocks {
		symbols = append(symbols, stock.Symbol)
	}

	// 采集最近一周的历史数据
	startDate := time.Now().AddDate(0, 0, -7).Format("20060102")
	endDate := time.Now().Format("20060102")

	if err := s.BatchCollectStockData(ctx, symbols, startDate, endDate); err != nil {
		return fmt.Errorf("批量采集股票数据失败: %w", err)
	}

	log.Println("周度数据采集任务完成")
	return nil
}

// CollectMonthlyData 采集月度数据（综合任务）
func (s *Service) CollectMonthlyData(ctx context.Context) error {
	log.Println("开始执行月度数据采集任务...")

	// 获取所有股票列表
	stocks, err := s.getAllStocks(ctx)
	if err != nil {
		return fmt.Errorf("获取股票列表失败: %w", err)
	}

	// 采集财务数据（季报、年报）
	currentYear := time.Now().Year()
	periods := []string{
		fmt.Sprintf("%d0331", currentYear), // Q1
		fmt.Sprintf("%d0630", currentYear), // Q2
		fmt.Sprintf("%d0930", currentYear), // Q3
		fmt.Sprintf("%d1231", currentYear-1), // 上年年报
	}

	for _, stock := range stocks {
		for _, period := range periods {
			if err := s.CollectFinancialData(ctx, stock.Symbol, period); err != nil {
				log.Printf("采集股票 %s 期间 %s 财务数据失败: %v", stock.Symbol, period, err)
			}
			// 避免请求过于频繁
			time.Sleep(500 * time.Millisecond)
		}
	}

	log.Println("月度数据采集任务完成")
	return nil
}

// getActiveStocks 获取活跃股票列表（示例实现）
func (s *Service) getActiveStocks(ctx context.Context, limit int) ([]*model.Stock, error) {
	// 这里应该根据实际业务逻辑获取活跃股票
	// 例如：根据成交量、市值等指标筛选
	// 暂时返回空列表，实际实现时需要调用stockRepo
	log.Printf("获取前 %d 只活跃股票（示例实现）", limit)
	return []*model.Stock{}, nil
}

// getAllStocks 获取所有股票列表（示例实现）
func (s *Service) getAllStocks(ctx context.Context) ([]*model.Stock, error) {
	// 这里应该调用stockRepo获取所有股票
	// 暂时返回空列表，实际实现时需要调用stockRepo
	log.Println("获取所有股票列表（示例实现）")
	return []*model.Stock{}, nil
}

// ValidateCollectionTask 验证采集任务参数
func (s *Service) ValidateCollectionTask(taskType string, params map[string]interface{}) error {
	switch taskType {
	case "stock_basic":
		// 股票基础数据采集无需额外参数
		return nil

	case "daily_market":
		// 验证交易日期参数
		if tradeDate, ok := params["trade_date"].(string); !ok || tradeDate == "" {
			return fmt.Errorf("缺少或无效的交易日期参数")
		}
		return nil

	case "stock_history":
		// 验证股票代码和日期范围
		symbol, symbolOk := params["symbol"].(string)
		startDate, startOk := params["start_date"].(string)
		endDate, endOk := params["end_date"].(string)

		if !symbolOk || symbol == "" {
			return fmt.Errorf("缺少或无效的股票代码参数")
		}
		if !startOk || startDate == "" {
			return fmt.Errorf("缺少或无效的开始日期参数")
		}
		if !endOk || endDate == "" {
			return fmt.Errorf("缺少或无效的结束日期参数")
		}
		return nil

	case "financial":
		// 验证股票代码和报告期
		symbol, symbolOk := params["symbol"].(string)
		period, periodOk := params["period"].(string)

		if !symbolOk || symbol == "" {
			return fmt.Errorf("缺少或无效的股票代码参数")
		}
		if !periodOk || period == "" {
			return fmt.Errorf("缺少或无效的报告期参数")
		}
		return nil

	case "macro":
		// 验证日期范围
		startDate, startOk := params["start_date"].(string)
		endDate, endOk := params["end_date"].(string)

		if !startOk || startDate == "" {
			return fmt.Errorf("缺少或无效的开始日期参数")
		}
		if !endOk || endDate == "" {
			return fmt.Errorf("缺少或无效的结束日期参数")
		}
		return nil

	case "realtime":
		// 验证股票代码列表
		if symbols, ok := params["symbols"].([]string); !ok || len(symbols) == 0 {
			return fmt.Errorf("缺少或无效的股票代码列表参数")
		}
		return nil

	default:
		return fmt.Errorf("不支持的采集任务类型: %s", taskType)
	}
}

// GetSupportedTaskTypes 获取支持的任务类型
func (s *Service) GetSupportedTaskTypes() []string {
	return []string{
		"stock_basic",    // 股票基础数据
		"daily_market",   // 日线行情数据
		"stock_history",  // 历史行情数据
		"financial",      // 财务数据
		"macro",          // 宏观数据
		"realtime",       // 实时数据
		"today_data",     // 今日综合数据
		"weekly_data",    // 周度综合数据
		"monthly_data",   // 月度综合数据
	}
}

// ExecuteCollectionTask 执行采集任务
func (s *Service) ExecuteCollectionTask(ctx context.Context, taskType string, params map[string]interface{}) error {
	// 验证任务参数
	if err := s.ValidateCollectionTask(taskType, params); err != nil {
		return fmt.Errorf("任务参数验证失败: %w", err)
	}

	// 执行对应的采集任务
	switch taskType {
	case "stock_basic":
		return s.CollectStockBasicData(ctx)

	case "daily_market":
		tradeDate := params["trade_date"].(string)
		return s.CollectDailyMarketData(ctx, tradeDate)

	case "stock_history":
		symbol := params["symbol"].(string)
		startDate := params["start_date"].(string)
		endDate := params["end_date"].(string)
		return s.CollectStockHistoryData(ctx, symbol, startDate, endDate)

	case "financial":
		symbol := params["symbol"].(string)
		period := params["period"].(string)
		return s.CollectFinancialData(ctx, symbol, period)

	case "macro":
		startDate := params["start_date"].(string)
		endDate := params["end_date"].(string)
		return s.CollectMacroData(ctx, startDate, endDate)

	case "realtime":
		symbols := params["symbols"].([]string)
		return s.CollectRealtimeData(ctx, symbols)

	case "today_data":
		return s.CollectTodayData(ctx)

	case "weekly_data":
		return s.CollectWeeklyData(ctx)

	case "monthly_data":
		return s.CollectMonthlyData(ctx)

	default:
		return fmt.Errorf("不支持的采集任务类型: %s", taskType)
	}
}