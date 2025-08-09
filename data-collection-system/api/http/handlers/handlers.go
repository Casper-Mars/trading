package handlers

import (
	"context"
	"net/http"
	"strconv"

	"data-collection-system/pkg/response"
	"data-collection-system/service/query"

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
func GetTasks(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取查询参数
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
		status := c.Query("status")
		taskType := c.Query("type")

		// 参数验证
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 || pageSize > 100 {
			pageSize = 20
		}

		// TODO: 实现获取任务列表逻辑
		// 这里应该调用任务服务来获取任务列表
		_ = status
		_ = taskType
		pagination := response.NewPagination(page, pageSize, 0)
		response.SuccessPage(c, []interface{}{}, pagination)
	}
}

// CreateTask 创建任务
func CreateTask(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		// TODO: 实现创建任务逻辑
		// 这里应该调用任务服务来创建任务
		response.Success(c, gin.H{
			"message": "任务创建成功",
			"task_id": "temp_task_id",
		})
	}
}

// UpdateTask 更新任务
func UpdateTask(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		// TODO: 实现更新任务逻辑
		c.JSON(http.StatusOK, gin.H{
			"message": "Update task endpoint",
			"id":      id,
		})
	}
}

// DeleteTask 删除任务
func DeleteTask(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		// TODO: 实现删除任务逻辑
		c.JSON(http.StatusOK, gin.H{
			"message": "Delete task endpoint",
			"id":      id,
		})
	}
}

// RunTask 运行任务
func RunTask(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		// TODO: 实现运行任务逻辑
		c.JSON(http.StatusOK, gin.H{
			"message": "Run task endpoint",
			"id":      id,
		})
	}
}

// GetTaskStatus 获取任务状态
func GetTaskStatus(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		// TODO: 实现获取任务状态逻辑
		c.JSON(http.StatusOK, gin.H{
			"message": "Get task status endpoint",
			"id":      id,
			"status":  "unknown",
		})
	}
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
