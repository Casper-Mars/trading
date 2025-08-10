package processing

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"data-collection-system/model"
	"data-collection-system/pkg/config"
	"data-collection-system/pkg/logger"
	"data-collection-system/repo/external/alicloud"

	"github.com/go-redis/redis/v8"
)

// ProcessedResult NLP处理结果
type ProcessedResult struct {
	RelatedStocks    []string            `json:"related_stocks"`
	SentimentScore   float64             `json:"sentiment_score"`
	SentimentLabel   string              `json:"sentiment_label"`
	Keywords         []string            `json:"keywords"`
	Entities         map[string][]string `json:"entities"`
	ImportanceLevel  int                 `json:"importance_level"`
	ProcessedAt      time.Time           `json:"processed_at"`
	CacheHit         bool                `json:"cache_hit"`
}

// AliCloudNLPProcessor 阿里云NLP处理器
type AliCloudNLPProcessor struct {
	aliCloudClient *alicloud.Client
	redisClient    *redis.Client
	config         *config.AliCloudNLPConfig
	// HTML标签清理正则
	htmlRegex *regexp.Regexp
	// 特殊字符清理正则
	specialCharRegex *regexp.Regexp
	// 股票代码正则
	stockRegex *regexp.Regexp
}

// NewAliCloudNLPProcessor 创建阿里云NLP处理器
func NewAliCloudNLPProcessor(cfg *config.AliCloudNLPConfig, redisClient *redis.Client) *AliCloudNLPProcessor {
	aliCloudClient := alicloud.NewClient(cfg)

	return &AliCloudNLPProcessor{
		aliCloudClient:   aliCloudClient,
		redisClient:      redisClient,
		config:           cfg,
		htmlRegex:        regexp.MustCompile(`<[^>]*>`),
		specialCharRegex: regexp.MustCompile(`[\x00-\x1f\x7f-\x9f]`),
		stockRegex:       regexp.MustCompile(`[0-9]{6}`),
	}
}

// ProcessNewsContent 处理新闻内容 - 实现NLPProcessorInterface接口
func (p *AliCloudNLPProcessor) ProcessNewsContent(news *model.NewsData) error {
	ctx := context.Background()
	result, err := p.processNewsContentInternal(ctx, news)
	if err != nil {
		return err
	}
	
	// 将结果设置到新闻数据中
	news.RelatedStocks = result.RelatedStocks
	// TODO: 根据NewsData模型添加其他字段
	
	return nil
}

// processNewsContentInternal 内部处理新闻内容的方法
func (p *AliCloudNLPProcessor) processNewsContentInternal(ctx context.Context, news *model.NewsData) (*ProcessedResult, error) {
	if news == nil {
		return nil, fmt.Errorf("news data is nil")
	}

	// 生成缓存键
	cacheKey := p.generateCacheKey(news.Title, news.Content)

	// 尝试从缓存获取结果
	if p.config.CacheEnabled {
		if result, err := p.getFromCache(ctx, cacheKey); err == nil {
			logger.Debug("NLP processing result found in cache")
			result.CacheHit = true
			return result, nil
		}
	}

	// 预处理文本
	cleanTitle := p.preprocessText(news.Title)
	cleanContent := p.preprocessText(news.Content)
	combinedText := cleanTitle + " " + cleanContent

	// 创建处理结果
	result := &ProcessedResult{
		ProcessedAt: time.Now(),
		CacheHit:    false,
	}

	// 并发处理各种NLP任务
	errChan := make(chan error, 4)

	// 情感分析
	go func() {
		if err := p.processSentiment(ctx, combinedText, result); err != nil {
			logger.Error("Sentiment analysis failed:", err)
			errChan <- fmt.Errorf("sentiment analysis: %w", err)
			return
		}
		errChan <- nil
	}()

	// 实体识别
	go func() {
		if err := p.processEntities(ctx, combinedText, result); err != nil {
			logger.Error("Entity extraction failed:", err)
			errChan <- fmt.Errorf("entity extraction: %w", err)
			return
		}
		errChan <- nil
	}()

	// 关键词提取
	go func() {
		if err := p.processKeywords(ctx, cleanTitle, cleanContent, result); err != nil {
			logger.Error("Keyword extraction failed:", err)
			errChan <- fmt.Errorf("keyword extraction: %w", err)
			return
		}
		errChan <- nil
	}()

	// 股票提取
	go func() {
		result.RelatedStocks = p.extractStocks(combinedText)
		errChan <- nil
	}()

	// 等待所有任务完成
	for i := 0; i < 4; i++ {
		if err := <-errChan; err != nil {
			logger.Error("NLP processing error:", err)
			// 继续处理其他任务，不因单个任务失败而终止
		}
	}

	// 计算重要性等级
	result.ImportanceLevel = p.calculateImportanceLevel(result, news)

	// 保存到缓存
	if p.config.CacheEnabled {
		if err := p.saveToCache(ctx, cacheKey, result); err != nil {
			logger.Error("Failed to save result to cache:", err)
		}
	}

	return result, nil
}

// preprocessText 预处理文本
func (p *AliCloudNLPProcessor) preprocessText(text string) string {
	// 移除HTML标签
	text = p.htmlRegex.ReplaceAllString(text, "")
	// 移除特殊字符
	text = p.specialCharRegex.ReplaceAllString(text, "")
	// 移除多余空格
	text = strings.TrimSpace(text)
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	return text
}

// processSentiment 处理情感分析
func (p *AliCloudNLPProcessor) processSentiment(ctx context.Context, text string, result *ProcessedResult) error {
	// 限制文本长度
	if len(text) > 1000 {
		text = text[:1000]
	}

	response, err := p.aliCloudClient.AnalyzeSentiment(text)
	if err != nil {
		return fmt.Errorf("sentiment analysis failed: %w", err)
	}

	// 解析情感标签和置信度
	result.SentimentLabel = response.Data.Sentiment
	result.SentimentScore = response.Data.Confidence

	// 转换为数值分数 (positive: 1, negative: -1, neutral: 0)
	switch strings.ToLower(response.Data.Sentiment) {
	case "positive":
		result.SentimentScore = response.Data.Confidence
	case "negative":
		result.SentimentScore = -response.Data.Confidence
	case "neutral":
		result.SentimentScore = 0
	default:
		result.SentimentScore = 0
	}

	return nil
}

// processEntities 处理实体识别
func (p *AliCloudNLPProcessor) processEntities(ctx context.Context, text string, result *ProcessedResult) error {
	// 限制文本长度
	if len(text) > 1000 {
		text = text[:1000]
	}

	response, err := p.aliCloudClient.ExtractEntities(text)
	if err != nil {
		return fmt.Errorf("entity extraction failed: %w", err)
	}

	// 初始化实体映射
	result.Entities = make(map[string][]string)

	// 解析实体
	for _, entity := range response.Data.Entities {
		if result.Entities[entity.Type] == nil {
			result.Entities[entity.Type] = make([]string, 0)
		}
		result.Entities[entity.Type] = append(result.Entities[entity.Type], entity.Value)
	}

	return nil
}

// processKeywords 处理关键词提取
func (p *AliCloudNLPProcessor) processKeywords(ctx context.Context, title, content string, result *ProcessedResult) error {
	// 组合标题和内容，限制长度
	combinedText := title + " " + content
	if len(combinedText) > 1000 {
		combinedText = combinedText[:1000]
	}

	response, err := p.aliCloudClient.ExtractKeywords(combinedText, 10)
	if err != nil {
		return fmt.Errorf("keyword extraction failed: %w", err)
	}

	// 解析关键词
	result.Keywords = make([]string, 0, len(response.Data.Keywords))
	for _, keyword := range response.Data.Keywords {
		result.Keywords = append(result.Keywords, keyword.Word)
	}

	return nil
}

// extractStocks 提取股票代码
func (p *AliCloudNLPProcessor) extractStocks(text string) []string {
	stockCodes := make(map[string]bool)

	// 使用正则表达式匹配6位数字的股票代码
	matches := p.stockRegex.FindAllString(text, -1)
	for _, match := range matches {
		if p.isValidStockCode(match) {
			stockCodes[match] = true
		}
	}

	// 转换为切片
	result := make([]string, 0, len(stockCodes))
	for code := range stockCodes {
		result = append(result, code)
	}

	return result
}

// isValidStockCode 验证股票代码是否有效
func (p *AliCloudNLPProcessor) isValidStockCode(code string) bool {
	if len(code) != 6 {
		return false
	}

	// 检查是否为纯数字
	if _, err := strconv.Atoi(code); err != nil {
		return false
	}

	// 简单的股票代码范围验证
	// 沪市：600xxx, 601xxx, 603xxx, 605xxx
	// 深市：000xxx, 001xxx, 002xxx, 003xxx
	// 创业板：300xxx
	// 科创板：688xxx
	firstThree := code[:3]
	switch firstThree {
	case "600", "601", "603", "605", "688": // 沪市和科创板
		return true
	case "000", "001", "002", "003", "300": // 深市和创业板
		return true
	default:
		return false
	}
}

// calculateImportanceLevel 计算重要性等级
func (p *AliCloudNLPProcessor) calculateImportanceLevel(result *ProcessedResult, news *model.NewsData) int {
	score := 0

	// 基于情感分析结果
	if result.SentimentScore != 0 {
		if result.SentimentScore > 0.7 || result.SentimentScore < -0.7 {
			score += 30 // 强烈情感
		} else if result.SentimentScore > 0.3 || result.SentimentScore < -0.3 {
			score += 20 // 中等情感
		} else {
			score += 10 // 轻微情感
		}
	}

	// 基于关联股票数量
	stockCount := len(result.RelatedStocks)
	if stockCount > 0 {
		score += stockCount * 15
		if stockCount >= 5 {
			score += 20 // 涉及多只股票的新闻更重要
		}
	}

	// 基于关键词数量和质量
	keywordCount := len(result.Keywords)
	if keywordCount > 0 {
		score += keywordCount * 5
	}

	// 基于实体数量
	entityCount := 0
	for _, entities := range result.Entities {
		entityCount += len(entities)
	}
	if entityCount > 0 {
		score += entityCount * 3
	}

	// 基于新闻来源权重
	if news.Source != "" {
		// 重要财经媒体加权
	importantSources := []string{"新华社", "人民日报", "央视新闻", "证券时报", "上海证券报", "中国证券报"}
		for _, source := range importantSources {
			if strings.Contains(news.Source, source) {
				score += 25
				break
			}
		}
	}

	// 转换为1-5的等级
	if score >= 100 {
		return 5 // 非常重要
	} else if score >= 80 {
		return 4 // 重要
	} else if score >= 60 {
		return 3 // 中等重要
	} else if score >= 40 {
		return 2 // 一般重要
	} else {
		return 1 // 不太重要
	}
}

// generateCacheKey 生成缓存键
func (p *AliCloudNLPProcessor) generateCacheKey(title, content string) string {
	hash := md5.Sum([]byte(title + content))
	return fmt.Sprintf("alicloud_nlp:%x", hash)
}

// getFromCache 从缓存获取结果
func (p *AliCloudNLPProcessor) getFromCache(ctx context.Context, key string) (*ProcessedResult, error) {
	data, err := p.redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var result ProcessedResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// saveToCache 保存结果到缓存
func (p *AliCloudNLPProcessor) saveToCache(ctx context.Context, key string, result *ProcessedResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return p.redisClient.Set(ctx, key, data, time.Duration(p.config.CacheTTL)*time.Second).Err()
}

// ProcessBatchNews 批量处理新闻 - 实现NLPProcessorInterface接口
func (p *AliCloudNLPProcessor) ProcessBatchNews(newsList []*model.NewsData) error {
	ctx := context.Background()
	_, err := p.processBatchNewsInternal(ctx, newsList)
	return err
}

// processBatchNewsInternal 内部批量处理新闻的方法
func (p *AliCloudNLPProcessor) processBatchNewsInternal(ctx context.Context, newsList []*model.NewsData) ([]*ProcessedResult, error) {
	results := make([]*ProcessedResult, 0, len(newsList))

	for _, news := range newsList {
		result, err := p.processNewsContentInternal(ctx, news)
		if err != nil {
			logger.Error("Failed to process news:", err)
			continue
		}
		results = append(results, result)

		// 限流控制
		if p.config.QPS > 0 {
			time.Sleep(time.Second / time.Duration(p.config.QPS))
		}
	}

	return results, nil
}

// ExtractRelatedStocks 提取相关股票 - 实现NLPProcessorInterface接口
func (p *AliCloudNLPProcessor) ExtractRelatedStocks(text string) []string {
	return p.extractStocks(text)
}

// AnalyzeSentiment 分析情感 - 实现NLPProcessorInterface接口
func (p *AliCloudNLPProcessor) AnalyzeSentiment(text string) float64 {
	ctx := context.Background()
	result := &ProcessedResult{}
	err := p.processSentiment(ctx, text, result)
	if err != nil {
		return 0.0
	}
	return result.SentimentScore
}

// ExtractKeywords 提取关键词 - 实现NLPProcessorInterface接口
func (p *AliCloudNLPProcessor) ExtractKeywords(text string) []string {
	ctx := context.Background()
	result := &ProcessedResult{}
	err := p.processKeywords(ctx, text, "", result)
	if err != nil {
		return []string{}
	}
	return result.Keywords
}

// ExtractEntities 提取实体 - 实现NLPProcessorInterface接口
func (p *AliCloudNLPProcessor) ExtractEntities(text string) map[string][]string {
	ctx := context.Background()
	result := &ProcessedResult{}
	err := p.processEntities(ctx, text, result)
	if err != nil {
		return map[string][]string{}
	}
	return result.Entities
}

// CalculateImportanceLevel 计算重要性等级 - 实现NLPProcessorInterface接口
func (p *AliCloudNLPProcessor) CalculateImportanceLevel(news *model.NewsData) int {
	ctx := context.Background()
	result, err := p.processNewsContentInternal(ctx, news)
	if err != nil {
		return 1
	}
	return result.ImportanceLevel
}