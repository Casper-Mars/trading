package processing

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"data-collection-system/model"
)

// NLPProcessor NLP处理器
type NLPProcessor struct {
	// 股票代码正则表达式
	stockRegex *regexp.Regexp
	// 公司名称映射
	companyNameMap map[string]string
	// 停用词列表
	stopWords map[string]bool
	// 情感词典
	sentimentDict map[string]float64
}

// NewNLPProcessor 创建NLP处理器实例
func NewNLPProcessor() *NLPProcessor {
	return &NLPProcessor{
		stockRegex:     regexp.MustCompile(`[0-9]{6}`),
		companyNameMap: initCompanyNameMap(),
		stopWords:      initStopWords(),
		sentimentDict:  initSentimentDict(),
	}
}

// ProcessNewsContent 处理新闻内容，提取关键信息
func (nlp *NLPProcessor) ProcessNewsContent(news *model.NewsData) error {
	if news == nil {
		return errors.New("news data is nil")
	}

	// 提取关联股票
	relatedStocks := nlp.ExtractRelatedStocks(news.Title + " " + news.Content)
	news.RelatedStocks = relatedStocks

	// 提取关键词
	// keywords := nlp.ExtractKeywords(news.Title + " " + news.Content)
	// TODO: 添加Keywords字段到NewsData模型或使用其他字段存储
	// keywordsJSON, _ := json.Marshal(keywords)
	// news.Keywords = keywordsJSON

	// 情感分析
	sentimentScore := nlp.AnalyzeSentiment(news.Title + " " + news.Content)
	news.SentimentScore = &sentimentScore

	// 计算重要性级别
	importanceLevel := nlp.CalculateImportanceLevel(news)
	news.ImportanceLevel = int8(importanceLevel)

	return nil
}

// ExtractRelatedStocks 从文本中提取相关股票代码
func (nlp *NLPProcessor) ExtractRelatedStocks(text string) []string {
	stockSet := make(map[string]bool)

	// 方法1: 通过股票代码正则匹配
	stockCodes := nlp.stockRegex.FindAllString(text, -1)
	for _, code := range stockCodes {
		// 验证是否为有效股票代码（简单验证）
		if nlp.isValidStockCode(code) {
			stockSet[code] = true
		}
	}

	// 方法2: 通过公司名称匹配
	for companyName, stockCode := range nlp.companyNameMap {
		if strings.Contains(text, companyName) {
			stockSet[stockCode] = true
		}
	}

	// 方法3: 通过股票简称匹配（如"平安银行"、"招商银行"等）
	stockNames := nlp.extractStockNames(text)
	for _, name := range stockNames {
		if code, exists := nlp.companyNameMap[name]; exists {
			stockSet[code] = true
		}
	}

	// 转换为切片并排序
	result := make([]string, 0, len(stockSet))
	for stock := range stockSet {
		result = append(result, stock)
	}
	sort.Strings(result)

	return result
}

// isValidStockCode 验证股票代码是否有效
func (nlp *NLPProcessor) isValidStockCode(code string) bool {
	// 简单验证：6位数字，且符合A股编码规则
	if len(code) != 6 {
		return false
	}

	// A股主板：600xxx, 601xxx, 603xxx, 605xxx (上交所)
	// A股主板：000xxx, 001xxx, 002xxx, 003xxx (深交所)
	// 创业板：300xxx (深交所)
	// 科创板：688xxx (上交所)
	firstThree := code[:3]
	validPrefixes := []string{"600", "601", "603", "605", "000", "001", "002", "003", "300", "688"}

	for _, prefix := range validPrefixes {
		if firstThree == prefix {
			return true
		}
	}

	return false
}

// extractStockNames 从文本中提取可能的股票名称
func (nlp *NLPProcessor) extractStockNames(text string) []string {
	// 使用正则表达式匹配可能的公司名称模式
	// 匹配包含"银行"、"保险"、"证券"、"科技"等关键词的公司名称
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`[\p{Han}]{2,10}银行`),
		regexp.MustCompile(`[\p{Han}]{2,10}保险`),
		regexp.MustCompile(`[\p{Han}]{2,10}证券`),
		regexp.MustCompile(`[\p{Han}]{2,10}科技`),
		regexp.MustCompile(`[\p{Han}]{2,10}能源`),
		regexp.MustCompile(`[\p{Han}]{2,10}电力`),
		regexp.MustCompile(`[\p{Han}]{2,10}地产`),
		regexp.MustCompile(`[\p{Han}]{2,10}医药`),
		regexp.MustCompile(`[\p{Han}]{2,10}汽车`),
	}

	nameSet := make(map[string]bool)
	for _, pattern := range patterns {
		matches := pattern.FindAllString(text, -1)
		for _, match := range matches {
			nameSet[match] = true
		}
	}

	result := make([]string, 0, len(nameSet))
	for name := range nameSet {
		result = append(result, name)
	}

	return result
}

// ExtractKeywords 从文本中提取关键词
func (nlp *NLPProcessor) ExtractKeywords(text string) []string {
	// 简单的关键词提取算法
	// 1. 分词（简单按标点和空格分割）
	words := nlp.tokenize(text)

	// 2. 过滤停用词和短词
	filteredWords := make([]string, 0)
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(word) >= 2 && !nlp.stopWords[word] && nlp.isValidKeyword(word) {
			filteredWords = append(filteredWords, word)
		}
	}

	// 3. 统计词频
	wordCount := make(map[string]int)
	for _, word := range filteredWords {
		wordCount[word]++
	}

	// 4. 按词频排序，取前10个
	type wordFreq struct {
		word  string
		count int
	}

	wordFreqs := make([]wordFreq, 0, len(wordCount))
	for word, count := range wordCount {
		wordFreqs = append(wordFreqs, wordFreq{word: word, count: count})
	}

	sort.Slice(wordFreqs, func(i, j int) bool {
		return wordFreqs[i].count > wordFreqs[j].count
	})

	// 取前10个关键词
	maxKeywords := 10
	if len(wordFreqs) < maxKeywords {
		maxKeywords = len(wordFreqs)
	}

	result := make([]string, maxKeywords)
	for i := 0; i < maxKeywords; i++ {
		result[i] = wordFreqs[i].word
	}

	return result
}

// tokenize 简单分词
func (nlp *NLPProcessor) tokenize(text string) []string {
	// 使用正则表达式分割文本
	re := regexp.MustCompile(`[\p{P}\p{S}\s]+`)
	words := re.Split(text, -1)

	// 过滤空字符串
	result := make([]string, 0)
	for _, word := range words {
		if strings.TrimSpace(word) != "" {
			result = append(result, word)
		}
	}

	return result
}

// isValidKeyword 判断是否为有效关键词
func (nlp *NLPProcessor) isValidKeyword(word string) bool {
	// 过滤纯数字
	if regexp.MustCompile(`^\d+$`).MatchString(word) {
		return false
	}

	// 过滤纯英文（除非是重要的英文缩写）
	if regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(word) {
		importantAbbr := map[string]bool{
			"GDP": true, "CPI": true, "PMI": true, "IPO": true,
			"CEO": true, "CFO": true, "AI": true, "5G": true,
		}
		return importantAbbr[strings.ToUpper(word)]
	}

	// 检查是否包含中文字符
	for _, r := range word {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}

	return false
}

// AnalyzeSentiment 情感分析
func (nlp *NLPProcessor) AnalyzeSentiment(text string) float64 {
	// 简单的基于词典的情感分析
	words := nlp.tokenize(text)
	totalScore := 0.0
	wordCount := 0

	for _, word := range words {
		if score, exists := nlp.sentimentDict[word]; exists {
			totalScore += score
			wordCount++
		}
	}

	if wordCount == 0 {
		return 0.0 // 中性
	}

	// 计算平均情感分数
	avgScore := totalScore / float64(wordCount)

	// 归一化到[-1, 1]范围
	if avgScore > 1.0 {
		avgScore = 1.0
	} else if avgScore < -1.0 {
		avgScore = -1.0
	}

	return avgScore
}

// CalculateImportanceLevel 计算新闻重要性级别
func (nlp *NLPProcessor) CalculateImportanceLevel(news *model.NewsData) int {
	score := 0

	// 基于关联股票数量
	if len(news.RelatedStocks) > 0 {
		score += len(news.RelatedStocks) * 2
	}

	// 基于关键词重要性
	importantKeywords := map[string]int{
		"重大": 5, "重要": 4, "紧急": 5, "突发": 4,
		"业绩": 3, "财报": 3, "分红": 2, "重组": 4,
		"并购": 4, "收购": 3, "投资": 2, "融资": 3,
		"上市": 3, "退市": 4, "停牌": 3, "复牌": 2,
		"涨停": 3, "跌停": 3, "暴涨": 3, "暴跌": 3,
	}

	// TODO: 修复Keywords字段引用
	// var keywords []string
	// if news.Keywords != nil {
	//	json.Unmarshal(news.Keywords, &keywords)
	// }
	// for _, keyword := range keywords {
	for _, keyword := range []string{} { // 临时修复
		if weight, exists := importantKeywords[keyword]; exists {
			score += weight
		}
	}

	// 基于情感强度
	if news.SentimentScore != nil {
		sentimentIntensity := int((*news.SentimentScore) * 10)
		if sentimentIntensity < 0 {
			sentimentIntensity = -sentimentIntensity
		}
		score += sentimentIntensity
	}

	// 基于标题长度（较长的标题通常包含更多信息）
	if len(news.Title) > 20 {
		score += 1
	}
	if len(news.Title) > 40 {
		score += 1
	}

	// 基于内容长度
	if len(news.Content) > 500 {
		score += 1
	}
	if len(news.Content) > 1000 {
		score += 1
	}

	// 转换为1-5级别
	if score >= 20 {
		return 5
	} else if score >= 15 {
		return 4
	} else if score >= 10 {
		return 3
	} else if score >= 5 {
		return 2
	} else {
		return 1
	}
}

// ExtractEntities 提取命名实体
func (nlp *NLPProcessor) ExtractEntities(text string) map[string][]string {
	entities := make(map[string][]string)

	// 提取人名（简单模式）
	personPattern := regexp.MustCompile(`[\p{Han}]{2,4}(?:先生|女士|总裁|董事长|CEO|CFO|总经理)`)
	persons := personPattern.FindAllString(text, -1)
	for i, person := range persons {
		// 移除后缀
		person = regexp.MustCompile(`(?:先生|女士|总裁|董事长|CEO|CFO|总经理)$`).ReplaceAllString(person, "")
		persons[i] = person
	}
	entities["person"] = persons

	// 提取机构名（简单模式）
	orgPattern := regexp.MustCompile(`[\p{Han}]{2,20}(?:公司|集团|银行|保险|证券|基金|信托)`)
	orgs := orgPattern.FindAllString(text, -1)
	entities["organization"] = orgs

	// 提取地名（简单模式）
	locationPattern := regexp.MustCompile(`[\p{Han}]{2,10}(?:省|市|区|县|镇|村)`)
	locations := locationPattern.FindAllString(text, -1)
	entities["location"] = locations

	return entities
}

// SummarizeText 文本摘要（简单实现）
func (nlp *NLPProcessor) SummarizeText(text string, maxSentences int) string {
	// 简单的抽取式摘要
	sentences := nlp.splitSentences(text)
	if len(sentences) <= maxSentences {
		return text
	}

	// 计算每个句子的重要性分数
	sentenceScores := make([]struct {
		sentence string
		score    float64
	}, len(sentences))

	for i, sentence := range sentences {
		score := nlp.calculateSentenceScore(sentence)
		sentenceScores[i] = struct {
			sentence string
			score    float64
		}{sentence: sentence, score: score}
	}

	// 按分数排序
	sort.Slice(sentenceScores, func(i, j int) bool {
		return sentenceScores[i].score > sentenceScores[j].score
	})

	// 取前N个句子
	summary := make([]string, maxSentences)
	for i := 0; i < maxSentences; i++ {
		summary[i] = sentenceScores[i].sentence
	}

	return strings.Join(summary, "")
}

// splitSentences 分句
func (nlp *NLPProcessor) splitSentences(text string) []string {
	// 按句号、问号、感叹号分割
	re := regexp.MustCompile(`[。！？]+`)
	sentences := re.Split(text, -1)

	// 过滤空句子
	result := make([]string, 0)
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence != "" {
			result = append(result, sentence)
		}
	}

	return result
}

// calculateSentenceScore 计算句子重要性分数
func (nlp *NLPProcessor) calculateSentenceScore(sentence string) float64 {
	score := 0.0

	// 基于长度（适中长度的句子通常更重要）
	length := len(sentence)
	if length >= 20 && length <= 100 {
		score += 1.0
	}

	// 基于关键词
	importantWords := []string{"重要", "重大", "关键", "核心", "主要", "首次", "突破"}
	for _, word := range importantWords {
		if strings.Contains(sentence, word) {
			score += 0.5
		}
	}

	// 基于数字（包含数字的句子通常更具体）
	if regexp.MustCompile(`\d+`).MatchString(sentence) {
		score += 0.3
	}

	return score
}

// initCompanyNameMap 初始化公司名称映射
func initCompanyNameMap() map[string]string {
	// 这里应该从数据库或配置文件加载
	// 简单示例
	return map[string]string{
		"平安银行":   "000001",
		"万科A":    "000002",
		"招商银行":   "600036",
		"工商银行":   "601398",
		"建设银行":   "601939",
		"中国银行":   "601988",
		"农业银行":   "601288",
		"中国平安":   "601318",
		"贵州茅台":   "600519",
		"五粮液":    "000858",
		"腾讯控股":   "00700", // 港股
		"阿里巴巴":   "09988", // 港股
	}
}

// initStopWords 初始化停用词
func initStopWords() map[string]bool {
	return map[string]bool{
		"的": true, "了": true, "在": true, "是": true, "我": true,
		"有": true, "和": true, "就": true, "不": true, "人": true,
		"都": true, "一": true, "一个": true, "上": true, "也": true,
		"很": true, "到": true, "说": true, "要": true, "去": true,
		"你": true, "会": true, "着": true, "没有": true, "看": true,
		"好": true, "自己": true, "这": true, "那": true, "里": true,
		"为": true, "能": true, "以": true, "时": true, "过": true,
		"他": true, "她": true, "它": true, "们": true, "这个": true,
		"那个": true, "什么": true, "怎么": true, "为什么": true,
	}
}

// initSentimentDict 初始化情感词典
func initSentimentDict() map[string]float64 {
	return map[string]float64{
		// 正面词汇
		"好": 0.5, "很好": 0.7, "优秀": 0.8, "杰出": 0.9, "卓越": 1.0,
		"增长": 0.6, "上涨": 0.7, "涨": 0.6, "升": 0.5, "高": 0.4,
		"盈利": 0.7, "收益": 0.6, "利润": 0.6, "赚": 0.5, "赢": 0.6,
		"成功": 0.8, "胜利": 0.8, "突破": 0.7, "创新": 0.6, "发展": 0.5,
		"机会": 0.5, "机遇": 0.6, "前景": 0.4, "潜力": 0.5, "希望": 0.5,
		
		// 负面词汇
		"坏": -0.5, "很坏": -0.7, "糟糕": -0.8, "恶劣": -0.9, "灾难": -1.0,
		"下跌": -0.7, "跌": -0.6, "降": -0.5, "低": -0.4, "减少": -0.5,
		"亏损": -0.7, "损失": -0.6, "赔": -0.6, "输": -0.6, "失败": -0.8,
		"危机": -0.8, "风险": -0.6, "威胁": -0.7, "困难": -0.6, "问题": -0.4,
		"担心": -0.5, "忧虑": -0.6, "恐慌": -0.8, "焦虑": -0.6, "紧张": -0.5,
		
		// 中性偏正面
		"稳定": 0.3, "平稳": 0.2, "正常": 0.1, "一般": 0.0, "普通": 0.0,
		
		// 中性偏负面
		"不确定": -0.3, "疑虑": -0.4, "质疑": -0.4, "怀疑": -0.3,
	}
}

// ProcessBatchNews 批量处理新闻数据
func (nlp *NLPProcessor) ProcessBatchNews(newsList []*model.NewsData) error {
	for _, news := range newsList {
		if err := nlp.ProcessNewsContent(news); err != nil {
			return fmt.Errorf("failed to process news %d: %w", news.ID, err)
		}
	}
	return nil
}

// NLPProcessingStats NLP处理统计信息
type NLPProcessingStats struct {
	TotalNews        int `json:"total_news"`
	ProcessedNews    int `json:"processed_news"`
	ExtractedStocks  int `json:"extracted_stocks"`
	ExtractedKeywords int `json:"extracted_keywords"`
	AvgSentiment     float64 `json:"avg_sentiment"`
}

// GetProcessingStats 获取处理统计信息
func (nlp *NLPProcessor) GetProcessingStats(newsList []*model.NewsData) NLPProcessingStats {
	stats := NLPProcessingStats{
		TotalNews: len(newsList),
	}

	totalSentiment := 0.0
	sentimentCount := 0
	totalStocks := 0
	totalKeywords := 0

	for _, news := range newsList {
		if news == nil {
			continue
		}

		stats.ProcessedNews++
		totalStocks += len(news.RelatedStocks)
		// TODO: 修复Keywords字段引用
		// var keywords []string
		// if news.Keywords != nil {
		//	json.Unmarshal(news.Keywords, &keywords)
		// }
		// totalKeywords += len(keywords)
		totalKeywords += 0 // 临时修复

		if news.SentimentScore != nil {
			totalSentiment += *news.SentimentScore
			sentimentCount++
		}
	}

	stats.ExtractedStocks = totalStocks
	stats.ExtractedKeywords = totalKeywords

	if sentimentCount > 0 {
		stats.AvgSentiment = totalSentiment / float64(sentimentCount)
	}

	return stats
}

// ExportProcessingReport 导出处理报告
func (nlp *NLPProcessor) ExportProcessingReport(newsList []*model.NewsData) ([]byte, error) {
	stats := nlp.GetProcessingStats(newsList)

	report := map[string]interface{}{
		"processing_stats": stats,
		"sample_news":      newsList[:min(len(newsList), 5)], // 取前5条作为样例
		"timestamp":        time.Now(),
	}

	return json.MarshalIndent(report, "", "  ")
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}