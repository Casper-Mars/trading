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
	"data-collection-system/service/query"
	"data-collection-system/service/task"
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()

	logger.Info("Redis connection established successfully")

	// 创建数据仓库管理器
	repoManager := mysqldao.NewRepositoryManager(db, rdb)
	
	// 创建采集服务
	collectionService, err := collection.NewService(cfg, repoManager)
	if err != nil {
		logger.Fatal("Failed to create collection service: %v", err)
	}
	
	// 创建处理服务
	processingService := processing.NewProcessingService(repoManager.News(), repoManager.Stock(), cfg, rdb)
	
	// 创建查询服务
	queryService := query.NewQueryService(repoManager)
	
	// 创建任务服务
	taskService := task.NewTaskService(repoManager.DataTask(), collectionService, rdb)
	
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
	router := routes.SetupRoutes(queryService, taskService, collectionService, db, rdb)

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
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
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
