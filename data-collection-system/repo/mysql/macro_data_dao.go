package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"data-collection-system/model"
)

// macroDataDAO 宏观数据访问层实现
type macroDataDAO struct {
	db *gorm.DB
}

// NewMacroDataDAO 创建宏观数据DAO实例
func NewMacroDataDAO(db *gorm.DB) MacroDataDAO {
	return &macroDataDAO{db: db}
}

// Create 创建宏观数据记录
func (m *macroDataDAO) Create(ctx context.Context, data *model.MacroData) error {
	if err := m.db.WithContext(ctx).Create(data).Error; err != nil {
		return fmt.Errorf("failed to create macro data: %w", err)
	}
	return nil
}

// BatchCreate 批量创建宏观数据记录
func (m *macroDataDAO) BatchCreate(ctx context.Context, data []*model.MacroData) error {
	if len(data) == 0 {
		return nil
	}
	if err := m.db.WithContext(ctx).CreateInBatches(data, 1000).Error; err != nil {
		return fmt.Errorf("failed to batch create macro data: %w", err)
	}
	return nil
}

// GetByIndicator 根据指标代码获取宏观数据
func (m *macroDataDAO) GetByIndicator(ctx context.Context, indicatorCode string, startDate, endDate time.Time, limit, offset int) ([]*model.MacroData, error) {
	var data []*model.MacroData
	query := m.db.WithContext(ctx).Where("indicator_code = ?", indicatorCode)
	
	if !startDate.IsZero() {
		query = query.Where("data_date >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("data_date <= ?", endDate)
	}
	
	query = query.Order("data_date DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&data).Error; err != nil {
		return nil, fmt.Errorf("failed to get macro data by indicator: %w", err)
	}
	return data, nil
}

// GetLatest 获取最新宏观数据
func (m *macroDataDAO) GetLatest(ctx context.Context, indicatorCode string) (*model.MacroData, error) {
	var data model.MacroData
	query := m.db.WithContext(ctx).Where("indicator_code = ?", indicatorCode)
	query = query.Order("data_date DESC")
	
	if err := query.First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest macro data: %w", err)
	}
	return &data, nil
}

// GetByPeriodType 根据周期类型获取宏观数据
func (m *macroDataDAO) GetByPeriodType(ctx context.Context, periodType string, limit, offset int) ([]*model.MacroData, error) {
	var data []*model.MacroData
	query := m.db.WithContext(ctx).Where("period_type = ?", periodType)
	query = query.Order("data_date DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&data).Error; err != nil {
		return nil, fmt.Errorf("failed to get macro data by period type: %w", err)
	}
	return data, nil
}

// GetByDateRange 根据日期范围获取宏观数据
func (m *macroDataDAO) GetByDateRange(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]*model.MacroData, error) {
	var data []*model.MacroData
	query := m.db.WithContext(ctx)
	
	if !startDate.IsZero() {
		query = query.Where("data_date >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("data_date <= ?", endDate)
	}
	
	query = query.Order("data_date DESC, indicator_code ASC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&data).Error; err != nil {
		return nil, fmt.Errorf("failed to get macro data by date range: %w", err)
	}
	return data, nil
}

// Update 更新宏观数据
func (m *macroDataDAO) Update(ctx context.Context, data *model.MacroData) error {
	if err := m.db.WithContext(ctx).Save(data).Error; err != nil {
		return fmt.Errorf("failed to update macro data: %w", err)
	}
	return nil
}

// Delete 删除宏观数据记录
func (m *macroDataDAO) Delete(ctx context.Context, id uint64) error {
	if err := m.db.WithContext(ctx).Delete(&model.MacroData{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete macro data: %w", err)
	}
	return nil
}

// Count 获取宏观数据总数
func (m *macroDataDAO) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := m.db.WithContext(ctx).Model(&model.MacroData{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count macro data: %w", err)
	}
	return count, nil
}