# 定时任务模块

## 概述

本模块使用 `github.com/robfig/cron/v3` 实现简单高效的定时任务调度，包含两个核心任务：

1. **股票数据采集任务** - 每天上午9点执行
2. **新闻数据采集任务** - 每小时执行一次

## 架构设计

```
api/cron/
├── cron.go          # 定时任务管理器
└── README.md        # 说明文档
```

## 核心组件

### CronManager

定时任务管理器，负责：
- 初始化 cron 调度器
- 注册定时任务
- 启动和停止调度器
- 执行具体的任务逻辑

### 任务配置

| 任务名称 | Cron表达式 | 执行频率 | 说明 |
|---------|-----------|----------|------|
| 股票数据采集 | `0 0 9 * * *` | 每天上午9点 | 采集股票价格、成交量等数据 |
| 新闻数据采集 | `0 0 * * * *` | 每小时 | 采集财经新闻并进行情感分析 |

## 使用方法

```go
// 创建任务执行器
taskExecutor := biz.NewTaskExecutor(collectionService, processingService)

// 创建定时任务管理器
cronManager := cron.NewCronManager(taskExecutor)

// 启动定时任务
if err := cronManager.Start(); err != nil {
    log.Fatal("Failed to start cron manager:", err)
}

// 停止定时任务
cronManager.Stop()
```

## 优势

1. **简单高效** - 使用成熟的第三方库，代码简洁
2. **易于维护** - 避免过度设计，降低复杂性
3. **稳定可靠** - robfig/cron 是 Go 生态中最流行的定时任务库
4. **扩展性好** - 需要新任务时，只需在 CronManager 中添加新的方法

## 扩展指南

如需添加新的定时任务：

1. 在 `CronManager` 中添加新的任务方法
2. 在 `Start()` 方法中注册新任务
3. 在 `TaskExecutor` 中添加对应的执行方法

示例：

```go
// 添加新任务方法
func (cm *CronManager) collectMacroData() {
    log.Println("开始执行宏观数据采集任务")
    // 执行逻辑
}

// 在 Start() 中注册
_, err = cm.cron.AddFunc("0 0 6 * * *", cm.collectMacroData)
```