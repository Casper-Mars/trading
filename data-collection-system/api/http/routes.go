package routes

import (
	"net/http"

	"data-collection-system/api/http/handlers"
	"data-collection-system/api/http/middleware"
	"data-collection-system/service/query"
	"data-collection-system/service/task"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// SetupRoutes 设置路由
func SetupRoutes(queryService *query.QueryService, taskService *task.TaskService, db *gorm.DB, rdb *redis.Client) *gin.Engine {
	r := gin.New()

	// 添加中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())

	// 创建处理器
	queryHandler := handlers.NewQueryHandler(queryService)
	taskHandler := handlers.NewTaskHandler(taskService)


	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Data collection system is running",
		})
	})

	// API版本分组
	v1 := r.Group("/api/v1")
	{
		// 数据查询相关路由
		data := v1.Group("/data")
		data.Use(middleware.ValidateParams()) // 添加参数验证中间件
		{
			// 股票数据路由
			stocks := data.Group("/stocks")
			{
				stocks.GET("/", queryHandler.GetStocks)
				stocks.GET("/:symbol", middleware.ValidateStockSymbol(), queryHandler.GetStockBySymbol)
			}

			// 行情数据路由
			market := data.Group("/market")
			{
				market.GET("/", queryHandler.GetMarketData)
				market.GET("/:symbol/latest", middleware.ValidateStockSymbol(), queryHandler.GetLatestMarketData)
			}

			// 财务数据路由
			financial := data.Group("/financial")
			{
				financial.GET("/", queryHandler.GetFinancialData)
				financial.GET("/:symbol/latest", middleware.ValidateStockSymbol(), queryHandler.GetLatestFinancialData)
			}

			// 新闻数据路由
			news := data.Group("/news")
			news.Use(middleware.ContentFilter()) // 添加内容过滤中间件
			{
				news.GET("/", queryHandler.GetNews)
				news.GET("/:id", queryHandler.GetNewsDetail)
				news.GET("/hot", queryHandler.GetHotNews)
				news.GET("/latest", queryHandler.GetLatestNews)
			}

			// 宏观数据路由
			data.GET("/macro", queryHandler.GetMacroData)
	}



	// 新闻管道相关路由
	news := v1.Group("/news")
	news.Use(middleware.SecurityCheck()) // 添加安全检查中间件
	{
		// 新闻数据管道 - 暂时移除，等待重构
		// pipeline := v1.Group("/pipeline")
		// {
		//	pipeline.POST("/trigger", newsPipelineHandler.TriggerNewsProcessing)
		//	pipeline.GET("/status", newsPipelineHandler.GetNewsProcessingStatus)
		//	pipeline.GET("/stats", newsPipelineHandler.GetNewsStats)
		//	pipeline.POST("/stats/reset", newsPipelineHandler.ResetNewsStats)
		//	pipeline.POST("/stop", newsPipelineHandler.StopNewsProcessing)
		//	pipeline.GET("/history", newsPipelineHandler.GetNewsProcessingHistory)
		//	pipeline.GET("/config", newsPipelineHandler.GetNewsProcessingConfig)
		// }
		}

		// 任务管理相关路由
		tasks := v1.Group("/tasks")
		tasks.Use(middleware.SecurityCheck()) // 添加安全检查中间件
		{
			tasks.GET("/", taskHandler.GetTasks)
			tasks.POST("/", middleware.ValidateJSON(struct {
				Name        string                 `json:"name" binding:"required"`
				Type        string                 `json:"type" binding:"required"`
				Description string                 `json:"description"`
				Config      map[string]interface{} `json:"config"`
				Schedule    string                 `json:"schedule"`
			}{}), taskHandler.CreateTask)
			tasks.PUT("/:id", taskHandler.UpdateTask)
			tasks.DELETE("/:id", taskHandler.DeleteTask)
			tasks.POST("/:id/run", taskHandler.RunTask)
			tasks.GET("/:id/status", taskHandler.GetTaskStatus)
		}

		// 系统监控相关路由
		monitor := v1.Group("/monitor")
		{
			monitor.GET("/stats", handlers.GetSystemStats(db, rdb))
			monitor.GET("/metrics", handlers.GetMetrics(db, rdb))
		}


	}

	return r
}
