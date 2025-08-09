package processor

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"data-collection-system/internal/models"
)

// DefaultQualityChecker 默认数据质量检查器
type DefaultQualityChecker struct {
}

// NewDefaultQualityChecker 创建默认质量检查器
func NewDefaultQualityChecker() *DefaultQualityChecker {
	return &DefaultQualityChecker{}
}

// CheckQuality 检查数据质量
func (q *DefaultQualityChecker) CheckQuality(ctx context.Context, dataType string, data interface{}) (*QualityReport, error) {
	switch dataType {
	case "market":
		marketData, ok := data.(*models.MarketData)
		if !ok {
			return nil, fmt.Errorf("invalid market data type")
		}
		return q.checkMarketDataQuality(marketData), nil
	case "financial":
		financialData, ok := data.(*models.FinancialData)
		if !ok {
			return nil, fmt.Errorf("invalid financial data type")
		}
		return q.checkFinancialDataQuality(financialData), nil
	case "news":
		newsData, ok := data.(*models.NewsData)
		if !ok {
			return nil, fmt.Errorf("invalid news data type")
		}
		return q.checkNewsDataQuality(newsData), nil
	case "macro":
		macroData, ok := data.(*models.MacroData)
		if !ok {
			return nil, fmt.Errorf("invalid macro data type")
		}
		return q.checkMacroDataQuality(macroData), nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
}

// checkMarketDataQuality 检查行情数据质量
func (q *DefaultQualityChecker) checkMarketDataQuality(data *models.MarketData) *QualityReport {
	report := &QualityReport{
		DataType:  "market",
		Issues:    make([]QualityIssue, 0),
		Metrics:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
	
	// 检查必填字段完整性
	completeness := q.checkMarketDataCompleteness(data, report)
	
	// 检查数据一致性
	consistency := q.checkMarketDataConsistency(data, report)
	
	// 检查数据合理性
	reasonableness := q.checkMarketDataReasonableness(data, report)
	
	// 检查时效性
	timeliness := q.checkMarketDataTimeliness(data, report)
	
	// 计算综合质量分数
	report.QualityScore = (completeness + consistency + reasonableness + timeliness) / 4.0
	
	// 记录指标
	report.Metrics["completeness"] = completeness
	report.Metrics["consistency"] = consistency
	report.Metrics["reasonableness"] = reasonableness
	report.Metrics["timeliness"] = timeliness
	report.Metrics["total_fields"] = q.countMarketDataFields(data)
	report.Metrics["filled_fields"] = q.countFilledMarketDataFields(data)
	
	return report
}

// checkMarketDataCompleteness 检查行情数据完整性
func (q *DefaultQualityChecker) checkMarketDataCompleteness(data *models.MarketData, report *QualityReport) float64 {
	totalFields := 0
	filledFields := 0
	
	// 必填字段
	requiredFields := map[string]interface{}{
		"Symbol":    data.Symbol,
		"TradeDate": data.TradeDate,
		"Period":    data.Period,
	}
	
	for field, value := range requiredFields {
		totalFields++
		if q.isFieldFilled(value) {
			filledFields++
		} else {
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "completeness",
				Severity:    "high",
				Description: fmt.Sprintf("Required field %s is missing", field),
				Field:       field,
			})
		}
	}
	
	// 可选但重要的字段
	importantFields := map[string]interface{}{
		"OpenPrice":  data.OpenPrice,
		"HighPrice":  data.HighPrice,
		"LowPrice":   data.LowPrice,
		"ClosePrice": data.ClosePrice,
		"Volume":     data.Volume,
	}
	
	for field, value := range importantFields {
		totalFields++
		if q.isFieldFilled(value) {
			filledFields++
		} else {
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "completeness",
				Severity:    "medium",
				Description: fmt.Sprintf("Important field %s is missing", field),
				Field:       field,
			})
		}
	}
	
	if totalFields == 0 {
		return 0.0
	}
	
	return float64(filledFields) / float64(totalFields)
}

// checkMarketDataConsistency 检查行情数据一致性
func (q *DefaultQualityChecker) checkMarketDataConsistency(data *models.MarketData, report *QualityReport) float64 {
	issueCount := 0
	totalChecks := 0
	
	// 检查价格逻辑关系
	if data.HighPrice > 0 && data.LowPrice > 0 {
		totalChecks++
		if data.HighPrice < data.LowPrice {
			issueCount++
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "consistency",
				Severity:    "high",
				Description: "High price is lower than low price",
				Field:       "HighPrice,LowPrice",
			})
		}
	}
	
	if data.OpenPrice > 0 && data.HighPrice > 0 && data.LowPrice > 0 {
		totalChecks++
		if data.OpenPrice > data.HighPrice || data.OpenPrice < data.LowPrice {
			issueCount++
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "consistency",
				Severity:    "high",
				Description: "Open price is outside high-low range",
				Field:       "OpenPrice",
			})
		}
	}
	
	if data.ClosePrice > 0 && data.HighPrice > 0 && data.LowPrice > 0 {
		totalChecks++
		if data.ClosePrice > data.HighPrice || data.ClosePrice < data.LowPrice {
			issueCount++
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "consistency",
				Severity:    "high",
				Description: "Close price is outside high-low range",
				Field:       "ClosePrice",
			})
		}
	}
	
	// 检查成交量和成交额的关系
	if data.Volume > 0 && data.Amount > 0 && data.ClosePrice > 0 {
		totalChecks++
		expectedAmount := float64(data.Volume) * data.ClosePrice
		actualAmount := data.Amount
		
		// 允许10%的误差
		if math.Abs(expectedAmount-actualAmount)/expectedAmount > 0.1 {
			issueCount++
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "consistency",
				Severity:    "medium",
				Description: "Volume and amount relationship is inconsistent",
				Field:       "Volume,Amount",
			})
		}
	}
	
	if totalChecks == 0 {
		return 1.0
	}
	
	return float64(totalChecks-issueCount) / float64(totalChecks)
}

// checkMarketDataReasonableness 检查行情数据合理性
func (q *DefaultQualityChecker) checkMarketDataReasonableness(data *models.MarketData, report *QualityReport) float64 {
	issueCount := 0
	totalChecks := 0
	
	// 检查价格合理性
	prices := []*float64{&data.OpenPrice, &data.HighPrice, &data.LowPrice, &data.ClosePrice}
	for i, price := range prices {
		if price != nil {
			totalChecks++
			// 价格应该大于0且小于合理上限
			if *price <= 0 || *price > 10000 {
				issueCount++
				fieldNames := []string{"OpenPrice", "HighPrice", "LowPrice", "ClosePrice"}
				report.Issues = append(report.Issues, QualityIssue{
					Type:        "reasonableness",
					Severity:    "high",
					Description: fmt.Sprintf("Price %f is unreasonable", *price),
					Field:       fieldNames[i],
					Value:       fmt.Sprintf("%.4f", *price),
				})
			}
		}
	}
	
	// 检查成交量合理性
	if data.Volume > 0 {
		totalChecks++
		if data.Volume > 1000000000 {
			issueCount++
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "reasonableness",
				Severity:    "medium",
				Description: fmt.Sprintf("Volume %d is unreasonable", data.Volume),
				Field:       "Volume",
				Value:       fmt.Sprintf("%d", data.Volume),
			})
		}
	}
	
	// 检查涨跌幅合理性 - 暂时跳过，因为模型中没有ChangePercent字段
	// if data.ChangePercent != nil {
	//	totalChecks++
	//	// 涨跌幅通常在-20%到20%之间
	//	if *data.ChangePercent < -20 || *data.ChangePercent > 20 {
	//		issueCount++
	//		report.Issues = append(report.Issues, QualityIssue{
	//			Type:        "reasonableness",
	//			Severity:    "medium",
	//			Description: fmt.Sprintf("Change percent %.2f%% is unusual", *data.ChangePercent),
	//			Field:       "ChangePercent",
	//			Value:       fmt.Sprintf("%.2f%%", *data.ChangePercent),
	//		})
	//	}
	// }
	
	if totalChecks == 0 {
		return 1.0
	}
	
	return float64(totalChecks-issueCount) / float64(totalChecks)
}

// checkMarketDataTimeliness 检查行情数据时效性
func (q *DefaultQualityChecker) checkMarketDataTimeliness(data *models.MarketData, report *QualityReport) float64 {
	now := time.Now()
	tradeDate := data.TradeDate
	
	// 检查交易日期是否过于陈旧（超过1年）
	if now.Sub(tradeDate) > 365*24*time.Hour {
		report.Issues = append(report.Issues, QualityIssue{
			Type:        "timeliness",
			Severity:    "low",
			Description: "Data is more than 1 year old",
			Field:       "TradeDate",
			Value:       tradeDate.Format("2006-01-02"),
		})
		return 0.5
	}
	
	// 检查是否是未来日期
	if tradeDate.After(now) {
		report.Issues = append(report.Issues, QualityIssue{
			Type:        "timeliness",
			Severity:    "high",
			Description: "Trade date is in the future",
			Field:       "TradeDate",
			Value:       tradeDate.Format("2006-01-02"),
		})
		return 0.0
	}
	
	return 1.0
}

// checkFinancialDataQuality 检查财务数据质量
func (q *DefaultQualityChecker) checkFinancialDataQuality(data *models.FinancialData) *QualityReport {
	report := &QualityReport{
		DataType:  "financial",
		Issues:    make([]QualityIssue, 0),
		Metrics:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
	
	// 检查完整性
	completeness := q.checkFinancialDataCompleteness(data, report)
	
	// 检查一致性
	consistency := q.checkFinancialDataConsistency(data, report)
	
	// 检查合理性
	reasonableness := q.checkFinancialDataReasonableness(data, report)
	
	// 检查时效性
	timeliness := q.checkFinancialDataTimeliness(data, report)
	
	// 计算综合质量分数
	report.QualityScore = (completeness + consistency + reasonableness + timeliness) / 4.0
	
	// 记录指标
	report.Metrics["completeness"] = completeness
	report.Metrics["consistency"] = consistency
	report.Metrics["reasonableness"] = reasonableness
	report.Metrics["timeliness"] = timeliness
	
	return report
}

// checkFinancialDataCompleteness 检查财务数据完整性
func (q *DefaultQualityChecker) checkFinancialDataCompleteness(data *models.FinancialData, report *QualityReport) float64 {
	totalFields := 0
	filledFields := 0
	
	// 必填字段
	requiredFields := map[string]interface{}{
		"Symbol":     data.Symbol,
		"ReportDate": data.ReportDate,
		"ReportType": data.ReportType,
	}
	
	for field, value := range requiredFields {
		totalFields++
		if q.isFieldFilled(value) {
			filledFields++
		} else {
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "completeness",
				Severity:    "high",
				Description: fmt.Sprintf("Required field %s is missing", field),
				Field:       field,
			})
		}
	}
	
	// 重要财务指标
	importantFields := map[string]interface{}{
		"Revenue":     data.Revenue,
		"NetProfit":   data.NetProfit,
		"TotalAssets": data.TotalAssets,
	}
	
	for field, value := range importantFields {
		totalFields++
		if q.isFieldFilled(value) {
			filledFields++
		} else {
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "completeness",
				Severity:    "medium",
				Description: fmt.Sprintf("Important field %s is missing", field),
				Field:       field,
			})
		}
	}
	
	if totalFields == 0 {
		return 0.0
	}
	
	return float64(filledFields) / float64(totalFields)
}

// checkFinancialDataConsistency 检查财务数据一致性
func (q *DefaultQualityChecker) checkFinancialDataConsistency(data *models.FinancialData, report *QualityReport) float64 {
	issueCount := 0
	totalChecks := 0
	
	// 检查资产负债表平衡 - 暂时跳过，因为模型中没有TotalLiabilities字段
	// if data.TotalAssets != nil && data.TotalLiabilities != nil && data.TotalEquity != nil {
	//	totalChecks++
	//	balance := *data.TotalAssets - (*data.TotalLiabilities + *data.TotalEquity)
		// 允许1%的误差
		// if math.Abs(balance) > math.Abs(*data.TotalAssets)*0.01 {
		//	issueCount++
		//	report.Issues = append(report.Issues, QualityIssue{
		//		Type:        "consistency",
		//		Severity:    "high",
		//		Description: "Balance sheet does not balance",
		//		Field:       "TotalAssets,TotalLiabilities,TotalEquity",
		//	})
		// }
	// }
	
	// 检查财务比率的合理性
	if data.ROE != nil && data.ROA != nil {
		totalChecks++
		// ROE通常应该大于等于ROA
		if *data.ROE < *data.ROA-1.0 { // 允许1%的误差
			issueCount++
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "consistency",
				Severity:    "medium",
				Description: "ROE is significantly lower than ROA",
				Field:       "ROE,ROA",
			})
		}
	}
	
	if totalChecks == 0 {
		return 1.0
	}
	
	return float64(totalChecks-issueCount) / float64(totalChecks)
}

// checkFinancialDataReasonableness 检查财务数据合理性
func (q *DefaultQualityChecker) checkFinancialDataReasonableness(data *models.FinancialData, report *QualityReport) float64 {
	issueCount := 0
	totalChecks := 0
	
	// 检查财务比率的合理范围
	if data.ROE != nil {
		totalChecks++
		if *data.ROE < -100 || *data.ROE > 100 {
			issueCount++
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "reasonableness",
				Severity:    "medium",
				Description: fmt.Sprintf("ROE %.2f%% is outside reasonable range", *data.ROE),
				Field:       "ROE",
				Value:       fmt.Sprintf("%.2f%%", *data.ROE),
			})
		}
	}
	
	if data.CurrentRatio != nil {
		totalChecks++
		if *data.CurrentRatio < 0 || *data.CurrentRatio > 20 {
			issueCount++
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "reasonableness",
				Severity:    "medium",
				Description: fmt.Sprintf("Current ratio %.2f is outside reasonable range", *data.CurrentRatio),
				Field:       "CurrentRatio",
				Value:       fmt.Sprintf("%.2f", *data.CurrentRatio),
			})
		}
	}
	
	if totalChecks == 0 {
		return 1.0
	}
	
	return float64(totalChecks-issueCount) / float64(totalChecks)
}

// checkFinancialDataTimeliness 检查财务数据时效性
func (q *DefaultQualityChecker) checkFinancialDataTimeliness(data *models.FinancialData, report *QualityReport) float64 {
	now := time.Now()
	reportDate := data.ReportDate
	
	// 检查报告日期是否过于陈旧（超过3年）
	if now.Sub(reportDate) > 3*365*24*time.Hour {
		report.Issues = append(report.Issues, QualityIssue{
			Type:        "timeliness",
			Severity:    "low",
			Description: "Financial data is more than 3 years old",
			Field:       "ReportDate",
			Value:       reportDate.Format("2006-01-02"),
		})
		return 0.5
	}
	
	// 检查是否是未来日期
	if reportDate.After(now) {
		report.Issues = append(report.Issues, QualityIssue{
			Type:        "timeliness",
			Severity:    "high",
			Description: "Report date is in the future",
			Field:       "ReportDate",
			Value:       reportDate.Format("2006-01-02"),
		})
		return 0.0
	}
	
	return 1.0
}

// checkNewsDataQuality 检查新闻数据质量
func (q *DefaultQualityChecker) checkNewsDataQuality(data *models.NewsData) *QualityReport {
	report := &QualityReport{
		DataType:  "news",
		Issues:    make([]QualityIssue, 0),
		Metrics:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
	
	// 检查完整性
	completeness := q.checkNewsDataCompleteness(data, report)
	
	// 检查内容质量
	contentQuality := q.checkNewsContentQuality(data, report)
	
	// 检查时效性
	timeliness := q.checkNewsDataTimeliness(data, report)
	
	// 计算综合质量分数
	report.QualityScore = (completeness + contentQuality + timeliness) / 3.0
	
	// 记录指标
	report.Metrics["completeness"] = completeness
	report.Metrics["content_quality"] = contentQuality
	report.Metrics["timeliness"] = timeliness
	report.Metrics["title_length"] = len(data.Title)
	report.Metrics["content_length"] = len(data.Content)
	
	return report
}

// checkNewsDataCompleteness 检查新闻数据完整性
func (q *DefaultQualityChecker) checkNewsDataCompleteness(data *models.NewsData, report *QualityReport) float64 {
	totalFields := 0
	filledFields := 0
	
	// 必填字段
	requiredFields := map[string]interface{}{
		"Title":       data.Title,
		"Content":     data.Content,
		"PublishTime": data.PublishTime,
		"Source":      data.Source,
	}
	
	for field, value := range requiredFields {
		totalFields++
		if q.isFieldFilled(value) {
			filledFields++
		} else {
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "completeness",
				Severity:    "high",
				Description: fmt.Sprintf("Required field %s is missing", field),
				Field:       field,
			})
		}
	}
	
	if totalFields == 0 {
		return 0.0
	}
	
	return float64(filledFields) / float64(totalFields)
}

// checkNewsContentQuality 检查新闻内容质量
func (q *DefaultQualityChecker) checkNewsContentQuality(data *models.NewsData, report *QualityReport) float64 {
	issueCount := 0
	totalChecks := 0
	
	// 检查标题长度
	totalChecks++
	if len(data.Title) < 10 || len(data.Title) > 200 {
		issueCount++
		report.Issues = append(report.Issues, QualityIssue{
			Type:        "content_quality",
			Severity:    "medium",
			Description: fmt.Sprintf("Title length %d is not optimal (10-200 characters)", len(data.Title)),
			Field:       "Title",
		})
	}
	
	// 检查内容长度
	totalChecks++
	if len(data.Content) < 50 {
		issueCount++
		report.Issues = append(report.Issues, QualityIssue{
			Type:        "content_quality",
			Severity:    "medium",
			Description: fmt.Sprintf("Content is too short (%d characters)", len(data.Content)),
			Field:       "Content",
		})
	}
	
	// 检查内容是否包含HTML标签（应该已被清洗）
	totalChecks++
	if strings.Contains(data.Content, "<") && strings.Contains(data.Content, ">") {
		issueCount++
		report.Issues = append(report.Issues, QualityIssue{
			Type:        "content_quality",
			Severity:    "low",
			Description: "Content may contain HTML tags",
			Field:       "Content",
		})
	}
	
	// 检查情感分数范围
	if data.SentimentScore != nil {
		totalChecks++
		if *data.SentimentScore < -1.0 || *data.SentimentScore > 1.0 {
			issueCount++
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "content_quality",
				Severity:    "medium",
				Description: fmt.Sprintf("Sentiment score %.2f is outside valid range [-1, 1]", *data.SentimentScore),
				Field:       "SentimentScore",
			})
		}
	}
	
	if totalChecks == 0 {
		return 1.0
	}
	
	return float64(totalChecks-issueCount) / float64(totalChecks)
}

// checkNewsDataTimeliness 检查新闻数据时效性
func (q *DefaultQualityChecker) checkNewsDataTimeliness(data *models.NewsData, report *QualityReport) float64 {
	now := time.Now()
	publishTime := data.PublishTime
	
	// 检查发布时间是否过于陈旧（超过1年）
	if now.Sub(publishTime) > 365*24*time.Hour {
		report.Issues = append(report.Issues, QualityIssue{
			Type:        "timeliness",
			Severity:    "low",
			Description: "News is more than 1 year old",
			Field:       "PublishTime",
			Value:       publishTime.Format("2006-01-02 15:04:05"),
		})
		return 0.5
	}
	
	// 检查是否是未来时间
	if publishTime.After(now.Add(5 * time.Minute)) {
		report.Issues = append(report.Issues, QualityIssue{
			Type:        "timeliness",
			Severity:    "high",
			Description: "Publish time is in the future",
			Field:       "PublishTime",
			Value:       publishTime.Format("2006-01-02 15:04:05"),
		})
		return 0.0
	}
	
	return 1.0
}

// checkMacroDataQuality 检查宏观数据质量
func (q *DefaultQualityChecker) checkMacroDataQuality(data *models.MacroData) *QualityReport {
	report := &QualityReport{
		DataType:  "macro",
		Issues:    make([]QualityIssue, 0),
		Metrics:   make(map[string]interface{}),
		Timestamp: time.Now(),
	}
	
	// 检查完整性
	completeness := q.checkMacroDataCompleteness(data, report)
	
	// 检查合理性
	reasonableness := q.checkMacroDataReasonableness(data, report)
	
	// 检查时效性
	timeliness := q.checkMacroDataTimeliness(data, report)
	
	// 计算综合质量分数
	report.QualityScore = (completeness + reasonableness + timeliness) / 3.0
	
	// 记录指标
	report.Metrics["completeness"] = completeness
	report.Metrics["reasonableness"] = reasonableness
	report.Metrics["timeliness"] = timeliness
	
	return report
}

// checkMacroDataCompleteness 检查宏观数据完整性
func (q *DefaultQualityChecker) checkMacroDataCompleteness(data *models.MacroData, report *QualityReport) float64 {
	totalFields := 0
	filledFields := 0
	
	// 必填字段
	requiredFields := map[string]interface{}{
		"IndicatorCode": data.IndicatorCode,
		"IndicatorName": data.IndicatorName,
		"DataDate":      data.DataDate,
		"Value":         data.Value,
	}
	
	for field, value := range requiredFields {
		totalFields++
		if q.isFieldFilled(value) {
			filledFields++
		} else {
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "completeness",
				Severity:    "high",
				Description: fmt.Sprintf("Required field %s is missing", field),
				Field:       field,
			})
		}
	}
	
	if totalFields == 0 {
		return 0.0
	}
	
	return float64(filledFields) / float64(totalFields)
}

// checkMacroDataReasonableness 检查宏观数据合理性
func (q *DefaultQualityChecker) checkMacroDataReasonableness(data *models.MacroData, report *QualityReport) float64 {
	if data.Value == 0 {
		return 1.0
	}

	issueCount := 0
	totalChecks := 1

	// 根据指标类型检查数值合理性
	indicatorCode := strings.ToLower(data.IndicatorCode)
	value := data.Value
	
	switch {
	case strings.Contains(indicatorCode, "cpi"):
		// CPI通常在-10%到50%之间
		if value < -10 || value > 50 {
			issueCount++
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "reasonableness",
				Severity:    "medium",
				Description: fmt.Sprintf("CPI value %.2f is outside reasonable range [-10, 50]", value),
				Field:       "Value",
				Value:       fmt.Sprintf("%.2f", value),
			})
		}
	case strings.Contains(indicatorCode, "gdp"):
		// GDP增长率通常在-20%到30%之间
		if value < -20 || value > 30 {
			issueCount++
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "reasonableness",
				Severity:    "medium",
				Description: fmt.Sprintf("GDP value %.2f is outside reasonable range [-20, 30]", value),
				Field:       "Value",
				Value:       fmt.Sprintf("%.2f", value),
			})
		}
	case strings.Contains(indicatorCode, "rate"):
		// 利率通常在-5%到50%之间
		if value < -5 || value > 50 {
			issueCount++
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "reasonableness",
				Severity:    "medium",
				Description: fmt.Sprintf("Interest rate value %.2f is outside reasonable range [-5, 50]", value),
				Field:       "Value",
				Value:       fmt.Sprintf("%.2f", value),
			})
		}
	default:
		// 通用检查：数值不应该是极端值
		if math.IsInf(value, 0) || math.IsNaN(value) {
			issueCount++
			report.Issues = append(report.Issues, QualityIssue{
				Type:        "reasonableness",
				Severity:    "high",
				Description: "Value is infinite or NaN",
				Field:       "Value",
				Value:       fmt.Sprintf("%f", value),
			})
		}
	}
	
	return float64(totalChecks-issueCount) / float64(totalChecks)
}

// checkMacroDataTimeliness 检查宏观数据时效性
func (q *DefaultQualityChecker) checkMacroDataTimeliness(data *models.MacroData, report *QualityReport) float64 {
	now := time.Now()
	dataDate := data.DataDate
	
	// 检查数据日期是否过于陈旧（超过5年）
	if now.Sub(dataDate) > 5*365*24*time.Hour {
		report.Issues = append(report.Issues, QualityIssue{
			Type:        "timeliness",
			Severity:    "low",
			Description: "Macro data is more than 5 years old",
			Field:       "DataDate",
			Value:       dataDate.Format("2006-01-02"),
		})
		return 0.5
	}
	
	// 检查是否是未来日期
	if dataDate.After(now) {
		report.Issues = append(report.Issues, QualityIssue{
			Type:        "timeliness",
			Severity:    "high",
			Description: "Data date is in the future",
			Field:       "DataDate",
			Value:       dataDate.Format("2006-01-02"),
		})
		return 0.0
	}
	
	return 1.0
}

// isFieldFilled 检查字段是否已填充
func (q *DefaultQualityChecker) isFieldFilled(value interface{}) bool {
	if value == nil {
		return false
	}
	
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return strings.TrimSpace(v.String()) != ""
	case reflect.Ptr:
		return !v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.Len() > 0
	default:
		return !v.IsZero()
	}
}

// countMarketDataFields 统计行情数据字段总数
func (q *DefaultQualityChecker) countMarketDataFields(data *models.MarketData) int {
	v := reflect.ValueOf(data).Elem()
	return v.NumField()
}

// countFilledMarketDataFields 统计已填充的行情数据字段数
func (q *DefaultQualityChecker) countFilledMarketDataFields(data *models.MarketData) int {
	v := reflect.ValueOf(data).Elem()
	count := 0
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if q.isFieldFilled(field.Interface()) {
			count++
		}
	}
	
	return count
}