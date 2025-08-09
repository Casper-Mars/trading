package dao

import (
	"context"
	"time"

	"data-collection-system/model"
	"data-collection-system/pkg/errors"

	"gorm.io/gorm"
)

// financialDataRepository 财务数据仓库实现
type financialDataRepository struct {
	db *gorm.DB
}

// NewFinancialDataRepository 创建财务数据仓库
func NewFinancialDataRepository(db *gorm.DB) FinancialDataRepository {
	return &financialDataRepository{
		db: db,
	}
}

// Create 创建财务数据记录
func (r *financialDataRepository) Create(ctx context.Context, data *model.FinancialData) error {
	if err := r.db.WithContext(ctx).Create(data).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to create financial data")
	}
	return nil
}

// BatchCreate 批量创建财务数据记录
func (r *financialDataRepository) BatchCreate(ctx context.Context, dataList []*model.FinancialData) error {
	if len(dataList) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).CreateInBatches(dataList, 100).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to batch create financial data")
	}
	return nil
}

// GetByID 根据ID获取财务数据
func (r *financialDataRepository) GetByID(ctx context.Context, id uint64) (*model.FinancialData, error) {
	var data model.FinancialData
	if err := r.db.WithContext(ctx).First(&data, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeDataNotFound, "financial data not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get financial data by id")
	}
	return &data, nil
}

// GetBySymbolAndReport 根据股票代码、报告日期和报告类型获取财务数据
func (r *financialDataRepository) GetBySymbolAndReport(ctx context.Context, symbol string, reportDate time.Time, reportType string) (*model.FinancialData, error) {
	var data model.FinancialData
	if err := r.db.WithContext(ctx).Where("symbol = ? AND report_date = ? AND report_type = ?", symbol, reportDate, reportType).First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeDataNotFound, "financial data not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get financial data by symbol and report")
	}
	return &data, nil
}

// GetBySymbol 根据股票代码获取财务数据列表
func (r *financialDataRepository) GetBySymbol(ctx context.Context, symbol string, limit int) ([]*model.FinancialData, error) {
	var dataList []*model.FinancialData
	query := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("report_date DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&dataList).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get financial data by symbol")
	}
	return dataList, nil
}

// GetByReportType 根据报告类型获取财务数据列表
func (r *financialDataRepository) GetByReportType(ctx context.Context, reportType string, limit int) ([]*model.FinancialData, error) {
	var dataList []*model.FinancialData
	query := r.db.WithContext(ctx).Where("report_type = ?", reportType).Order("report_date DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&dataList).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get financial data by report type")
	}
	return dataList, nil
}

// GetLatest 获取最新的财务数据
func (r *financialDataRepository) GetLatest(ctx context.Context, symbol string) (*model.FinancialData, error) {
	var data model.FinancialData
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Order("report_date DESC").First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeDataNotFound, "financial data not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get latest financial data")
	}
	return &data, nil
}

// Update 更新财务数据
func (r *financialDataRepository) Update(ctx context.Context, data *model.FinancialData) error {
	if err := r.db.WithContext(ctx).Save(data).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to update financial data")
	}
	return nil
}

// Delete 删除财务数据
func (r *financialDataRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&model.FinancialData{}, id).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to delete financial data")
	}
	return nil
}

// Count 获取财务数据总数
func (r *financialDataRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.FinancialData{}).Count(&count).Error; err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeDatabase, "failed to count financial data")
	}
	return count, nil
}