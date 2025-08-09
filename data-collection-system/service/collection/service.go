package collection

import (
	"context"
	"fmt"
	"log"
	"time"

	"data-collection-system/model"
	"data-collection-system/pkg/config"
	dao "data-collection-system/repo/mysql"
	"data-collection-system/repo/external/tushare"
)

// Service 数据采集业务服务
type Service struct {
	tushareService   *TushareService
	newsCrawler      *NewsCrawlerService // 新闻爬虫服务
	newsScheduler    *NewsScheduler      // 新闻调度器
	sentimentService *SentimentService   // 市场情绪和资金流向数据采集服务
}

// Config 采集服务配置
type Config struct {
	TushareToken string             `yaml:"tushare_token"`
	TushareURL   string             `yaml:"tushare_url"`
	RateLimit    int                `yaml:"rate_limit"`
	RetryCount   int                `yaml:"retry_count"`
	Timeout      int                `yaml:"timeout"`
	NewsCrawler  *NewsCrawlerConfig `yaml:"news_crawler"` // 新闻爬虫配置
}

// NewService 创建数据采集服务
func NewService(
	config *config.Config,
	repoManager dao.RepositoryManager,
) (*Service, error) {
	// 从配置中获取Tushare配置
	tushareConfig := &Config{
		TushareToken: config.Tushare.Token,
		TushareURL:   config.Tushare.BaseURL,
		RateLimit:    200, // 默认限流200次/分钟
		RetryCount:   3,   // 默认重试3次
		Timeout:      30,  // 默认超时30秒
		NewsCrawler:  nil, // 暂时不启用新闻爬虫
	}
	// 创建Tushare采集器
	tushareCollector, err := tushare.NewCollector(&tushare.Config{
		Token:      tushareConfig.TushareToken,
		BaseURL:    tushareConfig.TushareURL,
		RateLimit:  tushareConfig.RateLimit,
		RetryCount: tushareConfig.RetryCount,
		Timeout:    time.Duration(tushareConfig.Timeout) * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("创建Tushare采集器失败: %w", err)
	}

	// 直接使用RepositoryManager接口
	repoMgr := repoManager

	// 创建Tushare业务服务
	tushareService := NewTushareService(
		tushareCollector,
		repoMgr.Stock(),        // stockRepo
		repoMgr.MarketData(),   // marketRepo
		repoMgr.FinancialData(), // financialRepo
		repoMgr.MacroData(),    // macroRepo
	)

	// 创建市场情绪和资金流向数据采集服务
	sentimentService := NewSentimentService(
		tushareCollector,
		repoMgr.Sentiment(), // sentimentRepo - 需要在RepositoryManager中添加
	)

	// 创建新闻爬虫服务
	var newsCrawler *NewsCrawlerService
	var newsScheduler *NewsScheduler
	if tushareConfig.NewsCrawler != nil {
		newsCrawler = NewNewsCrawlerService(tushareConfig.NewsCrawler, repoMgr.News()) // newsRepo

		// 创建新闻调度器
		newsSources := GetDefaultNewsSources()                          // 获取默认新闻源配置
		newsScheduler = NewNewsScheduler(newsCrawler, newsSources, nil) // logger可以后续注入
	}

	return &Service{
		tushareService:   tushareService,
		newsCrawler:      newsCrawler,
		newsScheduler:    newsScheduler,
		sentimentService: sentimentService,
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
		fmt.Sprintf("%d0331", currentYear),   // Q1
		fmt.Sprintf("%d0630", currentYear),   // Q2
		fmt.Sprintf("%d0930", currentYear),   // Q3
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

	case "news_crawl":
		if _, ok := params["source_name"]; !ok {
			return fmt.Errorf("news_crawl任务缺少source_name参数")
		}
		return nil

	case "news_crawl_all":
		// 无需额外参数
		return nil

	// 市场情绪和资金流向数据采集任务验证
	case "money_flow", "northbound_fund", "northbound_top", "margin_trading", "etf_basic", "all_sentiment":
		return s.sentimentService.ValidateSentimentCollectionTask(taskType, params)

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
		"news_crawl",     // 新闻爬取
		"news_crawl_all", // 全量新闻爬取
		"money_flow",     // 个股资金流向数据采集
		"northbound_fund", // 北向资金数据采集
		"northbound_top",  // 北向资金十大成交股数据采集
		"margin_trading",  // 融资融券数据采集
		"etf_basic",       // ETF基础数据采集
		"all_sentiment",   // 所有市场情绪和资金流向数据采集
	}
}

// ==================== 新闻爬虫相关方法 ====================

// StartNewsScheduler 启动新闻调度器
func (s *Service) StartNewsScheduler(ctx context.Context) error {
	if s.newsScheduler == nil {
		return fmt.Errorf("新闻调度器未初始化")
	}
	return s.newsScheduler.Start(ctx)
}

// StopNewsScheduler 停止新闻调度器
func (s *Service) StopNewsScheduler() {
	if s.newsScheduler != nil {
		s.newsScheduler.Stop()
	}
}

// CrawlNews 爬取指定新闻源
func (s *Service) CrawlNews(ctx context.Context, sourceName string) error {
	if s.newsCrawler == nil {
		return fmt.Errorf("新闻爬虫服务未初始化")
	}

	// 查找新闻源
	newsSources := GetDefaultNewsSources()
	for _, source := range newsSources {
		if source.Name == sourceName {
			return s.newsCrawler.CrawlNewsSource(ctx, source)
		}
	}

	return fmt.Errorf("未找到新闻源: %s", sourceName)
}

// CrawlAllNews 爬取所有新闻源
func (s *Service) CrawlAllNews(ctx context.Context) error {
	if s.newsCrawler == nil {
		return fmt.Errorf("新闻爬虫服务未初始化")
	}

	newsSources := GetDefaultNewsSources()
	return s.newsCrawler.CrawlMultipleSources(ctx, newsSources)
}

// TriggerNewsCrawl 手动触发新闻爬取
func (s *Service) TriggerNewsCrawl(ctx context.Context, sourceName string) error {
	if s.newsScheduler == nil {
		return fmt.Errorf("新闻调度器未初始化")
	}
	return s.newsScheduler.TriggerCrawl(ctx, sourceName)
}

// TriggerNewsCrawlAll 手动触发全量新闻爬取
func (s *Service) TriggerNewsCrawlAll(ctx context.Context) error {
	if s.newsScheduler == nil {
		return fmt.Errorf("新闻调度器未初始化")
	}
	return s.newsScheduler.TriggerCrawlAll(ctx)
}

// GetNewsSchedulerStatus 获取新闻调度器状态
func (s *Service) GetNewsSchedulerStatus() map[string]interface{} {
	if s.newsScheduler == nil {
		return map[string]interface{}{
			"status": "disabled",
			"message": "新闻调度器未启用",
		}
	}
	return s.newsScheduler.GetStatus()
}

// CollectMoneyFlowData 采集个股资金流向数据
func (s *Service) CollectMoneyFlowData(ctx context.Context, symbol, tradeDate string) error {
	return s.sentimentService.CollectMoneyFlowData(ctx, symbol, tradeDate)
}

// CollectNorthboundFundData 采集北向资金数据
func (s *Service) CollectNorthboundFundData(ctx context.Context, tradeDate string) error {
	return s.sentimentService.CollectNorthboundFundData(ctx, tradeDate)
}

// CollectNorthboundTopStocksData 采集北向资金十大成交股数据
func (s *Service) CollectNorthboundTopStocksData(ctx context.Context, tradeDate, market string) error {
	return s.sentimentService.CollectNorthboundTopStocksData(ctx, tradeDate, market)
}

// CollectMarginTradingData 采集融资融券数据
func (s *Service) CollectMarginTradingData(ctx context.Context, tradeDate, exchange string) error {
	return s.sentimentService.CollectMarginTradingData(ctx, tradeDate, exchange)
}

// CollectETFBasicData 采集ETF基础数据
func (s *Service) CollectETFBasicData(ctx context.Context, market string) error {
	return s.sentimentService.CollectETFBasicData(ctx, market)
}

// CollectAllSentimentData 采集所有市场情绪和资金流向数据
func (s *Service) CollectAllSentimentData(ctx context.Context, tradeDate string) error {
	return s.sentimentService.CollectAllSentimentData(ctx, tradeDate)
}

// CollectActiveStocksMoneyFlow 采集活跃股票的资金流向数据
func (s *Service) CollectActiveStocksMoneyFlow(ctx context.Context, tradeDate string, symbols []string) error {
	return s.sentimentService.CollectActiveStocksMoneyFlow(ctx, tradeDate, symbols)
}

// GetSentimentCollectorStatus 获取市场情绪数据采集器状态
func (s *Service) GetSentimentCollectorStatus(ctx context.Context) (map[string]interface{}, error) {
	return s.sentimentService.GetSentimentCollectorStatus(ctx)
}

// AddNewsSource 添加新闻源
func (s *Service) AddNewsSource(source *NewsSource) error {
	if s.newsScheduler == nil {
		return fmt.Errorf("新闻调度器未初始化")
	}
	return s.newsScheduler.AddNewsSource(source)
}

// UpdateNewsSource 更新新闻源配置
func (s *Service) UpdateNewsSource(name string, source *NewsSource) error {
	if s.newsScheduler == nil {
		return fmt.Errorf("新闻调度器未初始化")
	}
	return s.newsScheduler.UpdateNewsSource(name, source)
}

// RemoveNewsSource 移除新闻源
func (s *Service) RemoveNewsSource(name string) error {
	if s.newsScheduler == nil {
		return fmt.Errorf("新闻调度器未初始化")
	}
	return s.newsScheduler.RemoveNewsSource(name)
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

	case "news_crawl":
		// 爬取指定新闻源
		sourceName, ok := params["source_name"].(string)
		if !ok || sourceName == "" {
			return fmt.Errorf("新闻爬取任务需要指定source_name参数")
		}
		return s.CrawlNews(ctx, sourceName)

	case "news_crawl_all":
		// 爬取所有新闻源
		return s.CrawlAllNews(ctx)

	// 市场情绪和资金流向数据采集任务
	case "money_flow", "northbound_fund", "northbound_top", "margin_trading", "etf_basic", "all_sentiment":
		return s.sentimentService.ExecuteSentimentCollectionTask(ctx, taskType, params)

	default:
		return fmt.Errorf("不支持的采集任务类型: %s", taskType)
	}
}
