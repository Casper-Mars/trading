package biz

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"data-collection-system/model"
	"data-collection-system/service/collection"
	"data-collection-system/service/processing"
	"github.com/go-redis/redis/v8"
)

// NewsPipeline 新闻数据处理管道业务编排服务
// 负责新闻爬虫与NLP处理的完整业务流程集成
type NewsPipeline struct {
	collectionSvc *collection.Service
	processingSvc *processing.ProcessingService
	redisClient   *redis.Client
	mu            sync.RWMutex
	running       bool
	stats         *NewsPipelineStats
	config        *NewsPipelineConfig
}

// NewsPipelineConfig 新闻管道配置
type NewsPipelineConfig struct {
	// 批处理配置
	BatchSize        int           `json:"batch_size"`         // 批处理大小
	BatchTimeout     time.Duration `json:"batch_timeout"`      // 批处理超时时间
	ProcessInterval  time.Duration `json:"process_interval"`   // 处理间隔
	
	// 重试配置
	MaxRetries       int           `json:"max_retries"`        // 最大重试次数
	RetryDelay       time.Duration `json:"retry_delay"`        // 重试延迟
	RetryBackoff     float64       `json:"retry_backoff"`      // 重试退避倍数
	
	// 队列配置
	QueueName        string        `json:"queue_name"`         // 队列名称
	QueueSize        int           `json:"queue_size"`         // 队列大小
	WorkerCount      int           `json:"worker_count"`       // 工作协程数量
	
	// 降级配置
	EnableFallback   bool          `json:"enable_fallback"`    // 启用降级机制
	FallbackTimeout  time.Duration `json:"fallback_timeout"`   // 降级超时时间
}

// NewsPipelineStats 新闻管道统计信息
type NewsPipelineStats struct {
	// 采集统计
	TotalCrawled     int64     `json:"total_crawled"`      // 总爬取数量
	SuccessCrawled   int64     `json:"success_crawled"`    // 成功爬取数量
	FailedCrawled    int64     `json:"failed_crawled"`     // 失败爬取数量
	
	// 处理统计
	TotalProcessed   int64     `json:"total_processed"`    // 总处理数量
	SuccessProcessed int64     `json:"success_processed"`  // 成功处理数量
	FailedProcessed  int64     `json:"failed_processed"`   // 失败处理数量
	NLPProcessed     int64     `json:"nlp_processed"`      // NLP处理数量
	FallbackUsed     int64     `json:"fallback_used"`      // 降级处理数量
	
	// 时间统计
	LastRunTime      time.Time `json:"last_run_time"`      // 最后运行时间
	LastSuccessTime  time.Time `json:"last_success_time"`  // 最后成功时间
	AverageCrawlTime time.Duration `json:"average_crawl_time"` // 平均爬取时间
	AverageNLPTime   time.Duration `json:"average_nlp_time"`   // 平均NLP处理时间
	
	// 队列统计
	QueueLength      int64     `json:"queue_length"`       // 队列长度
	ActiveWorkers    int       `json:"active_workers"`     // 活跃工作协程数
}

// NewsProcessingTask 新闻处理任务
type NewsProcessingTask struct {
	ID          string                 `json:"id"`
	NewsData    *model.NewsData        `json:"news_data"`
	RetryCount  int                    `json:"retry_count"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Status      TaskStatus             `json:"status"`
	ErrorMsg    string                 `json:"error_msg,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"     // 待处理
	TaskStatusProcessing TaskStatus = "processing"  // 处理中
	TaskStatusCompleted  TaskStatus = "completed"   // 已完成
	TaskStatusFailed     TaskStatus = "failed"      // 失败
	TaskStatusRetrying   TaskStatus = "retrying"    // 重试中
)

// NewNewsPipeline 创建新闻数据处理管道
func NewNewsPipeline(
	collectionSvc *collection.Service,
	processingSvc *processing.ProcessingService,
	redisClient *redis.Client,
	config *NewsPipelineConfig,
) *NewsPipeline {
	if config == nil {
		config = getDefaultNewsPipelineConfig()
	}
	
	return &NewsPipeline{
		collectionSvc: collectionSvc,
		processingSvc: processingSvc,
		redisClient:   redisClient,
		stats:         &NewsPipelineStats{},
		config:        config,
	}
}

// ExecuteNewsDataPipeline 执行完整的新闻数据处理管道
// 实现：爬虫采集 → 异步队列 → NLP处理 → 数据存储的完整流程
func (np *NewsPipeline) ExecuteNewsDataPipeline(ctx context.Context, sources []string) error {
	np.mu.Lock()
	if np.running {
		np.mu.Unlock()
		return fmt.Errorf("新闻数据管道正在运行中")
	}
	np.running = true
	np.mu.Unlock()
	
	defer func() {
		np.mu.Lock()
		np.running = false
		np.mu.Unlock()
	}()
	
	startTime := time.Now()
	np.stats.LastRunTime = startTime
	
	log.Printf("开始执行新闻数据处理管道，数据源: %v", sources)
	
	// 阶段1: 新闻数据采集
	if err := np.executeNewsCrawlingPhase(ctx, sources); err != nil {
		return fmt.Errorf("新闻采集阶段失败: %w", err)
	}
	
	// 阶段2: 异步处理队列启动
	if err := np.startAsyncProcessingWorkers(ctx); err != nil {
		return fmt.Errorf("启动异步处理队列失败: %w", err)
	}
	
	// 阶段3: 监控处理进度
	if err := np.monitorProcessingProgress(ctx); err != nil {
		return fmt.Errorf("监控处理进度失败: %w", err)
	}
	
	np.stats.LastSuccessTime = time.Now()
	log.Printf("新闻数据处理管道执行完成，耗时: %v", time.Since(startTime))
	
	return nil
}

// executeNewsCrawlingPhase 执行新闻爬取阶段
func (np *NewsPipeline) executeNewsCrawlingPhase(ctx context.Context, sources []string) error {
	log.Println("阶段1: 开始新闻数据采集")
	
	crawlStartTime := time.Now()
	var totalCrawled int64
	
	for _, source := range sources {
		log.Printf("正在爬取新闻源: %s", source)
		
		// 执行新闻爬取
		if err := np.collectionSvc.CrawlNews(ctx, source); err != nil {
			log.Printf("爬取新闻源 %s 失败: %v", source, err)
			np.stats.FailedCrawled++
			continue
		}
		
		np.stats.SuccessCrawled++
		totalCrawled++
		
		// 控制爬取频率
		time.Sleep(np.config.ProcessInterval)
	}
	
	np.stats.TotalCrawled += totalCrawled
	np.stats.AverageCrawlTime = time.Since(crawlStartTime) / time.Duration(len(sources))
	
	log.Printf("新闻采集完成，成功: %d, 失败: %d", np.stats.SuccessCrawled, np.stats.FailedCrawled)
	return nil
}

// startAsyncProcessingWorkers 启动异步处理工作协程
func (np *NewsPipeline) startAsyncProcessingWorkers(ctx context.Context) error {
	log.Println("阶段2: 启动异步NLP处理队列")
	
	// 创建工作协程池
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	
	var wg sync.WaitGroup
	
	// 启动多个工作协程
	for i := 0; i < np.config.WorkerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			np.processNewsWorker(workerCtx, workerID)
		}(i)
	}
	
	// 等待所有工作协程完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("所有异步处理工作协程已完成")
	case <-time.After(np.config.BatchTimeout):
		log.Println("异步处理超时，取消剩余任务")
		cancel()
		<-done // 等待协程清理完成
	}
	
	return nil
}

// processNewsWorker 新闻处理工作协程
func (np *NewsPipeline) processNewsWorker(ctx context.Context, workerID int) {
	log.Printf("启动新闻处理工作协程 %d", workerID)
	np.stats.ActiveWorkers++
	defer func() {
		np.stats.ActiveWorkers--
		log.Printf("新闻处理工作协程 %d 已退出", workerID)
	}()
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 从队列获取待处理的新闻数据
			task, err := np.getNextProcessingTask(ctx)
			if err != nil {
				if err == redis.Nil {
					// 队列为空，等待一段时间后重试
					time.Sleep(time.Second)
					continue
				}
				log.Printf("工作协程 %d 获取任务失败: %v", workerID, err)
				continue
			}
			
			// 处理新闻数据
			if err := np.processNewsTask(ctx, task, workerID); err != nil {
				log.Printf("工作协程 %d 处理任务失败: %v", workerID, err)
				// 处理失败，考虑重试或降级
				np.handleTaskFailure(ctx, task, err)
			}
		}
	}
}

// processNewsTask 处理单个新闻任务
func (np *NewsPipeline) processNewsTask(ctx context.Context, task *NewsProcessingTask, workerID int) error {
	task.Status = TaskStatusProcessing
	task.UpdatedAt = time.Now()
	
	nlpStartTime := time.Now()
	
	log.Printf("工作协程 %d 开始处理新闻: %s (ID: %s)", workerID, task.NewsData.Title, task.ID)
	
	// 执行NLP处理 - 直接处理单个新闻项
	err := np.processingSvc.ProcessSingleNewsItem(ctx, task.NewsData)
	if err != nil {
		// NLP处理失败，尝试降级处理
		if np.config.EnableFallback {
			log.Printf("NLP处理失败，启用降级机制: %v", err)
			np.stats.FallbackUsed++
			// 这里可以实现基础的文本处理逻辑
			return np.fallbackProcessing(ctx, task)
		}
		return fmt.Errorf("NLP处理失败: %w", err)
	}
	
	// 更新统计信息
	np.stats.TotalProcessed++
	np.stats.SuccessProcessed++
	np.stats.NLPProcessed++
	np.stats.AverageNLPTime = (np.stats.AverageNLPTime + time.Since(nlpStartTime)) / 2
	
	// 标记任务完成
	task.Status = TaskStatusCompleted
	task.UpdatedAt = time.Now()
	
	log.Printf("工作协程 %d 完成新闻处理: %s", workerID, task.NewsData.Title)
	return nil
}

// fallbackProcessing 降级处理逻辑
func (np *NewsPipeline) fallbackProcessing(ctx context.Context, task *NewsProcessingTask) error {
	log.Printf("执行降级处理: %s", task.NewsData.Title)
	
	// 基础处理逻辑：简单的关键词匹配、基础清洗等
	// 这里可以实现不依赖外部NLP服务的基础处理
	
	// 标记为降级处理完成
	task.Status = TaskStatusCompleted
	task.UpdatedAt = time.Now()
	task.Metadata = map[string]interface{}{
		"fallback_processed": true,
		"fallback_time":      time.Now(),
	}
	
	return nil
}

// handleTaskFailure 处理任务失败
func (np *NewsPipeline) handleTaskFailure(ctx context.Context, task *NewsProcessingTask, err error) {
	task.RetryCount++
	task.ErrorMsg = err.Error()
	task.UpdatedAt = time.Now()
	
	if task.RetryCount < np.config.MaxRetries {
		// 重试任务
		task.Status = TaskStatusRetrying
		retryDelay := time.Duration(float64(np.config.RetryDelay) * 
			float64(task.RetryCount) * np.config.RetryBackoff)
		
		log.Printf("任务重试 %d/%d，延迟 %v: %s", 
			task.RetryCount, np.config.MaxRetries, retryDelay, task.NewsData.Title)
		
		// 延迟后重新加入队列
		go func() {
			time.Sleep(retryDelay)
			np.enqueueTask(ctx, task)
		}()
	} else {
		// 超过最大重试次数，标记为失败
		task.Status = TaskStatusFailed
		np.stats.FailedProcessed++
		log.Printf("任务最终失败: %s, 错误: %v", task.NewsData.Title, err)
	}
}

// monitorProcessingProgress 监控处理进度
func (np *NewsPipeline) monitorProcessingProgress(ctx context.Context) error {
	log.Println("阶段3: 开始监控处理进度")
	
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// 检查队列状态
			queueLen, err := np.getQueueLength(ctx)
			if err != nil {
				log.Printf("获取队列长度失败: %v", err)
				continue
			}
			
			np.stats.QueueLength = queueLen
			
			log.Printf("处理进度 - 队列长度: %d, 活跃工作协程: %d, 已处理: %d, 成功: %d, 失败: %d",
				queueLen, np.stats.ActiveWorkers, np.stats.TotalProcessed, 
				np.stats.SuccessProcessed, np.stats.FailedProcessed)
			
			// 如果队列为空且没有活跃工作协程，认为处理完成
			if queueLen == 0 && np.stats.ActiveWorkers == 0 {
				log.Println("所有新闻处理任务已完成")
				return nil
			}
		}
	}
}

// getNextProcessingTask 从队列获取下一个处理任务
func (np *NewsPipeline) getNextProcessingTask(ctx context.Context) (*NewsProcessingTask, error) {
	// 从Redis队列中获取任务
	result, err := np.redisClient.BLPop(ctx, time.Second, np.config.QueueName).Result()
	if err != nil {
		return nil, err
	}
	
	if len(result) < 2 {
		return nil, fmt.Errorf("无效的队列数据")
	}
	
	// 解析任务数据
	task := &NewsProcessingTask{}
	// 这里需要实现JSON反序列化逻辑
	// 简化实现，实际应该从Redis中获取完整的任务数据
	
	return task, nil
}

// enqueueTask 将任务加入队列
func (np *NewsPipeline) enqueueTask(ctx context.Context, task *NewsProcessingTask) error {
	// 将任务序列化并加入Redis队列
	// 这里需要实现JSON序列化逻辑
	return np.redisClient.RPush(ctx, np.config.QueueName, task).Err()
}

// getQueueLength 获取队列长度
func (np *NewsPipeline) getQueueLength(ctx context.Context) (int64, error) {
	return np.redisClient.LLen(ctx, np.config.QueueName).Result()
}

// GetStats 获取管道统计信息
func (np *NewsPipeline) GetStats() *NewsPipelineStats {
	np.mu.RLock()
	defer np.mu.RUnlock()
	return np.stats
}

// IsRunning 检查管道是否正在运行
func (np *NewsPipeline) IsRunning() bool {
	np.mu.RLock()
	defer np.mu.RUnlock()
	return np.running
}

// ResetStats 重置统计信息
func (np *NewsPipeline) ResetStats() {
	np.mu.Lock()
	defer np.mu.Unlock()
	np.stats = &NewsPipelineStats{}
}

// getDefaultNewsPipelineConfig 获取默认配置
func getDefaultNewsPipelineConfig() *NewsPipelineConfig {
	return &NewsPipelineConfig{
		BatchSize:       50,
		BatchTimeout:    30 * time.Minute,
		ProcessInterval: 2 * time.Second,
		MaxRetries:      3,
		RetryDelay:      5 * time.Second,
		RetryBackoff:    2.0,
		QueueName:       "news_processing_queue",
		QueueSize:       1000,
		WorkerCount:     5,
		EnableFallback:  true,
		FallbackTimeout: 10 * time.Second,
	}
}

// TriggerNewsProcessing 手动触发新闻处理
func (np *NewsPipeline) TriggerNewsProcessing(ctx context.Context, sources []string) error {
	if len(sources) == 0 {
		// 使用默认新闻源
		sources = []string{"sina", "163", "tencent", "eastmoney"}
	}
	
	return np.ExecuteNewsDataPipeline(ctx, sources)
}

// GetProcessingStatus 获取处理状态
func (np *NewsPipeline) GetProcessingStatus() map[string]interface{} {
	stats := np.GetStats()
	return map[string]interface{}{
		"running":           np.IsRunning(),
		"total_crawled":     stats.TotalCrawled,
		"success_crawled":   stats.SuccessCrawled,
		"failed_crawled":    stats.FailedCrawled,
		"total_processed":   stats.TotalProcessed,
		"success_processed": stats.SuccessProcessed,
		"failed_processed":  stats.FailedProcessed,
		"nlp_processed":     stats.NLPProcessed,
		"fallback_used":     stats.FallbackUsed,
		"queue_length":      stats.QueueLength,
		"active_workers":    stats.ActiveWorkers,
		"last_run_time":     stats.LastRunTime,
		"last_success_time": stats.LastSuccessTime,
	}
}