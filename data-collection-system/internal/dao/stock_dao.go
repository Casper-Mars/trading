package dao

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"data-collection-system/internal/models"
)

// stockDAO 股票数据访问层实现
type stockDAO struct {
	db *gorm.DB
}

// NewStockDAO 创建股票DAO实例
func NewStockDAO(db *gorm.DB) StockDAO {
	return &stockDAO{db: db}
}

// Create 创建股票记录
func (s *stockDAO) Create(ctx context.Context, stock *models.Stock) error {
	if err := s.db.WithContext(ctx).Create(stock).Error; err != nil {
		return fmt.Errorf("failed to create stock: %w", err)
	}
	return nil
}

// GetBySymbol 根据股票代码获取股票信息
func (s *stockDAO) GetBySymbol(ctx context.Context, symbol string) (*models.Stock, error) {
	var stock models.Stock
	if err := s.db.WithContext(ctx).Where("symbol = ?", symbol).First(&stock).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get stock by symbol: %w", err)
	}
	return &stock, nil
}

// GetByExchange 根据交易所获取股票列表
func (s *stockDAO) GetByExchange(ctx context.Context, exchange string, limit, offset int) ([]*models.Stock, error) {
	var stocks []*models.Stock
	query := s.db.WithContext(ctx).Where("exchange = ?")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&stocks, exchange).Error; err != nil {
		return nil, fmt.Errorf("failed to get stocks by exchange: %w", err)
	}
	return stocks, nil
}

// GetByIndustry 根据行业获取股票列表
func (s *stockDAO) GetByIndustry(ctx context.Context, industry string, limit, offset int) ([]*models.Stock, error) {
	var stocks []*models.Stock
	query := s.db.WithContext(ctx).Where("industry = ?")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&stocks, industry).Error; err != nil {
		return nil, fmt.Errorf("failed to get stocks by industry: %w", err)
	}
	return stocks, nil
}

// GetActiveStocks 获取活跃股票列表
func (s *stockDAO) GetActiveStocks(ctx context.Context, limit, offset int) ([]*models.Stock, error) {
	var stocks []*models.Stock
	query := s.db.WithContext(ctx).Where("status = ?", models.StockStatusActive)
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&stocks).Error; err != nil {
		return nil, fmt.Errorf("failed to get active stocks: %w", err)
	}
	return stocks, nil
}

// Update 更新股票信息
func (s *stockDAO) Update(ctx context.Context, stock *models.Stock) error {
	if err := s.db.WithContext(ctx).Save(stock).Error; err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}
	return nil
}

// Delete 删除股票记录
func (s *stockDAO) Delete(ctx context.Context, symbol string) error {
	if err := s.db.WithContext(ctx).Where("symbol = ?", symbol).Delete(&models.Stock{}).Error; err != nil {
		return fmt.Errorf("failed to delete stock: %w", err)
	}
	return nil
}

// Count 获取股票总数
func (s *stockDAO) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Stock{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count stocks: %w", err)
	}
	return count, nil
}