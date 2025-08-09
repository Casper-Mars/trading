package processing

import (
	"html"
	"regexp"
	"strings"
	"unicode"

	"data-collection-system/model"
)

// DataCleaner 数据清洗器
type DataCleaner struct {
	htmlTagRegex     *regexp.Regexp
	whitespaceRegex  *regexp.Regexp
	specialCharRegex *regexp.Regexp
	urlRegex         *regexp.Regexp
}

// NewDataCleaner 创建数据清洗器实例
func NewDataCleaner() *DataCleaner {
	return &DataCleaner{
		htmlTagRegex:     regexp.MustCompile(`<[^>]*>`),
		whitespaceRegex:  regexp.MustCompile(`\s+`),
		specialCharRegex: regexp.MustCompile(`[\x00-\x1f\x7f-\x9f]`),
		urlRegex:         regexp.MustCompile(`https?://[^\s]+`),
	}
}

// CleanNewsData 清洗新闻数据
func (c *DataCleaner) CleanNewsData(news *model.NewsData) (*model.NewsData, error) {
	cleanedNews := *news // 创建副本

	// 清洗标题
	cleanedNews.Title = c.cleanText(news.Title)

	// 清洗内容
	cleanedNews.Content = c.cleanText(news.Content)

	// 清洗来源
	cleanedNews.Source = c.cleanText(news.Source)

	// 标准化重要性级别
		news.ImportanceLevel = int8(c.normalizeImportanceLevel(int(news.ImportanceLevel)))

	return &cleanedNews, nil
}

// cleanText 清洗文本内容
func (c *DataCleaner) cleanText(text string) string {
	if text == "" {
		return text
	}

	// 1. HTML解码
	text = html.UnescapeString(text)

	// 2. 移除HTML标签
	text = c.htmlTagRegex.ReplaceAllString(text, "")

	// 3. 移除URL链接
	text = c.urlRegex.ReplaceAllString(text, "")

	// 4. 移除控制字符
	text = c.specialCharRegex.ReplaceAllString(text, "")

	// 5. 标准化空白字符
	text = c.whitespaceRegex.ReplaceAllString(text, " ")

	// 6. 移除首尾空白
	text = strings.TrimSpace(text)

	// 7. 移除多余的标点符号
	text = c.cleanPunctuation(text)

	return text
}

// cleanPunctuation 清理标点符号
func (c *DataCleaner) cleanPunctuation(text string) string {
	// 移除连续的标点符号
	punctuationRegex := regexp.MustCompile(`[。！？；，、]{2,}`)
	text = punctuationRegex.ReplaceAllStringFunc(text, func(match string) string {
		// 保留第一个标点符号
		return string([]rune(match)[0])
	})

	// 移除多余的括号
	bracketRegex := regexp.MustCompile(`\(\s*\)|\[\s*\]|\{\s*\}`)
	text = bracketRegex.ReplaceAllString(text, "")

	return text
}

// normalizeImportanceLevel 标准化重要性级别
func (c *DataCleaner) normalizeImportanceLevel(level int) int {
	if level < 1 {
		return 1
	}
	if level > 5 {
		return 5
	}
	return level
}

// CleanMarketData 清洗行情数据
func (c *DataCleaner) CleanMarketData(data *model.MarketData) (*model.MarketData, error) {
	cleanedData := *data // 创建副本

	// 标准化股票代码
	cleanedData.Symbol = c.normalizeStockSymbol(data.Symbol)

	// 验证和修正价格数据
	cleanedData.OpenPrice = c.normalizePriceValue(data.OpenPrice)
	cleanedData.HighPrice = c.normalizePriceValue(data.HighPrice)
	cleanedData.LowPrice = c.normalizePriceValue(data.LowPrice)
	cleanedData.ClosePrice = c.normalizePriceValue(data.ClosePrice)

	// 验证价格逻辑关系
	if cleanedData.HighPrice < cleanedData.LowPrice {
		// 交换高低价
		cleanedData.HighPrice, cleanedData.LowPrice = cleanedData.LowPrice, cleanedData.HighPrice
	}

	// 确保开盘价和收盘价在合理范围内
	if cleanedData.OpenPrice < cleanedData.LowPrice {
		cleanedData.OpenPrice = cleanedData.LowPrice
	}
	if cleanedData.OpenPrice > cleanedData.HighPrice {
		cleanedData.OpenPrice = cleanedData.HighPrice
	}
	if cleanedData.ClosePrice < cleanedData.LowPrice {
		cleanedData.ClosePrice = cleanedData.LowPrice
	}
	if cleanedData.ClosePrice > cleanedData.HighPrice {
		cleanedData.ClosePrice = cleanedData.HighPrice
	}

	// 验证成交量和成交额
	if cleanedData.Volume < 0 {
		cleanedData.Volume = 0
	}
	if cleanedData.Amount < 0 {
		cleanedData.Amount = 0
	}

	return &cleanedData, nil
}

// normalizeStockSymbol 标准化股票代码
func (c *DataCleaner) normalizeStockSymbol(symbol string) string {
	// 移除空白字符
	symbol = strings.TrimSpace(symbol)
	
	// 转换为大写
	symbol = strings.ToUpper(symbol)
	
	// 移除非字母数字字符
	var result strings.Builder
	for _, r := range symbol {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		}
	}
	
	return result.String()
}

// normalizePriceValue 标准化价格值
func (c *DataCleaner) normalizePriceValue(price float64) float64 {
	// 处理负价格
	if price < 0 {
		return 0
	}
	
	// 处理异常高价格（可能是数据错误）
	if price > 10000 {
		// 可能是分为单位，转换为元
		if price > 100000 {
			return price / 100
		}
	}
	
	return price
}

// CleanFinancialData 清洗财务数据
func (c *DataCleaner) CleanFinancialData(data *model.FinancialData) (*model.FinancialData, error) {
	cleanedData := *data // 创建副本

	// 标准化股票代码
	cleanedData.Symbol = c.normalizeStockSymbol(data.Symbol)

	// 清洗财务指标值
	cleanedData.Revenue = c.normalizeFinancialValue(data.Revenue)
	cleanedData.NetProfit = c.normalizeFinancialValue(data.NetProfit)
	cleanedData.TotalAssets = c.normalizeFinancialValue(data.TotalAssets)
	cleanedData.TotalEquity = c.normalizeFinancialValue(data.TotalEquity)

	// 清洗比率指标
	cleanedData.ROE = c.normalizeRatioValue(data.ROE)
	cleanedData.ROA = c.normalizeRatioValue(data.ROA)
	cleanedData.GrossMargin = c.normalizeRatioValue(data.GrossMargin)
	cleanedData.NetMargin = c.normalizeRatioValue(data.NetMargin)
	cleanedData.CurrentRatio = c.normalizeRatioValue(data.CurrentRatio)

	return &cleanedData, nil
}

// normalizeFinancialValue 标准化财务数值
func (c *DataCleaner) normalizeFinancialValue(value *float64) *float64 {
	if value == nil {
		return nil
	}
	
	// 处理异常值
	if *value < -1e12 || *value > 1e12 {
		return nil // 设为空值
	}
	
	return value
}

// normalizeRatioValue 标准化比率值
func (c *DataCleaner) normalizeRatioValue(value *float64) *float64 {
	if value == nil {
		return nil
	}
	
	// 比率值通常在-1到1之间，或者0到100%之间
	if *value < -10 || *value > 10 {
		return nil // 设为空值
	}
	
	return value
}