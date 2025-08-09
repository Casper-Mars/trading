package dao

import (
	"context"
	"fmt"
	"time"

	"data-collection-system/model"
	"data-collection-system/pkg/errors"

	"gorm.io/gorm"
)

// newsRepository 新闻数据仓库实现
type newsRepository struct {
	db *gorm.DB
}

// NewNewsRepository 创建新闻数据仓库
func NewNewsRepository(db *gorm.DB) NewsRepository {
	return &newsRepository{
		db: db,
	}
}

// Create 创建新闻
func (r *newsRepository) Create(ctx context.Context, news *model.NewsData) error {
	if err := r.db.WithContext(ctx).Create(news).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "创建新闻失败")
	}
	return nil
}

// BatchCreate 批量创建新闻
func (r *newsRepository) BatchCreate(ctx context.Context, newsList []*model.NewsData) error {
	if len(newsList) == 0 {
		return nil
	}

	// 使用事务批量插入
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 分批插入，避免单次插入过多数据
	batchSize := 100
	for i := 0; i < len(newsList); i += batchSize {
		end := i + batchSize
		if end > len(newsList) {
			end = len(newsList)
		}

		batch := newsList[i:end]
		if err := tx.Create(&batch).Error; err != nil {
			tx.Rollback()
			return errors.Wrap(err, errors.ErrCodeDatabase, "批量创建新闻失败")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "提交事务失败")
	}

	return nil
}

// GetByID 根据ID获取新闻
func (r *newsRepository) GetByID(ctx context.Context, id uint) (*model.NewsData, error) {
	var news model.NewsData
	err := r.db.WithContext(ctx).First(&news, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeNotFound, "新闻不存在")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询新闻失败")
	}
	return &news, nil
}

// GetByURL 根据URL获取新闻
func (r *newsRepository) GetByURL(ctx context.Context, url string) (*model.NewsData, error) {
	var news model.NewsData
	err := r.db.WithContext(ctx).Where("url = ?", url).First(&news).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeNotFound, "新闻不存在")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询新闻失败")
	}
	return &news, nil
}

// Exists 检查新闻是否存在
func (r *newsRepository) Exists(ctx context.Context, url string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.NewsData{}).Where("url = ?", url).Count(&count).Error
	if err != nil {
		return false, errors.Wrap(err, errors.ErrCodeDatabase, "检查新闻存在性失败")
	}
	return count > 0, nil
}

// List 分页查询新闻
func (r *newsRepository) List(ctx context.Context, params *NewsQueryParams) ([]*model.NewsData, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.NewsData{})

	// 构建查询条件
	query = r.buildQueryConditions(query, params)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeDatabase, "查询新闻总数失败")
	}

	// 分页和排序
	if params.OrderBy == "" {
		params.OrderBy = "published_at"
	}
	if params.OrderDir == "" {
		params.OrderDir = "desc"
	}

	orderClause := fmt.Sprintf("%s %s", params.OrderBy, params.OrderDir)
	query = query.Order(orderClause)

	if params.PageSize > 0 {
		offset := (params.Page - 1) * params.PageSize
		query = query.Offset(offset).Limit(params.PageSize)
	}

	// 执行查询
	var newsList []*model.NewsData
	if err := query.Find(&newsList).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeDatabase, "查询新闻列表失败")
	}

	return newsList, total, nil
}

// buildQueryConditions 构建查询条件
func (r *newsRepository) buildQueryConditions(query *gorm.DB, params *NewsQueryParams) *gorm.DB {
	if params.Category != "" {
		query = query.Where("category = ?", params.Category)
	}

	if params.Source != "" {
		query = query.Where("source = ?", params.Source)
	}

	if !params.StartTime.IsZero() {
		query = query.Where("published_at >= ?", params.StartTime)
	}

	if !params.EndTime.IsZero() {
		query = query.Where("published_at <= ?", params.EndTime)
	}

	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ? OR summary LIKE ?", keyword, keyword, keyword)
	}

	if params.Status > 0 {
		query = query.Where("status = ?", params.Status)
	}

	return query
}

// Update 更新新闻
func (r *newsRepository) Update(ctx context.Context, news *model.NewsData) error {
	if err := r.db.WithContext(ctx).Save(news).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "更新新闻失败")
	}
	return nil
}

// Delete 删除新闻
func (r *newsRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.NewsData{}, id)
	if result.Error != nil {
		return errors.Wrap(result.Error, errors.ErrCodeDatabase, "删除新闻失败")
	}
	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "新闻不存在")
	}
	return nil
}

// Search 根据关键词搜索新闻
func (r *newsRepository) Search(ctx context.Context, params *NewsSearchParams) ([]*model.NewsData, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.NewsData{})

	// 构建搜索条件
	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ? OR summary LIKE ?", keyword, keyword, keyword)
	}

	if params.Category != "" {
		query = query.Where("category = ?", params.Category)
	}

	if params.Source != "" {
		query = query.Where("source = ?", params.Source)
	}

	if !params.StartTime.IsZero() {
		query = query.Where("published_at >= ?", params.StartTime)
	}

	if !params.EndTime.IsZero() {
		query = query.Where("published_at <= ?", params.EndTime)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeDatabase, "搜索新闻总数失败")
	}

	// 分页和排序
	query = query.Order("published_at DESC")
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	// 执行查询
	var newsList []*model.NewsData
	if err := query.Find(&newsList).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeDatabase, "搜索新闻失败")
	}

	return newsList, total, nil
}

// GetHotNews 获取热门新闻
func (r *newsRepository) GetHotNews(ctx context.Context, limit int, hours int) ([]*model.NewsData, error) {
	if limit <= 0 {
		limit = 10
	}
	if hours <= 0 {
		hours = 24
	}

	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	var newsList []*model.NewsData
	err := r.db.WithContext(ctx).
		Where("published_at >= ?", startTime).
		Order("view_count DESC, published_at DESC").
		Limit(limit).
		Find(&newsList).Error

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询热门新闻失败")
	}

	return newsList, nil
}

// GetLatestNews 获取最新新闻
func (r *newsRepository) GetLatestNews(ctx context.Context, limit int) ([]*model.NewsData, error) {
	if limit <= 0 {
		limit = 10
	}

	var newsList []*model.NewsData
	err := r.db.WithContext(ctx).
		Order("published_at DESC").
		Limit(limit).
		Find(&newsList).Error

	if err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询最新新闻失败")
	}

	return newsList, nil
}

// GetByCategory 根据分类获取新闻
func (r *newsRepository) GetByCategory(ctx context.Context, category string, limit int, offset int) ([]*model.NewsData, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.NewsData{}).Where("category = ?", category)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeDatabase, "查询分类新闻总数失败")
	}

	// 分页查询
	var newsList []*model.NewsData
	err := query.Order("published_at DESC").Limit(limit).Offset(offset).Find(&newsList).Error
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeDatabase, "查询分类新闻失败")
	}

	return newsList, total, nil
}

// GetBySource 根据来源获取新闻
func (r *newsRepository) GetBySource(ctx context.Context, source string, limit int, offset int) ([]*model.NewsData, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.NewsData{}).Where("source = ?", source)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeDatabase, "查询来源新闻总数失败")
	}

	// 分页查询
	var newsList []*model.NewsData
	err := query.Order("published_at DESC").Limit(limit).Offset(offset).Find(&newsList).Error
	if err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeDatabase, "查询来源新闻失败")
	}

	return newsList, total, nil
}

// GetStats 获取新闻统计信息
func (r *newsRepository) GetStats(ctx context.Context) (*NewsStats, error) {
	stats := &NewsStats{
		CategoryStats: make(map[string]int64),
		SourceStats:   make(map[string]int64),
	}

	// 总数统计
	if err := r.db.WithContext(ctx).Model(&model.NewsData{}).Count(&stats.TotalCount).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询新闻总数失败")
	}

	// 今日统计
	today := time.Now().Truncate(24 * time.Hour)
	if err := r.db.WithContext(ctx).Model(&model.NewsData{}).
		Where("created_at >= ?", today).
		Count(&stats.TodayCount).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询今日新闻数失败")
	}

	// 本周统计
	weekStart := time.Now().AddDate(0, 0, -int(time.Now().Weekday()))
	weekStart = weekStart.Truncate(24 * time.Hour)
	if err := r.db.WithContext(ctx).Model(&model.NewsData{}).
		Where("created_at >= ?", weekStart).
		Count(&stats.WeekCount).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询本周新闻数失败")
	}

	// 本月统计
	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Now().Location())
	if err := r.db.WithContext(ctx).Model(&model.NewsData{}).
		Where("created_at >= ?", monthStart).
		Count(&stats.MonthCount).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询本月新闻数失败")
	}

	// 分类统计
	var categoryResults []struct {
		Category string `json:"category"`
		Count    int64  `json:"count"`
	}
	if err := r.db.WithContext(ctx).Model(&model.NewsData{}).
		Select("category, COUNT(*) as count").
		Group("category").
		Scan(&categoryResults).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询分类统计失败")
	}
	for _, result := range categoryResults {
		stats.CategoryStats[result.Category] = result.Count
	}

	// 来源统计
	var sourceResults []struct {
		Source string `json:"source"`
		Count  int64  `json:"count"`
	}
	if err := r.db.WithContext(ctx).Model(&model.NewsData{}).
		Select("source, COUNT(*) as count").
		Group("source").
		Scan(&sourceResults).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "查询来源统计失败")
	}
	for _, result := range sourceResults {
		stats.SourceStats[result.Source] = result.Count
	}

	// 最后更新时间
	var lastNews model.NewsData
	if err := r.db.WithContext(ctx).Order("created_at DESC").First(&lastNews).Error; err == nil {
		stats.LastUpdateTime = lastNews.CreatedAt
	}

	return stats, nil
}

// CleanExpired 清理过期新闻
func (r *newsRepository) CleanExpired(ctx context.Context, days int) (int64, error) {
	if days <= 0 {
		days = 30 // 默认清理30天前的新闻
	}

	expiredTime := time.Now().AddDate(0, 0, -days)
	result := r.db.WithContext(ctx).
		Where("created_at < ?", expiredTime).
		Delete(&model.NewsData{})

	if result.Error != nil {
		return 0, errors.Wrap(result.Error, errors.ErrCodeDatabase, "清理过期新闻失败")
	}

	return result.RowsAffected, nil
}
