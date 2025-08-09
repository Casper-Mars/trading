package processing

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"data-collection-system/model"
	"data-collection-system/pkg/config"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// NewsRepository 新闻数据仓库接口
type NewsRepository interface {
	GetByTimeRange(ctx context.Context, startTime, endTime time.Time, limit, offset int) ([]*model.NewsData, error)
	Update(ctx context.Context, news *model.NewsData) error
}

// ProcessingService 数据处理服务
type ProcessingService struct {
	db             *gorm.DB
	newsRepo       NewsRepository
	cleaner        *DataCleaner
	validator      *DataValidator
	deduper        *DataDeduplicator
	qualityChecker *QualityChecker
	nlpProcessor   *BaiduNLPProcessor
}

// NewProcessingService 创建数据处理服务实例
func NewProcessingService(
	db *gorm.DB,
	newsRepo NewsRepository,
	cfg *config.Config,
	redisClient *redis.Client,
) *ProcessingService {
	// 创建百度AI NLP处理器
	nlpProcessor := NewBaiduNLPProcessor(&cfg.BaiduAI, redisClient)

	return &ProcessingService{
		db:             db,
		newsRepo:       newsRepo,
		cleaner:        NewDataCleaner(),
		validator:      NewDataValidator(),
		deduper:        NewDataDeduplicator(),
		qualityChecker: NewQualityChecker(),
		nlpProcessor:   nlpProcessor,
	}
}

// ProcessNewsData 处理新闻数据
func (s *ProcessingService) ProcessNewsData(ctx context.Context, limit int) error {
	// 获取未处理的新闻数据（使用时间范围查询最近的数据）
	startTime := time.Now().Add(-24 * time.Hour) // 最近24小时
	endTime := time.Now()
	newsList, err := s.newsRepo.GetByTimeRange(ctx, startTime, endTime, limit, 0)
	if err != nil {
		return fmt.Errorf("failed to get unprocessed news: %w", err)
	}

	log.Printf("Starting news processing, count: %d, limit: %d", len(newsList), limit)

	// 使用批量处理提高效率
	if len(newsList) > 5 {
		return s.processBatchNews(ctx, newsList)
	}

	// 少量数据使用单个处理
	for _, news := range newsList {
		if err := s.processNewsItem(ctx, news); err != nil {
			log.Printf("Failed to process news item, error: %v, news_id: %d", err, news.ID)
			continue
		}
	}

	log.Printf("News processing completed, processed_count: %d", len(newsList))
	return nil
}

// processBatchNews 批量处理新闻数据
func (s *ProcessingService) processBatchNews(ctx context.Context, newsList []*model.NewsData) error {
	log.Printf("Starting batch news processing, count: %d", len(newsList))

	// 1. 批量数据清洗和验证
	cleanedNews := make([]*model.NewsData, 0, len(newsList))
	for _, news := range newsList {
		// 数据清洗
		cleanedItem, err := s.cleaner.CleanNewsData(news)
		if err != nil {
			log.Printf("Failed to clean news data, error: %v, news_id: %d", err, news.ID)
			continue
		}

		// 数据验证
		if err := s.validator.ValidateNewsData(cleanedItem); err != nil {
			log.Printf("News data validation failed, error: %v, news_id: %d", err, news.ID)
			continue
		}

		cleanedNews = append(cleanedNews, cleanedItem)
	}

	// 2. 批量去重
	deduplicatedNews := s.deduper.DeduplicateNewsData(cleanedNews)
	log.Printf("Deduplication completed, original_count: %d, deduplicated_count: %d",
		len(cleanedNews), len(deduplicatedNews))

	// 3. 批量NLP处理
	nlpResults, err := s.nlpProcessor.ProcessBatchNews(ctx, deduplicatedNews)
	if err != nil {
		log.Printf("Batch NLP processing failed, error: %v", err)
		// 降级到单个处理
		return s.processBatchNewsFallback(ctx, deduplicatedNews)
	}

	// 4. 更新新闻数据并保存
	successCount := 0
	for i, news := range deduplicatedNews {
		if i < len(nlpResults) && nlpResults[i] != nil {
			// 使用NLP结果更新新闻
			s.updateNewsWithNLPResult(news, nlpResults[i])
		} else {
			// NLP处理失败，使用基础股票关联
			if err := s.associateStocks(ctx, news); err != nil {
				log.Printf("Basic stock association failed, error: %v, news_id: %d", err, news.ID)
			}
		}

		// 数据质量检查
		qualityReport := s.qualityChecker.CheckNewsDataQuality([]*model.NewsData{news})
		if qualityReport.QualityScore < 0.6 {
			log.Printf("Low quality news detected, score: %f, title: %s",
				qualityReport.QualityScore, news.Title)
		}

		// 标记为已处理
		news.MarkAsProcessed()

		// 更新到数据库
		if err := s.newsRepo.Update(ctx, news); err != nil {
			log.Printf("Failed to update processed news, error: %v, news_id: %d", err, news.ID)
			continue
		}

		successCount++
	}

	log.Printf("Batch news processing completed, total_count: %d, success_count: %d, failure_count: %d",
		len(deduplicatedNews), successCount, len(deduplicatedNews)-successCount)

	return nil
}

// processBatchNewsFallback 批量处理降级方案
func (s *ProcessingService) processBatchNewsFallback(ctx context.Context, newsList []*model.NewsData) error {
	log.Printf("Using fallback processing for batch news, count: %d", len(newsList))

	for _, news := range newsList {
		if err := s.processNewsItem(ctx, news); err != nil {
			log.Printf("Failed to process news item in fallback, error: %v, news_id: %d", err, news.ID)
			continue
		}
	}

	return nil
}

// processNewsItem 处理单条新闻数据
// ProcessSingleNewsItem 处理单个新闻项
func (s *ProcessingService) ProcessSingleNewsItem(ctx context.Context, news *model.NewsData) error {
	return s.processNewsItem(ctx, news)
}

func (s *ProcessingService) processNewsItem(ctx context.Context, news *model.NewsData) error {
	// 1. 数据清洗
	cleanedNews, err := s.cleaner.CleanNewsData(news)
	if err != nil {
		return fmt.Errorf("failed to clean news data: %w", err)
	}

	// 2. 数据验证
	if err := s.validator.ValidateNewsData(cleanedNews); err != nil {
		return fmt.Errorf("news data validation failed: %w", err)
	}

	// 3. 数据去重检查（简化处理，实际应该与数据库中的数据比较）
	deduplicatedList := s.deduper.DeduplicateNewsData([]*model.NewsData{cleanedNews})
	if len(deduplicatedList) == 0 {
		log.Printf("Duplicate news found, skipping, title: %s", cleanedNews.Title)
		return nil
	}

	// 4. NLP处理（百度AI增强）
	nlpResult, err := s.nlpProcessor.ProcessNewsContent(ctx, cleanedNews)
	if err != nil {
		log.Printf("NLP processing failed, error: %v, news_id: %d", err, cleanedNews.ID)
		// NLP处理失败不阻断整个流程，使用基础股票关联处理
		if err := s.associateStocks(ctx, cleanedNews); err != nil {
			log.Printf("Basic stock association failed, error: %v, news_id: %d", err, cleanedNews.ID)
		}
	} else {
		// 使用NLP处理结果更新新闻数据
		s.updateNewsWithNLPResult(cleanedNews, nlpResult)
		log.Printf("NLP processing completed, news_id: %d, sentiment_score: %f, importance_level: %s, related_stocks_count: %d, keywords_count: %d, cache_hit: %t",
			cleanedNews.ID, nlpResult.SentimentScore, nlpResult.ImportanceLevel, len(nlpResult.RelatedStocks), len(nlpResult.Keywords), nlpResult.CacheHit)
	}

	// 5. 数据质量检查
	qualityReport := s.qualityChecker.CheckNewsDataQuality([]*model.NewsData{cleanedNews})
	qualityScore := qualityReport.QualityScore
	if qualityScore < 0.6 {
		log.Printf("Low quality news detected, score: %f, title: %s", qualityScore, cleanedNews.Title)
	}

	// 6. 标记为已处理并更新
	cleanedNews.MarkAsProcessed()

	// 7. 更新到数据库
	if err := s.newsRepo.Update(ctx, cleanedNews); err != nil {
		return fmt.Errorf("failed to update processed news: %w", err)
	}

	log.Printf("Successfully processed news, title: %s, id: %d", cleanedNews.Title, cleanedNews.ID)
	return nil
}

// updateNewsWithNLPResult 使用NLP处理结果更新新闻数据
func (s *ProcessingService) updateNewsWithNLPResult(news *model.NewsData, nlpResult *ProcessedResult) {
	// 更新关联股票
	if len(nlpResult.RelatedStocks) > 0 {
		news.RelatedStocks = model.StringSlice(nlpResult.RelatedStocks)
	}

	// 更新情感分析结果
	if nlpResult.SentimentScore != 0 {
		news.SentimentScore = &nlpResult.SentimentScore
		// 根据得分设置情感倾向
		if nlpResult.SentimentScore > 0.6 {
			sentiment := int8(model.SentimentPositive)
			news.Sentiment = &sentiment
		} else if nlpResult.SentimentScore < 0.4 {
			sentiment := int8(model.SentimentNegative)
			news.Sentiment = &sentiment
		} else {
			sentiment := int8(model.SentimentNeutral)
			news.Sentiment = &sentiment
		}
	}

	// 更新关键词 - 注意NewsData模型中没有Keywords字段，这里可能需要存储到其他地方或扩展模型
	// if len(nlpResult.Keywords) > 0 {
	//     news.Keywords = model.StringSlice(nlpResult.Keywords)
	// }

	// 更新重要性级别
	if nlpResult.ImportanceLevel != 0 {
		news.ImportanceLevel = int8(nlpResult.ImportanceLevel)
	}

	// 存储实体信息到扩展字段（如果模型支持）
	if len(nlpResult.Entities) > 0 {
		// 可以将实体信息序列化存储到某个字段中
		// 这里暂时跳过，需要根据实际模型结构调整
	}
}

// associateStocks 关联股票信息（基础版本，作为NLP处理失败时的备选方案）
func (s *ProcessingService) associateStocks(ctx context.Context, news *model.NewsData) error {
	// 从新闻标题和内容中提取股票代码和公司名称
	stockCodes := s.extractStockCodes(news.Title + " " + news.Content)

	// 验证股票代码是否存在
	validStockCodes := make([]string, 0)
	for _, code := range stockCodes {
		// TODO: 需要实现股票验证逻辑
		validStockCodes = append(validStockCodes, code)
	}

	// 更新新闻的关联股票信息
	if len(validStockCodes) > 0 {
		news.RelatedStocks = validStockCodes
	}

	return nil
}

// extractStockCodes 从文本中提取股票代码
func (s *ProcessingService) extractStockCodes(text string) []string {
	codes := make([]string, 0)

	// 简单的股票代码提取逻辑
	// A股代码格式：6位数字
	words := strings.Fields(text)
	for _, word := range words {
		// 去除标点符号
		word = strings.Trim(word, ".,!?;:()[]{}")

		// 检查是否为6位数字
		if len(word) == 6 {
			isDigit := true
			for _, char := range word {
				if char < '0' || char > '9' {
					isDigit = false
					break
				}
			}
			if isDigit {
				codes = append(codes, word)
			}
		}
	}

	return codes
}

// ProcessMarketData 处理行情数据
func (s *ProcessingService) ProcessMarketData(ctx context.Context, limit int) error {
	// TODO: 实现行情数据处理逻辑
	log.Printf("Processing market data with limit: %d", limit)
	return nil
}

// ProcessFinancialData 处理财务数据
func (s *ProcessingService) ProcessFinancialData(ctx context.Context, limit int) error {
	// TODO: 实现财务数据处理逻辑
	log.Printf("Processing financial data with limit: %d", limit)
	return nil
}

// GetProcessingStats 获取处理统计信息
func (s *ProcessingService) GetProcessingStats(ctx context.Context) (*ServiceProcessingStats, error) {
	// TODO: 实现统计信息获取逻辑
	return &ServiceProcessingStats{
		TotalProcessed:   0,
		SuccessCount:     0,
		FailureCount:     0,
		DuplicateCount:   0,
		LastProcessedAt:  time.Now(),
		ProcessingTimeMs: 0,
	}, nil
}

// ServiceProcessingStats 服务处理统计信息
type ServiceProcessingStats struct {
	TotalProcessed   int64     `json:"total_processed"`
	SuccessCount     int64     `json:"success_count"`
	FailureCount     int64     `json:"failure_count"`
	DuplicateCount   int64     `json:"duplicate_count"`
	LastProcessedAt  time.Time `json:"last_processed_at"`
	ProcessingTimeMs int64     `json:"processing_time_ms"`
}
