package processing

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"data-collection-system/model"
	dao "data-collection-system/repo/mysql"
)

// ProcessingService 数据处理服务
type ProcessingService struct {
	newsRepo     dao.NewsDataDAO
	marketRepo   dao.MarketDataDAO
	financialRepo dao.FinancialDataDAO
	cleaner    *DataCleaner
	validator  *DataValidator
	deduper    *DataDeduplicator
	qualityChecker *QualityChecker
}

// NewProcessingService 创建数据处理服务实例
func NewProcessingService(newsRepo dao.NewsDataDAO, marketRepo dao.MarketDataDAO, financialRepo dao.FinancialDataDAO) *ProcessingService {
	return &ProcessingService{
		newsRepo:     newsRepo,
		marketRepo:   marketRepo,
		financialRepo: financialRepo,
		cleaner:    NewDataCleaner(),
		validator:  NewDataValidator(),
		deduper:    NewDataDeduplicator(),
		qualityChecker: NewQualityChecker(),
	}
}

// ProcessNewsData 处理新闻数据
func (s *ProcessingService) ProcessNewsData(ctx context.Context, limit int) error {
	// 获取未处理的新闻数据（使用时间范围查询最近的数据）
	startTime := time.Now().Add(-24 * time.Hour) // 最近24小时
	endTime := time.Now()
	newsList, err := s.newsRepo.GetByTimeRange(ctx, startTime, endTime, 100, 0)
	if err != nil {
		return fmt.Errorf("failed to get unprocessed news: %w", err)
	}

	log.Printf("Processing %d news items", len(newsList))

	for _, news := range newsList {
		if err := s.processNewsItem(ctx, news); err != nil {
			log.Printf("Failed to process news item %d: %v", news.ID, err)
			continue
		}
	}

	return nil
}

// processNewsItem 处理单条新闻数据
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
		log.Printf("Duplicate news found, skipping: %s", cleanedNews.Title)
		return nil
	}

	// 4. 股票关联处理
	if err := s.associateStocks(ctx, cleanedNews); err != nil {
		log.Printf("Failed to associate stocks for news %d: %v", cleanedNews.ID, err)
		// 不阻断处理流程
	}

	// 5. 数据质量检查
	qualityReport := s.qualityChecker.CheckNewsDataQuality([]*model.NewsData{cleanedNews})
	qualityScore := qualityReport.QualityScore
	if qualityScore < 0.6 {
		log.Printf("Low quality news detected (score: %.2f): %s", qualityScore, cleanedNews.Title)
	}

	// 6. 标记为已处理并更新
	cleanedNews.MarkAsProcessed()

	// 7. 更新到数据库
	if err := s.newsRepo.Update(ctx, cleanedNews); err != nil {
		return fmt.Errorf("failed to update processed news: %w", err)
	}

	log.Printf("Successfully processed news: %s", cleanedNews.Title)
	return nil
}

// associateStocks 关联股票信息
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