package task

import (
	"context"
	"fmt"
	"time"

	"data-collection-system/model"
	dao "data-collection-system/repo/mysql"
	"data-collection-system/service/collection"

	"github.com/go-redis/redis/v8"
)

// TaskService 任务服务
type TaskService struct {
	taskRepo       dao.DataTaskRepository
	collectionSvc  *collection.Service
	rdb           *redis.Client
}

// NewTaskService 创建任务服务
func NewTaskService(taskRepo dao.DataTaskRepository, collectionSvc *collection.Service, rdb *redis.Client) *TaskService {
	return &TaskService{
		taskRepo:      taskRepo,
		collectionSvc: collectionSvc,
		rdb:          rdb,
	}
}

// TaskQueryParams 任务查询参数
type TaskQueryParams struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   *int8  `form:"status"`
	Type     string `form:"type"`
}

// TaskResult 任务查询结果
type TaskResult struct {
	Data  []*model.DataTask `json:"data"`
	Total int64            `json:"total"`
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Type        string                 `json:"type" binding:"required"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Schedule    string                 `json:"schedule" binding:"required"`
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	Name        *string                `json:"name"`
	Type        *string                `json:"type"`
	Description *string                `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Schedule    *string                `json:"schedule"`
	Status      *int8                  `json:"status"`
}

// GetTasks 获取任务列表
func (s *TaskService) GetTasksWithParams(ctx context.Context, params *TaskQueryParams) (*TaskResult, error) {
	// 设置默认分页参数
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 20
	}

	// 查询任务列表
	tasks, total, err := s.taskRepo.List(ctx, params.Page, params.PageSize)
	if err != nil {
		return nil, fmt.Errorf("查询任务列表失败: %w", err)
	}

	// 如果有状态或类型过滤条件，需要在应用层过滤
	if params.Status != nil || params.Type != "" {
		filteredTasks := make([]*model.DataTask, 0)
		for _, task := range tasks {
			if params.Status != nil && task.Status != *params.Status {
				continue
			}
			if params.Type != "" && task.TaskType != params.Type {
				continue
			}
			filteredTasks = append(filteredTasks, task)
		}
		tasks = filteredTasks
		total = int64(len(filteredTasks))
	}

	return &TaskResult{
		Data:  tasks,
		Total: total,
	}, nil
}

// GetTaskByID 根据ID获取任务
func (s *TaskService) GetTaskByID(ctx context.Context, id uint64) (*model.DataTask, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %w", err)
	}
	return task, nil
}

// CreateTask 创建任务
func (s *TaskService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*model.DataTask, error) {
	// 验证任务类型
	if !model.IsValidTaskType(req.Type) {
		return nil, fmt.Errorf("无效的任务类型: %s", req.Type)
	}

	// 验证任务配置
	if s.collectionSvc != nil {
		if err := s.collectionSvc.ValidateCollectionTask(req.Type, req.Config); err != nil {
			return nil, fmt.Errorf("任务配置验证失败: %w", err)
		}
	}

	// 创建任务对象
	task := &model.DataTask{
		TaskName:    req.Name,
		TaskType:    req.Type,
		Description: req.Description,
		CronExpr:    req.Schedule,
		Status:      model.TaskStatusEnabled,
		Config:      model.TaskConfig(req.Config),
	}

	// 保存到数据库
	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("创建任务失败: %w", err)
	}

	return task, nil
}

// UpdateTask 更新任务
func (s *TaskService) UpdateTask(ctx context.Context, id uint64, req *UpdateTaskRequest) (*model.DataTask, error) {
	// 获取现有任务
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %w", err)
	}

	// 更新字段
	if req.Name != nil {
		task.TaskName = *req.Name
	}
	if req.Type != nil {
		if !model.IsValidTaskType(*req.Type) {
			return nil, fmt.Errorf("无效的任务类型: %s", *req.Type)
		}
		task.TaskType = *req.Type
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Schedule != nil {
		task.CronExpr = *req.Schedule
	}
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.Config != nil {
		// 验证任务配置
		if s.collectionSvc != nil {
			if err := s.collectionSvc.ValidateCollectionTask(task.TaskType, req.Config); err != nil {
				return nil, fmt.Errorf("任务配置验证失败: %w", err)
			}
		}
		task.Config = model.TaskConfig(req.Config)
	}

	// 保存更新
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("更新任务失败: %w", err)
	}

	return task, nil
}

// DeleteTask 删除任务
func (s *TaskService) DeleteTask(ctx context.Context, id uint64) error {
	if err := s.taskRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("删除任务失败: %w", err)
	}
	return nil
}

// RunTask 运行任务
func (s *TaskService) RunTask(ctx context.Context, id uint64) error {
	// 获取任务
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 检查任务状态
	if task.IsRunning() {
		return fmt.Errorf("任务正在运行中")
	}

	// 设置任务为运行状态
	task.SetRunning()
	task.IncrementRunCount()
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 执行任务
	var taskErr error
	if s.collectionSvc != nil {
		taskErr = s.collectionSvc.ExecuteCollectionTask(ctx, task.TaskType, task.Config)
	} else {
		taskErr = fmt.Errorf("采集服务未初始化")
	}

	// 更新任务状态和统计
	if taskErr != nil {
		task.IncrementFailureCount()
		task.Enable() // 恢复为启用状态
	} else {
		task.IncrementSuccessCount()
		task.Enable() // 恢复为启用状态
	}

	// 保存最终状态
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return fmt.Errorf("更新任务最终状态失败: %w", err)
	}

	return taskErr
}

// GetTaskStatus 获取任务状态
func (s *TaskService) GetTaskStatus(ctx context.Context, id uint64) (map[string]interface{}, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %w", err)
	}

	status := map[string]interface{}{
		"id":            task.ID,
		"name":          task.TaskName,
		"type":          task.TaskType,
		"status":        task.Status,
		"last_run_at":   task.LastRunAt,
		"next_run_at":   task.NextRunAt,
		"run_count":     task.RunCount,
		"success_count": task.SuccessCount,
		"failure_count": task.FailureCount,
		"success_rate":  task.GetSuccessRate(),
		"failure_rate":  task.GetFailureRate(),
	}

	return status, nil
}

// GetSystemStats 获取系统统计信息
func (s *TaskService) GetSystemStats(ctx context.Context) (map[string]interface{}, error) {
	// 获取任务统计
	totalTasks, err := s.taskRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取任务总数失败: %w", err)
	}

	// 获取所有任务并统计状态
	allTasks, _, err := s.taskRepo.List(ctx, 1, 1000) // 获取足够多的任务进行统计
	if err != nil {
		return nil, fmt.Errorf("获取任务列表失败: %w", err)
	}

	var enabledTasks, runningTasks int64
	for _, task := range allTasks {
		if task.Status == model.TaskStatusEnabled {
			enabledTasks++
		}
		if task.Status == model.TaskStatusRunning {
			runningTasks++
		}
	}

	stats := map[string]interface{}{
		"total_tasks":   totalTasks,
		"enabled_tasks": enabledTasks,
		"running_tasks": runningTasks,
		"task_types":    model.ValidTaskTypes,
		"timestamp":     time.Now(),
	}

	return stats, nil
}

// GetMetrics 获取系统指标
func (s *TaskService) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	// 获取所有任务进行统计
	allTasks, _, err := s.taskRepo.List(ctx, 1, 1000) // 获取足够多的任务进行统计
	if err != nil {
		return nil, fmt.Errorf("获取任务列表失败: %w", err)
	}

	// 获取各类型任务统计
	typeStats := make(map[string]int64)
	for _, taskType := range model.ValidTaskTypes {
		typeStats[taskType] = 0
	}
	for _, task := range allTasks {
		if _, exists := typeStats[task.TaskType]; exists {
			typeStats[task.TaskType]++
		}
	}

	// 获取状态统计
	statusStats := make(map[string]int64)
	statuses := []int8{model.TaskStatusDisabled, model.TaskStatusEnabled, model.TaskStatusRunning, model.TaskStatusPaused}
	for _, status := range statuses {
		statusStats[fmt.Sprintf("status_%d", status)] = 0
	}
	for _, task := range allTasks {
		statusStats[fmt.Sprintf("status_%d", task.Status)]++
	}

	metrics := map[string]interface{}{
		"task_type_stats": typeStats,
		"status_stats":    statusStats,
		"timestamp":       time.Now(),
	}

	return metrics, nil
}