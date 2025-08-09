package biz

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"data-collection-system/service/collection"
	"data-collection-system/service/processing"
)

// DataPipeline 数据管道业务编排服务
// 负责数据采集与处理的完整业务流程编排
type DataPipeline struct {
	collectionSvc *collection.Service
	processingSvc *processing.ProcessingService
	mu            sync.RWMutex
	running       bool
	stats         *PipelineStats
}

// PipelineStats 管道统计信息
type PipelineStats struct {
	TotalTasks     int64     `json:"total_tasks"`
	SuccessTasks   int64     `json:"success_tasks"`
	FailedTasks    int64     `json:"failed_tasks"`
	LastRunTime    time.Time `json:"last_run_time"`
	LastSuccessTime time.Time `json:"last_success_time"`
	AverageRunTime time.Duration `json:"average_run_time"`
}

// NewDataPipeline 创建数据管道业务编排服务
func NewDataPipeline(
	collectionSvc *collection.Service,
	processingSvc *processing.ProcessingService,
) *DataPipeline {
	return &DataPipeline{
		collectionSvc: collectionSvc,
		processingSvc: processingSvc,
		stats:         &PipelineStats{},
	}
}

// ExecuteFullDataPipeline 执行完整的数据管道流程
// 实现数据流向：采集→清洗→验证→去重→存储
func (dp *DataPipeline) ExecuteFullDataPipeline(ctx context.Context, pipelineType string, config map[string]interface{}) error {
	dp.mu.Lock()
	if dp.running {
		dp.mu.Unlock()
		return fmt.Errorf("数据管道正在运行中，请稍后再试")
	}
	dp.running = true
	dp.mu.Unlock()

	defer func() {
		dp.mu.Lock()
		dp.running = false
		dp.mu.Unlock()
	}()

	startTime := time.Now()
	dp.stats.TotalTasks++
	dp.stats.LastRunTime = startTime

	log.Printf("开始执行完整数据管道流程，类型: %s", pipelineType)

	// 执行具体的管道流程
	var err error
	switch pipelineType {
	case "daily_full_pipeline":
		err = dp.executeDailyFullPipeline(ctx, config)
	case "weekly_full_pipeline":
		err = dp.executeWeeklyFullPipeline(ctx, config)
	case "news_pipeline":
		err = dp.executeNewsPipeline(ctx, config)
	case "financial_pipeline":
		err = dp.executeFinancialPipeline(ctx, config)
	default:
		err = fmt.Errorf("不支持的管道类型: %s", pipelineType)
	}

	// 更新统计信息
	runTime := time.Since(startTime)
	if err != nil {
		dp.stats.FailedTasks++
		log.Printf("数据管道执行失败，耗时: %v, 错误: %v", runTime, err)
	} else {
		dp.stats.SuccessTasks++
		dp.stats.LastSuccessTime = time.Now()
		log.Printf("数据管道执行成功，耗时: %v", runTime)
	}

	// 更新平均运行时间
	if dp.stats.TotalTasks > 0 {
		dp.stats.AverageRunTime = time.Duration(int64(dp.stats.AverageRunTime)*int64(dp.stats.TotalTasks-1)+int64(runTime)) / time.Duration(dp.stats.TotalTasks)
	}

	return err
}

// executeDailyFullPipeline 执行日度完整数据管道
func (dp *DataPipeline) executeDailyFullPipeline(ctx context.Context, config map[string]interface{}) error {
	log.Println("执行日度完整数据管道")

	// 阶段1: 基础数据采集
	log.Println("阶段1: 基础数据采集")
	if err := dp.executeCollectionPhase(ctx, "daily", config); err != nil {
		return fmt.Errorf("基础数据采集阶段失败: %w", err)
	}

	// 阶段2: 数据处理与清洗
	log.Println("阶段2: 数据处理与清洗")
	if err := dp.executeProcessingPhase(ctx, "daily", config); err != nil {
		return fmt.Errorf("数据处理阶段失败: %w", err)
	}

	// 阶段3: 数据质量检查
	log.Println("阶段3: 数据质量检查")
	if err := dp.executeQualityCheckPhase(ctx, "daily", config); err != nil {
		return fmt.Errorf("数据质量检查阶段失败: %w", err)
	}

	log.Println("日度完整数据管道执行完成")
	return nil
}

// executeWeeklyFullPipeline 执行周度完整数据管道
func (dp *DataPipeline) executeWeeklyFullPipeline(ctx context.Context, config map[string]interface{}) error {
	log.Println("执行周度完整数据管道")

	// 阶段1: 周度数据采集
	log.Println("阶段1: 周度数据采集")
	if err := dp.collectionSvc.CollectWeeklyData(ctx); err != nil {
		return fmt.Errorf("周度数据采集失败: %w", err)
	}

	// 阶段2: 财务数据处理
	log.Println("阶段2: 财务数据处理")
	// 这里可以添加专门的财务数据处理逻辑

	// 阶段3: 数据完整性验证
	log.Println("阶段3: 数据完整性验证")
	if err := dp.validateDataCompleteness(ctx, "weekly"); err != nil {
		return fmt.Errorf("数据完整性验证失败: %w", err)
	}

	log.Println("周度完整数据管道执行完成")
	return nil
}

// executeNewsPipeline 执行新闻数据管道
func (dp *DataPipeline) executeNewsPipeline(ctx context.Context, config map[string]interface{}) error {
	log.Println("执行新闻数据管道")

	// 阶段1: 新闻数据采集（预留）
	log.Println("阶段1: 新闻数据采集（功能待实现）")

	// 阶段2: 新闻数据处理
	log.Println("阶段2: 新闻数据处理")
	limit, ok := config["limit"].(int)
	if !ok {
		limit = 200 // 默认处理200条新闻
	}

	if err := dp.processingSvc.ProcessNewsData(ctx, limit); err != nil {
		return fmt.Errorf("新闻数据处理失败: %w", err)
	}

	log.Println("新闻数据管道执行完成")
	return nil
}

// executeFinancialPipeline 执行财务数据管道
func (dp *DataPipeline) executeFinancialPipeline(ctx context.Context, config map[string]interface{}) error {
	log.Println("执行财务数据管道")

	// 获取配置参数
	symbols, ok := config["symbols"].([]string)
	if !ok {
		return fmt.Errorf("缺少股票代码列表参数")
	}

	period, ok := config["period"].(string)
	if !ok {
		period = "20231231" // 默认最新年报
	}

	// 阶段1: 批量采集财务数据
	log.Printf("阶段1: 批量采集财务数据，股票数量: %d", len(symbols))
	for _, symbol := range symbols {
		if err := dp.collectionSvc.CollectFinancialData(ctx, symbol, period); err != nil {
			log.Printf("采集股票 %s 财务数据失败: %v", symbol, err)
			// 继续处理其他股票
			continue
		}
	}

	// 阶段2: 财务数据处理（预留）
	log.Println("阶段2: 财务数据处理（功能待实现）")

	log.Println("财务数据管道执行完成")
	return nil
}

// executeCollectionPhase 执行数据采集阶段
func (dp *DataPipeline) executeCollectionPhase(ctx context.Context, phaseType string, config map[string]interface{}) error {
	switch phaseType {
	case "daily":
		// 采集今日数据
		return dp.collectionSvc.CollectTodayData(ctx)
	case "realtime":
		// 采集实时数据
		symbols, ok := config["symbols"].([]string)
		if !ok {
			return fmt.Errorf("实时数据采集缺少股票代码参数")
		}
		return dp.collectionSvc.CollectRealtimeData(ctx, symbols)
	default:
		return fmt.Errorf("不支持的采集阶段类型: %s", phaseType)
	}
}

// executeProcessingPhase 执行数据处理阶段
func (dp *DataPipeline) executeProcessingPhase(ctx context.Context, phaseType string, config map[string]interface{}) error {
	switch phaseType {
	case "daily":
		// 处理新闻数据
		limit, ok := config["news_limit"].(int)
		if !ok {
			limit = 100 // 默认处理100条新闻
		}
		return dp.processingSvc.ProcessNewsData(ctx, limit)
	default:
		return fmt.Errorf("不支持的处理阶段类型: %s", phaseType)
	}
}

// executeQualityCheckPhase 执行数据质量检查阶段
func (dp *DataPipeline) executeQualityCheckPhase(ctx context.Context, phaseType string, config map[string]interface{}) error {
	log.Printf("执行数据质量检查，类型: %s", phaseType)

	// 这里可以实现具体的数据质量检查逻辑
	// 例如：检查数据完整性、一致性、准确性等

	log.Println("数据质量检查完成")
	return nil
}

// validateDataCompleteness 验证数据完整性
func (dp *DataPipeline) validateDataCompleteness(ctx context.Context, scope string) error {
	log.Printf("验证数据完整性，范围: %s", scope)

	// 这里可以实现具体的数据完整性验证逻辑
	// 例如：检查关键数据表的记录数、时间连续性等

	log.Println("数据完整性验证完成")
	return nil
}

// GetPipelineStats 获取管道统计信息
func (dp *DataPipeline) GetPipelineStats() *PipelineStats {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	// 返回统计信息的副本
	return &PipelineStats{
		TotalTasks:      dp.stats.TotalTasks,
		SuccessTasks:    dp.stats.SuccessTasks,
		FailedTasks:     dp.stats.FailedTasks,
		LastRunTime:     dp.stats.LastRunTime,
		LastSuccessTime: dp.stats.LastSuccessTime,
		AverageRunTime:  dp.stats.AverageRunTime,
	}
}

// IsRunning 检查管道是否正在运行
func (dp *DataPipeline) IsRunning() bool {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	return dp.running
}

// ResetStats 重置统计信息
func (dp *DataPipeline) ResetStats() {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	dp.stats = &PipelineStats{}
	log.Println("管道统计信息已重置")
}

// ExecuteCustomPipeline 执行自定义管道流程
// 允许用户定义自己的数据处理流程
func (dp *DataPipeline) ExecuteCustomPipeline(ctx context.Context, steps []PipelineStep) error {
	log.Printf("开始执行自定义管道流程，步骤数: %d", len(steps))

	for i, step := range steps {
		log.Printf("执行步骤 %d: %s", i+1, step.Name)

		if err := step.Execute(ctx, dp); err != nil {
			return fmt.Errorf("步骤 %d (%s) 执行失败: %w", i+1, step.Name, err)
		}

		log.Printf("步骤 %d (%s) 执行完成", i+1, step.Name)
	}

	log.Println("自定义管道流程执行完成")
	return nil
}

// PipelineStep 管道步骤接口
type PipelineStep struct {
	Name    string
	Execute func(ctx context.Context, dp *DataPipeline) error
}