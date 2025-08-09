package handlers

import (
	"context"
	"net/http"
	"strconv"

	"data-collection-system/pkg/response"
	"data-collection-system/service/query"
	"data-collection-system/service/task"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// QueryHandler 查询处理器
type QueryHandler struct {
	queryService *query.QueryService
}

// NewQueryHandler 创建查询处理器
func NewQueryHandler(queryService *query.QueryService) *QueryHandler {
	return &QueryHandler{
		queryService: queryService,
	}
}

// TaskHandler 任务处理器
type TaskHandler struct {
	taskService *task.TaskService
}

// NewTaskHandler 创建任务处理器
func NewTaskHandler(taskService *task.TaskService) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
	}
}

// GetStocks 获取股票列表
func (h *QueryHandler) GetStocks(c *gin.Context) {
	ctx := context.Background()

	// 绑定查询参数
	var params query.StockQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	// 设置默认分页参数
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	// 查询股票数据
	result, err := h.queryService.GetStocks(ctx, &params)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 构建分页信息
	pagination := response.NewPagination(params.Page, params.PageSize, result.Total)

	// 返回分页响应
	response.SuccessPage(c, result.Data, pagination)
}

// GetStockBySymbol 根据股票代码获取股票信息
func (h *QueryHandler) GetStockBySymbol(c *gin.Context) {
	ctx := context.Background()
	symbol := c.Param("symbol")

	// 查询股票信息
	stock, err := h.queryService.GetStockBySymbol(ctx, symbol)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, stock)
}

// GetMarketData 获取行情数据
func (h *QueryHandler) GetMarketData(c *gin.Context) {
	ctx := context.Background()

	// 绑定查询参数
	var params query.MarketDataQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	// 设置默认分页参数
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 100
	}

	// 查询行情数据
	result, err := h.queryService.GetMarketData(ctx, &params)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 构建分页信息
	pagination := response.NewPagination(params.Page, params.PageSize, result.Total)

	// 返回分页响应
	response.SuccessPage(c, result.Data, pagination)
}

// GetLatestMarketData 获取最新行情数据
func (h *QueryHandler) GetLatestMarketData(c *gin.Context) {
	ctx := context.Background()
	symbol := c.Param("symbol")
	period := c.DefaultQuery("period", "daily")

	// 查询最新行情数据
	data, err := h.queryService.GetLatestMarketData(ctx, symbol, period)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, data)
}

// GetFinancialData 获取财务数据
func (h *QueryHandler) GetFinancialData(c *gin.Context) {
	ctx := context.Background()

	// 绑定查询参数
	var params query.FinancialDataQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	// 设置默认分页参数
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	// 查询财务数据
	result, err := h.queryService.GetFinancialData(ctx, &params)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 构建分页信息
	pagination := response.NewPagination(params.Page, params.PageSize, result.Total)

	// 返回分页响应
	response.SuccessPage(c, result.Data, pagination)
}

// GetLatestFinancialData 获取最新财务数据
func (h *QueryHandler) GetLatestFinancialData(c *gin.Context) {
	ctx := context.Background()
	symbol := c.Param("symbol")
	reportType := c.DefaultQuery("report_type", "annual")

	// 查询最新财务数据
	data, err := h.queryService.GetLatestFinancialData(ctx, symbol, reportType)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, data)
}

// GetNews 获取新闻数据
func (h *QueryHandler) GetNews(c *gin.Context) {
	ctx := context.Background()

	// 绑定查询参数
	var params query.NewsDataQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	// 设置默认分页参数
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	// 查询新闻数据
	result, err := h.queryService.GetNewsData(ctx, &params)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 构建分页信息
	pagination := response.NewPagination(params.Page, params.PageSize, result.Total)

	// 返回分页响应
	response.SuccessPage(c, result.Data, pagination)
}

// GetNewsDetail 获取新闻详情
func (h *QueryHandler) GetNewsDetail(c *gin.Context) {
	ctx := context.Background()

	// 获取新闻ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "新闻ID格式错误")
		return
	}

	// 查询新闻详情
	news, err := h.queryService.GetNewsDetail(ctx, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	if news == nil {
		response.NotFound(c, "新闻不存在")
		return
	}

	// 返回新闻详情
	response.Success(c, news)
}

// GetHotNews 获取热门新闻
func (h *QueryHandler) GetHotNews(c *gin.Context) {
	ctx := context.Background()

	// 获取查询参数
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 10
	}

	hoursStr := c.DefaultQuery("hours", "24")
	hours, err := strconv.Atoi(hoursStr)
	if err != nil || hours <= 0 || hours > 168 { // 最多7天
		hours = 24
	}

	// 查询热门新闻
	newsList, err := h.queryService.GetHotNews(ctx, limit, hours)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 返回热门新闻列表
	response.Success(c, gin.H{
		"news": newsList,
		"limit": limit,
		"hours": hours,
	})
}

// GetLatestNews 获取最新新闻
func (h *QueryHandler) GetLatestNews(c *gin.Context) {
	ctx := context.Background()

	// 获取查询参数
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	// 查询最新新闻
	newsList, err := h.queryService.GetLatestNews(ctx, limit)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 返回最新新闻列表
	response.Success(c, gin.H{
		"news": newsList,
		"limit": limit,
	})
}

// GetMacroData 获取宏观经济数据
func (h *QueryHandler) GetMacroData(c *gin.Context) {
	ctx := context.Background()

	// 绑定查询参数
	var params query.MacroDataQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	// 设置默认分页参数
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 50
	}

	// 查询宏观数据
	result, err := h.queryService.GetMacroData(ctx, &params)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 构建分页信息
	pagination := response.NewPagination(params.Page, params.PageSize, result.Total)

	// 返回分页响应
	response.SuccessPage(c, result.Data, pagination)
}

// GetTasks 获取任务列表
func (h *TaskHandler) GetTasks(c *gin.Context) {
	ctx := context.Background()
	
	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 参数验证
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	// 创建查询参数
	queryParams := &task.TaskQueryParams{
		Page:     page,
		PageSize: pageSize,
	}

	// 获取任务列表
	result, err := h.taskService.GetTasksWithParams(ctx, queryParams)
	if err != nil {
		response.Error(c, err)
		return
	}

	pagination := response.NewPagination(page, pageSize, result.Total)
	response.SuccessPage(c, result.Data, pagination)
}

// CreateTask 创建任务
func (h *TaskHandler) CreateTask(c *gin.Context) {
	ctx := context.Background()
	
	// 绑定请求体
	var req struct {
		Name        string                 `json:"name" binding:"required"`
		Type        string                 `json:"type" binding:"required"`
		Description string                 `json:"description"`
		Config      map[string]interface{} `json:"config"`
		Schedule    string                 `json:"schedule"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数格式错误")
		return
	}

	// 创建任务请求
	createReq := &task.CreateTaskRequest{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
		Config:      req.Config,
		Schedule:    req.Schedule,
	}

	// 创建任务
	taskResult, err := h.taskService.CreateTask(ctx, createReq)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "任务创建成功",
		"task":    taskResult,
	})
}

// UpdateTask 更新任务
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	ctx := context.Background()
	
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的任务ID")
		return
	}

	// 绑定请求体
	var req struct {
		Name        string                 `json:"name"`
		Type        string                 `json:"type"`
		Description string                 `json:"description"`
		Config      map[string]interface{} `json:"config"`
		Schedule    string                 `json:"schedule"`
		Status      *int8                  `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数格式错误")
		return
	}

	// 创建更新请求
	updateReq := &task.UpdateTaskRequest{
		Config: req.Config,
		Status: req.Status,
	}
	if req.Name != "" {
		updateReq.Name = &req.Name
	}
	if req.Type != "" {
		updateReq.Type = &req.Type
	}
	if req.Description != "" {
		updateReq.Description = &req.Description
	}
	if req.Schedule != "" {
		updateReq.Schedule = &req.Schedule
	}

	// 更新任务
	_, err = h.taskService.UpdateTask(ctx, taskID, updateReq)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "任务更新成功",
	})
}

// DeleteTask 删除任务
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	ctx := context.Background()
	
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的任务ID")
		return
	}

	// 删除任务
	err = h.taskService.DeleteTask(ctx, taskID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "任务删除成功",
	})
}

// RunTask 运行任务
func (h *TaskHandler) RunTask(c *gin.Context) {
	ctx := context.Background()
	
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的任务ID")
		return
	}

	// 运行任务
	err = h.taskService.RunTask(ctx, taskID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"message": "任务已启动",
	})
}

// GetTaskStatus 获取任务状态
func (h *TaskHandler) GetTaskStatus(c *gin.Context) {
	ctx := context.Background()
	
	id := c.Param("id")
	taskID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的任务ID")
		return
	}

	// 获取任务状态
	status, err := h.taskService.GetTaskStatus(ctx, taskID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"task_id": taskID,
		"status":  status,
	})
}

// GetSystemStats 获取系统统计信息
func GetSystemStats(db *gorm.DB, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现获取系统统计信息逻辑
		c.JSON(http.StatusOK, gin.H{
			"message": "Get system stats endpoint",
			"stats":   map[string]interface{}{},
		})
	}
}

// GetMetrics 获取系统指标
func GetMetrics(db *gorm.DB, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现获取系统指标逻辑
		c.JSON(http.StatusOK, gin.H{
			"message": "Get metrics endpoint",
			"metrics": map[string]interface{}{},
		})
	}
}
