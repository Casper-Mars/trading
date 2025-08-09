# 数据采集子系统

## 项目结构

采用简洁清晰的三层架构，遵循Go项目标准布局：

```
data-collection-system/
├── main.go                # 应用入口
├── config.yaml           # 配置文件
├── docker-compose.yml    # Docker配置
├── go.mod                 # Go模块文件
├── go.sum                 # Go依赖锁定
├── README.md              # 项目说明
│
├── api/                   # 接入层 (API Layer)
│   ├── http/              # HTTP接口
│   │   ├── handlers/      # HTTP处理器
│   │   ├── middleware/    # 中间件
│   │   └── routes.go      # 路由配置
│   ├── cron/              # 定时任务
│   │   ├── scheduler.go   # 任务调度器
│   │   ├── jobs/          # 定时任务定义
│   │   └── manager.go     # 任务管理器
│   └── grpc/              # gRPC接口(预留)
│
├── biz/                   # 业务编排层 (Business Layer)
│   └── collection.go      # 数据采集业务
├── service/               # 服务层 (Service Layer)
│   ├── collection/        # 数据采集服务
│   │   ├── service.go     # 采集服务
│   │   ├── tushare.go     # Tushare数据采集
│   │   ├── news.go        # 新闻数据采集
│   │   └── validator.go   # 数据验证器
│   ├── processing/        # 数据处理服务
│   │   ├── service.go     # 加工业务服务
│   │   ├── cleaner.go     # 数据清洗
│   │   ├── nlp.go         # NLP处理
│   │   └── enricher.go    # 数据增强
│   ├── query/             # 数据查询服务
│   │   ├── service.go     # 查询业务服务
│   │   ├── stock.go       # 股票数据查询
│   │   ├── news.go        # 新闻数据查询
│   │   └── aggregator.go  # 数据聚合
│   ├── task/              # 任务管理服务
│       ├── service.go     # 任务业务服务
│       ├── scheduler.go   # 任务调度
│       └── monitor.go     # 任务监控
│
├── repo/                  # 数据仓库层 (Repository Layer)
│   ├── mysql/             # MySQL数据访问
│   │   ├── stock.go       # 股票数据仓库
│   │   ├── news.go        # 新闻数据仓库
│   │   ├── market.go      # 市场数据仓库
│   │   ├── financial.go   # 财务数据仓库
│   │   └── task.go        # 任务数据仓库
│   ├── redis/             # Redis缓存访问
│   │   ├── cache.go       # 缓存操作
│   │   └── session.go     # 会话管理
│   └── external/          # 外部数据源
│       ├── tushare/       # Tushare API客户端
│       ├── crawler/       # 网页爬虫
│       └── nlp/           # NLP服务客户端
│
├── model/                 # 数据模型定义
│   ├── stock.go           # 股票模型
│   ├── news.go            # 新闻模型
│   ├── market.go          # 市场数据模型
│   ├── financial.go       # 财务数据模型
│   └── task.go            # 任务模型
│
├── pkg/                   # 公共包 (Shared Package)
│   ├── config/            # 配置管理
│   ├── logger/            # 日志组件
│   ├── errors/            # 错误定义
│   ├── utils/             # 工具函数
│   ├── types/             # 公共类型
│   └── constants/         # 常量定义
│
└── scripts/               # 脚本文件
    ├── migrate.sql        # 数据库迁移
    └── deploy.sh          # 部署脚本
```

#### 目录说明

**简洁架构分层设计**

* **api/**：接口层，负责外部接口适配，包括HTTP接口和定时任务
* **biz/**：业务逻辑层，负责业务流程编排和跨领域协调
* **module/**：模块层，按业务领域划分的核心业务服务
* **repo/**：数据仓库层，负责数据持久化和外部数据源访问
* **model/**：数据模型层，定义核心业务实体和数据结构
* **pkg/**：公共组件层，提供通用的技术组件和工具
* **scripts/**：脚本文件，包含部署和数据库迁移脚本

**各层职责**

1. **接口层 (api/)**
   - **http/**: HTTP接口适配和路由配置，包含handlers、middleware和routes
   - **cron/**: 定时任务调度，包含scheduler、jobs和manager
   - 请求参数验证和响应格式化
   - 中间件处理（认证、限流、日志等）

2. **业务逻辑层 (biz/)**
   - 业务流程编排和跨领域协调
   - 复杂业务场景的工作流管理
   - 任务执行编排，组合多个service完成具体业务
   - 跨服务的事务管理和数据一致性保证

3. **服务层 (service/)**
   - **collection/**: 数据采集服务，包含数据采集服务、Tushare采集、新闻采集等
   - **processing/**: 数据处理服务，包含数据清洗、NLP处理、数据增强等
   - **query/**: 数据查询服务，包含股票查询、新闻查询、数据聚合等
   - **task/**: 任务管理服务，只负责任务的CRUD操作，不包含业务执行逻辑

4. **数据仓库层 (repo/)**
   - **mysql/**: MySQL数据访问，包含各业务实体的DAO实现
   - **redis/**: Redis缓存访问，提供缓存操作和会话管理
   - **external/**: 外部数据源访问，包含Tushare、爬虫、NLP等客户端
   - 数据持久化实现
   - 外部服务集成

5. **数据模型层 (model/)**
   - 定义核心业务实体：股票、新闻、市场数据、财务数据、任务等
   - 数据传输对象(DTO)和值对象(VO)
   - 业务规则和约束定义

6. **公共组件层 (pkg/)**
   - **config/**: 配置管理组件
   - **logger/**: 日志组件
   - **errors/**: 错误处理组件
   - **response/**: 统一响应格式组件
   - **validator/**: 参数验证组件
   - **utils/**: 通用工具函数