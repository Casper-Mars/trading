package processing

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"data-collection-system/model"
	"data-collection-system/pkg/config"
	"data-collection-system/pkg/logger"
	"data-collection-system/repo/external/baidu"

	"github.com/go-redis/redis/v8"
)

// BaiduNLPProcessor 百度AI增强的NLP处理器
type BaiduNLPProcessor struct {
	baiduClient *baidu.Client
	redisClient *redis.Client
	config      *config.BaiduAIConfig
	// HTML标签清理正则
	htmlRegex *regexp.Regexp
	// 特殊字符清理正则
	specialCharRegex *regexp.Regexp
}

// NewBaiduNLPProcessor 创建百度AI NLP处理器
func NewBaiduNLPProcessor(cfg *config.BaiduAIConfig, redisClient *redis.Client) *BaiduNLPProcessor {
	baiduClient := baidu.NewClient(cfg)

	return &BaiduNLPProcessor{
		baiduClient:      baiduClient,
		redisClient:      redisClient,
		config:           cfg,
		htmlRegex:        regexp.MustCompile(`<[^>]*>`),
		specialCharRegex: regexp.MustCompile(`[\x00-\x1f\x7f-\x9f]`),
	}
}

// ProcessedResult NLP处理结果
type ProcessedResult struct {
	RelatedStocks   []string            `json:"related_stocks"`
	SentimentScore  float64             `json:"sentiment_score"`
	SentimentLabel  string              `json:"sentiment_label"`
	Keywords        []string            `json:"keywords"`
	Entities        map[string][]string `json:"entities"`
	ImportanceLevel int                 `json:"importance_level"`
	ProcessedAt     time.Time           `json:"processed_at"`
	CacheHit        bool                `json:"cache_hit"`
}

// ProcessNewsContent 处理新闻内容
func (p *BaiduNLPProcessor) ProcessNewsContent(ctx context.Context, news *model.NewsData) (*ProcessedResult, error) {
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
			logger.Error("Entity analysis failed:", err)
			errChan <- fmt.Errorf("entity analysis: %w", err)
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

	// 股票识别（本地处理）
	go func() {
		result.RelatedStocks = p.extractStocks(combinedText)
		errChan <- nil
	}()

	// 等待所有任务完成
	var errors []error
	for i := 0; i < 4; i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err)
		}
	}

	// 如果有错误但不是全部失败，记录警告但继续处理
	if len(errors) > 0 {
		logger.Warn("Some NLP processing tasks failed:", errors)
		// 如果所有任务都失败，返回错误
		if len(errors) == 4 {
			return nil, fmt.Errorf("all NLP processing tasks failed: %v", errors)
		}
	}

	// 计算重要性级别
	result.ImportanceLevel = p.calculateImportanceLevel(result, news)

	// 缓存结果
	if p.config.CacheEnabled {
		if err := p.saveToCache(ctx, cacheKey, result); err != nil {
			logger.Error("Failed to cache NLP result:", err)
		}
	}

	return result, nil
}

// preprocessText 预处理文本
func (p *BaiduNLPProcessor) preprocessText(text string) string {
	// 移除HTML标签
	text = p.htmlRegex.ReplaceAllString(text, "")

	// 移除特殊字符
	text = p.specialCharRegex.ReplaceAllString(text, "")

	// 移除多余空白
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// 去除首尾空白
	text = strings.TrimSpace(text)

	return text
}

// processSentiment 处理情感分析
func (p *BaiduNLPProcessor) processSentiment(ctx context.Context, text string, result *ProcessedResult) error {
	// 限制文本长度（百度API限制）
	if len(text) > 1000 {
		text = text[:1000]
	}

	resp, err := p.baiduClient.AnalyzeSentiment(ctx, text)
	if err != nil {
		return err
	}

	if len(resp.Items) > 0 {
		item := resp.Items[0]
		// 转换情感分数：0(负向)=-1, 1(中性)=0, 2(正向)=1
		switch item.Sentiment {
		case 0:
			result.SentimentScore = -item.Confidence
			result.SentimentLabel = "negative"
		case 1:
			result.SentimentScore = 0
			result.SentimentLabel = "neutral"
		case 2:
			result.SentimentScore = item.Confidence
			result.SentimentLabel = "positive"
		}
	}

	return nil
}

// processEntities 处理实体识别
func (p *BaiduNLPProcessor) processEntities(ctx context.Context, text string, result *ProcessedResult) error {
	// 限制文本长度
	if len(text) > 512 {
		text = text[:512]
	}

	resp, err := p.baiduClient.AnalyzeEntity(ctx, text)
	if err != nil {
		return err
	}

	entities := make(map[string][]string)
	for _, item := range resp.EntityAnalysis {
		if item.Status == "LINKED" && item.Confidence > 0.5 {
			category := item.Category.Level1
			if category == "" {
				category = "other"
			}
			entities[category] = append(entities[category], item.Mention)
		}
	}

	result.Entities = entities
	return nil
}

// processKeywords 处理关键词提取
func (p *BaiduNLPProcessor) processKeywords(ctx context.Context, title, content string, result *ProcessedResult) error {
	// 限制内容长度
	if len(content) > 2000 {
		content = content[:2000]
	}

	resp, err := p.baiduClient.ExtractKeywords(ctx, title, content)
	if err != nil {
		return err
	}

	keywords := make([]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		if item.Score > 0.3 { // 过滤低分关键词
			keywords = append(keywords, item.Tag)
		}
	}

	result.Keywords = keywords
	return nil
}

// extractStocks 提取股票代码（本地处理）
func (p *BaiduNLPProcessor) extractStocks(text string) []string {
	stockRegex := regexp.MustCompile(`[0-9]{6}`)
	stockCodes := stockRegex.FindAllString(text, -1)

	// 去重并验证
	stockSet := make(map[string]bool)
	for _, code := range stockCodes {
		if p.isValidStockCode(code) {
			stockSet[code] = true
		}
	}

	// 转换为切片
	result := make([]string, 0, len(stockSet))
	for stock := range stockSet {
		result = append(result, stock)
	}

	return result
}

// isValidStockCode 验证股票代码
func (p *BaiduNLPProcessor) isValidStockCode(code string) bool {
	if len(code) != 6 {
		return false
	}

	validPrefixes := []string{"600", "601", "603", "605", "000", "001", "002", "003", "300", "688"}
	firstThree := code[:3]

	for _, prefix := range validPrefixes {
		if firstThree == prefix {
			return true
		}
	}

	return false
}

// calculateImportanceLevel 计算重要性级别
func (p *BaiduNLPProcessor) calculateImportanceLevel(result *ProcessedResult, news *model.NewsData) int {
	score := 0

	// 基于关联股票数量
	score += len(result.RelatedStocks) * 2

	// 基于关键词数量和质量
	score += len(result.Keywords)

	// 基于实体数量
	for _, entities := range result.Entities {
		score += len(entities)
	}

	// 基于情感强度
	if result.SentimentScore != 0 {
		intensity := int(result.SentimentScore * 10)
		if intensity < 0 {
			intensity = -intensity
		}
		score += intensity
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

// generateCacheKey 生成缓存键
func (p *BaiduNLPProcessor) generateCacheKey(title, content string) string {
	data := title + "|" + content
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("nlp:baidu:%x", hash)
}

// getFromCache 从缓存获取结果
func (p *BaiduNLPProcessor) getFromCache(ctx context.Context, key string) (*ProcessedResult, error) {
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
func (p *BaiduNLPProcessor) saveToCache(ctx context.Context, key string, result *ProcessedResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	ttl := time.Duration(p.config.CacheTTL) * time.Second
	return p.redisClient.Set(ctx, key, data, ttl).Err()
}

// ProcessBatchNews 批量处理新闻
func (p *BaiduNLPProcessor) ProcessBatchNews(ctx context.Context, newsList []*model.NewsData) ([]*ProcessedResult, error) {
	results := make([]*ProcessedResult, len(newsList))
	errorChan := make(chan error, len(newsList))
	resultChan := make(chan struct {
		index  int
		result *ProcessedResult
	}, len(newsList))

	// 并发处理，但限制并发数
	semaphore := make(chan struct{}, 3) // 最多3个并发

	for i, news := range newsList {
		go func(index int, newsItem *model.NewsData) {
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			result, err := p.ProcessNewsContent(ctx, newsItem)
			if err != nil {
				errorChan <- fmt.Errorf("failed to process news %d: %w", newsItem.ID, err)
				return
			}

			resultChan <- struct {
				index  int
				result *ProcessedResult
			}{index: index, result: result}
		}(i, news)
	}

	// 收集结果
	var errors []error
	completedCount := 0

	for completedCount < len(newsList) {
		select {
		case result := <-resultChan:
			results[result.index] = result.result
			completedCount++
		case err := <-errorChan:
			errors = append(errors, err)
			completedCount++
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if len(errors) > 0 {
		logger.Warn("Some news processing failed:", errors)
	}

	return results, nil
}
