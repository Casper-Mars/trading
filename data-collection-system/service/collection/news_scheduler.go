package collection

import (
	"context"
	"fmt"
	"sync"
	"time"

	"data-collection-system/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewsScheduler 新闻爬虫调度器
type NewsScheduler struct {
	crawlerService *NewsCrawlerService
	newsSources    []*NewsSource
	logger         *logrus.Logger
	running        bool
	mu             sync.RWMutex
	stopChan       chan struct{}
	tickers        map[string]*time.Ticker
	errorHandler   ErrorHandler
}

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	HandleError(source string, err error)
}

// DefaultErrorHandler 默认错误处理器
type DefaultErrorHandler struct {
	logger *logrus.Logger
}

// HandleError 处理错误
func (h *DefaultErrorHandler) HandleError(source string, err error) {
	h.logger.Error("新闻源 %s 爬取失败: %v", source, err)
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	MaxConcurrency int           `yaml:"max_concurrency"` // 最大并发数
	DefaultFreq    time.Duration `yaml:"default_freq"`    // 默认更新频率
	RetryInterval  time.Duration `yaml:"retry_interval"`  // 重试间隔
	MaxRetries     int           `yaml:"max_retries"`     // 最大重试次数
	HealthCheck    time.Duration `yaml:"health_check"`    // 健康检查间隔
}

// NewNewsScheduler 创建新闻调度器
func NewNewsScheduler(
	crawlerService *NewsCrawlerService,
	newsSources []*NewsSource,
	logger *logrus.Logger,
) *NewsScheduler {
	if logger == nil {
		logger = logrus.New()
	}

	return &NewsScheduler{
		crawlerService: crawlerService,
		newsSources:    newsSources,
		logger:         logger,
		running:        false,
		stopChan:       make(chan struct{}),
		tickers:        make(map[string]*time.Ticker),
		errorHandler:   &DefaultErrorHandler{logger: logger},
	}
}

// SetErrorHandler 设置错误处理器
func (s *NewsScheduler) SetErrorHandler(handler ErrorHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errorHandler = handler
}

// Start 启动调度器
func (s *NewsScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return errors.New(errors.ErrCodeInvalidOperation, "调度器已在运行")
	}

	s.running = true
	s.logger.Info("启动新闻爬虫调度器")

	// 为每个新闻源创建定时器
	for _, source := range s.newsSources {
		if !source.Enabled {
			continue
		}

		freq := source.UpdateFreq
		if freq == 0 {
			freq = 30 * time.Minute // 默认30分钟
		}

		ticker := time.NewTicker(freq)
		s.tickers[source.Name] = ticker

		// 启动goroutine处理该新闻源
		go s.scheduleNewsSource(ctx, source, ticker)

		s.logger.Info("为新闻源 %s 创建定时任务，更新频率: %v", source.Name, freq)
	}

	// 启动健康检查
	go s.healthCheck(ctx)

	// 立即执行一次全量爬取
	go func() {
		if err := s.crawlerService.CrawlMultipleSources(ctx, s.getEnabledSources()); err != nil {
			s.errorHandler.HandleError("初始化爬取", err)
		}
	}()

	return nil
}

// Stop 停止调度器
func (s *NewsScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.logger.Info("停止新闻爬虫调度器")
	s.running = false

	// 停止所有定时器
	for name, ticker := range s.tickers {
		ticker.Stop()
		s.logger.Debug("停止新闻源 %s 的定时任务", name)
	}
	s.tickers = make(map[string]*time.Ticker)

	// 发送停止信号
	close(s.stopChan)

	// 停止爬虫服务
	s.crawlerService.Stop()
}

// scheduleNewsSource 调度单个新闻源
func (s *NewsScheduler) scheduleNewsSource(ctx context.Context, source *NewsSource, ticker *time.Ticker) {
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("上下文取消，停止新闻源 %s 的调度", source.Name)
			return
		case <-s.stopChan:
			s.logger.Info("收到停止信号，停止新闻源 %s 的调度", source.Name)
			return
		case <-ticker.C:
			s.logger.Debug("开始定时爬取新闻源: %s", source.Name)
			
			// 执行爬取任务
			if err := s.crawlWithRetry(ctx, source); err != nil {
				s.errorHandler.HandleError(source.Name, err)
			}
		}
	}
}

// crawlWithRetry 带重试的爬取
func (s *NewsScheduler) crawlWithRetry(ctx context.Context, source *NewsSource) error {
	maxRetries := 3
	retryInterval := 5 * time.Second

	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			s.logger.Info("重试爬取新闻源 %s，第 %d 次重试", source.Name, i)
			time.Sleep(retryInterval)
		}

		err := s.crawlerService.CrawlNewsSource(ctx, source)
		if err == nil {
			if i > 0 {
				s.logger.Info("重试成功，新闻源 %s 爬取完成", source.Name)
			}
			return nil
		}

		lastErr = err
		s.logger.Warn("爬取新闻源 %s 失败 (第 %d 次): %v", source.Name, i+1, err)
	}

	return fmt.Errorf("爬取新闻源 %s 失败，已重试 %d 次: %w", source.Name, maxRetries, lastErr)
}

// healthCheck 健康检查
func (s *NewsScheduler) healthCheck(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (s *NewsScheduler) performHealthCheck() {
	s.mu.RLock()
	running := s.running
	tickersCount := len(s.tickers)
	enabledSources := len(s.getEnabledSources())
	s.mu.RUnlock()

	s.logger.Debug("健康检查 - 运行状态: %v, 活跃定时器: %d, 启用的新闻源: %d", 
		running, tickersCount, enabledSources)

	// 检查是否有定时器异常停止
	if running && tickersCount != enabledSources {
		s.logger.Warn("检测到定时器数量异常，预期: %d, 实际: %d", enabledSources, tickersCount)
	}
}

// getEnabledSources 获取启用的新闻源
func (s *NewsScheduler) getEnabledSources() []*NewsSource {
	var enabled []*NewsSource
	for _, source := range s.newsSources {
		if source.Enabled {
			enabled = append(enabled, source)
		}
	}
	return enabled
}

// IsRunning 检查调度器是否在运行
func (s *NewsScheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetStatus 获取调度器状态
func (s *NewsScheduler) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"running":         s.running,
		"active_tickers":  len(s.tickers),
		"total_sources":   len(s.newsSources),
		"enabled_sources": len(s.getEnabledSources()),
	}

	// 添加每个新闻源的状态
	sources := make(map[string]interface{})
	for _, source := range s.newsSources {
		sources[source.Name] = map[string]interface{}{
			"enabled":     source.Enabled,
			"category":    source.Category,
			"update_freq": source.UpdateFreq.String(),
			"has_ticker":  s.tickers[source.Name] != nil,
		}
	}
	status["sources"] = sources

	return status
}

// UpdateNewsSource 更新新闻源配置
func (s *NewsScheduler) UpdateNewsSource(name string, source *NewsSource) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 查找并更新新闻源
	for i, src := range s.newsSources {
		if src.Name == name {
			// 停止旧的定时器
			if ticker, exists := s.tickers[name]; exists {
				ticker.Stop()
				delete(s.tickers, name)
			}

			// 更新配置
			s.newsSources[i] = source

			// 如果调度器在运行且新闻源启用，创建新的定时器
			if s.running && source.Enabled {
				freq := source.UpdateFreq
				if freq == 0 {
					freq = 30 * time.Minute
				}

				ticker := time.NewTicker(freq)
				s.tickers[source.Name] = ticker

				// 启动新的调度goroutine
				go s.scheduleNewsSource(context.Background(), source, ticker)
			}

			s.logger.Info("更新新闻源配置: %s", name)
			return nil
		}
	}

	return errors.Newf(errors.ErrCodeNotFound, "未找到新闻源: %s", name)
}

// AddNewsSource 添加新闻源
func (s *NewsScheduler) AddNewsSource(source *NewsSource) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已存在
	for _, src := range s.newsSources {
		if src.Name == source.Name {
			return errors.Newf(errors.ErrCodeAlreadyExists, "新闻源已存在: %s", source.Name)
		}
	}

	// 添加新闻源
	s.newsSources = append(s.newsSources, source)

	// 如果调度器在运行且新闻源启用，创建定时器
	if s.running && source.Enabled {
		freq := source.UpdateFreq
		if freq == 0 {
			freq = 30 * time.Minute
		}

		ticker := time.NewTicker(freq)
		s.tickers[source.Name] = ticker

		// 启动调度goroutine
		go s.scheduleNewsSource(context.Background(), source, ticker)
	}

	s.logger.Info("添加新闻源: %s", source.Name)
	return nil
}

// RemoveNewsSource 移除新闻源
func (s *NewsScheduler) RemoveNewsSource(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 停止定时器
	if ticker, exists := s.tickers[name]; exists {
		ticker.Stop()
		delete(s.tickers, name)
	}

	// 从列表中移除
	for i, src := range s.newsSources {
		if src.Name == name {
			s.newsSources = append(s.newsSources[:i], s.newsSources[i+1:]...)
			s.logger.Info("移除新闻源: %s", name)
			return nil
		}
	}

	return errors.Newf(errors.ErrCodeNotFound, "未找到新闻源: %s", name)
}

// TriggerCrawl 手动触发爬取
func (s *NewsScheduler) TriggerCrawl(ctx context.Context, sourceName string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return errors.New(errors.ErrCodeInvalidOperation, "调度器未运行")
	}

	// 查找新闻源
	for _, source := range s.newsSources {
		if source.Name == sourceName {
			if !source.Enabled {
				return errors.Newf(errors.ErrCodeInvalidOperation, "新闻源 %s 已禁用", sourceName)
			}

			s.logger.Info("手动触发爬取新闻源: %s", sourceName)
			return s.crawlWithRetry(ctx, source)
		}
	}

	return errors.Newf(errors.ErrCodeNotFound, "未找到新闻源: %s", sourceName)
}

// TriggerCrawlAll 手动触发全量爬取
func (s *NewsScheduler) TriggerCrawlAll(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return errors.New(errors.ErrCodeInvalidOperation, "调度器未运行")
	}

	s.logger.Info("手动触发全量爬取")
	return s.crawlerService.CrawlMultipleSources(ctx, s.getEnabledSources())
}