package dao

import (
	"context"
	"time"

	"data-collection-system/model"
	"data-collection-system/pkg/errors"

	"gorm.io/gorm"
)

// macroDataRepository 宏观数据仓库实现
type macroDataRepository struct {
	db *gorm.DB
}

// NewMacroDataRepository 创建宏观数据仓库
func NewMacroDataRepository(db *gorm.DB) MacroDataRepository {
	return &macroDataRepository{
		db: db,
	}
}

// Create 创建宏观数据记录
func (r *macroDataRepository) Create(ctx context.Context, data *model.MacroData) error {
	if err := r.db.WithContext(ctx).Create(data).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to create macro data")
	}
	return nil
}

// BatchCreate 批量创建宏观数据记录
func (r *macroDataRepository) BatchCreate(ctx context.Context, dataList []*model.MacroData) error {
	if len(dataList) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).CreateInBatches(dataList, 100).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to batch create macro data")
	}
	return nil
}

// GetByID 根据ID获取宏观数据
func (r *macroDataRepository) GetByID(ctx context.Context, id uint64) (*model.MacroData, error) {
	var data model.MacroData
	if err := r.db.WithContext(ctx).First(&data, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeDataNotFound, "macro data not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get macro data by id")
	}
	return &data, nil
}

// GetByIndicatorAndDate 根据指标代码和数据日期获取宏观数据
func (r *macroDataRepository) GetByIndicatorAndDate(ctx context.Context, indicatorCode string, dataDate time.Time) (*model.MacroData, error) {
	var data model.MacroData
	if err := r.db.WithContext(ctx).Where("indicator_code = ? AND data_date = ?", indicatorCode, dataDate).First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeDataNotFound, "macro data not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get macro data by indicator and date")
	}
	return &data, nil
}

// GetByIndicator 根据指标代码获取宏观数据列表
func (r *macroDataRepository) GetByIndicator(ctx context.Context, indicatorCode string, limit int) ([]*model.MacroData, error) {
	var dataList []*model.MacroData
	query := r.db.WithContext(ctx).Where("indicator_code = ?", indicatorCode).Order("data_date DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&dataList).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get macro data by indicator")
	}
	return dataList, nil
}

// GetByPeriodType 根据周期类型获取宏观数据列表
func (r *macroDataRepository) GetByPeriodType(ctx context.Context, periodType string, limit int) ([]*model.MacroData, error) {
	var dataList []*model.MacroData
	query := r.db.WithContext(ctx).Where("period_type = ?", periodType).Order("data_date DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&dataList).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get macro data by period type")
	}
	return dataList, nil
}

// GetByTimeRange 根据时间范围获取宏观数据
func (r *macroDataRepository) GetByTimeRange(ctx context.Context, indicatorCode string, startDate, endDate time.Time) ([]*model.MacroData, error) {
	var dataList []*model.MacroData
	query := r.db.WithContext(ctx).Where("indicator_code = ? AND data_date >= ? AND data_date <= ?", indicatorCode, startDate, endDate)
	if err := query.Order("data_date ASC").Find(&dataList).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get macro data by time range")
	}
	return dataList, nil
}

// Update 更新宏观数据
func (r *macroDataRepository) Update(ctx context.Context, data *model.MacroData) error {
	if err := r.db.WithContext(ctx).Save(data).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to update macro data")
	}
	return nil
}

// Delete 删除宏观数据
func (r *macroDataRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&model.MacroData{}, id).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to delete macro data")
	}
	return nil
}

// Count 获取宏观数据总数
func (r *macroDataRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.MacroData{}).Count(&count).Error; err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeDatabase, "failed to count macro data")
	}
	return count, nil
}