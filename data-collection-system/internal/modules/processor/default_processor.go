package processor

import (
	"context"
	"data-collection-system/internal/models"
)

// DefaultProcessor 默认数据处理器
type DefaultProcessor struct {
	newsProcessor *NewsProcessor
}

// NewDefaultProcessor 创建默认处理器
func NewDefaultProcessor() *DefaultProcessor {
	return &DefaultProcessor{
		newsProcessor: NewNewsProcessor(),
	}
}

// ProcessMarketData 处理行情数据
func (dp *DefaultProcessor) ProcessMarketData(ctx context.Context, data *models.MarketData) (*models.MarketData, error) {
	// 基础处理逻辑
	return data, nil
}

// ProcessFinancialData 处理财务数据
func (dp *DefaultProcessor) ProcessFinancialData(ctx context.Context, data *models.FinancialData) (*models.FinancialData, error) {
	// 基础处理逻辑
	return data, nil
}

// ProcessNewsData 处理新闻数据
func (dp *DefaultProcessor) ProcessNewsData(ctx context.Context, data *models.NewsData) (*models.NewsData, error) {
	err := dp.newsProcessor.ProcessNews(ctx, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// ProcessMacroData 处理宏观数据
func (dp *DefaultProcessor) ProcessMacroData(ctx context.Context, data *models.MacroData) (*models.MacroData, error) {
	// 基础处理逻辑
	return data, nil
}