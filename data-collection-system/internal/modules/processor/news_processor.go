package processor

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"data-collection-system/internal/models"
	"data-collection-system/pkg/logger"
)

// NewsProcessor 新闻数据处理器
type NewsProcessor struct {
	stockCodePattern *regexp.Regexp
	companyNames     map[string]string // 公司名称到股票代码的映射
	industryKeywords map[string][]string // 行业关键词映射
}

// NewNewsProcessor 创建新闻处理器
func NewNewsProcessor() *NewsProcessor {
	// 编译股票代码正则表达式
	stockCodePattern := regexp.MustCompile(`\b(\d{6}|[A-Z]{2,4})\b`)
	
	// 初始化公司名称映射（实际应用中应从数据库加载）
	companyNames := map[string]string{
		"中国平安":   "000001",
		"万科A":    "000002",
		"招商银行":   "600036",
		"贵州茅台":   "600519",
		"五粮液":    "000858",
		"格力电器":   "000651",
		"美的集团":   "000333",
		"腾讯控股":   "00700",
		"阿里巴巴":   "09988",
		"比亚迪":    "002594",
	}
	
	// 初始化行业关键词映射
	industryKeywords := map[string][]string{
		"银行": {"银行", "金融", "贷款", "存款", "利率", "央行", "货币政策"},
		"地产": {"房地产", "楼市", "房价", "土地", "开发商", "住宅", "商业地产"},
		"科技": {"科技", "互联网", "人工智能", "5G", "芯片", "半导体", "软件", "云计算"},
		"医药": {"医药", "生物", "疫苗", "药品", "医疗", "健康", "制药"},
		"汽车": {"汽车", "新能源车", "电动车", "燃油车", "自动驾驶", "车企"},
		"消费": {"消费", "零售", "电商", "品牌", "快消", "食品", "饮料"},
		"能源": {"石油", "天然气", "煤炭", "电力", "新能源", "光伏", "风电"},
		"制造": {"制造", "工业", "机械", "设备", "材料", "化工", "钢铁"},
	}
	
	return &NewsProcessor{
		stockCodePattern: stockCodePattern,
		companyNames:     companyNames,
		industryKeywords: industryKeywords,
	}
}

// ProcessNews 处理新闻数据
func (np *NewsProcessor) ProcessNews(ctx context.Context, news *models.NewsData) error {
	// 1. 内容预处理
	if err := np.preprocessContent(news); err != nil {
		logger.Error("Failed to preprocess news content", "error", err, "news_id", news.ID)
		return fmt.Errorf("preprocess content failed: %w", err)
	}
	
	// 2. 实体识别
	if err := np.extractEntities(news); err != nil {
		logger.Error("Failed to extract entities", "error", err, "news_id", news.ID)
		return fmt.Errorf("extract entities failed: %w", err)
	}
	
	// 3. 情感分析
	if err := np.analyzeSentiment(news); err != nil {
		logger.Error("Failed to analyze sentiment", "error", err, "news_id", news.ID)
		return fmt.Errorf("analyze sentiment failed: %w", err)
	}
	
	// 4. 股票关联
	if err := np.associateStocks(news); err != nil {
		logger.Error("Failed to associate stocks", "error", err, "news_id", news.ID)
		return fmt.Errorf("associate stocks failed: %w", err)
	}
	
	// 5. 行业映射
	if err := np.mapIndustries(news); err != nil {
		logger.Error("Failed to map industries", "error", err, "news_id", news.ID)
		return fmt.Errorf("map industries failed: %w", err)
	}
	
	logger.Info("News processed successfully", "news_id", news.ID, "title", news.Title)
	return nil
}

// preprocessContent 内容预处理
func (np *NewsProcessor) preprocessContent(news *models.NewsData) error {
	// 清理标题
	news.Title = np.cleanText(news.Title)
	
	// 清理内容
	news.Content = np.cleanText(news.Content)
	
	// 提取摘要（如果没有的话）- 暂时跳过，因为模型中没有Summary字段
	// if news.Summary == "" {
	//	news.Summary = np.extractSummary(news.Content)
	// }
	
	// 清理摘要 - 暂时跳过，因为模型中没有Summary字段
	// news.Summary = np.cleanText(news.Summary)
	
	return nil
}

// cleanText 清理文本内容
func (np *NewsProcessor) cleanText(text string) string {
	// 移除多余的空白字符
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	
	// 移除HTML标签
	text = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")
	
	// 移除特殊字符
	text = regexp.MustCompile(`[\x00-\x1f\x7f-\x9f]`).ReplaceAllString(text, "")
	
	// 移除多余的标点符号
	text = regexp.MustCompile(`[。，！？；：]{2,}`).ReplaceAllString(text, "。")
	
	// 去除首尾空格
	text = strings.TrimSpace(text)
	
	return text
}

// extractSummary 提取摘要
func (np *NewsProcessor) extractSummary(content string) string {
	// 简单的摘要提取：取前200个字符
	if len(content) <= 200 {
		return content
	}
	
	// 尝试在句号处截断
	summary := content[:200]
	lastPeriod := strings.LastIndex(summary, "。")
	if lastPeriod > 100 {
		summary = summary[:lastPeriod+1]
	}
	
	return summary
}

// extractEntities 实体识别
func (np *NewsProcessor) extractEntities(news *models.NewsData) error {
	entities := make(map[string][]string)
	
	// 提取股票代码
	stockCodes := np.extractStockCodes(news.Title + " " + news.Content)
	if len(stockCodes) > 0 {
		entities["stocks"] = stockCodes
	}
	
	// 提取公司名称
	companies := np.extractCompanies(news.Title + " " + news.Content)
	if len(companies) > 0 {
		entities["companies"] = companies
	}
	
	// 提取关键词
	keywords := np.extractKeywords(news.Title + " " + news.Content)
	if len(keywords) > 0 {
		entities["keywords"] = keywords
	}
	
	// 序列化实体信息 - 暂时跳过，因为模型中没有Entities字段
	// if len(entities) > 0 {
	//	entitiesJSON, err := json.Marshal(entities)
	//	if err != nil {
	//		return fmt.Errorf("marshal entities failed: %w", err)
	//	}
	//	news.Entities = string(entitiesJSON)
	// }
	
	return nil
}

// extractStockCodes 提取股票代码
func (np *NewsProcessor) extractStockCodes(text string) []string {
	matches := np.stockCodePattern.FindAllString(text, -1)
	codes := make([]string, 0)
	codeSet := make(map[string]bool)
	
	for _, match := range matches {
		// 验证是否为有效的股票代码格式
		if np.isValidStockCode(match) && !codeSet[match] {
			codes = append(codes, match)
			codeSet[match] = true
		}
	}
	
	return codes
}

// isValidStockCode 验证股票代码格式
func (np *NewsProcessor) isValidStockCode(code string) bool {
	// A股代码：6位数字
	if len(code) == 6 {
		for _, r := range code {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	}
	
	// 港股代码：2-4位字母
	if len(code) >= 2 && len(code) <= 4 {
		for _, r := range code {
			if r < 'A' || r > 'Z' {
				return false
			}
		}
		return true
	}
	
	return false
}

// extractCompanies 提取公司名称
func (np *NewsProcessor) extractCompanies(text string) []string {
	companies := make([]string, 0)
	companySet := make(map[string]bool)
	
	for company := range np.companyNames {
		if strings.Contains(text, company) && !companySet[company] {
			companies = append(companies, company)
			companySet[company] = true
		}
	}
	
	return companies
}

// extractKeywords 提取关键词
func (np *NewsProcessor) extractKeywords(text string) []string {
	keywords := make([]string, 0)
	keywordSet := make(map[string]bool)
	
	// 提取行业关键词
	for _, industryKeywords := range np.industryKeywords {
		for _, keyword := range industryKeywords {
			if strings.Contains(text, keyword) && !keywordSet[keyword] {
				keywords = append(keywords, keyword)
				keywordSet[keyword] = true
			}
		}
	}
	
	// 提取其他重要关键词
	importantKeywords := []string{
		"业绩", "财报", "营收", "利润", "亏损", "增长", "下滑",
		"并购", "重组", "上市", "退市", "停牌", "复牌",
		"分红", "配股", "增发", "回购", "减持", "增持",
		"监管", "政策", "法规", "处罚", "调查", "审计",
		"创新", "研发", "专利", "技术", "产品", "服务",
	}
	
	for _, keyword := range importantKeywords {
		if strings.Contains(text, keyword) && !keywordSet[keyword] {
			keywords = append(keywords, keyword)
			keywordSet[keyword] = true
		}
	}
	
	return keywords
}

// analyzeSentiment 情感分析
func (np *NewsProcessor) analyzeSentiment(news *models.NewsData) error {
	// 简单的基于关键词的情感分析
	text := news.Title + " " + news.Content
	sentimentScore := np.calculateSentimentScore(text)
	// 暂时跳过设置SentimentScore，因为模型中没有该字段
	// news.SentimentScore = &sentimentScore
	
	// 设置情感标签 - 暂时跳过，因为模型中没有SentimentLabel字段
	// if sentimentScore > 0.1 {
	//	news.SentimentLabel = "positive"
	// } else if sentimentScore < -0.1 {
	//	news.SentimentLabel = "negative"
	// } else {
	//	news.SentimentLabel = "neutral"
	// }
	
	// 避免未使用变量警告
	_ = sentimentScore
	
	return nil
}

// calculateSentimentScore 计算情感分数
func (np *NewsProcessor) calculateSentimentScore(text string) float64 {
	// 正面词汇
	positiveWords := []string{
		"增长", "上涨", "盈利", "利好", "突破", "创新", "成功", "优秀",
		"强劲", "稳定", "改善", "提升", "扩张", "发展", "机遇", "乐观",
		"买入", "推荐", "看好", "积极", "正面", "利润", "收益", "回升",
	}
	
	// 负面词汇
	negativeWords := []string{
		"下跌", "亏损", "利空", "风险", "危机", "困难", "失败", "恶化",
		"疲软", "不稳", "下滑", "减少", "收缩", "衰退", "威胁", "悲观",
		"卖出", "减持", "看空", "消极", "负面", "损失", "亏损", "暴跌",
	}
	
	positiveCount := 0
	negativeCount := 0
	
	// 统计正面词汇
	for _, word := range positiveWords {
		positiveCount += strings.Count(text, word)
	}
	
	// 统计负面词汇
	for _, word := range negativeWords {
		negativeCount += strings.Count(text, word)
	}
	
	// 计算情感分数
	totalWords := positiveCount + negativeCount
	if totalWords == 0 {
		return 0.0
	}
	
	sentimentScore := float64(positiveCount-negativeCount) / float64(totalWords)
	
	// 限制分数范围在[-1, 1]
	if sentimentScore > 1.0 {
		sentimentScore = 1.0
	} else if sentimentScore < -1.0 {
		sentimentScore = -1.0
	}
	
	return sentimentScore
}

// associateStocks 股票关联
func (np *NewsProcessor) associateStocks(news *models.NewsData) error {
	relatedStocks := make([]string, 0)
	stockSet := make(map[string]bool)
	
	text := news.Title + " " + news.Content
	
	// 通过股票代码关联
	stockCodes := np.extractStockCodes(text)
	for _, code := range stockCodes {
		if !stockSet[code] {
			relatedStocks = append(relatedStocks, code)
			stockSet[code] = true
		}
	}
	
	// 通过公司名称关联
	for company, code := range np.companyNames {
		if strings.Contains(text, company) && !stockSet[code] {
			relatedStocks = append(relatedStocks, code)
			stockSet[code] = true
		}
	}
	
	// 设置相关股票
	if len(relatedStocks) > 0 {
		news.RelatedStocks = relatedStocks
	}
	
	return nil
}

// mapIndustries 行业映射
func (np *NewsProcessor) mapIndustries(news *models.NewsData) error {
	industries := make([]string, 0)
	industrySet := make(map[string]bool)
	
	text := news.Title + " " + news.Content
	
	// 根据关键词匹配行业
	for industry, keywords := range np.industryKeywords {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) && !industrySet[industry] {
				industries = append(industries, industry)
				industrySet[industry] = true
				break // 找到一个关键词就足够了
			}
		}
	}
	
	// 设置主要行业分类 - 暂时跳过，因为模型中没有Category字段
	// if len(industries) > 0 {
	//	news.Category = industries[0] // 使用第一个匹配的行业作为主分类
	//	
	//	// 序列化所有相关行业 - 暂时跳过，因为模型中没有Tags字段
	//	industriesJSON, err := json.Marshal(industries)
	//	if err != nil {
	//		return fmt.Errorf("marshal industries failed: %w", err)
	//	}
	//	news.Tags = string(industriesJSON)
	// } else {
	//	// 默认分类
	//	news.Category = "综合"
	// }
	
	// 避免未使用变量警告
	_ = industries
	
	return nil
}

// GetProcessingStats 获取处理统计信息
func (np *NewsProcessor) GetProcessingStats() map[string]interface{} {
	return map[string]interface{}{
		"processor_type":      "news",
		"company_mappings":    len(np.companyNames),
		"industry_categories": len(np.industryKeywords),
		"last_updated":       time.Now(),
	}
}

// UpdateCompanyMappings 更新公司名称映射
func (np *NewsProcessor) UpdateCompanyMappings(mappings map[string]string) {
	np.companyNames = mappings
	logger.Info("Company mappings updated", "count", len(mappings))
}

// UpdateIndustryKeywords 更新行业关键词
func (np *NewsProcessor) UpdateIndustryKeywords(keywords map[string][]string) {
	np.industryKeywords = keywords
	logger.Info("Industry keywords updated", "industries", len(keywords))
}