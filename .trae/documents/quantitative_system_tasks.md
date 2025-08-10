# 任务拆分文档 (Tasks) — 量化交易系统后端

说明与标注规则
- 本文面向研发与测试团队，覆盖从MVP到迭代优化的实施计划
- 需求映射规则：x.y 表示“需求x 的验收标准 y”，对应 requirements 文档中的“需求N：…/验收标准”序号
- 优先级：以MVP为核心（P0），确保尽快交付可用版本

## 里程碑
- M0 项目基建与数据层奠定（1周）
- M1 核心功能（持仓CRUD、回测、AI方案、API）（2-3周）
- M2 调度自动化与稳定性（1周）
- M3 测试完善与性能守护（持续）

## 阶段0：项目基建与配置（M0）
- [ ] 0.1 建立项目骨架与依赖管理
  - 创建 Python FastAPI 项目骨架、目录与模块
  - 初始化依赖：fastapi, uvicorn, sqlmodel/sqlalchemy, pydantic-settings, httpx, redis, backtrader, pandas, numpy, apscheduler, loguru
  - 本地/开发/生产多环境配置模板
  - _Requirements: 5.6_
- [ ] 0.2 配置与密钥管理
  - settings.py 定义：数据库、Redis、阿里百炼(dashscope)、回测参数
  - 区分环境变量注入敏感信息（不写入日志）
  - _Requirements: 3.6, 5.5_
- [ ] 0.3 日志与错误处理基线
  - 结构化日志（loguru），统一异常中间件、错误码
  - 系统日志落库(system_logs)与采样策略
  - _Requirements: 5.5_

## 阶段1：数据模型与存储（M0）
- [ ] 1.1 建表与迁移脚本
  - 创建并执行迁移：positions, backtest_results, trading_plans, market_data_cache, system_logs
  - 基础索引与约束同步到文档
  - _Requirements: 1.1, 2.14, 3.14, 5.5_
- [ ] 1.2 ORM与Schema
  - 定义 ORM 模型与 Pydantic Schema 输入/输出
  - 金额精度、时间字段、JSON 字段处理
  - _Requirements: 1.1, 2.14_
- [ ] 1.3 Redis 缓存基线
  - 统一 key 规范与过期策略；封装 CacheRepo
  - _Requirements: 2.4, 2.7_

## 阶段2：数据访问层（Repositories）（M0）
- [ ] 2.1 PositionRepo
  - 实现增删改查、聚合（组合市值/盈亏）
  - _Requirements: 1.1-1.7_
- [ ] 2.2 BacktestRepo
  - 保存/查询回测任务与结果（含指标与原始数据JSON）
  - _Requirements: 2.14, 5.3_
- [ ] 2.3 PlanRepo
  - 以日期查询、保存方案、历史分页/范围查询
  - _Requirements: 3.14, 5.3_
- [ ] 2.4 External/DataCollection Client
  - 封装数据采集系统HTTP客户端（重试、超时、熔断占位）
  - _Requirements: 2.1, 2.2, 2.3_

## 阶段3：业务服务层（Services）（M1）
- [ ] 3.1 PositionService
  - 业务校验、组合指标计算、按需更新当前价格/市值（调用数据采集系统）
  - _Requirements: 1.1-1.7, 5.1, 5.6, 5.7_
- [ ] 3.2 DataService
  - 获取/清洗/预处理/缓存市场数据；技术指标计算接口
  - 处理缺失与异常值，返回标准化DataFrame
  - _Requirements: 2.1, 2.4, 2.5, 2.6, 2.7, 2.11_
- [ ] 3.3 BacktestService（backtrader 集成）
  - 初始化引擎、资金与手续费、添加分析器（Sharpe/Drawdown/Returns/Trades）
  - 数据源适配（PandasData），解析回测结果入库
  - 实现策略注册器与核心策略类别（与 docs/trading_strategies.md 对齐，纳入MVP）：
    - 技术分析策略：双均线、三重均线、MACD金叉、布林带突破、RSI反转
    - 动量策略：价格突破、向上缺口
    - 基本面策略：PE-PB双低（价值投资）、成长加速（成长投资）
    - 量化策略：多因子选股、股票配对交易、价差回归
    - 风险管理：等权重仓位、固定止损、动态止损（ATR）
  - 策略参数与验收标准补充：
    - 双均线：params={short_window, long_window, stop_loss_pct?}
      - 验收：金叉/死叉信号按窗口交叉正确触发；交易次数、持仓区间与手工样本一致；能产出 Sharpe/Drawdown/Returns/Trades 指标
    - 三重均线：params={short_window, mid_window, long_window}
      - 验收：多头/空头排列识别正确；回测可重复；指标产出完整
    - MACD金叉：params={fast_period, slow_period, signal_period}
      - 验收：金叉/死叉触发点与计算结果一致；直方图由负转正/正转负与信号匹配
    - 布林带突破：params={period, k, squeeze_threshold?, take_profit_atr_multiplier?}
      - 验收：挤压识别与上下轨突破信号正确；失败突破能触发止损
    - RSI反转：params={period, overbought, oversold}
      - 验收：超买/超卖与阈值符合；反转确认后入场/出场逻辑正确
    - 价格突破：params={breakout_lookback, confirm_volume_ratio?, stop_loss_pct?}
      - 验收：近N日高/低突破识别准确；量能确认可选；假突破能止损
    - 向上缺口：params={gap_pct_threshold, min_volume?, hold_days?}
      - 验收：跳空幅度判断正确；缺口回补与持有期规则正确执行
    - 多因子选股：params={factors[], weights[], rebalance_period}
      - 验收：打分/加权与排序选股正确；再平衡在周期边界正确执行
    - 股票配对交易：params={lookback, coint_pvalue, entry_z, exit_z, stop_z?}
      - 验收：协整检验与Z-score进出场阈值正确；价差回归到均值时平仓
    - 价差回归：params={mean_window, entry_std, exit_std}
      - 验收：偏离均值Nσ进场、回归<阈值出场逻辑正确
    - 等权重仓位：params={weight_per_position}
      - 验收：头寸分配与持仓上限/最小交易单位符合配置
    - 固定止损：params={stop_loss_pct}
      - 验收：价格回撤达到阈值即触发止损
    - 动态止损（ATR）：params={atr_period, atr_multiplier}
      - 验收：基于ATR的止损价计算与更新正确
  - 单元/回测用例：为上述每个策略新增UT（信号/边界/异常）与回测BT用例（固定样本数据，结果可复现），集成到CI
  - _Requirements: 2.1, 2.6-2.14, 5.3_
- [ ] 3.4 AIService（阿里百炼）
  - 实现 dashscope 客户端封装（或HTTP调用）：鉴权、超时、重试
  - 组装 Prompt（策略分析/风险评估/操作建议），解析响应为结构化建议
  - 超时与失败降级（返回上次方案或规则引擎占位）
  - _Requirements: 3.1-3.16_
- [ ] 3.5 PlanService
  - 将结构化建议格式化为 Markdown（表格/章节），保存与查询
  - _Requirements: 3.4-3.14, 5.1_

## 阶段4：API接口层（FastAPI）（M1）
- [ ] 4.1 持仓管理接口
  - GET/POST/PUT/DELETE /api/v1/positions
  - 组合市值/盈亏计算接口
  - _Requirements: 5.1-5.4, 5.7_
- [ ] 4.2 回测接口
  - POST /api/v1/backtest/run，GET /api/v1/backtest/results/{id}，GET /api/v1/backtest/strategies
  - 参数校验/异步执行/结果查询
  - _Requirements: 2.8-2.14, 5.3, 5.6_
- [ ] 4.3 方案接口
  - GET /api/v1/plans/today，GET /api/v1/plans/history?days=N，POST /api/v1/plans/generate
  - _Requirements: 3.7, 3.14, 5.1, 5.3_
- [ ] 4.4 系统状态接口
  - GET /api/v1/system/health, GET /api/v1/system/stats
  - 依赖检查：DB/Redis/外部服务可用性
  - _Requirements: 5.4, 4.6-4.7_

## 阶段5：定时调度（APScheduler）（M2）
- [ ] 5.1 定时任务编排（按交易日）
  - 15:30 数据更新 → 16:00 回测 → 18:00 AI分析 → 19:00 方案生成
  - 可配置时区/节假日跳过/失败告警占位
  - _Requirements: 4.1-4.8_
- [ ] 5.2 失败恢复与幂等
  - 失败记录/断点续跑/重试上限（数据获取重试3次）
  - _Requirements: 2.3, 4.6-4.7_

## 阶段6：质量保障与非功能（M2-M3）
- [ ] 6.1 单元测试
  - Repos/Services/API 覆盖；Mock 外部依赖（数据采集、阿里百炼）
  - 回测模块使用固定样本数据确保可重复
  - _Requirements: 2.9-2.13, 3.3, 3.7, 5.6_
- [ ] 6.2 集成与端到端测试
  - 从“持仓+数据→回测→AI→Markdown方案”的完整链路
  - 异常/边界/超时与降级路径
  - _Requirements: 3.3, 3.16, 4.6_
- [ ] 6.3 性能与稳定性基线
  - API <3s，AI调用<10s，数据获取<5min，回测批量<30min（并行与缓存）
  - 压测与指标采集（占位）
  - _Requirements: 2.2, 2.9, 5.6_

## 交付顺序（MVP优先）
1) 阶段0→阶段1→阶段2（完成数据面）
2) 阶段3（Position/Data/Backtest/AI/Plan 服务）
3) 阶段4（API）
4) 阶段5（调度）
5) 阶段6（测试与性能）

## 风险与应对
- 外部依赖不稳定：全链路超时/重试/降级；数据落盘与缓存优先
- 回测耗时：并行/缓存、策略分批、指标采集优化
- AI不确定性：提示词工程+结构化解析+置信度评估，失败回退到历史方案
