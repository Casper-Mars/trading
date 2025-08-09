package dao

import (
	"context"
	"time"

	"data-collection-system/model"
	"data-collection-system/pkg/errors"

	"gorm.io/gorm"
)

// marketDataRepository 行情数据仓库实现
type marketDataRepository struct {
	db *gorm.DB
}

// NewMarketDataRepository 创建行情数据仓库
func NewMarketDataRepository(db *gorm.DB) MarketDataRepository {
	return &marketDataRepository{
		db: db,
	}
}

// Create 创建行情数据记录
func (r *marketDataRepository) Create(ctx context.Context, data *model.MarketData) error {
	if err := r.db.WithContext(ctx).Create(data).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to create market data")
	}
	return nil
}

// BatchCreate 批量创建行情数据记录
func (r *marketDataRepository) BatchCreate(ctx context.Context, dataList []*model.MarketData) error {
	if len(dataList) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).CreateInBatches(dataList, 100).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to batch create market data")
	}
	return nil
}

// GetByID 根据ID获取行情数据
func (r *marketDataRepository) GetByID(ctx context.Context, id uint64) (*model.MarketData, error) {
	var data model.MarketData
	if err := r.db.WithContext(ctx).First(&data, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeDataNotFound, "market data not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get market data by id")
	}
	return &data, nil
}

// GetBySymbolAndTime 根据股票代码、交易时间和周期获取行情数据
func (r *marketDataRepository) GetBySymbolAndTime(ctx context.Context, symbol string, tradeTime time.Time, period string) (*model.MarketData, error) {
	var data model.MarketData
	if err := r.db.WithContext(ctx).Where("symbol = ? AND trade_time = ? AND period = ?", symbol, tradeTime, period).First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeDataNotFound, "market data not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get market data by symbol and time")
	}
	return &data, nil
}

// GetByTimeRange 根据时间范围获取行情数据
func (r *marketDataRepository) GetByTimeRange(ctx context.Context, symbol string, startTime, endTime time.Time, period string) ([]*model.MarketData, error) {
	var dataList []*model.MarketData
	query := r.db.WithContext(ctx).Where("symbol = ? AND period = ? AND trade_time >= ? AND trade_time <= ?", symbol, period, startTime, endTime)
	if err := query.Order("trade_time ASC").Find(&dataList).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get market data by time range")
	}
	return dataList, nil
}

// GetLatest 获取最新的行情数据
func (r *marketDataRepository) GetLatest(ctx context.Context, symbol string, period string) (*model.MarketData, error) {
	var data model.MarketData
	if err := r.db.WithContext(ctx).Where("symbol = ? AND period = ?", symbol, period).Order("trade_time DESC").First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeDataNotFound, "market data not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get latest market data")
	}
	return &data, nil
}

// Update 更新行情数据
func (r *marketDataRepository) Update(ctx context.Context, data *model.MarketData) error {
	if err := r.db.WithContext(ctx).Save(data).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to update market data")
	}
	return nil
}

// Delete 删除行情数据
func (r *marketDataRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&model.MarketData{}, id).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to delete market data")
	}
	return nil
}

// DeleteByTimeRange 根据时间范围删除行情数据
func (r *marketDataRepository) DeleteByTimeRange(ctx context.Context, symbol string, startTime, endTime time.Time, period string) error {
	if err := r.db.WithContext(ctx).Where("symbol = ? AND period = ? AND trade_time >= ? AND trade_time <= ?", symbol, period, startTime, endTime).Delete(&model.MarketData{}).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to delete market data by time range")
	}
	return nil
}

// Count 获取行情数据总数
func (r *marketDataRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.MarketData{}).Count(&count).Error; err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeDatabase, "failed to count market data")
	}
	return count, nil
}