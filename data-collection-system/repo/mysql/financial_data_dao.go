package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"data-collection-system/model"
)

// financialDataDAO 财务数据访问层实现
type financialDataDAO struct {
	db *gorm.DB
}

// NewFinancialDataDAO 创建财务数据DAO实例
func NewFinancialDataDAO(db *gorm.DB) FinancialDataDAO {
	return &financialDataDAO{db: db}
}

// Create 创建财务数据记录
func (f *financialDataDAO) Create(ctx context.Context, data *model.FinancialData) error {
	if err := f.db.WithContext(ctx).Create(data).Error; err != nil {
		return fmt.Errorf("failed to create financial data: %w", err)
	}
	return nil
}

// BatchCreate 批量创建财务数据记录
func (f *financialDataDAO) BatchCreate(ctx context.Context, data []*model.FinancialData) error {
	if len(data) == 0 {
		return nil
	}
	if err := f.db.WithContext(ctx).CreateInBatches(data, 1000).Error; err != nil {
		return fmt.Errorf("failed to batch create financial data: %w", err)
	}
	return nil
}

// GetBySymbol 根据股票代码获取财务数据
func (f *financialDataDAO) GetBySymbol(ctx context.Context, symbol string, startDate, endDate time.Time, reportType string, limit, offset int) ([]*model.FinancialData, error) {
	var data []*model.FinancialData
	query := f.db.WithContext(ctx).Where("symbol = ?", symbol)
	
	if !startDate.IsZero() {
		query = query.Where("report_date >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("report_date <= ?", endDate)
	}
	if reportType != "" {
		query = query.Where("report_type = ?", reportType)
	}
	
	query = query.Order("report_date DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&data).Error; err != nil {
		return nil, fmt.Errorf("failed to get financial data by symbol: %w", err)
	}
	return data, nil
}

// GetLatest 获取最新财务数据
func (f *financialDataDAO) GetLatest(ctx context.Context, symbol string, reportType string) (*model.FinancialData, error) {
	var data model.FinancialData
	query := f.db.WithContext(ctx).Where("symbol = ?", symbol)
	if reportType != "" {
		query = query.Where("report_type = ?", reportType)
	}
	query = query.Order("report_date DESC")
	
	if err := query.First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest financial data: %w", err)
	}
	return &data, nil
}

// GetByReportDate 根据报告日期获取财务数据
func (f *financialDataDAO) GetByReportDate(ctx context.Context, reportDate time.Time, reportType string, limit, offset int) ([]*model.FinancialData, error) {
	var data []*model.FinancialData
	query := f.db.WithContext(ctx).Where("report_date = ?", reportDate)
	if reportType != "" {
		query = query.Where("report_type = ?", reportType)
	}
	
	query = query.Order("symbol ASC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&data).Error; err != nil {
		return nil, fmt.Errorf("failed to get financial data by report date: %w", err)
	}
	return data, nil
}

// Update 更新财务数据
func (f *financialDataDAO) Update(ctx context.Context, data *model.FinancialData) error {
	if err := f.db.WithContext(ctx).Save(data).Error; err != nil {
		return fmt.Errorf("failed to update financial data: %w", err)
	}
	return nil
}

// Delete 删除财务数据记录
func (f *financialDataDAO) Delete(ctx context.Context, id uint64) error {
	if err := f.db.WithContext(ctx).Delete(&model.FinancialData{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete financial data: %w", err)
	}
	return nil
}

// Count 获取财务数据总数
func (f *financialDataDAO) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := f.db.WithContext(ctx).Model(&model.FinancialData{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count financial data: %w", err)
	}
	return count, nil
}