package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"data-collection-system/internal/models"
)

// dataTaskDAO 任务数据访问层实现
type dataTaskDAO struct {
	db *gorm.DB
}

// NewDataTaskDAO 创建任务DAO实例
func NewDataTaskDAO(db *gorm.DB) DataTaskDAO {
	return &dataTaskDAO{db: db}
}

// Create 创建任务记录
func (d *dataTaskDAO) Create(ctx context.Context, task *models.DataTask) error {
	if err := d.db.WithContext(ctx).Create(task).Error; err != nil {
		return fmt.Errorf("failed to create data task: %w", err)
	}
	return nil
}

// GetByID 根据ID获取任务
func (d *dataTaskDAO) GetByID(ctx context.Context, id uint64) (*models.DataTask, error) {
	var task models.DataTask
	if err := d.db.WithContext(ctx).First(&task, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get data task by id: %w", err)
	}
	return &task, nil
}

// GetByName 根据任务名称获取任务
func (d *dataTaskDAO) GetByName(ctx context.Context, taskName string) (*models.DataTask, error) {
	var task models.DataTask
	if err := d.db.WithContext(ctx).Where("task_name = ?", taskName).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get data task by name: %w", err)
	}
	return &task, nil
}

// GetByType 根据任务类型获取任务列表
func (d *dataTaskDAO) GetByType(ctx context.Context, taskType string, limit, offset int) ([]*models.DataTask, error) {
	var tasks []*models.DataTask
	query := d.db.WithContext(ctx).Where("task_type = ?", taskType)
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get data tasks by type: %w", err)
	}
	return tasks, nil
}

// GetByStatus 根据状态获取任务列表
func (d *dataTaskDAO) GetByStatus(ctx context.Context, status int8, limit, offset int) ([]*models.DataTask, error) {
	var tasks []*models.DataTask
	query := d.db.WithContext(ctx).Where("status = ?", status)
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get data tasks by status: %w", err)
	}
	return tasks, nil
}

// GetEnabledTasks 获取启用的任务列表
func (d *dataTaskDAO) GetEnabledTasks(ctx context.Context) ([]*models.DataTask, error) {
	var tasks []*models.DataTask
	if err := d.db.WithContext(ctx).Where("status = ?", models.TaskStatusEnabled).Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get enabled data tasks: %w", err)
	}
	return tasks, nil
}

// GetTasksToRun 获取需要运行的任务列表
func (d *dataTaskDAO) GetTasksToRun(ctx context.Context) ([]*models.DataTask, error) {
	var tasks []*models.DataTask
	now := time.Now()
	if err := d.db.WithContext(ctx).
		Where("status = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", models.TaskStatusEnabled, now).
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get tasks to run: %w", err)
	}
	return tasks, nil
}

// GetAll 获取所有任务
func (d *dataTaskDAO) GetAll(ctx context.Context, limit, offset int) ([]*models.DataTask, error) {
	var tasks []*models.DataTask
	query := d.db.WithContext(ctx)
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	
	if err := query.Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get all data tasks: %w", err)
	}
	return tasks, nil
}

// Update 更新任务
func (d *dataTaskDAO) Update(ctx context.Context, task *models.DataTask) error {
	if err := d.db.WithContext(ctx).Save(task).Error; err != nil {
		return fmt.Errorf("failed to update data task: %w", err)
	}
	return nil
}

// UpdateStatus 更新任务状态
func (d *dataTaskDAO) UpdateStatus(ctx context.Context, id uint64, status int8) error {
	if err := d.db.WithContext(ctx).Model(&models.DataTask{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return fmt.Errorf("failed to update data task status: %w", err)
	}
	return nil
}

// UpdateRunStats 更新任务运行统计信息
func (d *dataTaskDAO) UpdateRunStats(ctx context.Context, id uint64, lastRunAt, nextRunAt *time.Time, runCount, successCount, failureCount uint64) error {
	updates := map[string]interface{}{
		"run_count":     runCount,
		"success_count": successCount,
		"failure_count": failureCount,
	}
	
	if lastRunAt != nil {
		updates["last_run_at"] = lastRunAt
	}
	if nextRunAt != nil {
		updates["next_run_at"] = nextRunAt
	}
	
	if err := d.db.WithContext(ctx).Model(&models.DataTask{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update data task run stats: %w", err)
	}
	return nil
}

// Delete 删除任务
func (d *dataTaskDAO) Delete(ctx context.Context, id uint64) error {
	if err := d.db.WithContext(ctx).Delete(&models.DataTask{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete data task: %w", err)
	}
	return nil
}

// Count 获取任务总数
func (d *dataTaskDAO) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := d.db.WithContext(ctx).Model(&models.DataTask{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count data tasks: %w", err)
	}
	return count, nil
}