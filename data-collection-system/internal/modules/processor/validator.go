package processor

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"data-collection-system/internal/models"
)

// DefaultValidator 默认数据验证器
type DefaultValidator struct {
	stockCodePattern *regexp.Regexp
}

// NewDefaultValidator 创建默认验证器
func NewDefaultValidator() *DefaultValidator {
	// 股票代码正则表达式：支持A股、港股、美股格式
	stockCodePattern := regexp.MustCompile(`^[A-Z0-9]{2,10}\.[A-Z]{2,4}$|^[0-9]{6}$|^[A-Z]{1,5}$`)
	
	return &DefaultValidator{
		stockCodePattern: stockCodePattern,
	}
}

// ValidateMarketData 验证行情数据
func (v *DefaultValidator) ValidateMarketData(data *models.MarketData) error {
	if data == nil {
		return errors.New("market data is nil")
	}
	
	// 验证股票代码
	if err := v.validateStockCode(data.Symbol); err != nil {
		return fmt.Errorf("invalid symbol: %w", err)
	}
	
	// 验证交易日期
	if data.TradeDate.IsZero() {
		return errors.New("trade date is required")
	}
	
	// 验证交易日期不能是未来日期
	if data.TradeDate.After(time.Now()) {
		return errors.New("trade date cannot be in the future")
	}
	
	// 验证周期
	if data.Period == "" {
		return errors.New("period is required")
	}
	validPeriods := []string{"1m", "5m", "15m", "30m", "1h", "1d", "1w", "1M"}
	if !v.contains(validPeriods, data.Period) {
		return fmt.Errorf("invalid period: %s", data.Period)
	}
	
	// 验证价格数据
	if data.OpenPrice < 0 {
		return errors.New("open price cannot be negative")
	}
	if data.HighPrice < 0 {
		return errors.New("high price cannot be negative")
	}
	if data.LowPrice < 0 {
		return errors.New("low price cannot be negative")
	}
	if data.ClosePrice < 0 {
		return errors.New("close price cannot be negative")
	}
	
	// 验证价格逻辑关系
	if data.OpenPrice > 0 && data.HighPrice > 0 && data.OpenPrice > data.HighPrice {
		return errors.New("open price cannot be higher than high price")
	}
	if data.OpenPrice > 0 && data.LowPrice > 0 && data.OpenPrice < data.LowPrice {
		return errors.New("open price cannot be lower than low price")
	}
	if data.ClosePrice > 0 && data.HighPrice > 0 && data.ClosePrice > data.HighPrice {
		return errors.New("close price cannot be higher than high price")
	}
	if data.ClosePrice > 0 && data.LowPrice > 0 && data.ClosePrice < data.LowPrice {
		return errors.New("close price cannot be lower than low price")
	}
	
	// 验证成交量和成交额
	if data.Volume < 0 {
		return errors.New("volume cannot be negative")
	}
	if data.Amount < 0 {
		return errors.New("amount cannot be negative")
	}
	
	return nil
}

// ValidateFinancialData 验证财务数据
func (v *DefaultValidator) ValidateFinancialData(data *models.FinancialData) error {
	if data == nil {
		return errors.New("financial data is nil")
	}
	
	// 验证股票代码
	if err := v.validateStockCode(data.Symbol); err != nil {
		return fmt.Errorf("invalid symbol: %w", err)
	}
	
	// 验证报告日期
	if data.ReportDate.IsZero() {
		return errors.New("report date is required")
	}
	
	// 验证报告日期不能是未来日期
	if data.ReportDate.After(time.Now()) {
		return errors.New("report date cannot be in the future")
	}
	
	// 验证报告类型
	if data.ReportType == "" {
		return errors.New("report type is required")
	}
	validReportTypes := []string{"Q1", "Q2", "Q3", "A"}
	if !v.contains(validReportTypes, data.ReportType) {
		return fmt.Errorf("invalid report type: %s", data.ReportType)
	}
	
	// 验证财务指标的合理性
	if data.Revenue != nil && *data.Revenue < 0 {
		return errors.New("revenue cannot be negative")
	}
	
	// 验证资产负债表平衡 - 暂时跳过，因为模型中没有TotalLiabilities字段
	// if data.TotalAssets != nil && data.TotalLiabilities != nil && data.TotalEquity != nil {
	//	balance := *data.TotalAssets - (*data.TotalLiabilities + *data.TotalEquity)
	//	if math.Abs(balance) > 1000 { // 允许1000的误差
	//		return errors.New("balance sheet does not balance")
	//	}
	// }
	
	return nil
}

// ValidateNewsData 验证新闻数据
func (v *DefaultValidator) ValidateNewsData(data *models.NewsData) error {
	if data == nil {
		return errors.New("news data is nil")
	}
	
	// 验证标题
	if strings.TrimSpace(data.Title) == "" {
		return errors.New("title is required")
	}
	if len(data.Title) > 500 {
		return errors.New("title is too long (max 500 characters)")
	}
	
	// 验证内容
	if strings.TrimSpace(data.Content) == "" {
		return errors.New("content is required")
	}
	if len(data.Content) > 50000 {
		return errors.New("content is too long (max 50000 characters)")
	}
	
	// 验证发布时间
	if data.PublishTime.IsZero() {
		return errors.New("publish time is required")
	}
	
	// 验证发布时间不能是未来时间（允许5分钟误差）
	if data.PublishTime.After(time.Now().Add(5 * time.Minute)) {
		return errors.New("publish time cannot be in the future")
	}
	
	// 验证来源
	if strings.TrimSpace(data.Source) == "" {
		return errors.New("source is required")
	}
	
	// 验证分类
	if data.Category != "" {
		validCategories := []string{"market", "company", "industry", "policy", "macro", "international"}
		if !v.contains(validCategories, data.Category) {
			return fmt.Errorf("invalid category: %s", data.Category)
		}
	}
	
	// 验证情感分数
	if data.SentimentScore != nil {
		if *data.SentimentScore < -1.0 || *data.SentimentScore > 1.0 {
			return errors.New("sentiment score must be between -1.0 and 1.0")
		}
	}
	
	// 验证相关股票代码格式
	if len(data.RelatedStocks) > 0 {
		for _, stock := range data.RelatedStocks {
			stock = strings.TrimSpace(stock)
			if stock != "" {
				if err := v.validateStockCode(stock); err != nil {
					return fmt.Errorf("invalid related stock code %s: %w", stock, err)
				}
			}
		}
	}
	
	return nil
}

// ValidateMacroData 验证宏观数据
func (v *DefaultValidator) ValidateMacroData(data *models.MacroData) error {
	if data == nil {
		return errors.New("macro data is nil")
	}
	
	// 验证指标代码
	if strings.TrimSpace(data.IndicatorCode) == "" {
		return errors.New("indicator code is required")
	}
	
	// 验证指标名称
	if strings.TrimSpace(data.IndicatorName) == "" {
		return errors.New("indicator name is required")
	}
	
	// 验证数据日期
	if data.DataDate.IsZero() {
		return errors.New("data date is required")
	}
	
	// 验证数据日期不能是未来日期
	if data.DataDate.After(time.Now()) {
		return errors.New("data date cannot be in the future")
	}
	
	// 验证周期类型
	if data.PeriodType != "" {
		validPeriodTypes := []string{"daily", "weekly", "monthly", "quarterly", "yearly"}
		if !v.contains(validPeriodTypes, data.PeriodType) {
			return fmt.Errorf("invalid period type: %s", data.PeriodType)
		}
	}
	
	// 验证数值的合理性（根据指标类型）
	if data.Value != 0 {
		// GDP、CPI等指标的基本合理性检查
		if strings.Contains(strings.ToLower(data.IndicatorCode), "cpi") {
			// CPI通常在-10%到50%之间
			if data.Value < -10 || data.Value > 50 {
				return fmt.Errorf("CPI value %f is out of reasonable range", data.Value)
			}
		}
		if strings.Contains(strings.ToLower(data.IndicatorCode), "gdp") {
			// GDP增长率通常在-20%到30%之间
			if data.Value < -20 || data.Value > 30 {
				return fmt.Errorf("GDP value %f is out of reasonable range", data.Value)
			}
		}
		if strings.Contains(strings.ToLower(data.IndicatorCode), "rate") {
			// 利率通常在-5%到50%之间
			if data.Value < -5 || data.Value > 50 {
				return fmt.Errorf("interest rate value %f is out of reasonable range", data.Value)
			}
		}
	}
	
	return nil
}

// validateStockCode 验证股票代码格式
func (v *DefaultValidator) validateStockCode(code string) error {
	if strings.TrimSpace(code) == "" {
		return errors.New("stock code is required")
	}
	
	code = strings.TrimSpace(strings.ToUpper(code))
	
	// 使用正则表达式验证股票代码格式
	if !v.stockCodePattern.MatchString(code) {
		return fmt.Errorf("invalid stock code format: %s", code)
	}
	
	return nil
}

// contains 检查切片是否包含指定元素
func (v *DefaultValidator) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}