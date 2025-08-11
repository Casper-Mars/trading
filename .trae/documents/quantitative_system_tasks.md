# 任务拆分文档 (Tasks) — 量化交易系统后端

说明与标注规则

* 本文面向研发与测试团队，覆盖从MVP到迭代优化的实施计划

* 需求映射规则：x.y 表示“需求x 的验收标准 y”，对应 requirements 文档中的“需求N：…/验收标准”序号

* 优先级：以MVP为核心（P0），确保尽快交付可用版本

## 里程碑

* M0 项目基建与数据层奠定（1周）

* M1 核心功能（持仓CRUD、回测、AI方案、API）（2-3周）

* M2 调度自动化与稳定性（1周）

* M3 测试完善与性能守护（持续）

## 阶段0：项目基建与配置（M0）

* [x] 0.1 建立项目骨架与依赖管理

  * 使用 UV 初始化项目环境
  
  * 创建 Python FastAPI 项目骨架、目录与模块

  * 初始化依赖：fastapi, uvicorn, sqlmodel/sqlalchemy, pydantic-settings, httpx, redis, backtrader, pandas, numpy, apscheduler, loguru

  * 本地/开发/生产多环境配置模板

  * _Requirements: 5.6_

* [x] 0.2 配置与密钥管理

  * settings.py 定义：数据库、Redis、阿里百炼(dashscope)、回测参数

  * 区分环境变量注入敏感信息（不写入日志）

  * _Requirements: 3.6, 5.5_

* [x] 0.3 日志与错误处理基线

  * 结构化日志（loguru），统一异常中间件、错误码

  * 系统日志落库(system\_logs)与采样策略

  * _Requirements: 5.5_

## 阶段1：数据模型与存储（M0）

* [x] 1.1 建表与迁移脚本

  * 创建并执行迁移：positions, backtest_results, trading_plans, market_data_cache, system_logs

  * 基础索引与约束同步到文档

  * _Requirements: 1.1, 2.14, 3.14, 5.5_

* [x] 1.2 ORM与Schema

  * 定义 ORM 模型与 Pydantic Schema 输入/输出

  * 金额精度、时间字段、JSON 字段处理

  * _Requirements: 1.1, 2.14_

* [x] 1.3 Redis 缓存基线

  * 统一 key 规范与过期策略；封装 CacheRepo

  * _Requirements: 2.4, 2.7_

## 阶段2：数据访问层（Repositories）（M0）

* [x] 2.1 PositionRepo

  * 实现增删改查、聚合（组合市值/盈亏）

  * _Requirements: 1.1-1.7_

* [x] 2.2 BacktestRepo

  * 保存/查询回测任务与结果（含指标与原始数据JSON）

  * _Requirements: 2.14, 5.3_

*- [x] 2.3 PlanRepo

  * 以日期查询、保存方案、历史分页/范围查询

  * _Requirements: 3.14, 5.3_

*- [x] 2.4 External/DataCollection Client

  * 封装数据采集系统HTTP客户端（重试、超时、熔断占位）

  * _Requirements: 2.1, 2.2, 2.3_

## 阶段3：业务服务层（Services）（M1）

* [x] 3.1 PositionService

  * 业务校验、组合指标计算、按需更新当前价格/市值（调用数据采集系统）

  * _Requirements: 1.1-1.7, 5.1, 5.6, 5.7_

* [x] 3.2 DataService

  * 获取/清洗/预处理/缓存市场数据；技术指标计算接口

  * 处理缺失与异常值，返回标准化DataFrame

  * 按需调用数据采集系统API获取预分析的新闻情感、政策影响、事件严重性评分（不主动同步新闻数据）

  * _Requirements: 2.1, 2.4, 2.5, 2.6, 2.7, 2.11_

* [x] 3.3 BacktestService（基础框架）

  * 回测服务基础框架，包含策略回测、性能分析、风险评估等核心功能

  * 支持多策略对比和详细报告生成

  * _Requirements: 2.1, 2.6-2.14, 5.3_

* [x] 3.3.1 回测引擎初始化

  * 初始化引擎、资金与手续费、添加分析器（Sharpe/Drawdown/Returns/Trades）

  * 数据源适配（PandasData），解析回测结果入库

  * _Requirements: 2.1, 5.3_

* [x] 3.3.2 策略基类与注册器

  * 实现策略基类（BaseStrategy）

  * 实现策略注册器（StrategyManager），支持策略动态注册和管理

  * 注册策略类型：
    * 多因子策略：MultiFactorStrategy（唯一核心策略）

  * _Requirements: 2.6, 2.7_

* [ ] 3.3.3 多因子选股策略实现（MultiFactorStrategy）

  * 实现MultiFactorStrategy策略类，继承自BaseStrategy

  * 核心参数配置：
    * 四维度因子权重：technical_weight, fundamental_weight, news_weight, market_weight
    * 交易阈值：buy_threshold, sell_threshold, hold_threshold
    * 风险控制：max_position_size, stop_loss_pct, max_drawdown_pct
    * 置信度：min_confidence_score
    * 回测参数：rebalance_frequency, lookback_period

  * 策略主逻辑实现：
    * 集成FactorService进行四维度因子评分计算
    * 基于综合评分和阈值进行买卖决策
    * 实现风险管理和仓位控制
    * 记录评分历史和交易信号

  * 验收标准：
    * 因子评分计算正确，权重应用准确
    * 交易信号生成符合阈值设定
    * 风险控制机制有效执行
    * 再平衡在周期边界正确执行

  * _Requirements: 2.12_

* [ ] 3.3.4 风险管理策略实现

  * 等权重仓位策略：params={weight\_per\_position}

    * 验收：头寸分配与持仓上限/最小交易单位符合配置

  * 固定止损策略：params={stop\_loss\_pct}

    * 验收：价格回撤达到阈值即触发止损

  * 动态止损（ATR）策略：params={atr\_period, atr\_multiplier}

    * 验收：基于ATR的止损价计算与更新正确

  * _Requirements: 2.14_

* [ ] 3.3.5 多因子回测权重配置支持

  * 升级BacktestService的run_multi_factor_backtest方法，支持可选的factor_weights参数

  * 实现权重配置的动态应用：传入权重时创建临时FactorService实例，否则使用默认权重

  * 更新_calculate_factor_scores方法，支持使用指定的FactorService实例进行因子计算

  * 确保回测结果中记录使用的权重配置，便于结果分析和复现

  * 实现权重隔离：不同回测任务的权重配置互不影响

  * _Requirements: 2.12, 5.3_

* [ ] 3.5 FactorService（多因子评分服务）

  * 实现多因子评分核心服务，支持技术面、基本面、消息面、市场面四个维度的因子计算

  * 权重配置机制：
    * 提供默认四维度权重配置（技术面0.35、基本面0.25、消息面0.25、市场面0.15）
    * 支持初始化时传入自定义权重，实现权重验证和标准化
    * 实现动态权重管理：更新、获取、重置权重功能
    * 支持自定义权重的综合评分计算

  * 四维度因子评分实现：
    * 技术面因子：动量、反转、波动率、技术指标（MA、MACD、RSI、布林带等）
    * 基本面因子：盈利能力、估值水平、财务质量、成长性
    * 消息面因子：新闻情感、政策影响、事件驱动、市场关注度
    * 市场面因子：市场表现、资金流向、市场情绪、行业轮动

  * 核心功能：
    * 综合评分计算，支持批量股票评分
    * 缓存优化和分数标准化
    * 集成DataService获取技术面和基本面数据
    * 按需从数据采集系统API获取消息面数据

  * _Requirements: 2.12, 2.13, 5.3_

* [ ] 3.6 AIService（阿里百炼）

  * 实现 dashscope 客户端封装（或HTTP调用）：鉴权、超时、重试

  * 组装 Prompt（策略分析/风险评估/操作建议），解析响应为结构化建议

  * 超时与失败降级（返回上次方案或规则引擎占位）

  * _Requirements: 3.1-3.16_

* [ ] 3.7 PlanService

  * 将结构化建议格式化为 Markdown（表格/章节），保存与查询

  * _Requirements: 3.4-3.14, 5.1_



## 阶段4：业务编排层（Business Orchestration）（M1）

* [ ] 4.1 BaseOrchestrator 基础编排器

  * 实现编排器基类：统一的错误处理、日志记录、事务管理接口

  * 定义编排流程的通用模式：前置检查→服务调用→结果聚合→异常回滚

  * 提供编排上下文管理和跨服务的数据传递机制

  * _Requirements: 5.5, 5.6_

* [ ] 4.2 PlanOrchestrator 方案生成编排器

  * 编排完整的方案生成流程：数据获取→回测分析→AI分析→方案格式化→结果保存

  * 协调 DataService、BacktestService、AIService、PlanService 完成端到端业务

  * 处理各阶段的失败降级：数据缺失→使用历史数据，AI失败→规则引擎兜底

  * _Requirements: 3.1-3.16, 4.1-4.8_

* [ ] 4.3 BacktestOrchestrator 回测分析编排器

  * 编排回测任务执行流程：策略配置验证→数据准备→回测执行→结果分析→报告生成

  * 支持批量回测和并行执行优化，管理回测任务的生命周期

  * 集成策略注册器，支持动态策略选择和参数优化

  * _Requirements: 2.6-2.14, 5.3_

* [ ] 4.4 PositionOrchestrator 持仓管理编排器

  * 编排持仓相关的复合操作：持仓更新→价格刷新→盈亏计算→风险评估

  * 协调 PositionService 和 DataService 完成持仓数据的实时同步

  * 支持批量持仓操作和组合指标计算的事务一致性

  * _Requirements: 1.1-1.7, 5.1, 5.7_



## 阶段5：API接口层（FastAPI）（M1）

* [ ] 5.1 持仓管理接口

  * GET/POST/PUT/DELETE /api/v1/positions

  * 组合市值/盈亏计算接口

  * _Requirements: 5.1-5.4, 5.7_

* [ ] 5.2 回测接口

  * POST /api/v1/backtest/run，GET /api/v1/backtest/results/{id}，GET /api/v1/backtest/strategies

  * 升级多因子回测接口，支持可选的factor_weights参数传入自定义权重配置

  * 参数校验/异步执行/结果查询

  * _Requirements: 2.8-2.14, 5.3, 5.6_

* [ ] 5.2.1 因子评分接口

  * POST /api/v1/factors/calculate - 批量计算因子评分
    * 支持可选的factor_weights参数传入自定义权重配置
    * 返回四维度因子评分和综合评分
    * 支持单个股票和批量股票评分

  * 权重管理接口：
    * GET /api/v1/factors/weights/default - 获取默认因子权重配置
    * PUT /api/v1/factors/weights - 更新因子权重配置
    * POST /api/v1/factors/weights/reset - 重置为默认权重
    * POST /api/v1/factors/weights/validate - 验证权重配置

  * 功能特性：
    * 参数校验、错误处理、结果缓存
    * 权重配置验证和标准化
    * 支持权重配置的临时应用和持久化

  * _Requirements: 2.12, 2.13, 5.3_

* [ ] 5.3 方案接口

  * GET /api/v1/plans/today，GET /api/v1/plans/history?days=N，POST /api/v1/plans/generate

  * _Requirements: 3.7, 3.14, 5.1, 5.3_

* [ ] 5.4 系统状态接口

  * GET /api/v1/system/health, GET /api/v1/system/stats

  * 依赖检查：DB/Redis/外部服务可用性

  * _Requirements: 5.4, 4.6-4.7_

## 阶段6：定时调度（APScheduler）（M2）

* [ ] 6.1 定时任务编排（按交易日）

  * 任务流程：
    * 15:30 市场数据更新（技术面、基本面数据）
    * 15:45 多因子评分计算（四维度因子评分和综合评分）
    * 16:00 多因子策略回测（使用最新因子评分）
    * 18:00 AI分析（基于回测结果和因子评分）
    * 19:00 方案生成（整合所有分析结果）

  * 数据获取策略：
    * 技术面和基本面数据：主动从数据采集系统获取
    * 消息面数据：按需从数据采集系统API获取，不进行主动同步
    * 市场面数据：实时计算和缓存

  * 任务配置：
    * 可配置时区/节假日跳过/失败告警
    * 支持任务依赖和串行执行
    * 因子权重配置的定时更新和验证

  * _Requirements: 4.1-4.8, 2.12, 2.13_

* [ ] 6.2 失败恢复与幂等

  * 失败记录/断点续跑/重试上限（数据获取重试3次）

  * _Requirements: 2.3, 4.6-4.7_

## 阶段7：质量保障与非功能（M2-M3）

* [ ] 7.1 单元测试

  * Repos/Services/API 覆盖；Mock 外部依赖（数据采集、阿里百炼）

  * 回测模块使用固定样本数据确保可重复

  * FactorService测试：
    * 权重配置：验证、标准化、更新、重置功能
    * 四维度因子计算：技术面、基本面、消息面、市场面
    * 综合评分计算：权重应用、分数标准化
    * 缓存机制和批量处理

  * MultiFactorStrategy测试：
    * 策略参数配置和验证
    * 因子评分集成和交易信号生成
    * 风险管理和仓位控制
    * 回测结果的准确性和一致性

  * 策略注册器测试：
    * 策略注册和获取功能
    * MultiFactorStrategy的正确注册
    * 参数范围和验证机制

  * _Requirements: 2.9-2.13, 3.3, 3.7, 5.6_

* [ ] 7.2 集成与端到端测试

  * 完整业务链路测试：
    * "持仓+数据→多因子评分→多因子回测→AI分析→Markdown方案"的端到端流程
    * 验证数据流转的完整性和准确性
    * 测试各服务间的协调和依赖关系

  * 多因子回测端到端测试：
    * 自定义权重配置→四维度因子计算→MultiFactorStrategy回测→结果分析
    * 权重配置的隔离性和一致性验证
    * 回测结果的可重现性和准确性
    * 因子评分缓存和批量处理的性能

  * 异常处理和降级测试：
    * 数据缺失/异常值的处理
    * 外部服务超时和失败的降级
    * 权重配置错误的容错机制
    * AI服务失败的兜底策略

  * _Requirements: 3.3, 3.16, 4.6, 2.12, 2.13_

* [ ] 7.3 性能与稳定性基线

  * API <3s，AI调用<10s，数据获取<5min，回测批量<30min（并行与缓存）

  * 压测与指标采集（占位）

  * _Requirements: 2.2, 2.9, 5.6_

## 交付顺序（MVP优先）

1. 阶段0→阶段1→阶段2（完成数据面）
2. 阶段3（Services：Position/Data/Backtest/AI/Plan 服务）
3. 阶段4（业务编排：Base/Plan/Backtest/Position 编排器）
4. 阶段5（API）
5. 阶段6（调度）
6. 阶段7（测试与性能）

## 风险与应对

* 外部依赖不稳定：全链路超时/重试/降级；数据落盘与缓存优先

* 回测耗时：并行/缓存、策略分批、指标采集优化

* AI不确定：提示词工程+结构化解析+置信度评估，失败回退到历史方案