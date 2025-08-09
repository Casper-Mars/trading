package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"data-collection-system/model"
)

// newsDataDAO 新闻数据访问层实现
type newsDataDAO struct {
	db *gorm.DB
}

// NewNewsDataDAO 创建新闻数据DAO实例
func NewNewsDataDAO(db *gorm.DB) NewsDataDAO {
	return &newsDataDAO{db: db}
}

// Create 创建新闻数据记录
func (n *newsDataDAO) Create(ctx context.Context, news *model.NewsData) error {
	if err := n.db.WithContext(ctx).Create(news).Error; err != nil {
		return fmt.Errorf("failed to create news data: %w", err)
	}
	return nil
}

// BatchCreate 批量创建新闻数据记录
func (n *newsDataDAO) BatchCreate(ctx context.Context, news []*model.NewsData) error {
	if len(news) == 0 {
		return nil
	}
	if err := n.db.WithContext(ctx).CreateInBatches(news, 1000).Error; err != nil {
		return fmt.Errorf("failed to batch create news data: %w", err)
	}
	return nil
}

// GetByTimeRange 根据时间范围获取新闻数据
func (n *newsDataDAO) GetByTimeRange(ctx context.Context, startTime, endTime time.Time, limit, offset int) ([]*model.NewsData, error) {
	var news []*model.NewsData
	query := n.db.WithContext(ctx)
	
	if !startTime.IsZero() {
		query = query.Where("publish_time >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("publish_time <= ?", endTime)
	}
	
	query = query.Order("publish_time DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&news).Error; err != nil {
		return nil, fmt.Errorf("failed to get news data by time range: %w", err)
	}
	return news, nil
}

// GetByCategory 根据分类获取新闻数据
func (n *newsDataDAO) GetByCategory(ctx context.Context, category string, limit, offset int) ([]*model.NewsData, error) {
	var news []*model.NewsData
	query := n.db.WithContext(ctx).Where("category = ?", category)
	query = query.Order("publish_time DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&news).Error; err != nil {
		return nil, fmt.Errorf("failed to get news data by category: %w", err)
	}
	return news, nil
}

// GetBySentiment 根据情感倾向获取新闻数据
func (n *newsDataDAO) GetBySentiment(ctx context.Context, sentiment int8, limit, offset int) ([]*model.NewsData, error) {
	var news []*model.NewsData
	query := n.db.WithContext(ctx).Where("sentiment = ?", sentiment)
	query = query.Order("publish_time DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&news).Error; err != nil {
		return nil, fmt.Errorf("failed to get news data by sentiment: %w", err)
	}
	return news, nil
}

// GetByImportance 根据重要程度获取新闻数据
func (n *newsDataDAO) GetByImportance(ctx context.Context, minLevel int8, limit, offset int) ([]*model.NewsData, error) {
	var news []*model.NewsData
	query := n.db.WithContext(ctx).Where("importance_level >= ?", minLevel)
	query = query.Order("importance_level DESC, publish_time DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&news).Error; err != nil {
		return nil, fmt.Errorf("failed to get news data by importance: %w", err)
	}
	return news, nil
}

// GetByRelatedStock 根据相关股票获取新闻数据
func (n *newsDataDAO) GetByRelatedStock(ctx context.Context, symbol string, limit, offset int) ([]*model.NewsData, error) {
	var news []*model.NewsData
	query := n.db.WithContext(ctx).Where("JSON_CONTAINS(related_stocks, ?)", fmt.Sprintf(`"%s"`, symbol))
	query = query.Order("publish_time DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&news).Error; err != nil {
		return nil, fmt.Errorf("failed to get news data by related stock: %w", err)
	}
	return news, nil
}

// SearchByKeyword 根据关键词搜索新闻数据
func (n *newsDataDAO) SearchByKeyword(ctx context.Context, keyword string, limit, offset int) ([]*model.NewsData, error) {
	var news []*model.NewsData
	query := n.db.WithContext(ctx).Where("title LIKE ? OR content LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	query = query.Order("publish_time DESC")
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&news).Error; err != nil {
		return nil, fmt.Errorf("failed to search news data by keyword: %w", err)
	}
	return news, nil
}

// Update 更新新闻数据
func (n *newsDataDAO) Update(ctx context.Context, news *model.NewsData) error {
	if err := n.db.WithContext(ctx).Save(news).Error; err != nil {
		return fmt.Errorf("failed to update news data: %w", err)
	}
	return nil
}

// Delete 删除新闻数据记录
func (n *newsDataDAO) Delete(ctx context.Context, id uint64) error {
	if err := n.db.WithContext(ctx).Delete(&model.NewsData{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete news data: %w", err)
	}
	return nil
}

// Count 获取新闻数据总数
func (n *newsDataDAO) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := n.db.WithContext(ctx).Model(&model.NewsData{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count news data: %w", err)
	}
	return count, nil
}