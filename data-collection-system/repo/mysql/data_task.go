package dao

import (
	"context"
	"time"

	"data-collection-system/model"
	"data-collection-system/pkg/errors"

	"gorm.io/gorm"
)

// dataTaskRepository 数据任务仓库实现
type dataTaskRepository struct {
	db *gorm.DB
}

// NewDataTaskRepository 创建数据任务仓库
func NewDataTaskRepository(db *gorm.DB) DataTaskRepository {
	return &dataTaskRepository{
		db: db,
	}
}

// Create 创建数据任务记录
func (r *dataTaskRepository) Create(ctx context.Context, task *model.DataTask) error {
	if err := r.db.WithContext(ctx).Create(task).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to create data task")
	}
	return nil
}

// GetByID 根据ID获取数据任务
func (r *dataTaskRepository) GetByID(ctx context.Context, id uint64) (*model.DataTask, error) {
	var task model.DataTask
	if err := r.db.WithContext(ctx).First(&task, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeDataNotFound, "data task not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get data task by id")
	}
	return &task, nil
}

// GetByName 根据任务名称获取数据任务
func (r *dataTaskRepository) GetByName(ctx context.Context, taskName string) (*model.DataTask, error) {
	var task model.DataTask
	if err := r.db.WithContext(ctx).Where("task_name = ?", taskName).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeDataNotFound, "data task not found")
		}
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get data task by name")
	}
	return &task, nil
}

// List 分页获取数据任务列表
func (r *dataTaskRepository) List(ctx context.Context, page, pageSize int) ([]*model.DataTask, int64, error) {
	var tasks []*model.DataTask
	var total int64

	// 计算总数
	if err := r.db.WithContext(ctx).Model(&model.DataTask{}).Count(&total).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeDatabase, "failed to count data tasks")
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, 0, errors.Wrap(err, errors.ErrCodeDatabase, "failed to list data tasks")
	}

	return tasks, total, nil
}

// GetByType 根据任务类型获取数据任务列表
func (r *dataTaskRepository) GetByType(ctx context.Context, taskType string) ([]*model.DataTask, error) {
	var tasks []*model.DataTask
	if err := r.db.WithContext(ctx).Where("task_type = ?", taskType).Find(&tasks).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get data tasks by type")
	}
	return tasks, nil
}

// GetByStatus 根据任务状态获取数据任务列表
func (r *dataTaskRepository) GetByStatus(ctx context.Context, status int8) ([]*model.DataTask, error) {
	var tasks []*model.DataTask
	if err := r.db.WithContext(ctx).Where("status = ?", status).Find(&tasks).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get data tasks by status")
	}
	return tasks, nil
}

// GetPendingTasks 获取待执行的任务列表
func (r *dataTaskRepository) GetPendingTasks(ctx context.Context) ([]*model.DataTask, error) {
	var tasks []*model.DataTask
	now := time.Now()
	query := r.db.WithContext(ctx).Where("status = ? AND (next_run_at IS NULL OR next_run_at <= ?)", model.TaskStatusEnabled, now)
	if err := query.Order("next_run_at ASC").Find(&tasks).Error; err != nil {
		return nil, errors.Wrap(err, errors.ErrCodeDatabase, "failed to get pending data tasks")
	}
	return tasks, nil
}

// Update 更新数据任务
func (r *dataTaskRepository) Update(ctx context.Context, task *model.DataTask) error {
	if err := r.db.WithContext(ctx).Save(task).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to update data task")
	}
	return nil
}

// Delete 删除数据任务
func (r *dataTaskRepository) Delete(ctx context.Context, id uint64) error {
	if err := r.db.WithContext(ctx).Delete(&model.DataTask{}, id).Error; err != nil {
		return errors.Wrap(err, errors.ErrCodeDatabase, "failed to delete data task")
	}
	return nil
}

// Count 获取数据任务总数
func (r *dataTaskRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.DataTask{}).Count(&count).Error; err != nil {
		return 0, errors.Wrap(err, errors.ErrCodeDatabase, "failed to count data tasks")
	}
	return count, nil
}