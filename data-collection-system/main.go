package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"data-collection-system/api/cron"
	routes "data-collection-system/api/http"
	"data-collection-system/biz"
	mysqldao "data-collection-system/repo/mysql"
	"data-collection-system/service/collection"
	"data-collection-system/service/processing"
	"data-collection-system/pkg/config"
	"data-collection-system/pkg/database"
	"data-collection-system/pkg/logger"

	"github.com/gin-gonic/gin"

	"github.com/go-redis/redis/v8"
)

func main() {
	// 初始化配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	logger.Init(cfg.Log)

	// 初始化数据库连接
	db, err := database.Init(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			logger.Error("Failed to close database: %v", err)
		}
	}()

	// 初始化Redis连接
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 测试Redis连接
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()

	logger.Info("Redis connection established successfully")

	// 创建业务服务
	_ = mysqldao.NewRepositoryManager(db) // 暂时不使用，避免编译错误
	
	// 创建采集服务配置
	collectionConfig := &collection.Config{
		TushareToken: "", // 从配置文件获取
		TushareURL:   "http://api.tushare.pro",
		RateLimit:    200,
		RetryCount:   3,
		Timeout:      30,
		NewsCrawler:  nil, // 暂时不启用新闻爬虫
	}
	
	// 创建采集服务
	collectionService, err := collection.NewService(
		collectionConfig,
		nil, // stockRepo - 需要适配器
		nil, // marketRepo - 需要适配器
		nil, // financialRepo - 需要适配器
		nil, // macroRepo - 需要适配器
		nil, // newsRepo - 需要适配器
	)
	if err != nil {
		logger.Fatal("Failed to create collection service: %v", err)
	}
	
	// 创建处理服务
	processingService := processing.NewProcessingService(db, nil, cfg, rdb)
	
	// 创建任务执行器
	taskExecutor := biz.NewTaskExecutor(collectionService, processingService)
	
	// 创建定时任务管理器
	cronManager := cron.NewCronManager(taskExecutor)
	
	// 启动定时任务
	if err := cronManager.Start(); err != nil {
		logger.Fatal("Failed to start cron manager: %v", err)
	}
	logger.Info("Cron manager started successfully")

	// 设置Gin模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	router := routes.SetupRoutes(db, rdb)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	// 启动服务器
	go func() {
		log.Printf("Server starting on :%d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// 停止定时任务
	cronManager.Stop()
	logger.Info("Cron manager stopped")
	
	// 关闭HTTP服务器
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
