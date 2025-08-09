package processor

import (
	"context"
	"fmt"
	"time"

	"data-collection-system/internal/models"
	"data-collection-system/pkg/logger"
)

// DataProcessor 数据处理器接口
type DataProcessor interface {
	// ProcessMarketData 处理行情数据
	ProcessMarketData(ctx context.Context, data *models.MarketData) (*models.MarketData, error)
	// ProcessFinancialData 处理财务数据
	ProcessFinancialData(ctx context.Context, data *models.FinancialData) (*models.FinancialData, error)
	// ProcessNewsData 处理新闻数据
	ProcessNewsData(ctx context.Context, data *models.NewsData) (*models.NewsData, error)
	// ProcessMacroData 处理宏观数据
	ProcessMacroData(ctx context.Context, data *models.MacroData) (*models.MacroData, error)
}

// DataValidator 数据验证器接口
type DataValidator interface {
	// ValidateMarketData 验证行情数据
	ValidateMarketData(data *models.MarketData) error
	// ValidateFinancialData 验证财务数据
	ValidateFinancialData(data *models.FinancialData) error
	// ValidateNewsData 验证新闻数据
	ValidateNewsData(data *models.NewsData) error
	// ValidateMacroData 验证宏观数据
	ValidateMacroData(data *models.MacroData) error
}

// DataCleaner 数据清洗器接口
type DataCleaner interface {
	// CleanMarketData 清洗行情数据
	CleanMarketData(data *models.MarketData) *models.MarketData
	// CleanFinancialData 清洗财务数据
	CleanFinancialData(data *models.FinancialData) *models.FinancialData
	// CleanNewsData 清洗新闻数据
	CleanNewsData(data *models.NewsData) *models.NewsData
	// CleanMacroData 清洗宏观数据
	CleanMacroData(data *models.MacroData) *models.MacroData
}

// DataDeduplicator 数据去重器接口
type DataDeduplicator interface {
	// CheckDuplicate 检查数据是否重复
	CheckDuplicate(ctx context.Context, dataType string, key string) (bool, error)
	// MarkProcessed 标记数据已处理
	MarkProcessed(ctx context.Context, dataType string, key string) error
}

// QualityChecker 数据质量检查器接口
type QualityChecker interface {
	// CheckQuality 检查数据质量
	CheckQuality(ctx context.Context, dataType string, data interface{}) (*QualityReport, error)
}

// QualityReport 数据质量报告
type QualityReport struct {
	DataType     string                 `json:"data_type"`
	QualityScore float64                `json:"quality_score"`
	Issues       []QualityIssue         `json:"issues"`
	Metrics      map[string]interface{} `json:"metrics"`
	Timestamp    time.Time              `json:"timestamp"`
}

// QualityIssue 数据质量问题
type QualityIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Field       string `json:"field,omitempty"`
	Value       string `json:"value,omitempty"`
}

// ProcessorManager 数据处理管理器
type ProcessorManager struct {
	processor     DataProcessor
	validator     DataValidator
	cleaner       DataCleaner
	deduplicator  DataDeduplicator
	qualityChecker QualityChecker
}

// NewProcessorManager 创建数据处理管理器
func NewProcessorManager(
	processor DataProcessor,
	validator DataValidator,
	cleaner DataCleaner,
	deduplicator DataDeduplicator,
	qualityChecker QualityChecker,
) *ProcessorManager {
	return &ProcessorManager{
		processor:      processor,
		validator:      validator,
		cleaner:        cleaner,
		deduplicator:   deduplicator,
		qualityChecker: qualityChecker,
	}
}

// ProcessData 处理数据的通用方法
func (pm *ProcessorManager) ProcessData(ctx context.Context, dataType string, data interface{}) (interface{}, error) {
	switch dataType {
	case "market":
		marketData, ok := data.(*models.MarketData)
		if !ok {
			return nil, fmt.Errorf("invalid market data type")
		}
		return pm.processMarketData(ctx, marketData)
	case "financial":
		financialData, ok := data.(*models.FinancialData)
		if !ok {
			return nil, fmt.Errorf("invalid financial data type")
		}
		return pm.processFinancialData(ctx, financialData)
	case "news":
		newsData, ok := data.(*models.NewsData)
		if !ok {
			return nil, fmt.Errorf("invalid news data type")
		}
		return pm.processNewsData(ctx, newsData)
	case "macro":
		macroData, ok := data.(*models.MacroData)
		if !ok {
			return nil, fmt.Errorf("invalid macro data type")
		}
		return pm.processMacroData(ctx, macroData)
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
}

// processMarketData 处理行情数据
func (pm *ProcessorManager) processMarketData(ctx context.Context, data *models.MarketData) (*models.MarketData, error) {
	// 1. 数据清洗
	cleanedData := pm.cleaner.CleanMarketData(data)
	
	// 2. 数据验证
	if err := pm.validator.ValidateMarketData(cleanedData); err != nil {
		logger.Error("Market data validation failed", "error", err, "symbol", cleanedData.Symbol)
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// 3. 去重检查
	key := fmt.Sprintf("%s_%s_%s", cleanedData.Symbol, cleanedData.TradeDate.Format("2006-01-02"), cleanedData.Period)
	if isDuplicate, err := pm.deduplicator.CheckDuplicate(ctx, "market", key); err != nil {
		return nil, fmt.Errorf("duplicate check failed: %w", err)
	} else if isDuplicate {
		logger.Info("Duplicate market data detected", "key", key)
		return cleanedData, nil // 返回清洗后的数据，但不重复处理
	}
	
	// 4. 数据处理
	processedData, err := pm.processor.ProcessMarketData(ctx, cleanedData)
	if err != nil {
		return nil, fmt.Errorf("processing failed: %w", err)
	}
	
	// 5. 质量检查
	qualityReport, err := pm.qualityChecker.CheckQuality(ctx, "market", processedData)
	if err != nil {
		logger.Warn("Quality check failed", "error", err)
	} else if qualityReport.QualityScore < 0.8 {
		logger.Warn("Low quality data detected", "score", qualityReport.QualityScore, "issues", qualityReport.Issues)
	}
	
	// 6. 标记已处理
	if err := pm.deduplicator.MarkProcessed(ctx, "market", key); err != nil {
		logger.Warn("Failed to mark as processed", "error", err, "key", key)
	}
	
	return processedData, nil
}

// processFinancialData 处理财务数据
func (pm *ProcessorManager) processFinancialData(ctx context.Context, data *models.FinancialData) (*models.FinancialData, error) {
	// 1. 数据清洗
	cleanedData := pm.cleaner.CleanFinancialData(data)
	
	// 2. 数据验证
	if err := pm.validator.ValidateFinancialData(cleanedData); err != nil {
		logger.Error("Financial data validation failed", "error", err, "symbol", cleanedData.Symbol)
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// 3. 去重检查
	key := fmt.Sprintf("%s_%s_%s", cleanedData.Symbol, cleanedData.ReportDate.Format("2006-01-02"), cleanedData.ReportType)
	if isDuplicate, err := pm.deduplicator.CheckDuplicate(ctx, "financial", key); err != nil {
		return nil, fmt.Errorf("duplicate check failed: %w", err)
	} else if isDuplicate {
		logger.Info("Duplicate financial data detected", "key", key)
		return cleanedData, nil
	}
	
	// 4. 数据处理
	processedData, err := pm.processor.ProcessFinancialData(ctx, cleanedData)
	if err != nil {
		return nil, fmt.Errorf("processing failed: %w", err)
	}
	
	// 5. 质量检查
	qualityReport, err := pm.qualityChecker.CheckQuality(ctx, "financial", processedData)
	if err != nil {
		logger.Warn("Quality check failed", "error", err)
	} else if qualityReport.QualityScore < 0.8 {
		logger.Warn("Low quality data detected", "score", qualityReport.QualityScore, "issues", qualityReport.Issues)
	}
	
	// 6. 标记已处理
	if err := pm.deduplicator.MarkProcessed(ctx, "financial", key); err != nil {
		logger.Warn("Failed to mark as processed", "error", err, "key", key)
	}
	
	return processedData, nil
}

// processNewsData 处理新闻数据
func (pm *ProcessorManager) processNewsData(ctx context.Context, data *models.NewsData) (*models.NewsData, error) {
	// 1. 数据清洗
	cleanedData := pm.cleaner.CleanNewsData(data)
	
	// 2. 数据验证
	if err := pm.validator.ValidateNewsData(cleanedData); err != nil {
		logger.Error("News data validation failed", "error", err, "title", cleanedData.Title)
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// 3. 去重检查
	key := fmt.Sprintf("%s_%s", cleanedData.Title, cleanedData.PublishTime.Format("2006-01-02 15:04:05"))
	if isDuplicate, err := pm.deduplicator.CheckDuplicate(ctx, "news", key); err != nil {
		return nil, fmt.Errorf("duplicate check failed: %w", err)
	} else if isDuplicate {
		logger.Info("Duplicate news data detected", "key", key)
		return cleanedData, nil
	}
	
	// 4. 数据处理
	processedData, err := pm.processor.ProcessNewsData(ctx, cleanedData)
	if err != nil {
		return nil, fmt.Errorf("processing failed: %w", err)
	}
	
	// 5. 质量检查
	qualityReport, err := pm.qualityChecker.CheckQuality(ctx, "news", processedData)
	if err != nil {
		logger.Warn("Quality check failed", "error", err)
	} else if qualityReport.QualityScore < 0.8 {
		logger.Warn("Low quality data detected", "score", qualityReport.QualityScore, "issues", qualityReport.Issues)
	}
	
	// 6. 标记已处理
	if err := pm.deduplicator.MarkProcessed(ctx, "news", key); err != nil {
		logger.Warn("Failed to mark as processed", "error", err, "key", key)
	}
	
	return processedData, nil
}

// processMacroData 处理宏观数据
func (pm *ProcessorManager) processMacroData(ctx context.Context, data *models.MacroData) (*models.MacroData, error) {
	// 1. 数据清洗
	cleanedData := pm.cleaner.CleanMacroData(data)
	
	// 2. 数据验证
	if err := pm.validator.ValidateMacroData(cleanedData); err != nil {
		logger.Error("Macro data validation failed", "error", err, "indicator", cleanedData.IndicatorCode)
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// 3. 去重检查
	key := fmt.Sprintf("%s_%s", cleanedData.IndicatorCode, cleanedData.DataDate.Format("2006-01-02"))
	if isDuplicate, err := pm.deduplicator.CheckDuplicate(ctx, "macro", key); err != nil {
		return nil, fmt.Errorf("duplicate check failed: %w", err)
	} else if isDuplicate {
		logger.Info("Duplicate macro data detected", "key", key)
		return cleanedData, nil
	}
	
	// 4. 数据处理
	processedData, err := pm.processor.ProcessMacroData(ctx, cleanedData)
	if err != nil {
		return nil, fmt.Errorf("processing failed: %w", err)
	}
	
	// 5. 质量检查
	qualityReport, err := pm.qualityChecker.CheckQuality(ctx, "macro", processedData)
	if err != nil {
		logger.Warn("Quality check failed", "error", err)
	} else if qualityReport.QualityScore < 0.8 {
		logger.Warn("Low quality data detected", "score", qualityReport.QualityScore, "issues", qualityReport.Issues)
	}
	
	// 6. 标记已处理
	if err := pm.deduplicator.MarkProcessed(ctx, "macro", key); err != nil {
		logger.Warn("Failed to mark as processed", "error", err, "key", key)
	}
	
	return processedData, nil
}