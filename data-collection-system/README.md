# 数据采集子系统

数据采集子系统是量化交易平台的核心基础设施，采用Golang巨石架构，基于Gin框架构建高性能RESTful API服务。

## 功能特性

- 🚀 高性能数据采集：支持多种数据源的并发采集
- 📊 多样化数据支持：股票行情、财务数据、新闻资讯、宏观经济数据
- 🔄 智能任务调度：支持定时任务和手动触发
- 💾 高效数据存储：MySQL + Redis 双重存储架构
- 🌐 RESTful API：标准化的数据查询接口
- 📈 实时监控：系统健康检查和性能指标收集

## 技术栈

- **编程语言**: Go 1.21+
- **Web框架**: Gin v1.9+
- **数据库**: MySQL 8.0+
- **缓存**: Redis 7.0+
- **爬虫框架**: Colly v2.1+
- **配置管理**: Viper
- **日志**: Logrus

## 项目结构

```
data-collection-system/
├── cmd/
│   └── server/            # 主应用入口
├── internal/
│   ├── config/            # 配置管理
│   ├── database/          # 数据库连接
│   ├── cache/             # 缓存管理
│   ├── models/            # 数据模型
│   ├── modules/           # 功能模块
│   ├── handlers/          # HTTP处理器
│   ├── middleware/        # 中间件
│   ├── router/            # 路由配置
│   ├── bus/               # 内部消息总线
│   └── utils/             # 工具函数
├── pkg/
│   ├── logger/            # 日志组件
│   ├── validator/         # 参数验证
│   ├── errors/            # 错误处理
│   └── response/          # 响应封装
├── scripts/               # 脚本文件
├── configs/               # 配置文件
└── docs/                  # 文档
```

## 快速开始

### 环境要求

- Go 1.21+
- MySQL 8.0+
- Redis 7.0+

### 安装依赖

```bash
go mod tidy
```

### 配置环境变量

```bash
export DCS_DATABASE_PASSWORD="your_mysql_password"
export DCS_REDIS_PASSWORD="your_redis_password"
export DCS_TUSHARE_TOKEN="your_tushare_token"
```

### 启动服务

```bash
go run cmd/server/main.go
```

服务将在 `http://localhost:8080` 启动

### 健康检查

```bash
curl http://localhost:8080/health
```

## API 接口

### 数据查询

- `GET /api/v1/data/stocks` - 获取股票列表
- `GET /api/v1/data/stocks/:symbol` - 获取指定股票信息
- `GET /api/v1/data/market/:symbol` - 获取行情数据
- `GET /api/v1/data/financial/:symbol` - 获取财务数据
- `GET /api/v1/data/news` - 获取新闻数据
- `GET /api/v1/data/macro` - 获取宏观经济数据

### 任务管理

- `GET /api/v1/tasks` - 获取任务列表
- `POST /api/v1/tasks` - 创建任务
- `PUT /api/v1/tasks/:id` - 更新任务
- `DELETE /api/v1/tasks/:id` - 删除任务
- `POST /api/v1/tasks/:id/run` - 运行任务
- `GET /api/v1/tasks/:id/status` - 获取任务状态

### 系统监控

- `GET /api/v1/monitor/stats` - 获取系统统计信息
- `GET /api/v1/monitor/metrics` - 获取系统指标

## 配置说明

配置文件位于 `configs/config.yaml`，支持以下配置项：

- `server`: 服务器配置（端口、模式）
- `database`: 数据库配置
- `redis`: Redis配置
- `log`: 日志配置
- `tushare`: Tushare API配置
- `crawler`: 爬虫配置

敏感信息建议通过环境变量设置：
- `DCS_DATABASE_PASSWORD`: 数据库密码
- `DCS_REDIS_PASSWORD`: Redis密码
- `DCS_TUSHARE_TOKEN`: Tushare API Token

## 开发指南

### 编译项目

```bash
go build ./cmd/server
```

### 运行测试

```bash
go test ./...
```

### 代码格式化

```bash
go fmt ./...
```

## 部署

### Docker 部署

```bash
# 构建镜像
docker build -t data-collection-system .

# 运行容器
docker run -p 8080:8080 data-collection-system
```

### 生产环境

1. 设置环境变量
2. 配置数据库和Redis
3. 编译二进制文件
4. 启动服务

## 许可证

MIT License