# 数据采集子系统后端技术设计文档

## 文档信息

| 项目    | 内容          |
| ----- | ----------- |
| 子系统名称 | 量化平台数据采集子系统 |
| 文档版本  | v1.0        |
| 创建日期  | 2024-12-19  |
| 最后更新  | 2024-12-19  |
| 架构师   | AI Assistant |
| 文档状态  | 待评审         |
| 所属平台  | 量化交易平台      |

## 1. 架构概览（Overview Layer）

### 1.1 项目目录架构

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

### 1.2 架构分层说明

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

### 1.3 系统架构图

```mermaid
graph TB
    subgraph "外部系统"
        A[回测系统]
        B[量化系统]
    end
    
    subgraph "数据采集子系统"
        subgraph "API层"
            C[FastAPI HTTP Server]
            D[定时任务调度器]
        end
        
        subgraph "业务服务层"
            E[数据采集服务]
            F[NLP处理服务]
            G[数据质量服务]
            H[查询服务]
        end
        
        subgraph "数据访问层"
            I[MySQL Repository]
            J[Redis Repository]
            K[外部API Client]
        end
        
        subgraph "存储层"
            L[(MySQL 8.0)]
            M[(Redis 7.0)]
        end
    end
    
        subgraph "NLP处理层"
            P[FinBERT模型]
        end
    end
    
    subgraph "外部数据源"
        N[Tushare API]
        O[金融新闻网站]
    end
    
    A --> C
    B --> C
    C --> E
    C --> F
    C --> G
    C --> H
    D --> E
    D --> F
    E --> I
    F --> I
    G --> I
    H --> I
    H --> J
    E --> K
    F --> K
    I --> L
    J --> M
    K --> N
    K --> O
    G --> P
```

### 1.2 核心模块概述

1. **API层**：FastAPI HTTP服务器和APScheduler定时任务调度
2. **业务服务层**：数据采集、NLP处理、数据质量管控、查询服务
3. **NLP处理层**：本地部署的FinBERT模型，提供金融文本情感分析能力
4. **数据访问层**：MySQL/Redis数据访问和外部API客户端
5. **存储层**：MySQL主数据库和Redis缓存

### 1.3 技术栈选型

| 技术类别   | 选择方案        | 版本要求      | 选择理由           |
| ------ | ----------- | --------- | -------------- |
| 开发语言   | Python      | 3.9+      | AI生态丰富，开发效率高   |
| Web框架  | FastAPI     | 0.104+    | 高性能，自动文档生成     |
| ORM    | SQLAlchemy  | 2.0+      | 成熟稳定，类型安全      |
| 任务调度   | APScheduler | 3.10+     | Python原生，易集成   |
| HTTP客户端 | httpx       | 0.25+     | 异步支持，现代化API    |
| 数据验证   | Pydantic    | 2.0+      | 类型安全，数据验证      |
| 日志     | Loguru      | 0.7+      | 简单易用，结构化日志     |
| 配置管理   | Pydantic Settings | 2.0+ | 类型安全的配置管理      |
| NLP模型   | FinBERT     | -         | 金融领域专业NLP模型，本地部署 |
| 深度学习框架 | PyTorch     | 2.0+      | FinBERT模型推理引擎    |
| 模型库    | Transformers | 4.30+     | HuggingFace模型加载   |

### 1.4 数据流概览

```mermaid
flowchart LR
    A[定时任务] --> B[数据采集服务]
    B --> C[Tushare API]
    B --> D[新闻网站]
    B --> E[数据预处理]
    E --> F[MySQL存储]
    F --> G[NLP处理服务]
    G --> H[FinBERT模型]
    G --> I[情感分析结果]
    I --> F
    J[API查询] --> K[查询服务]
    K --> L[Redis缓存]
    K --> F
    K --> M[JSON响应]
```

## 2. 模块设计（Module Layer）

### 2.1 数据采集服务（Collection Service）

#### 2.1.1 模块职责
- 从Tushare API采集股票基础信息、行情数据、财务数据
- 从金融网站爬取新闻数据
- 数据预处理和格式标准化
- 增量数据更新机制

#### 2.1.2 模块架构

```mermaid
classDiagram
    class CollectionService {
        +collect_stock_basic()
        +collect_daily_data()
        +collect_financial_data()
        +collect_news_data()
        +validate_data()
    }
    
    class TushareClient {
        +get_stock_basic()
        +get_daily_data()
        +get_financial_data()
        +handle_rate_limit()
    }
    
    class NewsSpider {
        +crawl_news()
        +parse_content()
        +extract_entities()
        +respect_robots_txt()
    }
    
    CollectionService --> TushareClient
    CollectionService --> NewsSpider
```

#### 2.1.3 核心接口

```python
class CollectionService:
    async def collect_stock_basic(self) -> List[StockBasic]:
        """采集股票基础信息"""
        
    async def collect_daily_data(self, trade_date: str) -> List[StockDaily]:
        """采集日度行情数据"""
        
    async def collect_financial_data(self, period: str) -> List[FinancialData]:
        """采集财务数据"""
        
    async def collect_news_data(self, start_date: str, end_date: str) -> List[NewsData]:
        """采集新闻数据"""
```

### 2.2 NLP处理服务（NLP Service）

#### 2.2.1 模块职责
- 基于FinBERT模型进行新闻情感分析
- 实体识别和关键词提取
- 情感强度量化计算
- 批量处理和异步处理

#### 2.2.2 模块架构

```mermaid
classDiagram
    class NLPService {
        +analyze_sentiment()
        +extract_entities()
        +extract_keywords()
        +calculate_intensity()
        +batch_process()
    }
    
    class FinBERTClient {
        +load_model()
        +predict_sentiment()
        +extract_features()
        +handle_errors()
        +batch_predict()
        +optimize_inference()
    }
    
    class TextProcessor {
        +clean_text()
        +tokenize()
        +normalize()
        +validate_input()
    }
    
    NLPService --> FinBERTClient
    NLPService --> TextProcessor
```

#### 2.2.3 核心接口

```python
class NLPService:
    async def analyze_sentiment(self, text: str) -> SentimentResult:
        """情感分析"""
        
    async def extract_entities(self, text: str) -> List[Entity]:
        """实体识别"""
        
    async def extract_keywords(self, text: str) -> List[str]:
        """关键词提取"""
        
    async def batch_process_news(self, news_list: List[NewsData]) -> List[SentimentAnalysis]:
        """批量处理新闻"""

class FinBERTClient:
    def __init__(self, model_path: str = "ProsusAI/finbert"):
        """初始化FinBERT客户端（本地部署）"""
        
    def load_model(self) -> None:
        """加载FinBERT模型到本地内存
        
        技术实现：
        - 使用transformers库加载预训练模型
        - 支持模型量化以减少内存占用
        - 配置GPU/CPU推理环境
        """
        
    async def predict_sentiment(self, text: str) -> SentimentResult:
        """单条文本情感分析"""
        
    async def batch_predict(self, texts: List[str]) -> List[SentimentResult]:
        """批量文本情感分析（性能优化）"""
        
    def extract_features(self, text: str) -> torch.Tensor:
        """提取文本特征向量"""
        
    def optimize_inference(self) -> None:
        """推理性能优化
        
        优化策略：
        - 模型量化（INT8/FP16）
        - 批处理优化
        - 缓存机制
        - 异步推理
        """
        
    def handle_errors(self, error: Exception) -> None:
        """错误处理和重试机制"""
```

### 2.3 数据质量服务（Quality Service）

#### 2.3.1 模块职责
- 数据完整性验证
- 数据格式校验
- 异常数据检测和处理
- 数据清洗和标准化

#### 2.3.2 核心接口

```python
class QualityService:
    def validate_stock_data(self, data: StockData) -> ValidationResult:
        """股票数据验证"""
        
    def validate_news_data(self, data: NewsData) -> ValidationResult:
        """新闻数据验证"""
        
    def clean_data(self, data: Any) -> Any:
        """数据清洗"""
        
    def detect_anomalies(self, data: List[Any]) -> List[Anomaly]:
        """异常检测"""
```

### 2.4 查询服务（Query Service）

#### 2.4.1 模块职责
- 提供统一的数据查询接口
- 缓存管理和查询优化
- 分页查询和结果聚合
- 查询参数验证

#### 2.4.2 核心接口

```python
class QueryService:
    async def get_stock_basic(self, symbol: str) -> Optional[StockBasic]:
        """获取股票基础信息"""
        
    async def get_daily_data(self, symbol: str, start_date: str, end_date: str) -> List[StockDaily]:
        """获取行情数据"""
        
    async def get_news_with_sentiment(self, symbol: str, start_date: str, end_date: str) -> List[NewsWithSentiment]:
        """获取新闻和情感分析"""
        
    async def get_financial_data(self, symbol: str, period: str) -> List[FinancialData]:
        """获取财务数据"""
```

## 3. 接口设计（API Design）

### 3.1 接口规范

- **协议**：HTTP/1.1
- **数据格式**：JSON
- **编码**：UTF-8
- **认证**：无需认证（MVP版本）
- **基础URL**：`http://localhost:8000/api/v1`

### 3.2 统一响应格式

```python
class APIResponse(BaseModel):
    code: int = 200
    message: str = "success"
    data: Optional[Any] = None
    timestamp: datetime = Field(default_factory=datetime.now)
    
class ErrorResponse(BaseModel):
    code: int
    message: str
    detail: Optional[str] = None
    timestamp: datetime = Field(default_factory=datetime.now)
```

### 3.3 接口列表

#### 3.3.1 股票数据接口

```python
# 获取股票基础信息
GET /api/v1/stocks/basic/{symbol}
Response: APIResponse[StockBasic]

# 获取股票行情数据
GET /api/v1/stocks/daily/{symbol}?start_date={date}&end_date={date}&limit={int}&offset={int}
Response: APIResponse[List[StockDaily]]

# 获取财务数据
GET /api/v1/stocks/financial/{symbol}?period={period}&report_type={type}
Response: APIResponse[List[FinancialData]]
```

#### 3.3.2 新闻数据接口

```python
# 获取股票相关新闻
GET /api/v1/news?symbol={symbol}&start_date={date}&end_date={date}&limit={int}&offset={int}
Response: APIResponse[List[NewsData]]

# 获取新闻情感分析
GET /api/v1/news/sentiment?symbol={symbol}&start_date={date}&end_date={date}
Response: APIResponse[List[SentimentAnalysis]]
```

#### 3.3.3 手动数据采集接口

```python
# 触发全量数据采集
POST /api/v1/collection/full
Request: {"data_types": ["basic", "daily", "financial"], "force_update": false}
Response: APIResponse[TaskInfo]

# 触发增量数据采集
POST /api/v1/collection/incremental
Request: {"target_date": "2024-01-01", "data_types": ["daily"]}
Response: APIResponse[TaskInfo]

# 触发指定股票数据采集
POST /api/v1/collection/stocks
Request: {"symbols": ["000001.SZ", "000002.SZ"], "data_types": ["basic", "daily"], "start_date": "2024-01-01", "end_date": "2024-01-31"}
Response: APIResponse[TaskInfo]

# 触发日期范围数据采集
POST /api/v1/collection/range
Request: {"start_date": "2024-01-01", "end_date": "2024-01-31", "data_types": ["daily"], "symbols": []}
Response: APIResponse[TaskInfo]

# 获取采集任务状态
GET /api/v1/collection/tasks/{task_id}
Response: APIResponse[TaskStatus]

# 获取采集任务列表
GET /api/v1/collection/tasks?status={status}&limit={int}&offset={int}
Response: APIResponse[List[TaskInfo]]

# 取消采集任务
DELETE /api/v1/collection/tasks/{task_id}
Response: APIResponse[TaskInfo]
```

#### 3.3.4 系统状态接口

```python
# 获取系统状态
GET /api/v1/system/status
Response: APIResponse[SystemStatus]

# 获取数据同步状态
GET /api/v1/system/sync-status
Response: APIResponse[SyncStatus]
```

## 4. 数据库设计

### 4.1 数据库架构

- **主数据库**：MySQL 8.0+
- **缓存数据库**：Redis 7.0+
- **连接池**：SQLAlchemy连接池管理
- **事务管理**：支持ACID事务

### 4.2 表结构设计

#### 4.2.1 股票基础信息表 (stocks)

```sql
CREATE TABLE stocks (
    symbol VARCHAR(10) PRIMARY KEY COMMENT '股票代码',
    name VARCHAR(50) NOT NULL COMMENT '股票名称',
    industry VARCHAR(50) COMMENT '行业分类',
    list_date DATE COMMENT '上市日期',
    market VARCHAR(10) COMMENT '市场类型',
    status TINYINT DEFAULT 1 COMMENT '状态：1-正常，0-停牌',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_industry (industry),
    INDEX idx_list_date (list_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='股票基础信息表';
```

#### 4.2.2 股票行情表 (stock_daily)

```sql
CREATE TABLE stock_daily (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL COMMENT '股票代码',
    trade_date DATE NOT NULL COMMENT '交易日期',
    open_price DECIMAL(10,3) COMMENT '开盘价',
    high_price DECIMAL(10,3) COMMENT '最高价',
    low_price DECIMAL(10,3) COMMENT '最低价',
    close_price DECIMAL(10,3) COMMENT '收盘价',
    volume BIGINT COMMENT '成交量',
    amount DECIMAL(15,2) COMMENT '成交额',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_symbol_date (symbol, trade_date),
    INDEX idx_trade_date (trade_date),
    INDEX idx_symbol (symbol)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='股票日线行情表';
```

#### 4.2.3 财务数据表 (financial_data)

```sql
CREATE TABLE financial_data (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL COMMENT '股票代码',
    period VARCHAR(10) NOT NULL COMMENT '报告期',
    report_type VARCHAR(20) NOT NULL COMMENT '报表类型',
    data JSON COMMENT '财务数据JSON',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_symbol_period_type (symbol, period, report_type),
    INDEX idx_period (period)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='财务数据表';
```

#### 4.2.4 新闻数据表 (news)

```sql
CREATE TABLE news (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(500) NOT NULL COMMENT '新闻标题',
    content TEXT COMMENT '新闻内容',
    source VARCHAR(100) COMMENT '新闻来源',
    publish_time TIMESTAMP COMMENT '发布时间',
    url VARCHAR(500) COMMENT '新闻链接',
    related_stocks JSON COMMENT '关联股票代码',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_publish_time (publish_time),
    INDEX idx_source (source),
    FULLTEXT idx_title_content (title, content)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='新闻数据表';
```

#### 4.2.5 情感分析表 (sentiment_analysis)

```sql
CREATE TABLE sentiment_analysis (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    news_id BIGINT NOT NULL COMMENT '新闻ID',
    symbol VARCHAR(10) COMMENT '股票代码',
    sentiment VARCHAR(20) NOT NULL COMMENT '情感倾向',
    confidence DECIMAL(5,4) COMMENT '置信度',
    intensity INT COMMENT '情感强度1-10',
    entities JSON COMMENT '实体识别结果',
    keywords JSON COMMENT '关键词',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (news_id) REFERENCES news(id),
    INDEX idx_news_id (news_id),
    INDEX idx_symbol (symbol),
    INDEX idx_sentiment (sentiment)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='情感分析表';
```

### 4.3 索引策略

1. **主键索引**：所有表都有主键索引
2. **唯一索引**：防止数据重复（如股票代码+交易日期）
3. **复合索引**：支持多字段查询优化
4. **全文索引**：支持新闻内容搜索
5. **时间索引**：支持时间范围查询优化

### 4.4 数据迁移策略

```python
# 使用Alembic进行数据库版本管理
from alembic import command
from alembic.config import Config

def upgrade_database():
    """升级数据库到最新版本"""
    alembic_cfg = Config("alembic.ini")
    command.upgrade(alembic_cfg, "head")

def downgrade_database(revision: str):
    """回滚数据库到指定版本"""
    alembic_cfg = Config("alembic.ini")
    command.downgrade(alembic_cfg, revision)
```

## 5. 时序图设计

### 5.1 数据采集流程时序图

```mermaid
sequenceDiagram
    participant S as 定时调度器
    participant CS as 数据采集服务
    participant TC as Tushare客户端
    participant NS as 新闻爬虫
    participant QS as 数据质量服务
    participant DB as MySQL数据库
    
    S->>CS: 触发日度数据采集
    CS->>TC: 获取股票行情数据
    TC-->>CS: 返回行情数据
    CS->>NS: 爬取新闻数据
    NS-->>CS: 返回新闻数据
    CS->>QS: 数据质量验证
    QS-->>CS: 验证结果
    CS->>DB: 存储数据
    DB-->>CS: 存储确认
    CS-->>S: 采集完成
```

### 5.2 NLP处理流程时序图

```mermaid
sequenceDiagram
    participant S as 定时调度器
    participant NS as NLP服务
    participant FB as FinBERT模型
    participant TP as 文本处理器
    participant DB as MySQL数据库
    
    S->>NS: 触发NLP处理任务
    NS->>DB: 获取待处理新闻
    DB-->>NS: 返回新闻列表
    loop 处理每条新闻
        NS->>TP: 文本预处理
        TP-->>NS: 清洗后文本
        NS->>FB: 情感分析
        FB-->>NS: 分析结果
        NS->>DB: 存储分析结果
    end
    NS-->>S: 处理完成
```

### 5.3 API查询流程时序图

```mermaid
sequenceDiagram
    participant C as 客户端
    participant API as FastAPI服务
    participant QS as 查询服务
    participant Redis as Redis缓存
    participant DB as MySQL数据库
    
    C->>API: 发送查询请求
    API->>QS: 调用查询服务
    QS->>Redis: 检查缓存
    alt 缓存命中
        Redis-->>QS: 返回缓存数据
    else 缓存未命中
        QS->>DB: 查询数据库
        DB-->>QS: 返回数据
        QS->>Redis: 更新缓存
    end
    QS-->>API: 返回查询结果
    API-->>C: 返回JSON响应
```

## 6. 实现细节（Implementation Layer）

### 6.1 错误处理策略

```python
class DataCollectionError(Exception):
    """数据采集异常"""
    pass

class NLPProcessingError(Exception):
    """NLP处理异常"""
    pass

class DataValidationError(Exception):
    """数据验证异常"""
    pass

# 全局异常处理器
@app.exception_handler(DataCollectionError)
async def data_collection_exception_handler(request: Request, exc: DataCollectionError):
    return JSONResponse(
        status_code=500,
        content=ErrorResponse(
            code=5001,
            message="数据采集失败",
            detail=str(exc)
        ).dict()
    )
```

### 6.2 日志记录规范

```python
from loguru import logger

# 配置日志格式
logger.add(
    "logs/data_collection_{time:YYYY-MM-DD}.log",
    rotation="1 day",
    retention="30 days",
    format="{time:YYYY-MM-DD HH:mm:ss} | {level} | {name}:{function}:{line} | {message}",
    level="INFO"
)

# 使用示例
logger.info("开始采集股票数据", symbol="000001", date="2024-12-19")
logger.error("数据采集失败", error=str(e), symbol="000001")
```

### 6.3 性能优化考虑

1. **数据库连接池**：使用SQLAlchemy连接池管理
2. **异步处理**：使用asyncio进行异步数据处理
3. **批量操作**：批量插入和更新数据
4. **缓存策略**：热点数据Redis缓存
5. **分页查询**：大数据量查询分页处理

```python
# 批量插入示例
async def batch_insert_stock_data(data_list: List[StockDaily]):
    async with get_db_session() as session:
        session.add_all(data_list)
        await session.commit()

# 缓存装饰器
from functools import wraps

def cache_result(expire_time: int = 3600):
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            cache_key = f"{func.__name__}:{hash(str(args) + str(kwargs))}"
            cached_result = await redis_client.get(cache_key)
            if cached_result:
                return json.loads(cached_result)
            
            result = await func(*args, **kwargs)
            await redis_client.setex(cache_key, expire_time, json.dumps(result, default=str))
            return result
        return wrapper
    return decorator
```

### 6.4 安全性设计

1. **配置安全**：敏感信息使用环境变量
2. **SQL注入防护**：使用ORM参数化查询
3. **输入验证**：Pydantic模型验证
4. **访问控制**：API访问频率限制

```python
# 配置管理
class Settings(BaseSettings):
    database_url: str = Field(..., env="DATABASE_URL")
    redis_url: str = Field(..., env="REDIS_URL")
    tushare_token: str = Field(..., env="TUSHARE_TOKEN")
    
    class Config:
        env_file = ".env"

# 输入验证
class StockQueryParams(BaseModel):
    symbol: str = Field(..., regex=r'^[0-9]{6}$')
    start_date: date
    end_date: date
    limit: int = Field(default=100, ge=1, le=1000)
```

### 6.5 监控和告警

```python
# 健康检查接口
@app.get("/health")
async def health_check():
    try:
        # 检查数据库连接
        await check_database_connection()
        # 检查Redis连接
        await check_redis_connection()
        # 检查外部API
        await check_external_apis()
        
        return {"status": "healthy", "timestamp": datetime.now()}
    except Exception as e:
        logger.error(f"健康检查失败: {e}")
        raise HTTPException(status_code=503, detail="Service Unavailable")

# 性能监控
from prometheus_client import Counter, Histogram

api_requests_total = Counter('api_requests_total', 'Total API requests', ['method', 'endpoint'])
api_request_duration = Histogram('api_request_duration_seconds', 'API request duration')
```

## 7. 部署架构

### 7.1 环境规划

- **开发环境**：本地MacOS，Docker Compose
- **测试环境**：单机部署，模拟生产环境
- **生产环境**：单机部署，4核8G配置

### 7.2 容器化策略

```dockerfile
# Dockerfile
FROM python:3.9-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 8000

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8000:8000"
    environment:
      - DATABASE_URL=mysql://user:password@mysql:3306/trading
      - REDIS_URL=redis://redis:6379
    depends_on:
      - mysql
      - redis

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: trading
    volumes:
      - mysql_data:/var/lib/mysql

  redis:
    image: redis:7.0
    volumes:
      - redis_data:/data

volumes:
  mysql_data:
  redis_data:
```

### 7.3 CI/CD流程

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Set up Python
        uses: actions/setup-python@v2
        with:
          python-version: 3.9
          
      - name: Install dependencies
        run: |
          pip install -r requirements.txt
          
      - name: Run tests
        run: |
          pytest tests/
          
      - name: Build and deploy
        run: |
          docker-compose up -d --build
```

## 8. 附录

### 8.1 术语表

| 术语      | 定义                    |
| ------- | --------------------- |
| MVP     | 最小可行产品（Minimum Viable Product） |
| FinBERT | 金融领域的BERT模型           |
| ORM     | 对象关系映射（Object-Relational Mapping） |
| API     | 应用程序编程接口（Application Programming Interface） |
| NLP     | 自然语言处理（Natural Language Processing） |

### 8.2 参考资料

1. [FastAPI官方文档](https://fastapi.tiangolo.com/)
2. [SQLAlchemy官方文档](https://docs.sqlalchemy.org/)
3. [Pydantic官方文档](https://pydantic-docs.helpmanual.io/)
4. [FinBERT模型论文](https://arxiv.org/abs/1908.10063)
5. [Tushare API文档](https://tushare.pro/document/2)

### 8.3 变更记录

#### [2024-12-19] v1.0 初始版本创建

**创建内容**：
1. **架构设计**：基于Python技术栈的单体架构设计
2. **模块设计**：数据采集、NLP处理、数据质量、查询服务四大核心模块
3. **接口设计**：RESTful API接口规范和统一响应格式
4. **数据库设计**：MySQL表结构设计和索引优化策略
5. **时序图设计**：核心业务流程的时序图
6. **实现细节**：错误处理、日志记录、性能优化、安全性设计
7. **部署架构**：容器化部署和CI/CD流程

**设计特点**：
- 基于PRD v4.0和Requirements v1.0文档
- 采用Python单体架构，技术栈统一
- 使用FinBERT模型进行专业金融NLP处理
- 遵循MVP原则，专注核心功能实现
- 支持异步处理和高性能优化

**技术优势**：
- 开发效率高，技术栈统一
- 部署简单，适合MVP快速验证
- 扩展性好，支持后续功能迭代
- 专业性强，FinBERT模型金融领域效果优秀

#### [2024-12-19] v1.1 FinBERT模型架构修正

**修正内容**：
1. **架构图修正**：将FinBERT模型从"外部数据源"移动到"NLP处理层"
2. **技术栈补充**：添加PyTorch、Transformers等本地部署相关技术栈
3. **模块概述更新**：明确NLP处理层包含本地部署的FinBERT模型
4. **接口设计完善**：详细设计FinBERTClient的本地部署实现接口
5. **任务描述优化**：明确任务12中FinBERT模型的本地部署要求

**修正原因**：
- 根据PRD文档，FinBERT模型应为本地部署，而非外部服务
- 确保架构设计与产品需求保持一致
- 明确本地部署的技术实现方案和性能优化策略