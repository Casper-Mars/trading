package cron

import (
	"context"
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"data-collection-system/biz"
)

// CronManager 定时任务管理器
type CronManager struct {
	cron         *cron.Cron
	taskExecutor *biz.TaskExecutor
}

// NewCronManager 创建定时任务管理器
func NewCronManager(taskExecutor *biz.TaskExecutor) *CronManager {
	return &CronManager{
		cron:         cron.New(cron.WithSeconds()),
		taskExecutor: taskExecutor,
	}
}

// Start 启动定时任务
func (cm *CronManager) Start() error {
	// 注册股票数据采集任务 - 每天上午9点执行
	_, err := cm.cron.AddFunc("0 0 9 * * *", cm.collectStockData)
	if err != nil {
		return err
	}

	// 注册新闻数据采集任务 - 每小时执行一次
	_, err = cm.cron.AddFunc("0 0 * * * *", cm.collectNewsData)
	if err != nil {
		return err
	}

	cm.cron.Start()
	log.Println("定时任务已启动")
	return nil
}

// Stop 停止定时任务
func (cm *CronManager) Stop() {
	cm.cron.Stop()
	log.Println("定时任务已停止")
}

// collectStockData 股票数据采集任务
func (cm *CronManager) collectStockData() {
	log.Println("开始执行股票数据采集任务")
	
	ctx := context.Background()
	config := map[string]interface{}{
		"task_type": "stock_collection",
		"timestamp": time.Now(),
	}

	err := cm.taskExecutor.ExecuteStockDataCollection(ctx, config)
	if err != nil {
		log.Printf("股票数据采集任务执行失败: %v", err)
	} else {
		log.Println("股票数据采集任务执行成功")
	}
}

// collectNewsData 新闻数据采集任务
func (cm *CronManager) collectNewsData() {
	log.Println("开始执行新闻数据采集任务")
	
	ctx := context.Background()
	config := map[string]interface{}{
		"task_type": "news_collection",
		"timestamp": time.Now(),
	}

	err := cm.taskExecutor.ExecuteNewsDataCollection(ctx, config)
	if err != nil {
		log.Printf("新闻数据采集任务执行失败: %v", err)
	} else {
		log.Println("新闻数据采集任务执行成功")
	}
}