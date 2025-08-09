package processing

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"data-collection-system/model"
)

// DataValidator 数据验证器
type DataValidator struct {
	stockSymbolRegex *regexp.Regexp
	emailRegex       *regexp.Regexp
	urlRegex         *regexp.Regexp
}

// NewDataValidator 创建数据验证器实例
func NewDataValidator() *DataValidator {
	return &DataValidator{
		stockSymbolRegex: regexp.MustCompile(`^[A-Z0-9]{6}$`),
		emailRegex:       regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
		urlRegex:         regexp.MustCompile(`^https?://[^\s]+$`),
	}
}

// ValidateNewsData 验证新闻数据
func (v *DataValidator) ValidateNewsData(news *model.NewsData) error {
	if news == nil {
		return errors.New("news data is nil")
	}

	// 验证标题
	if err := v.validateTitle(news.Title); err != nil {
		return fmt.Errorf("title validation failed: %w", err)
	}

	// 验证内容
	if err := v.validateContent(news.Content); err != nil {
		return fmt.Errorf("content validation failed: %w", err)
	}

	// 验证来源
	if err := v.validateSource(news.Source); err != nil {
		return fmt.Errorf("source validation failed: %w", err)
	}

	// 验证发布时间
	if err := v.validatePublishTime(news.PublishTime); err != nil {
		return fmt.Errorf("publish time validation failed: %w", err)
	}

	// 验证重要性级别
	if err := v.validateImportanceLevel(int(news.ImportanceLevel)); err != nil {
		return fmt.Errorf("importance level validation failed: %w", err)
	}

	// 验证情感分析结果
	if err := v.validateSentiment(news.SentimentScore); err != nil {
		return fmt.Errorf("sentiment validation failed: %w", err)
	}

	// 验证关联股票
	if err := v.validateRelatedStocks(news.RelatedStocks); err != nil {
		return fmt.Errorf("related stocks validation failed: %w", err)
	}

	return nil
}

// validateTitle 验证标题
func (v *DataValidator) validateTitle(title string) error {
	if strings.TrimSpace(title) == "" {
		return errors.New("title cannot be empty")
	}

	if len(title) > 500 {
		return errors.New("title too long (max 500 characters)")
	}

	if len(title) < 5 {
		return errors.New("title too short (min 5 characters)")
	}

	return nil
}

// validateContent 验证内容
func (v *DataValidator) validateContent(content string) error {
	if strings.TrimSpace(content) == "" {
		return errors.New("content cannot be empty")
	}

	if len(content) > 50000 {
		return errors.New("content too long (max 50000 characters)")
	}

	if len(content) < 10 {
		return errors.New("content too short (min 10 characters)")
	}

	return nil
}

// validateSource 验证来源
func (v *DataValidator) validateSource(source string) error {
	if strings.TrimSpace(source) == "" {
		return errors.New("source cannot be empty")
	}

	if len(source) > 100 {
		return errors.New("source too long (max 100 characters)")
	}

	return nil
}

// validatePublishTime 验证发布时间
func (v *DataValidator) validatePublishTime(publishTime time.Time) error {
	if publishTime.IsZero() {
		return errors.New("publish time cannot be zero")
	}

	// 检查时间是否在合理范围内（不能是未来时间，不能太久远）
	now := time.Now()
	if publishTime.After(now) {
		return errors.New("publish time cannot be in the future")
	}

	// 不能超过10年前
	tenYearsAgo := now.AddDate(-10, 0, 0)
	if publishTime.Before(tenYearsAgo) {
		return errors.New("publish time too old (more than 10 years ago)")
	}

	return nil
}

// validateImportanceLevel 验证重要性级别
func (v *DataValidator) validateImportanceLevel(level int) error {
	if level < 1 || level > 5 {
		return errors.New("importance level must be between 1 and 5")
	}
	return nil
}

// validateSentiment 验证情感分析结果
func (v *DataValidator) validateSentiment(score *float64) error {
	if score == nil {
		return nil // 允许为空
	}

	if *score < -1.0 || *score > 1.0 {
		return errors.New("sentiment score must be between -1.0 and 1.0")
	}

	return nil
}

// validateRelatedStocks 验证关联股票
func (v *DataValidator) validateRelatedStocks(stocks []string) error {
	if len(stocks) > 20 {
		return errors.New("too many related stocks (max 20)")
	}

	for _, stock := range stocks {
		if !v.stockSymbolRegex.MatchString(stock) {
			return fmt.Errorf("invalid stock symbol format: %s", stock)
		}
	}

	return nil
}

// ValidateMarketData 验证行情数据
func (v *DataValidator) ValidateMarketData(data *model.MarketData) error {
	if data == nil {
		return errors.New("market data is nil")
	}

	// 验证股票代码
	if err := v.validateStockSymbol(data.Symbol); err != nil {
		return fmt.Errorf("symbol validation failed: %w", err)
	}

	// 验证交易日期
	if err := v.validateTradeDate(data.TradeDate); err != nil {
		return fmt.Errorf("trade date validation failed: %w", err)
	}

	// 验证价格数据
	if err := v.validatePriceData(data); err != nil {
		return fmt.Errorf("price data validation failed: %w", err)
	}

	// 验证成交量和成交额
	if err := v.validateVolumeData(data); err != nil {
		return fmt.Errorf("volume data validation failed: %w", err)
	}

	return nil
}

// validateStockSymbol 验证股票代码
func (v *DataValidator) validateStockSymbol(symbol string) error {
	if strings.TrimSpace(symbol) == "" {
		return errors.New("stock symbol cannot be empty")
	}

	if !v.stockSymbolRegex.MatchString(symbol) {
		return fmt.Errorf("invalid stock symbol format: %s (expected 6 alphanumeric characters)", symbol)
	}

	return nil
}

// validateTradeDate 验证交易日期
func (v *DataValidator) validateTradeDate(tradeDate time.Time) error {
	if tradeDate.IsZero() {
		return errors.New("trade date cannot be zero")
	}

	// 检查日期是否在合理范围内
	now := time.Now()
	if tradeDate.After(now) {
		return errors.New("trade date cannot be in the future")
	}

	// 不能超过30年前
	thirtyYearsAgo := now.AddDate(-30, 0, 0)
	if tradeDate.Before(thirtyYearsAgo) {
		return errors.New("trade date too old (more than 30 years ago)")
	}

	return nil
}

// validatePriceData 验证价格数据
func (v *DataValidator) validatePriceData(data *model.MarketData) error {
	// 检查价格是否为正数
	if data.OpenPrice <= 0 {
		return errors.New("open price must be positive")
	}
	if data.HighPrice <= 0 {
		return errors.New("high price must be positive")
	}
	if data.LowPrice <= 0 {
		return errors.New("low price must be positive")
	}
	if data.ClosePrice <= 0 {
		return errors.New("close price must be positive")
	}

	// 检查价格逻辑关系
	if data.HighPrice < data.LowPrice {
		return errors.New("high price cannot be less than low price")
	}

	if data.OpenPrice < data.LowPrice || data.OpenPrice > data.HighPrice {
		return errors.New("open price must be between low and high price")
	}

	if data.ClosePrice < data.LowPrice || data.ClosePrice > data.HighPrice {
		return errors.New("close price must be between low and high price")
	}

	// 检查价格是否在合理范围内（防止数据错误）
	maxPrice := 10000.0 // 假设最高价格不超过10000元
	if data.HighPrice > maxPrice {
		return fmt.Errorf("high price too high (max %.2f)", maxPrice)
	}

	return nil
}

// validateVolumeData 验证成交量和成交额数据
func (v *DataValidator) validateVolumeData(data *model.MarketData) error {
	// 成交量不能为负
	if data.Volume < 0 {
		return errors.New("volume cannot be negative")
	}

	// 成交额不能为负
	if data.Amount < 0 {
		return errors.New("amount cannot be negative")
	}

	// 检查成交量和成交额的逻辑关系
	if data.Volume > 0 && data.Amount == 0 {
		return errors.New("amount should be positive when volume is positive")
	}

	if data.Volume == 0 && data.Amount > 0 {
		return errors.New("volume should be positive when amount is positive")
	}

	// 检查成交额是否合理（简单的逻辑检查）
	if data.Volume > 0 && data.Amount > 0 {
		avgPrice := data.Amount / float64(data.Volume)
		if avgPrice < data.LowPrice*0.8 || avgPrice > data.HighPrice*1.2 {
			return errors.New("average price derived from volume and amount is inconsistent with price range")
		}
	}

	return nil
}

// ValidateFinancialData 验证财务数据
func (v *DataValidator) ValidateFinancialData(data *model.FinancialData) error {
	if data == nil {
		return errors.New("financial data is nil")
	}

	// 验证股票代码
	if err := v.validateStockSymbol(data.Symbol); err != nil {
		return fmt.Errorf("symbol validation failed: %w", err)
	}

	// 验证报告期
	if err := v.validateReportDate(data.ReportDate); err != nil {
		return fmt.Errorf("report date validation failed: %w", err)
	}

	// 验证报告类型
	if err := v.validateReportType(data.ReportType); err != nil {
		return fmt.Errorf("report type validation failed: %w", err)
	}

	// 验证财务指标的逻辑关系
	if err := v.validateFinancialLogic(data); err != nil {
		return fmt.Errorf("financial logic validation failed: %w", err)
	}

	return nil
}

// validateReportDate 验证报告期
func (v *DataValidator) validateReportDate(reportDate time.Time) error {
	if reportDate.IsZero() {
		return errors.New("report date cannot be zero")
	}

	// 报告期不能是未来时间
	now := time.Now()
	if reportDate.After(now) {
		return errors.New("report date cannot be in the future")
	}

	// 报告期不能太久远
	twentyYearsAgo := now.AddDate(-20, 0, 0)
	if reportDate.Before(twentyYearsAgo) {
		return errors.New("report date too old (more than 20 years ago)")
	}

	return nil
}

// validateReportType 验证报告类型
func (v *DataValidator) validateReportType(reportType string) error {
	validTypes := map[string]bool{
		"Q1": true,
		"Q2": true,
		"Q3": true,
		"A":  true,
	}

	if !validTypes[reportType] {
		return fmt.Errorf("invalid report type: %s (must be Q1, Q2, Q3, or A)", reportType)
	}

	return nil
}

// validateFinancialLogic 验证财务数据的逻辑关系
func (v *DataValidator) validateFinancialLogic(data *model.FinancialData) error {
	// 验证资产负债表逻辑
	if data.TotalAssets != nil && data.TotalEquity != nil {
		if *data.TotalEquity > *data.TotalAssets {
			return errors.New("total equity cannot be greater than total assets")
		}
	}

	// 验证比率指标范围
	if data.ROE != nil && (*data.ROE < -1 || *data.ROE > 1) {
		return errors.New("ROE should be between -1 and 1 (or -100% to 100%)")
	}

	if data.ROA != nil && (*data.ROA < -1 || *data.ROA > 1) {
		return errors.New("ROA should be between -1 and 1 (or -100% to 100%)")
	}

	if data.GrossMargin != nil && (*data.GrossMargin < -1 || *data.GrossMargin > 1) {
		return errors.New("gross margin should be between -1 and 1 (or -100% to 100%)")
	}

	if data.NetMargin != nil && (*data.NetMargin < -1 || *data.NetMargin > 1) {
		return errors.New("net margin should be between -1 and 1 (or -100% to 100%)")
	}

	if data.CurrentRatio != nil && *data.CurrentRatio < 0 {
		return errors.New("current ratio cannot be negative")
	}

	return nil
}