package processing

import (
	"fmt"
	"math"
	"strings"
	"time"

	"data-collection-system/model"
)

// QualityChecker 数据质量检查器
type QualityChecker struct {
	// 质量阈值配置
	config QualityConfig
}

// QualityConfig 质量检查配置
type QualityConfig struct {
	// 新闻数据质量阈值
	NewsMinTitleLength    int     // 标题最小长度
	NewsMinContentLength  int     // 内容最小长度
	NewsMaxDuplicateRate  float64 // 最大重复率
	NewsMinImportanceLevel int    // 最小重要性级别
	
	// 行情数据质量阈值
	MarketMaxPriceChange  float64 // 最大价格变动幅度
	MarketMinVolume       int64   // 最小成交量
	MarketMaxSpread       float64 // 最大价差比例
	
	// 财务数据质量阈值
	FinancialMaxROEChange float64 // ROE最大变动幅度
	FinancialMinAssets    float64 // 最小资产规模
}

// NewQualityChecker 创建数据质量检查器实例
func NewQualityChecker() *QualityChecker {
	return &QualityChecker{
		config: QualityConfig{
			// 新闻数据默认阈值
			NewsMinTitleLength:    10,
			NewsMinContentLength:  50,
			NewsMaxDuplicateRate:  0.3, // 30%
			NewsMinImportanceLevel: 1,
			
			// 行情数据默认阈值
			MarketMaxPriceChange:  0.2, // 20%
			MarketMinVolume:       0,
			MarketMaxSpread:       0.1, // 10%
			
			// 财务数据默认阈值
			FinancialMaxROEChange: 1.0, // 100%
			FinancialMinAssets:    0,
		},
	}
}

// QualityReport 质量检查报告
type QualityReport struct {
	DataType        string             `json:"data_type"`
	TotalCount      int                `json:"total_count"`
	ValidCount      int                `json:"valid_count"`
	InvalidCount    int                `json:"invalid_count"`
	QualityScore    float64            `json:"quality_score"`
	Issues          []QualityIssue     `json:"issues"`
	Recommendations []string           `json:"recommendations"`
	CheckTime       time.Time          `json:"check_time"`
}

// QualityIssue 质量问题
type QualityIssue struct {
	Type        string      `json:"type"`
	Severity    string      `json:"severity"` // high, medium, low
	Description string      `json:"description"`
	Count       int         `json:"count"`
	Examples    []string    `json:"examples,omitempty"`
	DataID      interface{} `json:"data_id,omitempty"`
}

// CheckNewsDataQuality 检查新闻数据质量
func (qc *QualityChecker) CheckNewsDataQuality(newsList []*model.NewsData) *QualityReport {
	report := &QualityReport{
		DataType:   "news",
		TotalCount: len(newsList),
		CheckTime:  time.Now(),
		Issues:     make([]QualityIssue, 0),
	}

	if len(newsList) == 0 {
		report.QualityScore = 0
		return report
	}

	validCount := 0
	issueStats := make(map[string]*QualityIssue)

	for _, news := range newsList {
		if news == nil {
			continue
		}

		isValid := true
		issues := qc.checkSingleNewsQuality(news)

		for _, issue := range issues {
			isValid = false
			key := fmt.Sprintf("%s_%s", issue.Type, issue.Severity)
			if existing, exists := issueStats[key]; exists {
				existing.Count++
				if len(existing.Examples) < 3 {
					existing.Examples = append(existing.Examples, issue.Description)
				}
			} else {
				issue.Count = 1
				issue.Examples = []string{issue.Description}
				issueStats[key] = &issue
			}
		}

		if isValid {
			validCount++
		}
	}

	report.ValidCount = validCount
	report.InvalidCount = report.TotalCount - validCount
	report.QualityScore = float64(validCount) / float64(report.TotalCount) * 100

	// 转换问题统计到报告
	for _, issue := range issueStats {
		report.Issues = append(report.Issues, *issue)
	}

	// 生成建议
	report.Recommendations = qc.generateNewsRecommendations(report)

	return report
}

// checkSingleNewsQuality 检查单条新闻数据质量
func (qc *QualityChecker) checkSingleNewsQuality(news *model.NewsData) []QualityIssue {
	issues := make([]QualityIssue, 0)

	// 检查标题质量
	if len(strings.TrimSpace(news.Title)) < qc.config.NewsMinTitleLength {
		issues = append(issues, QualityIssue{
			Type:        "title_too_short",
			Severity:    "high",
			Description: fmt.Sprintf("标题过短: %d 字符 (最小要求: %d)", len(news.Title), qc.config.NewsMinTitleLength),
			DataID:      news.ID,
		})
	}

	// 检查内容质量
	if len(strings.TrimSpace(news.Content)) < qc.config.NewsMinContentLength {
		issues = append(issues, QualityIssue{
			Type:        "content_too_short",
			Severity:    "high",
			Description: fmt.Sprintf("内容过短: %d 字符 (最小要求: %d)", len(news.Content), qc.config.NewsMinContentLength),
			DataID:      news.ID,
		})
	}

	// 检查重要性级别
		if news.ImportanceLevel < int8(qc.config.NewsMinImportanceLevel) {
		issues = append(issues, QualityIssue{
			Type:        "low_importance",
			Severity:    "medium",
			Description: fmt.Sprintf("重要性级别过低: %d (最小要求: %d)", news.ImportanceLevel, qc.config.NewsMinImportanceLevel),
			DataID:      news.ID,
		})
	}

	// 检查来源质量
	if strings.TrimSpace(news.Source) == "" {
		issues = append(issues, QualityIssue{
			Type:        "missing_source",
			Severity:    "medium",
			Description: "缺少新闻来源",
			DataID:      news.ID,
		})
	}

	// 检查发布时间
	if news.PublishTime.IsZero() {
		issues = append(issues, QualityIssue{
			Type:        "missing_publish_time",
			Severity:    "high",
			Description: "缺少发布时间",
			DataID:      news.ID,
		})
	} else if news.PublishTime.After(time.Now()) {
		issues = append(issues, QualityIssue{
			Type:        "future_publish_time",
			Severity:    "high",
			Description: "发布时间为未来时间",
			DataID:      news.ID,
		})
	}

	// 检查情感分析结果
	if news.SentimentScore != nil {
		if *news.SentimentScore < -1 || *news.SentimentScore > 1 {
			issues = append(issues, QualityIssue{
				Type:        "invalid_sentiment",
				Severity:    "medium",
				Description: fmt.Sprintf("情感分数超出范围: %.2f (应在-1到1之间)", *news.SentimentScore),
				DataID:      news.ID,
			})
		}
	}

	// 检查关联股票
	if len(news.RelatedStocks) > 20 {
		issues = append(issues, QualityIssue{
			Type:        "too_many_related_stocks",
			Severity:    "low",
			Description: fmt.Sprintf("关联股票过多: %d (建议不超过20个)", len(news.RelatedStocks)),
			DataID:      news.ID,
		})
	}

	return issues
}

// CheckMarketDataQuality 检查行情数据质量
func (qc *QualityChecker) CheckMarketDataQuality(dataList []*model.MarketData) *QualityReport {
	report := &QualityReport{
		DataType:   "market",
		TotalCount: len(dataList),
		CheckTime:  time.Now(),
		Issues:     make([]QualityIssue, 0),
	}

	if len(dataList) == 0 {
		report.QualityScore = 0
		return report
	}

	validCount := 0
	issueStats := make(map[string]*QualityIssue)

	for _, data := range dataList {
		if data == nil {
			continue
		}

		isValid := true
		issues := qc.checkSingleMarketDataQuality(data)

		for _, issue := range issues {
			isValid = false
			key := fmt.Sprintf("%s_%s", issue.Type, issue.Severity)
			if existing, exists := issueStats[key]; exists {
				existing.Count++
			} else {
				issue.Count = 1
				issueStats[key] = &issue
			}
		}

		if isValid {
			validCount++
		}
	}

	report.ValidCount = validCount
	report.InvalidCount = report.TotalCount - validCount
	report.QualityScore = float64(validCount) / float64(report.TotalCount) * 100

	// 转换问题统计到报告
	for _, issue := range issueStats {
		report.Issues = append(report.Issues, *issue)
	}

	// 生成建议
	report.Recommendations = qc.generateMarketRecommendations(report)

	return report
}

// checkSingleMarketDataQuality 检查单条行情数据质量
func (qc *QualityChecker) checkSingleMarketDataQuality(data *model.MarketData) []QualityIssue {
	issues := make([]QualityIssue, 0)

	// 检查价格逻辑
	if data.HighPrice < data.LowPrice {
		issues = append(issues, QualityIssue{
			Type:        "invalid_price_range",
			Severity:    "high",
			Description: fmt.Sprintf("最高价(%.2f)低于最低价(%.2f)", data.HighPrice, data.LowPrice),
			DataID:      fmt.Sprintf("%s_%s", data.Symbol, data.TradeDate.Format("2006-01-02")),
		})
	}

	// 检查开盘价是否在合理范围内
	if data.OpenPrice < data.LowPrice || data.OpenPrice > data.HighPrice {
		issues = append(issues, QualityIssue{
			Type:        "invalid_open_price",
			Severity:    "high",
			Description: fmt.Sprintf("开盘价(%.2f)超出最高最低价范围[%.2f, %.2f]", data.OpenPrice, data.LowPrice, data.HighPrice),
			DataID:      fmt.Sprintf("%s_%s", data.Symbol, data.TradeDate.Format("2006-01-02")),
		})
	}

	// 检查收盘价是否在合理范围内
	if data.ClosePrice < data.LowPrice || data.ClosePrice > data.HighPrice {
		issues = append(issues, QualityIssue{
			Type:        "invalid_close_price",
			Severity:    "high",
			Description: fmt.Sprintf("收盘价(%.2f)超出最高最低价范围[%.2f, %.2f]", data.ClosePrice, data.LowPrice, data.HighPrice),
			DataID:      fmt.Sprintf("%s_%s", data.Symbol, data.TradeDate.Format("2006-01-02")),
		})
	}

	// 检查价格变动幅度
	if data.OpenPrice > 0 {
		priceChange := math.Abs(data.ClosePrice-data.OpenPrice) / data.OpenPrice
		if priceChange > qc.config.MarketMaxPriceChange {
			issues = append(issues, QualityIssue{
				Type:        "excessive_price_change",
				Severity:    "medium",
				Description: fmt.Sprintf("价格变动过大: %.2f%% (阈值: %.2f%%)", priceChange*100, qc.config.MarketMaxPriceChange*100),
				DataID:      fmt.Sprintf("%s_%s", data.Symbol, data.TradeDate.Format("2006-01-02")),
			})
		}
	}

	// 检查成交量
	if data.Volume < qc.config.MarketMinVolume {
		issues = append(issues, QualityIssue{
			Type:        "low_volume",
			Severity:    "low",
			Description: fmt.Sprintf("成交量过低: %d (最小要求: %d)", data.Volume, qc.config.MarketMinVolume),
			DataID:      fmt.Sprintf("%s_%s", data.Symbol, data.TradeDate.Format("2006-01-02")),
		})
	}

	// 检查价差比例
	if data.LowPrice > 0 {
		spread := (data.HighPrice - data.LowPrice) / data.LowPrice
		if spread > qc.config.MarketMaxSpread {
			issues = append(issues, QualityIssue{
				Type:        "excessive_spread",
				Severity:    "medium",
				Description: fmt.Sprintf("价差过大: %.2f%% (阈值: %.2f%%)", spread*100, qc.config.MarketMaxSpread*100),
				DataID:      fmt.Sprintf("%s_%s", data.Symbol, data.TradeDate.Format("2006-01-02")),
			})
		}
	}

	return issues
}

// CheckFinancialDataQuality 检查财务数据质量
func (qc *QualityChecker) CheckFinancialDataQuality(dataList []*model.FinancialData) *QualityReport {
	report := &QualityReport{
		DataType:   "financial",
		TotalCount: len(dataList),
		CheckTime:  time.Now(),
		Issues:     make([]QualityIssue, 0),
	}

	if len(dataList) == 0 {
		report.QualityScore = 0
		return report
	}

	validCount := 0
	issueStats := make(map[string]*QualityIssue)

	for _, data := range dataList {
		if data == nil {
			continue
		}

		isValid := true
		issues := qc.checkSingleFinancialDataQuality(data)

		for _, issue := range issues {
			isValid = false
			key := fmt.Sprintf("%s_%s", issue.Type, issue.Severity)
			if existing, exists := issueStats[key]; exists {
				existing.Count++
			} else {
				issue.Count = 1
				issueStats[key] = &issue
			}
		}

		if isValid {
			validCount++
		}
	}

	report.ValidCount = validCount
	report.InvalidCount = report.TotalCount - validCount
	report.QualityScore = float64(validCount) / float64(report.TotalCount) * 100

	// 转换问题统计到报告
	for _, issue := range issueStats {
		report.Issues = append(report.Issues, *issue)
	}

	// 生成建议
	report.Recommendations = qc.generateFinancialRecommendations(report)

	return report
}

// checkSingleFinancialDataQuality 检查单条财务数据质量
func (qc *QualityChecker) checkSingleFinancialDataQuality(data *model.FinancialData) []QualityIssue {
	issues := make([]QualityIssue, 0)

	// 检查资产负债逻辑
	if data.TotalAssets != nil && data.TotalEquity != nil {
		if *data.TotalEquity > *data.TotalAssets {
			issues = append(issues, QualityIssue{
				Type:        "invalid_equity_assets_ratio",
				Severity:    "high",
				Description: fmt.Sprintf("总权益(%.2f)大于总资产(%.2f)", *data.TotalEquity, *data.TotalAssets),
				DataID:      fmt.Sprintf("%s_%s_%s", data.Symbol, data.ReportDate.Format("2006-01-02"), data.ReportType),
			})
		}
	}

	// 检查ROE合理性
	if data.ROE != nil {
		if math.Abs(*data.ROE) > qc.config.FinancialMaxROEChange {
			issues = append(issues, QualityIssue{
				Type:        "extreme_roe",
				Severity:    "medium",
				Description: fmt.Sprintf("ROE异常: %.2f%% (阈值: ±%.2f%%)", *data.ROE*100, qc.config.FinancialMaxROEChange*100),
				DataID:      fmt.Sprintf("%s_%s_%s", data.Symbol, data.ReportDate.Format("2006-01-02"), data.ReportType),
			})
		}
	}

	// 检查资产规模
	if data.TotalAssets != nil {
		if *data.TotalAssets < qc.config.FinancialMinAssets {
			issues = append(issues, QualityIssue{
				Type:        "low_assets",
				Severity:    "low",
				Description: fmt.Sprintf("资产规模过小: %.2f (最小要求: %.2f)", *data.TotalAssets, qc.config.FinancialMinAssets),
				DataID:      fmt.Sprintf("%s_%s_%s", data.Symbol, data.ReportDate.Format("2006-01-02"), data.ReportType),
			})
		}
	}

	// 检查比率指标范围
	if data.CurrentRatio != nil && *data.CurrentRatio < 0 {
		issues = append(issues, QualityIssue{
			Type:        "negative_current_ratio",
			Severity:    "high",
			Description: fmt.Sprintf("流动比率为负: %.2f", *data.CurrentRatio),
			DataID:      fmt.Sprintf("%s_%s_%s", data.Symbol, data.ReportDate.Format("2006-01-02"), data.ReportType),
		})
	}

	return issues
}

// generateNewsRecommendations 生成新闻数据质量改进建议
func (qc *QualityChecker) generateNewsRecommendations(report *QualityReport) []string {
	recommendations := make([]string, 0)

	for _, issue := range report.Issues {
		switch issue.Type {
		case "title_too_short":
			recommendations = append(recommendations, "建议过滤标题过短的新闻，或从其他数据源补充完整标题")
		case "content_too_short":
			recommendations = append(recommendations, "建议过滤内容过短的新闻，或从原始链接抓取完整内容")
		case "missing_source":
			recommendations = append(recommendations, "建议在数据采集时确保记录新闻来源信息")
		case "missing_publish_time":
			recommendations = append(recommendations, "建议在数据采集时确保获取准确的发布时间")
		case "invalid_sentiment":
			recommendations = append(recommendations, "建议检查情感分析算法，确保输出在有效范围内")
		}
	}

	if report.QualityScore < 80 {
		recommendations = append(recommendations, "整体数据质量偏低，建议优化数据采集和清洗流程")
	}

	return recommendations
}

// generateMarketRecommendations 生成行情数据质量改进建议
func (qc *QualityChecker) generateMarketRecommendations(report *QualityReport) []string {
	recommendations := make([]string, 0)

	for _, issue := range report.Issues {
		switch issue.Type {
		case "invalid_price_range":
			recommendations = append(recommendations, "建议检查数据源，确保价格数据的逻辑一致性")
		case "excessive_price_change":
			recommendations = append(recommendations, "建议对异常价格变动进行人工审核，可能存在停牌复牌等特殊情况")
		case "low_volume":
			recommendations = append(recommendations, "建议关注低成交量股票，可能存在流动性风险")
		}
	}

	return recommendations
}

// generateFinancialRecommendations 生成财务数据质量改进建议
func (qc *QualityChecker) generateFinancialRecommendations(report *QualityReport) []string {
	recommendations := make([]string, 0)

	for _, issue := range report.Issues {
		switch issue.Type {
		case "invalid_equity_assets_ratio":
			recommendations = append(recommendations, "建议检查财务数据源，确保资产负债表数据的逻辑一致性")
		case "extreme_roe":
			recommendations = append(recommendations, "建议对异常ROE数据进行人工审核，可能存在特殊会计处理")
		}
	}

	return recommendations
}

// SetQualityConfig 设置质量检查配置
func (qc *QualityChecker) SetQualityConfig(config QualityConfig) {
	qc.config = config
}

// GetQualityConfig 获取当前质量检查配置
func (qc *QualityChecker) GetQualityConfig() QualityConfig {
	return qc.config
}