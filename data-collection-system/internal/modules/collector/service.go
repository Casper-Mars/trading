package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"data-collection-system/internal/cache"
	"data-collection-system/internal/config"
	"data-collection-system/pkg/logger"
)

// CollectionTask 采集任务
type CollectionTask struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`        // 任务类型：stock_basic, daily_quote, minute_quote, financial, macro
	Collector   string                 `json:"collector"`   // 采集器名称
	Parameters  map[string]interface{} `json:"parameters"`  // 任务参数
	ScheduledAt time.Time              `json:"scheduled_at"` // 计划执行时间
	Status      string                 `json:"status"`      // 任务状态：pending, running, completed, failed
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Error       string                 `json:"error,omitempty"` // 错误信息
	Result      interface{}            `json:"result,omitempty"` // 执行结果
}

// CollectionService 数据采集服务
type CollectionService struct {
	manager      *CollectorManager
	cacheManager *cache.CacheManager
	config       *config.Config
	tasks        map[string]*CollectionTask
	taskMutex    sync.RWMutex
	workerPool   chan struct{} // 工作池，控制并发数
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// NewCollectionService 创建数据采集服务
func NewCollectionService(cfg *config.Config, cacheManager *cache.CacheManager) *CollectionService {
	ctx, cancel := context.WithCancel(context.Background())
	
	service := &CollectionService{
		manager:      NewCollectorManager(),
		cacheManager: cacheManager,
		config:       cfg,
		tasks:        make(map[string]*CollectionTask),
		workerPool:   make(chan struct{}, cfg.Crawler.Parallelism), // 使用爬虫配置的并发数
		ctx:          ctx,
		cancel:       cancel,
	}
	
	// 注册Tushare采集器
	tushareCollector := NewTushareCollector(cfg.Tushare, cacheManager)
	service.manager.RegisterCollector("tushare", tushareCollector)
	
	return service
}

// Start 启动采集服务
func (cs *CollectionService) Start() error {
	logger.Info("Starting collection service...")
	
	// 验证所有采集器连接
	results := cs.manager.ValidateAllConnections(cs.ctx)
	for name, err := range results {
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to validate collector %s: %v", name, err))
			return fmt.Errorf("collector %s validation failed: %w", name, err)
		}
		logger.Info(fmt.Sprintf("Collector %s validated successfully", name))
	}
	
	// 启动任务处理器
	cs.wg.Add(1)
	go cs.taskProcessor()
	
	logger.Info("Collection service started successfully")
	return nil
}

// Stop 停止采集服务
func (cs *CollectionService) Stop() {
	logger.Info("Stopping collection service...")
	
	cs.cancel()
	cs.wg.Wait()
	cs.manager.CloseAll()
	
	logger.Info("Collection service stopped")
}

// SubmitTask 提交采集任务
func (cs *CollectionService) SubmitTask(task *CollectionTask) error {
	if task.ID == "" {
		task.ID = generateTaskID()
	}
	
	task.Status = "pending"
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	
	cs.taskMutex.Lock()
	cs.tasks[task.ID] = task
	cs.taskMutex.Unlock()
	
	logger.Info(fmt.Sprintf("Task %s submitted: %s", task.ID, task.Type))
	return nil
}

// GetTask 获取任务信息
func (cs *CollectionService) GetTask(taskID string) (*CollectionTask, bool) {
	cs.taskMutex.RLock()
	defer cs.taskMutex.RUnlock()
	
	task, exists := cs.tasks[taskID]
	return task, exists
}

// ListTasks 列出所有任务
func (cs *CollectionService) ListTasks() []*CollectionTask {
	cs.taskMutex.RLock()
	defer cs.taskMutex.RUnlock()
	
	tasks := make([]*CollectionTask, 0, len(cs.tasks))
	for _, task := range cs.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// taskProcessor 任务处理器
func (cs *CollectionService) taskProcessor() {
	defer cs.wg.Done()
	
	ticker := time.NewTicker(1 * time.Second) // 每秒检查一次待处理任务
	defer ticker.Stop()
	
	for {
		select {
		case <-cs.ctx.Done():
			return
		case <-ticker.C:
			cs.processPendingTasks()
		}
	}
}

// processPendingTasks 处理待执行任务
func (cs *CollectionService) processPendingTasks() {
	cs.taskMutex.RLock()
	pendingTasks := make([]*CollectionTask, 0)
	for _, task := range cs.tasks {
		if task.Status == "pending" && time.Now().After(task.ScheduledAt) {
			pendingTasks = append(pendingTasks, task)
		}
	}
	cs.taskMutex.RUnlock()
	
	for _, task := range pendingTasks {
		select {
		case cs.workerPool <- struct{}{}: // 获取工作池槽位
			cs.wg.Add(1)
			go cs.executeTask(task)
		default:
			// 工作池已满，跳过此次处理
			continue
		}
	}
}

// executeTask 执行采集任务
func (cs *CollectionService) executeTask(task *CollectionTask) {
	defer cs.wg.Done()
	defer func() { <-cs.workerPool }() // 释放工作池槽位
	
	// 更新任务状态
	cs.updateTaskStatus(task.ID, "running", "", nil)
	
	logger.Info(fmt.Sprintf("Executing task %s: %s", task.ID, task.Type))
	
	// 执行任务
	result, err := cs.executeTaskByType(task)
	if err != nil {
		logger.Error(fmt.Sprintf("Task %s failed: %v", task.ID, err))
		cs.updateTaskStatus(task.ID, "failed", err.Error(), nil)
		return
	}
	
	logger.Info(fmt.Sprintf("Task %s completed successfully", task.ID))
	cs.updateTaskStatus(task.ID, "completed", "", result)
}

// executeTaskByType 根据任务类型执行任务
func (cs *CollectionService) executeTaskByType(task *CollectionTask) (interface{}, error) {
	collector, exists := cs.manager.GetCollector(task.Collector)
	if !exists {
		return nil, fmt.Errorf("collector %s not found", task.Collector)
	}
	
	ctx, cancel := context.WithTimeout(cs.ctx, 5*time.Minute) // 5分钟超时
	defer cancel()
	
	switch task.Type {
	case "stock_basic":
		return cs.executeStockBasicTask(ctx, collector, task.Parameters)
	case "daily_quote":
		return cs.executeDailyQuoteTask(ctx, collector, task.Parameters)
	case "minute_quote":
		return cs.executeMinuteQuoteTask(ctx, collector, task.Parameters)
	case "financial":
		return cs.executeFinancialTask(ctx, collector, task.Parameters)
	case "macro":
		return cs.executeMacroTask(ctx, collector, task.Parameters)
	default:
		return nil, fmt.Errorf("unsupported task type: %s", task.Type)
	}
}

// executeStockBasicTask 执行股票基础数据采集任务
func (cs *CollectionService) executeStockBasicTask(ctx context.Context, collector DataCollector, params map[string]interface{}) (interface{}, error) {
	stockCollector, ok := collector.(StockDataCollector)
	if !ok {
		return nil, fmt.Errorf("collector does not support stock data collection")
	}
	
	exchange, _ := params["exchange"].(string)
	if exchange == "" {
		exchange = "SSE" // 默认上交所
	}
	
	return stockCollector.CollectStockBasic(ctx, exchange)
}

// executeDailyQuoteTask 执行日线行情采集任务
func (cs *CollectionService) executeDailyQuoteTask(ctx context.Context, collector DataCollector, params map[string]interface{}) (interface{}, error) {
	stockCollector, ok := collector.(StockDataCollector)
	if !ok {
		return nil, fmt.Errorf("collector does not support stock data collection")
	}
	
	tsCode, _ := params["ts_code"].(string)

	
	if tsCode == "" {
		return nil, fmt.Errorf("ts_code is required")
	}
	
	startDate, _ := params["start_date"].(string)
	endDate, _ := params["end_date"].(string)
	
	return stockCollector.CollectMarketData(ctx, tsCode, "1d", startDate, endDate)
}

// executeMinuteQuoteTask 执行分钟级行情采集任务
func (cs *CollectionService) executeMinuteQuoteTask(ctx context.Context, collector DataCollector, params map[string]interface{}) (interface{}, error) {
	stockCollector, ok := collector.(StockDataCollector)
	if !ok {
		return nil, fmt.Errorf("collector does not support stock data collection")
	}
	
	tsCode, _ := params["ts_code"].(string)
	freq, _ := params["freq"].(string)

	
	if tsCode == "" {
		return nil, fmt.Errorf("ts_code is required")
	}
	if freq == "" {
		freq = "1min" // 默认1分钟
	}
	
	startDate, _ := params["start_date"].(string)
	endDate, _ := params["end_date"].(string)
	
	return stockCollector.CollectMarketData(ctx, tsCode, freq, startDate, endDate)
}

// executeFinancialTask 执行财务数据采集任务
func (cs *CollectionService) executeFinancialTask(ctx context.Context, collector DataCollector, params map[string]interface{}) (interface{}, error) {
	financialCollector, ok := collector.(FinancialDataCollector)
	if !ok {
		return nil, fmt.Errorf("collector does not support financial data collection")
	}
	
	tsCode, _ := params["ts_code"].(string)
	period, _ := params["period"].(string)

	
	if tsCode == "" {
		return nil, fmt.Errorf("ts_code is required")
	}
	if period == "" {
		period = "20231231" // 默认年报
	}
	
	return financialCollector.CollectFinancialData(ctx, tsCode, period, "", "")
}

// executeMacroTask 执行宏观数据采集任务
func (cs *CollectionService) executeMacroTask(ctx context.Context, collector DataCollector, params map[string]interface{}) (interface{}, error) {
	macroCollector, ok := collector.(MacroDataCollector)
	if !ok {
		return nil, fmt.Errorf("collector does not support macro data collection")
	}
	
	indicator, _ := params["indicator"].(string)

	
	if indicator == "" {
		return nil, fmt.Errorf("indicator is required")
	}
	
	return macroCollector.CollectMacroData(ctx, indicator, "", "", "")
}

// updateTaskStatus 更新任务状态
func (cs *CollectionService) updateTaskStatus(taskID, status, errorMsg string, result interface{}) {
	cs.taskMutex.Lock()
	defer cs.taskMutex.Unlock()
	
	if task, exists := cs.tasks[taskID]; exists {
		task.Status = status
		task.UpdatedAt = time.Now()
		task.Error = errorMsg
		task.Result = result
	}
}

// CreateStockBasicTask 创建股票基础数据采集任务
func (cs *CollectionService) CreateStockBasicTask(exchange string, scheduledAt time.Time) *CollectionTask {
	return &CollectionTask{
		ID:          generateTaskID(),
		Type:        "stock_basic",
		Collector:   "tushare",
		Parameters:  map[string]interface{}{"exchange": exchange},
		ScheduledAt: scheduledAt,
	}
}

// CreateDailyQuoteTask 创建日线行情采集任务
func (cs *CollectionService) CreateDailyQuoteTask(tsCode, startDate, endDate string, scheduledAt time.Time) *CollectionTask {
	return &CollectionTask{
		ID:   generateTaskID(),
		Type: "daily_quote",
		Collector: "tushare",
		Parameters: map[string]interface{}{
			"ts_code":    tsCode,
			"start_date": startDate,
			"end_date":   endDate,
		},
		ScheduledAt: scheduledAt,
	}
}

// CreateMinuteQuoteTask 创建分钟级行情采集任务
func (cs *CollectionService) CreateMinuteQuoteTask(tsCode, freq, startDate, endDate string, scheduledAt time.Time) *CollectionTask {
	return &CollectionTask{
		ID:   generateTaskID(),
		Type: "minute_quote",
		Collector: "tushare",
		Parameters: map[string]interface{}{
			"ts_code":    tsCode,
			"freq":       freq,
			"start_date": startDate,
			"end_date":   endDate,
		},
		ScheduledAt: scheduledAt,
	}
}

// CreateFinancialTask 创建财务数据采集任务
func (cs *CollectionService) CreateFinancialTask(tsCode, period, startDate, endDate string, scheduledAt time.Time) *CollectionTask {
	return &CollectionTask{
		ID:   generateTaskID(),
		Type: "financial",
		Collector: "tushare",
		Parameters: map[string]interface{}{
			"ts_code":    tsCode,
			"period":     period,
			"start_date": startDate,
			"end_date":   endDate,
		},
		ScheduledAt: scheduledAt,
	}
}

// CreateMacroTask 创建宏观数据采集任务
func (cs *CollectionService) CreateMacroTask(indicator, startDate, endDate string, scheduledAt time.Time) *CollectionTask {
	return &CollectionTask{
		ID:   generateTaskID(),
		Type: "macro",
		Collector: "tushare",
		Parameters: map[string]interface{}{
			"indicator":  indicator,
			"start_date": startDate,
			"end_date":   endDate,
		},
		ScheduledAt: scheduledAt,
	}
}

// GetCollectorStatus 获取采集器状态
func (cs *CollectionService) GetCollectorStatus() map[string]string {
	results := cs.manager.ValidateAllConnections(cs.ctx)
	status := make(map[string]string)
	for name, err := range results {
		if err != nil {
			status[name] = "error: " + err.Error()
		} else {
			status[name] = "healthy"
		}
	}
	return status
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}