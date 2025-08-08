package router

import (
	"net/http"

	"data-collection-system/internal/handlers"
	"data-collection-system/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// Setup 设置路由
func Setup(db *gorm.DB, rdb *redis.Client) *gin.Engine {
	r := gin.New()

	// 添加中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"message": "Data collection system is running",
		})
	})

	// API版本分组
	v1 := r.Group("/api/v1")
	{
		// 数据查询相关路由
		data := v1.Group("/data")
		{
			data.GET("/stocks", handlers.GetStocks(db))
			data.GET("/stocks/:symbol", handlers.GetStockBySymbol(db))
			data.GET("/market/:symbol", handlers.GetMarketData(db, rdb))
			data.GET("/financial/:symbol", handlers.GetFinancialData(db, rdb))
			data.GET("/news", handlers.GetNews(db, rdb))
			data.GET("/macro", handlers.GetMacroData(db, rdb))
		}

		// 任务管理相关路由
		tasks := v1.Group("/tasks")
		{
			tasks.GET("/", handlers.GetTasks(db))
			tasks.POST("/", handlers.CreateTask(db))
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