package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"data-collection-system/internal/models"
)

// marketDataDAO 行情数据访问层实现
type marketDataDAO struct {
	db *gorm.DB
}

// NewMarketDataDAO 创建行情数据DAO实例
func NewMarketDataDAO(db *gorm.DB) MarketDataDAO {
	return &marketDataDAO{db: db}
}

// Create 创建行情数据记录
func (m *marketDataDAO) Create(ctx context.Context, data *models.MarketData) error {
	if err := m.db.WithContext(ctx).Create(data).Error; err != nil {
		return fmt.Errorf("failed to create market data: %w", err)
	}
	return nil
}

// BatchCreate 批量创建行情数据记录
func (m *marketDataDAO) BatchCreate(ctx context.Context, data []*models.MarketData) error {
	if len(data) == 0 {
		return nil
	}
	if err := m.db.WithContext(ctx).CreateInBatches(data, 1000).Error; err != nil {
		return fmt.Errorf("failed to batch create market data: %w", err)
	}
	return nil
}

// GetBySymbol 根据股票代码获取行情数据
func (m *marketDataDAO) GetBySymbol(ctx context.Context, symbol string, startDate, endDate time.Time, period string, limit, offset int) ([]*models.MarketData, error) {
	var data []*models.MarketData
	query := m.db.WithContext(ctx).Where("symbol = ?", symbol)
	
	if !startDate.IsZero() {
		query = query.Where("trade_date >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("trade_date <= ?", endDate)
	}
	if period != "" {
		query = query.Where("period = ?", period)
	}
	
	query = query.Order("trade_date DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&data).Error; err != nil {
		return nil, fmt.Errorf("failed to get market data by symbol: %w", err)
	}
	return data, nil
}

// GetLatest 获取最新行情数据
func (m *marketDataDAO) GetLatest(ctx context.Context, symbol string, period string) (*models.MarketData, error) {
	var data models.MarketData
	query := m.db.WithContext(ctx).Where("symbol = ?", symbol)
	if period != "" {
		query = query.Where("period = ?", period)
	}
	query = query.Order("trade_date DESC, trade_time DESC")
	
	if err := query.First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest market data: %w", err)
	}
	return &data, nil
}

// GetByDate 根据日期获取行情数据
func (m *marketDataDAO) GetByDate(ctx context.Context, date time.Time, period string, limit, offset int) ([]*models.MarketData, error) {
	var data []*models.MarketData
	query := m.db.WithContext(ctx).Where("trade_date = ?", date)
	if period != "" {
		query = query.Where("period = ?", period)
	}
	
	query = query.Order("symbol ASC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&data).Error; err != nil {
		return nil, fmt.Errorf("failed to get market data by date: %w", err)
	}
	return data, nil
}

// Update 更新行情数据
func (m *marketDataDAO) Update(ctx context.Context, data *models.MarketData) error {
	if err := m.db.WithContext(ctx).Save(data).Error; err != nil {
		return fmt.Errorf("failed to update market data: %w", err)
	}
	return nil
}

// Delete 删除行情数据记录
func (m *marketDataDAO) Delete(ctx context.Context, id uint64) error {
	if err := m.db.WithContext(ctx).Delete(&models.MarketData{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete market data: %w", err)
	}
	return nil
}

// DeleteBySymbolAndDateRange 根据股票代码和日期范围删除行情数据
func (m *marketDataDAO) DeleteBySymbolAndDateRange(ctx context.Context, symbol string, startDate, endDate time.Time) error {
	query := m.db.WithContext(ctx).Where("symbol = ?", symbol)
	if !startDate.IsZero() {
		query = query.Where("trade_date >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("trade_date <= ?", endDate)
	}
	
	if err := query.Delete(&models.MarketData{}).Error; err != nil {
		return fmt.Errorf("failed to delete market data by symbol and date range: %w", err)
	}
	return nil
}

// Count 获取行情数据总数
func (m *marketDataDAO) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := m.db.WithContext(ctx).Model(&models.MarketData{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count market data: %w", err)
	}
	return count, nil
}