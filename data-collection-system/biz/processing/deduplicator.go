package processing

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"

	"data-collection-system/model"
)

// DataDeduplicator 数据去重器
type DataDeduplicator struct {
	// 可以添加缓存或数据库连接用于去重检查
}

// NewDataDeduplicator 创建数据去重器实例
func NewDataDeduplicator() *DataDeduplicator {
	return &DataDeduplicator{}
}

// DeduplicateNewsData 新闻数据去重
func (d *DataDeduplicator) DeduplicateNewsData(newsList []*model.NewsData) []*model.NewsData {
	if len(newsList) == 0 {
		return newsList
	}

	// 使用map进行去重，key为新闻的唯一标识
	uniqueNews := make(map[string]*model.NewsData)
	duplicateCount := 0

	for _, news := range newsList {
		if news == nil {
			continue
		}

		// 生成新闻的唯一标识
		uniqueKey := d.generateNewsUniqueKey(news)

		// 检查是否已存在
		if existing, exists := uniqueNews[uniqueKey]; exists {
			// 如果存在重复，选择更完整或更新的数据
			if d.shouldReplaceNews(existing, news) {
				uniqueNews[uniqueKey] = news
			}
			duplicateCount++
		} else {
			uniqueNews[uniqueKey] = news
		}
	}

	// 转换回切片
	result := make([]*model.NewsData, 0, len(uniqueNews))
	for _, news := range uniqueNews {
		result = append(result, news)
	}

	// 按发布时间排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].PublishTime.After(result[j].PublishTime)
	})

	return result
}

// generateNewsUniqueKey 生成新闻的唯一标识
func (d *DataDeduplicator) generateNewsUniqueKey(news *model.NewsData) string {
	// 方法1: 基于标题和来源的相似度
	titleKey := d.normalizeTitle(news.Title)
	sourceKey := strings.ToLower(strings.TrimSpace(news.Source))
	
	// 方法2: 基于内容的哈希值（用于精确去重）
	contentHash := d.generateContentHash(news.Content)
	
	// 使用标题+来源+内容哈希的组合生成唯一键
	return fmt.Sprintf("title_source_content:%s:%s:%s", titleKey, sourceKey, contentHash)
}

// normalizeTitle 标准化标题用于去重比较
func (d *DataDeduplicator) normalizeTitle(title string) string {
	// 转换为小写
	normalized := strings.ToLower(title)
	
	// 移除多余的空白字符
	normalized = strings.TrimSpace(normalized)
	normalized = strings.Join(strings.Fields(normalized), " ")
	
	// 移除常见的标点符号
	normalized = strings.ReplaceAll(normalized, "，", ",")
	normalized = strings.ReplaceAll(normalized, "。", ".")
	normalized = strings.ReplaceAll(normalized, "！", "!")
	normalized = strings.ReplaceAll(normalized, "？", "?")
	normalized = strings.ReplaceAll(normalized, "：", ":")
	normalized = strings.ReplaceAll(normalized, "；", ";")
	
	return normalized
}

// generateContentHash 生成内容的哈希值
func (d *DataDeduplicator) generateContentHash(content string) string {
	// 标准化内容
	normalizedContent := strings.ToLower(strings.TrimSpace(content))
	normalizedContent = strings.Join(strings.Fields(normalizedContent), " ")
	
	// 生成MD5哈希
	hash := md5.Sum([]byte(normalizedContent))
	return fmt.Sprintf("%x", hash)
}

// shouldReplaceNews 判断是否应该替换现有的新闻数据
func (d *DataDeduplicator) shouldReplaceNews(existing, new *model.NewsData) bool {
	// 优先选择更完整的数据
	existingScore := d.calculateNewsCompletenessScore(existing)
	newScore := d.calculateNewsCompletenessScore(new)
	
	if newScore > existingScore {
		return true
	}
	
	if newScore < existingScore {
		return false
	}
	
	// 如果完整度相同，选择更新的数据
	return new.CreatedAt.After(existing.CreatedAt)
}

// calculateNewsCompletenessScore 计算新闻数据的完整度分数
func (d *DataDeduplicator) calculateNewsCompletenessScore(news *model.NewsData) int {
	score := 0
	
	// 基础字段
	if news.Title != "" {
		score += 10
	}
	if news.Content != "" {
		score += 20
	}
	if news.Source != "" {
		score += 5
	}
	// URL字段在当前模型中不存在，跳过
	
	// 增强字段
	if news.SentimentScore != nil {
		score += 10
	}
	if len(news.RelatedStocks) > 0 {
		score += 15
	}
	if len(news.Content) > 500 {
		score += 10
	}
	if news.ImportanceLevel > 0 {
		score += 5
	}
	
	// 内容长度加分
	if len(news.Content) > 500 {
		score += 5
	}
	if len(news.Content) > 1000 {
		score += 5
	}
	
	return score
}

// DeduplicateMarketData 行情数据去重
func (d *DataDeduplicator) DeduplicateMarketData(dataList []*model.MarketData) []*model.MarketData {
	if len(dataList) == 0 {
		return dataList
	}

	// 使用map进行去重，key为股票代码+交易日期
	uniqueData := make(map[string]*model.MarketData)

	for _, data := range dataList {
		if data == nil {
			continue
		}

		// 生成唯一键：股票代码+交易日期
		uniqueKey := fmt.Sprintf("%s:%s", data.Symbol, data.TradeDate.Format("2006-01-02"))

		// 检查是否已存在
		if existing, exists := uniqueData[uniqueKey]; exists {
			// 如果存在重复，选择更新的数据
		if data.CreatedAt.After(existing.CreatedAt) {
			uniqueData[uniqueKey] = data
		}
		} else {
			uniqueData[uniqueKey] = data
		}
	}

	// 转换回切片
	result := make([]*model.MarketData, 0, len(uniqueData))
	for _, data := range uniqueData {
		result = append(result, data)
	}

	// 按交易日期排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].TradeDate.After(result[j].TradeDate)
	})

	return result
}

// DeduplicateFinancialData 财务数据去重
func (d *DataDeduplicator) DeduplicateFinancialData(dataList []*model.FinancialData) []*model.FinancialData {
	if len(dataList) == 0 {
		return dataList
	}

	// 使用map进行去重，key为股票代码+报告期+报告类型
	uniqueData := make(map[string]*model.FinancialData)

	for _, data := range dataList {
		if data == nil {
			continue
		}

		// 生成唯一键：股票代码+报告期+报告类型
		uniqueKey := fmt.Sprintf("%s:%s:%s", 
			data.Symbol, 
			data.ReportDate.Format("2006-01-02"), 
			data.ReportType)

		// 检查是否已存在
		if existing, exists := uniqueData[uniqueKey]; exists {
			// 如果存在重复，选择更完整的数据
			if d.shouldReplaceFinancialData(existing, data) {
				uniqueData[uniqueKey] = data
			}
		} else {
			uniqueData[uniqueKey] = data
		}
	}

	// 转换回切片
	result := make([]*model.FinancialData, 0, len(uniqueData))
	for _, data := range uniqueData {
		result = append(result, data)
	}

	// 按报告期排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].ReportDate.After(result[j].ReportDate)
	})

	return result
}

// shouldReplaceFinancialData 判断是否应该替换现有的财务数据
func (d *DataDeduplicator) shouldReplaceFinancialData(existing, new *model.FinancialData) bool {
	// 优先选择更完整的数据
	existingScore := d.calculateFinancialCompletenessScore(existing)
	newScore := d.calculateFinancialCompletenessScore(new)
	
	if newScore > existingScore {
		return true
	}
	
	if newScore < existingScore {
		return false
	}
	
	// 如果完整度相同，选择更新的数据
	return new.CreatedAt.After(existing.CreatedAt)
}

// calculateFinancialCompletenessScore 计算财务数据的完整度分数
func (d *DataDeduplicator) calculateFinancialCompletenessScore(data *model.FinancialData) int {
	score := 0
	
	// 基础字段
	if data.Symbol != "" {
		score += 5
	}
	if !data.ReportDate.IsZero() {
		score += 5
	}
	if data.ReportType != "" {
		score += 5
	}
	
	// 财务指标字段（每个非空字段加分）
	if data.Revenue != nil {
		score += 10
	}
	if data.NetProfit != nil {
		score += 10
	}
	if data.TotalAssets != nil {
		score += 5
	}
	if data.TotalEquity != nil {
		score += 5
	}
	if data.ROE != nil {
		score += 3
	}
	if data.ROA != nil {
		score += 3
	}
	if data.GrossMargin != nil {
		score += 2
	}
	if data.NetMargin != nil {
		score += 2
	}
	if data.CurrentRatio != nil {
		score += 2
	}
	
	return score
}

// FindDuplicateNews 查找重复的新闻数据（用于分析）
func (d *DataDeduplicator) FindDuplicateNews(newsList []*model.NewsData) map[string][]*model.NewsData {
	duplicates := make(map[string][]*model.NewsData)
	
	for _, news := range newsList {
		if news == nil {
			continue
		}
		
		uniqueKey := d.generateNewsUniqueKey(news)
		duplicates[uniqueKey] = append(duplicates[uniqueKey], news)
	}
	
	// 只返回有重复的数据
	result := make(map[string][]*model.NewsData)
	for key, items := range duplicates {
		if len(items) > 1 {
			result[key] = items
		}
	}
	
	return result
}

// GetDeduplicationStats 获取去重统计信息
type DeduplicationStats struct {
	OriginalCount   int
	UniqueCount     int
	DuplicateCount  int
	DuplicationRate float64
}

// GetNewsDeduplicationStats 获取新闻去重统计
func (d *DataDeduplicator) GetNewsDeduplicationStats(original, deduplicated []*model.NewsData) DeduplicationStats {
	originalCount := len(original)
	uniqueCount := len(deduplicated)
	duplicateCount := originalCount - uniqueCount
	
	duplicationRate := 0.0
	if originalCount > 0 {
		duplicationRate = float64(duplicateCount) / float64(originalCount) * 100
	}
	
	return DeduplicationStats{
		OriginalCount:   originalCount,
		UniqueCount:     uniqueCount,
		DuplicateCount:  duplicateCount,
		DuplicationRate: duplicationRate,
	}
}