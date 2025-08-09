package collector

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"data-collection-system/internal/cache"
	"data-collection-system/internal/config"
	"data-collection-system/internal/models"
	"data-collection-system/pkg/client"
	"data-collection-system/pkg/logger"
)

// TushareCollector Tushare数据采集器
type TushareCollector struct {
	client       *client.TushareClient
	cacheManager *cache.CacheManager
	config       config.TushareConfig
	retryConfig  RetryConfig
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries    int           // 最大重试次数
	InitialDelay  time.Duration // 初始延迟
	MaxDelay      time.Duration // 最大延迟
	BackoffFactor float64       // 退避因子
}

// CollectorError 采集器错误类型
type CollectorError struct {
	Operation string
	Code      string
	Message   string
	Retryable bool
	Cause     error
}

func (e *CollectorError) Error() string {
	return fmt.Sprintf("collector error [%s:%s]: %s", e.Operation, e.Code, e.Message)
}

func (e *CollectorError) Unwrap() error {
	return e.Cause
}

// NewTushareCollector 创建Tushare数据采集器
func NewTushareCollector(cfg config.TushareConfig, cacheManager *cache.CacheManager) *TushareCollector {
	tushareClient := client.NewTushareClient(cfg)
	
	// 默认重试配置
	retryConfig := RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}

	return &TushareCollector{
		client:       tushareClient,
		cacheManager: cacheManager,
		config:       cfg,
		retryConfig:  retryConfig,
	}
}

// Close 关闭采集器
func (tc *TushareCollector) Close() {
	if tc.client != nil {
		tc.client.Close()
	}
}

// retryWithBackoff 带退避的重试机制
func (tc *TushareCollector) retryWithBackoff(ctx context.Context, operation string, fn func() error) error {
	var lastErr error
	delay := tc.retryConfig.InitialDelay

	for attempt := 0; attempt <= tc.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// 等待退避时间
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			
			// 计算下次延迟时间
			delay = time.Duration(float64(delay) * tc.retryConfig.BackoffFactor)
			if delay > tc.retryConfig.MaxDelay {
				delay = tc.retryConfig.MaxDelay
			}
		}

		err := fn()
		if err == nil {
			if attempt > 0 {
				logger.Info(fmt.Sprintf("Operation %s succeeded after %d retries", operation, attempt))
			}
			return nil
		}

		lastErr = err
		
		// 检查是否为可重试错误
		if collectorErr, ok := err.(*CollectorError); ok && !collectorErr.Retryable {
			logger.Error(fmt.Sprintf("Non-retryable error in %s: %v", operation, err))
			return err
		}

		logger.Warn(fmt.Sprintf("Attempt %d/%d failed for %s: %v", attempt+1, tc.retryConfig.MaxRetries+1, operation, err))
	}

	return &CollectorError{
		Operation: operation,
		Code:      "MAX_RETRIES_EXCEEDED",
		Message:   fmt.Sprintf("Max retries (%d) exceeded", tc.retryConfig.MaxRetries),
		Retryable: false,
		Cause:     lastErr,
	}
}

// CollectStockBasic 采集股票基础数据
func (tc *TushareCollector) CollectStockBasic(ctx context.Context, exchange string) ([]*models.Stock, error) {
	var result []*models.Stock
	
	// 检查缓存
	cacheKey := cache.BuildKey(cache.KeyPrefixStock, "basic", exchange)
	if tc.cacheManager != nil {
		var cachedData []*models.Stock
		if err := tc.cacheManager.GetJSON(ctx, cacheKey, &cachedData); err == nil {
			logger.Info(fmt.Sprintf("Stock basic data loaded from cache for exchange: %s", exchange))
			return cachedData, nil
		}
	}

	err := tc.retryWithBackoff(ctx, "CollectStockBasic", func() error {
		resp, err := tc.client.GetStockBasic(ctx, exchange)
		if err != nil {
			return &CollectorError{
				Operation: "CollectStockBasic",
				Code:      "API_REQUEST_FAILED",
				Message:   "Failed to get stock basic data",
				Retryable: true,
				Cause:     err,
			}
		}

		if resp.Data == nil || len(resp.Data.Items) == 0 {
			return &CollectorError{
				Operation: "CollectStockBasic",
				Code:      "EMPTY_DATA",
				Message:   "No stock basic data returned",
				Retryable: false,
			}
		}

		// 解析数据
		result = make([]*models.Stock, 0, len(resp.Data.Items))
		for _, item := range resp.Data.Items {
			if len(item) < 12 {
				continue
			}

			// 解析上市日期
			var listDate *time.Time
			if dateStr := toString(item[9]); dateStr != "" {
				if parsedDate, err := time.Parse("20060102", dateStr); err == nil {
					listDate = &parsedDate
				}
			}

			stock := &models.Stock{
				Symbol:     toString(item[0]),
				Name:       toString(item[2]),
				Exchange:   toString(item[4]),
				Industry:   toString(item[6]),
				Sector:     toString(item[7]),
				ListDate:   listDate,
				Status:     1, // 默认为活跃状态
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			
			// 根据list_status设置状态
			if toString(item[8]) == "D" {
				stock.Status = 0 // 退市
			}
			
			result = append(result, stock)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 缓存数据
	if tc.cacheManager != nil {
		if err := tc.cacheManager.SetWithTTL(ctx, cacheKey, result, "daily"); err != nil {
			logger.Warn(fmt.Sprintf("Failed to cache stock basic data: %v", err))
		}
	}

	logger.Info(fmt.Sprintf("Collected %d stock basic records for exchange: %s", len(result), exchange))
	return result, nil
}

// CollectMarketData 采集行情数据（支持不同周期）
func (tc *TushareCollector) CollectMarketData(ctx context.Context, symbol, period, startDate, endDate string) ([]*models.MarketData, error) {
	var result []*models.MarketData
	
	// 检查缓存
	cacheKey := cache.BuildStockKey(symbol, fmt.Sprintf("%s_%s_%s", period, startDate, endDate))
	if tc.cacheManager != nil {
		var cachedData []*models.MarketData
		if err := tc.cacheManager.GetJSON(ctx, cacheKey, &cachedData); err == nil {
			logger.Info(fmt.Sprintf("Market data loaded from cache for %s (period: %s)", symbol, period))
			return cachedData, nil
		}
	}

	err := tc.retryWithBackoff(ctx, "CollectMarketData", func() error {
		var resp *client.TushareResponse
		var err error
		
		// 根据period参数调用不同的API
		switch period {
		case "1d", "daily":
			resp, err = tc.client.GetDailyQuote(ctx, symbol, startDate, endDate)
		case "1m", "5m", "15m", "30m", "60m":
			resp, err = tc.client.GetMinuteQuote(ctx, symbol, period, startDate, endDate)
		default:
			return &CollectorError{
				Operation: "CollectMarketData",
				Code:      "INVALID_PERIOD",
				Message:   fmt.Sprintf("Unsupported period: %s", period),
				Retryable: false,
			}
		}
		
		if err != nil {
			return &CollectorError{
				Operation: "CollectMarketData",
				Code:      "API_REQUEST_FAILED",
				Message:   fmt.Sprintf("Failed to get %s market data", period),
				Retryable: true,
				Cause:     err,
			}
		}

		if resp.Data == nil || len(resp.Data.Items) == 0 {
			return &CollectorError{
				Operation: "CollectMarketData",
				Code:      "EMPTY_DATA",
				Message:   fmt.Sprintf("No %s market data returned", period),
				Retryable: false,
			}
		}

		// 解析数据
		result = make([]*models.MarketData, 0, len(resp.Data.Items))
		for _, item := range resp.Data.Items {
			if len(item) < 10 {
				continue
			}

			// 解析交易日期和时间
			tradeDateStr := toString(item[1])
			var tradeDate, tradeTime time.Time
			var err error

			if period == "1d" {
				// 日线数据，只有日期
				tradeDate, err = time.Parse("20060102", tradeDateStr)
				if err != nil {
					continue
				}
				tradeTime = tradeDate
			} else {
				// 分钟数据，包含时间
				tradeTime, err = time.Parse("2006-01-02 15:04:05", tradeDateStr)
				if err != nil {
					continue
				}
				tradeDate = time.Date(tradeTime.Year(), tradeTime.Month(), tradeTime.Day(), 0, 0, 0, 0, tradeTime.Location())
			}

			marketData := &models.MarketData{
				Symbol:     symbol,
				TradeDate:  tradeDate,
				Period:     period,
				TradeTime:  tradeTime,
				OpenPrice:  toFloat64(item[2]),
				HighPrice:  toFloat64(item[3]),
				LowPrice:   toFloat64(item[4]),
				ClosePrice: toFloat64(item[5]),
				Volume:     int64(toFloat64(item[9])),
				Amount:     toFloat64(item[10]),
				CreatedAt:  time.Now(),
			}
			result = append(result, marketData)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 缓存数据
	if tc.cacheManager != nil {
		ttl := cache.TTLDaily
		// 根据period设置缓存时间
		switch period {
		case "1m", "5m":
			ttl = cache.TTLShort // 短周期数据缓存时间较短
		case "15m", "30m", "60m":
			ttl = time.Hour // 中等周期数据缓存1小时
		default:
			// 如果是当天数据，使用较短的缓存时间
			today := time.Now().Format("20060102")
			if endDate >= today {
				ttl = cache.TTLShort
			}
		}
		if err := tc.cacheManager.SetJSON(ctx, cacheKey, result, ttl); err != nil {
			logger.Warn(fmt.Sprintf("Failed to cache %s market data: %v", period, err))
		}
	}

	logger.Info(fmt.Sprintf("Collected %d %s market data records for %s", len(result), period, symbol))
	return result, nil
}



// CollectFinancialData 采集财务数据
func (tc *TushareCollector) CollectFinancialData(ctx context.Context, symbol, reportType string) (*models.FinancialData, error) {
	// 检查缓存
	cacheKey := cache.BuildStockKey(symbol, fmt.Sprintf("financial_%s", reportType))
	if tc.cacheManager != nil {
		var cachedData *models.FinancialData
		if err := tc.cacheManager.GetJSON(ctx, cacheKey, &cachedData); err == nil {
			logger.Info(fmt.Sprintf("Financial data loaded from cache for %s", symbol))
			return cachedData, nil
		}
	}

	// 获取最新的财务数据
	var resp *client.TushareResponse
	var err error
	
	// 根据reportType获取对应的财务数据
	switch reportType {
	case models.ReportTypeAnnual, models.ReportTypeQ1, models.ReportTypeQ2, models.ReportTypeQ3:
		resp, err = tc.client.GetIncomeStatement(ctx, symbol, reportType, "", "")
	default:
		return nil, &CollectorError{
			Operation: "CollectFinancialData",
			Code:      "INVALID_REPORT_TYPE",
			Message:   fmt.Sprintf("Unsupported report type: %s", reportType),
			Retryable: false,
		}
	}
	
	if err != nil {
		return nil, &CollectorError{
			Operation: "CollectFinancialData",
			Code:      "API_REQUEST_FAILED",
			Message:   "Failed to get financial data",
			Retryable: true,
			Cause:     err,
		}
	}
	
	if resp.Data == nil || len(resp.Data.Items) == 0 {
		return nil, &CollectorError{
			Operation: "CollectFinancialData",
			Code:      "EMPTY_DATA",
			Message:   "No financial data returned",
			Retryable: false,
		}
	}
	
	// 取最新的一条记录
	item := resp.Data.Items[0]
	if len(item) < 10 {
		return nil, &CollectorError{
			Operation: "CollectFinancialData",
			Code:      "INVALID_DATA",
			Message:   "Invalid financial data format",
			Retryable: false,
		}
	}
	
	// 解析报告日期
	reportDateStr := toString(item[1])
	reportDate, err := time.Parse("2006-01-02", reportDateStr)
	if err != nil {
		// 如果解析失败，使用当前时间
		reportDate = time.Now()
	}
	
	financialData := &models.FinancialData{
		Symbol:      symbol,
		ReportDate:  reportDate,
		ReportType:  reportType,
		Revenue:     &[]float64{toFloat64(item[2])}[0],
		NetProfit:   &[]float64{toFloat64(item[3])}[0],
		TotalAssets: &[]float64{toFloat64(item[4])}[0],
		TotalEquity: &[]float64{toFloat64(item[5])}[0],
		ROE:         &[]float64{toFloat64(item[6])}[0],
		ROA:         &[]float64{toFloat64(item[7])}[0],
		GrossMargin: &[]float64{toFloat64(item[8])}[0],
		NetMargin:   &[]float64{toFloat64(item[9])}[0],
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 缓存数据
	if tc.cacheManager != nil {
		if err := tc.cacheManager.SetJSON(ctx, cacheKey, financialData, cache.TTLDaily); err != nil {
			logger.Warn(fmt.Sprintf("Failed to cache financial data: %v", err))
		}
	}

	logger.Info(fmt.Sprintf("Collected financial data for %s (report type: %s)", symbol, reportType))
	return financialData, nil
}





// CollectMacroData 采集宏观经济数据
func (tc *TushareCollector) CollectMacroData(ctx context.Context, indicator string) (*models.MacroData, error) {
	var macroData *models.MacroData
	
	// 检查缓存
	cacheKey := cache.BuildKey(cache.KeyPrefixIndicator, "macro", indicator)
	if tc.cacheManager != nil {
		var cachedData *models.MacroData
		if err := tc.cacheManager.GetJSON(ctx, cacheKey, &cachedData); err == nil {
			logger.Info(fmt.Sprintf("Macro data loaded from cache for indicator: %s", indicator))
			return cachedData, nil
		}
	}

	err := tc.retryWithBackoff(ctx, "CollectMacroData", func() error {
		// 调用Tushare API获取宏观数据
		resp, err := tc.client.GetMacroData(ctx, indicator, "")
		if err != nil {
			return &CollectorError{
				Operation: "CollectMacroData",
				Code:      "API_REQUEST_FAILED",
				Message:   "Failed to get macro data",
				Retryable: true,
				Cause:     err,
			}
		}

		if resp == nil {
			return &CollectorError{
				Operation: "CollectMacroData",
				Code:      "EMPTY_DATA",
				Message:   "No macro data returned",
				Retryable: false,
			}
		}

		// 从Values中提取数据
		value := 0.0
		unit := ""
		if resp.Values != nil {
			if v, ok := resp.Values["value"]; ok {
				if val, ok := v.(float64); ok {
					value = val
				}
			}
			if u, ok := resp.Values["unit"]; ok {
				if unitStr, ok := u.(string); ok {
					unit = unitStr
				}
			}
		}
		
		// 转换为统一的MacroData格式
		macroData = &models.MacroData{
			IndicatorCode: indicator,
			IndicatorName: indicator,
			PeriodType:    resp.Period,
			DataDate:      time.Now(), // 使用当前时间，实际应该解析resp.Period
			Value:         value,
			Unit:          unit,
			CreatedAt:     time.Now(),
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 缓存数据
	if tc.cacheManager != nil {
		if err := tc.cacheManager.SetWithTTL(ctx, cacheKey, macroData, "weekly"); err != nil {
			logger.Warn(fmt.Sprintf("Failed to cache macro data: %v", err))
		}
	}

	logger.Info(fmt.Sprintf("Collected macro data for indicator: %s", indicator))
	return macroData, nil
}

// ValidateConnection 验证连接
func (tc *TushareCollector) ValidateConnection(ctx context.Context) error {
	return tc.client.ValidateToken(ctx)
}

// 辅助函数
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return 0
}