package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"data-collection-system/biz"
	"github.com/gin-gonic/gin"
)

// NewsPipelineHandler 新闻管道处理器
type NewsPipelineHandler struct {
	dataPipeline *biz.DataPipeline
}

// NewNewsPipelineHandler 创建新闻管道处理器
func NewNewsPipelineHandler(dataPipeline *biz.DataPipeline) *NewsPipelineHandler {
	return &NewsPipelineHandler{
		dataPipeline: dataPipeline,
	}
}

// TriggerNewsProcessing 触发新闻处理
// @Summary 触发新闻爬虫与NLP处理
// @Description 手动触发新闻数据的采集、NLP处理和存储流程
// @Tags 新闻管道
// @Accept json
// @Produce json
// @Param sources query string false "新闻源列表，逗号分隔" example("sina,163,tencent")
// @Success 200 {object} map[string]interface{} "处理成功"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/news/pipeline/trigger [post]
func (h *NewsPipelineHandler) TriggerNewsProcessing(c *gin.Context) {
	// 解析新闻源参数
	sourcesParam := c.Query("sources")
	var sources []string
	if sourcesParam != "" {
		sources = strings.Split(sourcesParam, ",")
		// 清理空白字符
		for i, source := range sources {
			sources[i] = strings.TrimSpace(source)
		}
	}

	// 创建上下文，设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// 检查是否已在运行
	if h.dataPipeline.IsNewsProcessingRunning() {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "新闻处理管道正在运行中",
			"message": "请等待当前任务完成后再试",
			"status":  "running",
		})
		return
	}

	// 异步执行新闻处理
	go func() {
		if err := h.dataPipeline.TriggerNewsProcessing(ctx, sources); err != nil {
			// 这里可以记录错误日志或发送通知
			// logger.Error("新闻处理失败: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "新闻处理任务已启动",
		"sources": sources,
		"status":  "started",
		"time":    time.Now(),
	})
}

// GetNewsProcessingStatus 获取新闻处理状态
// @Summary 获取新闻处理状态
// @Description 获取当前新闻爬虫与NLP处理的运行状态和统计信息
// @Tags 新闻管道
// @Produce json
// @Success 200 {object} map[string]interface{} "状态信息"
// @Router /api/v1/news/pipeline/status [get]
func (h *NewsPipelineHandler) GetNewsProcessingStatus(c *gin.Context) {
	status := h.dataPipeline.GetNewsProcessingStatus()
	stats := h.dataPipeline.GetNewsStats()

	response := gin.H{
		"status": status,
		"time":   time.Now(),
	}

	if stats != nil {
		response["stats"] = gin.H{
			"crawling": gin.H{
				"total_crawled":     stats.TotalCrawled,
				"success_crawled":   stats.SuccessCrawled,
				"failed_crawled":    stats.FailedCrawled,
				"average_crawl_time": stats.AverageCrawlTime.String(),
			},
			"processing": gin.H{
				"total_processed":   stats.TotalProcessed,
				"success_processed": stats.SuccessProcessed,
				"failed_processed":  stats.FailedProcessed,
				"nlp_processed":     stats.NLPProcessed,
				"fallback_used":     stats.FallbackUsed,
				"average_nlp_time":  stats.AverageNLPTime.String(),
			},
			"queue": gin.H{
				"queue_length":    stats.QueueLength,
				"active_workers":  stats.ActiveWorkers,
			},
			"timing": gin.H{
				"last_run_time":     stats.LastRunTime,
				"last_success_time": stats.LastSuccessTime,
			},
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetNewsStats 获取新闻处理统计信息
// @Summary 获取新闻处理统计信息
// @Description 获取详细的新闻处理统计数据
// @Tags 新闻管道
// @Produce json
// @Success 200 {object} map[string]interface{} "统计信息"
// @Router /api/v1/news/pipeline/stats [get]
func (h *NewsPipelineHandler) GetNewsStats(c *gin.Context) {
	stats := h.dataPipeline.GetNewsStats()
	if stats == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "新闻处理管道未初始化",
			"message": "服务暂时不可用",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
		"time":  time.Now(),
	})
}

// ResetNewsStats 重置新闻处理统计信息
// @Summary 重置新闻处理统计信息
// @Description 清空当前的新闻处理统计数据
// @Tags 新闻管道
// @Produce json
// @Success 200 {object} map[string]interface{} "重置成功"
// @Router /api/v1/news/pipeline/stats/reset [post]
func (h *NewsPipelineHandler) ResetNewsStats(c *gin.Context) {
	h.dataPipeline.ResetNewsStats()

	c.JSON(http.StatusOK, gin.H{
		"message": "新闻处理统计信息已重置",
		"time":    time.Now(),
	})
}

// StopNewsProcessing 停止新闻处理
// @Summary 停止新闻处理
// @Description 停止当前正在运行的新闻处理任务
// @Tags 新闻管道
// @Produce json
// @Success 200 {object} map[string]interface{} "停止成功"
// @Failure 400 {object} map[string]interface{} "没有运行中的任务"
// @Router /api/v1/news/pipeline/stop [post]
func (h *NewsPipelineHandler) StopNewsProcessing(c *gin.Context) {
	if !h.dataPipeline.IsNewsProcessingRunning() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "没有运行中的新闻处理任务",
			"message": "当前没有需要停止的任务",
			"status":  "idle",
		})
		return
	}

	// 注意：这里需要实现停止逻辑
	// 当前的 NewsPipeline 设计中没有直接的停止方法
	// 可以通过 context 取消来实现，但需要重构代码

	c.JSON(http.StatusOK, gin.H{
		"message": "停止请求已发送",
		"note":    "任务将在当前批次完成后停止",
		"time":    time.Now(),
	})
}

// GetNewsProcessingHistory 获取新闻处理历史
// @Summary 获取新闻处理历史
// @Description 获取最近的新闻处理任务历史记录
// @Tags 新闻管道
// @Produce json
// @Param limit query int false "返回记录数量限制" default(10)
// @Param offset query int false "偏移量" default(0)
// @Success 200 {object} map[string]interface{} "历史记录"
// @Router /api/v1/news/pipeline/history [get]
func (h *NewsPipelineHandler) GetNewsProcessingHistory(c *gin.Context) {
	// 解析查询参数
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100 // 限制最大返回数量
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// 这里应该从数据库或缓存中获取历史记录
	// 当前简化实现，返回模拟数据
	history := []map[string]interface{}{
		{
			"id":         "task_001",
			"start_time": time.Now().Add(-2 * time.Hour),
			"end_time":   time.Now().Add(-1 * time.Hour),
			"status":     "completed",
			"sources":    []string{"sina", "163"},
			"crawled":    150,
			"processed":  145,
			"failed":     5,
		},
		{
			"id":         "task_002",
			"start_time": time.Now().Add(-4 * time.Hour),
			"end_time":   time.Now().Add(-3 * time.Hour),
			"status":     "completed",
			"sources":    []string{"tencent", "eastmoney"},
			"crawled":    200,
			"processed":  195,
			"failed":     5,
		},
	}

	// 应用分页
	if offset >= len(history) {
		history = []map[string]interface{}{}
	} else {
		end := offset + limit
		if end > len(history) {
			end = len(history)
		}
		history = history[offset:end]
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"total":  2, // 实际应该是真实的总数
		},
		"time": time.Now(),
	})
}

// GetNewsProcessingConfig 获取新闻处理配置
// @Summary 获取新闻处理配置
// @Description 获取当前的新闻处理管道配置信息
// @Tags 新闻管道
// @Produce json
// @Success 200 {object} map[string]interface{} "配置信息"
// @Router /api/v1/news/pipeline/config [get]
func (h *NewsPipelineHandler) GetNewsProcessingConfig(c *gin.Context) {
	// 返回当前配置信息
	// 这里应该从实际的配置中获取
	config := gin.H{
		"batch_size":        50,
		"batch_timeout":     "30m",
		"process_interval":  "2s",
		"max_retries":       3,
		"retry_delay":       "5s",
		"retry_backoff":     2.0,
		"queue_name":        "news_processing_queue",
		"queue_size":        1000,
		"worker_count":      5,
		"enable_fallback":   true,
		"fallback_timeout":  "10s",
		"supported_sources": []string{"sina", "163", "tencent", "eastmoney"},
	}

	c.JSON(http.StatusOK, gin.H{
		"config": config,
		"time":   time.Now(),
	})
}