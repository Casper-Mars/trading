package dao

import (
	"context"
	"fmt"

	"data-collection-system/model"
	"data-collection-system/pkg/errors"

	"gorm.io/gorm"
)

// stockRepository 股票数据仓库实现
type stockRepository struct {
	db *gorm.DB
}

// NewStockRepository 创建股票数据仓库
func NewStockRepository(db *gorm.DB) StockRepository {
	return &stockRepository{
		db: db,
	}
}

// Create 创建股票记录
func (r *stockRepository) Create(ctx context.Context, stock *model.Stock) error {
	if err := r.db.WithContext(ctx).Create(stock).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to create stock")
	}
	return nil
}

// BatchCreate 批量创建股票记录
func (r *stockRepository) BatchCreate(ctx context.Context, stocks []*model.Stock) error {
	if len(stocks) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).CreateInBatches(stocks, 100).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to batch create stocks")
	}
	return nil
}

// GetByID 根据ID获取股票
func (r *stockRepository) GetByID(ctx context.Context, id uint64) (*model.Stock, error) {
	var stock model.Stock
	if err := r.db.WithContext(ctx).First(&stock, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeDataNotFound, "stock not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get stock by id")
	}
	return &stock, nil
}

// GetBySymbol 根据股票代码获取股票
func (r *stockRepository) GetBySymbol(ctx context.Context, symbol string) (*model.Stock, error) {
	var stock model.Stock
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&stock).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeDataNotFound, "stock not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get stock by symbol")
	}
	return &stock, nil
}

// List 获取股票列表（TushareService使用）
func (r *stockRepository) List(ctx context.Context, offset, limit int) ([]*model.Stock, error) {
	var stocks []*model.Stock
	if err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&stocks).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to list stocks")
	}
	return stocks, nil
}

// ListWithPagination 分页获取股票列表
func (r *stockRepository) ListWithPagination(ctx context.Context, page, pageSize int) ([]*model.Stock, int64, error) {
	var stocks []*model.Stock
	var total int64

	// 计算总数
	if err := r.db.WithContext(ctx).Model(&model.Stock{}).Count(&total).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeDatabase, "failed to count stocks")
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(&stocks).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeDatabase, "failed to list stocks")
	}

	return stocks, total, nil
}

// Update 更新股票信息
func (r *stockRepository) Update(ctx context.Context, stock *model.Stock) error {
	if err := r.db.WithContext(ctx).Save(stock).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to update stock")
	}
	return nil
}

// Delete 删除股票
func (r *stockRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&model.Stock{}, id).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to delete stock")
	}
	return nil
}

// GetByExchange 根据交易所获取股票列表
func (r *stockRepository) GetByExchange(ctx context.Context, exchange string) ([]*model.Stock, error) {
	var stocks []*model.Stock
	if err := r.db.WithContext(ctx).Where("exchange = ?", exchange).Find(&stocks).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get stocks by exchange")
	}
	return stocks, nil
}

// GetByIndustry 根据行业获取股票列表
func (r *stockRepository) GetByIndustry(ctx context.Context, industry string) ([]*model.Stock, error) {
	var stocks []*model.Stock
	if err := r.db.WithContext(ctx).Where("industry = ?", industry).Find(&stocks).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get stocks by industry")
	}
	return stocks, nil
}

// GetBySector 根据板块获取股票列表
func (r *stockRepository) GetBySector(ctx context.Context, sector string) ([]*model.Stock, error) {
	var stocks []*model.Stock
	if err := r.db.WithContext(ctx).Where("sector = ?", sector).Find(&stocks).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get stocks by sector")
	}
	return stocks, nil
}

// GetActiveStocks 获取正常交易的股票列表
func (r *stockRepository) GetActiveStocks(ctx context.Context) ([]*model.Stock, error) {
	var stocks []*model.Stock
	if err := r.db.WithContext(ctx).Where("status = ?", model.StockStatusActive).Find(&stocks).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get active stocks")
	}
	return stocks, nil
}

// Search 搜索股票（根据股票代码或名称）
func (r *stockRepository) Search(ctx context.Context, keyword string) ([]*model.Stock, error) {
	var stocks []*model.Stock
	searchPattern := fmt.Sprintf("%%%s%%", keyword)
	if err := r.db.WithContext(ctx).Where("symbol LIKE ? OR name LIKE ?", searchPattern, searchPattern).Find(&stocks).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to search stocks")
	}
	return stocks, nil
}

// Count 获取股票总数
func (r *stockRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Stock{}).Count(&count).Error; err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeDatabase, "failed to count stocks")
	}
	return count, nil
}