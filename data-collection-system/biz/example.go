package biz

import (
	"context"
	"log"
	"time"

	"data-collection-system/service/collection"
	"data-collection-system/service/processing"
)

// ExampleUsage 展示如何使用业务编排服务的示例
// 这个文件仅用于演示，实际使用时应该在具体的业务场景中调用
func ExampleUsage() {
	// 注意：这里的服务实例化仅为示例，实际使用时应该通过依赖注入获取
	// var collectionSvc *collection.Service
	// var processingSvc *processing.ProcessingService

	// 1. 创建业务编排器
	// orchestrator := NewOrchestrator(collectionSvc, processingSvc)

	// 2. 执行日度数据采集与处理
	// ctx := context.Background()
	// config := map[string]interface{}{
	//     "task_type": "daily_stock_data",
	//     "trade_date": time.Now().Format("20060102"),
	// }
	// err := orchestrator.ExecuteBusinessFlow(ctx, "data_collection_processing", config)
	// if err != nil {
	//     log.Printf("执行日度数据采集失败: %v", err)
	// }

	// 3. 执行完整数据管道
	// pipelineConfig := map[string]interface{}{
	//     "news_limit": 200,
	// }
	// err = orchestrator.ExecuteBusinessFlow(ctx, "daily_full_pipeline", pipelineConfig)
	// if err != nil {
	//     log.Printf("执行完整数据管道失败: %v", err)
	// }

	// 4. 获取系统状态
	// status, err := orchestrator.GetSystemStatus(ctx)
	// if err != nil {
	//     log.Printf("获取系统状态失败: %v", err)
	// } else {
	//     log.Printf("系统状态: %+v", status)
	// }

	log.Println("业务编排服务示例代码，请参考注释中的用法")
}

// CreateExampleTaskExecutor 创建示例任务执行器
func CreateExampleTaskExecutor(collectionSvc *collection.Service, processingSvc *processing.ProcessingService) *TaskExecutor {
	return NewTaskExecutor(collectionSvc, processingSvc)
}

// CreateExampleDataPipeline 创建示例数据管道
func CreateExampleDataPipeline(collectionSvc *collection.Service, processingSvc *processing.ProcessingService) *DataPipeline {
	return NewDataPipeline(collectionSvc, processingSvc)
}

// CreateExampleOrchestrator 创建示例业务编排器
func CreateExampleOrchestrator(collectionSvc *collection.Service, processingSvc *processing.ProcessingService) *Orchestrator {
	return NewOrchestrator(collectionSvc, processingSvc)
}

// ExampleDailyDataFlow 示例：执行日度数据流程
func ExampleDailyDataFlow(orchestrator *Orchestrator) error {
	ctx := context.Background()

	// 配置日度数据采集参数
	config := map[string]interface{}{
		"task_type":  "daily_stock_data",
		"trade_date": time.Now().Format("20060102"),
	}

	// 执行数据采集与处理
	return orchestrator.ExecuteBusinessFlow(ctx, "data_collection_processing", config)
}

// ExampleNewsDataFlow 示例：执行新闻数据流程
func ExampleNewsDataFlow(orchestrator *Orchestrator) error {
	ctx := context.Background()

	// 配置新闻数据处理参数
	config := map[string]interface{}{
		"task_type": "news_data",
		"limit":     150,
	}

	// 执行新闻数据采集与处理
	return orchestrator.ExecuteBusinessFlow(ctx, "data_collection_processing", config)
}

// ExampleBatchDataCollection 示例：执行批量数据采集
func ExampleBatchDataCollection(orchestrator *Orchestrator) error {
	ctx := context.Background()

	// 配置批量采集参数
	config := map[string]interface{}{
		"symbols":    []string{"000001.SZ", "000002.SZ", "600000.SH"},
		"start_date": "20240101",
		"end_date":   "20241231",
	}

	// 执行批量数据采集
	return orchestrator.ExecuteBusinessFlow(ctx, "batch_data_collection", config)
}

// ExampleFullPipeline 示例：执行完整数据管道
func ExampleFullPipeline(orchestrator *Orchestrator) error {
	ctx := context.Background()

	// 配置完整管道参数
	config := map[string]interface{}{
		"news_limit": 200,
	}

	// 执行日度完整管道
	return orchestrator.ExecuteBusinessFlow(ctx, "daily_full_pipeline", config)
}

// ExampleCustomPipeline 示例：执行自定义管道
func ExampleCustomPipeline(orchestrator *Orchestrator) error {
	ctx := context.Background()

	// 定义自定义管道步骤
	steps := []PipelineStep{
		{
			Name: "数据采集步骤",
			Execute: func(ctx context.Context, dp *DataPipeline) error {
				log.Println("执行自定义数据采集逻辑")
				// 这里可以调用具体的采集逻辑
				return nil
			},
		},
		{
			Name: "数据处理步骤",
			Execute: func(ctx context.Context, dp *DataPipeline) error {
				log.Println("执行自定义数据处理逻辑")
				// 这里可以调用具体的处理逻辑
				return nil
			},
		},
		{
			Name: "数据验证步骤",
			Execute: func(ctx context.Context, dp *DataPipeline) error {
				log.Println("执行自定义数据验证逻辑")
				// 这里可以调用具体的验证逻辑
				return nil
			},
		},
	}

	// 配置自定义管道参数
	config := map[string]interface{}{
		"steps": steps,
	}

	// 执行自定义管道
	return orchestrator.ExecuteBusinessFlow(ctx, "custom_pipeline", config)
}

// ExampleSystemHealthCheck 示例：系统健康检查
func ExampleSystemHealthCheck(orchestrator *Orchestrator) error {
	ctx := context.Background()

	// 执行系统健康检查
	if err := orchestrator.ValidateSystemHealth(ctx); err != nil {
		return err
	}

	// 获取系统状态
	status, err := orchestrator.GetSystemStatus(ctx)
	if err != nil {
		return err
	}

	log.Printf("系统状态检查完成: %+v", status)
	return nil
}

// ExampleBatchFlowExecution 示例：批量流程执行
func ExampleBatchFlowExecution(orchestrator *Orchestrator) error {
	ctx := context.Background()

	// 定义多个业务流程
	flows := []*BusinessFlowConfig{
		{
			FlowType: "data_collection_processing",
			Config: map[string]interface{}{
				"task_type":  "daily_stock_data",
				"trade_date": time.Now().Format("20060102"),
			},
			Timeout: 30 * time.Minute,
		},
		{
			FlowType: "news_pipeline",
			Config: map[string]interface{}{
				"limit": 100,
			},
			Timeout: 15 * time.Minute,
		},
		{
			FlowType: "comprehensive_data_update",
			Config:   map[string]interface{}{},
			Timeout:  45 * time.Minute,
		},
	}

	// 串行执行多个流程
	return orchestrator.BatchExecuteBusinessFlows(ctx, flows, false)
}
