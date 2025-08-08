# Redis缓存系统使用指南

## 概述

本模块提供了完整的Redis缓存解决方案，包括连接池管理、超时配置、缓存键命名规范和TTL策略。

## 功能特性

### 1. 增强的连接池配置
- 可配置的连接池大小和最小空闲连接数
- 支持连接重试和超时控制
- 自动空闲连接清理

### 2. 标准化键命名规范
- 统一的键前缀管理
- 层次化的键结构：`prefix:module:identifier[:subkey]`
- 预定义的业务域键前缀

### 3. 智能TTL策略
- 根据数据类型自动选择合适的过期时间
- 支持自定义TTL设置
- 提供多种预定义的TTL常量

### 4. 丰富的缓存操作
- JSON序列化支持
- 批量操作（MSet/MGet）
- 模式匹配删除
- 缓存状态检查

## 配置说明

### config.yaml配置

```yaml
redis:
  host: localhost
  port: 6379
  password: ""  # 建议通过环境变量设置
  db: 0
  # 连接池配置
  pool_size: 20          # 连接池大小
  min_idle_conns: 5      # 最小空闲连接数
  max_retries: 3         # 最大重试次数
  # 超时配置（秒）
  dial_timeout: 10       # 连接超时
  read_timeout: 5        # 读取超时
  write_timeout: 5       # 写入超时
  pool_timeout: 10       # 连接池超时
  idle_timeout: 300      # 空闲连接超时（5分钟）
  idle_check_frequency: 60  # 空闲连接检查频率（1分钟）
```

### 环境变量

```bash
export DCS_REDIS_PASSWORD="your_redis_password"
```

## 使用方法

### 1. 初始化Redis连接

```go
package main

import (
    "data-collection-system/internal/cache"
    "data-collection-system/internal/config"
)

func main() {
    // 加载配置
    cfg, err := config.Load()
    if err != nil {
        panic(err)
    }
    
    // 初始化Redis连接
    _, err = cache.Init(cfg.Redis)
    if err != nil {
        panic(err)
    }
    defer cache.Close()
    
    // 创建缓存管理器
    cm := cache.NewCacheManager()
    
    // 使用缓存管理器...
}
```

### 2. 基础缓存操作

```go
ctx := context.Background()
cm := cache.NewCacheManager()

// 设置缓存
data := map[string]interface{}{
    "code": "000001",
    "price": 12.50,
}
key := cache.BuildStockKey("000001", "realtime")
err := cm.SetJSON(ctx, key, data, cache.TTLShort)

// 获取缓存
var result map[string]interface{}
err = cm.GetJSON(ctx, key, &result)

// 检查缓存是否存在
exists, err := cm.IsExists(ctx, key)
```

### 3. 智能TTL设置

```go
// 根据数据类型自动选择TTL
err := cm.SetWithTTL(ctx, key, data, "realtime")  // 5分钟
err := cm.SetWithTTL(ctx, key, data, "news")      // 30分钟
err := cm.SetWithTTL(ctx, key, data, "daily")     // 24小时
```

### 4. 批量操作

```go
// 批量设置
batchData := map[string]interface{}{
    cache.BuildStockKey("000001", "realtime"): stockData1,
    cache.BuildStockKey("000002", "realtime"): stockData2,
}
err := cm.MSet(ctx, batchData)

// 批量获取
keys := []string{
    cache.BuildStockKey("000001", "realtime"),
    cache.BuildStockKey("000002", "realtime"),
}
results, err := cm.MGet(ctx, keys)
```

### 5. 模式删除

```go
// 删除所有股票实时数据
pattern := cache.BuildKey(cache.KeyPrefixStock, "realtime", "*")
err := cm.DeleteByPattern(ctx, pattern)
```

## 键命名规范

### 预定义前缀

- `stock`: 股票数据
- `news`: 新闻数据
- `sentiment`: 情感分析数据
- `indicator`: 技术指标数据
- `user`: 用户数据
- `session`: 会话数据

### 键构建函数

```go
// 通用键构建
key := cache.BuildKey("prefix", "module", "identifier", "subkey")

// 专用键构建函数
stockKey := cache.BuildStockKey("000001", "realtime")        // stock:realtime:000001
newsKey := cache.BuildNewsKey("news_001", "content")        // news:content:news_001
sentimentKey := cache.BuildSentimentKey("src_001", "score") // sentiment:score:src_001
indicatorKey := cache.BuildIndicatorKey("000001", "ma")     // indicator:ma:000001
userKey := cache.BuildUserKey("user_001", "profile")       // user:profile:user_001
sessionKey := cache.BuildSessionKey("sess_001")             // session:data:sess_001
```

## TTL策略

### 预定义TTL常量

```go
const (
    TTLShort   = 5 * time.Minute   // 短期缓存：5分钟
    TTLMedium  = 30 * time.Minute  // 中期缓存：30分钟
    TTLLong    = 2 * time.Hour     // 长期缓存：2小时
    TTLDaily   = 24 * time.Hour    // 日缓存：24小时
    TTLWeekly  = 7 * 24 * time.Hour // 周缓存：7天
    TTLSession = 30 * time.Minute  // 会话缓存：30分钟
)
```

### 数据类型与TTL映射

| 数据类型 | TTL | 说明 |
|---------|-----|------|
| realtime, quote | 5分钟 | 实时数据 |
| news, sentiment | 30分钟 | 新闻和情感数据 |
| indicator, analysis | 2小时 | 技术指标 |
| daily, historical | 24小时 | 历史数据 |
| weekly, report | 7天 | 周报数据 |
| session | 30分钟 | 会话数据 |

## 最佳实践

### 1. 键命名
- 使用预定义的键构建函数
- 保持键名简洁且有意义
- 避免使用特殊字符

### 2. TTL设置
- 根据数据更新频率选择合适的TTL
- 使用`SetWithTTL`方法自动选择TTL
- 定期检查和调整TTL策略

### 3. 性能优化
- 使用批量操作减少网络往返
- 合理设置连接池大小
- 监控缓存命中率

### 4. 错误处理
- 始终检查缓存操作的错误
- 实现缓存降级策略
- 记录缓存操作日志

### 5. 监控和维护
- 定期清理过期缓存
- 监控Redis内存使用
- 设置合适的内存淘汰策略

## 示例代码

完整的使用示例请参考 `example.go` 文件。

## 故障排除

### 常见问题

1. **连接超时**
   - 检查Redis服务是否运行
   - 验证网络连接
   - 调整超时配置

2. **内存不足**
   - 检查Redis内存使用
   - 调整TTL策略
   - 设置内存淘汰策略

3. **性能问题**
   - 监控连接池使用情况
   - 优化键命名和数据结构
   - 使用批量操作

### 日志监控

系统会记录以下关键事件：
- Redis连接建立和断开
- 缓存操作错误
- 连接池状态变化

通过日志可以及时发现和解决问题。