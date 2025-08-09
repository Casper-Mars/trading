package routes

import (
	"net/http"

	"data-collection-system/api/http/handlers"
	"data-collection-system/api/http/middleware"
	"data-collection-system/biz"
	mysqldao "data-collection-system/repo/mysql"
	"data-collection-system/service/collection"
	"data-collection-system/service/processing"
	"data-collection-system/service/query"
	"data-collection-system/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// SetupRoutes 设置路由
func SetupRoutes(db *gorm.DB, rdb *redis.Client) *gin.Engine {
	r := gin.New()

	// 添加中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())

	// 创建Repository管理器和查询服务
	repoManager := mysqldao.NewRepositoryManager(db)
	
	// 创建查询服务
	queryService := query.NewQueryService(repoManager)
	queryHandler := handlers.NewQueryHandler(queryService)

	// 创建业务服务
	// 创建默认配置
	cfg := &config.Config{} // 使用默认配置，实际应该从配置文件加载
	
	// 创建采集服务配置
	collectionConfig := &collection.Config{
		TushareToken: "", // 从配置文件获取
		TushareURL:   "http://api.tushare.pro",
		RateLimit:    200,
		RetryCount:   3,
		Timeout:      30,
		NewsCrawler:  nil, // 暂时不启用新闻爬虫
	}
	
	// 暂时使用 nil 仓库，后续需要创建适配器来解决接口不匹配问题
	collectionService, err := collection.NewService(
		collectionConfig,
		nil, // stockRepo - 需要适配器
		nil, // marketRepo - 需要适配器
		nil, // financialRepo - 需要适配器
		nil, // macroRepo - 需要适配器
		nil, // newsRepo - 需要适配器
	)
	if err != nil {
		panic("Failed to create collection service: " + err.Error())
	}
	
	// 暂时使用 nil 新闻仓库，后续需要创建适配器来解决接口不匹配问题
	processingService := processing.NewProcessingService(db, nil, cfg, rdb)
	dataPipeline := biz.NewDataPipeline(collectionService, processingService, rdb)
	newsPipelineHandler := handlers.NewNewsPipelineHandler(dataPipeline)


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
		data.GET("/news", queryHandler.GetNews)

		// 宏观数据路由
		data.GET("/macro", queryHandler.GetMacroData)
	}



	// 新闻管道相关路由
	news := v1.Group("/news")
	news.Use(middleware.SecurityCheck()) // 添加安全检查中间件
	{
		// 新闻处理管道路由
		pipeline := news.Group("/pipeline")
		{
			pipeline.POST("/trigger", newsPipelineHandler.TriggerNewsProcessing)
			pipeline.GET("/status", newsPipelineHandler.GetNewsProcessingStatus)
			pipeline.GET("/stats", newsPipelineHandler.GetNewsStats)
			pipeline.POST("/stats/reset", newsPipelineHandler.ResetNewsStats)
			pipeline.POST("/stop", newsPipelineHandler.StopNewsProcessing)
			pipeline.GET("/history", newsPipelineHandler.GetNewsProcessingHistory)
			pipeline.GET("/config", newsPipelineHandler.GetNewsProcessingConfig)
		}
		}

		// 任务管理相关路由
		tasks := v1.Group("/tasks")
		tasks.Use(middleware.SecurityCheck()) // 添加安全检查中间件
		{
			tasks.GET("/", handlers.GetTasks(db))
			tasks.POST("/", middleware.ValidateJSON(struct {
				Name        string                 `json:"name" binding:"required"`
				Type        string                 `json:"type" binding:"required"`
				Description string                 `json:"description"`
				Config      map[string]interface{} `json:"config"`
				Schedule    string                 `json:"schedule"`
			}{}), handlers.CreateTask(db))
			tasks.PUT("/:id", handlers.UpdateTask(db))
			tasks.DELETE("/:id", handlers.DeleteTask(db))
			tasks.POST("/:id/run", handlers.RunTask(db))
			tasks.GET("/:id/status", handlers.GetTaskStatus(db))
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
