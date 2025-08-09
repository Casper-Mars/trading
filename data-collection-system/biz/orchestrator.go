package biz

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"data-collection-system/service/collection"
	"data-collection-system/service/processing"

	"github.com/go-redis/redis/v8"
)

// Orchestrator 业务编排器 - 统一管理所有业务编排服务
// 负责协调各种业务编排服务，提供统一的业务编排入口
type Orchestrator struct {
	taskExecutor *TaskExecutor
	dataPipeline *DataPipeline
	mu           sync.RWMutex
	initialized  bool
}

// NewOrchestrator 创建业务编排器
func NewOrchestrator(
	collectionSvc *collection.Service,
	processingSvc *processing.ProcessingService,
	redisClient *redis.Client,
) *Orchestrator {
	// 创建任务执行编排服务
	taskExecutor := NewTaskExecutor(collectionSvc, processingSvc)

	// 创建数据管道业务编排服务
	dataPipeline := NewDataPipeline(collectionSvc, processingSvc, redisClient)

	return &Orchestrator{
		taskExecutor: taskExecutor,
		dataPipeline: dataPipeline,
		initialized:  true,
	}
}

// ExecuteBusinessFlow 执行业务流程
// 统一的业务流程执行入口，根据流程类型调用相应的编排服务
func (o *Orchestrator) ExecuteBusinessFlow(ctx context.Context, flowType string, config map[string]interface{}) error {
	o.mu.RLock()
	if !o.initialized {
		o.mu.RUnlock()
		return fmt.Errorf("业务编排器未初始化")
	}
	o.mu.RUnlock()

	log.Printf("开始执行业务流程，类型: %s", flowType)
	startTime := time.Now()

	var err error
	switch flowType {
	// 任务执行类流程
	case "data_collection_processing":
		taskType, ok := config["task_type"].(string)
		if !ok {
			return fmt.Errorf("缺少任务类型参数")
		}
		err = o.taskExecutor.ExecuteDataCollectionAndProcessing(ctx, taskType, config)

	case "batch_data_collection":
		symbols, ok := config["symbols"].([]string)
		if !ok {
			return fmt.Errorf("缺少股票代码列表参数")
		}
		startDate, _ := config["start_date"].(string)
		endDate, _ := config["end_date"].(string)
		err = o.taskExecutor.ExecuteBatchDataCollection(ctx, symbols, startDate, endDate)

	case "comprehensive_data_update":
		err = o.taskExecutor.ExecuteComprehensiveDataUpdate(ctx)

	// 数据管道类流程
	case "daily_full_pipeline":
		err = o.dataPipeline.ExecuteFullDataPipeline(ctx, "daily_full_pipeline", config)

	case "weekly_full_pipeline":
		err = o.dataPipeline.ExecuteFullDataPipeline(ctx, "weekly_full_pipeline", config)

	case "news_pipeline":
		err = o.dataPipeline.ExecuteFullDataPipeline(ctx, "news_pipeline", config)

	case "financial_pipeline":
		err = o.dataPipeline.ExecuteFullDataPipeline(ctx, "financial_pipeline", config)

	// 自定义流程
	case "custom_pipeline":
		steps, ok := config["steps"].([]PipelineStep)
		if !ok {
			return fmt.Errorf("缺少自定义流程步骤参数")
		}
		err = o.dataPipeline.ExecuteCustomPipeline(ctx, steps)

	default:
		err = fmt.Errorf("不支持的业务流程类型: %s", flowType)
	}

	elapsed := time.Since(startTime)
	if err != nil {
		log.Printf("业务流程执行失败，类型: %s, 耗时: %v, 错误: %v", flowType, elapsed, err)
	} else {
		log.Printf("业务流程执行成功，类型: %s, 耗时: %v", flowType, elapsed)
	}

	return err
}

// GetSystemStatus 获取系统状态
// 返回所有业务编排服务的状态信息
func (o *Orchestrator) GetSystemStatus(ctx context.Context) (map[string]interface{}, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if !o.initialized {
		return nil, fmt.Errorf("业务编排器未初始化")
	}

	status := make(map[string]interface{})

	// 获取数据采集状态
	collectionStatus, err := o.taskExecutor.GetCollectionStatus(ctx)
	if err != nil {
		log.Printf("获取数据采集状态失败: %v", err)
		collectionStatus = map[string]interface{}{"error": err.Error()}
	}
	status["collection_status"] = collectionStatus

	// 获取数据管道状态
	pipelineStats := o.dataPipeline.GetPipelineStats()
	status["pipeline_stats"] = pipelineStats
	status["pipeline_running"] = o.dataPipeline.IsRunning()

	// 系统整体状态
	status["system_initialized"] = o.initialized
	status["current_time"] = time.Now()

	return status, nil
}

// ValidateSystemHealth 验证系统健康状态
// 执行系统健康检查，确保各个组件正常工作
func (o *Orchestrator) ValidateSystemHealth(ctx context.Context) error {
	log.Println("开始系统健康检查")

	// 检查业务编排器状态
	if !o.initialized {
		return fmt.Errorf("业务编排器未初始化")
	}

	// 检查数据采集服务状态
	collectionStatus, err := o.taskExecutor.GetCollectionStatus(ctx)
	if err != nil {
		return fmt.Errorf("数据采集服务健康检查失败: %w", err)
	}
	log.Printf("数据采集服务状态: %+v", collectionStatus)

	// 检查数据管道状态
	if o.dataPipeline.IsRunning() {
		log.Println("数据管道正在运行中")
	} else {
		log.Println("数据管道处于空闲状态")
	}

	// 验证数据完整性
	if err := o.taskExecutor.ValidateDataIntegrity(ctx); err != nil {
		return fmt.Errorf("数据完整性验证失败: %w", err)
	}

	log.Println("系统健康检查完成，所有组件正常")
	return nil
}

// ExecuteMaintenanceTasks 执行系统维护任务
// 定期执行的系统维护和优化任务
func (o *Orchestrator) ExecuteMaintenanceTasks(ctx context.Context) error {
	log.Println("开始执行系统维护任务")

	// 重置管道统计信息（可选）
	// o.dataPipeline.ResetStats()

	// 执行数据完整性验证
	if err := o.taskExecutor.ValidateDataIntegrity(ctx); err != nil {
		log.Printf("数据完整性验证失败: %v", err)
		// 不阻断其他维护任务
	}

	// 可以添加其他维护任务，如：
	// - 清理过期数据
	// - 优化数据库索引
	// - 更新缓存
	// - 生成统计报告

	log.Println("系统维护任务执行完成")
	return nil
}

// GetTaskExecutor 获取任务执行编排服务
func (o *Orchestrator) GetTaskExecutor() *TaskExecutor {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.taskExecutor
}

// GetDataPipeline 获取数据管道业务编排服务
func (o *Orchestrator) GetDataPipeline() *DataPipeline {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.dataPipeline
}

// Shutdown 关闭业务编排器
// 优雅关闭所有业务编排服务
func (o *Orchestrator) Shutdown(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	log.Println("开始关闭业务编排器")

	// 等待数据管道完成当前任务
	for o.dataPipeline.IsRunning() {
		select {
		case <-ctx.Done():
			return fmt.Errorf("关闭超时: %w", ctx.Err())
		case <-time.After(1 * time.Second):
			log.Println("等待数据管道完成当前任务...")
		}
	}

	o.initialized = false
	log.Println("业务编排器已关闭")
	return nil
}

// BusinessFlowConfig 业务流程配置
type BusinessFlowConfig struct {
	FlowType string                 `json:"flow_type"`
	Config   map[string]interface{} `json:"config"`
	Timeout  time.Duration          `json:"timeout"`
}

// ExecuteBusinessFlowWithTimeout 带超时的业务流程执行
func (o *Orchestrator) ExecuteBusinessFlowWithTimeout(flowConfig *BusinessFlowConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), flowConfig.Timeout)
	defer cancel()

	return o.ExecuteBusinessFlow(ctx, flowConfig.FlowType, flowConfig.Config)
}

// BatchExecuteBusinessFlows 批量执行业务流程
// 支持并发执行多个业务流程
func (o *Orchestrator) BatchExecuteBusinessFlows(ctx context.Context, flows []*BusinessFlowConfig, concurrent bool) error {
	log.Printf("开始批量执行业务流程，流程数: %d, 并发: %v", len(flows), concurrent)

	if !concurrent {
		// 串行执行
		for i, flow := range flows {
			log.Printf("执行流程 %d/%d: %s", i+1, len(flows), flow.FlowType)
			if err := o.ExecuteBusinessFlow(ctx, flow.FlowType, flow.Config); err != nil {
				return fmt.Errorf("流程 %d (%s) 执行失败: %w", i+1, flow.FlowType, err)
			}
		}
	} else {
		// 并发执行
		var wg sync.WaitGroup
		errorChan := make(chan error, len(flows))

		for i, flow := range flows {
			wg.Add(1)
			go func(index int, f *BusinessFlowConfig) {
				defer wg.Done()
				log.Printf("并发执行流程 %d: %s", index+1, f.FlowType)
				if err := o.ExecuteBusinessFlow(ctx, f.FlowType, f.Config); err != nil {
					errorChan <- fmt.Errorf("流程 %d (%s) 执行失败: %w", index+1, f.FlowType, err)
				}
			}(i, flow)
		}

		wg.Wait()
		close(errorChan)

		// 检查是否有错误
		for err := range errorChan {
			if err != nil {
				return err
			}
		}
	}

	log.Println("批量业务流程执行完成")
	return nil
}
