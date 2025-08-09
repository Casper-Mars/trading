package processor

import (
	"html"
	"regexp"
	"strconv"
	"strings"

	"data-collection-system/internal/models"
)

// DefaultCleaner 默认数据清洗器
type DefaultCleaner struct {
	htmlTagRegex    *regexp.Regexp
	whitespaceRegex *regexp.Regexp
	specialCharRegex *regexp.Regexp
}

// NewDefaultCleaner 创建默认清洗器
func NewDefaultCleaner() *DefaultCleaner {
	return &DefaultCleaner{
		htmlTagRegex:     regexp.MustCompile(`<[^>]*>`),
		whitespaceRegex:  regexp.MustCompile(`\s+`),
		specialCharRegex: regexp.MustCompile(`[\x00-\x1f\x7f-\x9f]`),
	}
}

// CleanMarketData 清洗行情数据
func (c *DefaultCleaner) CleanMarketData(data *models.MarketData) *models.MarketData {
	if data == nil {
		return nil
	}
	
	// 创建副本避免修改原始数据
	cleaned := *data
	
	// 清洗股票代码
	cleaned.Symbol = c.cleanStockCode(data.Symbol)
	
	// 清洗周期
	cleaned.Period = c.cleanPeriod(data.Period)
	
	// 清洗价格数据（保留4位小数）
	cleaned.OpenPrice = c.cleanPriceValue(data.OpenPrice)
	cleaned.HighPrice = c.cleanPriceValue(data.HighPrice)
	cleaned.LowPrice = c.cleanPriceValue(data.LowPrice)
	cleaned.ClosePrice = c.cleanPriceValue(data.ClosePrice)
	
	// 清洗成交量（整数）
	cleaned.Volume = c.cleanVolumeValue(data.Volume)
	
	// 清洗成交额（保留2位小数）
	cleaned.Amount = c.cleanAmountValue(data.Amount)
	

	
	return &cleaned
}

// CleanFinancialData 清洗财务数据
func (c *DefaultCleaner) CleanFinancialData(data *models.FinancialData) *models.FinancialData {
	if data == nil {
		return nil
	}
	
	// 创建副本避免修改原始数据
	cleaned := *data
	
	// 清洗股票代码
	cleaned.Symbol = c.cleanStockCode(data.Symbol)
	
	// 清洗报告类型
	cleaned.ReportType = c.cleanReportType(data.ReportType)
	
	// 清洗财务数据（保留2位小数，单位：万元）
	cleaned.Revenue = c.cleanFinancialAmount(data.Revenue)
	cleaned.NetProfit = c.cleanFinancialAmount(data.NetProfit)
	cleaned.TotalAssets = c.cleanFinancialAmount(data.TotalAssets)
	cleaned.TotalEquity = c.cleanFinancialAmount(data.TotalEquity)
	
	// 清洗财务比率（保留4位小数）
	cleaned.ROE = c.cleanPercentage(data.ROE)
	cleaned.ROA = c.cleanPercentage(data.ROA)
	cleaned.GrossMargin = c.cleanPercentage(data.GrossMargin)
	cleaned.NetMargin = c.cleanPercentage(data.NetMargin)
	cleaned.CurrentRatio = c.cleanRatio(data.CurrentRatio)
	
	return &cleaned
}

// CleanNewsData 清洗新闻数据
func (c *DefaultCleaner) CleanNewsData(data *models.NewsData) *models.NewsData {
	if data == nil {
		return nil
	}
	
	// 创建副本避免修改原始数据
	cleaned := *data
	
	// 清洗标题
	cleaned.Title = c.cleanText(data.Title)
	
	// 清洗内容
	cleaned.Content = c.cleanNewsContent(data.Content)
	

	
	// 清洗来源
	cleaned.Source = c.cleanText(data.Source)
	
	// 清洗分类
	cleaned.Category = c.cleanCategory(data.Category)
	
	// 清洗相关股票和行业（StringSlice类型）
	cleaned.RelatedStocks = data.RelatedStocks
	cleaned.RelatedIndustries = data.RelatedIndustries
	
	// 清洗情感分数
	cleaned.SentimentScore = c.cleanSentimentScore(data.SentimentScore)
	
	return &cleaned
}

// CleanMacroData 清洗宏观数据
func (c *DefaultCleaner) CleanMacroData(data *models.MacroData) *models.MacroData {
	if data == nil {
		return nil
	}
	
	// 创建副本避免修改原始数据
	cleaned := *data
	
	// 清洗指标代码
	cleaned.IndicatorCode = c.cleanIndicatorCode(data.IndicatorCode)
	
	// 清洗指标名称
	cleaned.IndicatorName = c.cleanText(data.IndicatorName)
	
	// 清洗周期类型
	cleaned.PeriodType = c.cleanPeriodType(data.PeriodType)
	
	// 清洗数值（保留4位小数）
	cleaned.Value = c.cleanMacroValueDirect(data.Value)
	
	// 清洗单位
	cleaned.Unit = c.cleanText(data.Unit)
	

	
	return &cleaned
}

// cleanStockCode 清洗股票代码
func (c *DefaultCleaner) cleanStockCode(code string) string {
	// 去除空白字符并转为大写
	code = strings.TrimSpace(strings.ToUpper(code))
	
	// 移除特殊字符
	code = c.specialCharRegex.ReplaceAllString(code, "")
	
	return code
}

// cleanPeriod 清洗周期
func (c *DefaultCleaner) cleanPeriod(period string) string {
	period = strings.TrimSpace(strings.ToLower(period))
	
	// 标准化周期格式
	periodMap := map[string]string{
		"1min":   "1m",
		"1minute": "1m",
		"5min":   "5m",
		"5minute": "5m",
		"15min":  "15m",
		"15minute": "15m",
		"30min":  "30m",
		"30minute": "30m",
		"1hour":  "1h",
		"1hr":    "1h",
		"1day":   "1d",
		"daily":  "1d",
		"1week":  "1w",
		"weekly": "1w",
		"1month": "1M",
		"monthly": "1M",
	}
	
	if standardPeriod, exists := periodMap[period]; exists {
		return standardPeriod
	}
	
	return period
}

// cleanPrice 清洗价格数据
func (c *DefaultCleaner) cleanPrice(price *float64) *float64 {
	if price == nil {
		return nil
	}
	
	// 保留4位小数
	rounded := c.roundToDecimal(*price, 4)
	return &rounded
}

// cleanPriceValue 清洗价格值
func (c *DefaultCleaner) cleanPriceValue(price float64) float64 {
	if price < 0 {
		return 0
	}
	return c.roundToDecimal(price, 4)
}

// cleanVolume 清洗成交量
func (c *DefaultCleaner) cleanVolume(volume *int64) *int64 {
	if volume == nil {
		return nil
	}
	
	// 确保成交量为非负数
	if *volume < 0 {
		zero := int64(0)
		return &zero
	}
	
	return volume
}

// cleanVolumeValue 清洗成交量值
func (c *DefaultCleaner) cleanVolumeValue(volume int64) int64 {
	if volume < 0 {
		return 0
	}
	return volume
}

// cleanAmount 清洗成交额
func (c *DefaultCleaner) cleanAmount(amount *float64) *float64 {
	if amount == nil {
		return nil
	}
	
	// 确保成交额为非负数，保留2位小数
	if *amount < 0 {
		zero := 0.0
		return &zero
	}
	
	rounded := c.roundToDecimal(*amount, 2)
	return &rounded
}

// cleanAmountValue 清洗成交额值
func (c *DefaultCleaner) cleanAmountValue(amount float64) float64 {
	if amount < 0 {
		return 0
	}
	// 保留2位小数
	return c.roundToDecimal(amount, 2)
}

// cleanPercentage 清洗百分比数据
func (c *DefaultCleaner) cleanPercentage(percentage *float64) *float64 {
	if percentage == nil {
		return nil
	}
	
	// 保留4位小数
	rounded := c.roundToDecimal(*percentage, 4)
	return &rounded
}

// cleanRatio 清洗比率数据
func (c *DefaultCleaner) cleanRatio(ratio *float64) *float64 {
	if ratio == nil {
		return nil
	}
	
	// 确保比率为非负数，保留4位小数
	if *ratio < 0 {
		zero := 0.0
		return &zero
	}
	
	rounded := c.roundToDecimal(*ratio, 4)
	return &rounded
}

// cleanFinancialAmount 清洗财务金额
func (c *DefaultCleaner) cleanFinancialAmount(amount *float64) *float64 {
	if amount == nil {
		return nil
	}
	
	// 保留2位小数（单位：万元）
	rounded := c.roundToDecimal(*amount, 2)
	return &rounded
}

// cleanReportType 清洗报告类型
func (c *DefaultCleaner) cleanReportType(reportType string) string {
	reportType = strings.TrimSpace(strings.ToUpper(reportType))
	
	// 标准化报告类型
	typeMap := map[string]string{
		"Q1":       "Q1",
		"QUARTER1": "Q1",
		"第一季度":    "Q1",
		"Q2":       "Q2",
		"QUARTER2": "Q2",
		"第二季度":    "Q2",
		"Q3":       "Q3",
		"QUARTER3": "Q3",
		"第三季度":    "Q3",
		"A":        "A",
		"ANNUAL":   "A",
		"年报":      "A",
	}
	
	if standardType, exists := typeMap[reportType]; exists {
		return standardType
	}
	
	return reportType
}

// cleanText 清洗文本内容
func (c *DefaultCleaner) cleanText(text string) string {
	if text == "" {
		return text
	}
	
	// 移除HTML标签
	text = c.htmlTagRegex.ReplaceAllString(text, "")
	
	// HTML解码
	text = html.UnescapeString(text)
	
	// 移除控制字符
	text = c.specialCharRegex.ReplaceAllString(text, "")
	
	// 标准化空白字符
	text = c.whitespaceRegex.ReplaceAllString(text, " ")
	
	// 去除首尾空白
	text = strings.TrimSpace(text)
	
	return text
}

// cleanNewsContent 清洗新闻内容
func (c *DefaultCleaner) cleanNewsContent(content string) string {
	if content == "" {
		return content
	}
	
	// 基础文本清洗
	content = c.cleanText(content)
	
	// 移除常见的无用内容
	uselessPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\s*更多.*?请关注.*?`),
		regexp.MustCompile(`(?i)\s*本文.*?首发.*?`),
		regexp.MustCompile(`(?i)\s*免责声明.*?`),
		regexp.MustCompile(`(?i)\s*风险提示.*?`),
		regexp.MustCompile(`(?i)\s*\(.*?编辑.*?\)`),
	}
	
	for _, pattern := range uselessPatterns {
		content = pattern.ReplaceAllString(content, "")
	}
	
	// 再次清理空白字符
	content = c.whitespaceRegex.ReplaceAllString(content, " ")
	content = strings.TrimSpace(content)
	
	return content
}

// cleanCategory 清洗分类
func (c *DefaultCleaner) cleanCategory(category string) string {
	category = strings.TrimSpace(strings.ToLower(category))
	
	// 标准化分类
	categoryMap := map[string]string{
		"市场":        "market",
		"公司":        "company",
		"行业":        "industry",
		"政策":        "policy",
		"宏观":        "macro",
		"国际":        "international",
		"经济":        "macro",
		"股市":        "market",
		"企业":        "company",
		"板块":        "industry",
	}
	
	if standardCategory, exists := categoryMap[category]; exists {
		return standardCategory
	}
	
	return category
}

// cleanTags 清洗标签
func (c *DefaultCleaner) cleanTags(tags string) string {
	if tags == "" {
		return tags
	}
	
	// 分割标签
	tagList := strings.Split(tags, ",")
	cleanedTags := make([]string, 0, len(tagList))
	
	for _, tag := range tagList {
		tag = c.cleanText(tag)
		if tag != "" && len(tag) <= 50 { // 限制标签长度
			cleanedTags = append(cleanedTags, tag)
		}
	}
	
	return strings.Join(cleanedTags, ",")
}

// cleanRelatedStocks 清洗相关股票
func (c *DefaultCleaner) cleanRelatedStocks(stocks string) string {
	if stocks == "" {
		return stocks
	}
	
	// 分割股票代码
	stockList := strings.Split(stocks, ",")
	cleanedStocks := make([]string, 0, len(stockList))
	
	for _, stock := range stockList {
		stock = c.cleanStockCode(stock)
		if stock != "" {
			cleanedStocks = append(cleanedStocks, stock)
		}
	}
	
	return strings.Join(cleanedStocks, ",")
}

// cleanSentimentScore 清洗情感分数
func (c *DefaultCleaner) cleanSentimentScore(score *float64) *float64 {
	if score == nil {
		return nil
	}
	
	// 限制在-1到1之间
	if *score < -1.0 {
		minScore := -1.0
		return &minScore
	}
	if *score > 1.0 {
		maxScore := 1.0
		return &maxScore
	}
	
	// 保留4位小数
	rounded := c.roundToDecimal(*score, 4)
	return &rounded
}

// cleanIndicatorCode 清洗指标代码
func (c *DefaultCleaner) cleanIndicatorCode(code string) string {
	code = strings.TrimSpace(strings.ToUpper(code))
	
	// 移除特殊字符，保留字母、数字、下划线
	code = regexp.MustCompile(`[^A-Z0-9_]`).ReplaceAllString(code, "")
	
	return code
}

// cleanPeriodType 清洗周期类型
func (c *DefaultCleaner) cleanPeriodType(periodType string) string {
	periodType = strings.TrimSpace(strings.ToLower(periodType))
	
	// 标准化周期类型
	typeMap := map[string]string{
		"日":       "daily",
		"周":       "weekly",
		"月":       "monthly",
		"季":       "quarterly",
		"年":       "yearly",
		"day":     "daily",
		"week":    "weekly",
		"month":   "monthly",
		"quarter": "quarterly",
		"year":    "yearly",
	}
	
	if standardType, exists := typeMap[periodType]; exists {
		return standardType
	}
	
	return periodType
}

// cleanMacroValue 清洗宏观数据值
func (c *DefaultCleaner) cleanMacroValue(value *float64) *float64 {
	if value == nil {
		return nil
	}
	
	// 保留4位小数
	rounded := c.roundToDecimal(*value, 4)
	return &rounded
}

// cleanMacroValueDirect 清洗宏观数据值（直接值）
func (c *DefaultCleaner) cleanMacroValueDirect(value float64) float64 {
	// 保留4位小数
	return c.roundToDecimal(value, 4)
}

// roundToDecimal 四舍五入到指定小数位
func (c *DefaultCleaner) roundToDecimal(value float64, decimals int) float64 {
	multiplier := 1.0
	for i := 0; i < decimals; i++ {
		multiplier *= 10
	}
	
	rounded, _ := strconv.ParseFloat(strconv.FormatFloat(value*multiplier, 'f', 0, 64), 64)
	return rounded / multiplier
}