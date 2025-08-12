# 后端开发自测用例 - 数据采集子系统

## 文档信息

| 项目    | 内容          |
| ----- | ----------- |
| 子系统名称 | 量化平台数据采集子系统 |
| 文档版本  | v1.0        |
| 创建日期  | 2024-12-19  |
| 最后更新  | 2024-12-19  |
| 测试负责人 | 后端开发工程师 |
| 文档状态  | 待评审         |
| 所属平台  | 量化交易平台      |

## 1. 功能概述

本文档为数据采集子系统的后端开发自测用例，涵盖已实现的核心功能模块：
- Tushare数据采集服务
- 新闻数据采集和NLP处理服务
- 数据质量验证服务
- HTTP API接口服务
- 查询服务和缓存机制

**测试目标**：确保各模块基础功能正常，API接口能够正确响应，数据处理逻辑符合预期。

**测试范围**：开发阶段的功能验证，快速反馈代码质量问题。

## 2. API接口自测用例

### 2.1 股票数据API接口测试

#### DEV_API_STOCKS_001: 股票基础信息查询

**测试目标**: 验证股票基础信息查询接口的基本功能

**接口信息**:
- 路径: `GET /api/v1/stocks/basic`
- 文件: `api/routes/stocks.py`

**测试步骤**:
1. 启动FastAPI应用: `uvicorn main:app --reload`
2. 访问接口文档: `http://localhost:8000/docs`
3. 测试基础查询: `GET /api/v1/stocks/basic?limit=10`
4. 测试分页查询: `GET /api/v1/stocks/basic?page=1&limit=5`
5. 测试排序查询: `GET /api/v1/stocks/basic?sort_by=symbol&sort_order=asc`

**预期结果**:
- 返回状态码200
- 响应格式符合PaginatedResponse[StockBasicInfoResponse]结构
- 数据字段包含: symbol, name, industry, list_date等
- 分页信息正确: total, page, limit, has_next等

#### DEV_API_STOCKS_002: 股票日线数据查询

**测试目标**: 验证股票日线数据查询接口的功能

**接口信息**:
- 路径: `GET /api/v1/stocks/daily`
- 文件: `api/routes/stocks.py`

**测试步骤**:
1. 测试按股票代码查询: `GET /api/v1/stocks/daily?symbol=000001.SZ`
2. 测试日期范围查询: `GET /api/v1/stocks/daily?start_date=2024-01-01&end_date=2024-01-31`
3. 测试组合条件查询: `GET /api/v1/stocks/daily?symbol=000001.SZ&start_date=2024-01-01`
4. 测试无效参数: `GET /api/v1/stocks/daily?symbol=INVALID`

**预期结果**:
- 有效查询返回状态码200
- 数据字段包含: symbol, trade_date, open, high, low, close, volume等
- 无效参数返回适当错误信息
- 日期格式验证正确

#### DEV_API_STOCKS_003: 财务数据查询

**测试目标**: 验证财务数据查询接口的功能

**接口信息**:
- 路径: `GET /api/v1/stocks/financial`
- 文件: `api/routes/stocks.py`

**测试步骤**:
1. 测试按股票代码查询: `GET /api/v1/stocks/financial?symbol=000001.SZ`
2. 测试报告期查询: `GET /api/v1/stocks/financial?period=20231231`
3. 测试组合查询: `GET /api/v1/stocks/financial?symbol=000001.SZ&period=20231231`

**预期结果**:
- 返回状态码200
- 数据包含财务指标字段
- 报告期格式验证正确

### 2.2 新闻数据API接口测试

#### DEV_API_NEWS_001: 新闻数据查询

**测试目标**: 验证新闻数据查询接口的基本功能

**接口信息**:
- 路径: `GET /api/v1/news/`
- 文件: `api/routes/news.py`

**测试步骤**:
1. 测试基础查询: `GET /api/v1/news/?limit=10`
2. 测试按股票代码查询: `GET /api/v1/news/?symbol=000001.SZ`
3. 测试日期范围查询: `GET /api/v1/news/?start_date=2024-01-01&end_date=2024-01-31`
4. 测试关键词搜索: `GET /api/v1/news/?keyword=财报`

**预期结果**:
- 返回状态码200
- 响应格式符合PaginatedResponse[NewsDataResponse]结构
- 数据字段包含: title, content, source, publish_time, related_stocks等

#### DEV_API_NEWS_002: 情感分析结果查询

**测试目标**: 验证情感分析结果查询接口的功能

**接口信息**:
- 路径: `GET /api/v1/news/sentiment`
- 文件: `api/routes/news.py`

**测试步骤**:
1. 测试基础查询: `GET /api/v1/news/sentiment?limit=10`
2. 测试按股票代码查询: `GET /api/v1/news/sentiment?symbol=000001.SZ`
3. 测试情感类型过滤: `GET /api/v1/news/sentiment?sentiment=positive`

**预期结果**:
- 返回状态码200
- 数据字段包含: sentiment, confidence, intensity, entities等
- 情感类型枚举值正确

#### DEV_API_NEWS_003: 情感统计查询

**测试目标**: 验证情感统计接口的功能

**接口信息**:
- 路径: `GET /api/v1/news/sentiment/stats`
- 文件: `api/routes/news.py`

**测试步骤**:
1. 测试按股票代码统计: `GET /api/v1/news/sentiment/stats?symbol=000001.SZ`
2. 测试日期范围统计: `GET /api/v1/news/sentiment/stats?start_date=2024-01-01&end_date=2024-01-31`

**预期结果**:
- 返回状态码200
- 统计数据包含各情感类型的数量和比例

### 2.3 手动数据采集API接口测试

#### DEV_API_COLLECTION_001: 全量数据采集触发

**测试目标**: 验证手动触发全量数据采集接口的功能

**接口信息**:
- 路径: `POST /api/v1/collection/full`
- 文件: `api/routes/collection.py`

**测试步骤**:
1. 发送POST请求: `POST /api/v1/collection/full`
2. 检查响应中的task_id
3. 验证任务状态: `GET /api/v1/collection/tasks/{task_id}`
4. 监控任务执行进度
5. 验证数据库中数据更新情况

**预期结果**:
- 返回状态码202 (Accepted)
- 响应包含task_id和任务状态
- 任务能够正常执行
- 数据库中数据得到更新

#### DEV_API_COLLECTION_002: 增量数据采集触发

**测试目标**: 验证手动触发增量数据采集接口的功能

**接口信息**:
- 路径: `POST /api/v1/collection/incremental`
- 文件: `api/routes/collection.py`

**测试步骤**:
1. 发送POST请求: `POST /api/v1/collection/incremental`
2. 检查响应中的task_id
3. 验证任务状态和进度
4. 确认只采集最新数据

**预期结果**:
- 返回状态码202
- 任务正常执行
- 只更新增量数据

#### DEV_API_COLLECTION_003: 指定股票数据采集

**测试目标**: 验证指定股票代码的数据采集功能

**接口信息**:
- 路径: `POST /api/v1/collection/stocks`
- 文件: `api/routes/collection.py`

**测试步骤**:
1. 准备请求体: `{"stock_codes": ["000001.SZ", "000002.SZ"]}`
2. 发送POST请求
3. 验证任务创建和执行
4. 确认只采集指定股票数据

**预期结果**:
- 返回状态码202
- 任务针对指定股票执行
- 数据采集范围正确

#### DEV_API_COLLECTION_004: 日期范围数据采集

**测试目标**: 验证指定日期范围的数据采集功能

**接口信息**:
- 路径: `POST /api/v1/collection/range`
- 文件: `api/routes/collection.py`

**测试步骤**:
1. 准备请求体: `{"start_date": "2024-01-01", "end_date": "2024-01-31"}`
2. 发送POST请求
3. 验证任务创建和执行
4. 确认采集指定日期范围的数据

**预期结果**:
- 返回状态码202
- 任务按日期范围执行
- 数据时间范围正确

#### DEV_API_COLLECTION_005: 采集任务状态查询

**测试目标**: 验证采集任务状态查询功能

**接口信息**:
- 路径: `GET /api/v1/collection/tasks/{task_id}`
- 文件: `api/routes/collection.py`

**测试步骤**:
1. 先创建一个采集任务获取task_id
2. 查询任务状态: `GET /api/v1/collection/tasks/{task_id}`
3. 验证状态信息的完整性
4. 测试无效task_id的错误处理

**预期结果**:
- 有效task_id返回状态码200
- 响应包含status, progress, created_at等字段
- 无效task_id返回404错误

#### DEV_API_COLLECTION_006: 采集任务列表查询

**测试目标**: 验证采集任务列表查询功能

**接口信息**:
- 路径: `GET /api/v1/collection/tasks`
- 文件: `api/routes/collection.py`

**测试步骤**:
1. 创建多个采集任务
2. 查询任务列表: `GET /api/v1/collection/tasks?limit=10`
3. 测试状态过滤: `GET /api/v1/collection/tasks?status=running`
4. 测试分页功能

**预期结果**:
- 返回状态码200
- 响应格式符合分页结构
- 过滤和分页功能正常

#### DEV_API_COLLECTION_007: 采集任务取消

**测试目标**: 验证采集任务取消功能

**接口信息**:
- 路径: `DELETE /api/v1/collection/tasks/{task_id}`
- 文件: `api/routes/collection.py`

**测试步骤**:
1. 创建一个长时间运行的采集任务
2. 发送取消请求: `DELETE /api/v1/collection/tasks/{task_id}`
3. 验证任务状态变为cancelled
4. 确认任务实际停止执行

**预期结果**:
- 返回状态码200
- 任务状态更新为cancelled
- 任务实际停止执行

## 3. 数据处理自测用例

### 3.1 数据采集服务测试

#### DEV_DT_COLLECTION_001: Tushare数据采集

**测试目标**: 验证Tushare数据采集服务的基本功能

**相关文件**: `services/collection_service.py`, `clients/tushare_client.py`

**测试步骤**:
1. 检查Tushare API配置: 确认.env文件中TUSHARE_TOKEN配置正确
2. 测试连接: 调用`tushare_client.test_connection()`
3. 测试股票列表采集: 调用`collection_service.collect_stock_basic()`
4. 测试日线数据采集: 调用`collection_service.collect_daily_data(symbol='000001.SZ')`
5. 检查数据库: 验证数据是否正确存储到stocks和stock_daily表

**预期结果**:
- API连接成功
- 数据采集无异常
- 数据格式符合数据库模型定义
- 日志记录采集过程和结果

#### DEV_DT_COLLECTION_002: 新闻数据采集

**测试目标**: 验证新闻数据采集功能

**相关文件**: `services/collection_service.py`, `clients/news_crawler.py`

**测试步骤**:
1. 测试新闻爬虫: 调用`news_crawler.crawl_news()`
2. 检查robots.txt遵守: 验证访问频率控制
3. 测试数据清洗: 验证新闻内容预处理
4. 检查数据存储: 验证新闻数据存储到news表

**预期结果**:
- 新闻采集成功
- 内容格式正确
- 去重机制有效
- 错误处理正常

### 3.2 NLP处理服务测试

#### DEV_DT_NLP_001: FinBERT情感分析

**测试目标**: 验证FinBERT模型的情感分析功能

**相关文件**: `services/nlp_service.py`, `utils/finbert_client.py`

**测试步骤**:
1. 测试模型加载: 调用`finbert_client.load_model()`
2. 测试文本预处理: 调用`text_processor.preprocess(text)`
3. 测试情感分析: 调用`nlp_service.analyze_sentiment(text)`
4. 测试批量处理: 调用`nlp_service.batch_analyze(texts)`
5. 验证结果格式: 检查sentiment, confidence, intensity字段

**预期结果**:
- 模型加载成功
- 情感分析结果合理
- 置信度在0-1范围内
- 批量处理效率符合预期

#### DEV_DT_NLP_002: 实体识别和关键词提取

**测试目标**: 验证实体识别和关键词提取功能

**相关文件**: `services/nlp_service.py`

**测试步骤**:
1. 测试实体识别: 调用`nlp_service.extract_entities(text)`
2. 测试关键词提取: 调用`nlp_service.extract_keywords(text)`
3. 测试股票关联: 调用`nlp_service.link_to_stocks(entities)`

**预期结果**:
- 实体识别准确
- 关键词提取合理
- 股票关联逻辑正确

### 3.3 手动数据采集服务测试

#### DEV_DT_MANUAL_001: 手动全量采集服务

**测试目标**: 验证手动触发全量数据采集的服务层逻辑

**相关文件**: `services/collection_service.py`, `biz/data_collection_orchestrator.py`

**测试步骤**:
1. 调用手动全量采集: `collection_service.trigger_full_collection()`
2. 验证任务创建: 检查任务记录是否正确创建
3. 监控采集进度: 验证进度更新机制
4. 检查数据完整性: 验证所有股票基础信息和历史数据
5. 验证任务状态: 确认任务状态正确更新

**预期结果**:
- 任务成功创建并分配唯一ID
- 采集进度实时更新
- 数据库中数据完整更新
- 任务状态正确流转(pending→running→completed)

#### DEV_DT_MANUAL_002: 手动增量采集服务

**测试目标**: 验证手动触发增量数据采集的服务层逻辑

**相关文件**: `services/collection_service.py`

**测试步骤**:
1. 调用手动增量采集: `collection_service.trigger_incremental_collection()`
2. 验证时间范围: 确认只采集最新数据
3. 检查数据去重: 验证重复数据处理
4. 监控采集效率: 验证增量采集的性能
5. 验证数据一致性: 确认增量数据与存量数据一致

**预期结果**:
- 只采集指定时间范围的新数据
- 重复数据被正确处理
- 采集效率明显优于全量采集
- 数据一致性得到保证

#### DEV_DT_MANUAL_003: 指定股票采集服务

**测试目标**: 验证指定股票代码的数据采集功能

**相关文件**: `services/collection_service.py`

**测试步骤**:
1. 准备股票代码列表: `['000001.SZ', '000002.SZ', '600000.SH']`
2. 调用指定采集: `collection_service.trigger_stocks_collection(stock_codes)`
3. 验证采集范围: 确认只采集指定股票
4. 检查数据完整性: 验证指定股票的所有相关数据
5. 测试错误处理: 验证无效股票代码的处理

**预期结果**:
- 只采集指定股票的数据
- 数据完整性符合要求
- 无效股票代码被正确处理
- 任务执行效率高

#### DEV_DT_MANUAL_004: 日期范围采集服务

**测试目标**: 验证指定日期范围的数据采集功能

**相关文件**: `services/collection_service.py`

**测试步骤**:
1. 设定日期范围: `start_date='2024-01-01', end_date='2024-01-31'`
2. 调用范围采集: `collection_service.trigger_range_collection(start_date, end_date)`
3. 验证时间范围: 确认采集数据的时间范围正确
4. 检查数据连续性: 验证日期范围内数据的连续性
5. 测试边界条件: 验证开始和结束日期的处理

**预期结果**:
- 采集数据的时间范围准确
- 数据连续性良好
- 边界日期处理正确
- 任务执行稳定

#### DEV_DT_MANUAL_005: 采集任务管理服务

**测试目标**: 验证采集任务的管理和状态跟踪功能

**相关文件**: `services/task_service.py`, `repositories/task_repo.py`

**测试步骤**:
1. 创建多个采集任务: 测试并发任务管理
2. 查询任务状态: `task_service.get_task_status(task_id)`
3. 更新任务进度: `task_service.update_progress(task_id, progress)`
4. 取消运行任务: `task_service.cancel_task(task_id)`
5. 清理完成任务: 验证任务清理机制

**预期结果**:
- 支持多任务并发管理
- 任务状态查询准确
- 进度更新实时有效
- 任务取消功能正常
- 任务清理机制完善

### 3.4 数据质量服务测试

#### DEV_DT_QUALITY_001: 数据验证

**测试目标**: 验证数据质量验证功能

**相关文件**: `services/quality_service.py`

**测试步骤**:
1. 测试格式验证: 调用`quality_service.validate_format(data)`
2. 测试完整性检查: 调用`quality_service.check_completeness(data)`
3. 测试异常检测: 调用`quality_service.detect_anomalies(data)`
4. 测试数据清洗: 调用`quality_service.clean_data(data)`

**预期结果**:
- 格式验证准确
- 完整性检查有效
- 异常检测合理
- 数据清洗正确

### 3.5 查询服务测试

#### DEV_DT_QUERY_001: 缓存机制

**测试目标**: 验证Redis缓存机制

**相关文件**: `services/query_service.py`, `repositories/cache_repo.py`

**测试步骤**:
1. 测试缓存写入: 调用`cache_repo.set(key, value)`
2. 测试缓存读取: 调用`cache_repo.get(key)`
3. 测试缓存过期: 验证TTL机制
4. 测试查询缓存: 调用`query_service.get_cached_result(query)`

**预期结果**:
- 缓存读写正常
- TTL机制有效
- 查询性能提升

#### DEV_DT_QUERY_002: 分页和排序

**测试目标**: 验证分页和排序功能

**相关文件**: `services/query_service.py`

**测试步骤**:
1. 测试分页查询: 调用`query_service.paginate(page=1, limit=10)`
2. 测试排序查询: 调用`query_service.sort(sort_by='field', order='asc')`
3. 测试过滤查询: 调用`query_service.filter(filters={})`

**预期结果**:
- 分页逻辑正确
- 排序功能正常
- 过滤条件有效

## 4. 自测检查清单

### 4.1 环境检查

- [ ] Python虚拟环境激活: `uv sync`
- [ ] 数据库连接正常: MySQL和Redis服务启动
- [ ] 环境变量配置: .env文件配置完整
- [ ] 依赖包安装: 所有依赖包正常安装

### 4.2 代码质量检查

- [ ] 类型检查通过: `make type-check`
- [ ] 代码格式检查: `make lint`
- [ ] 代码格式化: `make format`
- [ ] 导入排序: `make sort-imports`

### 4.3 功能检查

- [ ] 应用启动正常: `uvicorn main:app --reload`
- [ ] API文档访问: `http://localhost:8000/docs`
- [ ] 健康检查接口: `GET /health`
- [ ] 数据库连接测试: 验证数据库依赖注入

### 4.4 核心服务检查

- [ ] Tushare客户端连接: 验证API密钥和网络连接
- [ ] 新闻爬虫功能: 验证网站访问和内容解析
- [ ] FinBERT模型加载: 验证模型文件和推理环境
- [ ] Redis缓存连接: 验证缓存读写功能
- [ ] 数据库CRUD操作: 验证基础数据操作

### 4.5 API接口检查

- [ ] 股票基础信息接口: 验证查询和分页功能
- [ ] 股票日线数据接口: 验证日期范围和过滤功能
- [ ] 财务数据接口: 验证报告期查询功能
- [ ] 新闻数据接口: 验证搜索和过滤功能
- [ ] 情感分析接口: 验证情感查询和统计功能

## 5. 常见问题解决方案

### 5.1 环境问题

**问题**: 虚拟环境依赖安装失败
**解决方案**: 
```bash
# 清理缓存重新安装
uv cache clean
uv sync --reinstall
```

**问题**: 数据库连接失败
**解决方案**: 
1. 检查MySQL和Redis服务状态
2. 验证.env文件中的数据库配置
3. 检查网络连接和防火墙设置

### 5.2 API问题

**问题**: API接口返回500错误
**解决方案**: 
1. 检查应用日志: `tail -f logs/app.log`
2. 验证数据库连接和数据完整性
3. 检查依赖注入配置

**问题**: API响应时间过长
**解决方案**: 
1. 检查数据库查询性能
2. 验证Redis缓存是否生效
3. 优化查询条件和索引

### 5.3 数据处理问题

**问题**: Tushare API调用失败
**解决方案**: 
1. 验证API密钥有效性
2. 检查API调用频率限制
3. 确认网络连接稳定性

**问题**: FinBERT模型加载失败
**解决方案**: 
1. 检查模型文件路径和权限
2. 验证transformers库版本兼容性
3. 确认GPU/CPU环境配置

**问题**: 新闻爬虫被反爬虫机制阻止
**解决方案**: 
1. 调整访问频率和间隔时间
2. 检查User-Agent和请求头设置
3. 遵守robots.txt规则

### 5.4 性能问题

**问题**: 数据库查询性能差
**解决方案**: 
1. 检查数据库索引配置
2. 优化SQL查询语句
3. 考虑数据分页和缓存策略

**问题**: 内存使用过高
**解决方案**: 
1. 检查批量处理的数据量
2. 优化模型推理的批次大小
3. 及时释放不需要的对象引用

## 修改记录

### [2024-12-19] v1.0 初始版本创建

**创建内容**：
1. **测试用例设计**：基于已实现的核心功能模块设计自测用例
2. **API接口测试**：覆盖stocks和news两个主要API模块的测试用例
3. **数据处理测试**：包含数据采集、NLP处理、质量验证、查询服务的测试用例
4. **自测检查清单**：提供完整的开发自测检查项目
5. **问题解决方案**：针对常见开发问题提供解决方案

**测试特点**：
- 面向后端开发工程师的快速自测
- 覆盖已实现的核心功能模块
- 提供具体的测试步骤和预期结果
- 包含环境配置和问题排查指导
- 支持开发阶段的快速反馈和质量保证

**技术对齐**：
- 基于当前已实现的代码结构和功能
- 符合FastAPI + SQLAlchemy + Redis技术栈
- 支持FinBERT模型的NLP处理测试
- 遵循Python开发最佳实践

### [2024-12-19] v1.1 手动数据采集功能测试用例新增

**新增内容**：
1. **手动采集API接口测试**：新增7个API接口测试用例（DEV_API_COLLECTION_001-007）
   - 全量数据采集触发测试
   - 增量数据采集触发测试
   - 指定股票数据采集测试
   - 日期范围数据采集测试
   - 采集任务状态查询测试
   - 采集任务列表查询测试
   - 采集任务取消测试

2. **手动采集服务层测试**：新增5个数据处理测试用例（DEV_DT_MANUAL_001-005）
   - 手动全量采集服务测试
   - 手动增量采集服务测试
   - 指定股票采集服务测试
   - 日期范围采集服务测试
   - 采集任务管理服务测试

3. **测试结构优化**：重新组织测试用例结构，将数据质量测试调整为3.4节

**测试覆盖增强**：
- 新增手动数据采集API的完整接口测试
- 增强了采集任务管理和状态跟踪的测试
- 完善了服务层业务逻辑的验证测试
- 加强了错误处理和异常情况的测试覆盖

**开发自测支持**：
- 提供具体的API测试步骤和curl命令
- 包含服务层方法调用的测试指导
- 支持快速验证手动采集功能的正确性
- 确保开发阶段的质量保证