# 数据采集子系统

## 项目结构

采用现代化Python项目结构，遵循分层架构和领域驱动设计原则：

```
quantitative-system/
├── main.py                    # 应用入口
├── pyproject.toml            # 项目配置和依赖管理
├── uv.lock                   # 依赖锁定文件
├── .env.example              # 环境变量示例
├── .gitignore                # Git忽略文件
├── README.md                 # 项目说明文档
├── Makefile                  # 构建和开发命令
├── mypy.ini                  # 类型检查配置
├── .pre-commit-config.yaml   # 代码质量检查配置
│
├── config/                   # 配置管理层
│   ├── __init__.py
│   ├── settings.py          # 应用配置
│   ├── database.py          # 数据库配置
│   └── logging.py           # 日志配置
│
├── api/                      # API接口层
│   ├── __init__.py
│   ├── routes/              # 路由定义
│   ├── dependencies.py     # 依赖注入
│   └── middleware.py        # 中间件
│
├── scheduler/                # 调度层
│   ├── __init__.py
│   ├── jobs.py              # 任务定义
│   ├── scheduler.py         # 调度器
│   └── manager.py           # 任务管理器
│
├── biz/                      # 业务编排层
│   ├── __init__.py
│   ├── data_collection_orchestrator.py  # 数据采集编排器
│   ├── nlp_processing_orchestrator.py   # NLP处理编排器
│   ├── quality_control_orchestrator.py  # 质量控制编排器
│   └── base_orchestrator.py             # 编排器基类
│
├── services/                 # 业务服务层
│   ├── __init__.py
│   ├── collection_service.py    # 数据采集服务
│   ├── nlp_service.py          # NLP处理服务
│   ├── quality_service.py      # 数据质量服务
│   ├── query_service.py        # 查询服务
│   ├── config_service.py       # 配置管理服务
│   └── task_service.py         # 任务管理服务
│
├── models/                   # 数据模型层
│   ├── __init__.py
│   ├── database.py          # SQLAlchemy数据库模型
│   ├── schemas.py           # Pydantic请求/响应模型
│   ├── enums.py             # 枚举定义
│   └── types.py             # 自定义类型
│
├── repositories/             # 数据访问层
│   ├── __init__.py
│   ├── stock_repo.py        # 股票数据仓库
│   ├── news_repo.py         # 新闻数据仓库
│   ├── task_repo.py         # 任务数据仓库
│   ├── cache_repo.py        # 缓存数据仓库
│   └── base_repo.py         # 仓库基类
│
├── clients/                  # 外部客户端层
│   ├── __init__.py
│   ├── tushare_client.py    # Tushare API客户端
│   └── news_crawler.py      # 新闻爬虫客户端
│
├── strategies/               # 策略模块（预留）
│   ├── __init__.py
│   └── base_strategy.py     # 策略基类
│
├── utils/                    # 工具模块
│   ├── __init__.py
│   ├── logger.py            # 日志工具
│   ├── exceptions.py        # 异常定义
│   ├── validators.py        # 数据验证
│   ├── helpers.py           # 辅助函数
│   └── constants.py         # 常量定义
│
├── tests/                    # 测试模块
│   ├── __init__.py
│   ├── unit/                # 单元测试
│   ├── integration/         # 集成测试
│   └── fixtures/            # 测试数据
│
├── scripts/                  # 脚本文件
│   ├── migrate.py           # 数据库迁移
│   ├── init_data.py         # 初始化数据
│   └── deploy.sh            # 部署脚本
│
├── docs/                     # 文档目录
│   └── api.md               # API文档
│

```

## 架构分层说明

**分层设计原则**：
- **单一职责**：每层只负责特定的业务逻辑
- **依赖倒置**：上层依赖下层的抽象接口，而非具体实现
- **松耦合**：层与层之间通过接口交互，降低耦合度
- **高内聚**：同一层内的模块功能相关性强

**各层职责**：

1. **API接口层 (api/)**
   - HTTP接口适配和路由配置
   - 请求参数验证和响应格式化
   - 中间件处理（认证、限流、日志等）
   - 依赖注入管理

2. **调度层 (scheduler/)**
   - 定时任务调度和管理
   - 任务执行状态监控
   - 任务失败重试机制
   - 任务优先级管理

3. **业务编排层 (biz/)**
   - 复杂业务流程编排
   - 跨服务协调和事务管理
   - 业务规则执行
   - 工作流管理

4. **业务服务层 (services/)**
   - 核心业务逻辑实现
   - 单一业务领域服务
   - 业务规则验证
   - 服务间协作

5. **数据访问层 (repositories/)**
   - 数据持久化操作
   - 数据查询和缓存
   - 数据访问抽象
   - 事务管理

6. **外部客户端层 (clients/)**
   - 第三方API集成
   - 外部服务调用
   - 数据源适配
   - 网络通信处理

7. **数据模型层 (models/)**
   - 业务实体定义
   - 数据传输对象
   - 数据验证规则
   - 类型定义

8. **工具模块层 (utils/)**
   - 通用工具函数
   - 公共组件
   - 异常处理
   - 常量定义









# 量化系统

## 项目环境

使用 UV 管理项目的Python运行环境。

## 项目结构

```
quantitative-system/
├── main.py                    # 应用入口
├── config/                    # 配置管理
│   ├── __init__.py
│   ├── settings.py           # 应用配置
│   └── database.py           # 数据库配置
├── api/                       # API接口层
│   ├── __init__.py
│   ├── routes/               # 路由定义
│   │   ├── __init__.py
│   │   ├── positions.py      # 持仓管理接口
│   │   ├── plans.py          # 方案管理接口
│   │   └── system.py         # 系统状态接口
│   ├── dependencies.py       # 依赖注入
│   └── middleware.py         # 中间件
├── scheduler/                 # 调度层
│   ├── __init__.py
│   ├── jobs.py               # 任务定义
│   ├── scheduler.py          # 调度器
│   └── manager.py            # 任务管理器
├── biz/                       # 业务编排层
│   ├── __init__.py
│   ├── plan_orchestrator.py  # 方案生成编排器
│   ├── backtest_orchestrator.py # 回测分析编排器
│   ├── position_orchestrator.py # 持仓管理编排器
│   └── base_orchestrator.py  # 编排器基类
├── services/                  # 业务服务层
│   ├── __init__.py
│   ├── position_service.py   # 持仓管理服务
│   ├── backtest_service.py   # 回测服务
│   ├── ai_service.py         # AI分析服务
│   ├── plan_service.py       # 方案生成服务
│   └── data_service.py       # 数据整合服务
├── models/                    # 数据模型
│   ├── __init__.py
│   ├── database.py           # 数据库模型
│   ├── schemas.py            # Pydantic模型
│   └── enums.py              # 枚举定义
├── repositories/              # 数据访问层
│   ├── __init__.py
│   ├── position_repo.py      # 持仓数据访问
│   ├── backtest_repo.py      # 回测数据访问
│   ├── plan_repo.py          # 方案数据访问
│   └── cache_repo.py         # 缓存数据访问
├── strategies/                # 交易策略
│   ├── __init__.py
│   ├── base_strategy.py      # 策略基类
│   ├── ma_strategy.py        # 均线策略
│   ├── macd_strategy.py      # MACD策略
│   └── rsi_strategy.py       # RSI策略
├── utils/                     # 工具模块
│   ├── __init__.py
│   ├── logger.py             # 日志工具
│   ├── exceptions.py         # 异常定义
│   ├── validators.py         # 数据验证
│   └── helpers.py            # 辅助函数
└── tests/                     # 测试代码
    ├── __init__.py
    ├── test_biz/
    ├── test_services/
    ├── test_repositories/
    └── test_strategies/
```