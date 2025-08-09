package collector

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"data-collection-system/pkg/logger"
)

// CollectorHandler 数据采集器HTTP处理器
type CollectorHandler struct {
	service *CollectionService
}

// NewCollectorHandler 创建数据采集器处理器
func NewCollectorHandler(service *CollectionService) *CollectorHandler {
	return &CollectorHandler{
		service: service,
	}
}

// RegisterRoutes 注册路由
func (h *CollectorHandler) RegisterRoutes(router *gin.RouterGroup) {
	collector := router.Group("/collector")
	{
		// 采集器状态
		collector.GET("/status", h.GetCollectorStatus)
		
		// 任务管理
		tasks := collector.Group("/tasks")
		{
			tasks.POST("/stock-basic", h.CreateStockBasicTask)
			tasks.POST("/daily-quote", h.CreateDailyQuoteTask)
			tasks.POST("/minute-quote", h.CreateMinuteQuoteTask)
			tasks.POST("/financial", h.CreateFinancialTask)
			tasks.POST("/macro", h.CreateMacroTask)
			tasks.GET("/", h.ListTasks)
			tasks.GET("/:id", h.GetTask)
		}
		
		// 数据采集
		data := collector.Group("/data")
		{
			data.GET("/stock-basic", h.CollectStockBasic)
			data.GET("/daily-quote", h.CollectDailyQuote)
			data.GET("/minute-quote", h.CollectMinuteQuote)
			data.GET("/financial", h.CollectFinancialData)
			data.GET("/macro", h.CollectMacroData)
		}
	}
}

// GetCollectorStatus 获取采集器状态
func (h *CollectorHandler) GetCollectorStatus(c *gin.Context) {
	status := h.service.GetCollectorStatus()
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "success",
		"data": status,
	})
}

// CreateStockBasicTaskRequest 创建股票基础数据采集任务请求
type CreateStockBasicTaskRequest struct {
	Exchange    string `json:"exchange" binding:"required"`
	ScheduledAt string `json:"scheduled_at,omitempty"` // RFC3339格式
}

// CreateStockBasicTask 创建股票基础数据采集任务
func (h *CollectorHandler) CreateStockBasicTask(c *gin.Context) {
	var req CreateStockBasicTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}
	
	scheduledAt := time.Now()
	if req.ScheduledAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ScheduledAt); err == nil {
			scheduledAt = t
		}
	}
	
	task := h.service.CreateStockBasicTask(req.Exchange, scheduledAt)
	if err := h.service.SubmitTask(task); err != nil {
		logger.Error("Failed to submit stock basic task: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Failed to submit task",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "Task created successfully",
		"data": task,
	})
}

// CreateDailyQuoteTaskRequest 创建日线行情采集任务请求
type CreateDailyQuoteTaskRequest struct {
	TSCode      string `json:"ts_code" binding:"required"`
	StartDate   string `json:"start_date,omitempty"`
	EndDate     string `json:"end_date,omitempty"`
	ScheduledAt string `json:"scheduled_at,omitempty"`
}

// CreateDailyQuoteTask 创建日线行情采集任务
func (h *CollectorHandler) CreateDailyQuoteTask(c *gin.Context) {
	var req CreateDailyQuoteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}
	
	scheduledAt := time.Now()
	if req.ScheduledAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ScheduledAt); err == nil {
			scheduledAt = t
		}
	}
	
	task := h.service.CreateDailyQuoteTask(req.TSCode, req.StartDate, req.EndDate, scheduledAt)
	if err := h.service.SubmitTask(task); err != nil {
		logger.Error("Failed to submit daily quote task: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Failed to submit task",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "Task created successfully",
		"data": task,
	})
}

// CreateMinuteQuoteTaskRequest 创建分钟级行情采集任务请求
type CreateMinuteQuoteTaskRequest struct {
	TSCode      string `json:"ts_code" binding:"required"`
	Freq        string `json:"freq" binding:"required"`
	StartDate   string `json:"start_date,omitempty"`
	EndDate     string `json:"end_date,omitempty"`
	ScheduledAt string `json:"scheduled_at,omitempty"`
}

// CreateMinuteQuoteTask 创建分钟级行情采集任务
func (h *CollectorHandler) CreateMinuteQuoteTask(c *gin.Context) {
	var req CreateMinuteQuoteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}
	
	scheduledAt := time.Now()
	if req.ScheduledAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ScheduledAt); err == nil {
			scheduledAt = t
		}
	}
	
	task := h.service.CreateMinuteQuoteTask(req.TSCode, req.Freq, req.StartDate, req.EndDate, scheduledAt)
	if err := h.service.SubmitTask(task); err != nil {
		logger.Error("Failed to submit minute quote task: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Failed to submit task",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "Task created successfully",
		"data": task,
	})
}

// CreateFinancialTaskRequest 创建财务数据采集任务请求
type CreateFinancialTaskRequest struct {
	TSCode      string `json:"ts_code" binding:"required"`
	Period      string `json:"period,omitempty"`
	StartDate   string `json:"start_date,omitempty"`
	EndDate     string `json:"end_date,omitempty"`
	ScheduledAt string `json:"scheduled_at,omitempty"`
}

// CreateFinancialTask 创建财务数据采集任务
func (h *CollectorHandler) CreateFinancialTask(c *gin.Context) {
	var req CreateFinancialTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}
	
	scheduledAt := time.Now()
	if req.ScheduledAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ScheduledAt); err == nil {
			scheduledAt = t
		}
	}
	
	task := h.service.CreateFinancialTask(req.TSCode, req.Period, req.StartDate, req.EndDate, scheduledAt)
	if err := h.service.SubmitTask(task); err != nil {
		logger.Error("Failed to submit financial task: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Failed to submit task",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "Task created successfully",
		"data": task,
	})
}

// CreateMacroTaskRequest 创建宏观数据采集任务请求
type CreateMacroTaskRequest struct {
	Indicator   string `json:"indicator" binding:"required"`
	StartDate   string `json:"start_date,omitempty"`
	EndDate     string `json:"end_date,omitempty"`
	ScheduledAt string `json:"scheduled_at,omitempty"`
}

// CreateMacroTask 创建宏观数据采集任务
func (h *CollectorHandler) CreateMacroTask(c *gin.Context) {
	var req CreateMacroTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}
	
	scheduledAt := time.Now()
	if req.ScheduledAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ScheduledAt); err == nil {
			scheduledAt = t
		}
	}
	
	task := h.service.CreateMacroTask(req.Indicator, req.StartDate, req.EndDate, scheduledAt)
	if err := h.service.SubmitTask(task); err != nil {
		logger.Error("Failed to submit macro task: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Failed to submit task",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "Task created successfully",
		"data": task,
	})
}

// ListTasks 列出所有任务
func (h *CollectorHandler) ListTasks(c *gin.Context) {
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := c.Query("status")
	taskType := c.Query("type")
	
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	
	allTasks := h.service.ListTasks()
	
	// 过滤任务
	filteredTasks := make([]*CollectionTask, 0)
	for _, task := range allTasks {
		if status != "" && task.Status != status {
			continue
		}
		if taskType != "" && task.Type != taskType {
			continue
		}
		filteredTasks = append(filteredTasks, task)
	}
	
	// 分页
	total := len(filteredTasks)
	start := (page - 1) * size
	end := start + size
	
	if start >= total {
		filteredTasks = []*CollectionTask{}
	} else {
		if end > total {
			end = total
		}
		filteredTasks = filteredTasks[start:end]
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "success",
		"data": gin.H{
			"tasks": filteredTasks,
			"pagination": gin.H{
				"page":  page,
				"size":  size,
				"total": total,
			},
		},
	})
}

// GetTask 获取任务详情
func (h *CollectorHandler) GetTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"message": "Task ID is required",
		})
		return
	}
	
	task, exists := h.service.GetTask(taskID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"message": "Task not found",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "success",
		"data": task,
	})
}

// CollectStockBasic 直接采集股票基础数据
func (h *CollectorHandler) CollectStockBasic(c *gin.Context) {
	exchange := c.DefaultQuery("exchange", "SSE")
	
	collector, exists := h.service.manager.GetStockCollector("tushare")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Tushare collector not available",
		})
		return
	}
	
	data, err := collector.CollectStockBasic(c.Request.Context(), exchange)
	if err != nil {
		logger.Error("Failed to collect stock basic data: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Failed to collect data: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "success",
		"data": data,
	})
}

// CollectDailyQuote 直接采集日线行情数据
func (h *CollectorHandler) CollectDailyQuote(c *gin.Context) {
	tsCode := c.Query("ts_code")
	if tsCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"message": "ts_code is required",
		})
		return
	}
	

	
	collector, exists := h.service.manager.GetStockCollector("tushare")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Tushare collector not available",
		})
		return
	}
	
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	
	data, err := collector.CollectMarketData(c.Request.Context(), tsCode, "1d", startDate, endDate)
	if err != nil {
		logger.Error("Failed to collect daily quote data: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Failed to collect data: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "success",
		"data": data,
	})
}

// CollectMinuteQuote 直接采集分钟级行情数据
func (h *CollectorHandler) CollectMinuteQuote(c *gin.Context) {
	tsCode := c.Query("ts_code")
	if tsCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"message": "ts_code is required",
		})
		return
	}
	
	freq := c.DefaultQuery("freq", "1min")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	
	collector, exists := h.service.manager.GetStockCollector("tushare")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Tushare collector not available",
		})
		return
	}
	
	data, err := collector.CollectMarketData(c.Request.Context(), tsCode, freq, startDate, endDate)
	if err != nil {
		logger.Error("Failed to collect minute quote data: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Failed to collect data: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "success",
		"data": data,
	})
}

// CollectFinancialData 直接采集财务数据
func (h *CollectorHandler) CollectFinancialData(c *gin.Context) {
	tsCode := c.Query("ts_code")
	if tsCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"message": "ts_code is required",
		})
		return
	}
	
	period := c.Query("period")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	
	collector, exists := h.service.manager.GetFinancialCollector("tushare")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Tushare collector not available",
		})
		return
	}
	
	data, err := collector.CollectFinancialData(c.Request.Context(), tsCode, period, startDate, endDate)
	if err != nil {
		logger.Error("Failed to collect financial data: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Failed to collect data: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "success",
		"data": data,
	})
}

// CollectMacroData 直接采集宏观数据
func (h *CollectorHandler) CollectMacroData(c *gin.Context) {
	indicator := c.Query("indicator")
	if indicator == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"message": "indicator is required",
		})
		return
	}
	
	collector, exists := h.service.manager.GetMacroCollector("tushare")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Tushare collector not available",
		})
		return
	}
	
	data, err := collector.CollectMacroData(c.Request.Context(), indicator, "", "", "")
	if err != nil {
		logger.Error("Failed to collect macro data: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"message": "Failed to collect data: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "success",
		"data": data,
	})
}